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
	"time"

	"aivo/core/domain"
)

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
