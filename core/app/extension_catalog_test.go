package app

import (
	"context"
	"testing"

	"aivo/core/domain"
)

func TestToolAssemblyDefersLongTailTools(t *testing.T) {
	registry := NewRegistry()
	for _, tool := range []domain.Tool{
		phase6EchoTool{spec: domain.ToolSpec{Name: "read", Description: "Read file", InputSchema: map[string]any{"type": "object"}, Category: "filesystem", Capability: "filesystem.read", Toolsets: []string{"safe", "coding"}}},
		phase6EchoTool{spec: domain.ToolSpec{Name: "bash", Description: "Run Bash", InputSchema: map[string]any{"type": "object"}, Category: "process", Capability: "process.exec", Toolsets: []string{"coding"}}},
		phase6EchoTool{spec: domain.ToolSpec{Name: "edit", Description: "Edit file", InputSchema: map[string]any{"type": "object"}, Category: "filesystem", Capability: "filesystem.write", Toolsets: []string{"coding"}}},
		phase6EchoTool{spec: domain.ToolSpec{Name: "write", Description: "Write file", InputSchema: map[string]any{"type": "object"}, Category: "filesystem", Capability: "filesystem.write", Toolsets: []string{"coding"}}},
		phase6EchoTool{spec: domain.ToolSpec{Name: "update_plan", Description: "Update plan", InputSchema: map[string]any{"type": "object"}, Category: "plan", Capability: "plan.write", Toolsets: []string{"safe", "personal"}}},
		phase6EchoTool{spec: domain.ToolSpec{Name: "ask_user", Description: "Ask user", InputSchema: map[string]any{"type": "object"}, Category: "interaction", Capability: "user.question", Toolsets: []string{"safe", "personal", "coding"}}},
		phase6EchoTool{spec: domain.ToolSpec{Name: "automation_list", Description: "List automations", InputSchema: map[string]any{"type": "object"}, Category: "automation", Capability: "scheduler.read", Toolsets: []string{"safe", "personal"}}},
		phase6EchoTool{spec: domain.ToolSpec{Name: "extension_echo", Description: "Echo extension text", InputSchema: map[string]any{"type": "object"}, Category: "extension", Capability: "extension.read", Toolsets: []string{"extension", "coding"}}},
	} {
		if err := registry.Register(tool); err != nil {
			t.Fatal(err)
		}
	}
	assembly := AssembleToolSpecs(registry, registry.Specs())
	if assembly.Activated {
		t.Fatalf("assembly.Activated = true, want only the core surface")
	}
	names := map[string]int{}
	for _, spec := range assembly.Specs {
		names[spec.Name]++
	}
	for _, name := range []string{"read", "bash", "edit", "write", "update_plan", "ask_user"} {
		if names[name] != 1 {
			t.Fatalf("visible %s count = %d; specs = %#v", name, names[name], assembly.Specs)
		}
	}
	for _, name := range []string{ToolSearchName, ToolListName, ToolDetailName, ToolCallName} {
		if names[name] != 0 {
			t.Fatalf("legacy bridge tool %s was directly visible: %#v", name, assembly.Specs)
		}
	}
	for _, name := range []string{"automation_list", "extension_echo"} {
		if names[name] != 0 {
			t.Fatalf("long-tail tool %s was directly visible: %#v", name, assembly.Specs)
		}
	}

	if assembly.DeferredCount != 2 {
		t.Fatalf("deferred count = %d, want automation and extension candidates", assembly.DeferredCount)
	}
}

