package app

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"aivo/core/domain"
)

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
