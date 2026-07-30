package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestToolAssemblyDefersLongTailTools(t *testing.T) {
	registry := NewRegistry()
	for _, tool := range []domain.Tool{
		phase6EchoTool{spec: domain.ToolSpec{Name: "read_file", Description: "Read file", InputSchema: map[string]any{"type": "object"}, Category: "filesystem", Capability: "filesystem.read", Toolsets: []string{"safe", "coding"}}},
		phase6EchoTool{spec: domain.ToolSpec{Name: "apply_patch", Description: "Patch files", InputSchema: map[string]any{"type": "object"}, Category: "filesystem", Capability: "filesystem.patch", Toolsets: []string{"coding"}}},
		phase6EchoTool{spec: domain.ToolSpec{Name: "update_plan", Description: "Update plan", InputSchema: map[string]any{"type": "object"}, Category: "plan", Capability: "plan.write", Toolsets: []string{"safe", "personal"}}},
		phase6EchoTool{spec: domain.ToolSpec{Name: "automation_list", Description: "List automations", InputSchema: map[string]any{"type": "object"}, Category: "automation", Capability: "scheduler.read", Toolsets: []string{"safe", "personal"}}},
		phase6EchoTool{spec: domain.ToolSpec{Name: "plugin_echo", Description: "Echo plugin text", InputSchema: map[string]any{"type": "object"}, Category: "plugin", Capability: "plugin.read", Toolsets: []string{"plugin", "coding"}}},
	} {
		if err := registry.Register(tool); err != nil {
			t.Fatal(err)
		}
	}
	var runtime *ToolRuntime
	if err := registry.RegisterScoped(NewToolSearchTool(registry), domain.ToolSourceBridge, "tool_discovery", ""); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterScoped(NewToolDetailTool(registry), domain.ToolSourceBridge, "tool_discovery", ""); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterScoped(NewToolCallTool(registry, func() *ToolRuntime { return runtime }), domain.ToolSourceBridge, "tool_discovery", ""); err != nil {
		t.Fatal(err)
	}

	assembly := AssembleToolSpecs(registry, registry.Specs())
	if !assembly.Activated {
		t.Fatalf("assembly.Activated = false, want long-tail deferral")
	}
	names := map[string]int{}
	for _, spec := range assembly.Specs {
		names[spec.Name]++
	}
	for _, name := range []string{"read_file", "apply_patch", "update_plan", ToolResolveName} {
		if names[name] != 1 {
			t.Fatalf("visible %s count = %d; specs = %#v", name, names[name], assembly.Specs)
		}
	}
	for _, name := range []string{ToolSearchName, ToolListName, ToolDetailName, ToolCallName} {
		if names[name] != 0 {
			t.Fatalf("legacy bridge tool %s was directly visible: %#v", name, assembly.Specs)
		}
	}
	for _, name := range []string{"automation_list", "plugin_echo"} {
		if names[name] != 0 {
			t.Fatalf("long-tail tool %s was directly visible: %#v", name, assembly.Specs)
		}
	}

	runtime = NewToolRuntime(registry, t.TempDir())
	search := runtime.ExecuteWithContext(context.Background(), domain.ChatToolCall{ID: "search", Name: ToolSearchName, Arguments: json.RawMessage(`{"query":"automation"}`)}, domain.ToolExecutionContext{AllowedToolsets: []string{"safe", "coding", "personal"}})
	if !search.OK || !strings.Contains(search.Content, "automation_list") {
		t.Fatalf("search = %#v, want deferred automation_list", search)
	}
}

func TestToolAssemblyCanExplicitlyActivateMatchedTools(t *testing.T) {
	registry := NewRegistry()
	for _, tool := range []domain.Tool{
		phase6EchoTool{spec: domain.ToolSpec{Name: "read_file", Description: "Read file", InputSchema: map[string]any{"type": "object"}, Category: "filesystem", Capability: "filesystem.read", Toolsets: []string{"safe", "coding"}}},
		phase6EchoTool{spec: domain.ToolSpec{Name: "automation_list", Description: "List automations", InputSchema: map[string]any{"type": "object"}, Category: "automation", Capability: "scheduler.read", Toolsets: []string{"safe", "personal"}}},
		phase6EchoTool{spec: domain.ToolSpec{Name: "plugin_echo", Description: "Echo plugin text", InputSchema: map[string]any{"type": "object"}, Category: "plugin", Capability: "plugin.read", Toolsets: []string{"plugin", "coding"}}},
	} {
		if err := registry.Register(tool); err != nil {
			t.Fatal(err)
		}
	}
	var runtime *ToolRuntime
	if err := registry.RegisterScoped(NewToolSearchTool(registry), domain.ToolSourceBridge, "tool_discovery", ""); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterScoped(NewToolDetailTool(registry), domain.ToolSourceBridge, "tool_discovery", ""); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterScoped(NewToolCallTool(registry, func() *ToolRuntime { return runtime }), domain.ToolSourceBridge, "tool_discovery", ""); err != nil {
		t.Fatal(err)
	}

	activated := map[string]bool{"plugin_echo": true}

	assembly := AssembleToolSpecsWithActivated(registry, registry.Specs(), activated)
	names := map[string]int{}
	for _, spec := range assembly.Specs {
		names[spec.Name]++
	}
	if names["plugin_echo"] != 1 {
		t.Fatalf("plugin_echo visible count = %d; specs = %#v", names["plugin_echo"], assembly.Specs)
	}
	if names["automation_list"] != 0 {
		t.Fatalf("unmatched automation_list was directly visible: %#v", assembly.Specs)
	}
	if names[ToolResolveName] != 1 || names[ToolSearchName] != 0 || names[ToolListName] != 0 || names[ToolDetailName] != 0 || names[ToolCallName] != 0 {
		t.Fatalf("bridge tools missing after partial activation: %#v", assembly.Specs)
	}
}

