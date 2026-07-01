package providers

import "context"

type FakeProvider struct{}

func NewFakeProvider() *FakeProvider {
	return &FakeProvider{}
}

func (p *FakeProvider) Completion(ctx context.Context, req CompletionRequest, filepath, languageID string, numSuggestions int) ([]string, error) {
	// TODO: implement
	return nil, nil
}

func (p *FakeProvider) Chat(ctx context.Context, query, content, filepath, languageID string) (*ChatResponse, error) {
	// TODO: implement
	return nil, nil
}
