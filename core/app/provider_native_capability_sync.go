package app

import (
	"context"
	"strings"
	"time"

	"aivo/core/domain"
)

const dynamicProviderCapabilityCacheTTL = 24 * time.Hour

func (s *Service) ensureDynamicProviderCapabilities(ctx context.Context, route ResolvedModelRoute) {
	if s == nil || s.store == nil || !routeDeclaresDynamicCapabilities(route) {
		return
	}
	providerID := normalizeProviderID(firstNonEmpty(route.Provider.ID, route.Definition.ID))
	modelID := strings.TrimSpace(route.Model.ModelID)
	if providerID == "" || modelID == "" || s.cachedProviderCapabilityMetadataKnown(ctx, route) {
		return
	}

	s.providerCapabilitySyncMu.Lock()
	if s.providerCapabilitySynced == nil {
		s.providerCapabilitySynced = map[string]bool{}
	}
	key := dynamicCapabilitySyncKey(route)
	if s.providerCapabilitySynced[key] {
		s.providerCapabilitySyncMu.Unlock()
		return
	}
	s.providerCapabilitySynced[key] = true
	s.providerCapabilitySyncMu.Unlock()

	provider := route.Provider
	provider.ID = firstNonEmpty(provider.ID, route.Definition.ID)
	provider.Type = firstNonEmpty(provider.Type, string(route.Transport))
	provider.BaseURL = firstNonEmpty(provider.BaseURL, route.Definition.DefaultBaseURL)
	models, defaultModel, err := s.fetchProviderModels(ctx, provider)
	if err != nil {
		return
	}
	s.rememberRefreshedModels(ctx, provider, route.Definition.DisplayName, models, defaultModel)
}

func (s *Service) cachedProviderCapabilityMetadataKnown(ctx context.Context, route ResolvedModelRoute) bool {
	providerID := normalizeProviderID(firstNonEmpty(route.Provider.ID, route.Definition.ID))
	modelID := strings.TrimSpace(route.Model.ModelID)
	cache, err := s.store.LoadProviderModelCache(ctx, providerID)
	if err != nil || cache == nil {
		return false
	}
	model, ok := findModelInfo(cache.Models, modelID)
	if !ok || !routeCapabilityMetadataKnown(route, model) {
		return false
	}
	refreshedAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(cache.RefreshedAt))
	if err != nil {
		return false
	}
	age := s.now().Sub(refreshedAt)
	return age >= 0 && age < dynamicProviderCapabilityCacheTTL
}

func routeDeclaresDynamicCapabilities(route ResolvedModelRoute) bool {
	return isChatGPTCodexRoute(route) || modelFetchDeclaresCapabilities(route.Definition.ModelFetch)
}

func routeCapabilityMetadataKnown(route ResolvedModelRoute, model domain.ModelInfo) bool {
	if isChatGPTCodexRoute(route) {
		return containsString(model.DeclaredCapabilities, codexShellCapability) &&
			containsString(model.DeclaredCapabilities, codexRuntimeCapability) &&
			containsString(model.DeclaredCapabilities, codexWebSearchCapability)
	}
	return model.NativeToolsKnown || len(model.DeclaredCapabilities) > 0
}

func dynamicCapabilitySyncKey(route ResolvedModelRoute) string {
	providerID := normalizeProviderID(firstNonEmpty(route.Provider.ID, route.Definition.ID))
	if isChatGPTCodexRoute(route) {
		return providerID + ":chatgpt-codex"
	}
	return providerID
}

func isChatGPTCodexRoute(route ResolvedModelRoute) bool {
	return route.Definition.BuiltIn && normalizeProviderID(route.Definition.ID) == "openai" &&
		route.Transport == TransportOpenAIResponses && isOAuthCredential(route.Credential)
}

func modelFetchDeclaresCapabilities(strategy ModelFetchStrategy) bool {
	switch strategy {
	case ModelFetchAnthropic, ModelFetchMistral, ModelFetchOpenRouter, ModelFetchCerebras:
		return true
	default:
		return false
	}
}

func dynamicallyDiscoveredNativeTools(model domain.ModelInfo) map[string]bool {
	if !model.NativeToolsKnown {
		return nil
	}
	out := map[string]bool{}
	for _, name := range model.NativeTools {
		name = strings.TrimSpace(name)
		if name != "" {
			out[name] = true
		}
	}
	return out
}
