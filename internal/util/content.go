package util

import (
	"strconv"
	"strings"
)

type ContentParts struct {
	ContentBefore           string
	ContentAfter            string
	LastCharacter           string
	LastLine                string
	ContentImmediatelyAfter string
}

func GetContent(contents string, line, column int) ContentParts {
	lines := strings.Split(contents, "\n")

	if line < 0 {
		line = 0
	}

	if line >= len(lines) {
		line = len(lines) - 1
	}

	beforeLines := make([]string, line+1)
	copy(beforeLines, lines[:line+1])

	if column >= 0 && column < len(beforeLines[line]) {
		beforeLines[line] = beforeLines[line][:column]
	}

	lastLine := beforeLines[len(beforeLines)-1]
	contentBefore := strings.Join(beforeLines, "\n")
	var contentAfter string

	if line+1 < len(lines) {
		contentAfter = strings.Join(lines[line+1:], "\n")
	}

	var lastCharacter string

	if len(contentBefore) > 0 {
		lastCharacter = string(contentBefore[len(contentBefore)-1])
	}

	var contentImmediatelyAfter string

	if line < len(lines) && column < len(lines[line]) {
		contentImmediatelyAfter = lines[line][column:]
	}

	return ContentParts{
		ContentBefore:           contentBefore,
		ContentAfter:            contentAfter,
		LastCharacter:           lastCharacter,
		LastLine:                lastLine,
		ContentImmediatelyAfter: contentImmediatelyAfter,
	}
}

func GetContentPadding(text string) int {
	lines := strings.Split(text, "\n")
	minPadding := 99999

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}

		padding := len(line) - len(strings.TrimLeft(line, " \t"))
		if padding < minPadding {
			minPadding = padding
		}
	}

	if minPadding == 99999 {
		return 0
	}
	return minPadding
}

func PadContent(text string, padding int) string {
	if padding <= 0 {
		return text
	}

	lines := strings.Split(text, "\n")
	paddingStr := strings.Repeat(" ", padding)

	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = paddingStr + line
		}
	}

	return strings.Join(lines, "\n")
}

func TruncateContext(before, after string, maxChars int) (string, string) {
	if maxChars <= 0 {
		return before, after
	}

	total := len(before) + len(after)
	if total <= maxChars {
		return before, after
	}

	excess := total - maxChars
	beforeLen := len(before)

	removeBefore := excess * beforeLen / total
	removeAfter := excess - removeBefore

	if removeBefore > 0 {
		start := removeBefore
		if idx := strings.Index(before[start:], "\n"); idx >= 0 {
			start += idx
		}
		before = "// ... [truncated " + strconv.Itoa(removeBefore) + " chars]\n" + before[start:]
	}

	if removeAfter > 0 {
		end := len(after) - removeAfter
		if end > 0 {
			beforeCut := after[:end]
			if idx := strings.LastIndex(beforeCut, "\n"); idx >= 0 {
				end = idx + 1
			}
		} else {
			end = 0
		}
		after = after[:end] + "\n// ... [truncated " + strconv.Itoa(removeAfter) + " chars]"
	}

	return before, after
}

func UniqueStrings(items []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(items))

	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
