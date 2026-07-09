package app

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strings"

	"aivo/core/domain"
)

func (s *Service) resolveCredentialWithDefinition(ctx context.Context, provider domain.ProviderConfig, def ProviderDefinition) (llmCredential, error) {
	auth, err := s.store.LoadProviderAuth(ctx, provider.ID)
	if err != nil {
		return llmCredential{}, err
	}
	if auth != nil {
		resolvedAuth, err := s.resolveProviderAuthSecrets(ctx, *auth)
		if err != nil {
			return llmCredential{}, err
		}
		auth = &resolvedAuth
		if isOAuthMethod(auth.Method) {
			if auth.AccessToken == "" && auth.RefreshToken == "" {
				return llmCredential{}, errors.New("OAuth credentials are missing")
			}
			return llmCredential{Method: auth.Method, AccessToken: auth.AccessToken, Refresh: auth.RefreshToken, ExpiresAt: auth.ExpiresAt, AccountID: auth.AccountID, AuthRecord: auth}, nil
		}
		if strings.TrimSpace(auth.APIKey) != "" {
			return llmCredential{Method: auth.Method, APIKey: strings.TrimSpace(auth.APIKey), AuthRecord: auth}, nil
		}
	}
	for _, envName := range credentialEnvCandidates(provider, def) {
		if value := lookupEnv(strings.TrimSpace(envName)); value != "" {
			return llmCredential{Method: "env", APIKey: value}, nil
		}
	}
	if strings.TrimSpace(provider.APIKey) != "" {
		return llmCredential{Method: "api-key", APIKey: strings.TrimSpace(provider.APIKey)}, nil
	}
	if providerAllowsNoCredential(provider, def) {
		return llmCredential{Method: "none"}, nil
	}
	return llmCredential{}, errors.New("credentials are not configured for provider " + provider.ID)
}

func credentialEnvCandidates(provider domain.ProviderConfig, def ProviderDefinition) []string {
	var out []string
	out = append(out, splitEnvCandidates(provider.APIKeyEnv)...)
	out = append(out, def.APIKeyEnvVars...)
	seen := map[string]bool{}
	filtered := out[:0]
	for _, item := range out {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		filtered = append(filtered, item)
	}
	return filtered
}

func splitEnvCandidates(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	fields := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' || r == '|' || r == ' ' || r == '\n' || r == '\t' })
	var out []string
	for _, field := range fields {
		if field = strings.TrimSpace(field); field != "" {
			out = append(out, field)
		}
	}
	return out
}

var lookupEnv = func(name string) string { return strings.TrimSpace(os.Getenv(name)) }

func providerAllowsNoCredential(provider domain.ProviderConfig, def ProviderDefinition) bool {
	if def.DefaultAuthType == AuthAWSSDK || containsAuthType(def.AuthTypes, AuthAWSSDK) {
		return true
	}
	if def.DefaultAuthType == AuthNone || containsAuthType(def.AuthTypes, AuthNone) {
		return true
	}
	base := strings.TrimSpace(provider.BaseURL)
	parsed, err := url.Parse(base)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func containsAuthType(items []AuthType, target AuthType) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
