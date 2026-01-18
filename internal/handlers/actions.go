package handlers

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/leona/helix-assist/internal/config"
	"github.com/leona/helix-assist/internal/lsp"
	"github.com/leona/helix-assist/internal/providers"
	"github.com/leona/helix-assist/internal/util"
)

type CodeActionCommand struct {
	Key   string
	Label string
	Query string
}

// Commands available for code actions.
var Commands = []CodeActionCommand{
	{Key: "resolveDiagnostics", Label: "Resolve diagnostics", Query: "Resolve the diagnostics for this code."},
	{Key: "generateDocs", Label: "Generate documentation", Query: "Add documentation to this code."},
	{Key: "improveCode", Label: "Improve code", Query: "Improve this code."},
	{Key: "refactorFromComment", Label: "Refactor code from a comment", Query: "Refactor this code based on the comment."},
	{Key: "writeTest", Label: "Write a unit test", Query: "Write a unit test for this code. Do not include any imports."},
}

func CommandKeys() []string {
	keys := make([]string, len(Commands))
	for i, cmd := range Commands {
		keys[i] = cmd.Key
	}
	return keys
}

type ActionHandler struct {
	cfg      *config.Config
	registry *providers.Registry
}

func NewActionHandler(cfg *config.Config, registry *providers.Registry) *ActionHandler {
	return &ActionHandler{
		cfg:      cfg,
		registry: registry,
	}
}

func (h *ActionHandler) Register(svc *lsp.Service) {
	svc.On(lsp.EventCodeAction, func(svc *lsp.Service, msg *lsp.JSONRPCMessage) {
		var params lsp.CodeActionParams

		if err := json.Unmarshal(msg.Params, &params); err != nil {
			svc.Logger.Log("codeAction parse error:", err.Error())
			return
		}

		svc.Buffers.SetCurrentURI(params.TextDocument.URI)
		actions := make([]lsp.CodeAction, 0, len(Commands))

		for _, cmd := range Commands {
			diagnosticMsgs := make([]string, 0, len(params.Context.Diagnostics))

			for _, d := range params.Context.Diagnostics {
				diagnosticMsgs = append(diagnosticMsgs, d.Message)
			}

			actions = append(actions, lsp.CodeAction{
				Title:       cmd.Label,
				Kind:        "quickfix",
				Diagnostics: []any{},
				Command: &lsp.Command{
					Title:   cmd.Label,
					Command: cmd.Key,
					Arguments: []any{
						map[string]any{
							"range":       params.Range,
							"query":       cmd.Query,
							"diagnostics": diagnosticMsgs,
						},
					},
				},
			})
		}

		svc.Send(&lsp.JSONRPCMessage{
			ID:     msg.ID,
			Result: actions,
		})
	})

	svc.On(lsp.EventExecuteCommand, func(svc *lsp.Service, msg *lsp.JSONRPCMessage) {
		h.executeCommand(svc, msg)
	})
}

func (h *ActionHandler) executeCommand(svc *lsp.Service, msg *lsp.JSONRPCMessage) {
	defer func() {
		if r := recover(); r != nil {
			svc.Logger.Log("executeCommand panic:", r)
		}
	}()

	var params lsp.ExecuteCommandParams

	if err := json.Unmarshal(msg.Params, &params); err != nil {
		svc.Logger.Log("executeCommand parse error:", err.Error())
		return
	}

	if len(params.Arguments) == 0 {
		svc.Logger.Log("executeCommand: no arguments")
		return
	}

	argBytes, err := json.Marshal(params.Arguments[0])

	if err != nil {
		svc.Logger.Log("executeCommand: marshal arg error:", err.Error())
		return
	}

	var cmdArg lsp.CommandArgument

	if err := json.Unmarshal(argBytes, &cmdArg); err != nil {
		svc.Logger.Log("executeCommand: parse arg error:", err.Error())
		return
	}

	query := cmdArg.Query

	for _, cmd := range Commands {
		if cmd.Key == params.Command {
			if query == "" {
				query = cmd.Query
			}
			break
		}
	}

	currentURI := svc.Buffers.CurrentURI()

	if currentURI == "" {
		svc.Logger.Log("executeCommand: no current URI")
		return
	}

	var progress *util.ProgressIndicator

	if h.cfg.EnableProgressSpinner {
		progress = util.NewProgressIndicator(svc, h.cfg, cmdArg.Range, h.cfg.ActionTimeout)
		progress.Start()
		defer progress.Stop()
	} else {
		svc.SendDiagnostics([]lsp.Diagnostic{
			{
				Message:  "Executing " + params.Command + "...",
				Range:    cmdArg.Range,
				Severity: lsp.SeverityInformation,
			},
		}, h.cfg.ActionTimeout)
		defer svc.ResetDiagnostics()
	}

	content := svc.Buffers.GetContentFromRange(currentURI, cmdArg.Range)
	padding := util.GetContentPadding(content)

	buffer, ok := svc.Buffers.Get(currentURI)
	if !ok {
		svc.Logger.Log("executeCommand: buffer not found")
		return
	}

	svc.Logger.Log("chat request content:", content)

	if len(cmdArg.Diagnostics) > 0 {
		query += "\n\nDiagnostics: " + strings.Join(cmdArg.Diagnostics, "\n- ")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(h.cfg.ActionTimeout)*time.Millisecond)
	defer cancel()

	resp, err := h.registry.Chat(ctx, query, content, currentURI, buffer.LanguageID)
	if err != nil {
		svc.Logger.Log("chat failed:", err.Error())
		svc.SendDiagnostics([]lsp.Diagnostic{
			{
				Message:  err.Error(),
				Severity: lsp.SeverityError,
				Range:    cmdArg.Range,
			},
		}, h.cfg.ActionTimeout)
		return
	}

	if resp.Result == "" {
		svc.Logger.Log("chat: no completion found")
		svc.SendDiagnostics([]lsp.Diagnostic{
			{
				Message:  "No completion found",
				Severity: lsp.SeverityError,
				Range:    cmdArg.Range,
			},
		}, h.cfg.ActionTimeout)
		return
	}

	result := util.PadContent(strings.TrimSpace(resp.Result), padding) + "\n"
	svc.Logger.Log("received chat result:", result)

	svc.Send(&lsp.JSONRPCMessage{
		Method: lsp.EventApplyEdit,
		ID:     msg.ID,
		Params: mustMarshal(lsp.ApplyWorkspaceEditParams{
			Label: params.Command,
			Edit: lsp.WorkspaceEdit{
				Changes: map[string][]lsp.TextEdit{
					currentURI: {
						{
							Range:   cmdArg.Range,
							NewText: result,
						},
					},
				},
			},
		}),
	})

	svc.ResetDiagnostics()
}

func mustMarshal(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