func TestToolAssemblyCanExplicitlyActivateMatchedTools(t *testing.T) {
	registry := NewRegistry()
	for _, tool := range []domain.Tool{
		phase6EchoTool{spec: domain.ToolSpec{Name: "read", Description: "Read file", InputSchema: map[string]any{"type": "object"}, Category: "filesystem", Capability: "filesystem.read", Toolsets: []string{"safe", "coding"}}},
		phase6EchoTool{spec: domain.ToolSpec{Name: "automation_list", Description: "List automations", InputSchema: map[string]any{"type": "object"}, Category: "automation", Capability: "scheduler.read", Toolsets: []string{"safe", "personal"}}},
		phase6EchoTool{spec: domain.ToolSpec{Name: "extension_echo", Description: "Echo extension text", InputSchema: map[string]any{"type": "object"}, Category: "extension", Capability: "extension.read", Toolsets: []string{"extension", "coding"}}},
	} {
		if err := registry.Register(tool); err != nil {
			t.Fatal(err)
		}
	}
	activated := map[string]bool{"extension_echo": true}

	assembly := AssembleToolSpecsWithActivated(registry, registry.Specs(), activated)
	names := map[string]int{}
	for _, spec := range assembly.Specs {
		names[spec.Name]++
	}
	if names["extension_echo"] != 1 {
		t.Fatalf("extension_echo visible count = %d; specs = %#v", names["extension_echo"], assembly.Specs)
	}
	if names["automation_list"] != 0 {
		t.Fatalf("unmatched automation_list was directly visible: %#v", assembly.Specs)
	}
	if names["read"] != 1 || names[ToolResolveName] != 0 || names[ToolSearchName] != 0 || names[ToolListName] != 0 || names[ToolDetailName] != 0 || names[ToolCallName] != 0 {
		t.Fatalf("core/extension surface is not minimal after activation: %#v", assembly.Specs)
	}
}

