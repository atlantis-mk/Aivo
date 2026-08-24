package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"aivo/core/domain"
)

func prepareMCPOAuthSecrets(ctx context.Context, secrets SecretStore, server domain.MCPServerConfig) (domain.MCPServerConfig, error) {
	if secrets == nil {
		secrets = NewMemorySecretStore()
	}
	if strings.TrimSpace(server.OAuthAccessToken) != "" {
		ref := firstNonEmptyApp(server.OAuthAccessTokenRef, mcpSecretRef(server, "access-token"))
		if err := secrets.Put(ctx, ref, server.OAuthAccessToken); err != nil {
			return domain.MCPServerConfig{}, err
		}
		server.OAuthAccessTokenRef = ref
		server.OAuthAccessToken = ""
	}
	if strings.TrimSpace(server.OAuthRefreshToken) != "" {
		ref := firstNonEmptyApp(server.OAuthRefreshTokenRef, mcpSecretRef(server, "refresh-token"))
		if err := secrets.Put(ctx, ref, server.OAuthRefreshToken); err != nil {
			return domain.MCPServerConfig{}, err
		}
		server.OAuthRefreshTokenRef = ref
		server.OAuthRefreshToken = ""
	}
	return server, nil
}

func resolveMCPOAuthSecrets(ctx context.Context, secrets SecretStore, server domain.MCPServerConfig) (domain.MCPServerConfig, error) {
	if secrets == nil || server.AuthType != domain.MCPAuthOAuth {
		return server, nil
	}
	if strings.TrimSpace(server.OAuthAccessToken) == "" && strings.TrimSpace(server.OAuthAccessTokenRef) != "" {
		value, err := secrets.Get(ctx, server.OAuthAccessTokenRef)
		if err != nil {
			return domain.MCPServerConfig{}, err
		}
		server.OAuthAccessToken = value
	}
	if strings.TrimSpace(server.OAuthRefreshToken) == "" && strings.TrimSpace(server.OAuthRefreshTokenRef) != "" {
		value, err := secrets.Get(ctx, server.OAuthRefreshTokenRef)
		if err != nil {
			return domain.MCPServerConfig{}, err
		}
		server.OAuthRefreshToken = value
	}
	return server, nil
}

func resolveMCPAuthSecrets(ctx context.Context, secrets SecretStore, server domain.MCPServerConfig) (domain.MCPServerConfig, error) {
	if server.AuthType == domain.MCPAuthBearer && strings.TrimSpace(server.BearerTokenRef) != "" {
		if secrets == nil {
			return domain.MCPServerConfig{}, errors.New("mcp bearer credential store is unavailable")
		}
		value, err := secrets.Get(ctx, server.BearerTokenRef)
		if err != nil {
			return domain.MCPServerConfig{}, err
		}
		if strings.TrimSpace(value) == "" {
			return domain.MCPServerConfig{}, errors.New("mcp bearer credential is unavailable")
		}
		server.BearerToken = value
		return server, nil
	}
	return resolveMCPOAuthSecrets(ctx, secrets, server)
}

func mcpSecretRef(server domain.MCPServerConfig, kind string) string {
	return "mcp-auth/" + cleanSecretRefPart(server.ID) + "/" + cleanSecretRefPart(kind)
}

func mcpOAuthConnected(server domain.MCPServerConfig) bool {
	if strings.TrimSpace(server.OAuthAccessTokenRef) == "" && strings.TrimSpace(server.OAuthAccessToken) == "" {
		return false
	}
	if strings.TrimSpace(server.OAuthExpiresAt) == "" {
		return true
	}
	expiresAt, err := time.Parse(time.RFC3339, server.OAuthExpiresAt)
	if err != nil {
		return true
	}
	if time.Now().Before(expiresAt.Add(-30 * time.Second)) {
		return true
	}
	return strings.TrimSpace(server.OAuthRefreshTokenRef) != "" || strings.TrimSpace(server.OAuthRefreshToken) != ""
}
