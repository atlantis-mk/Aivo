package app

import (
	"context"
	"errors"
	"strings"

	"aivo/core/domain"
)

func (s *Service) ConnectProvider(ctx context.Context, input domain.ProviderConnectInput) (domain.CatalogState, error) {
	cfg, _, err := s.providerConfigFromInput(input)
	if err != nil {
		return domain.CatalogState{}, err
	}
	providerID := cfg.ID
	appCfg, err := s.AppConfig(ctx)
	if err != nil {
		return domain.CatalogState{}, err
	}
	appCfg.Provider = &cfg
	appCfg.DefaultModel = &domain.ModelRef{ProviderID: cfg.ID, ModelID: cfg.Model}
	if err := s.store.SaveProvider(ctx, cfg); err != nil {
		return domain.CatalogState{}, err
	}
	method := strings.TrimSpace(input.Method)
	if method != "" && method != "env" {
		existingAuth, _ := s.store.LoadProviderAuth(ctx, providerID)
		shouldSaveAuth := !isOAuthMethod(method) || existingAuth == nil
		if shouldSaveAuth {
			if err := s.saveProviderAuth(ctx, domain.ProviderAuthRecord{
				ProviderID: providerID,
				Method:     method,
				AccountID:  connectAccountLabel(providerID, method, strings.TrimSpace(input.APIKey), strings.TrimSpace(input.APIKeyEnv)),
				APIKey:     strings.TrimSpace(input.APIKey),
				UpdatedAt:  domain.NowString(s.now()),
			}); err != nil {
				return domain.CatalogState{}, err
			}
		}
	}
	if err := s.store.SaveConfig(ctx, appCfg); err != nil {
		return domain.CatalogState{}, err
	}
	return s.Catalog(ctx)
}

func (s *Service) SaveProvider(ctx context.Context, input domain.ProviderConnectInput) (domain.CatalogState, error) {
	cfg, _, err := s.providerConfigFromInput(input)
	if err != nil {
		return domain.CatalogState{}, err
	}
	if err := s.store.SaveProvider(ctx, cfg); err != nil {
		return domain.CatalogState{}, err
	}
	method := strings.TrimSpace(input.Method)
	if method != "" && method != "env" && strings.TrimSpace(input.APIKey) != "" {
		if err := s.saveProviderAuth(ctx, domain.ProviderAuthRecord{
			ProviderID: cfg.ID,
			Method:     method,
			AccountID:  connectAccountLabel(cfg.ID, method, strings.TrimSpace(input.APIKey), strings.TrimSpace(input.APIKeyEnv)),
			APIKey:     strings.TrimSpace(input.APIKey),
			UpdatedAt:  domain.NowString(s.now()),
		}); err != nil {
			return domain.CatalogState{}, err
		}
	}
	return s.Catalog(ctx)
}

func (s *Service) DeleteProvider(ctx context.Context, providerID string) (domain.CatalogState, error) {
	providerID = s.normalizeProviderID(providerID)
	if providerID == "" {
		return domain.CatalogState{}, errors.New("provider is required")
	}
	auths, _ := s.store.ListProviderAuths(ctx, providerID)
	for _, auth := range auths {
		next := auth
		_ = s.deleteProviderAuthSecrets(ctx, &next)
	}
	if err := s.store.DeleteProvider(ctx, providerID); err != nil {
		return domain.CatalogState{}, err
	}
	cfg, err := s.AppConfig(ctx)
	if err == nil {
		changed := false
		if cfg.Provider != nil && s.normalizeProviderID(cfg.Provider.ID) == providerID {
			cfg.Provider = nil
			changed = true
		}
		if cfg.DefaultModel != nil && s.normalizeProviderID(cfg.DefaultModel.ProviderID) == providerID {
			cfg.DefaultModel = nil
			changed = true
		}
		if cfg.AuxiliaryModel != nil && s.normalizeProviderID(cfg.AuxiliaryModel.ProviderID) == providerID {
			cfg.AuxiliaryModel = nil
			changed = true
		}
		nextFallbacks := cfg.FallbackModels[:0]
		for _, model := range cfg.FallbackModels {
			if s.normalizeProviderID(model.ProviderID) == providerID {
				changed = true
				continue
			}
			nextFallbacks = append(nextFallbacks, model)
		}
		cfg.FallbackModels = nextFallbacks
		if changed {
			_ = s.store.SaveConfig(ctx, cfg)
		}
	}
	return s.Catalog(ctx)
}

