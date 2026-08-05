package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"aivo/core/domain"
)

type toolNameTestTool struct{ name string }

func (t toolNameTestTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{Name: t.name, InputSchema: map[string]any{"type": "object"}}
}

func (t toolNameTestTool) Execute(context.Context, json.RawMessage, domain.ToolExecutionContext) domain.ToolResult {
	return domain.ToolResult{Name: t.name, OK: true}
}

func TestRegistryRejectsProviderUnsafeToolNames(t *testing.T) {
	for _, name := range []string{"mcp.chrome.list_tabs", "chrome tabs", "工具", " padded", strings.Repeat("a", 65)} {
		t.Run(name, func(t *testing.T) {
			registry := NewRegistry()
			if err := registry.Register(toolNameTestTool{name: name}); err == nil {
				t.Fatalf("Register(%q) succeeded, want global tool-name validation error", name)
			}
		})
	}
}

func TestRegistryAcceptsGlobalToolNameBoundaryAndRejectsBatchAtomically(t *testing.T) {
	registry := NewRegistry()
	boundary := strings.Repeat("a", toolNameMaxLength-2) + "_-"
	if err := registry.Register(toolNameTestTool{name: boundary}); err != nil {
		t.Fatalf("64-byte Provider-safe name rejected: %v", err)
	}
	batch := []domain.Tool{toolNameTestTool{name: "example_safe"}, toolNameTestTool{name: "example.invalid"}}
	if err := registry.RegisterScopedBatch(batch, domain.ToolSourceExtension, "example", "v1"); err == nil {
		t.Fatal("invalid batch succeeded")
	}
	if _, ok := registry.Get("example_safe"); ok {
		t.Fatal("valid prefix of invalid batch was registered")
	}
}

func TestManifestAndGeneratedMCPToolsUseGlobalCanonicalNames(t *testing.T) {
	invalidManifest := []byte(`{
		"schemaVersion":1,"id":"com.example.invalid","name":"Invalid","version":"1","apiVersion":"1",
		"runtime":{"type":"builtin"},
		"contributes":{"tools":[{"name":"example.invalid","schema":{"type":"object"}}]}
	}`)
	if _, err := LoadBuiltinExtensionManifest(invalidManifest); err == nil {
		t.Fatal("Manifest v1 accepted a dotted tool name")
	}

	server := domain.MCPServerConfig{ID: "Docs.Server", Name: "Docs Server"}
	left := domain.MCPToolRecord{Name: "search.docs"}
	right := domain.MCPToolRecord{Name: "search-docs"}
	leftName := mcpToolName(server, left)
	if leftName != "mcp_docs_server_search_docs" || !providerSafeToolName(leftName) {
		t.Fatalf("generated MCP name = %q, want one Provider-safe canonical name", leftName)
	}
	if got := mcpToolName(server, left); got != leftName {
		t.Fatalf("generated MCP name changed: %q != %q", got, leftName)
	}
	if collided := mcpToolName(server, right); collided != leftName {
		t.Fatalf("collision fixture did not collide: %q != %q", collided, leftName)
	}
	registry := NewRegistry()
	if err := registry.RegisterScoped(toolNameTestTool{name: leftName}, domain.ToolSourceExtension, "mcp_docs_server", "v1"); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterScoped(toolNameTestTool{name: leftName}, domain.ToolSourceExtension, "mcp_docs_server", "v1"); err == nil {
		t.Fatal("canonical MCP collision was not rejected")
	}
	if left.Name != "search.docs" {
		t.Fatalf("upstream MCP name changed to %q", left.Name)
	}
}
