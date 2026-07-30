package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aivo/core/domain"
)

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

func TestReadFileToolAddsNestedInstructionsOnlyToModelContent(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pkg", "AGENTS.md"), []byte("Use package rules."), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pkg", "file.go"), []byte("package pkg"), 0o600); err != nil {
		t.Fatal(err)
	}
	result := NewReadFileTool(root).Execute(context.Background(), json.RawMessage(`{"path":"pkg/file.go"}`), domain.ToolExecutionContext{})
	if result.Content != "package pkg" {
		t.Fatalf("content = %q", result.Content)
	}
	if !strings.Contains(result.ModelContent, "Use package rules.") || !strings.Contains(result.ModelContent, "pkg/AGENTS.md") {
		t.Fatalf("model content = %q", result.ModelContent)
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

func TestListFilesSkipsHiddenEntriesUnlessIncluded(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"README.md":                "visible",
		".env.local":               "hidden file",
		".config/settings.json":    "hidden dir",
		"src/.generated/config.ts": "hidden nested dir",
		"src/visible.ts":           "visible nested",
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

	tool := NewListFilesTool(root)
	result := tool.Execute(context.Background(), json.RawMessage(`{}`), domain.ToolExecutionContext{})
	if !result.OK {
		t.Fatalf("list failed: %#v", result)
	}
	for _, want := range []string{"README.md", "src/visible.ts"} {
		if !strings.Contains(result.Content, want) {
			t.Fatalf("content = %q, want %s", result.Content, want)
		}
	}
	for _, hidden := range []string{".env.local", ".config/settings.json", "src/.generated/config.ts"} {
		if strings.Contains(result.Content, hidden) {
			t.Fatalf("content = %q, should hide %s by default", result.Content, hidden)
		}
	}

	withHidden := tool.Execute(context.Background(), json.RawMessage(`{"includeHidden":true}`), domain.ToolExecutionContext{})
	if !withHidden.OK {
		t.Fatalf("list with hidden failed: %#v", withHidden)
	}
	for _, want := range []string{".env.local", ".config/settings.json", "src/.generated/config.ts"} {
		if !strings.Contains(withHidden.Content, want) {
			t.Fatalf("content = %q, want hidden entry %s", withHidden.Content, want)
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
