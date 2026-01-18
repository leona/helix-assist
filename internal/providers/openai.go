package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/leona/helix-assist/internal/util"
)

type OpenAIProvider struct {
	apiKey   string
	model    string
	endpoint string
	timeout  time.Duration
}

func NewOpenAIProvider(apiKey, model, endpoint string, timeoutMs int) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey:   apiKey,
		model:    model,
		endpoint: strings.TrimSuffix(endpoint, "/"),
		timeout:  time.Duration(timeoutMs) * time.Millisecond,
	}
}

type responsesRequest struct {
	Model        string                 `json:"model"`
	Input        string                 `json:"input"`
	Instructions string                 `json:"instructions,omitempty"`
	Store        bool                   `json:"store"`
	ServiceTier  string                 `json:"service_tier,omitempty"`
	MaxToolCalls int                    `json:"max_tool_calls,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
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
	instructions := fmt.Sprintf(`You are a %s code completion assistant. Complete the code at the cursor position.

Rules:
- Output ONLY the code that should be inserted at the cursor
- Do NOT include any code that already exists before or after the cursor
- Do NOT add explanations, comments, or markdown formatting
- Do NOT repeat existing code
- Generate syntactically correct %s code`, languageID, languageID)

	userPrompt := fmt.Sprintf("File: %s\n\nCode before cursor:\n%s\n\n<CURSOR>\n\nCode after cursor:\n%s", filepath, req.ContentBefore, req.ContentAfter)

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

	instructions := fmt.Sprintf(`You are an AI programming assistant.
Follow the user's requirements carefully & to the letter.
- Each code block starts with `+"```"+` and // FILEPATH.
- You always answer with %s code.
- When the user asks you to document something, you must answer in the form of a %s code block.
Your expertise is strictly limited to software development topics.
Keep your answers short and impersonal.`, languageID, languageID)

	userContent := fmt.Sprintf("I have the following code in the selection:\n```%s\n// FILEPATH: %s\n%s\n\n%s", languageID, cleanFilepath, content, query)

	respReq := responsesRequest{
		Model:        p.model,
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

	resp, err := p.doRequest(ctx, "/responses", respReq)
	if err != nil {
		return nil, err
	}

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

	result := util.ExtractCodeBlock(filepath, resultText, languageID)
	return &ChatResponse{Result: result}, nil
}

func (p *OpenAIProvider) doRequest(ctx context.Context, endpoint string, body any) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	fmt.Fprintf(os.Stderr, "DEBUG: Request to %s\n", endpoint)
	fmt.Fprintf(os.Stderr, "DEBUG: Body: %s\n", string(jsonBody))

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

	fmt.Fprintf(os.Stderr, "DEBUG: Response status: %d\n", resp.StatusCode)
	fmt.Fprintf(os.Stderr, "DEBUG: Response body: %s\n", string(respBody))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
