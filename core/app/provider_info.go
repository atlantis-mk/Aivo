package app

import (
	"sort"
	"strings"

	"aivo/core/domain"
)

func providerInfoFromDefinition(def ProviderDefinition) domain.ProviderInfo {
	env := ""
	if len(def.APIKeyEnvVars) > 0 {
		env = def.APIKeyEnvVars[0]
	}
	models := append([]domain.ModelInfo(nil), def.Models...)
	if len(models) == 0 && def.DefaultModelID != "" {
		models = []domain.ModelInfo{model(def.ID, def.DefaultModelID, def.DefaultModelID, true, 0, nil)}
	}
	return domain.ProviderInfo{
		ID: def.ID, Name: def.DisplayName, Description: def.Description, Type: string(def.Transport), BaseURL: def.DefaultBaseURL,
		BuiltIn: def.BuiltIn, Custom: !def.BuiltIn, Experimental: def.Experimental, Deprecated: def.Deprecated, Environment: env,
		DefaultModelID: def.DefaultModelID, Models: models, AuthMethods: authMethodsForDefinition(def),
		ModelRefresh: &domain.ProviderModelRefresh{
			Strategy: string(def.ModelFetch), Status: "idle", ModelCount: len(models), Refreshable: providerModelRefreshable(def),
			ParserType: parserTypeForModelFetch(def.ModelFetch), Endpoint: modelEndpointForDefinition(def),
		},
		Profile: &domain.ProviderProfile{
			ID: def.ID, DisplayName: def.DisplayName, ProviderType: string(def.Transport), InteractiveAuth: providerSupportsInteractiveAuth(def),
			ModelFetch: string(def.ModelFetch),
			ParserType: parserTypeForModelFetch(def.ModelFetch), ModelEndpoint: modelEndpointForDefinition(def), MessageShape: string(def.Transport),
			SupportedExtras: supportedExtrasForTransport(def.Transport),
			RequestProfile:  requestProfilePointer(def.RequestProfile),
		},
	}
}

func requestProfilePointer(profile domain.ProviderRequestProfile) *domain.ProviderRequestProfile {
	if len(profile.Headers) == 0 && len(profile.Params) == 0 && len(profile.ModelOverrides) == 0 {
		return nil
	}
	cloned := cloneRequestProfile(profile)
	return &cloned
}

func authMethodsForDefinition(def ProviderDefinition) []domain.ProviderAuthMethod {
	var methods []domain.ProviderAuthMethod
	seen := map[string]bool{}
	add := func(id, label, description string, stable bool) {
		if seen[id] {
			return
		}
		seen[id] = true
		methods = append(methods, domain.ProviderAuthMethod{ID: id, Label: label, Stable: stable, Available: true, Description: description})
	}
	for _, authType := range def.AuthTypes {
		switch authType {
		case AuthOAuthBrowser:
			if def.ID == "openai" {
				add("oauth-browser", "ChatGPT Pro/Plus (browser)", "OpenAI browser OAuth with PKCE and localhost callback.", false)
			} else {
				add("oauth-browser", "OAuth browser", "Browser OAuth with localhost callback.", false)
			}
		case AuthOAuthDevice:
			if def.ID == "openai" {
				add("oauth-headless", "ChatGPT Pro/Plus (headless)", "OpenAI device authorization flow.", false)
			} else {
				add("oauth-headless", "OAuth device code", "Device authorization flow for headless environments.", false)
			}
		case AuthAPIKey:
			add("api-key", "API Key", "", true)
			if len(def.APIKeyEnvVars) > 0 {
				add("env", "Credential reference", strings.Join(def.APIKeyEnvVars, ", "), true)
			}
		case AuthNone:
			add("none", "No credential", "Use for local or unauthenticated compatible endpoints.", true)
		case AuthAWSSDK:
			add("aws-sdk", "AWS SDK", "Resolve credentials from the AWS SDK chain.", true)
		case AuthExternalProcess:
			add("external-process", "External process", "Resolve credentials from an external provider process.", false)
		}
	}
	return methods
}

