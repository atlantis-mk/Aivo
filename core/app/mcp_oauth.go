package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"aivo/core/domain"
)

const mcpOAuthDiscoveryLimit = 2 << 20

const (
	mcpOAuthPort = 1456
	mcpOAuthPath = "/mcp/oauth/callback"
)

type mcpOAuthFlow struct {
	ServerID      string
	Status        string
	State         string
	Verifier      string
	URL           string
	Error         string
	ClientID      string
	TokenEndpoint string
	Resource      string
	RedirectURI   string
	ExpiresAt     time.Time
}

type mcpOAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

func discoverMCPOAuth(ctx context.Context, server domain.MCPServerConfig, challenge string) (domain.MCPOAuthDiscoveryResult, error) {
	result := domain.MCPOAuthDiscoveryResult{ServerID: server.ID}
	if server.Transport == "" {
		server.Transport = domain.MCPTransportStdio
	}
	if server.Transport == domain.MCPTransportStdio {
		return result, errors.New("mcp oauth discovery only applies to http transports")
	}
	serverURL, err := url.Parse(strings.TrimSpace(server.URL))
	if err != nil || !serverURL.IsAbs() || (serverURL.Scheme != "http" && serverURL.Scheme != "https") {
		return result, errors.New("absolute http MCP server URL is required for oauth discovery")
	}

	httpClient := &http.Client{Timeout: mcpHTTPTimeout(server)}
	resourceMetadataURL := mcpWWWAuthenticateParam(challenge, "resource_metadata")
	if resourceMetadataURL != "" {
		metadata, err := fetchMCPDiscoveryJSON(ctx, httpClient, resourceMetadataURL)
		if err != nil {
			result.DiscoveryErrors = append(result.DiscoveryErrors, fmt.Sprintf("%s: %v", resourceMetadataURL, err))
		} else {
			result.ResourceMetadataURL = resourceMetadataURL
			applyMCPResourceMetadata(&result, metadata)
		}
	}
	if result.ResourceMetadata == nil {
		for _, candidate := range mcpProtectedResourceMetadataURLs(*serverURL) {
			metadata, err := fetchMCPDiscoveryJSON(ctx, httpClient, candidate)
			if err != nil {
				result.DiscoveryErrors = append(result.DiscoveryErrors, fmt.Sprintf("%s: %v", candidate, err))
				continue
			}
			result.ResourceMetadataURL = candidate
			applyMCPResourceMetadata(&result, metadata)
			break
		}
	}

	issuers := append([]string{}, result.AuthorizationServers...)
	if strings.TrimSpace(server.OAuthIssuerURL) != "" {
		issuers = append([]string{strings.TrimSpace(server.OAuthIssuerURL)}, issuers...)
	}
	if len(issuers) == 0 {
		issuers = append(issuers, mcpOrigin(*serverURL))
	}
	for _, issuer := range uniqueNonEmptyStrings(issuers) {
		authMetadata, discoveryURL, err := discoverMCPOAuthServerMetadata(ctx, httpClient, issuer)
		if err != nil {
			result.DiscoveryErrors = append(result.DiscoveryErrors, fmt.Sprintf("%s: %v", issuer, err))
			continue
		}
		_ = discoveryURL
		applyMCPAuthorizationMetadata(&result, issuer, authMetadata)
		break
	}
	if result.SelectedIssuer == "" && server.OAuthIssuerURL == "" && result.ResourceMetadata == nil {
		return result, errors.New("mcp oauth discovery failed: protected resource metadata and authorization server metadata were not found")
	}
	if result.SelectedIssuer == "" && len(result.DiscoveryErrors) > 0 {
		return result, fmt.Errorf("mcp oauth discovery failed: %s", strings.Join(result.DiscoveryErrors, "; "))
	}
	result.RequiresDynamicClientReg = strings.TrimSpace(server.OAuthClientID) == "" && result.RegistrationEndpoint != ""
	return result, nil
}

