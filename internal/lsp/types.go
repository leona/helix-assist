package lsp

import "encoding/json"

const (
	EventDidOpen            = "textDocument/didOpen"
	EventDidChange          = "textDocument/didChange"
	EventCompletion         = "textDocument/completion"
	EventCodeAction         = "textDocument/codeAction"
	EventApplyEdit          = "workspace/applyEdit"
	EventExecuteCommand     = "workspace/executeCommand"
	EventInitialize         = "initialize"
	EventInitialized        = "initialized"
	EventShutdown           = "shutdown"
	EventExit               = "exit"
	EventPublishDiagnostics = "textDocument/publishDiagnostics"
	EventProgress           = "$/progress"
	EventShowMessage        = "window/showMessage"
)

type WorkDoneProgressBegin struct {
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Message     string `json:"message,omitempty"`
	Percentage  int    `json:"percentage,omitempty"`
	Cancellable bool   `json:"cancellable,omitempty"`
}

type WorkDoneProgressReport struct {
	Kind       string `json:"kind"`
	Message    string `json:"message,omitempty"`
	Percentage int    `json:"percentage,omitempty"`
}

type WorkDoneProgressEnd struct {
	Kind    string `json:"kind"`
	Message string `json:"message,omitempty"`
}

type ProgressParams struct {
	Token string `json:"token"`
	Value any    `json:"value"`
}

type MessageType int

const (
	MessageTypeError   MessageType = 1
	MessageTypeWarning MessageType = 2
	MessageTypeInfo    MessageType = 3
	MessageTypeLog     MessageType = 4
)

type DiagnosticSeverity int

const (
	SeverityError       DiagnosticSeverity = 1
	SeverityWarning     DiagnosticSeverity = 2
	SeverityInformation DiagnosticSeverity = 3
	SeverityHint        DiagnosticSeverity = 4
)

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Diagnostic struct {
	Message  string             `json:"message"`
	Range    Range              `json:"range"`
	Source   string             `json:"source,omitempty"`
	Severity DiagnosticSeverity `json:"severity,omitempty"`
}

type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type InitializeParams struct {
	ProcessID    int    `json:"processId"`
	RootURI      string `json:"rootUri"`
	Capabilities any    `json:"capabilities"`
}

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier
	Version int `json:"version"`
}

type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type DidOpenParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type ContentChange struct {
	Text string `json:"text"`
}

type DidChangeParams struct {
	TextDocument   VersionedTextDocumentIdentifier `json:"textDocument"`
	ContentChanges []ContentChange                 `json:"contentChanges"`
}

type CompletionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type CompletionItem struct {
	Label               string     `json:"label"`
	Kind                int        `json:"kind,omitempty"`
	Detail              string     `json:"detail,omitempty"`
	InsertText          string     `json:"insertText,omitempty"`
	InsertTextFormat    int        `json:"insertTextFormat,omitempty"`
	SortText            string     `json:"sortText,omitempty"`
	Preselect           bool       `json:"preselect,omitempty"`
	AdditionalTextEdits []TextEdit `json:"additionalTextEdits,omitempty"`
}

type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

type CodeActionContext struct {
	Diagnostics []Diagnostic `json:"diagnostics"`
}

type CodeActionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Context      CodeActionContext      `json:"context"`
}

type Command struct {
	Title     string `json:"title"`
	Command   string `json:"command"`
	Arguments []any  `json:"arguments,omitempty"`
}

type CodeAction struct {
	Title       string   `json:"title"`
	Kind        string   `json:"kind,omitempty"`
	Diagnostics []any    `json:"diagnostics,omitempty"`
	Command     *Command `json:"command,omitempty"`
}

type ExecuteCommandParams struct {
	Command   string `json:"command"`
	Arguments []any  `json:"arguments"`
}

type CommandArgument struct {
	Range       Range    `json:"range"`
	Query       string   `json:"query"`
	Diagnostics []string `json:"diagnostics,omitempty"`
}

type WorkspaceEdit struct {
	Changes map[string][]TextEdit `json:"changes"`
}

type ApplyWorkspaceEditParams struct {
	Label string        `json:"label"`
	Edit  WorkspaceEdit `json:"edit"`
}

type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

type ShowMessageParams struct {
	Type    MessageType `json:"type"`
	Message string      `json:"message"`
}

type ServerCapabilities struct {
	TextDocumentSync       int                    `json:"textDocumentSync"`
	CompletionProvider     *CompletionOptions     `json:"completionProvider,omitempty"`
	CodeActionProvider     bool                   `json:"codeActionProvider,omitempty"`
	ExecuteCommandProvider *ExecuteCommandOptions `json:"executeCommandProvider,omitempty"`
}

type CompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

type ExecuteCommandOptions struct {
	Commands []string `json:"commands"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
}
