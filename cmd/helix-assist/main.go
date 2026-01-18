package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/leona/helix-assist/internal/config"
	"github.com/leona/helix-assist/internal/handlers"
	"github.com/leona/helix-assist/internal/lsp"
	"github.com/leona/helix-assist/internal/providers"
)

func main() {
	cfg := config.Load()

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %s\n", err.Error())
		os.Exit(1)
	}

	logger := lsp.NewLogger(cfg.LogFile)
	defer logger.Close()
	logger.Log("Starting helix-assist", "handler:", cfg.Handler)
	logger.Log("triggerCharacters:", cfg.TriggerCharacters)
	registry := providers.NewRegistry()

	if cfg.OpenAIKey != "" {
		openaiProvider := providers.NewOpenAIProvider(
			cfg.OpenAIKey,
			cfg.OpenAIModel,
			cfg.OpenAIEndpoint,
			cfg.FetchTimeout,
		)
		registry.Register("openai", openaiProvider)
		logger.Log("Registered OpenAI provider", "model:", cfg.OpenAIModel)
	}

	if cfg.AnthropicKey != "" {
		anthropicProvider := providers.NewAnthropicProvider(
			cfg.AnthropicKey,
			cfg.AnthropicModel,
			cfg.AnthropicEndpoint,
			cfg.FetchTimeout,
		)
		registry.Register("anthropic", anthropicProvider)
		logger.Log("Registered Anthropic provider", "model:", cfg.AnthropicModel)
	}

	if err := registry.SetCurrent(cfg.Handler); err != nil {
		fmt.Fprintf(os.Stderr, "Provider error: %s\n", err.Error())
		os.Exit(1)
	}

	if cfg.DebugQuery != "" {
		logger.Log("Debug mode: testing provider with query:", cfg.DebugQuery)
		debugMode(cfg, registry, logger)
		return
	}

	capabilities := lsp.ServerCapabilities{
		TextDocumentSync: 1,
		CompletionProvider: &lsp.CompletionOptions{
			TriggerCharacters: cfg.TriggerCharacters,
		},
		CodeActionProvider: true,
		ExecuteCommandProvider: &lsp.ExecuteCommandOptions{
			Commands: handlers.CommandKeys(),
		},
	}

	svc := lsp.NewService(capabilities, logger)
	completionHandler := handlers.NewCompletionHandler(cfg, registry)
	completionHandler.Register(svc)
	actionHandler := handlers.NewActionHandler(cfg, registry)
	actionHandler.Register(svc)
	logger.Log("LSP service initialized, listening on stdin")

	if err := svc.Start(); err != nil {
		logger.Log("LSP service error:", err.Error())
		os.Exit(1)
	}
}

func debugMode(cfg *config.Config, registry *providers.Registry, logger *lsp.Logger) {
	ctx := context.Background()
	logger.Log("Calling completion with query:", cfg.DebugQuery)
	fmt.Printf("Query: %s\n", cfg.DebugQuery)
	fmt.Printf("Provider: %s\n", cfg.Handler)
	fmt.Printf("Num suggestions: %d\n", cfg.NumSuggestions)
	fmt.Println(strings.Repeat("-", 80))
	fmt.Println("Sending request...")

	results, err := registry.Completion(ctx, providers.CompletionRequest{
		ContentBefore: cfg.DebugQuery,
		ContentAfter:  "",
	}, "debug.js", "javascript", cfg.NumSuggestions)

	if err != nil {
		logger.Log("Completion error:", err.Error())
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	logger.Log("Received", len(results), "completions")
	fmt.Printf("\nReceived %d completion(s):\n\n", len(results))

	for i, result := range results {
		fmt.Printf("--- Suggestion %d ---\n", i+1)
		fmt.Println(result)
		fmt.Println()
	}
}
