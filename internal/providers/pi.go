package providers

import "context"

type PiProvider struct {
	cli string
}

func NewPiProvider(cli string) *PiProvider {
	return &PiProvider{cli: cli}
}

func (p *PiProvider) Completion(ctx context.Context, req CompletionRequest, filepath, languageID string, numSuggestions int) ([]string, error) {
	// TODO: implement
	return nil, nil
}

func (p *PiProvider) Chat(ctx context.Context, query, content, filepath, languageID string) (*ChatResponse, error) {
	// TODO: implement
	return nil, nil
}
