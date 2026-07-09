package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"aivo/core/domain"
)

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