func providerReadiness(provider domain.ProviderInfo, def ProviderDefinition, authRecord *domain.ProviderAuthRecord) *domain.ProviderReadiness {
	ready := provider.Connected
	source := provider.ConnectionSource
	authMode := ""
	if authRecord != nil {
		authMode = authRecord.Method
		source = authRecord.Method
	}
	if source == "" && provider.Environment != "" {
		source = "env"
	}
	reason := ""
	if !ready {
		switch def.DefaultAuthType {
		case AuthAWSSDK:
			if lookupEnv("AWS_ACCESS_KEY_ID") != "" && lookupEnv("AWS_SECRET_ACCESS_KEY") != "" {
				ready = true
				authMode = "aws-sdk"
				source = "aws-sdk"
			} else {
				reason = "AWS credentials are not configured"
			}
		case AuthNone:
			ready = true
			authMode = "none"
			source = "none"
		default:
			reason = "credentials are not configured"
		}
	}
	return &domain.ProviderReadiness{Ready: ready, AuthMode: authMode, Source: source, Environment: provider.Environment, Reason: reason}
}

func (s *Service) RefreshProviderModels(ctx context.Context, input domain.ProviderConnectInput) (domain.CatalogState, error) {
	providerInput := input
	if providerRefreshUsesPersistedConfig(input) {
		providerInput = s.persistedProviderRefreshInput(ctx, input)
	}
	provider, _, err := s.providerConfigFromInput(providerInput)
	if err != nil {
		return domain.CatalogState{}, err
	}
	models, defaultModel, err := s.fetchProviderModels(ctx, provider)
	if err != nil {
		return domain.CatalogState{}, err
	}
	s.rememberRefreshedModels(ctx, provider, input.Name, models, defaultModel)
	return s.Catalog(ctx)
}

func providerRefreshUsesPersistedConfig(input domain.ProviderConnectInput) bool {
	return strings.TrimSpace(input.ProviderID) != "" &&
		strings.TrimSpace(input.Type) == "" &&
		strings.TrimSpace(input.BaseURL) == "" &&
		strings.TrimSpace(input.APIKey) == "" &&
		strings.TrimSpace(input.APIKeyEnv) == "" &&
		strings.TrimSpace(input.ModelID) == "" &&
		strings.TrimSpace(input.Method) == "" &&
		len(input.Headers) == 0 &&
		len(input.RequestParams) == 0
}

func (s *Service) persistedProviderRefreshInput(ctx context.Context, input domain.ProviderConnectInput) domain.ProviderConnectInput {
	providerID := s.normalizeProviderID(input.ProviderID)
	providers, err := s.store.ListProviders(ctx)
	if err != nil {
		return input
	}
	for _, saved := range providers {
		if s.normalizeProviderID(saved.ID) != providerID {
			continue
		}
		input.ProviderID = saved.ID
		input.Type = saved.Type
		input.BaseURL = saved.BaseURL
		input.APIKeyEnv = saved.APIKeyEnv
		input.ModelID = saved.Model
		input.Headers = cloneStringMap(saved.Headers)
		input.RequestParams = cloneAnyMap(saved.RequestParams)
		return input
	}
	return input
}

func (s *Service) ListProviderCallEvents(ctx context.Context, input domain.ProviderCallEventsInput) ([]domain.ProviderCallEvent, error) {
	providerID := s.normalizeProviderID(input.ProviderID)
	return s.store.ListProviderCallEvents(ctx, providerID, input.Limit)
}

func (s *Service) GetProviderUsage(ctx context.Context, input domain.ProviderUsageInput) (domain.ProviderUsageStats, error) {
	providerID := s.normalizeProviderID(input.ProviderID)
	limit := input.Limit
	if limit <= 0 {
		limit = 1000
	}
	events, err := s.store.ListProviderCallEvents(ctx, providerID, limit)
	if err != nil {
		return domain.ProviderUsageStats{}, err
	}
	stats := domain.ProviderUsageStats{ProviderID: providerID}
	for _, event := range events {
		stats.CallCount++
		stats.InputTokens += event.InputTokens
		stats.OutputTokens += event.OutputTokens
		stats.TotalTokens += event.TotalTokens
		stats.CostMicros += event.CostMicros
		stats.Estimated = stats.Estimated || event.Estimated
		if stats.LastCallAt == "" || event.CreatedAt > stats.LastCallAt {
			stats.LastCallAt = event.CreatedAt
		}
		if event.Status == "success" {
			stats.SuccessCount++
			continue
		}
		stats.FailureCount++
		if event.ErrorClass != "" && (stats.LastErrorClass == "" || event.CreatedAt >= stats.LastCallAt) {
			stats.LastErrorClass = event.ErrorClass
			stats.LastErrorMessage = event.ErrorMessage
			stats.LastErrorProvider = event.ProviderID
		}
	}
	return stats, nil
}

