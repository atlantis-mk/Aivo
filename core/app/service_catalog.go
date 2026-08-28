package app

import (
	"context"

	"aivo/core/domain"
)

func (s *Service) Catalog(ctx context.Context) (domain.CatalogState, error) {
	cfg, err := s.AppConfig(ctx)
	if err != nil {
		return domain.CatalogState{}, err
	}
	providers := s.applyRefreshedProviderModels(s.defaultProviders())
	if savedProviders, err := s.store.ListProviders(ctx); err == nil {
		providers = s.mergeSavedProviders(providers, savedProviders)
	}
	if caches, err := s.store.ListProviderModelCaches(ctx); err == nil {
		providers = s.mergeProviderModelCaches(providers, caches)
	}
	if health, err := s.store.ListProviderHealth(ctx); err == nil {
		providers = mergeProviderHealth(providers, health)
	}
	var connected []string
	var connectedProviders []domain.ProviderInfo
	for i := range providers {
		def := s.providerDefinitionForConfig(domain.ProviderConfig{ID: providers[i].ID, Type: providers[i].Type, BaseURL: providers[i].BaseURL, APIKeyEnv: providers[i].Environment, Model: providers[i].DefaultModelID})
		authRecord, _ := s.store.LoadProviderAuth(ctx, providers[i].ID)
		authRecords, _ := s.store.ListProviderAuths(ctx, providers[i].ID)
		providers[i].Accounts = providerAccountsFromAuth(authRecords)
		if (cfg.Provider != nil && providers[i].ID == cfg.Provider.ID) || len(authRecords) > 0 || authRecord != nil {
			providers[i].Connected = true
			source := "credential-reference"
			if authRecord != nil {
				source = authRecord.Method
			}
			providers[i].ConnectionSource = source
			providers[i].Auth = &domain.AuthInfo{
				Type:      source,
				Connected: true,
				Source:    source,
			}
			if providers[i].Environment != "" {
				providers[i].Auth.Environment = providers[i].Environment
			}
			if authRecord != nil {
				providers[i].Auth.LastValidatedAt = authRecord.UpdatedAt
				providers[i].Auth.ConnectedAt = authRecord.UpdatedAt
			}
			connected = append(connected, providers[i].ID)
			connectedProviders = append(connectedProviders, providers[i])
		}
		if authRecord != nil && isOAuthMethod(authRecord.Method) {
			route := ResolvedModelRoute{
				Provider:   domain.ProviderConfig{ID: providers[i].ID, Type: providers[i].Type, BaseURL: providers[i].BaseURL, Model: providers[i].DefaultModelID},
				Definition: def,
				Transport:  def.Transport,
				BaseURL:    firstNonEmpty(providers[i].BaseURL, def.DefaultBaseURL),
				Credential: llmCredential{Method: authRecord.Method, AccessToken: authRecord.AccessToken, AccountID: authRecord.AccountID, AuthRecord: authRecord},
			}
			capabilities := capabilitiesForProviderAccount(route)
			if capabilities.NamespaceTools || capabilities.ImageGeneration || capabilities.WebSearch {
				providers[i].NativeCapabilities = &domain.ProviderNativeCapabilities{
					NamespaceTools: capabilities.NamespaceTools, ImageGeneration: capabilities.ImageGeneration,
					WebSearch: capabilities.WebSearch, Source: "provider-account",
				}
			}
		}
		if providers[i].Readiness == nil {
			providers[i].Readiness = providerReadiness(providers[i], def, authRecord)
		}
		if providers[i].ModelRefresh != nil {
			providers[i].ModelRefresh.ModelCount = len(providers[i].Models)
		}
	}
	sortProviderInfo(providers)
	return domain.CatalogState{
		Providers:          providers,
		Models:             flattenModels(providers),
		Connected:          connected,
		DefaultModel:       cfg.DefaultModel,
		ConnectedProviders: connectedProviders,
		PopularProviders:   providers,
	}, nil
}

// CatalogForProject applies project-scoped provider extensions before building
// the catalog used by that workspace.
func (s *Service) CatalogForProject(ctx context.Context, projectPath string) (domain.CatalogState, error) {
	registry := s.providerRegistryForProject(projectPath)
	s.modelRefreshMu.Lock()
	models := make(map[string][]domain.ModelInfo, len(s.refreshedModels))
	for key, value := range s.refreshedModels {
		models[key] = append([]domain.ModelInfo{}, value...)
	}
	defaults := cloneStringMap(s.refreshedDefault)
	infos := make(map[string]domain.ProviderInfo, len(s.refreshedInfo))
	for key, value := range s.refreshedInfo {
		infos[key] = value
	}
	s.modelRefreshMu.Unlock()
	scoped := &Service{
		store: s.store, now: s.now, providers: registry,
		refreshedModels: models, refreshedDefault: defaults, refreshedInfo: infos,
	}
	return scoped.Catalog(ctx)
}

