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

var reasoningModels = map[string]bool{
	"gpt-5.2":           true,
	"gpt-5.1":           true,
	"gpt-5":             true,
	"gpt-5-mini":        true,
	"gpt-5.2-codex":     true,
	"gpt-5.1-codex-max": true,
	"gpt-5.1-codex":     true,
	"gpt-5-codex":       true,
}

type OpenAIProvider struct {
	apiKey    string
	model     string
	chatModel string
	endpoint  string
	timeout   time.Duration
	logger    *lsp.Logger
}

func isReasoningModel(model string) bool {
	return reasoningModels[model]
}

func NewOpenAIProvider(apiKey, model, chatModel, endpoint string, timeoutMs int, logger *lsp.Logger) *OpenAIProvider {
	if chatModel == "" {
		chatModel = model
	}
	return &OpenAIProvider{
		apiKey:    apiKey,
		model:     model,
		chatModel: chatModel,
		endpoint:  strings.TrimSuffix(endpoint, "/"),
		timeout:   time.Duration(timeoutMs) * time.Millisecond,
		logger:    logger,
	}
}

type reasoningConfig struct {
	Effort string `json:"effort"`
}

type responsesRequest struct {
	Model        string                 `json:"model"`
	Input        string                 `json:"input"`
	Instructions string                 `json:"instructions,omitempty"`
	Store        bool                   `json:"store"`
	ServiceTier  string                 `json:"service_tier,omitempty"`
	MaxToolCalls int                    `json:"max_tool_calls,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Reasoning    *reasoningConfig       `json:"reasoning,omitempty"`
}

type responsesResponse struct {
	Output []struct {
		Type    string `json:"type"`
		Role    string `json:"role,omitempty"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

func (p *OpenAIProvider) Completion(ctx context.Context, req CompletionRequest, filepath, languageID string, numSuggestions int) ([]string, error) {
	instructions := BuildCompletionSystemPrompt(languageID)
	userPrompt := BuildCompletionUserPrompt(filepath, req.ContentBefore, req.ContentAfter)

	results := make([]string, 0, numSuggestions)

	for i := 0; i < numSuggestions; i++ {
		respReq := responsesRequest{
			Model:        p.model,
			Instructions: instructions,
			Input:        userPrompt,
			Store:        false,
			ServiceTier:  "priority",
			MaxToolCalls: 0,
			Metadata: map[string]interface{}{
				"language": languageID,
				"filepath": filepath,
			},
		}

		if isReasoningModel(p.model) {
			respReq.Reasoning = &reasoningConfig{
				Effort: "minimal",
			}
		}

		resp, err := p.doRequest(ctx, "/responses", respReq)
		if err != nil {
			if len(results) > 0 {
				break
			}
			return nil, err
		}

		var respResp responsesResponse
		if err := json.Unmarshal(resp, &respResp); err != nil {
			if len(results) > 0 {
				break
			}
			return nil, fmt.Errorf("parse response: %w", err)
		}

		for _, output := range respResp.Output {
			if output.Type == "message" {
				for _, content := range output.Content {
					if content.Type == "output_text" && content.Text != "" {
						results = append(results, content.Text)
					}
				}
			}
		}
	}

	return util.UniqueStrings(results), nil
}

func (p *OpenAIProvider) Chat(ctx context.Context, query, content, filepath, languageID string) (*ChatResponse, error) {
	cleanFilepath := strings.TrimPrefix(filepath, "file://")

	instructions := BuildChatSystemPrompt(languageID)
	userContent := BuildChatUserPrompt(languageID, cleanFilepath, content, query)

	respReq := responsesRequest{
		Model:        p.chatModel,
		Instructions: instructions,
		Input:        userContent,
		Store:        false,
		ServiceTier:  "priority",
		MaxToolCalls: 0,
		Metadata: map[string]interface{}{
			"language": languageID,
			"filepath": filepath,
		},
	}

	if isReasoningModel(p.chatModel) {
		respReq.Reasoning = &reasoningConfig{
			Effort: "minimal",
		}
	}

	jsonReq, _ := json.MarshalIndent(respReq, "", "  ")
	p.logger.Log("DEBUG [OpenAI Chat]: Request:", string(jsonReq))
	resp, err := p.doRequest(ctx, "/responses", respReq)

	if err != nil {
		return nil, err
	}

	p.logger.Log("DEBUG [OpenAI Chat]: Raw response:", string(resp))
	var respResp responsesResponse

	if err := json.Unmarshal(resp, &respResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	var resultText string
	for _, output := range respResp.Output {
		if output.Type == "message" {
			for _, content := range output.Content {
				if content.Type == "output_text" {
					resultText = content.Text
					break
				}
			}
			if resultText != "" {
				break
			}
		}
	}

	if resultText == "" {
		return nil, fmt.Errorf("no completion found")
	}

	p.logger.Log("DEBUG [OpenAI Chat]: Extracted text:", resultText)
	return &ChatResponse{Result: resultText}, nil
}

func (p *OpenAIProvider) doRequest(ctx context.Context, endpoint string, body any) ([]byte, error) {
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
