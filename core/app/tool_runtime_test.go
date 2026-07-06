package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"aivo/core/domain"
)

type testTool struct {
	name         string
	content      string
	modelContent string
}

func (t testTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{Name: t.name, Description: "test", InputSchema: map[string]any{"type": "object"}}
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

func TestReadOnlyToolRegistryOmitsGitToolsOutsideRepository(t *testing.T) {
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
	if _, ok := registry.Get("web_fetch"); !ok {
		t.Fatal("web_fetch should be registered in read-only registry")
	}
	if _, ok := registry.Get("web_search"); !ok {
		t.Fatal("web_search should be registered in read-only registry")
	}
	if _, ok := registry.Get("read_diagnostics"); ok {
		t.Fatal("read_diagnostics should not be registered in read-only registry")
	}
}

func TestReadOnlyToolRegistryIncludesGitToolsInsideRepository(t *testing.T) {
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
	if _, ok := registry.Get("git_status"); !ok {
		t.Fatal("git_status should be registered inside a git work tree")
	}
	if _, ok := registry.Get("git_diff"); !ok {
		t.Fatal("git_diff should be registered inside a git work tree")
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

func TestCodingToolRegistryIncludesQualityTools(t *testing.T) {
	registry, err := NewCodingToolRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"read_diagnostics", "format_code"} {
		if _, ok := registry.Get(name); !ok {
			t.Fatalf("%s should be registered in coding registry", name)
		}
	}
}

func TestCommandToolSpecsPreferDedicatedToolsBeforeBash(t *testing.T) {
	root := t.TempDir()
	bash := NewBashTool(root, nil).Spec()
	if bash.RiskLevel != "critical" {
		t.Fatalf("bash risk level = %q, want critical", bash.RiskLevel)
	}
	for _, required := range []string{
		"Escape-hatch shell execution",
		"Prefer run_tests",
		"read_diagnostics",
		"format_code",
		"no safer dedicated tool",
	} {
		if !strings.Contains(bash.Description, required) {
			t.Fatalf("bash description missing %q: %q", required, bash.Description)
		}
	}
	properties, ok := bash.InputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("bash schema properties = %#v", bash.InputSchema["properties"])
	}
	command, ok := properties["command"].(map[string]any)
	if !ok {
		t.Fatalf("bash command schema = %#v", properties["command"])
	}
	commandDescription, _ := command["description"].(string)
	if !strings.Contains(commandDescription, "Do not use bash for test, lint, build, diagnostics, or formatting") {
		t.Fatalf("bash command description should steer to dedicated tools: %q", commandDescription)
	}

	runTests := NewRunTestsTool(root, nil).Spec()
	if !strings.Contains(runTests.Description, "Preferred tool for declared test, lint, or build commands") ||
		!strings.Contains(runTests.Description, "Use this instead of bash") {
		t.Fatalf("run_tests description should be preferred over bash: %q", runTests.Description)
	}
	readDiagnostics := NewReadDiagnosticsTool(root, nil).Spec()
	if !strings.Contains(readDiagnostics.Description, "Preferred tool for declared diagnostics") ||
		!strings.Contains(readDiagnostics.Description, "falling back to bash") {
		t.Fatalf("read_diagnostics description should be preferred over bash: %q", readDiagnostics.Description)
	}
	formatCode := NewFormatCodeTool(root, nil).Spec()
	if !strings.Contains(formatCode.Description, "Preferred tool for formatter-backed source rewrites") ||
		!strings.Contains(formatCode.Description, "instead of falling back to bash") {
		t.Fatalf("format_code description should be preferred over bash: %q", formatCode.Description)
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

func TestParseDiagnosticsProblems(t *testing.T) {
	problems := parseDiagnosticsProblems("pkg/demo.go:12:3: undefined: value\nsrc/app.tsx:9: missing semicolon\nsrc/main.ts(22,7): error TS2304: Cannot find name 'x'.\n")
	if len(problems) != 3 {
		t.Fatalf("problems = %#v", problems)
	}
	if problems[0]["file"] != "pkg/demo.go" || problems[0]["line"] != 12 || problems[0]["column"] != 3 {
		t.Fatalf("first problem = %#v", problems[0])
	}
	if problems[1]["file"] != "src/app.tsx" || problems[1]["line"] != 9 {
		t.Fatalf("second problem = %#v", problems[1])
	}
	if problems[2]["file"] != "src/main.ts" || problems[2]["line"] != 22 || problems[2]["column"] != 7 {
		t.Fatalf("third problem = %#v", problems[2])
	}
}

func TestDiagnosticsCommandSupportsAllTarget(t *testing.T) {
	command, err := diagnosticsCommand(diagnosticsInput{Target: "all", Kind: "auto"})
	if err != nil {
		t.Fatal(err)
	}
	if command != "npm run diagnostics" {
		t.Fatalf("command = %q, want npm run diagnostics", command)
	}
}

func TestFormatCodeFormatsGoFileAndReportsDiff(t *testing.T) {
	root := t.TempDir()
	if _, err := exec.LookPath("gofmt"); err != nil {
		t.Skipf("gofmt unavailable: %v", err)
	}
	path := filepath.Join(root, "main.go")
	if err := os.WriteFile(path, []byte("package main\nfunc main(){println(\"hi\")}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tool := NewFormatCodeTool(root, nil)
	result := tool.Execute(context.Background(), json.RawMessage(`{"paths":["main.go"]}`), domain.ToolExecutionContext{WorkspaceRoot: root})
	if !result.OK {
		t.Fatalf("result = %#v", result)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(raw); !strings.Contains(got, "func main() {") {
		t.Fatalf("formatted content = %q", got)
	}
	if len(result.Files) != 1 || result.Files[0].Path != "main.go" || !strings.Contains(result.Files[0].Diff, "+func main() {") {
		t.Fatalf("files = %#v", result.Files)
	}
}

func TestFormatCodeUsesProjectLocalPrettier(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "node_modules", ".bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	prettier := filepath.Join(binDir, "prettier")
	script := "#!/bin/sh\nshift\nfor file in \"$@\"; do\n  printf 'const value = 1;\\n' > \"$file\"\ndone\n"
	if err := os.WriteFile(prettier, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "app.ts")
	if err := os.WriteFile(path, []byte("const value=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tool := NewFormatCodeTool(root, nil)
	result := tool.Execute(context.Background(), json.RawMessage(`{"paths":["app.ts"]}`), domain.ToolExecutionContext{WorkspaceRoot: root})
	if !result.OK {
		t.Fatalf("result = %#v", result)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(raw); got != "const value = 1;\n" {
		t.Fatalf("formatted content = %q", got)
	}
	commands, _ := result.Structured["formatterCommands"].([]map[string]any)
	if len(commands) != 1 || commands[0]["formatter"] != "prettier" || !strings.Contains(commands[0]["command"].(string), "node_modules/.bin/prettier") {
		t.Fatalf("formatterCommands = %#v", result.Structured["formatterCommands"])
	}
	if len(result.Files) != 1 || result.Files[0].Path != "app.ts" || !strings.Contains(result.Files[0].Diff, "+const value = 1;") {
		t.Fatalf("files = %#v", result.Files)
	}
}

func TestFormatCodeCanRunProjectLocalESLintFixAfterPrettier(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "node_modules", ".bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	prettier := filepath.Join(binDir, "prettier")
	prettierScript := "#!/bin/sh\nshift\nfor file in \"$@\"; do\n  printf 'const value = 1\\n' > \"$file\"\ndone\n"
	if err := os.WriteFile(prettier, []byte(prettierScript), 0o700); err != nil {
		t.Fatal(err)
	}
	eslint := filepath.Join(binDir, "eslint")
	eslintScript := "#!/bin/sh\nshift\nfor file in \"$@\"; do\n  printf 'const value = 1;\\n' > \"$file\"\ndone\n"
	if err := os.WriteFile(eslint, []byte(eslintScript), 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "app.ts")
	if err := os.WriteFile(path, []byte("const value=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tool := NewFormatCodeTool(root, nil)
	result := tool.Execute(context.Background(), json.RawMessage(`{"paths":["app.ts"],"eslintFix":true}`), domain.ToolExecutionContext{WorkspaceRoot: root})
	if !result.OK {
		t.Fatalf("result = %#v", result)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(raw); got != "const value = 1;\n" {
		t.Fatalf("formatted content = %q", got)
	}
	commands, _ := result.Structured["formatterCommands"].([]map[string]any)
	if len(commands) != 2 || commands[0]["formatter"] != "prettier" || commands[1]["formatter"] != "eslint" {
		t.Fatalf("formatterCommands = %#v", result.Structured["formatterCommands"])
	}
	if !strings.Contains(commands[1]["command"].(string), "node_modules/.bin/eslint") || !strings.Contains(commands[1]["command"].(string), "--fix") {
		t.Fatalf("eslint command = %#v", commands[1]["command"])
	}
	if len(result.Files) != 1 || result.Files[0].Path != "app.ts" || !strings.Contains(result.Files[0].Diff, "+const value = 1;") {
		t.Fatalf("files = %#v", result.Files)
	}
}

func TestFormatCommandPlansLimitESLintFixToScriptFiles(t *testing.T) {
	root := t.TempDir()
	plans := formatCommandPlans(root, []string{"app.ts", "README.md", "package.json"}, true)
	if len(plans) != 2 {
		t.Fatalf("plans = %#v, want prettier and eslint", plans)
	}
	if plans[0].Formatter != "prettier" || plans[1].Formatter != "eslint" {
		t.Fatalf("plans = %#v, want prettier then eslint", plans)
	}
	if strings.Join(plans[1].Paths, ",") != "app.ts" {
		t.Fatalf("eslint paths = %#v, want only app.ts", plans[1].Paths)
	}
}

func TestFormatCodeRejectsUnsupportedFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "app.bin"), []byte("not source\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	result := NewFormatCodeTool(root, nil).Execute(context.Background(), json.RawMessage(`{"paths":["app.bin"]}`), domain.ToolExecutionContext{WorkspaceRoot: root})
	if result.OK || result.ToolError == nil {
		t.Fatalf("result = %#v, want unsupported formatter failure", result)
	}
}

func TestApplyPatchRequiresApprovalThenAppliesAfterSavedRule(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	registry, runtime := service.toolsForWorkspace(root)
	if registry == nil || runtime == nil {
		t.Fatal("tool runtime was not created")
	}
	resolvedCh := make(chan domain.PermissionRequest, 1)
	service.SetPermissionResolvedHook(func(request domain.PermissionRequest) {
		resolvedCh <- request
	})
	patch := "*** Begin Patch\n*** Update File: README.md\n@@\n-old\n+new\n*** End Patch\n"
	call := domain.ChatToolCall{ID: "call_patch", Name: "apply_patch", Arguments: json.RawMessage(`{"patchText":` + strconv.Quote(patch) + `}`)}
	execCtx := domain.ToolExecutionContext{WorkspaceRoot: root, SessionID: "s1", TurnID: "t1"}
	resultCh := make(chan domain.ToolResult, 1)
	go func() {
		resultCh <- runtime.ExecuteWithContext(ctx, call, execCtx)
	}()
	var request domain.PermissionRequest
	for i := 0; i < 40; i++ {
		requests, err := service.ListPermissionRequests(ctx, "s1", domain.PermissionRequestStatusPending)
		if err != nil {
			t.Fatal(err)
		}
		if len(requests) > 0 {
			request = requests[0]
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if request.ID == "" {
		t.Fatal("permission request was not created")
	}
	if content, _ := os.ReadFile(filepath.Join(root, "README.md")); string(content) != "old\n" {
		t.Fatalf("file changed before approval: %q", content)
	}
	if _, err := service.ApprovePermissionRequest(ctx, domain.ApprovePermissionRequestInput{RequestID: request.ID, Remember: true}); err != nil {
		t.Fatal(err)
	}
	select {
	case resolved := <-resolvedCh:
		if resolved.ID != request.ID || resolved.Status != domain.PermissionRequestStatusApproved {
			t.Fatalf("resolved = %#v, want approved %s", resolved, request.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("permission resolved hook was not called")
	}
	var result domain.ToolResult
	select {
	case result = <-resultCh:
	case <-time.After(time.Second):
		t.Fatal("tool execution was not woken after approval")
	}
	if !result.OK {
		t.Fatalf("approved patch failed: %#v", result)
	}
	if len(result.Files) != 1 || result.Files[0].FullPath != filepath.ToSlash(filepath.Join(root, "README.md")) {
		t.Fatalf("files = %#v, want full path for patched file", result.Files)
	}
	if content, _ := os.ReadFile(filepath.Join(root, "README.md")); string(content) != "new\n" {
		t.Fatalf("file content = %q, want new", content)
	}
}

func TestAgentLoopPersistsToolCallWhilePermissionPending(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		if requestCount == 1 {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"tool_calls":[{"id":"call_edit","type":"function","function":{"name":"edit_file","arguments":"{\"path\":\"README.md\",\"oldString\":\"old\",\"newString\":\"new\"}"}}]}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"done"}}]}`))
	}))
	defer server.Close()
	if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{ProviderID: "custom-api", Type: "openai-compatible", BaseURL: server.URL, ModelID: "test-model", APIKey: "test-key", Method: "api-key"}); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, ProjectPath: root})
	if err != nil {
		t.Fatal(err)
	}
	run, err := service.SubmitSessionMessageStreaming(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "edit README"})
	if err != nil {
		t.Fatal(err)
	}

	var request domain.PermissionRequest
	var toolCalls []domain.ToolCall
	for i := 0; i < 40; i++ {
		requests, err := service.ListPermissionRequests(ctx, session.ID, domain.PermissionRequestStatusPending)
		if err != nil {
			t.Fatal(err)
		}
		toolCalls, err = service.ListToolCalls(ctx, session.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(requests) > 0 && len(toolCalls) > 0 {
			request = requests[0]
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if request.ID == "" {
		t.Fatal("permission request was not created")
	}
	if len(toolCalls) != 1 {
		t.Fatalf("toolCalls = %#v, want one pending visible tool call", toolCalls)
	}
	if toolCalls[0].ID != "call_edit" || toolCalls[0].TurnID != run.Turn.ID || toolCalls[0].Status != domain.ToolCallStatusPending {
		t.Fatalf("toolCalls[0] = %#v, want pending approval call_edit for prepared turn", toolCalls[0])
	}
	if pendingID, _ := toolCalls[0].Result["pendingApprovalId"].(string); pendingID != request.ID {
		t.Fatalf("toolCalls[0].Result[pendingApprovalId] = %q, want %q", pendingID, request.ID)
	}
	if request.ToolCallID != toolCalls[0].ID {
		t.Fatalf("request.ToolCallID = %q, want %q", request.ToolCallID, toolCalls[0].ID)
	}
	if _, err := service.ApprovePermissionRequest(ctx, domain.ApprovePermissionRequestInput{RequestID: request.ID}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 40; i++ {
		toolCalls, err = service.ListToolCalls(ctx, session.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(toolCalls) == 1 && toolCalls[0].Status == domain.ToolCallStatusSuccess {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("toolCalls = %#v, want call_edit to complete after approval", toolCalls)
}

func TestApplyPatchRejectsSensitivePathDeterministically(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	root := t.TempDir()
	registry, runtime := service.toolsForWorkspace(root)
	if registry == nil || runtime == nil {
		t.Fatal("tool runtime was not created")
	}
	patch := "*** Begin Patch\n*** Add File: .env\n+TOKEN=x\n*** End Patch\n"
	result := runtime.ExecuteWithContext(context.Background(), domain.ChatToolCall{
		ID: "call_secret", Name: "apply_patch", Arguments: json.RawMessage(`{"patchText":` + strconv.Quote(patch) + `}`),
	}, domain.ToolExecutionContext{WorkspaceRoot: root, SessionID: "s1", TurnID: "t1"})
	if result.OK || result.PermissionRequested || result.ToolError == nil || result.ToolError.Code != "permission_denied" {
		t.Fatalf("result = %#v, want deterministic permission denial", result)
	}
}

func TestPermissionModeFullAccessAllowsWorkspaceWrite(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, ProjectPath: root})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetPermissionMode(ctx, domain.PermissionModeInput{SessionID: session.ID, Mode: domain.PermissionModeFullAccess}); err != nil {
		t.Fatal(err)
	}
	_, runtime := service.toolsForWorkspace(root)
	patch := "*** Begin Patch\n*** Update File: README.md\n@@\n-old\n+new\n*** End Patch\n"
	result := runtime.ExecuteWithContext(ctx, domain.ChatToolCall{
		ID: "call_patch", Name: "apply_patch", Arguments: json.RawMessage(`{"patchText":` + strconv.Quote(patch) + `}`),
	}, domain.ToolExecutionContext{WorkspaceRoot: root, SessionID: session.ID, TurnID: "t1"})
	if !result.OK || result.PermissionRequested {
		t.Fatalf("result = %#v, want full access to allow workspace patch", result)
	}
	if content, _ := os.ReadFile(filepath.Join(root, "README.md")); string(content) != "new\n" {
		t.Fatalf("file content = %q, want new", content)
	}
}

func TestPermissionModeRequestApprovalOverridesFullAccess(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, ProjectPath: root})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetPermissionMode(ctx, domain.PermissionModeInput{SessionID: session.ID, Mode: domain.PermissionModeFullAccess}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetPermissionMode(ctx, domain.PermissionModeInput{SessionID: session.ID, Mode: domain.PermissionModeRequestApproval}); err != nil {
		t.Fatal(err)
	}
	_, runtime := service.toolsForWorkspace(root)
	patch := "*** Begin Patch\n*** Update File: README.md\n@@\n-old\n+new\n*** End Patch\n"
	approvalCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	result := runtime.ExecuteWithContext(approvalCtx, domain.ChatToolCall{
		ID: "call_patch", Name: "apply_patch", Arguments: json.RawMessage(`{"patchText":` + strconv.Quote(patch) + `}`),
	}, domain.ToolExecutionContext{WorkspaceRoot: root, SessionID: session.ID, TurnID: "t1"})
	if result.OK || !result.PermissionRequested || result.PendingApprovalID == "" {
		t.Fatalf("result = %#v, want request approval to create pending request", result)
	}
	if content, _ := os.ReadFile(filepath.Join(root, "README.md")); string(content) != "old\n" {
		t.Fatalf("file changed before approval: %q", content)
	}
}

func TestApplyPatchRejectsLegacyPatchArgument(t *testing.T) {
	root := t.TempDir()
	tool := NewApplyPatchTool(root)
	patch := "*** Begin Patch\n*** Add File: hello.txt\n+hello\n*** End Patch\n"
	result := tool.Execute(context.Background(), json.RawMessage(`{"patch":`+strconv.Quote(patch)+`}`), domain.ToolExecutionContext{WorkspaceRoot: root})
	if result.OK || result.ToolError == nil || !strings.Contains(result.Error, "patchText is required") {
		t.Fatalf("result = %#v, want patchText required failure", result)
	}
}

func TestWriteFileToolCreatesFile(t *testing.T) {
	root := t.TempDir()
	tool := NewWriteFileTool(root)
	result := tool.Execute(context.Background(), json.RawMessage(`{"path":"docs/summary.md","content":"hello\n"}`), domain.ToolExecutionContext{WorkspaceRoot: root})
	if !result.OK {
		t.Fatalf("write_file failed: %#v", result)
	}
	if len(result.Files) != 1 || result.Files[0].Type != "add" || result.Files[0].Additions != 1 || result.Files[0].Deletions != 0 {
		t.Fatalf("files = %#v, want created file with +1 -0", result.Files)
	}
	if result.Files[0].FullPath != filepath.ToSlash(filepath.Join(root, "docs", "summary.md")) {
		t.Fatalf("fullPath = %q, want absolute file path", result.Files[0].FullPath)
	}
	content, err := os.ReadFile(filepath.Join(root, "docs", "summary.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "hello\n" {
		t.Fatalf("content = %q, want hello", content)
	}
}

func TestWriteFileToolRejectsContentOverLineLimit(t *testing.T) {
	root := t.TempDir()
	tool := NewWriteFileTool(root)
	content := strings.Repeat("line\n", maxDirectWriteLines+1)
	result := tool.Execute(context.Background(), json.RawMessage(`{"path":"docs/long.md","content":`+strconv.Quote(content)+`}`), domain.ToolExecutionContext{WorkspaceRoot: root})
	if result.OK || result.ToolError == nil || !strings.Contains(result.Error, "exceeds 150 lines") || !strings.Contains(result.Error, "apply_patch") {
		t.Fatalf("result = %#v, want line limit failure", result)
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "long.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("long file should not be created, stat err = %v", err)
	}
}

func TestEditFileToolReplacesExactText(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "README.md")
	if err := os.WriteFile(target, []byte("one\ntwo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tool := NewEditFileTool(root)
	result := tool.Execute(context.Background(), json.RawMessage(`{"path":"README.md","oldString":"two","newString":"three"}`), domain.ToolExecutionContext{WorkspaceRoot: root})
	if !result.OK {
		t.Fatalf("edit_file failed: %#v", result)
	}
	if len(result.Files) != 1 || result.Files[0].Type != "edit" || result.Files[0].Additions != 1 || result.Files[0].Deletions != 1 {
		t.Fatalf("files = %#v, want edited file with +1 -1", result.Files)
	}
	if result.Files[0].FullPath != filepath.ToSlash(target) {
		t.Fatalf("fullPath = %q, want absolute file path", result.Files[0].FullPath)
	}
	content, _ := os.ReadFile(target)
	if string(content) != "one\nthree\n" {
		t.Fatalf("content = %q, want edited content", content)
	}
}

func TestReadFileToolReturnsSnapshotMetadata(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "README.md")
	if err := os.WriteFile(target, []byte("one\ntwo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tool := NewReadFileTool(root)
	result := tool.Execute(context.Background(), json.RawMessage(`{"path":"README.md"}`), domain.ToolExecutionContext{WorkspaceRoot: root})
	if !result.OK {
		t.Fatalf("read_file failed: %#v", result)
	}
	snapshot, _ := result.Structured["snapshot"].(fileSnapshot)
	if snapshot.Path != "README.md" || snapshot.SHA256 == "" || snapshot.ID == "" || snapshot.LineRange != "all" {
		t.Fatalf("snapshot = %#v, want path/hash/id/line range", result.Structured["snapshot"])
	}
}

func TestEditFileToolRejectsStaleExpectedHash(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "README.md")
	if err := os.WriteFile(target, []byte("one\ntwo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot, raw, err := readFileSnapshot("README.md", target, "all", false)
	if err != nil || string(raw) == "" {
		t.Fatalf("snapshot failed: %#v %v", snapshot, err)
	}
	if err := os.WriteFile(target, []byte("external\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tool := NewEditFileTool(root)
	result := tool.Execute(context.Background(), json.RawMessage(`{"path":"README.md","oldString":"one","newString":"three","expectedHash":"`+snapshot.SHA256+`"}`), domain.ToolExecutionContext{WorkspaceRoot: root})
	if result.OK || result.ToolError == nil || result.ToolError.Code != "stale_file" || !result.ToolError.Retry {
		t.Fatalf("result = %#v, want retryable stale_file", result)
	}
	content, _ := os.ReadFile(target)
	if string(content) != "external\n" {
		t.Fatalf("content = %q, want external edit preserved", content)
	}
}

func TestApplyPatchToolAcceptsFreeformRawPatch(t *testing.T) {
	root := t.TempDir()
	tool := NewApplyPatchTool(root)
	patch := "*** Begin Patch\n*** Add File: docs/spec.md\n+hello\n*** End Patch\n"
	result := tool.Execute(context.Background(), json.RawMessage(patch), domain.ToolExecutionContext{WorkspaceRoot: root})
	if !result.OK {
		t.Fatalf("apply_patch failed: %#v", result)
	}
	content, err := os.ReadFile(filepath.Join(root, "docs", "spec.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "hello\n" {
		t.Fatalf("content = %q, want hello", content)
	}
}

func TestEditFileToolRejectsReplacementOverLineLimit(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "README.md")
	if err := os.WriteFile(target, []byte("start\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tool := NewEditFileTool(root)
	replacement := strings.Repeat("line\n", maxDirectEditArgLines+1)
	result := tool.Execute(context.Background(), json.RawMessage(`{"path":"README.md","oldString":"start\n","newString":`+strconv.Quote(replacement)+`}`), domain.ToolExecutionContext{WorkspaceRoot: root})
	if result.OK || result.ToolError == nil || !strings.Contains(result.Error, "exceeds 150 lines") || !strings.Contains(result.Error, "apply_patch") {
		t.Fatalf("result = %#v, want line limit failure", result)
	}
	content, _ := os.ReadFile(target)
	if string(content) != "start\n" {
		t.Fatalf("content = %q, want original content", content)
	}
}

func TestEditFileToolRejectsAmbiguousMatch(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "README.md")
	if err := os.WriteFile(target, []byte("same\nsame\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tool := NewEditFileTool(root)
	result := tool.Execute(context.Background(), json.RawMessage(`{"path":"README.md","oldString":"same","newString":"next"}`), domain.ToolExecutionContext{WorkspaceRoot: root})
	if result.OK || result.ToolError == nil || !strings.Contains(result.Error, "multiple times") {
		t.Fatalf("result = %#v, want ambiguous match failure", result)
	}
}

func TestSafeJoinRejectsTraversalAndSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	if _, err := safeJoin(root, "../outside"); err == nil {
		t.Fatal("safeJoin allowed traversal")
	}
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := safeJoin(root, "link/secret.txt"); err == nil {
		t.Fatal("safeJoin allowed symlink escape")
	}
}

func TestReadFileToolReadsAndRejectsUnsafeInputs(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("TOKEN=x"), 0o600); err != nil {
		t.Fatal(err)
	}
	tool := NewReadFileTool(root)
	if result := tool.Execute(context.Background(), json.RawMessage(`{"path":"README.md"}`), domain.ToolExecutionContext{}); !result.OK || result.Content != "hello" {
		t.Fatalf("read result = %#v", result)
	}
	for name, args := range map[string]string{
		"outside":   `{"path":"../README.md"}`,
		"sensitive": `{"path":".env"}`,
		"directory": `{"path":"."}`,
	} {
		t.Run(name, func(t *testing.T) {
			if result := tool.Execute(context.Background(), json.RawMessage(args), domain.ToolExecutionContext{}); result.OK {
				t.Fatalf("result = %#v, want failure", result)
			}
		})
	}
}

func TestReadFileToolReadsLineRange(t *testing.T) {
	root := t.TempDir()
	content := "one\ntwo\nthree\nfour\n"
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	tool := NewReadFileTool(root)
	result := tool.Execute(context.Background(), json.RawMessage(`{"path":"README.md","offset":2,"limit":2}`), domain.ToolExecutionContext{})
	if !result.OK {
		t.Fatalf("read range failed: %#v", result)
	}
	if !strings.Contains(result.Content, "2|two") || !strings.Contains(result.Content, "3|three") {
		t.Fatalf("content = %q, want numbered selected lines", result.Content)
	}
	if !strings.Contains(result.Content, "offset 4") {
		t.Fatalf("content = %q, want continuation offset hint", result.Content)
	}
}

func TestReadFileToolAllowsUTF8ProbeEndingMidRune(t *testing.T) {
	root := t.TempDir()
	content := strings.Repeat("一", 342) + "\nsecond\n"
	if err := os.WriteFile(filepath.Join(root, "README摘要.md"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	tool := NewReadFileTool(root)
	result := tool.Execute(context.Background(), json.RawMessage(`{"path":"README摘要.md","offset":1,"limit":2}`), domain.ToolExecutionContext{})
	if !result.OK {
		t.Fatalf("read range failed: %#v", result)
	}
	if !strings.Contains(result.Content, "2|second") {
		t.Fatalf("content = %q, want UTF-8 text lines", result.Content)
	}
}

func TestListFilesIgnoresGeneratedDirectories(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"node_modules", ".git", "dist", "build", "src"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, dir, "file.txt"), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	result := NewListFilesTool(root).Execute(context.Background(), json.RawMessage(`{}`), domain.ToolExecutionContext{})
	if !result.OK {
		t.Fatalf("list failed: %#v", result)
	}
	if !strings.Contains(result.Content, "src/file.txt") {
		t.Fatalf("content = %q, want src/file.txt", result.Content)
	}
	for _, ignored := range []string{"node_modules", ".git", "dist/file.txt", "build/file.txt"} {
		if strings.Contains(result.Content, ignored) {
			t.Fatalf("content = %q, should ignore %s", result.Content, ignored)
		}
	}
}

func TestGlobToolFindsPathMatches(t *testing.T) {
	root := t.TempDir()
	for _, file := range []string{
		"README.md",
		"docs/README.txt",
		"src/main.go",
		"src/main_test.go",
		"node_modules/ignored.go",
	} {
		path := filepath.Join(root, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	tool := NewGlobTool(root)
	result := tool.Execute(context.Background(), json.RawMessage(`{"pattern":"**/README.{md,txt}"}`), domain.ToolExecutionContext{})
	if !result.OK {
		t.Fatalf("glob failed: %#v", result)
	}
	for _, want := range []string{"README.md", "docs/README.txt"} {
		if !strings.Contains(result.Content, want) {
			t.Fatalf("content = %q, want %s", result.Content, want)
		}
	}

	result = tool.Execute(context.Background(), json.RawMessage(`{"pattern":"*.go","path":"src"}`), domain.ToolExecutionContext{})
	if !result.OK || !strings.Contains(result.Content, "src/main.go") || !strings.Contains(result.Content, "src/main_test.go") {
		t.Fatalf("scoped glob result = %#v", result)
	}
	if strings.Contains(result.Content, "node_modules") {
		t.Fatalf("content = %q, should ignore generated directories", result.Content)
	}
	if result := tool.Execute(context.Background(), json.RawMessage(`{"pattern":""}`), domain.ToolExecutionContext{}); result.OK {
		t.Fatalf("empty pattern result = %#v, want failure", result)
	}
}

func TestSearchFilesFindsMatchesAndRejectsEmptyQuery(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\nfunc ToolRegistry() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tool := NewSearchFilesTool(root)
	result := tool.Execute(context.Background(), json.RawMessage(`{"query":"ToolRegistry"}`), domain.ToolExecutionContext{})
	if !result.OK || !strings.Contains(result.Content, "main.go:2:func ToolRegistry() {}") {
		t.Fatalf("search result = %#v", result)
	}
	if result := tool.Execute(context.Background(), json.RawMessage(`{"query":""}`), domain.ToolExecutionContext{}); result.OK {
		t.Fatalf("empty query result = %#v, want failure", result)
	}
}

func TestSearchFilesSupportsPathGlobAndLimit(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"src/main.go":       "needle one\nneedle two\n",
		"src/main.tsx":      "needle tsx\n",
		"docs/README.md":    "needle docs\n",
		"src/nested/lib.go": "needle nested\n",
	}
	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	tool := NewSearchFilesTool(root)
	result := tool.Execute(context.Background(), json.RawMessage(`{"query":"needle","path":"src","fileGlob":"**/*.go","limit":2}`), domain.ToolExecutionContext{})
	if !result.OK {
		t.Fatalf("search failed: %#v", result)
	}
	if !strings.Contains(result.Content, "src/main.go:1:needle one") {
		t.Fatalf("content = %q, want src/main.go match", result.Content)
	}
	for _, excluded := range []string{"main.tsx", "docs/README.md"} {
		if strings.Contains(result.Content, excluded) {
			t.Fatalf("content = %q, should exclude %s", result.Content, excluded)
		}
	}
	if strings.Count(result.Content, "needle") != 2 || !strings.Contains(result.Content, "[truncated: showing first 2 matches]") {
		t.Fatalf("content = %q, want limit truncation", result.Content)
	}
}

func TestWorkspaceDiscoveryToolsRespectGitignore(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		".gitignore":          "*.log\ngenerated/\nignored.txt\n!keep.log\n",
		"visible.go":          "package main\nfunc VisibleSymbol() {}\nneedle visible\n",
		"ignored.txt":         "needle ignored\n",
		"generated/hidden.go": "package main\nfunc HiddenSymbol() {}\nneedle hidden\n",
		"debug.log":           "needle debug\n",
		"keep.log":            "needle keep\n",
	}
	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	list := NewListFilesTool(root).Execute(context.Background(), json.RawMessage(`{}`), domain.ToolExecutionContext{})
	if !list.OK {
		t.Fatalf("list failed: %#v", list)
	}
	for _, want := range []string{"visible.go", "keep.log"} {
		if !strings.Contains(list.Content, want) {
			t.Fatalf("list content = %q, want %s", list.Content, want)
		}
	}
	for _, ignored := range []string{"ignored.txt", "generated/hidden.go", "debug.log"} {
		if strings.Contains(list.Content, ignored) {
			t.Fatalf("list content = %q, should ignore %s", list.Content, ignored)
		}
	}

	glob := NewGlobTool(root).Execute(context.Background(), json.RawMessage(`{"pattern":"**/*"}`), domain.ToolExecutionContext{})
	if !glob.OK || strings.Contains(glob.Content, "generated/hidden.go") || strings.Contains(glob.Content, "ignored.txt") || !strings.Contains(glob.Content, "keep.log") {
		t.Fatalf("glob result = %#v, want gitignore filtering with negation", glob)
	}

	search := NewSearchFilesTool(root).Execute(context.Background(), json.RawMessage(`{"query":"needle"}`), domain.ToolExecutionContext{})
	if !search.OK || !strings.Contains(search.Content, "visible.go") || !strings.Contains(search.Content, "keep.log") {
		t.Fatalf("search result = %#v, want visible matches", search)
	}
	for _, ignored := range []string{"ignored.txt", "generated/hidden.go", "debug.log"} {
		if strings.Contains(search.Content, ignored) {
			t.Fatalf("search content = %q, should ignore %s", search.Content, ignored)
		}
	}

	symbols := NewLSPSymbolSearchTool(root).Execute(context.Background(), json.RawMessage(`{"query":"Symbol"}`), domain.ToolExecutionContext{})
	if !symbols.OK || !strings.Contains(symbols.Content, "VisibleSymbol") || strings.Contains(symbols.Content, "HiddenSymbol") {
		t.Fatalf("symbols result = %#v, want gitignore filtering", symbols)
	}
}

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
		requestCount++
		var body struct {
			Messages []map[string]any `json:"messages"`
			Tools    []any            `json:"tools"`
			Stream   bool             `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.Tools) == 0 {
			t.Fatal("tools were not exposed")
		}
		if !body.Stream {
			t.Fatal("tool-enabled request should stream")
		}
		w.Header().Set("Content-Type", "application/json")
		if requestCount == 1 {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"tool_calls":[{"id":"call_read","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"README.md\"}"}}]}}]}`))
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
		requestCount++
		var body struct {
			Messages []map[string]any `json:"messages"`
			Tools    []any            `json:"tools"`
			Stream   bool             `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if !body.Stream {
			t.Fatal("tool-enabled request should stream")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		if requestCount == 1 {
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_read\",\"type\":\"function\",\"function\":{\"name\":\"read_file\",\"arguments\":\"{\\\"path\\\"\"}}]}}]}\n\n"))
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

func TestAgentLoopPlainChatAndMaxSteps(t *testing.T) {
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
	t.Run("max steps", func(t *testing.T) {
		service, cleanup := newSessionTestService(t)
		defer cleanup()
		ctx := context.Background()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"tool_calls":[{"id":"call_loop","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"README.md\"}"}}]}}]}`))
		}))
		defer server.Close()
		if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{ProviderID: "custom-api", Type: "openai-compatible", BaseURL: server.URL, ModelID: "test-model", APIKey: "test-key", Method: "api-key"}); err != nil {
			t.Fatal(err)
		}
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("loop"), 0o600); err != nil {
			t.Fatal(err)
		}
		session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, ProjectPath: root})
		if err != nil {
			t.Fatal(err)
		}
		_, err = service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "loop"})
		if !errors.Is(err, ErrMaxStepsExceeded) {
			t.Fatalf("err = %v, want ErrMaxStepsExceeded", err)
		}
	})
}

func TestLSPFallbackToolsReturnStructuredResults(t *testing.T) {
	root := t.TempDir()
	source := "package main\n\n// TODO: tighten behavior\nfunc Target() {}\nfunc Caller() { Target() }\n"
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	registry, err := NewReadOnlyToolRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
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
	registry, err := NewReadOnlyToolRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
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

func writeFakeLSP(output io.Writer, payload any) {
	raw, _ := json.Marshal(payload)
	_, _ = fmt.Fprintf(output, "Content-Length: %d\r\n\r\n", len(raw))
	_, _ = output.Write(raw)
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
