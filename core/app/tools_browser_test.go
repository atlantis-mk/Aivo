package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestBrowserScreenshotSavesToWorkspaceByDefault(t *testing.T) {
	root := t.TempDir()
	var bridgeArgs map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Tool string         `json:"tool"`
			Args map[string]any `json:"args"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode bridge request: %v", err)
		}
		bridgeArgs = payload.Args
		response := browserBridgeResponse{
			OK: true,
			Structured: map[string]any{
				"url":      "http://127.0.0.1:5173",
				"title":    "Aivo",
				"mimeType": "image/png",
				"dataUrl":  "data:image/png;base64,cG5n",
				"bytes":    30,
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("encode bridge response: %v", err)
		}
	}))
	defer server.Close()

	tool := newBrowserTool("browser_screenshot", "screenshot", "Capture screenshot.", "browser.screenshot", "low", browserScreenshotSchema())
	tool.bridgeURL = server.URL
	tool.client = server.Client()

	result := tool.Execute(context.Background(), json.RawMessage(`{}`), domain.ToolExecutionContext{WorkspaceRoot: root})
	if !result.OK {
		t.Fatalf("result OK = false, error = %q", result.Error)
	}
	if len(result.Files) != 1 {
		t.Fatalf("files len = %d, want 1", len(result.Files))
	}
	file := result.Files[0]
	if !strings.HasPrefix(file.Path, "screenshot/browser-") || !strings.HasSuffix(file.Path, ".png") {
		t.Fatalf("file path = %q, want screenshot/browser-*.png", file.Path)
	}
	if file.Type != "add" {
		t.Fatalf("file type = %q, want add", file.Type)
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("eval workspace root: %v", err)
	}
	if !strings.HasPrefix(file.FullPath, filepath.ToSlash(filepath.Join(realRoot, "screenshot"))) {
		t.Fatalf("full path = %q, want under workspace screenshot dir", file.FullPath)
	}
	data, err := os.ReadFile(file.FullPath)
	if err != nil {
		t.Fatalf("read screenshot: %v", err)
	}
	if string(data) != "png" {
		t.Fatalf("screenshot data = %q, want png", string(data))
	}
	if saved, _ := result.Structured["saved"].(bool); !saved {
		t.Fatalf("structured saved = %v, want true", result.Structured["saved"])
	}
	if result.Structured["savedPath"] != file.Path {
		t.Fatalf("savedPath = %v, want %q", result.Structured["savedPath"], file.Path)
	}
	if !strings.Contains(result.Content, "Saved: "+file.Path) {
		t.Fatalf("content does not mention saved path: %q", result.Content)
	}
	if _, ok := bridgeArgs["save"]; ok {
		t.Fatalf("save argument leaked to browser bridge: %#v", bridgeArgs)
	}
}

func TestBrowserScreenshotCanSkipSaving(t *testing.T) {
	root := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := browserBridgeResponse{
			OK: true,
			Structured: map[string]any{
				"mimeType": "image/png",
				"dataUrl":  "data:image/png;base64,cG5n",
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("encode bridge response: %v", err)
		}
	}))
	defer server.Close()

	tool := newBrowserTool("browser_screenshot", "screenshot", "Capture screenshot.", "browser.screenshot", "low", browserScreenshotSchema())
	tool.bridgeURL = server.URL
	tool.client = server.Client()

	result := tool.Execute(context.Background(), json.RawMessage(`{"save":false}`), domain.ToolExecutionContext{WorkspaceRoot: root})
	if !result.OK {
		t.Fatalf("result OK = false, error = %q", result.Error)
	}
	if len(result.Files) != 0 {
		t.Fatalf("files len = %d, want 0", len(result.Files))
	}
	if saved, _ := result.Structured["saved"].(bool); saved {
		t.Fatalf("structured saved = true, want false")
	}
	if _, err := os.Stat(filepath.Join(root, "screenshot")); !os.IsNotExist(err) {
		t.Fatalf("screenshot dir stat error = %v, want not exist", err)
	}
}
