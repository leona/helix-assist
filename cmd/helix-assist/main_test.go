package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/leona/helix-assist/internal/lsp"
)

var runLive = flag.Bool("live", false, "run live LLM tests")

type testClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	nextID int
}

func buildBinary(t *testing.T) string {
	t.Helper()
	mod, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		t.Fatalf("go env GOMOD: %v", err)
	}
	root := filepath.Dir(strings.TrimSpace(string(mod)))
	bin := filepath.Join(t.TempDir(), "helix-assist")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/helix-assist")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

func newClient(t *testing.T, bin string, args ...string) *testClient {
	t.Helper()
	cmd := exec.Command(bin, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	c := &testClient{cmd: cmd, stdin: stdin, stdout: bufio.NewReader(stdout), nextID: 1}
	t.Cleanup(func() {
		c.shutdown(t)
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	return c
}

func (c *testClient) request(t *testing.T, method string, params any) lsp.JSONRPCMessage {
	t.Helper()
	id := c.nextID
	c.nextID++
	msg := lsp.JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
		Params:  mustMarshal(params),
	}
	c.send(t, msg)
	return c.waitFor(t, id)
}

func (c *testClient) notify(t *testing.T, method string, params any) {
	t.Helper()
	msg := lsp.JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  method,
		Params:  mustMarshal(params),
	}
	c.send(t, msg)
}

func (c *testClient) send(t *testing.T, msg lsp.JSONRPCMessage) {
	t.Helper()
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	if _, err := c.stdin.Write([]byte(header)); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if _, err := c.stdin.Write(data); err != nil {
		t.Fatalf("write body: %v", err)
	}
}

func (c *testClient) waitFor(t *testing.T, id int) lsp.JSONRPCMessage {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		msg := c.readMessage(t)
		if msg.ID != nil && *msg.ID == id {
			return msg
		}
	}
	t.Fatalf("timeout waiting for response id %d", id)
	return lsp.JSONRPCMessage{}
}

func (c *testClient) readMessage(t *testing.T) lsp.JSONRPCMessage {
	t.Helper()
	var length int
	for {
		line, err := c.stdout.ReadString('\n')
		if err != nil {
			t.Fatalf("read header: %v", err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length:") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			length, _ = strconv.Atoi(v)
		}
	}
	if length == 0 {
		return c.readMessage(t)
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(c.stdout, data); err != nil {
		t.Fatalf("read content: %v", err)
	}
	var msg lsp.JSONRPCMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return msg
}

func (c *testClient) shutdown(t *testing.T) {
	if c.stdin == nil {
		return
	}
	_ = c.request(t, "shutdown", nil)
	c.notify(t, "exit", nil)
	_ = c.stdin.Close()
	_ = c.cmd.Wait()
	c.stdin = nil
}

func mustMarshal(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

func parseResult[T any](t *testing.T, result any) T {
	t.Helper()
	raw, _ := json.Marshal(result)
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	return v
}

func testURI(t *testing.T) string {
	return fmt.Sprintf("file://%s/test.rs", t.TempDir())
}

func initClient(t *testing.T, c *testClient) {
	t.Helper()
	resp := c.request(t, "initialize", lsp.InitializeParams{
		ProcessID: 1,
		RootURI:   "file:///tmp/test",
		Capabilities: map[string]any{
			"textDocument": map[string]any{},
		},
	})
	if resp.Result == nil {
		t.Fatal("initialize returned no result")
	}
	c.notify(t, "initialized", map[string]any{})
}

func didOpen(t *testing.T, c *testClient, uri string) {
	t.Helper()
	c.notify(t, "textDocument/didOpen", lsp.DidOpenParams{
		TextDocument: lsp.TextDocumentItem{
			URI:        uri,
			LanguageID: "rust",
			Version:    1,
			Text:       "fn main() {\n\n}\n",
		},
	})
}

func didChange(t *testing.T, c *testClient, uri string) {
	t.Helper()
	c.notify(t, "textDocument/didChange", lsp.DidChangeParams{
		TextDocument: lsp.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: lsp.TextDocumentIdentifier{URI: uri},
			Version:                2,
		},
		ContentChanges: []lsp.ContentChange{{Text: "fn main() {\n\n}\n"}},
	})
}

func TestFakeFlow(t *testing.T) {
	bin := buildBinary(t)
	c := newClient(t, bin, "--handler", "fake", "--log-file", "/dev/null")
	uri := testURI(t)

	initClient(t, c)
	didOpen(t, c, uri)
	didChange(t, c, uri)

	comp := c.request(t, "textDocument/completion", lsp.CompletionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: uri},
		Position:     lsp.Position{Line: 0, Character: 11},
	})
	if comp.Result == nil {
		t.Fatal("completion returned no result")
	}
	list := parseResult[lsp.CompletionList](t, comp.Result)
	if len(list.Items) == 0 {
		t.Fatal("expected completion items")
	}

	action := c.request(t, "textDocument/codeAction", lsp.CodeActionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: uri},
		Range: lsp.Range{
			Start: lsp.Position{Line: 0, Character: 0},
			End:   lsp.Position{Line: 1, Character: 0},
		},
		Context: lsp.CodeActionContext{Diagnostics: []lsp.Diagnostic{}},
	})
	if action.Result == nil {
		t.Fatal("codeAction returned no result")
	}
	actions := parseResult[[]lsp.CodeAction](t, action.Result)
	if len(actions) != 3 {
		t.Fatalf("expected 3 code actions, got %d", len(actions))
	}

	exec := c.request(t, "workspace/executeCommand", lsp.ExecuteCommandParams{
		Command: "improveCode",
		Arguments: []any{
			lsp.CommandArgument{
				Range: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 0},
					End:   lsp.Position{Line: 1, Character: 1},
				},
				Query:       "Improve this code",
				Diagnostics: []string{},
			},
		},
	})
	if exec.Method != lsp.EventApplyEdit {
		t.Fatalf("expected workspace/applyEdit, got %s", exec.Method)
	}
	if exec.Params == nil {
		t.Fatal("applyEdit params missing")
	}
	edit := parseResult[lsp.ApplyWorkspaceEditParams](t, exec.Params)
	if edit.Edit.Changes == nil || len(edit.Edit.Changes) == 0 {
		t.Fatal("expected non-empty workspace edit changes")
	}

	shutdown := c.request(t, "shutdown", nil)
	if shutdown.Result != nil {
		t.Fatal("expected null result for shutdown")
	}
}

