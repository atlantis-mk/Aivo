package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"aivo/core/domain"
)

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
		ID:          "oauth_discovery_mcp",
		Name:        "oauth_discovery_mcp",
		Description: "Discover and authorize test MCP resources",
		Transport:   domain.MCPTransportStreamableHTTP,
		URL:         httpServer.URL + "/mcp",
		AuthType:    domain.MCPAuthOAuth,
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
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", mcpOAuthPort))
	if err != nil {
		t.Skipf("OAuth callback port is already in use: %v", err)
	}
	_ = listener.Close()
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
			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			if r.Method == http.MethodDelete {
				w.WriteHeader(http.StatusOK)
				return
			}
			if got := r.Header.Get("Authorization"); got != "Bearer mcp-access-token" {
				t.Fatalf("authorization = %q, want saved OAuth access token", got)
			}
			var request struct {
				ID     json.RawMessage `json:"id"`
				Method string          `json:"method"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if len(request.ID) == 0 {
				w.WriteHeader(http.StatusAccepted)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(request.ID), "result": mcpHelperResult(request.Method)})
		default:
			http.NotFound(w, r)
		}
	}))
	defer httpServer.Close()

	server := domain.MCPServerConfig{
		ID:          "oauth_flow_mcp",
		Name:        "oauth_flow_mcp",
		Description: "Exercise the OAuth browser flow for MCP",
		Transport:   domain.MCPTransportStreamableHTTP,
		URL:         httpServer.URL + "/mcp",
		AuthType:    domain.MCPAuthOAuth,
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
			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			if r.Method == http.MethodDelete {
				w.WriteHeader(http.StatusOK)
				return
			}
			if got := r.Header.Get("Authorization"); got != "Bearer new-access-token" {
				t.Fatalf("authorization = %q, want refreshed OAuth access token", got)
			}
			var request struct {
				ID     json.RawMessage `json:"id"`
				Method string          `json:"method"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if len(request.ID) == 0 {
				w.WriteHeader(http.StatusAccepted)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(request.ID), "result": mcpHelperResult(request.Method)})
		default:
			http.NotFound(w, r)
		}
	}))
	defer httpServer.Close()

	server := domain.MCPServerConfig{
		ID:                    "oauth_refresh_mcp",
		Name:                  "oauth_refresh_mcp",
		Description:           "Refresh OAuth credentials for MCP calls",
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
		Description:    "Validate OAuth scopes for MCP calls",
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
