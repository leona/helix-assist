package providers

import (
	"context"
	"encoding/json"
	"fmt"
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

// let mut contents: Vec<_> = history
//     .into_iter()
//     .map(|msg| content_builder(&msg.role, &msg.content))
//     .collect();
// contents.push(content_builder("user", &prompt));
// let request_body = serde_json::json!({"contents": contents});
// let response: serializer::Response = http::post(&url, request_body, None).await?;
//

type GeminiPart struct {
	Text string
}

type GeminiContent struct {
	Role  string
	Parts GeminiPart
}

type GeminiRequest struct {
	Contents []GeminiContent
}

type GeminiCandidate struct {
	Contents struct {
		Parts []GeminiPart
	}
}

type GeminiResponse struct {
	Candidates []GeminiCandidate
}

func (p *GeminiProvider) Completion(ctx context.Context, req CompletionRequest, filepath, languageID string, numSuggestions int) ([]string, error) {
	systemPrompt := BuildCompletionSystemPrompt(languageID)
	userPrompt := BuildCompletionUserPrompt(filepath, req.ContentBefore, req.ContentAfter)

	results := make([]string, 0, numSuggestions)

	for range numSuggestions {
		apiReq := GeminiRequest{
			Contents: []GeminiContent{
				{Role: "model", Parts: GeminiPart{Text: systemPrompt}},
				{Role: "user", Parts: GeminiPart{Text: userPrompt}},
			},
		}

		resp, err := p.doRequest(ctx, "", apiReq) // TODO: add endpoint on this

		if err != nil {
			if len(results) > 0 {
				break
			}

			return nil, err
		}

		var apiResp GeminiResponse
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
	return nil, nil
}
