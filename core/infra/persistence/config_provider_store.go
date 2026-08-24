package persistence

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aivo/core/domain"
)

func (s *Store) LoadConfig(ctx context.Context) (domain.AppConfig, error) {
	var row appConfigRow
	err := s.db.WithContext(ctx).Where("id = ?", 1).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return defaultAppConfig(), nil
	}
	if err != nil {
		return domain.AppConfig{}, err
	}
	cfg := defaultAppConfig()
	cfg.Initialized = row.Initialized == 1
	cfg.InitialWorkspacePath = row.InitialWorkspacePath
	cfg.ReasoningEffort = row.ReasoningEffort
	cfg.ServiceTier = row.ServiceTier
	cfg.AuxiliaryModel = decodeModelRef(row.AuxiliaryModel)
	cfg.FallbackModels = decodeModelRefs(row.FallbackModels)
	cfg.ProviderPolicy = decodeProviderRuntimePolicy(row.ProviderPolicy)
	cfg.WebSearch = decodeWebSearchConfig(row.WebSearch)
	cfg.NativeTools = decodeNativeToolsConfig(row.NativeTools)
	if row.ProviderID != "" {
		cfg.Provider = &domain.ProviderConfig{ID: row.ProviderID, Type: row.ProviderType, BaseURL: row.BaseURL, APIKeyEnv: row.APIKeyEnv, Model: row.Model, Headers: decodeHeaders(row.Headers), RequestParams: decodeAnyMap(row.RequestParams)}
		if row.Model != "" {
			cfg.DefaultModel = &domain.ModelRef{ProviderID: row.ProviderID, ModelID: row.Model}
		}
	}
	return cfg, nil
}

func defaultAppConfig() domain.AppConfig {
	return domain.AppConfig{
		Providers:       domain.ProviderSettings{Custom: map[string]domain.ProviderConfig{}},
		ProviderPolicy:  defaultProviderRuntimePolicy(),
		ReasoningEffort: "medium",
		ServiceTier:     "default",
		Persistence:     domain.PersistenceRolloutConfig{Configured: true, JournalEnabled: true, DualWriteValidation: true, ReadPath: "sqlite"},
		WebSearch:       defaultWebSearchConfig(),
		NativeTools:     domain.NativeToolsConfig{},
	}
}

func defaultWebSearchConfig() domain.WebSearchConfig {
	return domain.WebSearchConfig{
		Mode:          domain.WebSearchModeLive,
		Route:         domain.WebSearchRouteAuto,
		LocalProvider: "duckduckgo",
	}
}

func defaultProviderRuntimePolicy() domain.ProviderRuntimePolicy {
	enableFallback := true
	bufferStreamingFallback := true
	return domain.ProviderRuntimePolicy{
		EnableFallback:           &enableFallback,
		BufferStreamingFallback:  &bufferStreamingFallback,
		MaxRetries:               1,
		RetryBaseDelayMs:         100,
		RateLimitCooldownSeconds: 30,
	}
}

func (s *Store) SaveConfig(ctx context.Context, cfg domain.AppConfig) error {
	var providerID, providerType, baseURL, apiKeyEnv, model, headers, requestParams string
	if cfg.Provider != nil {
		providerID, providerType, baseURL, apiKeyEnv, model = cfg.Provider.ID, cfg.Provider.Type, cfg.Provider.BaseURL, cfg.Provider.APIKeyEnv, cfg.Provider.Model
		headers = encodeHeaders(cfg.Provider.Headers)
		requestParams = encodeAnyMap(cfg.Provider.RequestParams)
	}
	initialized := 0
	if cfg.Initialized {
		initialized = 1
	}
	row := appConfigRow{ID: 1, Initialized: initialized, InitialWorkspacePath: cfg.InitialWorkspacePath, ProviderID: providerID, ProviderType: providerType, BaseURL: baseURL, APIKeyEnv: apiKeyEnv, Model: model, AuxiliaryModel: encodeModelRef(cfg.AuxiliaryModel), ReasoningEffort: cfg.ReasoningEffort, ServiceTier: cfg.ServiceTier, FallbackModels: encodeModelRefs(cfg.FallbackModels), ProviderPolicy: encodeProviderRuntimePolicy(cfg.ProviderPolicy), WebSearch: encodeWebSearchConfig(cfg.WebSearch), NativeTools: encodeNativeToolsConfig(cfg.NativeTools), Headers: headers, RequestParams: requestParams, UpdatedAt: domain.NowString(time.Now())}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
}
