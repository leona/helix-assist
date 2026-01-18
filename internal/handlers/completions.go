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

type CompletionHandler struct {
	cfg       *config.Config
	registry  *providers.Registry
	debouncer *util.Debouncer
}

func NewCompletionHandler(cfg *config.Config, registry *providers.Registry) *CompletionHandler {
	return &CompletionHandler{
		cfg:       cfg,
		registry:  registry,
		debouncer: util.NewDebouncer(),
	}
}

func (h *CompletionHandler) Register(svc *lsp.Service) {
	svc.On(lsp.EventCompletion, func(svc *lsp.Service, msg *lsp.JSONRPCMessage) {
		var params lsp.CompletionParams
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			svc.Logger.Log("completion parse error:", err.Error())
			return
		}

		buffer, ok := svc.Buffers.Get(params.TextDocument.URI)
		if !ok {
			h.sendEmptyCompletion(svc, msg.ID)
			return
		}

		lastContentVersion := buffer.Version
		content := util.GetContent(buffer.Text, params.Position.Line, params.Position.Character)

		// Skip if last character is a dot (likely method/property access)
		if content.LastCharacter == "." {
			h.sendEmptyCompletion(svc, msg.ID)
			return
		}

		h.debouncer.Debounce("completion", func() {
			h.doCompletion(svc, msg, params, lastContentVersion, content)
		}, time.Duration(h.cfg.Debounce)*time.Millisecond)
	})
}

func (h *CompletionHandler) doCompletion(svc *lsp.Service, msg *lsp.JSONRPCMessage, params lsp.CompletionParams, lastContentVersion int, content util.ContentParts) {
	defer func() {
		if r := recover(); r != nil {
			svc.Logger.Log("completion panic:", r)
			h.sendEmptyCompletion(svc, msg.ID)
		}
	}()

	buffer, ok := svc.Buffers.Get(params.TextDocument.URI)
	if !ok {
		h.sendEmptyCompletion(svc, msg.ID)
		return
	}

	if buffer.Version > lastContentVersion {
		svc.Logger.Log("skipping completion - content is stale")
		svc.ResetDiagnostics()
		h.sendEmptyCompletion(svc, msg.ID)
		return
	}

	content = util.GetContent(buffer.Text, params.Position.Line, params.Position.Character)
	svc.Logger.Log("calling completion", "language:", buffer.LanguageID)

	diagRange := lsp.Range{
		Start: lsp.Position{Line: params.Position.Line, Character: 0},
		End:   lsp.Position{Line: params.Position.Line + 1, Character: 0},
	}

	var progress *util.ProgressIndicator

	if h.cfg.EnableProgressSpinner {
		progress = util.NewProgressIndicator(svc, h.cfg, diagRange, h.cfg.CompletionTimeout)
		progress.Start()
		defer progress.Stop()
	} else {
		svc.SendDiagnostics([]lsp.Diagnostic{
			{
				Message:  "Fetching completion...",
				Severity: lsp.SeverityInformation,
				Range:    diagRange,
			},
		}, h.cfg.CompletionTimeout)
		defer svc.ResetDiagnostics()
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(h.cfg.CompletionTimeout)*time.Millisecond)
	defer cancel()

	hints, err := h.registry.Completion(ctx, providers.CompletionRequest{
		ContentBefore: content.ContentBefore,
		ContentAfter:  content.ContentAfter,
	}, params.TextDocument.URI, buffer.LanguageID, h.cfg.NumSuggestions)

	if err != nil {
		svc.Logger.Log("completion error:", err.Error())
		svc.SendDiagnostics([]lsp.Diagnostic{
			{
				Message:  err.Error(),
				Severity: lsp.SeverityError,
				Range: lsp.Range{
					Start: lsp.Position{Line: params.Position.Line, Character: 0},
					End:   lsp.Position{Line: params.Position.Line + 1, Character: 0},
				},
			},
		}, h.cfg.CompletionTimeout)
		return
	}

	svc.Logger.Log("completion hints:", len(hints))

	items := make([]lsp.CompletionItem, 0, len(hints))
	for _, hint := range hints {
		item := h.buildCompletionItem(hint, content, params.Position)
		items = append(items, item)
	}

	svc.Send(&lsp.JSONRPCMessage{
		ID: msg.ID,
		Result: lsp.CompletionList{
			IsIncomplete: false,
			Items:        items,
		},
	})

	svc.ResetDiagnostics()
}

func (h *CompletionHandler) buildCompletionItem(hint string, content util.ContentParts, position lsp.Position) lsp.CompletionItem {
	hint = strings.TrimSpace(hint)

	lastLineTrimmed := strings.TrimSpace(content.LastLine)

	if strings.HasPrefix(hint, lastLineTrimmed) {
		hint = strings.TrimSpace(hint[len(lastLineTrimmed):])
	}

	lines := strings.Split(hint, "\n")
	cleanLine := position.Line + len(lines) - 1
	cleanCharacter := len(lines[len(lines)-1])

	if cleanLine == position.Line {
		cleanCharacter += position.Character
	}

	label := lines[0]

	if len(label) > 20 {
	} else if len(hint) > 20 {
		label = strings.TrimSpace(hint[:20])
	}

	return lsp.CompletionItem{
		Label:            label,
		Kind:             1,
		Preselect:        true,
		Detail:           hint,
		InsertText:       hint,
		InsertTextFormat: 1,
		SortText:         "00000",
		AdditionalTextEdits: []lsp.TextEdit{
			{
				Range: lsp.Range{
					Start: lsp.Position{Line: cleanLine, Character: cleanCharacter},
					End:   lsp.Position{Line: cleanLine, Character: cleanCharacter + len(content.ContentImmediatelyAfter)},
				},
				NewText: "",
			},
		},
	}
}

func (h *CompletionHandler) sendEmptyCompletion(svc *lsp.Service, id *int) {
	svc.Send(&lsp.JSONRPCMessage{
		ID: id,
		Result: lsp.CompletionList{
			IsIncomplete: false,
			Items:        []lsp.CompletionItem{},
		},
	})
}
