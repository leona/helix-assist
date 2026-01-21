package providers

import "fmt"

func BuildCompletionSystemPrompt(languageID string) string {
	return fmt.Sprintf(`You are a %s code completion assistant. Complete the code at the cursor position.

Rules:
- Output ONLY the code that should be inserted at the cursor position
- Do NOT include any code that already exists before or after the cursor
- Do NOT add explanations, comments, or markdown formatting
- Do NOT repeat existing code
- Do NOT include comments
- Generate syntactically correct %s code that fits seamlessly between the before and after content

Context awareness:
- CAREFULLY examine the code after the cursor - it shows what already exists
- If the code after cursor contains closing delimiters (}, ), ], etc.), DO NOT add them again
- If you're completing in the middle of a statement (e.g., inside an object, array, or parameter list), complete ONLY the current item
- When the code after cursor shows more content in the same block, DO NOT close that block
- Only add closing delimiters if they are NOT already present in the code after cursor

Completion style:
- Prefer multi-line completions that form complete, meaningful additions
- Provide meaningful placeholder values or expressions where appropriate
- When completing control structures that are NOT yet closed in the after-cursor code, provide complete blocks with braces`, languageID, languageID)
}

func BuildCompletionUserPrompt(filepath, contentBefore, contentAfter string) string {
	return fmt.Sprintf(`File: %s

Code before cursor:
%s

<CURSOR>

Code after cursor (DO NOT duplicate or close delimiters that already exist here):
%s

Complete the code at the <CURSOR> position. The completion must fit seamlessly between the before and after sections.`, filepath, contentBefore, contentAfter)
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
