package app

import (
	"context"
	"strings"

	"aivo/core/domain"
)

func (s *Service) rememberRefreshedModels(ctx context.Context, provider domain.ProviderConfig, name string, models []domain.ModelInfo, defaultModel string) {
	s.modelRefreshMu.Lock()
	defer s.modelRefreshMu.Unlock()
	providerID := provider.ID
	copied := append([]domain.ModelInfo(nil), models...)
	s.refreshedModels[providerID] = copied
	s.refreshedDefault[providerID] = defaultModel
	s.refreshedInfo[providerID] = domain.ProviderInfo{
		ID:             providerID,
		Name:           firstNonEmpty(strings.TrimSpace(name), providerID),
		Type:           provider.Type,
		BaseURL:        provider.BaseURL,
		BuiltIn:        false,
		Custom:         true,
		Environment:    provider.APIKeyEnv,
		DefaultModelID: defaultModel,
		Models:         copied,
		AuthMethods:    providerAuthMethods(providerID, provider.APIKeyEnv),
	}
	def := s.providerDefinitionForConfig(provider)
	strategy := def.ModelFetch
	if catalogContainsCodexDeclarations(copied) {
		strategy = ModelFetchOpenAICodexAccount
	}
	_ = s.store.SaveProviderModelCache(ctx, domain.ProviderModelCache{
		ProviderID: providerID, Models: copied, DefaultModel: defaultModel, Strategy: string(strategy),
		ParserType: parserTypeForModelFetch(strategy), Endpoint: modelEndpointForFetchStrategy(def, strategy), CacheSource: "remote",
		Status: "ready", RefreshedAt: domain.NowString(s.now()), UpdatedAt: domain.NowString(s.now()),
	})
}

func catalogContainsCodexDeclarations(models []domain.ModelInfo) bool {
	for _, model := range models {
		if containsString(model.DeclaredCapabilities, codexRuntimeCapability) || containsString(model.DeclaredCapabilities, codexShellCapability) {
			return true
		}
	}
	return false
}

func (s *Service) applyRefreshedProviderModels(providers []domain.ProviderInfo) []domain.ProviderInfo {
	s.modelRefreshMu.Lock()
	defer s.modelRefreshMu.Unlock()
	seen := map[string]bool{}
	for i := range providers {
		seen[providers[i].ID] = true
		models := s.refreshedModels[providers[i].ID]
		if len(models) == 0 {
			continue
		}
		providers[i].Models = append([]domain.ModelInfo(nil), models...)
		if defaultModel := s.refreshedDefault[providers[i].ID]; defaultModel != "" {
			providers[i].DefaultModelID = defaultModel
		}
	}
	for providerID, provider := range s.refreshedInfo {
		if seen[providerID] {
			continue
		}
		providers = append(providers, provider)
	}
	return providers
}

func (s *Service) defaultModelFor(providerID string) string {
	s.modelRefreshMu.Lock()
	defer s.modelRefreshMu.Unlock()
	if model := s.refreshedDefault[providerID]; model != "" {
		return model
	}
	return defaultModelFor(providerID)
}
