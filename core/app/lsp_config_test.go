package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"aivo/core/domain"
)

func TestBuiltInLSPCatalogCoversCommonLanguages(t *testing.T) {
	cases := map[string]string{
		"main.go": "go", "app.ts": "typescript", "main.py": "python", "lib.rs": "rust", "App.java": "java",
		"main.cpp": "cpp", "App.cs": "csharp", "app.rb": "ruby", "index.php": "php", "init.lua": "lua", "run.sh": "shellscript",
		"app.ex": "elixir", "main.zig": "zig", "App.fs": "fsharp", "App.swift": "swift", "Main.kt": "kotlin",
		"config.yaml": "yaml", "schema.prisma": "prisma", "main.dart": "dart", "main.ml": "ocaml", "main.tf": "terraform",
		"paper.tex": "latex", "main.gleam": "gleam", "core.clj": "clojure", "flake.nix": "nix", "doc.typ": "typst",
		"Main.hs": "haskell", "main.jl": "julia", "App.vue": "vue", "App.svelte": "svelte", "page.astro": "astro", "Dockerfile": "dockerfile",
	}
	for path, language := range cases {
		resolved, ok := resolveLSPDefinitionForPath("", path)
		if !ok || languageIDForLSPDefinition(resolved.Definition, path) != language {
			t.Fatalf("%s resolved to %#v, want %s", path, resolved, language)
		}
	}
}

func TestConfiguredLSPDefinitionOverridesBuiltInAndAddsLanguage(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	writeRuntimeConfigTestFile(t, filepath.Join(root, ".aivo", "config.json"), `{
  "languageServers": {
    "basedpyright": {
      "languageIds": ["python"],
      "extensions": [".py"],
      "rootMarkers": ["pyproject.toml"],
      "command": "custom-pyright",
      "args": ["--stdio"],
      "timeoutSeconds": 7
    },
    "elixir-ls": {
      "languageIds": ["elixir"],
      "extensions": [".ex", ".exs"],
      "rootMarkers": ["mix.exs"],
      "command": "elixir-ls"
    }
  }
}`)
	python, ok := resolveLSPDefinitionForPath(root, filepath.Join(root, "main.py"))
	if !ok || python.Definition.Command != "custom-pyright" || python.Definition.TimeoutSeconds != 7 {
		t.Fatalf("python definition = %#v", python)
	}
	elixir, ok := resolveLSPDefinitionForPath(root, filepath.Join(root, "lib.ex"))
	if !ok || languageIDForLSPDefinition(elixir.Definition, "lib.ex") != "elixir" {
		t.Fatalf("elixir definition = %#v", elixir)
	}
}

func TestLSPManagerRestartsClientWhenDefinitionRevisionChanges(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "app.ts"), []byte("export const value = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	binDir := t.TempDir()
	fakeServer := filepath.Join(binDir, "typescript-language-server")
	script := "#!/bin/sh\nAIVO_FAKE_LSP=1 AIVO_FAKE_LSP_FILE=app.ts exec \"$AIVO_TEST_BINARY\" -test.run=TestFakeLSPServer -- \"$@\"\n"
	if err := os.WriteFile(fakeServer, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AIVO_TEST_BINARY", os.Args[0])
	writeLSPRevisionConfig(t, root, fakeServer, "one")
	manager := newBoundedLSPManager()
	defer manager.Close()
	status, err := manager.Status(context.Background(), root)
	if err != nil || status.Status != domain.CodeIntelligenceStatusReady {
		t.Fatalf("first status = %#v err = %v", status, err)
	}
	key := root + "\x00typescript-language-server\x00typescript"
	first := manager.clients[key]
	writeLSPRevisionConfig(t, root, fakeServer, "two")
	status, err = manager.Status(context.Background(), root)
	if err != nil || status.Status != domain.CodeIntelligenceStatusReady {
		t.Fatalf("second status = %#v err = %v", status, err)
	}
	if first == manager.clients[key] {
		t.Fatal("language server client was not restarted after definition revision changed")
	}
}

func TestLSPSelectsNearestMonorepoRootAndStrictDenoServer(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	packageRoot := filepath.Join(root, "packages", "web")
	if err := os.MkdirAll(packageRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, "package.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, "app.ts"), []byte("export const value = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	binDir := t.TempDir()
	fakeServer := filepath.Join(binDir, "typescript-language-server")
	script := "#!/bin/sh\nAIVO_FAKE_LSP=1 AIVO_FAKE_LSP_FILE=app.ts exec \"$AIVO_TEST_BINARY\" -test.run=TestFakeLSPServer -- \"$@\"\n"
	if err := os.WriteFile(fakeServer, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("AIVO_TEST_BINARY", os.Args[0])
	manager := newBoundedLSPManager()
	defer manager.Close()
	selection, err := manager.clientForPath(context.Background(), root, "packages/web/app.ts")
	if err != nil || selection.status.Status != domain.CodeIntelligenceStatusReady {
		t.Fatalf("selection = %#v err = %v", selection, err)
	}
	if canonicalLSPPath(selection.serverRoot) != canonicalLSPPath(packageRoot) || selection.resolved.Name != "typescript-language-server" {
		t.Fatalf("selection = %#v", selection)
	}

	denoRoot := filepath.Join(root, "tools", "script")
	if err := os.MkdirAll(denoRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(denoRoot, "deno.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	deno, ok := resolveLSPDefinitionForPath(root, filepath.Join(denoRoot, "mod.ts"))
	if !ok || deno.Name != "deno" {
		t.Fatalf("Deno definition = %#v ok = %v", deno, ok)
	}
}

func writeLSPRevisionConfig(t *testing.T, root string, command string, revision string) {
	t.Helper()
	content := fmt.Sprintf(`{
  "languageServers": {
    "typescript-language-server": {
      "languageIds": ["typescript", "typescriptreact", "javascript", "javascriptreact"],
      "extensions": [".ts", ".tsx", ".js", ".jsx"],
      "rootMarkers": ["package.json"],
      "command": %q,
      "args": ["--stdio"],
      "initializationOptions": {"revision": %q}
    }
  }
}`, command, revision)
	writeRuntimeConfigTestFile(t, filepath.Join(root, ".aivo", "config.json"), content)
}
