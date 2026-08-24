package app

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestAgentLoopExecutesToolAndAppendsToolResult(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("Aivo README"), 0o600); err != nil {
		t.Fatal(err)
	}
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []map[string]any `json:"messages"`
			Tools    []any            `json:"tools"`
			Stream   bool             `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.Tools) == 0 {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"tools\":[],\"reason\":\"core read is sufficient\"}"}}]}`))
			return
		}
		requestCount++
		if len(body.Tools) == 0 {
			t.Fatal("tools were not exposed")
		}
		if !body.Stream {
			t.Fatal("tool-enabled request should stream")
		}
		w.Header().Set("Content-Type", "application/json")
		if requestCount == 1 {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"tool_calls":[{"id":"call_read","type":"function","function":{"name":"read","arguments":"{\"path\":\"README.md\"}"}}]}}]}`))
			return
		}
		foundToolResult := false
		for _, message := range body.Messages {
			if message["role"] == "tool" && strings.Contains(message["content"].(string), "Aivo README") {
				foundToolResult = true
			}
		}
		if !foundToolResult {
			t.Fatalf("second request messages missing tool result: %#v", body.Messages)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"README says Aivo README"}}]}`))
	}))
	defer server.Close()
	if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{ProviderID: "custom-api", Type: "openai-compatible", BaseURL: server.URL, ModelID: "test-model", APIKey: "test-key", Method: "api-key"}); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, ProjectPath: root})
	if err != nil {
		t.Fatal(err)
	}
	run, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "总结 README.md"})
	if err != nil {
		t.Fatal(err)
	}
	if run.AssistantEvent == nil || run.AssistantEvent.Content != "README says Aivo README" || requestCount != 2 {
		t.Fatalf("run = %#v requestCount=%d", run, requestCount)
	}
	toolCalls, err := service.ListToolCalls(ctx, session.ID)
	if err != nil || len(toolCalls) != 1 || toolCalls[0].Status != domain.ToolCallStatusSuccess {
		t.Fatalf("tool calls = %#v, %v", toolCalls, err)
	}
}

func TestAgentLoopStreamsTextAfterStreamedToolCall(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("Aivo README"), 0o600); err != nil {
		t.Fatal(err)
	}
	var deltas []string
	service.SetAssistantDeltaHook(func(sessionID string, turnID string, delta string) {
		deltas = append(deltas, delta)
	})
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []map[string]any `json:"messages"`
			Tools    []any            `json:"tools"`
			Stream   bool             `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.Tools) == 0 {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"tools\":[],\"reason\":\"core read is sufficient\"}"}}]}`))
			return
		}
		requestCount++
		if !body.Stream {
			t.Fatal("tool-enabled request should stream")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		if requestCount == 1 {
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_read\",\"type\":\"function\",\"function\":{\"name\":\"read\",\"arguments\":\"{\\\"path\\\"\"}}]}}]}\n\n"))
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\":\\\"README.md\\\"}\"}}]},\"finish_reason\":\"tool_calls\"}]}\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			return
		}
		foundToolResult := false
		for _, message := range body.Messages {
			if message["role"] == "tool" && strings.Contains(message["content"].(string), "Aivo README") {
				foundToolResult = true
			}
		}
		if !foundToolResult {
			t.Fatalf("second request messages missing tool result: %#v", body.Messages)
		}
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"README\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\" streamed\"},\"finish_reason\":\"stop\"}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()
	if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{ProviderID: "custom-api", Type: "openai-compatible", BaseURL: server.URL, ModelID: "test-model", APIKey: "test-key", Method: "api-key"}); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, ProjectPath: root})
	if err != nil {
		t.Fatal(err)
	}
	run, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "总结 README.md"})
	if err != nil {
		t.Fatal(err)
	}
	if run.AssistantEvent == nil || run.AssistantEvent.Content != "README streamed" || requestCount != 2 {
		t.Fatalf("run = %#v requestCount=%d", run, requestCount)
	}
	if strings.Join(deltas, "") != "README streamed" {
		t.Fatalf("deltas = %#v, want streamed final text only", deltas)
	}
}