func TestListToolCatalogWithoutWorkspaceIncludesGlobalTools(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	entries, err := service.ListToolCatalog(context.Background(), domain.ToolCatalogInput{})
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, entry := range entries {
		names[entry.Name] = true
	}
	for _, name := range []string{"web_fetch", "web_search", "agent_mode_list", ToolResolveName} {
		if !names[name] {
			t.Fatalf("global tool catalog missing %s; entries = %#v", name, entries)
		}
	}
	for _, name := range []string{"browser_state", "browser_navigate", "browser_snapshot"} {
		if names[name] {
			t.Fatalf("global tool catalog still exposes removed tool %s; entries = %#v", name, entries)
		}
	}
}

func TestListToolCatalogWithoutWorkspaceUsesCachedExternalTools(t *testing.T) {
	store := &memoryProviderStore{
		plugins: []domain.PluginInstall{{
			ID: "fixture-plugin", Enabled: true, Status: domain.PluginStatusEnabled,
			Manifest: domain.PluginManifest{
				ID: "fixture-plugin", Version: "1",
				Entrypoint: domain.PluginEntrypoint{Command: "definitely-not-a-real-plugin-command"},
				Tools: []domain.PluginDeclaredTool{{
					Name: "plugin_manifest_tool", Description: "Manifest declared tool",
					InputSchema: map[string]any{"type": "object"}, Capability: "plugin.read", Toolsets: []string{"plugin", "coding"},
				}},
			},
		}},
		mcpServers: []domain.MCPServerConfig{{
			ID: "cached-mcp", Name: "Cached MCP", Transport: domain.MCPTransportStdio,
			Command: "definitely-not-a-real-mcp-command", Enabled: true,
		}},
		mcpTools: map[string][]domain.MCPToolRecord{
			"cached-mcp": {{
				ID: "cached-mcp:list", ServerID: "cached-mcp", Name: "list",
				Description: "Cached MCP tool", InputSchema: map[string]any{"type": "object"},
				Capability: "mcp.read", RiskLevel: "medium",
			}},
		},
	}
	service := NewService(store)
	entries, err := service.ListToolCatalog(context.Background(), domain.ToolCatalogInput{WorkspaceRoot: ""})
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, entry := range entries {
		names[entry.Name] = true
	}
	if !names["plugin_manifest_tool"] {
		t.Fatalf("missing cached plugin manifest tool; entries = %#v", entries)
	}
	if !names["mcp_Cached_MCP_list"] {
		t.Fatalf("missing cached MCP tool; entries = %#v", entries)
	}
}

func TestListToolCatalogWithWorkspaceUsesCachedExternalTools(t *testing.T) {
	store := &memoryProviderStore{
		plugins: []domain.PluginInstall{{
			ID: "fixture-plugin", Enabled: true, Status: domain.PluginStatusEnabled,
			Manifest: domain.PluginManifest{
				ID: "fixture-plugin", Version: "1",
				Entrypoint: domain.PluginEntrypoint{Command: "definitely-not-a-real-plugin-command"},
				Tools: []domain.PluginDeclaredTool{{
					Name: "plugin_manifest_tool", Description: "Manifest declared tool",
					InputSchema: map[string]any{"type": "object"}, Capability: "plugin.read", Toolsets: []string{"plugin", "coding"},
				}},
			},
		}},
		mcpServers: []domain.MCPServerConfig{{
			ID: "cached-mcp", Name: "Cached MCP", Transport: domain.MCPTransportStdio,
			Command: "definitely-not-a-real-mcp-command", Enabled: true,
		}},
		mcpTools: map[string][]domain.MCPToolRecord{
			"cached-mcp": {{
				ID: "cached-mcp:list", ServerID: "cached-mcp", Name: "list",
				Description: "Cached MCP tool", InputSchema: map[string]any{"type": "object"},
				Capability: "mcp.read", RiskLevel: "medium",
			}},
		},
	}
	service := NewService(store)
	entries, err := service.ListToolCatalog(context.Background(), domain.ToolCatalogInput{WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, entry := range entries {
		names[entry.Name] = true
	}
	for _, name := range []string{"read_file", "bash", "plugin_manifest_tool", "mcp_Cached_MCP_list"} {
		if !names[name] {
			t.Fatalf("missing %s; entries = %#v", name, entries)
		}
	}
}
