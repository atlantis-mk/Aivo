package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"aivo/core/domain"
)

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
