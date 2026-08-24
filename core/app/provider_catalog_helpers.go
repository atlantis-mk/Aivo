package app

import "aivo/core/domain"

func buildProviderAuthMethods(id string, env string) []domain.ProviderAuthMethod {
	methods := []domain.ProviderAuthMethod{{ID: "env", Label: "Credential reference", Stable: true, Available: true, Description: env}}
	if id == "openai" {
		methods = append([]domain.ProviderAuthMethod{
			{
				ID:          "oauth-browser",
				Label:       "ChatGPT Pro/Plus (browser)",
				Stable:      false,
				Available:   true,
				Description: "OpenAI browser OAuth with PKCE and localhost callback",
			},
			{
				ID:          "oauth-headless",
				Label:       "ChatGPT Pro/Plus (headless)",
				Stable:      false,
				Available:   true,
				Description: "OpenAI device authorization flow",
			},
			{
				ID:        "api-key",
				Label:     "API Key",
				Stable:    true,
				Available: true,
			},
		}, methods...)
	}
	if id != "openai" && env != "" {
		methods = append(methods, domain.ProviderAuthMethod{ID: "api-key", Label: "API Key", Stable: true, Available: true})
	}
	if id == "custom-api" {
		methods = append(methods, domain.ProviderAuthMethod{ID: "api-key", Label: "API Key", Stable: true, Available: true})
	}
	return methods
}

func flattenProviderModels(providers []domain.ProviderInfo) []domain.ModelInfo {
	var out []domain.ModelInfo
	for _, provider := range providers {
		out = append(out, provider.Models...)
	}
	return out
}