func TestAgentLoopPlainChat(t *testing.T) {
	t.Run("plain chat", func(t *testing.T) {
		service, cleanup := newSessionTestService(t)
		defer cleanup()
		ctx := context.Background()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"plain reply"}}]}`))
		}))
		defer server.Close()
		if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{ProviderID: "custom-api", Type: "openai-compatible", BaseURL: server.URL, ModelID: "test-model", APIKey: "test-key", Method: "api-key"}); err != nil {
			t.Fatal(err)
		}
		session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, ProjectPath: t.TempDir()})
		if err != nil {
			t.Fatal(err)
		}
		run, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "你好"})
		if err != nil {
			t.Fatal(err)
		}
		if run.AssistantEvent == nil || run.AssistantEvent.Content != "plain reply" {
			t.Fatalf("run = %#v", run)
		}
	})
}

func TestLSPFallbackToolsReturnStructuredResults(t *testing.T) {
	root := t.TempDir()
	source := "package main\n\n// TODO: tighten behavior\nfunc Target() {}\nfunc Caller() { Target() }\n"
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	registry := newLSPTestRegistry(t, root)
	for _, name := range []string{"lsp_diagnostics", "lsp_definition", "lsp_references", "lsp_symbol_search"} {
		if _, ok := registry.Get(name); !ok {
			t.Fatalf("tool %s is not registered", name)
		}
	}
	runtime := NewToolRuntime(registry, root)
	ctx := context.Background()
	diagnostics := runtime.ExecuteWithContext(ctx, domain.ChatToolCall{ID: "diag", Name: "lsp_diagnostics", Arguments: json.RawMessage(`{"path":"main.go"}`)}, domain.ToolExecutionContext{WorkspaceRoot: root})
	if !diagnostics.OK || diagnostics.Structured["status"] == nil || !strings.Contains(diagnostics.Content, "TODO") {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
	definition := runtime.ExecuteWithContext(ctx, domain.ChatToolCall{ID: "def", Name: "lsp_definition", Arguments: json.RawMessage(`{"path":"main.go","line":5,"character":18}`)}, domain.ToolExecutionContext{WorkspaceRoot: root})
	if !definition.OK || !strings.Contains(definition.Content, "func Target") {
		t.Fatalf("definition = %#v", definition)
	}
	references := runtime.ExecuteWithContext(ctx, domain.ChatToolCall{ID: "refs", Name: "lsp_references", Arguments: json.RawMessage(`{"path":"main.go","line":5,"character":18}`)}, domain.ToolExecutionContext{WorkspaceRoot: root})
	if !references.OK || !strings.Contains(references.Content, "Caller") {
		t.Fatalf("references = %#v", references)
	}
	symbols := runtime.ExecuteWithContext(ctx, domain.ChatToolCall{ID: "symbols", Name: "lsp_symbol_search", Arguments: json.RawMessage(`{"query":"Target"}`)}, domain.ToolExecutionContext{WorkspaceRoot: root})
	if !symbols.OK || symbols.Structured["status"] == nil || !strings.Contains(symbols.Content, "Target") {
		t.Fatalf("symbols = %#v", symbols)
	}
}

func TestBoundedLSPManagerStartsFakeGoServer(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fake\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc FakeSymbol() {}\nfunc main() { FakeSymbol() }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	binDir := t.TempDir()
	fakeGopls := filepath.Join(binDir, "gopls")
	script := "#!/bin/sh\nAIVO_FAKE_LSP=1 exec \"$AIVO_TEST_BINARY\" -test.run=TestFakeLSPServer --\n"
	if err := os.WriteFile(fakeGopls, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AIVO_TEST_BINARY", os.Args[0])
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	manager := newBoundedLSPManager()
	restore := setCodeIntelligenceServiceForTest(manager)
	defer restore()

	ctx := context.Background()
	status, err := manager.Status(ctx, root)
	if err != nil {
		t.Fatalf("status err = %v", err)
	}
	if status.Status != domain.CodeIntelligenceStatusReady || status.Language != "go" || status.Source != "gopls" {
		t.Fatalf("status = %#v", status)
	}
	registry := newLSPTestRegistry(t, root)
	runtime := NewToolRuntime(registry, root)
	diagnostics := runtime.ExecuteWithContext(ctx, domain.ChatToolCall{ID: "diag", Name: "lsp_diagnostics", Arguments: json.RawMessage(`{"path":"main.go"}`)}, domain.ToolExecutionContext{WorkspaceRoot: root})
	if !diagnostics.OK || !strings.Contains(diagnostics.Content, "fake compile error") {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
	symbols := runtime.ExecuteWithContext(ctx, domain.ChatToolCall{ID: "sym", Name: "lsp_symbol_search", Arguments: json.RawMessage(`{"query":"FakeSymbol"}`)}, domain.ToolExecutionContext{WorkspaceRoot: root})
	if !symbols.OK || !strings.Contains(symbols.Content, "FakeSymbol") {
		t.Fatalf("symbols = %#v", symbols)
	}
	symbolStatus, _ := symbols.Structured["status"].(domain.CodeIntelligenceStatus)
	if symbolStatus.Status != domain.CodeIntelligenceStatusReady {
		t.Fatalf("symbol status = %#v", symbols.Structured["status"])
	}
	definition := runtime.ExecuteWithContext(ctx, domain.ChatToolCall{ID: "def", Name: "lsp_definition", Arguments: json.RawMessage(`{"path":"main.go","line":4,"character":15}`)}, domain.ToolExecutionContext{WorkspaceRoot: root})
	if !definition.OK || !strings.Contains(definition.Content, "func FakeSymbol") {
		t.Fatalf("definition = %#v", definition)
	}
	references := runtime.ExecuteWithContext(ctx, domain.ChatToolCall{ID: "refs", Name: "lsp_references", Arguments: json.RawMessage(`{"path":"main.go","line":4,"character":15}`)}, domain.ToolExecutionContext{WorkspaceRoot: root})
	if !references.OK || !strings.Contains(references.Content, "func main") {
		t.Fatalf("references = %#v", references)
	}
}

func newLSPTestRegistry(t *testing.T, root string) *Registry {
	t.Helper()
	registry := NewRegistry()
	for _, tool := range []domain.Tool{
		NewLSPDiagnosticsTool(root),
		NewLSPDefinitionTool(root),
		NewLSPReferencesTool(root),
		NewLSPSymbolSearchTool(root),
	} {
		if err := registry.RegisterScoped(tool, domain.ToolSourceExtension, "test.code-intelligence", "v1"); err != nil {
			t.Fatal(err)
		}
	}
	return registry
}

func TestBoundedLSPManagerStartsFakeTypeScriptServer(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"type":"module"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "app.ts"), []byte("export function FakeSymbol() {}\nFakeSymbol()\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	binDir := t.TempDir()
	fakeServer := filepath.Join(binDir, "typescript-language-server")
	script := "#!/bin/sh\nAIVO_FAKE_LSP=1 AIVO_FAKE_LSP_FILE=app.ts exec \"$AIVO_TEST_BINARY\" -test.run=TestFakeLSPServer -- \"$@\"\n"
	if err := os.WriteFile(fakeServer, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AIVO_TEST_BINARY", os.Args[0])
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	manager := newBoundedLSPManager()
	defer manager.Close()

	status, err := manager.Status(context.Background(), root)
	if err != nil {
		t.Fatalf("status err = %v", err)
	}
	if status.Status != domain.CodeIntelligenceStatusReady || status.Language != "typescript" || status.Source != "typescript-language-server" {
		t.Fatalf("status = %#v", status)
	}
	symbols, symbolStatus, err := manager.Symbols(context.Background(), root, "FakeSymbol", "", "", 10)
	if err != nil {
		t.Fatalf("symbols err = %v", err)
	}
	if symbolStatus.Status != domain.CodeIntelligenceStatusReady || len(symbols) == 0 || symbols[0].Path != "app.ts" {
		t.Fatalf("symbols = %#v status=%#v", symbols, symbolStatus)
	}
}

func TestFakeLSPServer(t *testing.T) {
	if os.Getenv("AIVO_FAKE_LSP") != "1" {
		return
	}
	fakeLSPServe(os.Stdin, os.Stdout)
	os.Exit(0)
}

func fakeLSPServe(input io.Reader, output io.Writer) {
	reader := bufio.NewReader(input)
	for {
		length, err := readLSPContentLength(reader)
		if err != nil {
			return
		}
		raw := make([]byte, length)
		if _, err := io.ReadFull(reader, raw); err != nil {
			return
		}
		var message struct {
			ID     json.RawMessage `json:"id,omitempty"`
			Method string          `json:"method,omitempty"`
			Params json.RawMessage `json:"params,omitempty"`
		}
		if err := json.Unmarshal(raw, &message); err != nil {
			continue
		}
		switch message.Method {
		case "initialize":
			writeFakeLSP(output, map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(message.ID), "result": map[string]any{"capabilities": map[string]any{}}})
		case "textDocument/didOpen":
			var params struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
			}
			_ = json.Unmarshal(message.Params, &params)
			writeFakeLSP(output, map[string]any{"jsonrpc": "2.0", "method": "textDocument/publishDiagnostics", "params": map[string]any{
				"uri": params.TextDocument.URI,
				"diagnostics": []map[string]any{{
					"range":    map[string]any{"start": map[string]any{"line": 1, "character": 0}, "end": map[string]any{"line": 1, "character": 4}},
					"severity": 1,
					"source":   "fake-gopls",
					"message":  "fake compile error",
				}},
			}})
		case "workspace/symbol":
			uri := fileURI(filepath.Join(mustGetwd(), fakeLSPFile()))
			writeFakeLSP(output, map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(message.ID), "result": []map[string]any{{
				"name": "FakeSymbol",
				"kind": 12,
				"location": map[string]any{
					"uri":   uri,
					"range": map[string]any{"start": map[string]any{"line": 2, "character": 5}, "end": map[string]any{"line": 2, "character": 15}},
				},
			}}})
		case "textDocument/definition":
			uri := fileURI(filepath.Join(mustGetwd(), fakeLSPFile()))
			writeFakeLSP(output, map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(message.ID), "result": []map[string]any{{
				"uri":   uri,
				"range": map[string]any{"start": map[string]any{"line": 2, "character": 5}, "end": map[string]any{"line": 2, "character": 15}},
			}}})
		case "textDocument/references":
			uri := fileURI(filepath.Join(mustGetwd(), fakeLSPFile()))
			writeFakeLSP(output, map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(message.ID), "result": []map[string]any{{
				"uri":   uri,
				"range": map[string]any{"start": map[string]any{"line": 3, "character": 14}, "end": map[string]any{"line": 3, "character": 24}},
			}}})
		}
	}
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func fakeLSPFile() string {
	if file := strings.TrimSpace(os.Getenv("AIVO_FAKE_LSP_FILE")); file != "" {
		return file
	}
	return "main.go"
}
