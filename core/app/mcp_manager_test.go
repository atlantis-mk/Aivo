package app

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

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

func TestMCPProbePersistsCapabilitiesForList(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	root := t.TempDir()
	server := domain.MCPServerConfig{
		ID:        "persist_mcp",
		Name:      "persist_mcp",
		Transport: domain.MCPTransportStdio,
		Command:   os.Args[0],
		Args:      []string{"-test.run=TestMCPProbeHelperProcess", "--", "mcp-helper"},
		Roots:     []string{root},
		AuthType:  domain.MCPAuthOAuth, BearerTokenEnv: "AIVO_MCP_TOKEN",
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
		ID:        "call_mcp",
		Name:      "call_mcp",
		Transport: domain.MCPTransportStdio,
		Command:   os.Args[0],
		Args:      []string{"-test.run=TestMCPProbeHelperProcess", "--", "mcp-helper"},
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
		ID:        "reconnect_mcp",
		Name:      "reconnect_mcp",
		Transport: domain.MCPTransportStdio,
		Command:   os.Args[0],
		Args:      []string{"-test.run=TestMCPReconnectHelperProcess", "--", "mcp-reconnect-helper"},
		Env:       map[string]string{"AIVO_MCP_RECONNECT_FILE": marker},
		Enabled:   true,
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
		ID:        "resource_mcp",
		Name:      "resource_mcp",
		Transport: domain.MCPTransportStdio,
		Command:   os.Args[0],
		Args:      []string{"-test.run=TestMCPProbeHelperProcess", "--", "mcp-helper"},
		Enabled:   true,
	}
	if _, err := service.SaveMCPServer(ctx, domain.SaveMCPServerInput{Server: server}); err != nil {
		t.Fatal(err)
	}
	defer service.mcpManager.connections.Close()

	registry := NewRegistry()
	service.mcpManager.RegisterEnabledTools(ctx, registry)
	for _, name := range []string{
		"mcp_resource_mcp_list_resources",
		"mcp_resource_mcp_list_resource_templates",
		"mcp_resource_mcp_read_resource",
	} {
		if _, ok := registry.Get(name); !ok {
			t.Fatalf("missing registered MCP resource utility %s; catalog = %#v", name, registry.CatalogEntries())
		}
	}
	runtime := NewToolRuntime(registry, t.TempDir())
	result := runtime.ExecuteWithContext(ctx, domain.ChatToolCall{
		ID: "read", Name: "mcp_resource_mcp_read_resource", Arguments: json.RawMessage(`{"uri":"file:///README.md"}`),
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
		ID:        "changed_manager_mcp",
		Name:      "changed_manager_mcp",
		Transport: domain.MCPTransportStdio,
		Command:   os.Args[0],
		Args:      []string{"-test.run=TestMCPToolsChangedHelperProcess", "--", "mcp-tools-changed-helper"},
		Enabled:   true,
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

func TestMCPStreamableHTTPProbeAndPrompt(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	t.Setenv("AIVO_TEST_MCP_TOKEN", "test-token")
	requests := 0
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("authorization = %q, want bearer token from env", got)
		}
		if r.Header.Get("MCP-Protocol-Version") == "" {
			t.Fatalf("missing MCP-Protocol-Version header")
		}
		var request struct {
			ID     string `json:"id"`
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": mcpHelperResult(request.Method)})
	}))
	defer httpServer.Close()

	server := domain.MCPServerConfig{
		ID:        "http_mcp",
		Name:      "http_mcp",
		Transport: domain.MCPTransportStreamableHTTP,
		URL:       httpServer.URL,
		AuthType:  domain.MCPAuthBearer, BearerTokenEnv: "AIVO_TEST_MCP_TOKEN",
	}
	if _, err := service.SaveMCPServer(ctx, domain.SaveMCPServerInput{Server: server}); err != nil {
		t.Fatal(err)
	}
	probe, err := service.ProbeMCPServer(ctx, domain.MCPProbeInput{ServerID: server.ID})
	if err != nil || !probe.OK {
		t.Fatalf("probe = %#v err = %v, want ok", probe, err)
	}
	if len(probe.Tools) != 1 || probe.Tools[0].Name != "echo" || len(probe.ResourceTemplates) != 1 {
		t.Fatalf("probe = %#v, want helper capabilities", probe)
	}
	prompt, err := service.GetMCPPrompt(ctx, domain.MCPPromptGetInput{ServerID: server.ID, Name: "review", Arguments: map[string]string{"path": "README.md"}})
	if err != nil {
		t.Fatal(err)
	}
	if prompt.Content != "user: Review README.md" {
		t.Fatalf("prompt = %#v, want normalized prompt content", prompt)
	}
	if requests < 6 {
		t.Fatalf("requests = %d, want probe and prompt calls", requests)
	}
}

