package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"aivo/core/domain"
)

type testTool struct {
	name           string
	namespace      string
	content        string
	modelContent   string
	selectionGroup *domain.ToolSelectionGroup
}

func (t testTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{Name: t.name, Namespace: t.namespace, Description: "test", InputSchema: map[string]any{"type": "object"}, SelectionGroup: t.selectionGroup}
}

func (t testTool) Execute(context.Context, json.RawMessage, domain.ToolExecutionContext) domain.ToolResult {
	content := t.content
	if content == "" {
		content = "ok"
	}
	return domain.ToolResult{Name: t.name, OK: true, Content: content, ModelContent: t.modelContent}
}

type flakyTimeoutTool struct {
	attempts int32
}

func (t *flakyTimeoutTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{Name: "flaky_timeout", Description: "flaky timeout", InputSchema: map[string]any{"type": "object"}}
}

func (t *flakyTimeoutTool) Execute(ctx context.Context, _ json.RawMessage, _ domain.ToolExecutionContext) domain.ToolResult {
	attempt := atomic.AddInt32(&t.attempts, 1)
	if attempt < 3 {
		<-ctx.Done()
		return domain.ToolResult{Name: "flaky_timeout", OK: false, Error: ctx.Err().Error()}
	}
	return domain.ToolResult{Name: "flaky_timeout", OK: true, Content: "ok after retry"}
}

