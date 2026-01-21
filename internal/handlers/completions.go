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
		h.sendEmptyCompletion(svc, msg.ID)
		return
	}

	content = util.GetContent(buffer.Text, params.Position.Line, params.Position.Character)
	svc.Logger.Log("calling completion", "language:", buffer.LanguageID)

	var progress *util.ProgressIndicator

	if h.cfg.EnableProgressSpinner {
		progress = util.NewProgressIndicator(svc, h.cfg)
		progress.Start()
		defer progress.Stop()
	} else {
		svc.SendShowMessage(lsp.MessageTypeInfo, "Fetching completion...")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(h.cfg.CompletionTimeout)*time.Millisecond)
	defer cancel()
	contentAfter := content.ContentImmediatelyAfter

	if content.ContentAfter != "" {
		if contentAfter != "" {
			contentAfter += "\n" + content.ContentAfter
		} else {
			contentAfter = content.ContentAfter
		}
	}

	hints, err := h.registry.Completion(ctx, providers.CompletionRequest{
		ContentBefore: content.ContentBefore,
		ContentAfter:  contentAfter,
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
		}, 0)
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
}

func findOverlapSuffix(hint, suffix string) int {
	if suffix == "" {
		return 0
	}

	hint = strings.TrimRight(hint, " \t")
	maxOverlap := len(hint)

	if len(suffix) < maxOverlap {
		maxOverlap = len(suffix)
	}

	for i := maxOverlap; i > 0; i-- {
		hintSuffix := hint[len(hint)-i:]
		suffixPrefix := suffix[:i]

		if hintSuffix == suffixPrefix {
			return i
		}
	}

	return 0
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

	overlapLen := findOverlapSuffix(hint, content.ContentImmediatelyAfter)

	var additionalEdits []lsp.TextEdit

	if overlapLen > 0 {
		additionalEdits = append(additionalEdits, lsp.TextEdit{
			Range: lsp.Range{
				Start: lsp.Position{Line: cleanLine, Character: cleanCharacter},
				End:   lsp.Position{Line: cleanLine, Character: cleanCharacter + overlapLen},
			},
			NewText: "",
		})
	} else if content.ContentImmediatelyAfter != "" {
		firstChar := content.ContentImmediatelyAfter[0]

		if firstChar == ')' || firstChar == '}' || firstChar == ']' || firstChar == '>' {
			restOfLine := content.ContentImmediatelyAfter[1:]
			isIsolated := len(content.ContentImmediatelyAfter) == 1 ||
				len(strings.TrimLeft(restOfLine, " \t")) == 0 ||
				restOfLine[0] == '\n' || restOfLine[0] == '\r'

			if isIsolated {
				additionalEdits = append(additionalEdits, lsp.TextEdit{
					Range: lsp.Range{
						Start: lsp.Position{Line: cleanLine, Character: cleanCharacter},
						End:   lsp.Position{Line: cleanLine, Character: cleanCharacter + 1},
					},
					NewText: "",
				})
			}
		}
	}

	return lsp.CompletionItem{
		Label:               label,
		Kind:                1,
		Preselect:           true,
		Detail:              hint,
		InsertText:          hint,
		InsertTextFormat:    1,
		SortText:            "00000",
		AdditionalTextEdits: additionalEdits,
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