func TestMCPHTTPUnauthorizedIncludesOAuthResourceMetadata(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="https://mcp.example.test/.well-known/oauth-protected-resource"`)
		http.Error(w, "missing token", http.StatusUnauthorized)
	}))
	defer httpServer.Close()

	_, err := probeMCPServer(context.Background(), domain.MCPServerConfig{
		ID:        "oauth_mcp",
		Name:      "oauth_mcp",
		Transport: domain.MCPTransportStreamableHTTP,
		URL:       httpServer.URL,
		AuthType:  domain.MCPAuthOAuth, BearerTokenEnv: "AIVO_EMPTY_MCP_TOKEN",
	})
	if err == nil || !strings.Contains(err.Error(), "AIVO_EMPTY_MCP_TOKEN") {
		t.Fatalf("err = %v, want missing token env error before request", err)
	}

	t.Setenv("AIVO_EMPTY_MCP_TOKEN", "expired")
	_, err = probeMCPServer(context.Background(), domain.MCPServerConfig{
		ID:        "oauth_mcp",
		Name:      "oauth_mcp",
		Transport: domain.MCPTransportStreamableHTTP,
		URL:       httpServer.URL,
		AuthType:  domain.MCPAuthOAuth, BearerTokenEnv: "AIVO_EMPTY_MCP_TOKEN",
	})
	if err == nil || !strings.Contains(err.Error(), "oauth resource metadata: https://mcp.example.test/.well-known/oauth-protected-resource") {
		t.Fatalf("err = %v, want oauth resource metadata", err)
	}
}

func TestMCPOAuthDiscoveryFindsResourceAndAuthorizationMetadata(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	var httpServer *httptest.Server
	httpServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("MCP-Protocol-Version") == "" {
			t.Fatalf("missing MCP-Protocol-Version header")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/oauth-protected-resource/mcp":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"resource":              httpServer.URL + "/mcp",
				"authorization_servers": []string{httpServer.URL},
				"scopes_supported":      []string{"files:read"},
			})
		case "/.well-known/oauth-authorization-server":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issuer":                                         httpServer.URL,
				"authorization_endpoint":                         httpServer.URL + "/authorize",
				"token_endpoint":                                 httpServer.URL + "/token",
				"registration_endpoint":                          httpServer.URL + "/register",
				"code_challenge_methods_supported":               []string{"S256"},
				"response_types_supported":                       []string{"code"},
				"grant_types_supported":                          []string{"authorization_code", "refresh_token"},
				"authorization_response_iss_parameter_supported": true,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer httpServer.Close()

	server := domain.MCPServerConfig{
		ID:        "oauth_discovery_mcp",
		Name:      "oauth_discovery_mcp",
		Transport: domain.MCPTransportStreamableHTTP,
		URL:       httpServer.URL + "/mcp",
		AuthType:  domain.MCPAuthOAuth,
	}
	if _, err := service.SaveMCPServer(ctx, domain.SaveMCPServerInput{Server: server}); err != nil {
		t.Fatal(err)
	}
	result, err := service.DiscoverMCPOAuth(ctx, domain.MCPOAuthDiscoveryInput{ServerID: server.ID})
	if err != nil {
		t.Fatal(err)
	}
	if result.ResourceMetadataURL != httpServer.URL+"/.well-known/oauth-protected-resource/mcp" || result.Resource != httpServer.URL+"/mcp" {
		t.Fatalf("resource metadata = %#v, want path-specific protected resource metadata", result)
	}
	if result.SelectedIssuer != httpServer.URL || result.AuthorizationEndpoint != httpServer.URL+"/authorize" || result.TokenEndpoint != httpServer.URL+"/token" {
		t.Fatalf("authorization metadata = %#v, want discovered OAuth endpoints", result)
	}
	if !result.RequiresDynamicClientReg || result.RegistrationEndpoint != httpServer.URL+"/register" {
		t.Fatalf("registration = %#v, want dynamic client registration available", result)
	}
	if len(result.ScopesSupported) != 1 || result.ScopesSupported[0] != "files:read" || len(result.CodeChallengeMethods) != 1 || result.CodeChallengeMethods[0] != "S256" {
		t.Fatalf("capabilities = %#v, want scopes and PKCE support", result)
	}
}

func TestMCPOAuthBrowserFlowStoresTokenAndUsesItForHTTPMCP(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	secrets := NewMemorySecretStore()
	service.SetSecretStore(secrets)
	ctx := context.Background()
	var httpServer *httptest.Server
	httpServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/oauth-protected-resource/mcp":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"resource":              httpServer.URL + "/mcp",
				"authorization_servers": []string{httpServer.URL},
				"scopes_supported":      []string{"files:read"},
			})
		case "/.well-known/oauth-authorization-server":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issuer":                                         httpServer.URL,
				"authorization_endpoint":                         httpServer.URL + "/authorize",
				"token_endpoint":                                 httpServer.URL + "/token",
				"registration_endpoint":                          httpServer.URL + "/register",
				"code_challenge_methods_supported":               []string{"S256"},
				"response_types_supported":                       []string{"code"},
				"grant_types_supported":                          []string{"authorization_code"},
				"authorization_response_iss_parameter_supported": true,
			})
		case "/register":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			redirects, _ := payload["redirect_uris"].([]any)
			if len(redirects) != 1 || redirects[0] != mcpOAuthRedirectURI() {
				t.Fatalf("registration payload = %#v, want localhost redirect", payload)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"client_id": "dynamic-client"})
		case "/token":
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if r.Form.Get("client_id") != "dynamic-client" || r.Form.Get("code") != "test-code" || r.Form.Get("code_verifier") == "" || r.Form.Get("resource") != httpServer.URL+"/mcp" {
				t.Fatalf("token form = %#v", r.Form)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "mcp-access-token", "refresh_token": "mcp-refresh-token", "expires_in": 3600, "token_type": "Bearer"})
		case "/mcp":
			if got := r.Header.Get("Authorization"); got != "Bearer mcp-access-token" {
				t.Fatalf("authorization = %q, want saved OAuth access token", got)
			}
			var request struct {
				ID     string `json:"id"`
				Method string `json:"method"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": mcpHelperResult(request.Method)})
		default:
			http.NotFound(w, r)
		}
	}))
	defer httpServer.Close()

	server := domain.MCPServerConfig{
		ID:        "oauth_flow_mcp",
		Name:      "oauth_flow_mcp",
		Transport: domain.MCPTransportStreamableHTTP,
		URL:       httpServer.URL + "/mcp",
		AuthType:  domain.MCPAuthOAuth,
	}
	if _, err := service.SaveMCPServer(ctx, domain.SaveMCPServerInput{Server: server}); err != nil {
		t.Fatal(err)
	}
	start, err := service.StartMCPOAuth(ctx, domain.MCPOAuthStartInput{ServerID: server.ID})
	if err != nil {
		t.Fatal(err)
	}
	authURL, err := url.Parse(start.URL)
	if err != nil {
		t.Fatal(err)
	}
	if authURL.Query().Get("client_id") != "dynamic-client" || authURL.Query().Get("scope") != "files:read" || authURL.Query().Get("code_challenge_method") != "S256" {
		t.Fatalf("authorize URL = %s", start.URL)
	}
	callbackURL := authURL.Query().Get("redirect_uri") + "?code=test-code&state=" + url.QueryEscape(authURL.Query().Get("state"))
	resp, err := http.Get(callbackURL)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("callback status = %d", resp.StatusCode)
	}
	status, err := service.GetMCPOAuthStatus(ctx, domain.MCPOAuthStatusInput{ServerID: server.ID})
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "success" || !status.Connected {
		t.Fatalf("status = %#v, want connected success", status)
	}
	saved, err := service.mcpManager.store.GetMCPServer(ctx, server.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.OAuthClientID != "dynamic-client" || saved.OAuthAccessTokenRef == "" || saved.OAuthRefreshTokenRef == "" || saved.OAuthAccessToken != "" {
		t.Fatalf("saved server = %#v, want token refs and dynamic client id", saved)
	}
	probe, err := service.ProbeMCPServer(ctx, domain.MCPProbeInput{ServerID: server.ID})
	if err != nil || !probe.OK {
		t.Fatalf("probe = %#v err = %v, want authorized probe", probe, err)
	}
}