func TestPiLiveFlow(t *testing.T) {
	if !*runLive {
		t.Skip("skipping live test; use -live flag")
	}
	bin := buildBinary(t)
	c := newClient(t, bin,
		"--handler", "pi --print --no-session --no-tools --no-extensions --mode text --thinking off --model deepseek/deepseek-v4-flash",
		"--log-file", "/dev/null",
	)
	uri := testURI(t)

	initClient(t, c)
	didOpen(t, c, uri)
	didChange(t, c, uri)

	comp := c.request(t, "textDocument/completion", lsp.CompletionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: uri},
		Position:     lsp.Position{Line: 0, Character: 11},
	})
	if comp.Result == nil {
		t.Fatal("completion returned no result")
	}
	list := parseResult[lsp.CompletionList](t, comp.Result)
	if len(list.Items) == 0 {
		t.Fatal("expected completion items from pi")
	}

	c.request(t, "textDocument/codeAction", lsp.CodeActionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: uri},
		Range: lsp.Range{
			Start: lsp.Position{Line: 0, Character: 0},
			End:   lsp.Position{Line: 1, Character: 0},
		},
		Context: lsp.CodeActionContext{Diagnostics: []lsp.Diagnostic{}},
	})

	exec := c.request(t, "workspace/executeCommand", lsp.ExecuteCommandParams{
		Command: "improveCode",
		Arguments: []any{
			lsp.CommandArgument{
				Range: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 0},
					End:   lsp.Position{Line: 1, Character: 1},
				},
				Query:       "Improve this code",
				Diagnostics: []string{},
			},
		},
	})
	if exec.Method != lsp.EventApplyEdit {
		t.Fatalf("expected workspace/applyEdit, got %s", exec.Method)
	}
	if exec.Params == nil {
		t.Fatal("applyEdit params missing")
	}
	edit := parseResult[lsp.ApplyWorkspaceEditParams](t, exec.Params)
	if edit.Edit.Changes == nil || len(edit.Edit.Changes) == 0 {
		t.Fatal("expected non-empty workspace edit changes")
	}

	c.shutdown(t)
}