func (m *MCPManager) startOAuth(ctx context.Context, input domain.MCPOAuthStartInput) (domain.MCPOAuthStartResult, error) {
	serverID := strings.TrimSpace(input.ServerID)
	if serverID == "" {
		return domain.MCPOAuthStartResult{}, errors.New("serverId is required")
	}
	server, err := m.store.GetMCPServer(ctx, serverID)
	if err != nil {
		return domain.MCPOAuthStartResult{}, err
	}
	if server.AuthType != domain.MCPAuthOAuth || server.Transport == domain.MCPTransportStdio {
		return domain.MCPOAuthStartResult{}, errors.New("mcp oauth browser flow requires an OAuth HTTP/SSE server")
	}
	discovery, err := discoverMCPOAuth(ctx, server, "")
	if err != nil {
		return domain.MCPOAuthStartResult{}, err
	}
	if strings.TrimSpace(discovery.AuthorizationEndpoint) == "" || strings.TrimSpace(discovery.TokenEndpoint) == "" {
		return domain.MCPOAuthStartResult{}, errors.New("mcp oauth discovery did not provide authorization and token endpoints")
	}
	if err := m.ensureMCPOAuthServer(); err != nil {
		return domain.MCPOAuthStartResult{}, err
	}
	clientID := strings.TrimSpace(server.OAuthClientID)
	if clientID == "" && strings.TrimSpace(discovery.RegistrationEndpoint) != "" {
		clientID, err = registerMCPOAuthClient(ctx, discovery.RegistrationEndpoint, mcpOAuthRedirectURI(), server)
		if err != nil {
			return domain.MCPOAuthStartResult{}, err
		}
		server.OAuthClientID = clientID
		if _, err := m.store.SaveMCPServer(ctx, server); err != nil {
			return domain.MCPOAuthStartResult{}, err
		}
	}
	if clientID == "" {
		return domain.MCPOAuthStartResult{}, errors.New("mcp oauth requires oauthClientId or dynamic client registration")
	}
	verifier, challenge, err := generatePKCE()
	if err != nil {
		return domain.MCPOAuthStartResult{}, err
	}
	state, err := randomToken(32)
	if err != nil {
		return domain.MCPOAuthStartResult{}, err
	}
	redirectURI := mcpOAuthRedirectURI()
	resource := firstNonEmptyApp(discovery.Resource, canonicalMCPOAuthResource(server.URL))
	authURL, err := buildMCPOAuthAuthorizeURL(discovery.AuthorizationEndpoint, clientID, redirectURI, challenge, state, resource, mcpOAuthScope(server, discovery))
	if err != nil {
		return domain.MCPOAuthStartResult{}, err
	}
	flow := &mcpOAuthFlow{
		ServerID: server.ID, Status: "pending", State: state, Verifier: verifier, URL: authURL,
		ClientID: clientID, TokenEndpoint: discovery.TokenEndpoint, Resource: resource, RedirectURI: redirectURI,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	m.mu.Lock()
	m.oauthFlows[state] = flow
	m.mu.Unlock()
	return domain.MCPOAuthStartResult{
		ServerID: server.ID, Status: flow.Status, URL: flow.URL,
		Instructions: "Complete MCP authorization in your browser.",
		ExpiresAt:    domain.NowString(flow.ExpiresAt),
	}, nil
}

func (m *MCPManager) oauthStatus(ctx context.Context, input domain.MCPOAuthStatusInput) (domain.MCPOAuthStatus, error) {
	serverID := strings.TrimSpace(input.ServerID)
	if serverID == "" {
		return domain.MCPOAuthStatus{}, errors.New("serverId is required")
	}
	m.mu.Lock()
	for _, flow := range m.oauthFlows {
		if flow.ServerID == serverID {
			status := domain.MCPOAuthStatus{ServerID: serverID, Status: flow.Status, Error: flow.Error, Connected: flow.Status == "success", ClientID: flow.ClientID, ExpiresAt: domain.NowString(flow.ExpiresAt)}
			m.mu.Unlock()
			return status, nil
		}
	}
	m.mu.Unlock()
	server, err := m.store.GetMCPServer(ctx, serverID)
	if err != nil {
		return domain.MCPOAuthStatus{}, err
	}
	return domain.MCPOAuthStatus{
		ServerID: server.ID, Status: "idle", Connected: mcpOAuthConnected(server), ExpiresAt: server.OAuthExpiresAt,
		ClientID: server.OAuthClientID, TokenSource: firstNonEmptyApp(server.OAuthAccessTokenRef, server.BearerTokenEnv),
	}, nil
}

func (m *MCPManager) ensureMCPOAuthServer() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.oauthServer != nil {
		return nil
	}
	mux := http.NewServeMux()
	mux.HandleFunc(mcpOAuthPath, m.handleMCPOAuthCallback)
	server := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", mcpOAuthPort), Handler: mux}
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return err
	}
	m.oauthServer = server
	go func() {
		_ = server.Serve(listener)
	}()
	return nil
}

