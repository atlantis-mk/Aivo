package app

import (
	"context"
	"encoding/json"
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

func TestToolCallUsesPinnedRegistrationFromSnapshot(t *testing.T) {
	registry := NewRegistry()
	if err := registry.RegisterScoped(phase6EchoTool{spec: domain.ToolSpec{Name: "extension_echo", Description: "old", InputSchema: map[string]any{"type": "object"}, Category: "extension", Capability: "extension.read", Toolsets: []string{"extension", "coding"}}, text: "old"}, domain.ToolSourceExtension, "example", "v1"); err != nil {
		t.Fatal(err)
	}
	oldIdentity, ok := registry.IdentityFor("extension_echo")
	if !ok {
		t.Fatal("missing old identity")
	}
	if err := registry.RegisterScoped(phase6EchoTool{spec: domain.ToolSpec{Name: "extension_echo", Description: "new", InputSchema: map[string]any{"type": "object"}, Category: "extension", Capability: "extension.read", Toolsets: []string{"extension", "coding"}}, text: "new"}, domain.ToolSourceExtension, "example", "v2"); err != nil {
		t.Fatal(err)
	}
	runtime := NewToolRuntime(registry, t.TempDir())
	result := runtime.ExecuteWithContext(context.Background(), domain.ChatToolCall{ID: "call_1", Name: "extension_echo", Arguments: json.RawMessage(`{}`)}, domain.ToolExecutionContext{
		AllowedToolsets:       []string{"extension", "coding"},
		ExpectedRegistrations: map[string]domain.ToolRegistrationIdentity{"extension_echo": oldIdentity},
	})
	if !result.OK || result.Content != "old" {
		t.Fatalf("result = %#v, want old snapshot generation", result)
	}
}

func TestToolSearchAndToolCallBridge(t *testing.T) {
	registry := NewRegistry()
	if err := registry.RegisterScoped(phase6EchoTool{spec: domain.ToolSpec{Name: "extension_echo", Description: "Echo extension text", InputSchema: map[string]any{"type": "object"}, Category: "extension", Capability: "extension.read", Toolsets: []string{"extension", "coding"}}, text: "hello"}, domain.ToolSourceExtension, "example", "v1"); err != nil {
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
	search := runtime.ExecuteWithContext(context.Background(), domain.ChatToolCall{ID: "search", Name: ToolSearchName, Arguments: json.RawMessage(`{"query":"echo"}`)}, domain.ToolExecutionContext{AllowedToolsets: []string{"coding", "extension", "safe"}})
	if !search.OK || !strings.Contains(search.Content, "extension_echo") {
		t.Fatalf("search = %#v, want extension_echo", search)
	}
	call := runtime.ExecuteWithContext(context.Background(), domain.ChatToolCall{ID: "call", Name: ToolCallName, Arguments: json.RawMessage(`{"name":"extension_echo","arguments":{}}`)}, domain.ToolExecutionContext{AllowedToolsets: []string{"coding", "extension"}})
	if !call.OK || call.Content != "hello" || call.Name != "extension_echo" {
		t.Fatalf("call = %#v, want underlying extension result", call)
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
	if err := registry.RegisterScoped(phase6EchoTool{spec: domain.ToolSpec{Name: "extension_echo", Description: "Echo extension text", InputSchema: map[string]any{"type": "object"}, Category: "extension", Capability: "extension.read", Toolsets: []string{"extension", "coding"}}, text: "hello"}, domain.ToolSourceExtension, "example", "v1"); err != nil {
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

	search := runtime.ExecuteWithContext(ctx, domain.ChatToolCall{ID: "search", Name: ToolSearchName, Arguments: json.RawMessage(`{"query":"echo"}`)}, domain.ToolExecutionContext{AllowedToolsets: []string{"safe", "coding", "extension"}})
	if !search.OK || !strings.Contains(search.Content, "extension_echo") {
		t.Fatalf("search = %#v, want extension_echo", search)
	}
	if persisted := service.rememberedDeferredTools(ctx, session.ID); len(persisted) != 0 {
		t.Fatalf("persisted after search = %#v, want empty", persisted)
	}
	assembly := AssembleToolSpecsWithActivated(registry, registry.Specs(), map[string]bool{})
	names := map[string]bool{}
	for _, spec := range assembly.Specs {
		names[spec.Name] = true
	}
	if names["extension_echo"] {
		t.Fatalf("assembled specs = %#v, want extension_echo deferred after search", assembly.Specs)
	}
	if names[ResourceResolveName] || names[ToolCallName] || names[ToolSearchName] {
		t.Fatalf("assembled specs = %#v, want no legacy discovery bridge", assembly.Specs)
	}

	call := domain.ChatToolCall{ID: "call", Name: ToolCallName, Arguments: json.RawMessage(`{"name":"extension_echo","arguments":{}}`)}
	result := runtime.ExecuteWithContext(ctx, call, domain.ToolExecutionContext{AllowedToolsets: []string{"safe", "coding", "extension"}})
	if !result.OK || result.Name != "extension_echo" {
		t.Fatalf("result = %#v, want extension_echo call", result)
	}
	used := deferredToolNameUsedByCall(registry, call, result)
	if used != "extension_echo" {
		t.Fatalf("used = %q, want extension_echo", used)
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
	if names["extension_echo"] {
		t.Fatalf("assembled specs = %#v, want extension_echo to stay deferred after use", assembly.Specs)
	}
}

func TestResourceResolveActivatesDeferredToolsForNextStep(t *testing.T) {
	registry := NewRegistry()
	if err := registry.RegisterScoped(phase6EchoTool{spec: domain.ToolSpec{Name: "extension_echo", Description: "Echo extension text", InputSchema: map[string]any{"type": "object"}, Category: "extension", Capability: "extension.read", Toolsets: []string{"extension", "coding"}}, text: "hello"}, domain.ToolSourceExtension, "example", "v1"); err != nil {
		t.Fatal(err)
	}
	resolver := func(_ context.Context, request ResourceResolveRequest) (ResourceResolveDecision, error) {
		if len(request.Candidates) != 1 || request.Candidates[0].Name != "extension_echo" {
			t.Fatalf("candidates = %#v, want extension_echo only", request.Candidates)
		}
		return ResourceResolveDecision{Names: []string{"extension_echo"}, Reason: "echo capability matched"}, nil
	}
	if err := registry.RegisterScoped(NewResourceResolveTool(registry, resolver), domain.ToolSourceBridge, "tool_discovery", ""); err != nil {
		t.Fatal(err)
	}
	runtime := NewToolRuntime(registry, t.TempDir())

	result := runtime.ExecuteWithContext(context.Background(), domain.ChatToolCall{ID: "resolve", Name: ResourceResolveName, Arguments: json.RawMessage(`{"mode":"use","intent":"echo extension text"}`)}, domain.ToolExecutionContext{SessionID: "session_1", AllowedToolsets: []string{"safe", "coding", "extension"}})
	if !result.OK || !strings.Contains(result.Content, "extension_echo") {
		t.Fatalf("result = %#v, want extension_echo selected", result)
	}
}

func TestResourceResolveRequiresExplicitMode(t *testing.T) {
	mode, err := normalizeResourceResolveMode("")
	if err == nil || mode != "" || !strings.Contains(err.Error(), "mode is required") {
		t.Fatalf("mode = %q, err = %v, want missing mode rejected", mode, err)
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
		Name: "extension_echo", Description: "Echo extension text", InputSchema: map[string]any{"type": "object"},
		Category: "extension", Capability: "extension.read", Toolsets: []string{"extension", "coding"},
	}, text: "echo"}, domain.ToolSourceExtension, "example", "v1"); err != nil {
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

func TestToolDiscoveryBridgeUsesFilteredSnapshotCatalog(t *testing.T) {
	registry := NewRegistry()
	if err := registry.RegisterScoped(phase6EchoTool{spec: domain.ToolSpec{
		Name: "mcp_linear_list_issues", Description: "List Linear issues", InputSchema: map[string]any{"type": "object"},
		Namespace: "Linear", Category: "mcp", Capability: "mcp.read", Toolsets: []string{"mcp", "coding"},
	}, text: "issues"}, domain.ToolSourceMCP, "linear-server", "v1"); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterScoped(phase6EchoTool{spec: domain.ToolSpec{
		Name: "extension_echo", Description: "Echo extension text", InputSchema: map[string]any{"type": "object"},
		Category: "extension", Capability: "extension.read", Toolsets: []string{"extension", "coding"},
	}, text: "echo"}, domain.ToolSourceExtension, "example", "v1"); err != nil {
		t.Fatal(err)
	}
	var runtime *ToolRuntime
	searchTool := NewToolSearchTool(registry)
	listTool := NewToolListTool(registry)
	detailTool := NewToolDetailTool(registry)
	callTool := NewToolCallTool(registry, func() *ToolRuntime { return runtime })
	if err := registry.RegisterScoped(searchTool, domain.ToolSourceBridge, "tool_discovery", ""); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterScoped(listTool, domain.ToolSourceBridge, "tool_discovery", ""); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterScoped(detailTool, domain.ToolSourceBridge, "tool_discovery", ""); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterScoped(callTool, domain.ToolSourceBridge, "tool_discovery", ""); err != nil {
		t.Fatal(err)
	}
	runtime = NewToolRuntime(registry, t.TempDir())
	snapshot := &domain.ToolSnapshot{Tools: []domain.ToolSnapshotEntry{
		{Name: ToolSearchName},
		{Name: ToolListName},
		{Name: ToolDetailName},
		{Name: ToolCallName},
		{Name: "mcp_linear_list_issues"},
	}}
	execCtx := domain.ToolExecutionContext{AllowedToolsets: []string{"safe", "coding", "mcp", "extension"}, ToolSnapshot: snapshot}

	search := searchTool.Execute(context.Background(), json.RawMessage(`{"query":"echo","limit":10}`), execCtx)
	if !search.OK || strings.Contains(search.Content, "extension_echo") || search.Structured["availableDeferredCount"] != 1 {
		t.Fatalf("search = %#v, want only snapshot-visible deferred tools", search)
	}
	list := listTool.Execute(context.Background(), json.RawMessage(`{"includeCore":false,"limit":10}`), execCtx)
	if !list.OK || strings.Contains(list.Content, "extension_echo") || !strings.Contains(list.Content, "mcp_linear_list_issues") {
		t.Fatalf("list = %#v, want only snapshot-visible MCP tool", list)
	}
	hiddenDetail := detailTool.Execute(context.Background(), json.RawMessage(`{"name":"extension_echo"}`), execCtx)
	if hiddenDetail.OK || !strings.Contains(hiddenDetail.Error, "tool is not available") {
		t.Fatalf("hidden detail = %#v, want filtered-out tool hidden", hiddenDetail)
	}
	hiddenCall := callTool.Execute(context.Background(), json.RawMessage(`{"name":"extension_echo","arguments":{}}`), execCtx)
	if hiddenCall.OK || hiddenCall.ToolError == nil || hiddenCall.ToolError.Code != "tool_not_advertised" {
		t.Fatalf("hidden call = %#v, want snapshot rejection", hiddenCall)
	}
}
