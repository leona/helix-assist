package providers

import (
	"context"
	"time"
)

type FakeProvider struct{}

func NewFakeProvider() *FakeProvider {
	return &FakeProvider{}
}

func (p *FakeProvider) Completion(ctx context.Context, req CompletionRequest, filepath, languageID string, numSuggestions int) ([]string, error) {
	time.Sleep(500 * time.Millisecond)
	return []string{"// fake completion\nfake_insertion();"}, nil
}

func (p *FakeProvider) Chat(ctx context.Context, query, content, filepath, languageID string) (*ChatResponse, error) {
	time.Sleep(500 * time.Millisecond)
	return &ChatResponse{Result: "// fake improved result\nfake_improved();"}, nil
}
