package util

import (
	"regexp"
	"strings"
)

func ExtractCodeBlock(filepath, text, language string) string {
	pattern := regexp.MustCompile("```" + regexp.QuoteMeta(language) + `([\s\S]*?)` + "```")
	matches := pattern.FindAllString(text, -1)

	if len(matches) == 0 {
		return ""
	}

	result := matches[0]

	cleanFilepath := strings.TrimPrefix(filepath, "file://")
	result = strings.Replace(result, "// FILEPATH: "+cleanFilepath+"\n", "", 1)

	lines := strings.Split(result, "\n")
	if len(lines) < 2 {
		return ""
	}

	contentLines := lines[1 : len(lines)-1]
	return strings.Join(contentLines, "\n") + "\n"
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
