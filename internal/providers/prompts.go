package providers

import "fmt"

func BuildCompletionSystemPrompt(languageID string) string {
	return fmt.Sprintf(`You are a %s code completion assistant. Complete the code at the cursor position.

Rules:
- Output ONLY the code that should be inserted at the cursor
- Do NOT include any code that already exists before or after the cursor
- Do NOT add explanations, comments, or markdown formatting
- Do NOT repeat existing code
- Do NOT include comments
- Generate syntactically correct %s code`, languageID, languageID)
}

func BuildCompletionUserPrompt(filepath, contentBefore, contentAfter string) string {
	return fmt.Sprintf("File: %s\n\nCode before cursor:\n%s\n\n<CURSOR>\n\nCode after cursor:\n%s", filepath, contentBefore, contentAfter)
}

func BuildChatSystemPrompt(languageID string) string {
	return fmt.Sprintf(`You are an AI programming assistant specialized in %s.

Rules:
- Output ONLY the corrected/improved code that should replace the selection
- DO NOT include explanations, markdown formatting, or code block delimiters
- DO NOT include any text before or after the code
- DO NOT add extra comments unless specifically requested
- Preserve the original indentation and formatting style
- Generate syntactically correct %s code
- Follow the user's requirements precisely
- If diagnostics are provided, fix them in the code
- Remove any implementation comments after addressing them`, languageID, languageID)
}

func BuildChatUserPrompt(languageID, filepath, content, query string) string {
	return fmt.Sprintf(`File: %s
Language: %s

Selected code:
%s

Task: %s`, filepath, languageID, content, query)
}
