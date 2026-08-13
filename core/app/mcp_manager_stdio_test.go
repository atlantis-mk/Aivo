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

func TestMCPProbeDiscoversPromptsAndResources(t *testing.T) {
	server := domain.MCPServerConfig{
		ID:        "test_mcp",
		Name:      "test_mcp",
		Transport: domain.MCPTransportStdio,
		Command:   os.Args[0],
		Args:      []string{"-test.run=TestMCPProbeHelperProcess", "--", "mcp-helper"},
	}
	result, err := probeMCPServer(context.Background(), server)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Tools) != 1 || result.Tools[0].Name != "echo" {
		t.Fatalf("tools = %#v, want echo tool", result.Tools)
	}
	if len(result.Prompts) != 1 || result.Prompts[0].Name != "review" || len(result.Prompts[0].Arguments) != 1 || !result.Prompts[0].Arguments[0].Required {
		t.Fatalf("prompts = %#v, want review prompt with required argument", result.Prompts)
	}
	if len(result.Resources) != 1 || result.Resources[0].URI != "file:///README.md" || result.Resources[0].Template {
		t.Fatalf("resources = %#v, want README resource", result.Resources)
	}
	if len(result.ResourceTemplates) != 1 || result.ResourceTemplates[0].URITemplate != "file:///{path}" || !result.ResourceTemplates[0].Template {
		t.Fatalf("templates = %#v, want file template", result.ResourceTemplates)
	}
}

func TestMCPManagerAllowsBlankAndBoundsFunctionalDescription(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	base := domain.MCPServerConfig{
		ID: "description_mcp", Name: "description_mcp", Transport: domain.MCPTransportStdio, Command: "npx",
	}
	saved, err := service.SaveMCPServer(ctx, domain.SaveMCPServerInput{Server: base})
	if err != nil || saved.Description != "" {
		t.Fatalf("blank description save = %#v, err = %v", saved, err)
	}
	base.Description = strings.Repeat("a", 501)
	if _, err := service.SaveMCPServer(ctx, domain.SaveMCPServerInput{Server: base}); err == nil || !strings.Contains(err.Error(), "at most 500 bytes") {
		t.Fatalf("oversized description error = %v", err)
	}
	base.Description = "  Query and update Linear issues  "
	saved, err = service.SaveMCPServer(ctx, domain.SaveMCPServerInput{Server: base})
	if err != nil {
		t.Fatal(err)
	}
	if saved.Description != "Query and update Linear issues" {
		t.Fatalf("saved description = %q", saved.Description)
	}
}

