package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/leona/helix-assist/internal/lsp"
	"github.com/leona/helix-assist/internal/providers"
	testing "github.com/leona/helix-assist/internal/testing"
)

func main() {
	testDir := flag.String("testdir", "", "Directory containing test files")
	testFile := flag.String("file", "", "Single test file to run")
	provider := flag.String("provider", "openai", "Provider to use (openai or anthropic)")
	language := flag.String("language", "", "Filter tests by language (optional)")
	numSuggestions := flag.Int("num-suggestions", 1, "Number of completions to request")
	timeoutMs := flag.Int("timeout", 15000, "Completion timeout in milliseconds")
	noColor := flag.Bool("no-color", false, "Disable colored output")

	openaiKey := flag.String("openai-key", os.Getenv("OPENAI_API_KEY"), "OpenAI API key")
	openaiModel := flag.String("openai-model", getEnvOrDefault("OPENAI_MODEL", "gpt-4.1-mini"), "OpenAI model")
	openaiEndpoint := flag.String("openai-endpoint", getEnvOrDefault("OPENAI_ENDPOINT", "https://api.openai.com/v1"), "OpenAI API endpoint")

	anthropicKey := flag.String("anthropic-key", os.Getenv("ANTHROPIC_API_KEY"), "Anthropic API key")
	anthropicModel := flag.String("anthropic-model", getEnvOrDefault("ANTHROPIC_MODEL", "claude-sonnet-4-5"), "Anthropic model")
	anthropicEndpoint := flag.String("anthropic-endpoint", getEnvOrDefault("ANTHROPIC_ENDPOINT", "https://api.anthropic.com"), "Anthropic API endpoint")

	flag.Parse()

	if *testDir == "" && *testFile == "" {
		fmt.Fprintf(os.Stderr, "Error: Either --testdir or --file must be specified\n")
		flag.Usage()
		os.Exit(1)
	}

	if *testDir != "" && *testFile != "" {
		fmt.Fprintf(os.Stderr, "Error: Cannot specify both --testdir and --file\n")
		flag.Usage()
		os.Exit(1)
	}

	if *provider != "openai" && *provider != "anthropic" {
		fmt.Fprintf(os.Stderr, "Error: Provider must be 'openai' or 'anthropic'\n")
		os.Exit(1)
	}

	logger := lsp.NewLogger("/dev/null")
	defer logger.Close()

	registry := providers.NewRegistry()

	if *provider == "openai" {
		if *openaiKey == "" {
			fmt.Fprintf(os.Stderr, "Error: OpenAI API key is required. Set OPENAI_API_KEY or use --openai-key\n")
			os.Exit(1)
		}
		openaiProvider := providers.NewOpenAIProvider(
			*openaiKey,
			*openaiModel,
			*openaiEndpoint,
			*timeoutMs,
			logger,
		)
		registry.Register("openai", openaiProvider)
		if err := registry.SetCurrent("openai"); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else if *provider == "anthropic" {
		if *anthropicKey == "" {
			fmt.Fprintf(os.Stderr, "Error: Anthropic API key is required. Set ANTHROPIC_API_KEY or use --anthropic-key\n")
			os.Exit(1)
		}
		anthropicProvider := providers.NewAnthropicProvider(
			*anthropicKey,
			*anthropicModel,
			*anthropicEndpoint,
			*timeoutMs,
			logger,
		)
		registry.Register("anthropic", anthropicProvider)
		if err := registry.SetCurrent("anthropic"); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	var testCases []*testing.TestCase
	var err error

	if *testFile != "" {
		testCase, err := testing.ParseTestFile(*testFile)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing test file: %v\n", err)
			os.Exit(1)
		}
		testCases = []*testing.TestCase{testCase}
	} else {
		testCases, err = testing.LoadTestCases(*testDir, *language)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading test cases: %v\n", err)
			os.Exit(1)
		}
	}

	log.Println("running tests")

	runnerConfig := &testing.RunnerConfig{
		Provider:       *provider,
		NumSuggestions: *numSuggestions,
		Timeout:        time.Duration(*timeoutMs) * time.Millisecond,
	}
	runner := testing.NewRunner(registry, runnerConfig)

	ctx := context.Background()
	results, err := runner.RunBatch(ctx, testCases)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running tests: %v\n", err)
		os.Exit(1)
	}

	formatter := testing.NewFormatter(!*noColor)
	output := formatter.FormatBatch(results, *provider)
	fmt.Print(output)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