func TestToolRegistryRegistersQueriesAndRejectsDuplicate(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(testTool{name: "alpha"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := registry.Get("alpha"); !ok {
		t.Fatal("registered tool not found")
	}
	if _, ok := registry.Get("missing"); ok {
		t.Fatal("missing tool should not be found")
	}
	if len(registry.Specs()) != 1 || registry.Specs()[0].Name != "alpha" {
		t.Fatalf("specs = %#v", registry.Specs())
	}
	if err := registry.Register(testTool{name: "alpha"}); err == nil {
		t.Fatal("duplicate tool registration succeeded")
	}
}

func TestToolRuntimeRejectsMismatchedResponseNamespace(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(testTool{name: "imagegen", namespace: "image_gen"}); err != nil {
		t.Fatal(err)
	}
	runtime := NewToolRuntime(registry, t.TempDir())
	result := runtime.Execute(context.Background(), domain.ChatToolCall{ID: "call-1", Namespace: "other", Name: "imagegen", Arguments: json.RawMessage(`{}`)})
	if result.OK || result.ToolError == nil || result.ToolError.Code != "tool_namespace_mismatch" {
		t.Fatalf("result = %+v", result)
	}
}

func TestToolRegistrySupportsExplicitGroupsAndIndividualRegistrations(t *testing.T) {
	registry := NewRegistry()
	group := &domain.ToolSelectionGroup{ID: "extension_calendar", Name: "Calendar", Description: "Read and update calendars"}
	if err := registry.RegisterScopedBatch([]domain.Tool{
		testTool{name: "calendar_list", selectionGroup: group},
		testTool{name: "calendar_update", selectionGroup: group},
	}, domain.ToolSourceExtension, "calendar", "v1"); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterScoped(testTool{name: "notes_read"}, domain.ToolSourceExtension, "notes", "v1"); err != nil {
		t.Fatal(err)
	}

	entries := registry.CatalogEntries()
	if len(entries) != 3 || entries[0].SelectionGroup == nil || entries[1].SelectionGroup == nil || entries[2].SelectionGroup != nil {
		t.Fatalf("catalog entries = %#v, want two explicitly grouped tools and one individual tool", entries)
	}
	if *entries[0].SelectionGroup != *group || *entries[1].SelectionGroup != *group {
		t.Fatalf("catalog group metadata = %#v, %#v", entries[0].SelectionGroup, entries[1].SelectionGroup)
	}
	entries[0].SelectionGroup.Name = "mutated"
	if next := registry.CatalogEntries(); next[0].SelectionGroup == nil || next[0].SelectionGroup.Name != "Calendar" {
		t.Fatalf("catalog leaked mutable selection-group metadata: %#v", next)
	}
}

func TestToolRegistryRejectsInvalidOrInconsistentSelectionGroups(t *testing.T) {
	for name, group := range map[string]*domain.ToolSelectionGroup{
		"invalid id": {ID: "calendar.group", Name: "Calendar"},
		"blank name": {ID: "calendar_group", Name: "  "},
	} {
		t.Run(name, func(t *testing.T) {
			registry := NewRegistry()
			if err := registry.Register(testTool{name: "calendar_list", selectionGroup: group}); err == nil {
				t.Fatalf("registered invalid selection group %#v", group)
			}
		})
	}

	registry := NewRegistry()
	err := registry.RegisterScopedBatch([]domain.Tool{
		testTool{name: "calendar_list", selectionGroup: &domain.ToolSelectionGroup{ID: "calendar_group", Name: "Calendar"}},
		testTool{name: "calendar_update", selectionGroup: &domain.ToolSelectionGroup{ID: "calendar_group", Name: "Different"}},
	}, domain.ToolSourceExtension, "calendar", "v1")
	if err == nil || !strings.Contains(err.Error(), "inconsistent metadata") {
		t.Fatalf("inconsistent group metadata error = %v", err)
	}

	registry = NewRegistry()
	if err := registry.RegisterScoped(
		testTool{name: "calendar_list", selectionGroup: &domain.ToolSelectionGroup{ID: "calendar_group", Name: "Calendar"}},
		domain.ToolSourceExtension, "calendar", "v1",
	); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterScoped(
		testTool{name: "calendar_update", selectionGroup: &domain.ToolSelectionGroup{ID: "calendar_group", Name: "Different"}},
		domain.ToolSourceExtension, "calendar", "v1",
	); err == nil || !strings.Contains(err.Error(), "inconsistent metadata") {
		t.Fatalf("separate inconsistent group registration error = %v", err)
	}
	if err := registry.RegisterScoped(
		testTool{name: "other_calendar", selectionGroup: &domain.ToolSelectionGroup{ID: "calendar_group", Name: "Calendar"}},
		domain.ToolSourceExtension, "other", "v1",
	); err == nil || !strings.Contains(err.Error(), "another source") {
		t.Fatalf("cross-source group registration error = %v", err)
	}
}

func TestToolRuntimeRetriesTimeouts(t *testing.T) {
	tool := &flakyTimeoutTool{}
	registry := NewRegistry()
	if err := registry.Register(tool); err != nil {
		t.Fatal(err)
	}
	runtime := NewToolRuntime(registry, t.TempDir())
	runtime.Timeout = 20 * time.Millisecond

	result := runtime.Execute(context.Background(), domain.ChatToolCall{ID: "call_flaky", Name: "flaky_timeout", Arguments: json.RawMessage(`{}`)})
	if !result.OK || result.Content != "ok after retry" {
		t.Fatalf("result = %#v, want successful retry", result)
	}
	if atomic.LoadInt32(&tool.attempts) != 3 {
		t.Fatalf("attempts = %d, want 3", atomic.LoadInt32(&tool.attempts))
	}
	if result.Structured["attempts"] != 3 {
		t.Fatalf("structured attempts = %#v, want 3", result.Structured["attempts"])
	}
}

func TestReadOnlyToolRegistryContainsOnlyReadOutsideRepository(t *testing.T) {
	registry, err := NewReadOnlyToolRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := registry.Get("git_status"); ok {
		t.Fatal("git_status should not be registered outside a git work tree")
	}
	if _, ok := registry.Get("git_diff"); ok {
		t.Fatal("git_diff should not be registered outside a git work tree")
	}
	if specs := registry.Specs(); len(specs) != 1 || specs[0].Name != "read" {
		t.Fatalf("read-only specs = %#v, want exactly read", specs)
	}
}

func TestReadOnlyToolRegistryContainsOnlyReadInsideRepository(t *testing.T) {
	root := t.TempDir()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}

	registry, err := NewReadOnlyToolRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	if specs := registry.Specs(); len(specs) != 1 || specs[0].Name != "read" {
		t.Fatalf("read-only specs = %#v, want exactly read", specs)
	}
}

func TestToolRuntimeTruncatesOutput(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(testTool{name: "large", content: strings.Repeat("x", 20)}); err != nil {
		t.Fatal(err)
	}
	runtime := NewToolRuntime(registry, t.TempDir())
	runtime.MaxOutputChars = 5

	result := runtime.Execute(context.Background(), domain.ChatToolCall{ID: "call_large", Name: "large", Arguments: json.RawMessage(`{}`)})
	if !result.OK || !result.Truncated || result.OriginalSize != 20 {
		t.Fatalf("result = %#v, want truncated successful result with original size", result)
	}
	if !strings.Contains(result.Content, "[truncated: content exceeded 5 characters; full output retained at ") {
		t.Fatalf("content = %q, want truncation marker", result.Content)
	}
	if len(result.RetainedOutputRefs) != 1 {
		t.Fatalf("retained refs = %#v, want one ref", result.RetainedOutputRefs)
	}
	retained, err := os.ReadFile(result.RetainedOutputRefs[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(retained) != strings.Repeat("x", 20) {
		t.Fatalf("retained output = %q, want full output", retained)
	}
}

func TestToolRuntimeTruncatesModelContent(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(testTool{name: "large_model", content: "ok", modelContent: strings.Repeat("m", 20)}); err != nil {
		t.Fatal(err)
	}
	runtime := NewToolRuntime(registry, t.TempDir())
	runtime.MaxOutputChars = 5

	result := runtime.Execute(context.Background(), domain.ChatToolCall{ID: "call_large_model", Name: "large_model", Arguments: json.RawMessage(`{}`)})
	if !result.OK || !result.Truncated || result.OriginalSize != 20 {
		t.Fatalf("result = %#v, want truncated successful result with model original size", result)
	}
	if result.Content != "ok" {
		t.Fatalf("content = %q, want visible output unchanged", result.Content)
	}
	if !strings.Contains(result.ModelContent, "[truncated: model content exceeded 5 characters; full output retained at ") {
		t.Fatalf("model content = %q, want truncation marker", result.ModelContent)
	}
	if len(result.RetainedOutputRefs) != 1 {
		t.Fatalf("retained refs = %#v, want one ref", result.RetainedOutputRefs)
	}
	retained, err := os.ReadFile(result.RetainedOutputRefs[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(retained) != strings.Repeat("m", 20) {
		t.Fatalf("retained output = %q, want full model output", retained)
	}
}

func TestReadRetainedOutputBoundsRefsAndSupportsChunks(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ref, err := retainSandboxOutput(SandboxRequest{SessionID: "s1", ToolCallID: "call1"}, "content", "hello world")
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.ReadRetainedOutput(context.Background(), domain.RetainedOutputReadInput{Ref: ref, Offset: 6, Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "world" || result.Offset != 6 || result.NextOffset != 11 || result.Size != 11 || result.Truncated {
		t.Fatalf("result = %#v, want bounded chunk", result)
	}
	if _, err := service.ReadRetainedOutput(context.Background(), domain.RetainedOutputReadInput{Ref: filepath.Join(t.TempDir(), "outside.log")}); err == nil {
		t.Fatal("expected outside retained output ref to be rejected")
	}
}

func TestWebFetchExtractsReadableText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html><head><title>Docs Page</title><style>.x{}</style></head><body><h1>Install Aivo</h1><script>hidden()</script><p>Run npm test.</p></body></html>`))
	}))
	defer server.Close()
	tool := NewWebFetchTool()
	tool.client = server.Client()

	result := tool.Execute(context.Background(), json.RawMessage(`{"url":"`+server.URL+`","maxChars":1000}`), domain.ToolExecutionContext{})
	if !result.OK {
		t.Fatalf("result = %#v", result)
	}
	if !strings.Contains(result.Content, "Title: Docs Page") || !strings.Contains(result.Content, "Install Aivo") || !strings.Contains(result.Content, "Run npm test.") {
		t.Fatalf("content = %q", result.Content)
	}
	if strings.Contains(result.Content, "hidden()") || strings.Contains(result.Content, ".x{}") {
		t.Fatalf("content leaked script/style text: %q", result.Content)
	}
}

func TestWebFetchRejectsNonHTTPURL(t *testing.T) {
	result := NewWebFetchTool().Execute(context.Background(), json.RawMessage(`{"url":"file:///etc/passwd"}`), domain.ToolExecutionContext{})
	if result.OK || result.ToolError == nil {
		t.Fatalf("result = %#v, want invalid url failure", result)
	}
}

func TestWebSearchParsesResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("q"); got != "aivo docs" {
			t.Fatalf("query = %q", got)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`
			<html><body>
				<a class="result__a" href="https://example.com/docs">Aivo Docs</a>
				<a class="result__snippet" href="https://example.com/docs">Desktop agent documentation.</a>
				<a class="result__a" href="/l/?uddg=https%3A%2F%2Fexample.org%2Frelease">Release Notes</a>
				<div class="result__snippet">Latest release details.</div>
			</body></html>`))
	}))
	defer server.Close()
	tool := NewWebSearchTool()
	tool.client = server.Client()
	tool.searchURL = server.URL + "/search"

	result := tool.Execute(context.Background(), json.RawMessage(`{"query":"aivo docs","limit":2}`), domain.ToolExecutionContext{})
	if !result.OK {
		t.Fatalf("result = %#v", result)
	}
	if !strings.Contains(result.Content, "1. Aivo Docs") || !strings.Contains(result.Content, "https://example.com/docs") {
		t.Fatalf("content = %q", result.Content)
	}
	if !strings.Contains(result.Content, "2. Release Notes") || !strings.Contains(result.Content, "https://example.org/release") {
		t.Fatalf("content = %q", result.Content)
	}
	if result.Structured["provider"] != "duckduckgo" {
		t.Fatalf("provider = %#v, want duckduckgo", result.Structured["provider"])
	}
}

func TestCodingToolRegistryOmitsLegacyQualityTools(t *testing.T) {
	registry, err := NewCodingToolRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"read_diagnostics", "format_code"} {
		if _, ok := registry.Get(name); ok {
			t.Fatalf("legacy tool %s must not be registered in coding registry", name)
		}
	}
}

func TestExecCommandSpecMatchesCodexStyleSchema(t *testing.T) {
	root := t.TempDir()
	execCommand := NewExecCommandTool(root, nil, nil).Spec()
	if execCommand.Name != ExecCommandToolName {
		t.Fatalf("exec command name = %q, want %q", execCommand.Name, ExecCommandToolName)
	}
	if execCommand.RiskLevel != "critical" {
		t.Fatalf("exec command risk level = %q, want critical", execCommand.RiskLevel)
	}
	if !strings.Contains(execCommand.Description, "Runs a command in a PTY, returning output or a session ID for ongoing interaction.") {
		t.Fatalf("exec command description = %q", execCommand.Description)
	}
	properties, ok := execCommand.InputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("exec command schema properties = %#v", execCommand.InputSchema["properties"])
	}
	required, ok := execCommand.InputSchema["required"].([]string)
	if !ok || !reflect.DeepEqual(required, []string{"cmd"}) {
		t.Fatalf("exec command required = %#v, want cmd", execCommand.InputSchema["required"])
	}
	wantDescriptions := map[string]string{
		"cmd":                 "Shell command to execute.",
		"workdir":             "Working directory for the command. Defaults to the turn cwd.",
		"shell":               "Shell binary to launch. Defaults to the user's default shell.",
		"login":               "True runs the shell with -l/-i semantics; false disables them. Defaults to true.",
		"tty":                 "True allocates a PTY for the command; false or omitted uses plain pipes.",
		"yield_time_ms":       "Wait before yielding output.",
		"max_output_tokens":   "Output token budget. Defaults to 10000 tokens; larger requests may be capped by policy.",
		"sandbox_permissions": "Per-command sandbox override. Defaults to `use_default`; use `require_escalated` for unsandboxed execution.",
		"justification":       "User-facing approval question for `require_escalated`; omit otherwise.",
		"prefix_rule":         "Reusable approval prefix for `cmd`, only with `sandbox_permissions: \"require_escalated\"`; for example [\"git\", \"pull\"].",
	}
	for name, want := range wantDescriptions {
		property, ok := properties[name].(map[string]any)
		if !ok {
			t.Fatalf("exec command property %s = %#v", name, properties[name])
		}
		description, _ := property["description"].(string)
		if !strings.Contains(description, want) {
			t.Fatalf("exec command property %s description = %q, want %q", name, description, want)
		}
	}
	if len(properties) != len(wantDescriptions) {
		t.Fatalf("exec command properties = %#v", properties)
	}
}

func TestWorkspaceHasGitUsesRevParse(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := t.TempDir()
	nested := filepath.Join(root, "packages", "app")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if workspaceHasGit(nested) {
		t.Fatal("workspaceHasGit = true before git init")
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}
	if !workspaceHasGit(nested) {
		t.Fatal("workspaceHasGit = false, want nested workspace inside initialized git repository")
	}
}

func TestWorkspaceHasGitTimesOutQuickly(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	gitPath := filepath.Join(bin, "git")
	if err := os.WriteFile(gitPath, []byte("#!/bin/sh\nsleep 5\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+oldPath)
	start := time.Now()
	if workspaceHasGit(root) {
		t.Fatal("workspaceHasGit = true with hanging git command")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("workspaceHasGit took %s, want quick timeout", elapsed)
	}
}

func TestLSPSymbolSearchFindsWorkspaceDefinitions(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o700); err != nil {
		t.Fatal(err)
	}
	goSource := "package demo\n\ntype Runner struct{}\n\nfunc BuildRunner() {}\nfunc (r *Runner) RunTask() {}\n"
	if err := os.WriteFile(filepath.Join(root, "runner.go"), []byte(goSource), 0o600); err != nil {
		t.Fatal(err)
	}
	tsSource := "export interface RunnerOptions {}\nexport function createRunner() {}\nconst runnerValue = 1\n"
	if err := os.WriteFile(filepath.Join(root, "src", "runner.ts"), []byte(tsSource), 0o600); err != nil {
		t.Fatal(err)
	}

	result := NewLSPSymbolSearchTool(root).Execute(context.Background(), json.RawMessage(`{"query":"Runner","limit":10}`), domain.ToolExecutionContext{WorkspaceRoot: root})
	if !result.OK {
		t.Fatalf("result = %#v", result)
	}
	for _, want := range []string{"runner.go:3 type Runner", "runner.go:5 function BuildRunner", "src/runner.ts:1 interface RunnerOptions", "src/runner.ts:2 function createRunner"} {
		if !strings.Contains(result.Content, want) {
			t.Fatalf("content = %q, missing %q", result.Content, want)
		}
	}
	structured, ok := result.Structured["results"].([]map[string]any)
	if !ok || len(structured) < 4 {
		t.Fatalf("structured results = %#v", result.Structured["results"])
	}
}

func TestLSPSymbolSearchHonorsPathKindAndIgnoredDirs(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "pkg"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\nfunc VisibleRunner() {}\ntype VisibleType struct{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "node_modules", "pkg", "hidden.go"), []byte("package pkg\nfunc HiddenRunner() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	result := NewLSPSymbolSearchTool(root).Execute(context.Background(), json.RawMessage(`{"query":"Visible","kind":"function","path":"."}`), domain.ToolExecutionContext{WorkspaceRoot: root})
	if !result.OK {
		t.Fatalf("result = %#v", result)
	}
	if !strings.Contains(result.Content, "function VisibleRunner") || strings.Contains(result.Content, "VisibleType") || strings.Contains(result.Content, "HiddenRunner") {
		t.Fatalf("content = %q", result.Content)
	}
}

func writeFakeLSP(output io.Writer, payload any) {
	raw, _ := json.Marshal(payload)
	_, _ = fmt.Fprintf(output, "Content-Length: %d\r\n\r\n", len(raw))
	_, _ = output.Write(raw)
}