func TestMCPProbePersistsCapabilitiesForList(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	root := t.TempDir()
	server := domain.MCPServerConfig{
		ID:          "persist_mcp",
		Name:        "persist_mcp",
		Description: "Persist discovered MCP capabilities",
		Transport:   domain.MCPTransportStdio,
		Command:     os.Args[0],
		Args:        []string{"-test.run=TestMCPProbeHelperProcess", "--", "mcp-helper"},
		Roots:       []string{root},
		AuthType:    domain.MCPAuthOAuth, BearerTokenEnv: "AIVO_MCP_TOKEN",
		OAuthIssuerURL: "https://auth.example.test", OAuthClientID: "aivo", OAuthScopes: []string{"mcp"},
	}
	if _, err := service.SaveMCPServer(ctx, domain.SaveMCPServerInput{Server: server}); err != nil {
		t.Fatal(err)
	}
	probe, err := service.ProbeMCPServer(ctx, domain.MCPProbeInput{ServerID: server.ID})
	if err != nil || !probe.OK {
		t.Fatalf("probe = %#v err = %v, want ok", probe, err)
	}
	items, err := service.ListMCPServers(ctx, domain.MCPServerListInput{IncludeDisabled: true, IncludeTools: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %#v, want one server", items)
	}
	item := items[0]
	if len(item.Tools) != 1 || len(item.Prompts) != 1 || len(item.Resources) != 1 || len(item.ResourceTemplates) != 1 {
		t.Fatalf("item = %#v, want persisted tools/prompts/resources/templates", item)
	}
	if len(item.Server.Roots) != 1 || item.Server.Roots[0] != root {
		t.Fatalf("roots = %#v, want persisted root", item.Server.Roots)
	}
	if item.Server.AuthType != domain.MCPAuthOAuth || item.Server.BearerTokenEnv != "AIVO_MCP_TOKEN" || item.Server.OAuthIssuerURL != "https://auth.example.test" || len(item.Server.OAuthScopes) != 1 || item.Server.OAuthScopes[0] != "mcp" {
		t.Fatalf("server auth = %#v, want persisted oauth metadata", item.Server)
	}
}

func TestMCPPromptGetAndResourceRead(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	server := domain.MCPServerConfig{
		ID:          "call_mcp",
		Name:        "call_mcp",
		Description: "Read prompts and resources from the test MCP",
		Transport:   domain.MCPTransportStdio,
		Command:     os.Args[0],
		Args:        []string{"-test.run=TestMCPProbeHelperProcess", "--", "mcp-helper"},
	}
	if _, err := service.SaveMCPServer(ctx, domain.SaveMCPServerInput{Server: server}); err != nil {
		t.Fatal(err)
	}
	prompt, err := service.GetMCPPrompt(ctx, domain.MCPPromptGetInput{ServerID: server.ID, Name: "review", Arguments: map[string]string{"path": "README.md"}})
	if err != nil {
		t.Fatal(err)
	}
	if prompt.Content != "user: Review README.md" || len(prompt.Messages) != 1 || prompt.Messages[0].Role != "user" {
		t.Fatalf("prompt = %#v, want normalized prompt content", prompt)
	}
	resource, err := service.ReadMCPResource(ctx, domain.MCPResourceReadInput{ServerID: server.ID, URI: "file:///README.md"})
	if err != nil {
		t.Fatal(err)
	}
	if resource.Content != "# Aivo\n" || len(resource.Contents) != 1 || resource.Contents[0].MimeType != "text/markdown" {
		t.Fatalf("resource = %#v, want normalized resource content", resource)
	}
}

func TestMCPManagerReusesStdioConnectionForToolCalls(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	server := domain.MCPServerConfig{
		ID:             "long_lived_mcp",
		Name:           "long_lived_mcp",
		Description:    "Reuse the long-lived MCP test connection",
		Transport:      domain.MCPTransportStdio,
		Command:        os.Args[0],
		Args:           []string{"-test.run=TestMCPLongLivedHelperProcess", "--", "mcp-long-lived-helper"},
		Enabled:        true,
		TimeoutSeconds: 5,
	}
	if _, err := service.SaveMCPServer(ctx, domain.SaveMCPServerInput{Server: server}); err != nil {
		t.Fatal(err)
	}
	defer service.mcpManager.connections.Close()

	saved, err := service.mcpManager.store.GetMCPServer(ctx, server.ID)
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.mcpManager.callMCPTool(ctx, saved, "counter", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.mcpManager.callMCPTool(ctx, saved, "counter", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if textFromMCPToolContent(first) != "1" || textFromMCPToolContent(second) != "2" {
		t.Fatalf("tool results = %#v then %#v, want counter to increment on one long-lived connection", first, second)
	}
}

func TestMCPManagerReconnectsAfterFailedToolCall(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	marker := filepath.Join(t.TempDir(), "failed-once")
	server := domain.MCPServerConfig{
		ID:          "reconnect_mcp",
		Name:        "reconnect_mcp",
		Description: "Reconnect the MCP test server after failure",
		Transport:   domain.MCPTransportStdio,
		Command:     os.Args[0],
		Args:        []string{"-test.run=TestMCPReconnectHelperProcess", "--", "mcp-reconnect-helper"},
		Env:         map[string]string{"AIVO_MCP_RECONNECT_FILE": marker},
		Enabled:     true,
	}
	if _, err := service.SaveMCPServer(ctx, domain.SaveMCPServerInput{Server: server}); err != nil {
		t.Fatal(err)
	}
	defer service.mcpManager.connections.Close()
	saved, err := service.mcpManager.store.GetMCPServer(ctx, server.ID)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.mcpManager.callMCPTool(ctx, saved, "recover", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if textFromMCPToolContent(result) != "recovered" {
		t.Fatalf("tool result = %#v, want retry on a fresh MCP connection", result)
	}
}

func TestSanitizeMCPErrorRedactsCredentials(t *testing.T) {
	message := sanitizeMCPError(`failed with Bearer secret-token and token=abc123 password="hunter2"`)
	for _, leaked := range []string{"secret-token", "abc123", "hunter2"} {
		if strings.Contains(message, leaked) {
			t.Fatalf("sanitized message %q still contains %q", message, leaked)
		}
	}
	if count := strings.Count(message, "[redacted]"); count < 3 {
		t.Fatalf("sanitized message = %q, want redactions", message)
	}
}

func TestMCPRegisterEnabledToolsIncludesResourceUtilities(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	server := domain.MCPServerConfig{
		ID:          "resource_mcp",
		Name:        "resource_mcp",
		Description: "Expose resource utilities from the test MCP",
		Transport:   domain.MCPTransportStdio,
		Command:     os.Args[0],
		Args:        []string{"-test.run=TestMCPProbeHelperProcess", "--", "mcp-helper"},
		Enabled:     true,
	}
	if _, err := service.SaveMCPServer(ctx, domain.SaveMCPServerInput{Server: server}); err != nil {
		t.Fatal(err)
	}
	defer service.mcpManager.connections.Close()

	registry := NewRegistry()
	service.mcpManager.RegisterEnabledTools(ctx, registry)
	for _, name := range []string{
		"mcp_host_resource_mcp_list_resources",
		"mcp_host_resource_mcp_list_resource_templates",
		"mcp_host_resource_mcp_read_resource",
	} {
		if _, ok := registry.Get(name); !ok {
			t.Fatalf("missing registered MCP resource utility %s; catalog = %#v", name, registry.CatalogEntries())
		}
	}
	runtime := NewToolRuntime(registry, t.TempDir())
	result := runtime.ExecuteWithContext(ctx, domain.ChatToolCall{
		ID: "read", Name: "mcp_host_resource_mcp_read_resource", Arguments: json.RawMessage(`{"uri":"file:///README.md"}`),
	}, domain.ToolExecutionContext{AllowedToolsets: []string{"mcp", "coding"}})
	if !result.OK || !strings.Contains(result.Content, "# Aivo") {
		t.Fatalf("read resource result = %#v, want README content", result)
	}
}

func TestMCPManagerProbePersistsToolsListChangedRefresh(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	server := domain.MCPServerConfig{
		ID:          "changed_manager_mcp",
		Name:        "changed_manager_mcp",
		Description: "Refresh changed tools from the test MCP",
		Transport:   domain.MCPTransportStdio,
		Command:     os.Args[0],
		Args:        []string{"-test.run=TestMCPToolsChangedHelperProcess", "--", "mcp-tools-changed-helper"},
		Enabled:     true,
	}
	if _, err := service.SaveMCPServer(ctx, domain.SaveMCPServerInput{Server: server}); err != nil {
		t.Fatal(err)
	}
	defer service.mcpManager.connections.Close()
	tools, err := service.mcpManager.store.ListMCPTools(ctx, server.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 || tools[0].Name != "after" {
		t.Fatalf("tools = %#v, want refreshed tool after list_changed notification", tools)
	}
}

func TestMCPStdioRespondsToRootsListRequest(t *testing.T) {
	root := t.TempDir()
	result, err := probeMCPServer(context.Background(), domain.MCPServerConfig{
		ID:        "roots_mcp",
		Name:      "roots_mcp",
		Transport: domain.MCPTransportStdio,
		Command:   os.Args[0],
		Args:      []string{"-test.run=TestMCPRootsHelperProcess", "--", "mcp-roots-helper"},
		Roots:     []string{root},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Tools) != 1 || result.Tools[0].Name != "echo" {
		t.Fatalf("result = %#v, want helper capabilities after roots/list", result)
	}
}

func TestMCPStdioRefreshesToolsAfterListChangedNotification(t *testing.T) {
	result, err := probeMCPServer(context.Background(), domain.MCPServerConfig{
		ID:        "changed_mcp",
		Name:      "changed_mcp",
		Transport: domain.MCPTransportStdio,
		Command:   os.Args[0],
		Args:      []string{"-test.run=TestMCPToolsChangedHelperProcess", "--", "mcp-tools-changed-helper"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Tools) != 1 || result.Tools[0].Name != "after" {
		t.Fatalf("tools = %#v, want refreshed after tool", result.Tools)
	}
}

func TestReadMCPServerLogReturnsBoundedTail(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	server := domain.MCPServerConfig{ID: "log/server", Name: "log/server", Description: "Read bounded MCP server logs", Transport: domain.MCPTransportStdio}
	if _, err := service.SaveMCPServer(ctx, domain.SaveMCPServerInput{Server: server}); err != nil {
		t.Fatal(err)
	}
	path, err := mcpLogPath(server.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(path, filepath.Join(home, ".aivo", "logs")) || strings.Contains(filepath.Base(path), "/") {
		t.Fatalf("path = %q, want sanitized path under test home", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("0123456789"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := service.ReadMCPServerLog(ctx, domain.MCPServerLogInput{ServerID: server.ID, Limit: 4, Tail: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "6789" || result.Offset != 6 || result.NextOffset != 10 || result.Size != 10 || result.Truncated {
		t.Fatalf("result = %#v, want tail chunk", result)
	}
}
