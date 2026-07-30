package app

import (
	"context"
	"strings"

	"aivo/core/domain"
)

type providerRegistryContextKey struct{}

func withProviderRegistry(ctx context.Context, registry *ProviderRegistry) context.Context {
	if registry == nil {
		return ctx
	}
	return context.WithValue(ctx, providerRegistryContextKey{}, registry)
}

func (s *Service) providerRegistryFromContext(ctx context.Context) *ProviderRegistry {
	if registry, ok := ctx.Value(providerRegistryContextKey{}).(*ProviderRegistry); ok && registry != nil {
		return registry
	}
	s.providersMu.RLock()
	defer s.providersMu.RUnlock()
	if s.providers != nil {
		return NewProviderRegistry(s.providers.Definitions(), nil)
	}
	return defaultProviderRegistry
}

func (s *Service) registerRuntimeProviderExtensions(projectPath string) {
	if s == nil {
		return
	}
	for id, extension := range loadEffectiveRuntimeConfig(projectPath).Config.ProviderExtensions {
		definition, ok := providerDefinitionFromRuntimeExtension(id, extension)
		if !ok {
			continue
		}
		_ = s.RegisterProviderDefinition(definition)
	}
}

func (s *Service) refreshProviderExtensions(projectPath string) {
	if s == nil {
		return
	}
	registry := s.providerRegistryForProject(projectPath)
	s.providersMu.Lock()
	s.providers = registry
	s.providersMu.Unlock()
}

func (s *Service) providerRegistryForProject(projectPath string) *ProviderRegistry {
	registry := NewDefaultProviderRegistry()
	for _, definition := range loadProviderEcosystemDefinitions(registry) {
		_ = registry.RegisterDefinition(definition)
	}
	s.providersMu.RLock()
	contributions := make([]ProviderDefinition, 0, len(s.providerContributions))
	for _, definition := range s.providerContributions {
		contributions = append(contributions, definition)
	}
	s.providersMu.RUnlock()
	for _, definition := range contributions {
		_ = registry.RegisterDefinition(definition)
	}
	for id, extension := range loadEffectiveRuntimeConfig(projectPath).Config.ProviderExtensions {
		if definition, ok := providerDefinitionFromRuntimeExtension(id, extension); ok {
			_ = registry.RegisterDefinition(definition)
		}
	}
	if s.pluginManager != nil {
		if plugins, err := s.pluginManager.List(context.Background(), domain.PluginListInput{IncludeDisabled: true}); err == nil {
			for _, plugin := range plugins {
				if !plugin.Plugin.Enabled {
					continue
				}
				for id, extension := range plugin.Plugin.Manifest.Providers {
					if definition, ok := providerDefinitionFromRuntimeExtension(id, extension); ok {
						_ = registry.RegisterDefinition(definition)
					}
				}
			}
		}
	}
	return registry
}

func providerDefinitionFromRuntimeExtension(id string, extension domain.ProviderExtensionDefinition) (ProviderDefinition, bool) {
	id = normalizeProviderKey(id)
	protocol := strings.TrimSpace(strings.ToLower(extension.Protocol))
	if id == "" || protocol == "" {
		return ProviderDefinition{}, false
	}
	definition := ProviderDefinition{
		ID: id, DisplayName: firstNonEmpty(strings.TrimSpace(extension.DisplayName), id), DefaultBaseURL: strings.TrimRight(strings.TrimSpace(extension.BaseURL), "/"),
		DefaultModelID: "default", ModelFetch: ModelFetchOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey, AuthNone},
		DefaultAuthType: AuthAPIKey, APIKeyEnvVars: nonEmptyStrings(extension.CredentialRef), BuiltIn: false,
		Command: strings.TrimSpace(extension.Command), Args: append([]string{}, extension.Args...),
	}
	switch protocol {
	case "openai", "openai-compatible", "openai_compatible":
		definition.Transport = TransportOpenAICompatible
	case "openai-chat", "openai_chat":
		definition.Transport = TransportOpenAIChat
	case "openai-responses", "openai_responses":
		definition.Transport = TransportOpenAIResponses
	case "anthropic", "anthropic_messages":
		definition.Transport = TransportAnthropicMessages
		definition.ModelFetch = ModelFetchAnthropic
	case "google", "gemini", "google_gemini":
		definition.Transport = TransportGoogleGemini
		definition.ModelFetch = ModelFetchGoogle
	case "command", "external_process":
		if definition.Command == "" {
			return ProviderDefinition{}, false
		}
		definition.Transport = TransportExternalProcess
		definition.ModelFetch = ModelFetchStatic
		definition.AuthTypes = []AuthType{AuthExternalProcess, AuthNone}
		definition.DefaultAuthType = AuthExternalProcess
	default:
		return ProviderDefinition{}, false
	}
	for _, modelID := range extension.Models {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		definition.Models = append(definition.Models, domain.ModelInfo{ID: modelID, ProviderID: id, Name: modelID})
	}
	if len(definition.Models) > 0 {
		definition.DefaultModelID = definition.Models[0].ID
		definition.Models[0].Recommended = true
	}
	return definition, true
}

func nonEmptyStrings(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return []string{strings.TrimSpace(value)}
}