func TestMCPOAuthRefreshesExpiredTokenBeforeHTTPMCPCall(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	secrets := NewMemorySecretStore()
	service.SetSecretStore(secrets)
	ctx := context.Background()
	refreshCalls := 0
	var httpServer *httptest.Server
	httpServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/oauth-protected-resource/mcp":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"resource":              httpServer.URL + "/mcp",
				"authorization_servers": []string{httpServer.URL},
				"scopes_supported":      []string{"files:read"},
			})
		case "/.well-known/oauth-authorization-server":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issuer":                 httpServer.URL,
				"authorization_endpoint": httpServer.URL + "/authorize",
				"token_endpoint":         httpServer.URL + "/token",
			})
		case "/token":
			refreshCalls++
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("client_id") != "refresh-client" || r.Form.Get("refresh_token") != "old-refresh-token" || r.Form.Get("resource") != httpServer.URL+"/mcp" {
				t.Fatalf("refresh form = %#v", r.Form)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "new-access-token", "expires_in": 3600, "token_type": "Bearer"})
		case "/mcp":
			if got := r.Header.Get("Authorization"); got != "Bearer new-access-token" {
				t.Fatalf("authorization = %q, want refreshed OAuth access token", got)
			}
			var request struct {
				ID     string `json:"id"`
				Method string `json:"method"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": mcpHelperResult(request.Method)})
		default:
			http.NotFound(w, r)
		}
	}))
	defer httpServer.Close()

	server := domain.MCPServerConfig{
		ID:                    "oauth_refresh_mcp",
		Name:                  "oauth_refresh_mcp",
		Transport:             domain.MCPTransportStreamableHTTP,
		URL:                   httpServer.URL + "/mcp",
		AuthType:              domain.MCPAuthOAuth,
		OAuthClientID:         "refresh-client",
		OAuthAccessToken:      "expired-access-token",
		OAuthRefreshToken:     "old-refresh-token",
		OAuthExpiresAt:        domain.NowString(time.Now().Add(-time.Minute)),
		OAuthScopes:           []string{"files:read"},
		TimeoutSeconds:        5,
		ConnectTimeoutSeconds: 5,
	}
	prepared, err := prepareMCPOAuthSecrets(ctx, secrets, server)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveMCPServer(ctx, domain.SaveMCPServerInput{Server: prepared}); err != nil {
		t.Fatal(err)
	}
	probe, err := service.ProbeMCPServer(ctx, domain.MCPProbeInput{ServerID: server.ID})
	if err != nil || !probe.OK {
		t.Fatalf("probe = %#v err = %v, want refresh then authorized probe", probe, err)
	}
	if refreshCalls != 1 {
		t.Fatalf("refreshCalls = %d, want one refresh", refreshCalls)
	}
	saved, err := service.mcpManager.store.GetMCPServer(ctx, server.ID)
	if err != nil {
		t.Fatal(err)
	}
	refreshed, err := resolveMCPOAuthSecrets(ctx, secrets, saved)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.OAuthAccessToken != "new-access-token" || refreshed.OAuthRefreshToken != "old-refresh-token" || !mcpOAuthConnected(saved) {
		t.Fatalf("refreshed server = %#v, want new access token and retained refresh token", refreshed)
	}
}

func TestMCPOAuthInsufficientScopeChallengePersistsRequestedScopes(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	var httpServer *httptest.Server
	httpServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("WWW-Authenticate", `Bearer error="insufficient_scope", error_description="Need write access", resource_metadata="`+httpServer.URL+`/.well-known/oauth-protected-resource/mcp", scope="files:read files:write"`)
		http.Error(w, `{"error":"insufficient_scope"}`, http.StatusForbidden)
	}))
	defer httpServer.Close()

	server := domain.MCPServerConfig{
		ID:             "oauth_scope_mcp",
		Name:           "oauth_scope_mcp",
		Transport:      domain.MCPTransportStreamableHTTP,
		URL:            httpServer.URL + "/mcp",
		AuthType:       domain.MCPAuthOAuth,
		OAuthClientID:  "scope-client",
		OAuthScopes:    []string{"files:read"},
		OAuthExpiresAt: domain.NowString(time.Now().Add(time.Hour)),
		TimeoutSeconds: 5,
	}
	if _, err := service.SaveMCPServer(ctx, domain.SaveMCPServerInput{Server: server}); err != nil {
		t.Fatal(err)
	}
	probe, err := service.ProbeMCPServer(ctx, domain.MCPProbeInput{ServerID: server.ID})
	if err != nil {
		t.Fatal(err)
	}
	if probe.OK || !strings.Contains(probe.Error, "reconnect OAuth to grant scopes: files:read files:write") {
		t.Fatalf("probe = %#v, want recoverable insufficient_scope error", probe)
	}
	saved, err := service.mcpManager.store.GetMCPServer(ctx, server.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(saved.OAuthScopes, "files:read") || !containsString(saved.OAuthScopes, "files:write") {
		t.Fatalf("scopes = %#v, want persisted requested scopes", saved.OAuthScopes)
	}
	if saved.Status != domain.MCPServerStatusError || !strings.Contains(saved.Error, "additional scopes") {
		t.Fatalf("saved status/error = %s/%q, want actionable oauth error", saved.Status, saved.Error)
	}
}

func TestMCPWWWAuthenticateParamsParsesQuotedScope(t *testing.T) {
	params := mcpWWWAuthenticateParams(`Bearer error="insufficient_scope", error_description="Need write access", resource_metadata="https://mcp.example/.well-known/oauth-protected-resource", scope="files:read files:write"`)
	if params["error"] != "insufficient_scope" || params["error_description"] != "Need write access" || params["scope"] != "files:read files:write" || params["resource_metadata"] == "" {
		t.Fatalf("params = %#v, want quoted auth challenge fields", params)
	}
}

func TestMCPSSEProbeParsesEventStreamResponses(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID     string `json:"id"`
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": mcpHelperResult(request.Method)})
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: message\n"))
		_, _ = w.Write([]byte("data: " + string(raw) + "\n\n"))
	}))
	defer httpServer.Close()

	result, err := probeMCPServer(context.Background(), domain.MCPServerConfig{
		ID:        "sse_mcp",
		Name:      "sse_mcp",
		Transport: domain.MCPTransportSSE,
		URL:       httpServer.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Tools) != 1 || result.Tools[0].Name != "echo" || len(result.Prompts) != 1 {
		t.Fatalf("result = %#v, want helper capabilities from SSE", result)
	}
}

func TestMCPHTTPRespondsToRootsListRequest(t *testing.T) {
	root := t.TempDir()
	rootsResponse := make(chan []any, 1)
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if method, _ := request["method"].(string); method == "" {
			result, _ := request["result"].(map[string]any)
			roots, _ := result["roots"].([]any)
			rootsResponse <- roots
			w.WriteHeader(http.StatusAccepted)
			return
		}
		requestID, _ := request["id"].(string)
		method, _ := request["method"].(string)
		if method == "initialize" {
			params, _ := request["params"].(map[string]any)
			capabilities, _ := params["capabilities"].(map[string]any)
			if _, ok := capabilities["roots"].(map[string]any); !ok {
				t.Fatalf("initialize params = %#v, want roots capability", params)
			}
			rootRequest, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": "roots-request", "method": "roots/list"})
			initResponse, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": requestID, "result": mcpHelperResult(method)})
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: message\n"))
			_, _ = w.Write([]byte("data: " + string(rootRequest) + "\n\n"))
			_, _ = w.Write([]byte("event: message\n"))
			_, _ = w.Write([]byte("data: " + string(initResponse) + "\n\n"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": requestID, "result": mcpHelperResult(method)})
	}))
	defer httpServer.Close()

	result, err := probeMCPServer(context.Background(), domain.MCPServerConfig{
		ID:        "http_roots_mcp",
		Name:      "http_roots_mcp",
		Transport: domain.MCPTransportStreamableHTTP,
		URL:       httpServer.URL,
		Roots:     []string{root},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Tools) != 1 || result.Tools[0].Name != "echo" {
		t.Fatalf("result = %#v, want helper capabilities after HTTP roots/list", result)
	}
	select {
	case roots := <-rootsResponse:
		if len(roots) != 1 {
			t.Fatalf("roots response = %#v, want one root", roots)
		}
		item, _ := roots[0].(map[string]any)
		uri, _ := item["uri"].(string)
		if !strings.HasPrefix(uri, "file://") {
			t.Fatalf("roots response = %#v, want file URI", roots)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for HTTP roots/list response")
	}
}

func TestMCPHTTPRefreshesToolsAfterListChangedNotification(t *testing.T) {
	toolsListCalls := 0
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID     string `json:"id"`
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		switch request.Method {
		case "tools/list":
			toolsListCalls++
			name := "before"
			if toolsListCalls > 1 {
				name = "after"
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": mcpToolsChangedHelperTools(name)})
		case "prompts/list":
			response, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": mcpHelperResult(request.Method)})
			notification, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "notifications/tools/list_changed"})
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: message\n"))
			_, _ = w.Write([]byte("data: " + string(notification) + "\n\n"))
			_, _ = w.Write([]byte("event: message\n"))
			_, _ = w.Write([]byte("data: " + string(response) + "\n\n"))
		default:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": mcpHelperResult(request.Method)})
		}
	}))
	defer httpServer.Close()

	result, err := probeMCPServer(context.Background(), domain.MCPServerConfig{
		ID:        "changed_http_mcp",
		Name:      "changed_http_mcp",
		Transport: domain.MCPTransportStreamableHTTP,
		URL:       httpServer.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if toolsListCalls != 2 {
		t.Fatalf("toolsListCalls = %d, want refresh after list_changed notification", toolsListCalls)
	}
	if len(result.Tools) != 1 || result.Tools[0].Name != "after" {
		t.Fatalf("tools = %#v, want refreshed after tool", result.Tools)
	}
	if len(result.Prompts) != 1 {
		t.Fatalf("prompts = %#v, want prompt response after notification event", result.Prompts)
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
	server := domain.MCPServerConfig{ID: "log/server", Name: "log/server", Transport: domain.MCPTransportStdio}
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

func TestMCPProbeHelperProcess(t *testing.T) {
	if !hasArg("mcp-helper") {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var request struct {
			ID     string `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil || request.ID == "" {
			continue
		}
		writeMCPHelperResponse(request.ID, mcpHelperResult(request.Method))
	}
	os.Exit(0)
}

