package testing

import "time"

type TestCase struct {
	FilePath      string
	LanguageID    string
	ContentBefore string
	ContentAfter  string
	OriginalText  string
	CursorLine    int
	CursorColumn  int
}

type RunnerConfig struct {
	Provider       string
	NumSuggestions int
	Timeout        time.Duration
}

type TestResult struct {
	TestCase    *TestCase
	Suggestions []string
	Duration    time.Duration
	Error       error
}
