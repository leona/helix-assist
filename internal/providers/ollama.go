package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/leona/helix-assist/internal/lsp"
	"github.com/leona/helix-assist/internal/util"
)

type OllamaProvider struct {
	model      string
	chatModel  string
	endpoint   string
	disableFIM bool
	timeout    time.Duration
	logger     *lsp.Logger
}

func NewOllamaProvider(model, chatModel, endpoint string, disableFIM bool, timeoutMs int, logger *lsp.Logger) *OllamaProvider {
	if chatModel == "" {
		chatModel = model
	}
	return &OllamaProvider{
		model:      model,
		chatModel:  chatModel,
		endpoint:   strings.TrimSuffix(endpoint, "/"),
		disableFIM: disableFIM,
		timeout:    time.Duration(timeoutMs) * time.Millisecond,
		logger:     logger,
	}
}

type ollamaGenerateRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Suffix  string                 `json:"suffix,omitempty"`
	System  string                 `json:"system,omitempty"`
	Raw     bool                   `json:"raw,omitempty"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options,omitempty"`
}

type ollamaGenerateResponse struct {
	Response string `json:"response"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatRequest struct {
	Model    string                 `json:"model"`
	Messages []ollamaMessage        `json:"messages"`
	Stream   bool                   `json:"stream"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

type ollamaChatResponse struct {
	Message ollamaMessage `json:"message"`
}

func (p *OllamaProvider) Completion(ctx context.Context, req CompletionRequest, filepath, languageID string, numSuggestions int) ([]string, error) {
	temperature := 0.0
	if numSuggestions > 1 {
		temperature = 0.4
	}

	results := make([]string, 0, numSuggestions)

	for i := 0; i < numSuggestions; i++ {
		var apiReq ollamaGenerateRequest
		if p.disableFIM {
			apiReq = ollamaGenerateRequest{
				Model:   p.model,
				Prompt:  BuildCompletionUserPrompt(filepath, req.ContentBefore, req.ContentAfter),
				System:  BuildCompletionSystemPrompt(languageID),
				Raw:     false,
				Stream:  false,
				Options: map[string]interface{}{
					"temperature": temperature,
					"num_predict": 256,
				},
			}
		} else {
			// Native FIM by providing prompt and suffix
			apiReq = ollamaGenerateRequest{
				Model:   p.model,
				Prompt:  req.ContentBefore,
				Suffix:  req.ContentAfter,
				Raw:     true,
				Stream:  false,
				Options: map[string]interface{}{
					"temperature": temperature,
					"num_predict": 256,
					"stop": []string{"<|fim_prefix|>", "<|fim_suffix|>", "<|fim_middle|>", "<|file_sep|>", "<|endoftext|>"},
				},
			}
		}

		resp, err := p.doRequest(ctx, "/api/generate", apiReq)
		if err != nil {
			if len(results) > 0 {
				break
			}
			return nil, err
		}

		var apiResp ollamaGenerateResponse
		if err := json.Unmarshal(resp, &apiResp); err != nil {
			if len(results) > 0 {
				break
			}
			return nil, fmt.Errorf("parse response: %w", err)
		}

		if apiResp.Response != "" {
			cleanResponse := stripMarkdown(apiResp.Response)
			results = append(results, cleanResponse)
		}
	}

	return util.UniqueStrings(results), nil
}

func stripMarkdown(text string) string {
	// 1. Remove the leading ```language tag (and optional whitespace/newline)
	reStart := regexp.MustCompile("(?i)^\\s*```[a-z0-9+\\-]*\\s*\\n?")
	text = reStart.ReplaceAllString(text, "")
	
	// 2. Remove the trailing ``` (and optional newline)
	reEnd := regexp.MustCompile("\\n?\\s*```\\s*$")
	text = reEnd.ReplaceAllString(text, "")
	
	return text
}

func (p *OllamaProvider) Chat(ctx context.Context, query, content, filepath, languageID string) (*ChatResponse, error) {
	cleanFilepath := strings.TrimPrefix(filepath, "file://")

	systemPrompt := BuildChatSystemPrompt(languageID)
	userContent := BuildChatUserPrompt(languageID, cleanFilepath, content, query)

	apiReq := ollamaChatRequest{
		Model:  p.chatModel,
		Stream: false,
		Messages: []ollamaMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
		Options: map[string]interface{}{
			"temperature": 0.1,
		},
	}

	jsonReq, _ := json.MarshalIndent(apiReq, "", "  ")
	p.logger.Log("DEBUG [Ollama Chat]: Request:", string(jsonReq))

	resp, err := p.doRequest(ctx, "/api/chat", apiReq)
	if err != nil {
		return nil, err
	}

	p.logger.Log("DEBUG [Ollama Chat]: Raw response:", string(resp))

	var apiResp ollamaChatResponse
	if err := json.Unmarshal(resp, &apiResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if apiResp.Message.Content == "" {
		return nil, fmt.Errorf("no completion found")
	}

	cleanResponse := stripMarkdown(apiResp.Message.Content)

	p.logger.Log("DEBUG [Ollama Chat]: Extracted text:", cleanResponse)
	return &ChatResponse{Result: cleanResponse}, nil
}

func (p *OllamaProvider) doRequest(ctx context.Context, endpoint string, body any) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	url := p.endpoint + endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