func (m *MCPManager) handleMCPOAuthCallback(w http.ResponseWriter, r *http.Request) {
	if err := m.completeMCPOAuthCallback(r.Context(), r.URL.Query()); err != nil {
		writeOAuthHTML(w, false, err.Error())
		return
	}
	writeOAuthHTML(w, true, "")
}

func (m *MCPManager) completeMCPOAuthCallback(ctx context.Context, query url.Values) error {
	state := query.Get("state")
	m.mu.Lock()
	flow := m.oauthFlows[state]
	m.mu.Unlock()
	if flow == nil || state == "" || time.Now().After(flow.ExpiresAt) {
		return errors.New("invalid or expired MCP OAuth callback")
	}
	if errText := query.Get("error"); errText != "" {
		message := firstNonEmptyApp(query.Get("error_description"), errText)
		m.failMCPOAuthFlow(flow, message)
		return errors.New(message)
	}
	code := query.Get("code")
	if code == "" {
		m.failMCPOAuthFlow(flow, "OAuth callback did not include a code")
		return errors.New("OAuth callback did not include a code")
	}
	tokens, err := exchangeMCPOAuthCode(ctx, flow, code)
	if err != nil {
		m.failMCPOAuthFlow(flow, err.Error())
		return err
	}
	server, err := m.store.GetMCPServer(ctx, flow.ServerID)
	if err != nil {
		m.failMCPOAuthFlow(flow, err.Error())
		return err
	}
	if err := m.saveMCPOAuthTokens(ctx, server, tokens); err != nil {
		m.failMCPOAuthFlow(flow, err.Error())
		return err
	}
	m.mu.Lock()
	flow.Status = "success"
	m.mu.Unlock()
	return nil
}

func (m *MCPManager) failMCPOAuthFlow(flow *mcpOAuthFlow, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	flow.Status = "error"
	flow.Error = message
}

func (m *MCPManager) saveMCPOAuthTokens(ctx context.Context, server domain.MCPServerConfig, tokens mcpOAuthTokenResponse) error {
	expiresIn := tokens.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	server.OAuthAccessToken = tokens.AccessToken
	server.OAuthRefreshToken = tokens.RefreshToken
	server.OAuthExpiresAt = domain.NowString(time.Now().Add(time.Duration(expiresIn) * time.Second))
	prepared, err := prepareMCPOAuthSecrets(ctx, m.secrets, server)
	if err != nil {
		return err
	}
	_, err = m.store.SaveMCPServer(ctx, prepared)
	return err
}

func (m *MCPManager) refreshOAuthIfNeeded(ctx context.Context, server domain.MCPServerConfig) (domain.MCPServerConfig, error) {
	if server.AuthType != domain.MCPAuthOAuth {
		return server, nil
	}
	if strings.TrimSpace(server.OAuthAccessToken) == "" {
		return server, nil
	}
	if !mcpOAuthTokenExpired(server, 60*time.Second) {
		return server, nil
	}
	if strings.TrimSpace(server.OAuthRefreshToken) == "" {
		return server, errors.New("mcp oauth access token is expired and refresh token is missing")
	}
	if strings.TrimSpace(server.OAuthClientID) == "" {
		return server, errors.New("mcp oauth access token is expired and oauthClientId is missing")
	}
	discovery, err := discoverMCPOAuth(ctx, server, "")
	if err != nil {
		return server, err
	}
	if strings.TrimSpace(discovery.TokenEndpoint) == "" {
		return server, errors.New("mcp oauth discovery did not provide token endpoint")
	}
	resource := firstNonEmptyApp(discovery.Resource, canonicalMCPOAuthResource(server.URL))
	tokens, err := refreshMCPOAuthToken(ctx, discovery.TokenEndpoint, server.OAuthClientID, server.OAuthRefreshToken, resource, mcpOAuthScope(server, discovery))
	if err != nil {
		return server, err
	}
	if tokens.RefreshToken == "" {
		tokens.RefreshToken = server.OAuthRefreshToken
	}
	if err := m.saveMCPOAuthTokens(ctx, server, tokens); err != nil {
		return server, err
	}
	next, err := m.store.GetMCPServer(ctx, server.ID)
	if err != nil {
		return server, err
	}
	return resolveMCPOAuthSecrets(ctx, m.secrets, next)
}

func mcpOAuthTokenExpired(server domain.MCPServerConfig, skew time.Duration) bool {
	expiresAt := strings.TrimSpace(server.OAuthExpiresAt)
	if expiresAt == "" {
		return false
	}
	parsed, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return false
	}
	return time.Now().Add(skew).After(parsed)
}