func (s *Service) ValidateProvider(ctx context.Context, input domain.ProviderConnectInput) (domain.ProviderValidationResult, error) {
	provider, def, err := s.providerConfigFromInput(input)
	if err != nil {
		return domain.ProviderValidationResult{}, err
	}
	transport := inferTransport(provider.ID, provider.Type, provider.BaseURL)
	now := domain.NowString(s.now())
	result := domain.ProviderValidationResult{
		ProviderID: provider.ID,
		Status:     "checking",
		Transport:  string(transport),
		BaseURL:    provider.BaseURL,
		CheckedAt:  now,
	}
	credential, err := s.resolveCredentialWithDefinition(ctx, provider, def)
	if err != nil {
		result.Status = "failed"
		result.Error = safeProviderError(err)
		_ = s.store.SaveProviderValidation(ctx, result)
		return result, nil
	}
	result.AuthMode = credential.Method
	result.Source = credential.Method
	result.Environment = provider.APIKeyEnv
	if def.ModelFetch == ModelFetchStatic {
		models := append([]domain.ModelInfo(nil), def.Models...)
		result.Ready = true
		result.Status = "ready"
		result.DefaultModel = firstNonEmpty(provider.Model, def.DefaultModelID)
		result.ModelCount = len(models)
		result.Models = models
		_ = s.store.SaveProviderValidation(ctx, result)
		return result, nil
	}
	models, defaultModel, err := s.fetchProviderModels(ctx, provider)
	if err != nil {
		if cache, cacheErr := s.store.LoadProviderModelCache(ctx, provider.ID); cacheErr == nil && cache != nil && len(cache.Models) > 0 {
			result.Ready = true
			result.Status = "stale-cache"
			result.DefaultModel = cache.DefaultModel
			result.ModelCount = len(cache.Models)
			result.Models = cache.Models
			result.Error = safeProviderError(err)
			_ = s.store.SaveProviderValidation(ctx, result)
			return result, nil
		}
		result.Status = "failed"
		result.Error = safeProviderError(err)
		_ = s.store.SaveProviderValidation(ctx, result)
		return result, nil
	}
	result.Ready = true
	result.Status = "ready"
	result.DefaultModel = defaultModel
	result.ModelCount = len(models)
	result.Models = models
	s.rememberRefreshedModels(ctx, provider, input.Name, models, defaultModel)
	_ = s.store.SaveProviderValidation(ctx, result)
	return result, nil
}

func (s *Service) DeleteProviderAccount(ctx context.Context, accountID string) (domain.CatalogState, error) {
	if strings.TrimSpace(accountID) == "" {
		return domain.CatalogState{}, errors.New("account id is required")
	}
	if auth, err := s.store.GetProviderAuth(ctx, accountID); err == nil {
		_ = s.deleteProviderAuthSecrets(ctx, auth)
	}
	if err := s.store.DeleteProviderAuth(ctx, accountID); err != nil {
		return domain.CatalogState{}, err
	}
	return s.Catalog(ctx)
}

func (s *Service) CompleteInitialization(ctx context.Context, input domain.CompleteInitializationInput) (domain.AppConfig, error) {
	appName, err := domain.ResolveInitializationAppName(input.AppName)
	if err != nil {
		return domain.AppConfig{}, err
	}
	requestedWorkspacePath := strings.TrimSpace(input.InitialWorkspacePath)
	if requestedWorkspacePath == "" {
		var err error
		requestedWorkspacePath, err = managedWorkspaceRoot()
		if err != nil {
			return domain.AppConfig{}, err
		}
	}
	workspacePath, err := ensureInitialWorkspaceDirectory(requestedWorkspacePath)
	if err != nil {
		return domain.AppConfig{}, err
	}
	cfg, err := s.AppConfig(ctx)
	if err != nil {
		return domain.AppConfig{}, err
	}
	cfg.Initialized = true
	cfg.AppName = appName
	cfg.InitialWorkspacePath = workspacePath
	if input.Provider != nil {
		cfg.Provider = input.Provider
		if input.Provider.Model != "" {
			cfg.DefaultModel = &domain.ModelRef{ProviderID: input.Provider.ID, ModelID: input.Provider.Model}
		}
	}
	if err := s.store.SaveConfig(ctx, cfg); err != nil {
		return domain.AppConfig{}, err
	}
	return s.AppConfig(ctx)
}
