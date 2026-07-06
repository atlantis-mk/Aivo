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

type phase6EchoTool struct {
	spec domain.ToolSpec
	text string
}

func (t phase6EchoTool) Spec() domain.ToolSpec { return t.spec }
func (t phase6EchoTool) Execute(context.Context, json.RawMessage, domain.ToolExecutionContext) domain.ToolResult {
	return domain.ToolResult{Name: t.spec.Name, OK: true, Content: t.text}
}

func TestLoadPluginManifestRejectsEscapingCWD(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "aivo.plugin.json"), []byte(`{
		"id":"bad",
		"name":"bad",
		"entrypoint":{"command":"node","cwd":"../outside"}
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, _, err := LoadPluginManifest(root)
	if err == nil || !strings.Contains(err.Error(), "escapes plugin root") {
		t.Fatalf("LoadPluginManifest err = %v, want escape rejection", err)
	}
}

func TestToolCallRejectsStaleRegistration(t *testing.T) {
	registry := NewRegistry()
	if err := registry.RegisterScoped(phase6EchoTool{spec: domain.ToolSpec{Name: "plugin_echo", Description: "old", InputSchema: map[string]any{"type": "object"}, Category: "plugin", Capability: "plugin.read", Toolsets: []string{"plugin", "coding"}}, text: "old"}, domain.ToolSourcePlugin, "p1", "v1"); err != nil {
		t.Fatal(err)
	}
	oldIdentity, ok := registry.IdentityFor("plugin_echo")
	if !ok {
		t.Fatal("missing old identity")
	}
	if err := registry.RegisterScoped(phase6EchoTool{spec: domain.ToolSpec{Name: "plugin_echo", Description: "new", InputSchema: map[string]any{"type": "object"}, Category: "plugin", Capability: "plugin.read", Toolsets: []string{"plugin", "coding"}}, text: "new"}, domain.ToolSourcePlugin, "p1", "v2"); err != nil {
		t.Fatal(err)
	}
	runtime := NewToolRuntime(registry, t.TempDir())
	result := runtime.ExecuteWithContext(context.Background(), domain.ChatToolCall{ID: "call_1", Name: "plugin_echo", Arguments: json.RawMessage(`{}`)}, domain.ToolExecutionContext{
		AllowedToolsets:       []string{"plugin", "coding"},
		ExpectedRegistrations: map[string]domain.ToolRegistrationIdentity{"plugin_echo": oldIdentity},
	})
	if result.OK || result.ToolError == nil || result.ToolError.Code != "stale_tool_registration" {
		t.Fatalf("result = %#v, want stale registration failure", result)
	}
}

func TestToolSearchAndToolCallBridge(t *testing.T) {
	registry := NewRegistry()
	if err := registry.RegisterScoped(phase6EchoTool{spec: domain.ToolSpec{Name: "plugin_echo", Description: "Echo plugin text", InputSchema: map[string]any{"type": "object"}, Category: "plugin", Capability: "plugin.read", Toolsets: []string{"plugin", "coding"}}, text: "hello"}, domain.ToolSourcePlugin, "p1", "v1"); err != nil {
		t.Fatal(err)
	}
	var runtime *ToolRuntime
	if err := registry.RegisterScoped(NewToolSearchTool(registry), domain.ToolSourceBridge, "tool_discovery", ""); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterScoped(NewToolCallTool(registry, func() *ToolRuntime { return runtime }), domain.ToolSourceBridge, "tool_discovery", ""); err != nil {
		t.Fatal(err)
	}
	runtime = NewToolRuntime(registry, t.TempDir())
	search := runtime.ExecuteWithContext(context.Background(), domain.ChatToolCall{ID: "search", Name: ToolSearchName, Arguments: json.RawMessage(`{"query":"echo"}`)}, domain.ToolExecutionContext{AllowedToolsets: []string{"coding", "plugin", "safe"}})
	if !search.OK || !strings.Contains(search.Content, "plugin_echo") {
		t.Fatalf("search = %#v, want plugin_echo", search)
	}
	call := runtime.ExecuteWithContext(context.Background(), domain.ChatToolCall{ID: "call", Name: ToolCallName, Arguments: json.RawMessage(`{"name":"plugin_echo","arguments":{}}`)}, domain.ToolExecutionContext{AllowedToolsets: []string{"coding", "plugin"}})
	if !call.OK || call.Content != "hello" || call.Name != "plugin_echo" {
		t.Fatalf("call = %#v, want underlying plugin result", call)
	}
}

func TestDeferredToolSearchAndCallDoNotPersistDirectInjection(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	registry := NewRegistry()
	if err := registry.RegisterScoped(phase6EchoTool{spec: domain.ToolSpec{Name: "plugin_echo", Description: "Echo plugin text", InputSchema: map[string]any{"type": "object"}, Category: "plugin", Capability: "plugin.read", Toolsets: []string{"plugin", "coding"}}, text: "hello"}, domain.ToolSourcePlugin, "p1", "v1"); err != nil {
		t.Fatal(err)
	}
	var runtime *ToolRuntime
	if err := registry.RegisterScoped(NewToolSearchTool(registry), domain.ToolSourceBridge, "tool_discovery", ""); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterScoped(NewToolCallTool(registry, func() *ToolRuntime { return runtime }), domain.ToolSourceBridge, "tool_discovery", ""); err != nil {
		t.Fatal(err)
	}
	runtime = NewToolRuntime(registry, t.TempDir())

	search := runtime.ExecuteWithContext(ctx, domain.ChatToolCall{ID: "search", Name: ToolSearchName, Arguments: json.RawMessage(`{"query":"echo"}`)}, domain.ToolExecutionContext{AllowedToolsets: []string{"safe", "coding", "plugin"}})
	if !search.OK || !strings.Contains(search.Content, "plugin_echo") {
		t.Fatalf("search = %#v, want plugin_echo", search)
	}
	if persisted := service.rememberedDeferredTools(ctx, session.ID); len(persisted) != 0 {
		t.Fatalf("persisted after search = %#v, want empty", persisted)
	}
	assembly := AssembleToolSpecsWithActivated(registry, registry.Specs(), map[string]bool{})
	names := map[string]bool{}
	for _, spec := range assembly.Specs {
		names[spec.Name] = true
	}
	if names["plugin_echo"] {
		t.Fatalf("assembled specs = %#v, want plugin_echo deferred after search", assembly.Specs)
	}
	if !names[ToolResolveName] || names[ToolCallName] {
		t.Fatalf("assembled specs = %#v, want only tool_resolve bridge", assembly.Specs)
	}

	call := domain.ChatToolCall{ID: "call", Name: ToolCallName, Arguments: json.RawMessage(`{"name":"plugin_echo","arguments":{}}`)}
	result := runtime.ExecuteWithContext(ctx, call, domain.ToolExecutionContext{AllowedToolsets: []string{"safe", "coding", "plugin"}})
	if !result.OK || result.Name != "plugin_echo" {
		t.Fatalf("result = %#v, want plugin_echo call", result)
	}
	used := deferredToolNameUsedByCall(registry, call, result)
	if used != "plugin_echo" {
		t.Fatalf("used = %q, want plugin_echo", used)
	}
	if _, err := service.store.UpsertSessionExecutionState(ctx, domain.SessionExecutionState{SessionID: session.ID, Status: domain.ExecutionStatusIdle, Reason: "turn complete"}); err != nil {
		t.Fatal(err)
	}
	recovered := NewService(service.store)
	remembered := recovered.rememberedDeferredTools(ctx, session.ID)
	if len(remembered) != 0 {
		t.Fatalf("remembered = %#v, want no deferred direct injection", remembered)
	}
	assembly = AssembleToolSpecsWithActivated(registry, registry.Specs(), remembered)
	names = map[string]bool{}
	for _, spec := range assembly.Specs {
		names[spec.Name] = true
	}
	if names["plugin_echo"] {
		t.Fatalf("assembled specs = %#v, want plugin_echo to stay deferred after use", assembly.Specs)
	}
}

func TestToolResolveActivatesDeferredToolsForNextStep(t *testing.T) {
	registry := NewRegistry()
	if err := registry.RegisterScoped(phase6EchoTool{spec: domain.ToolSpec{Name: "plugin_echo", Description: "Echo plugin text", InputSchema: map[string]any{"type": "object"}, Category: "plugin", Capability: "plugin.read", Toolsets: []string{"plugin", "coding"}}, text: "hello"}, domain.ToolSourcePlugin, "p1", "v1"); err != nil {
		t.Fatal(err)
	}
	activated := map[string]bool{}
	resolver := func(_ context.Context, request ToolResolveRequest) (ToolResolveDecision, error) {
		if len(request.Candidates) != 1 || request.Candidates[0].Name != "plugin_echo" {
			t.Fatalf("candidates = %#v, want plugin_echo only", request.Candidates)
		}
		return ToolResolveDecision{Names: []string{"plugin_echo"}, Reason: "echo capability matched"}, nil
	}
	activate := func(_ context.Context, sessionID string, toolName string) error {
		if sessionID != "session_1" {
			t.Fatalf("sessionID = %q, want session_1", sessionID)
		}
		activated[toolName] = true
		return nil
	}
	if err := registry.RegisterScoped(NewToolResolveTool(registry, resolver, activate), domain.ToolSourceBridge, "tool_discovery", ""); err != nil {
		t.Fatal(err)
	}
	runtime := NewToolRuntime(registry, t.TempDir())

	result := runtime.ExecuteWithContext(context.Background(), domain.ChatToolCall{ID: "resolve", Name: ToolResolveName, Arguments: json.RawMessage(`{"intent":"echo plugin text"}`)}, domain.ToolExecutionContext{SessionID: "session_1", AllowedToolsets: []string{"safe", "coding", "plugin"}})
	if !result.OK || !activated["plugin_echo"] || !strings.Contains(result.Content, "plugin_echo") {
		t.Fatalf("result = %#v activated = %#v, want plugin_echo activated", result, activated)
	}
	assembly := AssembleToolSpecsWithActivated(registry, registry.Specs(), activated)
	names := map[string]int{}
	for _, spec := range assembly.Specs {
		names[spec.Name]++
	}
	if names["plugin_echo"] != 1 {
		t.Fatalf("plugin_echo visible count = %d; specs = %#v", names["plugin_echo"], assembly.Specs)
	}
}

func TestToolSearchIncludesNamespaceAndSourceDiagnostics(t *testing.T) {
	registry := NewRegistry()
	if err := registry.RegisterScoped(phase6EchoTool{spec: domain.ToolSpec{
		Name: "mcp_srv_list_issues", Description: "List issues", InputSchema: map[string]any{"type": "object"},
		Namespace: "Linear", NamespaceDescription: "Issue tracker", Category: "mcp", Capability: "mcp.read", Toolsets: []string{"mcp", "coding"},
	}, text: "issues"}, domain.ToolSourceMCP, "linear-server", "v1"); err != nil {
		t.Fatal(err)
	}
	var runtime *ToolRuntime
	if err := registry.RegisterScoped(NewToolSearchTool(registry), domain.ToolSourceBridge, "tool_discovery", ""); err != nil {
		t.Fatal(err)
	}
	runtime = NewToolRuntime(registry, t.TempDir())

	search := runtime.ExecuteWithContext(context.Background(), domain.ChatToolCall{ID: "search", Name: ToolSearchName, Arguments: json.RawMessage(`{"query":"linear"}`)}, domain.ToolExecutionContext{AllowedToolsets: []string{"safe", "coding", "mcp"}})
	if !search.OK || !strings.Contains(search.Content, "mcp_srv_list_issues") || !strings.Contains(search.Content, "List issues") || strings.Contains(search.Content, "inputSchema") {
		t.Fatalf("search = %#v, want list-style namespace match without schema", search)
	}
	matches, _ := search.Structured["matches"].([]map[string]any)
	if len(matches) != 1 || matches[0]["namespace"] != "Linear" || matches[0]["source"] != domain.ToolSourceMCP {
		t.Fatalf("matches = %#v, want list metadata", search.Structured["matches"])
	}
	missing := runtime.ExecuteWithContext(context.Background(), domain.ChatToolCall{ID: "search2", Name: ToolSearchName, Arguments: json.RawMessage(`{"query":"does-not-match"}`)}, domain.ToolExecutionContext{AllowedToolsets: []string{"safe", "coding", "mcp"}})
	sourceCounts, _ := missing.Structured["sourceCounts"].(map[string]int)
	if !missing.OK || missing.Structured["availableDeferredCount"] != 1 || sourceCounts[domain.ToolSourceMCP] != 1 {
		t.Fatalf("missing = %#v, want MCP source diagnostics", missing)
	}
}

func TestToolListAndDetailBridge(t *testing.T) {
	registry := NewRegistry()
	if err := registry.RegisterScoped(phase6EchoTool{spec: domain.ToolSpec{
		Name: "mcp_linear_list_issues", Description: "List Linear issues", InputSchema: map[string]any{"type": "object", "properties": map[string]any{"team": map[string]any{"type": "string"}}},
		Namespace: "Linear", Category: "mcp", Capability: "mcp.read", Toolsets: []string{"mcp", "coding"},
	}, text: "issues"}, domain.ToolSourceMCP, "linear-server", "v1"); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterScoped(phase6EchoTool{spec: domain.ToolSpec{
		Name: "plugin_echo", Description: "Echo plugin text", InputSchema: map[string]any{"type": "object"},
		Category: "plugin", Capability: "plugin.read", Toolsets: []string{"plugin", "coding"},
	}, text: "echo"}, domain.ToolSourcePlugin, "p1", "v1"); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterScoped(NewToolListTool(registry), domain.ToolSourceBridge, "tool_discovery", ""); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterScoped(NewToolDetailTool(registry), domain.ToolSourceBridge, "tool_discovery", ""); err != nil {
		t.Fatal(err)
	}
	runtime := NewToolRuntime(registry, t.TempDir())

	list := runtime.ExecuteWithContext(context.Background(), domain.ChatToolCall{ID: "list", Name: ToolListName, Arguments: json.RawMessage(`{"source":"mcp","limit":100}`)}, domain.ToolExecutionContext{AllowedToolsets: []string{"safe", "coding", "mcp"}})
	if !list.OK || !strings.Contains(list.Content, "mcp_linear_list_issues") || strings.Contains(list.Content, "properties") {
		t.Fatalf("list = %#v, want MCP names/descriptions without schemas", list)
	}
	if list.Structured["total"] != 1 {
		t.Fatalf("list total = %#v, want 1", list.Structured["total"])
	}
	detail := runtime.ExecuteWithContext(context.Background(), domain.ChatToolCall{ID: "detail", Name: ToolDetailName, Arguments: json.RawMessage(`{"name":"mcp_linear_list_issues"}`)}, domain.ToolExecutionContext{AllowedToolsets: []string{"safe", "coding", "mcp"}})
	if !detail.OK || !strings.Contains(detail.Content, "properties") || !strings.Contains(detail.Content, "team") {
		t.Fatalf("detail = %#v, want full schema", detail)
	}
}

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

func TestBrowserToolsStayDeferredAfterSearch(t *testing.T) {
	registry := NewRegistry()
	for _, tool := range NewBrowserTools() {
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
	runtime = NewToolRuntime(registry, t.TempDir())

	assembly := AssembleToolSpecs(registry, registry.Specs())
	names := map[string]int{}
	for _, spec := range assembly.Specs {
		names[spec.Name]++
	}
	if names["browser_state"] != 0 || names["browser_click"] != 0 {
		t.Fatalf("browser tools were directly visible before search: %#v", assembly.Specs)
	}
	if names[ToolResolveName] != 1 || names[ToolSearchName] != 0 || names[ToolListName] != 0 || names[ToolDetailName] != 0 || names[ToolCallName] != 0 {
		t.Fatalf("bridge tools missing for deferred browser tools: %#v", assembly.Specs)
	}

	activated := map[string]bool{}
	search := runtime.ExecuteWithContext(context.Background(), domain.ChatToolCall{ID: "search", Name: ToolSearchName, Arguments: json.RawMessage(`{"query":"browser state","limit":1}`)}, domain.ToolExecutionContext{AllowedToolsets: []string{"safe", "browser"}})
	if !search.OK || !strings.Contains(search.Content, "browser_") {
		t.Fatalf("search = %#v, want a browser tool match", search)
	}

	assembly = AssembleToolSpecsWithActivated(registry, registry.Specs(), activated)
	names = map[string]int{}
	for _, spec := range assembly.Specs {
		names[spec.Name]++
	}
	for _, name := range []string{"browser_state", "browser_navigate", "browser_snapshot", "browser_click", "browser_fill", "browser_press_key", "browser_evaluate", "browser_screenshot", "browser_console_messages", "browser_network_requests"} {
		if names[name] != 0 {
			t.Fatalf("browser tool %s became directly visible after search; specs = %#v", name, assembly.Specs)
		}
	}
}

func TestBrowserToolUseRemembersGroup(t *testing.T) {
	registry := NewRegistry()
	for _, tool := range NewBrowserTools() {
		if err := registry.Register(tool); err != nil {
			t.Fatal(err)
		}
	}
	call := domain.ChatToolCall{ID: "call", Name: "browser_state", Arguments: json.RawMessage(`{}`)}
	result := domain.ToolResult{Name: "browser_state", OK: true}
	used := deferredToolNameUsedByCall(registry, call, result)
	if used != deferredBrowserToolGroupKey {
		t.Fatalf("used = %q, want browser group", used)
	}
	assembly := AssembleToolSpecsWithActivated(registry, registry.Specs(), map[string]bool{used: true})
	names := map[string]int{}
	for _, spec := range assembly.Specs {
		names[spec.Name]++
	}
	if names["browser_state"] != 1 || names["browser_click"] != 1 || names["browser_network_requests"] != 1 {
		t.Fatalf("remembered browser group did not inject all browser tools: %#v", assembly.Specs)
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
	for _, name := range []string{"web_fetch", "web_search", "browser_state", "browser_navigate", "browser_snapshot", "agent_mode_list", ToolResolveName} {
		if !names[name] {
			t.Fatalf("global tool catalog missing %s; entries = %#v", name, entries)
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
