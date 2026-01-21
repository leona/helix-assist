package testing

import (
	"context"
	"fmt"
	"time"

	"github.com/leona/helix-assist/internal/providers"
)

type Runner struct {
	registry *providers.Registry
	config   *RunnerConfig
}

func NewRunner(registry *providers.Registry, config *RunnerConfig) *Runner {
	return &Runner{
		registry: registry,
		config:   config,
	}
}

func (r *Runner) RunTest(ctx context.Context, tc *TestCase) (*TestResult, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
	defer cancel()
	startTime := time.Now()

	req := providers.CompletionRequest{
		ContentBefore: tc.ContentBefore,
		ContentAfter:  tc.ContentAfter,
	}

	suggestions, err := r.registry.Completion(
		timeoutCtx,
		req,
		tc.FilePath,
		tc.LanguageID,
		r.config.NumSuggestions,
	)

	duration := time.Since(startTime)

	return &TestResult{
		TestCase:    tc,
		Suggestions: suggestions,
		Duration:    duration,
		Error:       err,
	}, nil
}

func (r *Runner) RunBatch(ctx context.Context, testCases []*TestCase) ([]*TestResult, error) {
	results := make([]*TestResult, 0, len(testCases))

	for _, tc := range testCases {
		result, err := r.RunTest(ctx, tc)
		if err != nil {
			return nil, fmt.Errorf("failed to run test %s: %w", tc.FilePath, err)
		}
		results = append(results, result)
	}

	return results, nil
}
