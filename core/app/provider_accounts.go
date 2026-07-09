package app

import (
	"strings"

	"aivo/core/domain"
)

func providerAccountsFromAuth(records []domain.ProviderAuthRecord) []domain.ProviderAccountInfo {
	accounts := make([]domain.ProviderAccountInfo, 0, len(records))
	seen := map[string]bool{}
	for _, record := range records {
		if record.Method == "env" {
			continue
		}
		accountID := strings.TrimSpace(record.AccountID)
		if accountID == "" {
			accountID = accountLabelFromAuth(record)
		}
		key := record.ProviderID + "\x00" + record.Method + "\x00" + accountID
		if seen[key] {
			continue
		}
		seen[key] = true
		id := record.ID
		if id == "" {
			id = record.ProviderID + ":" + record.Method + ":" + accountID
		}
		displayName := strings.TrimSpace(record.DisplayName)
		if displayName == "" && record.ProviderID == "openai" && (record.Method == "oauth-browser" || record.Method == "oauth-headless") {
			displayName = extractOpenAIAccountDisplayName(openAITokenResponse{AccessToken: record.AccessToken})
		}
		if displayName == "" {
			displayName = accountID
		}
		accounts = append(accounts, domain.ProviderAccountInfo{
			ID:          id,
			ProviderID:  record.ProviderID,
			Method:      record.Method,
			AccountID:   accountID,
			DisplayName: displayName,
			ConnectedAt: record.UpdatedAt,
		})
	}
	return accounts
}

func connectAccountLabel(providerID string, method string, apiKey string, env string) string {
	if apiKey != "" {
		if len(apiKey) > 8 {
			return "..." + apiKey[len(apiKey)-6:]
		}
		return "API Key"
	}
	if env != "" {
		return env
	}
	if method == "env" {
		return defaultEnvFor(providerID)
	}
	if method == "oauth-browser" || method == "oauth-headless" {
		return "OpenAI"
	}
	return "默认账号"
}

func accountLabelFromAuth(record domain.ProviderAuthRecord) string {
	if record.APIKey != "" {
		return connectAccountLabel(record.ProviderID, record.Method, record.APIKey, "")
	}
	if record.APIKeyRef != "" {
		return "API Key"
	}
	if record.AccessTokenRef != "" || record.RefreshTokenRef != "" {
		if record.ProviderID == "openai" {
			return "OpenAI"
		}
		return "OAuth account"
	}
	return connectAccountLabel(record.ProviderID, record.Method, "", "")
}

func normalizeHeaders(headers map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range headers {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
