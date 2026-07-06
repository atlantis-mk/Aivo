package app

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestApplyPatchDraftFilesParsesPartialAddFileArguments(t *testing.T) {
	args := json.RawMessage(`{"patchText":"*** Begin Patch\n*** Add File: docs/spec.md\n+one\n+two`)

	patchText, files := applyPatchDraftFiles(args, "")
	if patchText == "" {
		t.Fatal("patchText is empty")
	}
	if len(files) != 1 {
		t.Fatalf("files = %#v, want one file", files)
	}
	file := files[0]
	if file.Path != "docs/spec.md" || file.Type != "add" || file.Additions != 2 || file.Deletions != 0 {
		t.Fatalf("file = %#v", file)
	}
}

func TestApplyPatchDraftFilesParsesPartialUpdateArguments(t *testing.T) {
	args := json.RawMessage(`{"patchText":"*** Begin Patch\n*** Update File: src/app.ts\n@@\n-old\n+new`)

	_, files := applyPatchDraftFiles(args, "")
	if len(files) != 1 {
		t.Fatalf("files = %#v, want one file", files)
	}
	file := files[0]
	if file.Path != "src/app.ts" || file.Type != "update" || file.Additions != 1 || file.Deletions != 1 {
		t.Fatalf("file = %#v", file)
	}
}

func TestApplyPatchDraftFilesIncludesFullPath(t *testing.T) {
	root := t.TempDir()
	args := json.RawMessage(`{"patchText":"*** Begin Patch\n*** Update File: src/app.ts\n@@\n-old\n+new"}`)

	_, files := applyPatchDraftFiles(args, root)
	if len(files) != 1 {
		t.Fatalf("files = %#v, want one file", files)
	}
	if files[0].FullPath != filepath.ToSlash(filepath.Join(root, "src", "app.ts")) {
		t.Fatalf("fullPath = %q, want absolute file path", files[0].FullPath)
	}
}
