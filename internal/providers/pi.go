package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/leona/helix-assist/internal/lsp"
)

type piCompletionResponse struct {
	Completions []string `json:"completions"`
}

type piChatResponse struct {
	Result string `json:"result"`
}

type PiProvider struct {
	cli    string
	logger *lsp.Logger
}

func NewPiProvider(cli string, logger *lsp.Logger) *PiProvider {
	return &PiProvider{cli: cli, logger: logger}
}

func (p *PiProvider) Completion(ctx context.Context, req CompletionRequest, filepath, languageID string, numSuggestions int) ([]string, error) {
	prompt := buildPiCompletionPrompt(filepath, languageID, req.ContentBefore, req.ContentAfter, numSuggestions)
	stdout, err := p.run(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var resp piCompletionResponse
	if err := json.Unmarshal(stdout, &resp); err != nil {
		if extracted := extractJSONObject(stdout); extracted != nil {
			if err := json.Unmarshal(extracted, &resp); err != nil {
				return nil, fmt.Errorf("parse completion response: %w", err)
			}
		} else {
			return nil, fmt.Errorf("parse completion response: %w", err)
		}
	}

	return resp.Completions, nil
}

func (p *PiProvider) Chat(ctx context.Context, query, content, filepath, languageID string) (*ChatResponse, error) {
	prompt := buildPiChatPrompt(query, content, filepath, languageID)
	stdout, err := p.run(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var resp piChatResponse
	if err := json.Unmarshal(stdout, &resp); err != nil {
		if extracted := extractJSONObject(stdout); extracted != nil {
			if err := json.Unmarshal(extracted, &resp); err != nil {
				return nil, fmt.Errorf("parse chat response: %w", err)
			}
		} else {
			return nil, fmt.Errorf("parse chat response: %w", err)
		}
	}

	return &ChatResponse{Result: resp.Result}, nil
}

func (p *PiProvider) run(ctx context.Context, prompt string) ([]byte, error) {
	args := strings.Fields(p.cli)
	if len(args) == 0 {
		return nil, fmt.Errorf("empty pi cli command")
	}

	args = append(args, prompt)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	p.logger.Log("DEBUG [Pi run]: command:", cmd.String())

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("pi command failed: %w: %s", err, stderr.String())
		}
		return nil, fmt.Errorf("pi command failed: %w", err)
	}

	if stderr.Len() > 0 {
		p.logger.Log("DEBUG [Pi run]: stderr:", stderr.String())
	}

	p.logger.Log("DEBUG [Pi run]: stdout:", stdout.String())
	return stdout.Bytes(), nil
}

func buildPiCompletionPrompt(filepath, languageID, contentBefore, contentAfter string, numSuggestions int) string {
	return fmt.Sprintf(`%s

%s

IMPORTANT: Respond ONLY with a JSON object in this exact format, containing up to %d completions. Do not include markdown, explanations, or any text outside the JSON object:
{"completions": ["<completion_1>", "<completion_2>"]}`+
		`
Each completion must be a single string containing only the code that fits at the cursor position.`,
		BuildCompletionSystemPrompt(languageID),
		BuildCompletionUserPrompt(filepath, contentBefore, contentAfter),
		numSuggestions,
	)
}

func buildPiChatPrompt(query, content, filepath, languageID string) string {
	return fmt.Sprintf(`%s

%s

IMPORTANT: Respond ONLY with a JSON object in this exact format. Do not include markdown, explanations, or any text outside the JSON object:
{"result": "<corrected_or_improved_code>"}`,
		BuildChatSystemPrompt(languageID),
		BuildChatUserPrompt(languageID, filepath, content, query),
	)
}

func extractJSONObject(data []byte) []byte {
	s := string(data)
	start := strings.Index(s, "{")
	if start == -1 {
		return nil
	}
	end := strings.LastIndex(s, "}")
	if end == -1 || end <= start {
		return nil
	}
	return []byte(s[start : end+1])
}
