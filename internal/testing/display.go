package testing

import (
	"fmt"
	"strings"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorGray   = "\033[90m"
	colorRed    = "\033[31m"
)

type Formatter struct {
	useColor bool
}

func NewFormatter(useColor bool) *Formatter {
	return &Formatter{
		useColor: useColor,
	}
}

func (f *Formatter) color(code, text string) string {
	if !f.useColor {
		return text
	}
	return code + text + colorReset
}

func (f *Formatter) FormatResult(result *TestResult, providerName string) string {
	var sb strings.Builder

	sb.WriteString(f.color(colorYellow, "================================================================================\n"))
	sb.WriteString(f.color(colorYellow, fmt.Sprintf("Test: %s\n", result.TestCase.FilePath)))
	sb.WriteString(f.color(colorYellow, fmt.Sprintf("Language: %s\n", result.TestCase.LanguageID)))
	sb.WriteString(f.color(colorYellow, fmt.Sprintf("Provider: %s\n", providerName)))

	if result.Error != nil {
		sb.WriteString(f.color(colorRed, fmt.Sprintf("Error: %v\n", result.Error)))
	} else {
		sb.WriteString(f.color(colorYellow, fmt.Sprintf("Duration: %dms\n", result.Duration.Milliseconds())))
	}

	sb.WriteString(f.color(colorYellow, "================================================================================\n\n"))

	if result.Error == nil && len(result.Suggestions) > 0 {
		completion := result.Suggestions[0]

		codeWithCompletion := result.TestCase.ContentBefore +
			f.color(colorGreen, completion) +
			f.color(colorGray, " ← [COMPLETION]") +
			result.TestCase.ContentAfter

		sb.WriteString(codeWithCompletion)
		sb.WriteString("\n\n")

		if len(result.Suggestions) > 1 {
			sb.WriteString(fmt.Sprintf("Suggestions (%d):\n", len(result.Suggestions)))
		} else {
			sb.WriteString("Suggestion (1):\n")
		}

		for i, suggestion := range result.Suggestions {
			lines := strings.Split(suggestion, "\n")
			if len(lines) == 1 {
				sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, suggestion))
			} else {
				sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, lines[0]))
				for _, line := range lines[1:] {
					sb.WriteString(fmt.Sprintf("     %s\n", line))
				}
			}
		}
	} else if result.Error == nil {
		sb.WriteString(f.color(colorGray, "No suggestions returned\n"))
	}

	sb.WriteString("\n")

	return sb.String()
}

func (f *Formatter) FormatBatch(results []*TestResult, providerName string) string {
	var sb strings.Builder

	for _, result := range results {
		sb.WriteString(f.FormatResult(result, providerName))
	}

	sb.WriteString(f.color(colorYellow, "================================================================================\n"))
	sb.WriteString(f.color(colorYellow, "Summary\n"))
	sb.WriteString(f.color(colorYellow, "================================================================================\n"))

	totalTests := len(results)
	completedTests := 0
	errorTests := 0
	var totalDuration time.Duration

	for _, result := range results {
		if result.Error != nil {
			errorTests++
		} else {
			completedTests++
			totalDuration += result.Duration
		}
	}

	sb.WriteString(fmt.Sprintf("  Total tests: %d\n", totalTests))
	sb.WriteString(fmt.Sprintf("  Completed: %s\n", f.color(colorGreen, fmt.Sprintf("%d", completedTests))))
	if errorTests > 0 {
		sb.WriteString(fmt.Sprintf("  Errors: %s\n", f.color(colorRed, fmt.Sprintf("%d", errorTests))))
	} else {
		sb.WriteString(fmt.Sprintf("  Errors: %d\n", errorTests))
	}

	if completedTests > 0 {
		avgDuration := totalDuration / time.Duration(completedTests)
		sb.WriteString(fmt.Sprintf("  Average duration: %dms\n", avgDuration.Milliseconds()))
	}

	sb.WriteString("\n")

	return sb.String()
}
