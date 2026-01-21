package testing

import (
	"fmt"
	"path/filepath"
	"strings"
)

var extensionToLanguage = map[string]string{
	".js":   "javascript",
	".jsx":  "javascriptreact",
	".ts":   "typescript",
	".tsx":  "typescriptreact",
	".py":   "python",
	".go":   "go",
	".rs":   "rust",
	".java": "java",
	".c":    "c",
	".cpp":  "cpp",
	".cc":   "cpp",
	".cxx":  "cpp",
	".rb":   "ruby",
	".php":  "php",
	".cs":   "csharp",
	".sh":   "shellscript",
	".bash": "shellscript",
	".lua":  "lua",
	".vim":  "vim",
	".sql":  "sql",
	".html": "html",
	".css":  "css",
	".json": "json",
	".xml":  "xml",
	".yaml": "yaml",
	".yml":  "yaml",
	".md":   "markdown",
}

func DetectLanguage(filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return "", fmt.Errorf("file has no extension: %s", filename)
	}

	languageID, ok := extensionToLanguage[ext]
	if !ok {
		return "", fmt.Errorf("unsupported file extension: %s", ext)
	}

	return languageID, nil
}
