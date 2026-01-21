package testing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const cursorMarker = "<CURSOR>"

func ParseTestFile(path string) (*TestCase, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	originalText := string(content)
	cursorIndex := strings.Index(originalText, cursorMarker)

	if cursorIndex == -1 {
		return nil, fmt.Errorf("file does not contain %s marker", cursorMarker)
	}

	textBeforeMarker := originalText[:cursorIndex]
	line := strings.Count(textBeforeMarker, "\n")
	lastNewline := strings.LastIndex(textBeforeMarker, "\n")
	column := cursorIndex - lastNewline - 1
	contentBefore := originalText[:cursorIndex]
	contentAfter := originalText[cursorIndex+len(cursorMarker):]
	languageID, err := DetectLanguage(path)

	if err != nil {
		return nil, fmt.Errorf("failed to detect language: %w", err)
	}

	return &TestCase{
		FilePath:      path,
		LanguageID:    languageID,
		ContentBefore: contentBefore,
		ContentAfter:  contentAfter,
		OriginalText:  originalText,
		CursorLine:    line,
		CursorColumn:  column,
	}, nil
}

func LoadTestCases(dir string, languageFilter string) ([]*TestCase, error) {
	var testCases []*TestCase

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		tc, err := ParseTestFile(path)

		if err != nil {
			return nil
		}

		if languageFilter != "" && tc.LanguageID != languageFilter {
			return nil
		}

		testCases = append(testCases, tc)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	if len(testCases) == 0 {
		return nil, fmt.Errorf("no test cases found in directory: %s", dir)
	}

	return testCases, nil
}
