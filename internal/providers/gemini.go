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

type GeminiProvider struct {
	apiKey    string
	model     string
	chatModel string
	endpoint  string
	timeout   time.Duration
	logger    *lsp.Logger
}

func NewGeminiProvider(apiKey, model, chatModel, endpoint string, timeoutMs int, logger *lsp.Logger) *GeminiProvider {
	if chatModel == "" {
		chatModel = model
	}

	return &GeminiProvider{
		apiKey:    apiKey,
		model:     model,
		chatModel: chatModel,
		endpoint:  strings.TrimSuffix(endpoint, "/"),
		timeout:   time.Duration(timeoutMs) * time.Millisecond,
		logger:    logger,
	}
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiContent struct {
	Role  string     `json:"role"`
	Parts GeminiPart `json:"parts"`
}

type geminiRequest struct {
	Contents []GeminiContent `json:"contents"`
}

type GeminiCandidate struct {
	Contents struct {
		Parts []GeminiPart `json:"parts"`
	} `json:"contents"`
}

type geminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
}

func (p *GeminiProvider) Completion(ctx context.Context, req CompletionRequest, filepath, languageID string, numSuggestions int) ([]string, error) {
	systemPrompt := BuildCompletionSystemPrompt(languageID)
	userPrompt := BuildCompletionUserPrompt(filepath, req.ContentBefore, req.ContentAfter)

	results := make([]string, 0, numSuggestions)

	for range numSuggestions {
		apiReq := geminiRequest{
			Contents: []GeminiContent{
				{Role: "model", Parts: GeminiPart{Text: systemPrompt}},
				{Role: "user", Parts: GeminiPart{Text: userPrompt}},
			},
		}

		resp, err := p.doRequest(ctx, "/v1beta/models", apiReq)

		if err != nil {
			if len(results) > 0 {
				break
			}

			return nil, err
		}

		var apiResp geminiResponse
		if err := json.Unmarshal(resp, &apiResp); err != nil {
			return nil, fmt.Errorf("parse response, %w", err)
		}

		if len(apiResp.Candidates) == 0 {
			return nil, fmt.Errorf("no completion found")
		}

		for _, candidate := range apiResp.Candidates {
			for _, part := range candidate.Contents.Parts {
				results = append(results, part.Text)
			}
		}

	}
	return util.UniqueStrings(results), nil
}

func (p *GeminiProvider) doRequest(ctx context.Context, endpoint string, body any) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	url := p.endpoint + endpoint + "/" + p.model + ":generateContent"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", p.apiKey)

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

func (p *GeminiProvider) Chat(ctx context.Context, query, content, filepath, languageID string) (*ChatResponse, error) {
	cleanFilePath := strings.TrimPrefix(filepath, "file://")

	systemPrompt := BuildChatSystemPrompt(languageID)
	userContent := BuildChatUserPrompt(languageID, cleanFilePath, content, query)

	apiReq := geminiRequest{
		Contents: []GeminiContent{
			{Role: "model", Parts: GeminiPart{Text: systemPrompt}},
			{Role: "user", Parts: GeminiPart{Text: userContent}},
		},
	}

	jsonReq, _ := json.MarshalIndent(apiReq, "", "  ")
	p.logger.Log("DEBUG [Gemini Chat]: Request:", string(jsonReq))

	resp, err := p.doRequest(ctx, "/v1beta/models", apiReq)
	if err != nil {
		return nil, err
	}

	p.logger.Log("DEBUG [Gemini Chat]: Raw response:", string(resp))

	var apiResp geminiResponse
	if err := json.Unmarshal(resp, &apiResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(apiResp.Candidates) == 0 {
		return nil, fmt.Errorf("no completion found")
	}

	var resultText string
	for _, candidate := range apiResp.Candidates {
		for _, part := range candidate.Contents.Parts {
			resultText = part.Text
			break
		}
	}

	p.logger.Log("DEBUG [Gemini Chat]: Extracted text:", resultText)
	return &ChatResponse{Result: resultText}, nil
}
