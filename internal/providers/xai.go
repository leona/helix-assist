package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/leona/helix-assist/internal/lsp"
	"github.com/leona/helix-assist/internal/util"
)

type XAIProvider struct {
	apiKey    string
	model     string
	chatModel string
	endpoint  string
	timeout   time.Duration
	logger    *lsp.Logger
}

func NewXAIProvider(apiKey, model, chatModel, endpoint string, timeoutMs int, logger *lsp.Logger) *XAIProvider {
	if chatModel == "" {
		chatModel = model
	}
	if endpoint == "" {
		endpoint = "https://api.x.ai"
	}
	return &XAIProvider{
		apiKey:    apiKey,
		model:     model,
		chatModel: chatModel,
		endpoint:  strings.TrimSuffix(endpoint, "/"),
		timeout:   time.Duration(timeoutMs) * time.Millisecond,
		logger:    logger,
	}
}

type xaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type xaiRequest struct {
	Model       string       `json:"model"`
	MaxTokens   int          `json:"max_tokens"`
	Messages    []xaiMessage `json:"messages"`
	Temperature float64      `json:"temperature,omitempty"`
}

type xaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (p *XAIProvider) Completion(ctx context.Context, req CompletionRequest, filepath, languageID string, numSuggestions int) ([]string, error) {
	systemPrompt := BuildCompletionSystemPrompt(languageID)
	userPrompt := BuildCompletionUserPrompt(filepath, req.ContentBefore, req.ContentAfter)

	temperature := 0.0
	if numSuggestions > 1 {
		temperature = 0.4
	}

	results := make([]string, 0, numSuggestions)

	for i := 0; i < numSuggestions; i++ {
		apiReq := xaiRequest{
			Model:     p.model,
			MaxTokens: 256,
			Messages: []xaiMessage{
				{Role: "system", Content: systemPrompt},
				{Role: "user", Content: userPrompt},
			},
			Temperature: temperature,
		}

		resp, err := p.doRequest(ctx, "/v1/chat/completions", apiReq)
		if err != nil {
			if len(results) > 0 {
				break
			}
			return nil, err
		}

		var apiResp xaiResponse
		if err := json.Unmarshal(resp, &apiResp); err != nil {
			if len(results) > 0 {
				break
			}
			return nil, fmt.Errorf("parse response: %w", err)
		}

		for _, choice := range apiResp.Choices {
			if choice.Message.Content != "" {
				content := strings.TrimPrefix(choice.Message.Content, "CODE:")
				results = append(results, content)
			}
		}
	}

	return util.UniqueStrings(results), nil
}

func (p *XAIProvider) Chat(ctx context.Context, query, content, filepath, languageID string) (*ChatResponse, error) {
	cleanFilepath := strings.TrimPrefix(filepath, "file://")

	systemPrompt := BuildChatSystemPrompt(languageID)
	userContent := BuildChatUserPrompt(languageID, cleanFilepath, content, query)

	apiReq := xaiRequest{
		Model:     p.chatModel,
		MaxTokens: 8192,
		Messages: []xaiMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
		Temperature: 0.1,
	}

	jsonReq, _ := json.MarshalIndent(apiReq, "", "  ")
	p.logger.Log("DEBUG [XAI Chat]: Request:", string(jsonReq))

	resp, err := p.doRequest(ctx, "/v1/chat/completions", apiReq)
	if err != nil {
		return nil, err
	}

	p.logger.Log("DEBUG [XAI Chat]: Raw response:", string(resp))

	var apiResp xaiResponse
	if err := json.Unmarshal(resp, &apiResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(apiResp.Choices) == 0 || apiResp.Choices[0].Message.Content == "" {
		return nil, fmt.Errorf("no completion found")
	}

	text := apiResp.Choices[0].Message.Content
	p.logger.Log("DEBUG [XAI Chat]: Extracted text:", text)
	return &ChatResponse{Result: text}, nil
}

func (p *XAIProvider) doRequest(ctx context.Context, endpoint string, body any) ([]byte, error) {
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
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

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
