package app

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"aivo/core/domain"
)

func (s *Service) ResolveModelRoute(ctx context.Context, cfg domain.AppConfig, requestedModel *domain.ModelRef) (ResolvedModelRoute, error) {
	registry := s.providerRegistryFromContext(ctx)
	normalize := registry.Normalize
	provider, modelRef := resolveActiveProvider(cfg)
	if requestedModel != nil && strings.TrimSpace(requestedModel.ModelID) != "" {
		requestedProviderID := normalize(requestedModel.ProviderID)
		if requestedProviderID != "" && requestedProviderID != normalize(provider.ID) {
			provider = providerConfigForModelRequestWithRegistry(cfg, registry, requestedProviderID, strings.TrimSpace(requestedModel.ModelID))
			modelRef = domain.ModelRef{ProviderID: provider.ID, ModelID: provider.Model}
		} else {
			modelRef.ModelID = strings.TrimSpace(requestedModel.ModelID)
			if provider.ID == "" {
				provider.ID = normalizeProviderID(requestedModel.ProviderID)
			}
			modelRef.ProviderID = provider.ID
			provider.Model = modelRef.ModelID
		}
	}
	provider.ID = normalize(provider.ID)
	if provider.ID == "" {
		return ResolvedModelRoute{}, errors.New("provider is not configured")
	}
	def := providerDefinitionForConfigWithRegistry(registry, provider)
	provider.Type = firstNonEmpty(provider.Type, string(def.Transport))
	if provider.BaseURL == "" {
		provider.BaseURL = def.DefaultBaseURL
	}
	if provider.APIKeyEnv == "" && len(def.APIKeyEnvVars) > 0 {
		provider.APIKeyEnv = def.APIKeyEnvVars[0]
	}
	if provider.Model == "" {
		provider.Model = firstNonEmpty(modelRef.ModelID, def.DefaultModelID)
	}
	modelRef = domain.ModelRef{ProviderID: provider.ID, ModelID: normalizeModelIDForProvider(provider.ID, provider.Model)}
	if modelRef.ModelID == "" {
		return ResolvedModelRoute{}, errors.New("model is not configured")
	}
	transport := inferTransport(provider.ID, provider.Type, provider.BaseURL)
	credential, err := s.resolveCredentialWithDefinition(ctx, provider, def)
	if err != nil {
		return ResolvedModelRoute{}, err
	}
	return ResolvedModelRoute{Provider: provider, Model: modelRef, Definition: def, Transport: transport, BaseURL: provider.BaseURL, Credential: credential}, nil
}

func providerDefinitionForConfigWithRegistry(registry *ProviderRegistry, provider domain.ProviderConfig) ProviderDefinition {
	if registry != nil {
		if definition, ok := registry.Definition(provider.ID); ok {
			return definition
		}
	}
	return providerDefinitionForConfig(provider)
}

func providerConfigForModelRequestWithRegistry(cfg domain.AppConfig, registry *ProviderRegistry, providerID string, modelID string) domain.ProviderConfig {
	normalize := registry.Normalize
	if cfg.Provider != nil && normalize(cfg.Provider.ID) == normalize(providerID) {
		provider := *cfg.Provider
		provider.ID = normalize(provider.ID)
		if provider.Type == "" {
			if definition, ok := registry.Definition(provider.ID); ok {
				provider.Type = string(definition.Transport)
			} else {
				provider.Type = string(inferTransport(provider.ID, "", provider.BaseURL))
			}
		}
		provider.Model = modelID
		return provider
	}
	if definition, ok := registry.Definition(providerID); ok {
		apiKeyEnv := ""
		if len(definition.APIKeyEnvVars) > 0 {
			apiKeyEnv = definition.APIKeyEnvVars[0]
		}
		return domain.ProviderConfig{ID: definition.ID, Type: string(definition.Transport), BaseURL: definition.DefaultBaseURL, APIKeyEnv: apiKeyEnv, Model: modelID}
	}
	return domain.ProviderConfig{ID: normalize(providerID), Type: string(inferTransport(providerID, "", "")), BaseURL: defaultBaseURLFor(providerID), Model: modelID}
}

func inferTransport(providerID string, providerType string, baseURL string) TransportType {
	providerID = normalizeProviderID(providerID)
	providerType = strings.TrimSpace(strings.ToLower(providerType))
	if def, ok := providerDefinition(providerID); ok {
		if detected := inferTransportFromURL(baseURL); detected != "" && providerID == "custom-api" {
			return detected
		}
		return def.Transport
	}
	switch providerType {
	case string(TransportAzureOpenAI), "azure", "azure-openai":
		return TransportAzureOpenAI
	case string(TransportOpenAIResponses), "openai", "codex_responses":
		return TransportOpenAIResponses
	case string(TransportAnthropicMessages), "anthropic", "claude":
		return TransportAnthropicMessages
	case string(TransportGoogleGemini), "google", "gemini":
		return TransportGoogleGemini
	case string(TransportGoogleVertex), "google-vertex", "vertex", "vertex-ai":
		return TransportGoogleVertex
	case string(TransportBedrockConverse), "bedrock", "aws":
		return TransportBedrockConverse
	case string(TransportGitHubCopilot), "github-copilot", "copilot":
		return TransportGitHubCopilot
	case string(TransportOpenAIChat):
		return TransportOpenAIChat
	case string(TransportOpenAICompatible), "openai-compatible", "":
		if detected := inferTransportFromURL(baseURL); detected != "" {
			return detected
		}
		return TransportOpenAICompatible
	default:
		if detected := inferTransportFromURL(baseURL); detected != "" {
			return detected
		}
		return TransportOpenAICompatible
	}
}

func inferTransportFromURL(raw string) TransportType {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	host := strings.ToLower(parsed.Hostname())
	path := strings.ToLower(strings.TrimRight(parsed.Path, "/"))
	switch {
	case strings.HasSuffix(host, ".openai.azure.com") || strings.HasSuffix(host, ".services.ai.azure.com"):
		return TransportAzureOpenAI
	case host == "api.openai.com" || host == "api.x.ai":
		return TransportOpenAIResponses
	case host == "api.anthropic.com" || strings.HasSuffix(path, "/anthropic") || strings.HasSuffix(path, "/anthropic/v1"):
		return TransportAnthropicMessages
	case host == "api.kimi.com" && strings.Contains(path, "/coding"):
		return TransportAnthropicMessages
	case strings.Contains(host, "generativelanguage.googleapis.com"):
		return TransportGoogleGemini
	case strings.Contains(host, "aiplatform.googleapis.com") && strings.Contains(path, "/publishers/google"):
		return TransportGoogleVertex
	case strings.HasPrefix(host, "bedrock-runtime.") && strings.HasSuffix(host, ".amazonaws.com"):
		return TransportBedrockConverse
	case host == "api.githubcopilot.com" || host == "api.individual.githubcopilot.com" || host == "api.business.githubcopilot.com":
		return TransportGitHubCopilot
	default:
		return ""
	}
}