func mergeProviderHealth(providers []domain.ProviderInfo, health []domain.ProviderHealth) []domain.ProviderInfo {
	if len(health) == 0 {
		return providers
	}
	index := map[string]int{}
	for i := range providers {
		index[providers[i].ID] = i
	}
	for _, item := range health {
		providerID := normalizeProviderID(item.ProviderID)
		if providerID == "" {
			continue
		}
		item.ProviderID = providerID
		if i, ok := index[providerID]; ok {
			providers[i].Health = &item
		}
	}
	return providers
}

func (s *Service) mergeSavedProviders(providers []domain.ProviderInfo, saved []domain.ProviderConfig) []domain.ProviderInfo {
	index := map[string]int{}
	for i := range providers {
		index[providers[i].ID] = i
	}
	for _, cfg := range saved {
		cfg.ID = normalizeProviderID(cfg.ID)
		if cfg.ID == "" {
			continue
		}
		def := s.providerDefinitionForConfig(cfg)
		info := providerInfoFromDefinition(def)
		info.ID = cfg.ID
		info.Type = firstNonEmpty(cfg.Type, info.Type)
		info.BaseURL = firstNonEmpty(cfg.BaseURL, info.BaseURL)
		info.Environment = firstNonEmpty(cfg.APIKeyEnv, info.Environment)
		info.DefaultModelID = firstNonEmpty(cfg.Model, info.DefaultModelID)
		info.Custom = !info.BuiltIn || !s.isBuiltInProvider(cfg.ID)
		if info.DefaultModelID != "" && !modelListContains(info.Models, info.DefaultModelID) {
			info.Models = append([]domain.ModelInfo{{ID: info.DefaultModelID, ProviderID: cfg.ID, Name: info.DefaultModelID, Recommended: true}}, info.Models...)
		}
		if i, ok := index[cfg.ID]; ok {
			providers[i].BaseURL = info.BaseURL
			providers[i].Environment = info.Environment
			providers[i].DefaultModelID = info.DefaultModelID
			providers[i].Type = info.Type
			if info.DefaultModelID != "" {
				markRecommended(providers[i].Models, info.DefaultModelID)
			}
			continue
		}
		index[cfg.ID] = len(providers)
		providers = append(providers, info)
	}
	return providers
}

func (s *Service) mergeProviderModelCaches(providers []domain.ProviderInfo, caches []domain.ProviderModelCache) []domain.ProviderInfo {
	index := map[string]int{}
	for i := range providers {
		index[providers[i].ID] = i
	}
	for _, cache := range caches {
		providerID := normalizeProviderID(cache.ProviderID)
		if providerID == "" || len(cache.Models) == 0 {
			continue
		}
		models := append([]domain.ModelInfo(nil), cache.Models...)
		defaultModel := cache.DefaultModel
		if defaultModel == "" {
			defaultModel = models[0].ID
		}
		markRecommended(models, defaultModel)
		if i, ok := index[providerID]; ok {
			providers[i].Models = models
			providers[i].DefaultModelID = defaultModel
			if providers[i].ModelRefresh == nil {
				providers[i].ModelRefresh = &domain.ProviderModelRefresh{}
			}
			providers[i].ModelRefresh.Status = firstNonEmpty(cache.Status, "ready")
			providers[i].ModelRefresh.LastRefresh = cache.RefreshedAt
			providers[i].ModelRefresh.Error = cache.Error
			providers[i].ModelRefresh.ModelCount = len(models)
			providers[i].ModelRefresh.CacheSource = firstNonEmpty(cache.CacheSource, "sqlite")
			providers[i].ModelRefresh.ParserType = cache.ParserType
			providers[i].ModelRefresh.Endpoint = cache.Endpoint
			providers[i].ModelRefresh.Stale = cache.Status == "stale"
			continue
		}
		def := s.providerDefinitionForConfig(domain.ProviderConfig{ID: providerID, Type: "", Model: defaultModel})
		info := providerInfoFromDefinition(def)
		info.ID = providerID
		info.Models = models
		info.DefaultModelID = defaultModel
		info.Custom = !s.isBuiltInProvider(providerID)
		if info.ModelRefresh == nil {
			info.ModelRefresh = &domain.ProviderModelRefresh{}
		}
		info.ModelRefresh.Status = firstNonEmpty(cache.Status, "ready")
		info.ModelRefresh.LastRefresh = cache.RefreshedAt
		info.ModelRefresh.ModelCount = len(models)
		info.ModelRefresh.CacheSource = firstNonEmpty(cache.CacheSource, "sqlite")
		info.ModelRefresh.ParserType = cache.ParserType
		info.ModelRefresh.Endpoint = cache.Endpoint
		index[providerID] = len(providers)
		providers = append(providers, info)
	}
	return providers
}

func (s *Service) isBuiltInProvider(providerID string) bool {
	def, ok := s.providerDefinition(providerID)
	return ok && def.BuiltIn
}

func modelListContains(models []domain.ModelInfo, modelID string) bool {
	for _, item := range models {
		if item.ID == modelID {
			return true
		}
	}
	return false
}

func flattenModels(providers []domain.ProviderInfo) []domain.ModelInfo {
	return flattenProviderModels(providers)
}
