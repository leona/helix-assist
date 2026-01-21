package testing

import "time"

// TestCase represents a single completion test case parsed from a file
type TestCase struct {
	FilePath      string // Path to the test file
	LanguageID    string // Language identifier (e.g., "javascript", "python")
	ContentBefore string // Code before <CURSOR>
	ContentAfter  string // Code after <CURSOR>
	OriginalText  string // Full file with <CURSOR> marker
	CursorLine    int    // Line number of cursor (0-based)
	CursorColumn  int    // Column number of cursor (0-based)
}

// RunnerConfig holds configuration for the test runner
type RunnerConfig struct {
	Provider       string        // Provider name (e.g., "openai", "anthropic")
	NumSuggestions int           // Number of completions to request
	Timeout        time.Duration // Timeout for completion requests
}

// TestResult holds the result of running a single test
type TestResult struct {
	TestCase    *TestCase     // The test case that was run
	Suggestions []string      // Completion suggestions returned by provider
	Duration    time.Duration // How long the completion took
	Error       error         // Error if completion failed
}
