package providers

import (
	"context"
	"fmt"
	"sync"
)

type CompletionRequest struct {
	ContentBefore string
	ContentAfter  string
}

type ChatResponse struct {
	Result string
}

type Provider interface {
	Completion(ctx context.Context, req CompletionRequest, filepath, languageID string, numSuggestions int) ([]string, error)
	Chat(ctx context.Context, query, content, filepath, languageID string) (*ChatResponse, error)
}

type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
	current   string
}

func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

func (r *Registry) Register(name string, provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[name] = provider
}

func (r *Registry) SetCurrent(name string) error {
	r.mu.RLock()
	_, ok := r.providers[name]
	r.mu.RUnlock()

	if !ok {
		return fmt.Errorf("provider not found: %s", name)
	}

	r.mu.Lock()
	r.current = name
	r.mu.Unlock()
	return nil
}

func (r *Registry) Get() (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.current == "" {
		return nil, fmt.Errorf("no provider configured")
	}

	provider, ok := r.providers[r.current]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", r.current)
	}

	return provider, nil
}

func (r *Registry) Completion(ctx context.Context, req CompletionRequest, filepath, languageID string, numSuggestions int) ([]string, error) {
	provider, err := r.Get()
	if err != nil {
		return nil, err
	}
	return provider.Completion(ctx, req, filepath, languageID, numSuggestions)
}

func (r *Registry) Chat(ctx context.Context, query, content, filepath, languageID string) (*ChatResponse, error) {
	provider, err := r.Get()
	if err != nil {
		return nil, err
	}
	return provider.Chat(ctx, query, content, filepath, languageID)
}