func TestMCPRootsHelperProcess(t *testing.T) {
	if !hasArg("mcp-roots-helper") {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		os.Exit(1)
	}
	var initRequest struct {
		ID     string         `json:"id"`
		Method string         `json:"method"`
		Params map[string]any `json:"params"`
	}
	if err := json.Unmarshal(scanner.Bytes(), &initRequest); err != nil || initRequest.ID == "" || initRequest.Method != "initialize" {
		os.Exit(1)
	}
	rawRootRequest, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": "roots-request", "method": "roots/list"})
	_, _ = os.Stdout.Write(append(rawRootRequest, '\n'))
	if !scanner.Scan() {
		os.Exit(1)
	}
	var rootsResponse struct {
		Result struct {
			Roots []struct {
				URI string `json:"uri"`
			} `json:"roots"`
		} `json:"result"`
	}
	if err := json.Unmarshal(scanner.Bytes(), &rootsResponse); err != nil {
		writeMCPHelperError(initRequest.ID, "invalid roots/list response")
		os.Exit(0)
	}
	if len(rootsResponse.Result.Roots) != 1 || !strings.HasPrefix(rootsResponse.Result.Roots[0].URI, "file://") {
		writeMCPHelperError(initRequest.ID, "unexpected roots/list response: "+string(scanner.Bytes()))
		os.Exit(0)
	}
	capabilities, _ := initRequest.Params["capabilities"].(map[string]any)
	if _, ok := capabilities["roots"].(map[string]any); !ok {
		writeMCPHelperError(initRequest.ID, "initialize did not advertise roots capability")
		os.Exit(0)
	}
	writeMCPHelperResponse(initRequest.ID, mcpHelperResult("initialize"))
	for scanner.Scan() {
		var request struct {
			ID     string `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil || request.ID == "" {
			continue
		}
		writeMCPHelperResponse(request.ID, mcpHelperResult(request.Method))
	}
	os.Exit(0)
}

func TestMCPToolsChangedHelperProcess(t *testing.T) {
	if !hasArg("mcp-tools-changed-helper") {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	toolsListCalls := 0
	for scanner.Scan() {
		var request struct {
			ID     string `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil || request.ID == "" {
			continue
		}
		switch request.Method {
		case "tools/list":
			toolsListCalls++
			name := "before"
			if toolsListCalls > 1 {
				name = "after"
			}
			writeMCPHelperResponse(request.ID, mcpToolsChangedHelperTools(name))
		case "prompts/list":
			rawNotification, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "notifications/tools/list_changed"})
			_, _ = os.Stdout.Write(append(rawNotification, '\n'))
			writeMCPHelperResponse(request.ID, mcpHelperResult(request.Method))
		default:
			writeMCPHelperResponse(request.ID, mcpHelperResult(request.Method))
		}
	}
	os.Exit(0)
}