func refreshMCPOAuthToken(ctx context.Context, endpoint string, clientID string, refreshToken string, resource string, scopes []string) (mcpOAuthTokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", clientID)
	if strings.TrimSpace(resource) != "" {
		form.Set("resource", resource)
	}
	if len(scopes) > 0 {
		form.Set("scope", strings.Join(scopes, " "))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return mcpOAuthTokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("MCP-Protocol-Version", "2025-03-26")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return mcpOAuthTokenResponse{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, mcpOAuthDiscoveryLimit))
	if err != nil {
		return mcpOAuthTokenResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mcpOAuthTokenResponse{}, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, bounded(strings.TrimSpace(string(raw)), 500))
	}
	var tokens mcpOAuthTokenResponse
	if err := json.Unmarshal(raw, &tokens); err != nil {
		return mcpOAuthTokenResponse{}, err
	}
	if strings.TrimSpace(tokens.AccessToken) == "" {
		return mcpOAuthTokenResponse{}, errors.New("token refresh did not return access_token")
	}
	return tokens, nil
}

func registerMCPOAuthClient(ctx context.Context, endpoint string, redirectURI string, server domain.MCPServerConfig) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"client_name":                firstNonEmptyApp(server.DisplayName, server.Name, "Aivo"),
		"redirect_uris":              []string{redirectURI},
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
		"token_endpoint_auth_method": "none",
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("MCP-Protocol-Version", "2025-03-26")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, mcpOAuthDiscoveryLimit))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("dynamic client registration failed with status %d: %s", resp.StatusCode, bounded(strings.TrimSpace(string(raw)), 500))
	}
	var payload struct {
		ClientID string `json:"client_id"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", err
	}
	if strings.TrimSpace(payload.ClientID) == "" {
		return "", errors.New("dynamic client registration did not return client_id")
	}
	return strings.TrimSpace(payload.ClientID), nil
}

func buildMCPOAuthAuthorizeURL(endpoint string, clientID string, redirectURI string, challenge string, state string, resource string, scopes []string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	values := parsed.Query()
	values.Set("response_type", "code")
	values.Set("client_id", clientID)
	values.Set("redirect_uri", redirectURI)
	values.Set("state", state)
	values.Set("code_challenge", challenge)
	values.Set("code_challenge_method", "S256")
	if strings.TrimSpace(resource) != "" {
		values.Set("resource", resource)
	}
	if len(scopes) > 0 {
		values.Set("scope", strings.Join(scopes, " "))
	}
	parsed.RawQuery = values.Encode()
	return parsed.String(), nil
}

func exchangeMCPOAuthCode(ctx context.Context, flow *mcpOAuthFlow, code string) (mcpOAuthTokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", flow.RedirectURI)
	form.Set("client_id", flow.ClientID)
	form.Set("code_verifier", flow.Verifier)
	if strings.TrimSpace(flow.Resource) != "" {
		form.Set("resource", flow.Resource)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, flow.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return mcpOAuthTokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("MCP-Protocol-Version", "2025-03-26")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return mcpOAuthTokenResponse{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, mcpOAuthDiscoveryLimit))
	if err != nil {
		return mcpOAuthTokenResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mcpOAuthTokenResponse{}, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, bounded(strings.TrimSpace(string(raw)), 500))
	}
	var tokens mcpOAuthTokenResponse
	if err := json.Unmarshal(raw, &tokens); err != nil {
		return mcpOAuthTokenResponse{}, err
	}
	if strings.TrimSpace(tokens.AccessToken) == "" {
		return mcpOAuthTokenResponse{}, errors.New("token exchange did not return access_token")
	}
	return tokens, nil
}

func mcpOAuthRedirectURI() string {
	return fmt.Sprintf("http://127.0.0.1:%d%s", mcpOAuthPort, mcpOAuthPath)
}

func mcpOAuthScope(server domain.MCPServerConfig, discovery domain.MCPOAuthDiscoveryResult) []string {
	if len(server.OAuthScopes) > 0 {
		return uniqueNonEmptyStrings(server.OAuthScopes)
	}
	return uniqueNonEmptyStrings(discovery.ScopesSupported)
}

func canonicalMCPOAuthResource(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || !parsed.IsAbs() {
		return strings.TrimSpace(rawURL)
	}
	parsed.Fragment = ""
	parsed.RawQuery = ""
	if parsed.Path == "/" {
		parsed.Path = ""
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	return parsed.String()
}

func mcpHTTPTimeout(server domain.MCPServerConfig) time.Duration {
	timeout := time.Duration(server.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return timeout
}

func fetchMCPDiscoveryJSON(ctx context.Context, client *http.Client, rawURL string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("MCP-Protocol-Version", "2025-03-26")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, mcpOAuthDiscoveryLimit))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, bounded(strings.TrimSpace(string(body)), 500))
	}
	var metadata map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(body), &metadata); err != nil {
		return nil, err
	}
	return metadata, nil
}

func mcpProtectedResourceMetadataURLs(serverURL url.URL) []string {
	origin := mcpOrigin(serverURL)
	candidates := []string{}
	path := strings.TrimRight(serverURL.EscapedPath(), "/")
	if path != "" {
		candidates = append(candidates, origin+"/.well-known/oauth-protected-resource"+path)
	}
	candidates = append(candidates, origin+"/.well-known/oauth-protected-resource")
	return uniqueNonEmptyStrings(candidates)
}

func mcpOrigin(serverURL url.URL) string {
	return strings.ToLower(serverURL.Scheme) + "://" + strings.ToLower(serverURL.Host)
}

func applyMCPResourceMetadata(result *domain.MCPOAuthDiscoveryResult, metadata map[string]any) {
	result.ResourceMetadata = metadata
	if resource, _ := metadata["resource"].(string); strings.TrimSpace(resource) != "" {
		result.Resource = strings.TrimSpace(resource)
	}
	result.AuthorizationServers = stringSliceFromAny(metadata["authorization_servers"])
	result.ScopesSupported = stringSliceFromAny(metadata["scopes_supported"])
}

func discoverMCPOAuthServerMetadata(ctx context.Context, client *http.Client, issuer string) (map[string]any, string, error) {
	candidates, err := mcpAuthorizationMetadataURLs(issuer)
	if err != nil {
		return nil, "", err
	}
	var failures []string
	for _, candidate := range candidates {
		metadata, err := fetchMCPDiscoveryJSON(ctx, client, candidate)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", candidate, err))
			continue
		}
		if got, _ := metadata["issuer"].(string); got != issuer {
			failures = append(failures, fmt.Sprintf("%s: issuer %q does not match %q", candidate, got, issuer))
			continue
		}
		return metadata, candidate, nil
	}
	return nil, "", errors.New(strings.Join(failures, "; "))
}

func mcpAuthorizationMetadataURLs(issuer string) ([]string, error) {
	parsed, err := url.Parse(strings.TrimSpace(issuer))
	if err != nil || !parsed.IsAbs() || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, errors.New("authorization server issuer must be an absolute http URL")
	}
	origin := mcpOrigin(*parsed)
	path := strings.TrimRight(parsed.EscapedPath(), "/")
	if path == "" {
		return []string{
			origin + "/.well-known/oauth-authorization-server",
			origin + "/.well-known/openid-configuration",
		}, nil
	}
	return []string{
		origin + "/.well-known/oauth-authorization-server" + path,
		origin + "/.well-known/openid-configuration" + path,
		origin + path + "/.well-known/openid-configuration",
	}, nil
}

func applyMCPAuthorizationMetadata(result *domain.MCPOAuthDiscoveryResult, issuer string, metadata map[string]any) {
	result.SelectedIssuer = issuer
	result.AuthorizationMetadata = metadata
	result.AuthorizationEndpoint, _ = metadata["authorization_endpoint"].(string)
	result.TokenEndpoint, _ = metadata["token_endpoint"].(string)
	result.RegistrationEndpoint, _ = metadata["registration_endpoint"].(string)
	result.IntrospectionEndpoint, _ = metadata["introspection_endpoint"].(string)
	result.RevocationEndpoint, _ = metadata["revocation_endpoint"].(string)
	result.CodeChallengeMethods = stringSliceFromAny(metadata["code_challenge_methods_supported"])
	result.ResponseTypesSupported = stringSliceFromAny(metadata["response_types_supported"])
	result.GrantTypesSupported = stringSliceFromAny(metadata["grant_types_supported"])
}

func stringSliceFromAny(value any) []string {
	if raw, ok := value.([]string); ok {
		return uniqueNonEmptyStrings(raw)
	}
	raw, _ := value.([]any)
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
			out = append(out, strings.TrimSpace(text))
		}
	}
	return uniqueNonEmptyStrings(out)
}
