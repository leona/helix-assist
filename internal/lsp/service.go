package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Logger struct {
	mu      sync.Mutex
	file    *os.File
	enabled bool
}

func expandHome(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}

	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, path[1:]), nil
}

func NewLogger(path string) *Logger {
	l := &Logger{}

	if path != "" {
		expandedPath, err := expandHome(path)

		if err != nil {
			return l
		}

		dir := filepath.Dir(expandedPath)

		if err := os.MkdirAll(dir, 0755); err != nil {
			return l
		}

		f, err := os.OpenFile(expandedPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)

		if err == nil {
			l.file = f
			l.enabled = true
		}
	}
	return l
}

func (l *Logger) Log(args ...any) {
	if !l.enabled {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	parts := make([]string, 0, len(args)+3)
	parts = append(parts, "APP", time.Now().Format(time.RFC3339), "-->")

	for _, arg := range args {
		parts = append(parts, fmt.Sprintf("%v", arg))
	}

	l.file.WriteString(strings.Join(parts, " ") + "\n\n")
}

func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}

type EventHandler func(svc *Service, msg *JSONRPCMessage)

type Service struct {
	Buffers      *BufferStore
	Capabilities ServerCapabilities
	Logger       *Logger
	handlers     map[string][]EventHandler
	mu           sync.RWMutex
	stdin        io.Reader
	stdout       io.Writer
}

func NewService(capabilities ServerCapabilities, logger *Logger) *Service {
	svc := &Service{
		Buffers:      NewBufferStore(),
		Capabilities: capabilities,
		Logger:       logger,
		handlers:     make(map[string][]EventHandler),
		stdin:        os.Stdin,
		stdout:       os.Stdout,
	}
	svc.registerDefaultHandlers()
	return svc
}

func (s *Service) registerDefaultHandlers() {
	s.On(EventInitialize, func(svc *Service, msg *JSONRPCMessage) {
		svc.Send(&JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Result: InitializeResult{
				Capabilities: svc.Capabilities,
			},
		})
	})

	s.On(EventDidOpen, func(svc *Service, msg *JSONRPCMessage) {
		var params DidOpenParams
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			svc.Logger.Log("didOpen parse error:", err.Error())
			return
		}

		svc.Buffers.Set(&Buffer{
			URI:        params.TextDocument.URI,
			Text:       params.TextDocument.Text,
			LanguageID: params.TextDocument.LanguageID,
			Version:    params.TextDocument.Version,
		})

		svc.Logger.Log("received didOpen", "language:", params.TextDocument.LanguageID)
	})

	s.On(EventDidChange, func(svc *Service, msg *JSONRPCMessage) {
		var params DidChangeParams
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			svc.Logger.Log("didChange parse error:", err.Error())
			return
		}

		if len(params.ContentChanges) > 0 {
			svc.Buffers.UpdateText(
				params.TextDocument.URI,
				params.TextDocument.Version,
				params.ContentChanges[0].Text,
			)
		}

		svc.Logger.Log("received didChange", "version:", params.TextDocument.Version, "uri:", params.TextDocument.URI)
	})

	s.On(EventShutdown, func(svc *Service, msg *JSONRPCMessage) {
		svc.Logger.Log("received shutdown request")

		if msg.ID != nil {
			svc.Send(&JSONRPCMessage{
				JSONRPC: "2.0",
				ID:      msg.ID,
				Result:  nil,
			})
		}
	})

	s.On(EventExit, func(svc *Service, msg *JSONRPCMessage) {
		svc.Logger.Log("received exit notification")
		os.Exit(0)
	})
}

func (s *Service) On(method string, handler EventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = append(s.handlers[method], handler)
}

func (s *Service) emit(method string, msg *JSONRPCMessage) {
	s.mu.RLock()
	handlers := s.handlers[method]
	s.mu.RUnlock()

	for _, handler := range handlers {
		go func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					s.Logger.Log("handler panic:", method, r)
				}
			}()
			h(s, msg)
		}(handler)
	}
}

func (s *Service) Send(msg *JSONRPCMessage) {
	msg.JSONRPC = "2.0"
	data, err := json.Marshal(msg)

	if err != nil {
		s.Logger.Log("marshal error:", err.Error())
		return
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	s.mu.Lock()
	s.stdout.Write([]byte(header))
	s.stdout.Write(data)
	s.mu.Unlock()

	s.Logger.Log("sent:", string(data))
}

func (s *Service) SendDiagnostics(diagnostics []Diagnostic, timeoutMs int) {
	uri := s.Buffers.CurrentURI()
	if uri == "" {
		return
	}

	for i := range diagnostics {
		diagnostics[i].Source = "helix-gpt"
	}

	s.Logger.Log("sending diagnostics:", len(diagnostics))

	s.Send(&JSONRPCMessage{
		Method: EventPublishDiagnostics,
		Params: mustMarshal(PublishDiagnosticsParams{
			URI:         uri,
			Diagnostics: diagnostics,
		}),
	})

	if timeoutMs > 0 {
		go func() {
			time.Sleep(time.Duration(timeoutMs) * time.Millisecond)
			s.Send(&JSONRPCMessage{
				Method: EventPublishDiagnostics,
				Params: mustMarshal(PublishDiagnosticsParams{
					URI:         uri,
					Diagnostics: []Diagnostic{},
				}),
			})
		}()
	}
}

func (s *Service) ResetDiagnostics() {
	uri := s.Buffers.CurrentURI()

	if uri == "" {
		return
	}

	s.Send(&JSONRPCMessage{
		Method: EventPublishDiagnostics,
		Params: mustMarshal(PublishDiagnosticsParams{
			URI:         uri,
			Diagnostics: []Diagnostic{},
		}),
	})
}

func (s *Service) SendProgressBegin(token, title string) {
	s.Send(&JSONRPCMessage{
		Method: EventProgress,
		Params: mustMarshal(ProgressParams{
			Token: token,
			Value: WorkDoneProgressBegin{
				Kind:  "begin",
				Title: title,
			},
		}),
	})
}

func (s *Service) SendProgressReport(token, message string) {
	s.Send(&JSONRPCMessage{
		Method: EventProgress,
		Params: mustMarshal(ProgressParams{
			Token: token,
			Value: WorkDoneProgressReport{
				Kind:    "report",
				Message: message,
			},
		}),
	})
}

func (s *Service) SendProgressEnd(token string) {
	s.Send(&JSONRPCMessage{
		Method: EventProgress,
		Params: mustMarshal(ProgressParams{
			Token: token,
			Value: WorkDoneProgressEnd{
				Kind: "end",
			},
		}),
	})
}

func (s *Service) Start() error {
	reader := bufio.NewReader(s.stdin)

	for {
		contentLength := 0

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return fmt.Errorf("read header: %w", err)
			}

			line = strings.TrimSpace(line)
			if line == "" {
				break
			}

			if strings.HasPrefix(line, "Content-Length:") {
				lengthStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
				contentLength, _ = strconv.Atoi(lengthStr)
			}
		}

		if contentLength == 0 {
			continue
		}

		content := make([]byte, contentLength)
		_, err := io.ReadFull(reader, content)
		if err != nil {
			return fmt.Errorf("read content: %w", err)
		}

		var msg JSONRPCMessage
		if err := json.Unmarshal(content, &msg); err != nil {
			s.Logger.Log("parse error:", err.Error(), string(content))
			continue
		}

		if msg.Method != EventDidChange && msg.Method != EventDidOpen {
			s.Logger.Log("received:", string(content))
		}

		s.emit(msg.Method, &msg)
	}
}

func mustMarshal(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
