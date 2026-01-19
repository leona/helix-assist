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

type AnthropicProvider struct {
	apiKey   string
	model    string
	endpoint string
	timeout  time.Duration
	logger   *lsp.Logger
}

func NewAnthropicProvider(apiKey, model, endpoint string, timeoutMs int, logger *lsp.Logger) *AnthropicProvider {
	return &AnthropicProvider{
		apiKey:   apiKey,
		model:    model,
		endpoint: strings.TrimSuffix(endpoint, "/"),
		timeout:  time.Duration(timeoutMs) * time.Millisecond,
		logger:   logger,
	}
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicCacheControl struct {
	Type string `json:"type"`
}

type anthropicSystemContent struct {
	Type         string                 `json:"type"`
	Text         string                 `json:"text"`
	CacheControl *anthropicCacheControl `json:"cache_control,omitempty"`
}

type anthropicRequest struct {
	Model       string                   `json:"model"`
	MaxTokens   int                      `json:"max_tokens"`
	System      []anthropicSystemContent `json:"system,omitempty"`
	Messages    []anthropicMessage       `json:"messages"`
	Temperature float64                  `json:"temperature,omitempty"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

func (p *AnthropicProvider) Completion(ctx context.Context, req CompletionRequest, filepath, languageID string, numSuggestions int) ([]string, error) {
	systemPrompt := BuildCompletionSystemPrompt(languageID)
	userPrompt := BuildCompletionUserPrompt(filepath, req.ContentBefore, req.ContentAfter)

	temperature := 0.0

	if numSuggestions > 1 {
		temperature = 0.4
	}

	results := make([]string, 0, numSuggestions)

	for i := 0; i < numSuggestions; i++ {
		apiReq := anthropicRequest{
			Model:     p.model,
			MaxTokens: 256,
			System: []anthropicSystemContent{
				{
					Type:         "text",
					Text:         systemPrompt,
					CacheControl: &anthropicCacheControl{Type: "ephemeral"},
				},
			},
			Temperature: temperature,
			Messages: []anthropicMessage{
				{Role: "user", Content: userPrompt},
			},
		}

		resp, err := p.doRequest(ctx, "/v1/messages", apiReq)

		if err != nil {
			if len(results) > 0 {
				break
			}
			return nil, err
		}

		var apiResp anthropicResponse

		if err := json.Unmarshal(resp, &apiResp); err != nil {
			if len(results) > 0 {
				break
			}
			return nil, fmt.Errorf("parse response: %w", err)
		}

		for _, content := range apiResp.Content {
			if content.Type == "text" && content.Text != "" {
				results = append(results, content.Text)
			}
		}
	}

	return util.UniqueStrings(results), nil
}

func (p *AnthropicProvider) Chat(ctx context.Context, query, content, filepath, languageID string) (*ChatResponse, error) {
	cleanFilepath := strings.TrimPrefix(filepath, "file://")

	systemPrompt := BuildChatSystemPrompt(languageID)
	userContent := BuildChatUserPrompt(languageID, cleanFilepath, content, query)

	apiReq := anthropicRequest{
		Model:     p.model,
		MaxTokens: 8192,
		System: []anthropicSystemContent{
			{
				Type: "text",
				Text: systemPrompt,
			},
		},
		Temperature: 0.1,
		Messages: []anthropicMessage{
			{Role: "user", Content: userContent},
		},
	}

	jsonReq, _ := json.MarshalIndent(apiReq, "", "  ")
	p.logger.Log("DEBUG [Anthropic Chat]: Request:", string(jsonReq))

	resp, err := p.doRequest(ctx, "/v1/messages", apiReq)
	if err != nil {
		return nil, err
	}

	p.logger.Log("DEBUG [Anthropic Chat]: Raw response:", string(resp))

	var apiResp anthropicResponse
	if err := json.Unmarshal(resp, &apiResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("no completion found")
	}

	var resultText string
	for _, content := range apiResp.Content {
		if content.Type == "text" {
			resultText = content.Text
			break
		}
	}

	p.logger.Log("DEBUG [Anthropic Chat]: Extracted text:", resultText)
	return &ChatResponse{Result: resultText}, nil
}

func (p *AnthropicProvider) doRequest(ctx context.Context, endpoint string, body any) ([]byte, error) {
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
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

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
