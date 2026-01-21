package config

import (
	"flag"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Handler                string
	OpenAIKey              string
	OpenAIModel            string
	OpenAIModelForChat     string
	OpenAIEndpoint         string
	AnthropicKey           string
	AnthropicModel         string
	AnthropicModelForChat  string
	AnthropicEndpoint      string
	Debounce               int
	TriggerCharacters      []string
	NumSuggestions         int
	LogFile                string
	FetchTimeout           int
	ActionTimeout          int
	CompletionTimeout      int
	DebugQuery             string
	EnableProgressSpinner  bool
	ProgressUpdateInterval int
}

func DefaultConfig() *Config {
	return &Config{
		Handler:                "openai",
		OpenAIModel:            "gpt-4.1-mini",
		OpenAIModelForChat:     "gpt-5",
		OpenAIEndpoint:         "https://api.openai.com/v1",
		AnthropicModel:         "claude-haiku-4-5",
		AnthropicModelForChat:  "claude-sonnet-4-5",
		AnthropicEndpoint:      "https://api.anthropic.com",
		Debounce:               200,
		TriggerCharacters:      []string{"{", "(", " "},
		NumSuggestions:         1,
		FetchTimeout:           15000,
		ActionTimeout:          15000,
		CompletionTimeout:      15000,
		EnableProgressSpinner:  true,
		ProgressUpdateInterval: 200,
	}
}

// CLI flags take precedence over environment variables.
func Load() *Config {
	cfg := DefaultConfig()

	// Define flags
	handler := flag.String("handler", getEnvOrDefault("HANDLER", cfg.Handler), "Provider: openai or anthropic")
	openaiKey := flag.String("openai-key", getEnvOrDefault("OPENAI_API_KEY", ""), "OpenAI API key")
	openaiModel := flag.String("openai-model", getEnvOrDefault("OPENAI_MODEL", cfg.OpenAIModel), "OpenAI model")
	openaiEndpoint := flag.String("openai-endpoint", getEnvOrDefault("OPENAI_ENDPOINT", cfg.OpenAIEndpoint), "OpenAI API endpoint")
	anthropicKey := flag.String("anthropic-key", getEnvOrDefault("ANTHROPIC_API_KEY", ""), "Anthropic API key")
	anthropicModel := flag.String("anthropic-model", getEnvOrDefault("ANTHROPIC_MODEL", cfg.AnthropicModel), "Anthropic model")
	anthropicEndpoint := flag.String("anthropic-endpoint", getEnvOrDefault("ANTHROPIC_ENDPOINT", cfg.AnthropicEndpoint), "Anthropic API endpoint")
	openaiModelForChat := flag.String("openai-model-for-chat", getEnvOrDefault("OPENAI_MODEL_FOR_CHAT", cfg.OpenAIModelForChat), "OpenAI model for chat actions (defaults to openai-model)")
	anthropicModelForChat := flag.String("anthropic-model-for-chat", getEnvOrDefault("ANTHROPIC_MODEL_FOR_CHAT", cfg.AnthropicModelForChat), "Anthropic model for chat actions (defaults to anthropic-model)")
	debounce := flag.Int("debounce", getEnvOrDefaultInt("DEBOUNCE", cfg.Debounce), "Debounce delay (ms)")
	triggerChars := flag.String("trigger-chars", getEnvOrDefault("TRIGGER_CHARACTERS", "{||(|| "), "Completion trigger characters (separated by ||)")
	numSuggestions := flag.Int("num-suggestions", getEnvOrDefaultInt("NUM_SUGGESTIONS", cfg.NumSuggestions), "Number of suggestions")
	logFile := flag.String("log-file", getEnvOrDefault("LOG_FILE", "~/.cache/helix-assist.log"), "Log file path")
	fetchTimeout := flag.Int("fetch-timeout", getEnvOrDefaultInt("FETCH_TIMEOUT", cfg.FetchTimeout), "Fetch timeout (ms)")
	actionTimeout := flag.Int("action-timeout", getEnvOrDefaultInt("ACTION_TIMEOUT", cfg.ActionTimeout), "Action timeout (ms)")
	completionTimeout := flag.Int("completion-timeout", getEnvOrDefaultInt("COMPLETION_TIMEOUT", cfg.CompletionTimeout), "Completion timeout (ms)")
	debugQuery := flag.String("debug-query", "", "Debug mode: test provider with a query and exit")
	enableProgressSpinner := flag.Bool("enable-progress-spinner", getEnvOrDefaultBool("ENABLE_PROGRESS_SPINNER", cfg.EnableProgressSpinner), "Enable animated progress spinner")
	progressUpdateInterval := flag.Int("progress-update-interval", getEnvOrDefaultInt("PROGRESS_UPDATE_INTERVAL", cfg.ProgressUpdateInterval), "Progress update interval (ms)")

	flag.Parse()

	cfg.Handler = *handler
	cfg.OpenAIKey = *openaiKey
	cfg.OpenAIModel = *openaiModel
	cfg.OpenAIModelForChat = *openaiModelForChat
	cfg.OpenAIEndpoint = *openaiEndpoint
	cfg.AnthropicKey = *anthropicKey
	cfg.AnthropicModel = *anthropicModel
	cfg.AnthropicModelForChat = *anthropicModelForChat
	cfg.AnthropicEndpoint = *anthropicEndpoint
	cfg.Debounce = *debounce
	cfg.TriggerCharacters = strings.Split(*triggerChars, "||")
	cfg.NumSuggestions = *numSuggestions
	cfg.LogFile = *logFile
	cfg.FetchTimeout = *fetchTimeout
	cfg.ActionTimeout = *actionTimeout
	cfg.CompletionTimeout = *completionTimeout
	cfg.DebugQuery = *debugQuery
	cfg.EnableProgressSpinner = *enableProgressSpinner
	cfg.ProgressUpdateInterval = *progressUpdateInterval

	return cfg
}

func (c *Config) Validate() error {
	if c.Handler != "openai" && c.Handler != "anthropic" {
		return &ConfigError{Message: "handler must be 'openai' or 'anthropic'"}
	}

	if c.Handler == "openai" && c.OpenAIKey == "" {
		return &ConfigError{Message: "OpenAI API key is required when using openai handler"}
	}

	if c.Handler == "anthropic" && c.AnthropicKey == "" {
		return &ConfigError{Message: "Anthropic API key is required when using anthropic handler"}
	}

	return nil
}

type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string {
	return e.Message
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvOrDefaultInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvOrDefaultBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