func TestMCPLongLivedHelperProcess(t *testing.T) {
	if !hasArg("mcp-long-lived-helper") {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	counter := 0
	for scanner.Scan() {
		var request struct {
			ID     string `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil || request.ID == "" {
			continue
		}
		switch request.Method {
		case "tools/list":
			writeMCPHelperResponse(request.ID, map[string]any{"tools": []any{map[string]any{
				"name":        "counter",
				"description": "Increment process-local counter",
				"inputSchema": map[string]any{"type": "object"},
			}}})
		case "tools/call":
			counter++
			writeMCPHelperResponse(request.ID, map[string]any{"content": []any{map[string]any{"type": "text", "text": strconv.Itoa(counter)}}})
		default:
			writeMCPHelperResponse(request.ID, mcpHelperResult(request.Method))
		}
	}
	os.Exit(0)
}

func TestMCPReconnectHelperProcess(t *testing.T) {
	if !hasArg("mcp-reconnect-helper") {
		return
	}
	marker := os.Getenv("AIVO_MCP_RECONNECT_FILE")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var request struct {
			ID     string `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil || request.ID == "" {
			continue
		}
		switch request.Method {
		case "tools/list":
			writeMCPHelperResponse(request.ID, map[string]any{"tools": []any{map[string]any{
				"name":        "recover",
				"description": "Recover after reconnect",
				"inputSchema": map[string]any{"type": "object"},
			}}})
		case "tools/call":
			if marker != "" {
				if _, err := os.Stat(marker); os.IsNotExist(err) {
					_ = os.WriteFile(marker, []byte("failed"), 0o600)
					os.Exit(0)
				}
			}
			writeMCPHelperResponse(request.ID, map[string]any{"content": []any{map[string]any{"type": "text", "text": "recovered"}}})
		default:
			writeMCPHelperResponse(request.ID, mcpHelperResult(request.Method))
		}
	}
	os.Exit(0)
}

func hasArg(value string) bool {
	for _, arg := range os.Args {
		if arg == value {
			return true
		}
	}
	return false
}

func textFromMCPToolContent(result map[string]any) string {
	blocks, _ := result["content"].([]any)
	for _, block := range blocks {
		item, _ := block.(map[string]any)
		if item["type"] == "text" {
			text, _ := item["text"].(string)
			return text
		}
	}
	return ""
}

func writeMCPHelperResponse(id string, result map[string]any) {
	raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
	_, _ = os.Stdout.Write(append(raw, '\n'))
}

func writeMCPHelperError(id string, message string) {
	raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": -32000, "message": message}})
	_, _ = os.Stdout.Write(append(raw, '\n'))
}

func mcpToolsChangedHelperTools(name string) map[string]any {
	return map[string]any{"tools": []any{map[string]any{
		"name":        name,
		"description": "Dynamic tool",
		"inputSchema": map[string]any{"type": "object"},
	}}}
}

func mcpHelperResult(method string) map[string]any {
	switch method {
	case "initialize":
		return map[string]any{"protocolVersion": "2025-03-26", "capabilities": map[string]any{}}
	case "tools/list":
		return map[string]any{"tools": []any{map[string]any{
			"name":        "echo",
			"description": "Echo text",
			"inputSchema": map[string]any{"type": "object"},
		}}}
	case "prompts/list":
		return map[string]any{"prompts": []any{map[string]any{
			"name":        "review",
			"description": "Review code",
			"arguments": []any{map[string]any{
				"name":        "path",
				"description": "Path to review",
				"required":    true,
			}},
		}}}
	case "resources/list":
		return map[string]any{"resources": []any{map[string]any{
			"uri":         "file:///README.md",
			"name":        "README.md",
			"description": "Project readme",
			"mimeType":    "text/markdown",
		}}}
	case "resources/templates/list":
		return map[string]any{"resourceTemplates": []any{map[string]any{
			"uriTemplate": "file:///{path}",
			"name":        "Project file",
			"description": "Read a project file",
			"mimeType":    "text/plain",
		}}}
	case "prompts/get":
		return map[string]any{
			"description": "Review code",
			"messages": []any{map[string]any{
				"role": "user",
				"content": map[string]any{
					"type": "text",
					"text": "Review README.md",
				},
			}},
		}
	case "resources/read":
		return map[string]any{"contents": []any{map[string]any{
			"uri":      "file:///README.md",
			"mimeType": "text/markdown",
			"text":     "# Aivo\n",
		}}}
	default:
		return map[string]any{}
	}
}