func defaultProviders() []domain.ProviderInfo {
	defs := providerDefinitions()
	out := make([]domain.ProviderInfo, 0, len(defs))
	for _, def := range defs {
		out = append(out, providerInfoFromDefinition(def))
	}
	return out
}

func defaultEnvFor(providerID string) string {
	if def, ok := providerDefinition(providerID); ok && len(def.APIKeyEnvVars) > 0 {
		return def.APIKeyEnvVars[0]
	}
	return ""
}

func defaultEnvCandidatesFor(providerID string) []string {
	if def, ok := providerDefinition(providerID); ok {
		return append([]string(nil), def.APIKeyEnvVars...)
	}
	return nil
}

func defaultModelFor(providerID string) string {
	if def, ok := providerDefinition(providerID); ok {
		return def.DefaultModelID
	}
	return "default"
}

func defaultBaseURLFor(providerID string) string {
	if def, ok := providerDefinition(providerID); ok {
		return def.DefaultBaseURL
	}
	return ""
}

func providerTypeFor(providerID string) string {
	if def, ok := providerDefinition(providerID); ok {
		return string(def.Transport)
	}
	return normalizeProviderID(providerID)
}

func providerModelRefreshable(def ProviderDefinition) bool {
	return def.ModelFetch != ModelFetchStatic && def.ModelFetch != ModelFetchDisabled
}

func providerSupportsInteractiveAuth(def ProviderDefinition) bool {
	return containsAuthType(def.AuthTypes, AuthOAuthBrowser) || containsAuthType(def.AuthTypes, AuthOAuthDevice) || containsAuthType(def.AuthTypes, AuthExternalProcess)
}

func parserTypeForModelFetch(strategy ModelFetchStrategy) string {
	switch strategy {
	case ModelFetchAnthropic:
		return "anthropic"
	case ModelFetchMistral:
		return "mistral"
	case ModelFetchOpenRouter:
		return "openrouter"
	case ModelFetchCerebras:
		return "cerebras"
	case ModelFetchGoogle:
		return "google"
	case ModelFetchOpenAICodexAccount:
		return "openai-codex"
	case ModelFetchOpenAICompatible:
		return "openai-compatible"
	default:
		return string(strategy)
	}
}

func modelEndpointForDefinition(def ProviderDefinition) string {
	return modelEndpointForFetchStrategy(def, def.ModelFetch)
}

func modelEndpointForFetchStrategy(def ProviderDefinition, strategy ModelFetchStrategy) string {
	if strategy == ModelFetchOpenAICodexAccount && def.BuiltIn && def.ID == "openai" {
		return chatGPTCodexModelsURL
	}
	if def.DefaultBaseURL == "" || strategy == ModelFetchDisabled || strategy == ModelFetchStatic {
		return ""
	}
	if def.ModelFetch == ModelFetchCerebras && def.BuiltIn && def.ID == "cerebras" {
		return cerebrasPublicModelsURL
	}
	return strings.TrimRight(def.DefaultBaseURL, "/") + "/models"
}

func supportedExtrasForTransport(transport TransportType) []string {
	switch transport {
	case TransportOpenAIResponses:
		return []string{"reasoning", "service_tier", "tools", "streaming"}
	case TransportAzureOpenAI:
		return []string{"reasoning", "tools", "streaming"}
	case TransportAnthropicMessages:
		return []string{"thinking", "tools", "streaming"}
	case TransportGoogleGemini, TransportGoogleVertex:
		return []string{"thinking", "tools", "streaming"}
	case TransportGitHubCopilot:
		return []string{"reasoning", "tools", "streaming"}
	default:
		return []string{"tools", "streaming"}
	}
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func sortProviderInfo(providers []domain.ProviderInfo) {
	sort.SliceStable(providers, func(i, j int) bool {
		if providers[i].BuiltIn != providers[j].BuiltIn {
			return providers[i].BuiltIn
		}
		return providers[i].Name < providers[j].Name
	})
}
