package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveProjectInstructionsOrdersGlobalRootAndNestedScopes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	writeInstructionTestFile(t, filepath.Join(home, ".config", "aivo", "AGENTS.md"), "global rule")
	writeInstructionTestFile(t, filepath.Join(root, "AGENTS.md"), "root rule")
	writeInstructionTestFile(t, filepath.Join(root, "src", "AGENTS.md"), "src rule")
	writeInstructionTestFile(t, filepath.Join(root, "docs", "AGENTS.md"), "docs rule")

	got := resolveProjectInstructions(root, []string{"src/main.go"})
	if !strings.Contains(got, "global rule") || !strings.Contains(got, "root rule") || !strings.Contains(got, "src rule") {
		t.Fatalf("instructions = %q", got)
	}
	if strings.Contains(got, "docs rule") {
		t.Fatalf("unrelated nested instructions leaked: %q", got)
	}
	if strings.Index(got, "global rule") > strings.Index(got, "root rule") || strings.Index(got, "root rule") > strings.Index(got, "src rule") {
		t.Fatalf("instruction scope order = %q", got)
	}
}

func TestResolveProjectInstructionsIgnoresExternalAndSymlinkTargets(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	external := t.TempDir()
	writeInstructionTestFile(t, filepath.Join(root, "AGENTS.md"), "root rule")
	writeInstructionTestFile(t, filepath.Join(external, "AGENTS.md"), "external rule")
	if err := os.Symlink(filepath.Join(external, "AGENTS.md"), filepath.Join(root, "CLAUDE.md")); err != nil {
		t.Fatal(err)
	}
	got := resolveProjectInstructions(root, []string{filepath.Join(external, "file.go")})
	if !strings.Contains(got, "root rule") || strings.Contains(got, "external rule") {
		t.Fatalf("instructions = %q", got)
	}
}

func TestResolveConfiguredRuntimeInstructionsLoadsGlobAndBoundedURL(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	writeInstructionTestFile(t, filepath.Join(root, "rules", "go.md"), "local configured rule")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("remote configured rule"))
	}))
	defer server.Close()
	writeInstructionTestFile(t, filepath.Join(root, "aivo.json"), `{"instructions":["rules/*.md","`+server.URL+`"]}`)
	got := resolveConfiguredRuntimeInstructions(context.Background(), root)
	if !strings.Contains(got, "local configured rule") || !strings.Contains(got, "remote configured rule") || !strings.Contains(got, server.URL) {
		t.Fatalf("configured instructions = %q", got)
	}
}

func writeInstructionTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
