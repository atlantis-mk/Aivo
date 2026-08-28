package persistence

import (
	"context"
	"errors"
	"strings"
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
	if strings.TrimSpace(row.AppName) != "" {
		cfg.AppName, err = domain.NormalizeAppName(row.AppName)
		if err != nil {
			return domain.AppConfig{}, err
		}
	}
	cfg.InitialWorkspacePath = row.InitialWorkspacePath
	cfg.ReasoningEffort = row.ReasoningEffort
	cfg.ServiceTier = row.ServiceTier
	cfg.DefaultPermissionMode = normalizeStoredDefaultPermissionMode(row.DefaultPermissionMode)
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
		AppName:               domain.DefaultAppName,
		Providers:             domain.ProviderSettings{Custom: map[string]domain.ProviderConfig{}},
		ProviderPolicy:        defaultProviderRuntimePolicy(),
		ReasoningEffort:       "medium",
		ServiceTier:           "default",
		DefaultPermissionMode: domain.PermissionModeRequestApproval,
		Persistence:           domain.PersistenceRolloutConfig{Configured: true, JournalEnabled: true, DualWriteValidation: true, ReadPath: "sqlite"},
		WebSearch:             defaultWebSearchConfig(),
		NativeTools:           domain.NativeToolsConfig{},
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
	appName := strings.TrimSpace(cfg.AppName)
	if appName == "" {
		appName = domain.DefaultAppName
	}
	var err error
	appName, err = domain.NormalizeAppName(appName)
	if err != nil {
		return err
	}
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
	row := appConfigRow{ID: 1, Initialized: initialized, AppName: appName, InitialWorkspacePath: cfg.InitialWorkspacePath, ProviderID: providerID, ProviderType: providerType, BaseURL: baseURL, APIKeyEnv: apiKeyEnv, Model: model, AuxiliaryModel: encodeModelRef(cfg.AuxiliaryModel), ReasoningEffort: cfg.ReasoningEffort, ServiceTier: cfg.ServiceTier, DefaultPermissionMode: normalizeStoredDefaultPermissionMode(cfg.DefaultPermissionMode), FallbackModels: encodeModelRefs(cfg.FallbackModels), ProviderPolicy: encodeProviderRuntimePolicy(cfg.ProviderPolicy), WebSearch: encodeWebSearchConfig(cfg.WebSearch), NativeTools: encodeNativeToolsConfig(cfg.NativeTools), Headers: headers, RequestParams: requestParams, UpdatedAt: domain.NowString(time.Now())}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
}

func normalizeStoredDefaultPermissionMode(mode string) string {
	if strings.TrimSpace(mode) == domain.PermissionModeFullAccess {
		return domain.PermissionModeFullAccess
	}
	return domain.PermissionModeRequestApproval
}