func TestPreCallActivationSeparatesManualAutomaticAndManualOnlyPolicies(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	registry := NewRegistry()
	for _, spec := range []domain.ToolSpec{
		{Name: "example_manual", Description: "manual", InputSchema: map[string]any{"type": "object"}, Category: "extension", Toolsets: []string{"coding"}, ActivationPolicy: "manual"},
		{Name: "example_default", Description: "default", InputSchema: map[string]any{"type": "object"}, Category: "extension", Toolsets: []string{"coding"}, ActivationPolicy: "default"},
		{Name: "example_auto", Description: "auto", InputSchema: map[string]any{"type": "object"}, Category: "extension", Toolsets: []string{"coding"}, ActivationPolicy: "auto"},
	} {
		if err := registry.RegisterScoped(phase6EchoTool{spec: spec}, domain.ToolSourceExtension, "example", "v1"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := service.SetSessionActiveTools(ctx, domain.SessionActiveToolsInput{SessionID: session.ID, ToolNames: []string{"example_manual"}}); err != nil {
		t.Fatal(err)
	}
	activated, candidates := service.preCallToolCandidates(ctx, session.ID, "turn-1", registry, registry.Specs())
	if activated["example_manual"] != "manual" {
		t.Fatalf("activated = %#v, want only the manual conversation tool", activated)
	}
	if activated["example_auto"] != "" || activated["example_default"] != "" {
		t.Fatalf("activated = %#v, automatic candidates must not activate without a resolver match", activated)
	}
	if len(candidates) != 2 || candidates[0].Name != "example_auto" || candidates[1].Name != "example_default" {
		t.Fatalf("candidates = %#v, want auto and legacy-default tools to require selection", candidates)
	}
	assembly := AssembleToolSpecsWithSources(registry, registry.Specs(), activated)
	if len(assembly.Specs) != 1 || len(assembly.Snapshot.Tools) != 1 || assembly.Specs[0].Name != "example_manual" {
		t.Fatalf("assembly = %#v, want exactly the manual tool before automatic selection", assembly)
	}
}

func TestListToolCatalogWithoutWorkspaceContainsGloballyManageableBuiltins(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	entries, err := service.ListToolCatalog(context.Background(), domain.ToolCatalogInput{})
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, entry := range entries {
		names[entry.Name] = true
	}
	for _, name := range []string{projectQueryToolName, projectAddToolName, projectAssociateToolName} {
		if !names[name] {
			t.Fatalf("global catalog missing builtin project extension tool %q; entries = %#v", name, entries)
		}
		entry, _ := catalogEntryNamed(entries, name)
		if entry.SelectionGroup == nil || entry.SelectionGroup.ID != "extension_aivo_projects_projects" || entry.SelectionGroup.Name != "Aivo Projects" {
			t.Fatalf("builtin project tool %q group = %#v", name, entry.SelectionGroup)
		}
	}
	if !names[toolRegistrationMCPName] {
		t.Fatalf("global catalog missing builtin tool registration extension tool %q; entries = %#v", toolRegistrationMCPName, entries)
	}
	if registrationTool, _ := catalogEntryNamed(entries, toolRegistrationMCPName); registrationTool.SelectionGroup != nil {
		t.Fatalf("individual builtin tool unexpectedly grouped: %#v", registrationTool)
	}
	for _, name := range []string{"read", "bash", "edit", "write", "update_plan", "ask_user", "grep", "find", "ls"} {
		if !names[name] {
			t.Fatalf("global catalog missing manageable builtin tool %q; entries = %#v", name, entries)
		}
	}
	for _, legacy := range []string{"search_files", "glob", "list_files"} {
		if names[legacy] {
			t.Fatalf("global catalog still exposes removed tool name %q; entries = %#v", legacy, entries)
		}
	}
	if names["aivo_tools_list_mcp"] {
		t.Fatalf("global catalog still exposes removed MCP source-list tool; entries = %#v", entries)
	}
	if len(names) != 13 {
		t.Fatalf("global catalog does not contain nine core/optional builtins and builtin Host extensions; entries = %#v", entries)
	}
}

func TestToolRegistryReservesHostSelectionControlName(t *testing.T) {
	registry := NewRegistry()
	tool := phase6EchoTool{spec: domain.ToolSpec{Name: ToolResolveName, InputSchema: map[string]any{"type": "object"}, Toolsets: []string{"coding"}}}
	if err := registry.RegisterScoped(tool, domain.ToolSourceExtension, "malicious", "v1"); err == nil {
		t.Fatal("extension registered the Host-owned tool_resolve name")
	}
	if err := registry.RegisterScoped(NewToolResolveTool(registry, nil, nil), domain.ToolSourceBridge, "tool_selection", "v1"); err != nil {
		t.Fatalf("Host control registration failed: %v", err)
	}
}

func TestToolRegistryReservesDefaultHostControlNames(t *testing.T) {
	for _, name := range []string{"update_plan", "ask_user"} {
		registry := NewRegistry()
		tool := phase6EchoTool{spec: domain.ToolSpec{Name: name, InputSchema: map[string]any{"type": "object"}, Toolsets: []string{"coding"}}}
		if err := registry.RegisterScoped(tool, domain.ToolSourceExtension, "malicious", "v1"); err == nil {
			t.Fatalf("extension registered the Host-owned %s name", name)
		}
	}
}

func TestListToolCatalogWithoutWorkspaceAdaptsCachedEnabledMCPTools(t *testing.T) {
	store := &memoryProviderStore{
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
	var adapted domain.ToolCatalogEntry
	for _, entry := range entries {
		names[entry.Name] = true
		if entry.Name == "mcp_cached_mcp_list" {
			adapted = entry
		}
	}
	if names["mcp_Cached_MCP_list"] || !names["mcp_cached_mcp_list"] {
		t.Fatalf("catalog did not expose the namespaced MCP adapters; entries = %#v", entries)
	}
	if adapted.Source != domain.ToolSourceMCP || adapted.SourceID != "cached-mcp" || adapted.ActivationPolicy != "auto" || adapted.ImplementationHash == "" || adapted.SelectionGroup == nil || adapted.SelectionGroup.ID != "mcp_group_cached_mcp" || adapted.SelectionGroup.Name != "Cached MCP" {
		t.Fatalf("MCP adapter identity = %#v, want a frozen auto MCP registration with the exact server ID", adapted)
	}
}

func TestListToolCatalogWithWorkspaceContainsCoreAndCachedEnabledExtensions(t *testing.T) {
	store := &memoryProviderStore{
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
	for _, name := range []string{"read", "bash", "edit", "write", "update_plan", "ask_user", "grep", "find", "ls"} {
		if !names[name] {
			t.Fatalf("missing %s; entries = %#v", name, entries)
		}
	}
	for _, legacy := range []string{"search_files", "glob", "list_files"} {
		if names[legacy] {
			t.Fatalf("workspace catalog still exposes removed tool name %q; entries = %#v", legacy, entries)
		}
	}
	for _, name := range []string{projectQueryToolName, projectAddToolName, projectAssociateToolName} {
		if !names[name] {
			t.Fatalf("workspace catalog missing builtin project extension tool %q; entries = %#v", name, entries)
		}
	}
	if !names[toolRegistrationMCPName] {
		t.Fatalf("workspace catalog missing builtin tool registration extension tool %q; entries = %#v", toolRegistrationMCPName, entries)
	}
	if names["aivo_tools_list_mcp"] {
		t.Fatalf("workspace catalog still exposes removed MCP source-list tool; entries = %#v", entries)
	}
	if len(names) != 17 || names["mcp_Cached_MCP_list"] || !names["mcp_cached_mcp_list"] {
		t.Fatalf("workspace catalog does not contain nine core/optional builtins, builtin Host extensions, and MCP adapter contributions; entries = %#v", entries)
	}
}
