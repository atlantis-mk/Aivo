package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	sqlite "github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aivo/core/domain"
)

type Store struct {
	db    *gorm.DB
	sqlDB *sql.DB
}

func OpenDefault() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".aivo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return Open(filepath.Join(dir, "aivo.db"))
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	store := &Store{db: db, sqlDB: sqlDB}
	if err := store.migrate(context.Background()); err != nil {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.sqlDB == nil {
		return nil
	}
	return s.sqlDB.Close()
}

func (s *Store) migrate(ctx context.Context) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.AutoMigrate(&schemaVersionRow{}, &appConfigRow{}, &providerRow{}, &providerModelCacheRow{}, &providerValidationRow{}, &providerHealthRow{}, &providerCallEventRow{}, &projectRow{}, &sessionRow{}); err != nil {
			return err
		}
		if err := migrateProviderAuth(ctx, tx); err != nil {
			return err
		}
		if err := tx.AutoMigrate(&turnRow{}, &sessionEventRow{}, &toolCallRow{}, &sessionExecutionStateRow{}, &pendingSessionInputRow{}, &permissionRequestRow{}, &questionRequestRow{}, &permissionRuleRow{}, &sessionSummaryRow{}, &sessionCheckpointRow{}, &codingContextRow{}, &agentRunRow{}, &todoItemRow{}, &scheduledJobRow{}, &pluginInstallRow{}, &pluginDiagnosticRow{}, &mcpServerRow{}, &mcpToolRow{}, &mcpPromptRow{}, &mcpResourceRow{}, &toolRegistrationRow{}); err != nil {
			return err
		}
		if err := migrateLegacyMessages(ctx, tx); err != nil {
			return err
		}
		now := domain.NowString(time.Now())
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&schemaVersionRow{Version: 1, AppliedAt: now}).Error
	})
}

type legacyMessageRow struct {
	ID          string `gorm:"column:id"`
	SessionID   string `gorm:"column:session_id"`
	Role        string `gorm:"column:role"`
	Text        string `gorm:"column:text"`
	TimeCreated string `gorm:"column:time_created"`
}

func migrateLegacyMessages(ctx context.Context, tx *gorm.DB) error {
	migrator := tx.WithContext(ctx).Migrator()
	if !migrator.HasTable("messages") {
		return nil
	}
	var rows []legacyMessageRow
	if err := tx.WithContext(ctx).Table("messages").Order("time_created ASC").Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		role := strings.TrimSpace(row.Role)
		eventType := ""
		switch role {
		case domain.EventRoleUser:
			eventType = domain.EventTypeUserMessage
		case domain.EventRoleAssistant:
			eventType = domain.EventTypeAssistantMessage
		default:
			continue
		}
		eventID := strings.TrimSpace(row.ID)
		if eventID == "" {
			eventID = uuid.NewString()
		}
		event := sessionEventRow{
			ID:          eventID,
			SessionID:   strings.TrimSpace(row.SessionID),
			Type:        eventType,
			Role:        role,
			Visibility:  domain.EventVisibilityNormal,
			Content:     row.Text,
			TimeCreated: row.TimeCreated,
		}
		if event.SessionID == "" || event.TimeCreated == "" {
			continue
		}
		if err := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&event).Error; err != nil {
			return err
		}
	}
	return migrator.DropTable("messages")
}

func migrateProviderAuth(ctx context.Context, tx *gorm.DB) error {
	migrator := tx.WithContext(ctx).Migrator()
	if migrator.HasTable(&providerAuthRow{}) && !migrator.HasColumn(&providerAuthRow{}, "id") {
		if err := migrator.RenameTable("provider_auth", "provider_auth_old"); err != nil {
			return err
		}
		if err := tx.WithContext(ctx).AutoMigrate(&providerAuthRow{}); err != nil {
			return err
		}
		var legacy []legacyProviderAuthRow
		if err := tx.WithContext(ctx).Table("provider_auth_old").Find(&legacy).Error; err != nil {
			return err
		}
		records := make([]providerAuthRow, 0, len(legacy))
		for _, item := range legacy {
			records = append(records, providerAuthRow{
				ID:           item.ProviderID + ":" + item.Method + ":" + item.UpdatedAt,
				ProviderID:   item.ProviderID,
				Method:       item.Method,
				AccessToken:  item.AccessToken,
				RefreshToken: item.RefreshToken,
				ExpiresAt:    item.ExpiresAt,
				AccountID:    item.AccountID,
				DisplayName:  "",
				APIKey:       item.APIKey,
				UpdatedAt:    item.UpdatedAt,
			})
		}
		if len(records) > 0 {
			if err := tx.WithContext(ctx).Create(&records).Error; err != nil {
				return err
			}
		}
		return migrator.DropTable("provider_auth_old")
	}
	return tx.WithContext(ctx).AutoMigrate(&providerAuthRow{})
}

func (s *Store) hasColumn(ctx context.Context, table string, column string) (bool, error) {
	return s.db.WithContext(ctx).Migrator().HasColumn(table, column), nil
}

func (s *Store) SaveProviderAuth(ctx context.Context, auth domain.ProviderAuthRecord) error {
	updatedAt := auth.UpdatedAt
	if updatedAt == "" {
		updatedAt = domain.NowString(time.Now())
	}
	id := strings.TrimSpace(auth.ID)
	accountID := strings.TrimSpace(auth.AccountID)
	if id == "" && accountID != "" {
		var existing providerAuthRow
		err := s.db.WithContext(ctx).
			Where("provider_id = ? AND method = ? AND account_id = ?", auth.ProviderID, auth.Method, accountID).
			Order("updated_at DESC").
			First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		id = existing.ID
	}
	if id == "" {
		id = uuid.NewString()
	}
	row := providerAuthRow{
		ID:              id,
		ProviderID:      auth.ProviderID,
		Method:          auth.Method,
		AccessToken:     auth.AccessToken,
		AccessTokenRef:  auth.AccessTokenRef,
		RefreshToken:    auth.RefreshToken,
		RefreshTokenRef: auth.RefreshTokenRef,
		ExpiresAt:       auth.ExpiresAt,
		AccountID:       accountID,
		DisplayName:     strings.TrimSpace(auth.DisplayName),
		APIKey:          auth.APIKey,
		APIKeyRef:       auth.APIKeyRef,
		UpdatedAt:       updatedAt,
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if accountID != "" {
			if err := tx.Where("provider_id = ? AND method = ? AND account_id = ? AND id <> ?", auth.ProviderID, auth.Method, accountID, id).Delete(&providerAuthRow{}).Error; err != nil {
				return err
			}
		}
		return tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
	})
}

func (s *Store) LoadProviderAuth(ctx context.Context, providerID string) (*domain.ProviderAuthRecord, error) {
	var row providerAuthRow
	err := s.db.WithContext(ctx).Where("provider_id = ?", providerID).Order("updated_at DESC").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	auth := providerAuthFromRow(row)
	return &auth, nil
}

func (s *Store) GetProviderAuth(ctx context.Context, id string) (*domain.ProviderAuthRecord, error) {
	var row providerAuthRow
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	auth := providerAuthFromRow(row)
	return &auth, nil
}

func (s *Store) ListProviderAuths(ctx context.Context, providerID string) ([]domain.ProviderAuthRecord, error) {
	var rows []providerAuthRow
	if err := s.db.WithContext(ctx).Where("provider_id = ?", providerID).Order("updated_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	records := make([]domain.ProviderAuthRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, providerAuthFromRow(row))
	}
	return records, nil
}

func (s *Store) DeleteProviderAuth(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&providerAuthRow{ID: id}).Error
}

func providerAuthFromRow(row providerAuthRow) domain.ProviderAuthRecord {
	return domain.ProviderAuthRecord{
		ID: row.ID, ProviderID: row.ProviderID, Method: row.Method, AccessToken: row.AccessToken, AccessTokenRef: row.AccessTokenRef,
		RefreshToken: row.RefreshToken, RefreshTokenRef: row.RefreshTokenRef, ExpiresAt: row.ExpiresAt, AccountID: row.AccountID,
		DisplayName: row.DisplayName, APIKey: row.APIKey, APIKeyRef: row.APIKeyRef, UpdatedAt: row.UpdatedAt,
	}
}

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
	row := appConfigRow{ID: 1, Initialized: initialized, ProviderID: providerID, ProviderType: providerType, BaseURL: baseURL, APIKeyEnv: apiKeyEnv, Model: model, AuxiliaryModel: encodeModelRef(cfg.AuxiliaryModel), ReasoningEffort: cfg.ReasoningEffort, ServiceTier: cfg.ServiceTier, FallbackModels: encodeModelRefs(cfg.FallbackModels), ProviderPolicy: encodeProviderRuntimePolicy(cfg.ProviderPolicy), WebSearch: encodeWebSearchConfig(cfg.WebSearch), NativeTools: encodeNativeToolsConfig(cfg.NativeTools), Headers: headers, RequestParams: requestParams, UpdatedAt: domain.NowString(time.Now())}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
}

func (s *Store) SaveProvider(ctx context.Context, provider domain.ProviderConfig) error {
	now := domain.NowString(time.Now())
	row := providerRow{ID: provider.ID, Type: provider.Type, BaseURL: provider.BaseURL, APIKeyEnv: provider.APIKeyEnv, Model: provider.Model, Headers: encodeHeaders(provider.Headers), RequestParams: encodeAnyMap(provider.RequestParams), Status: "ready", LastValidationAt: now, UpdatedAt: now}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
}

func (s *Store) ListProviders(ctx context.Context) ([]domain.ProviderConfig, error) {
	var rows []providerRow
	if err := s.db.WithContext(ctx).Order("updated_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	providers := make([]domain.ProviderConfig, 0, len(rows))
	for _, row := range rows {
		providers = append(providers, domain.ProviderConfig{
			ID:            row.ID,
			Type:          row.Type,
			BaseURL:       row.BaseURL,
			APIKeyEnv:     row.APIKeyEnv,
			Model:         row.Model,
			Headers:       decodeHeaders(row.Headers),
			RequestParams: decodeAnyMap(row.RequestParams),
		})
	}
	return providers, nil
}

func (s *Store) DeleteProvider(ctx context.Context, providerID string) error {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return nil
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&providerRow{ID: providerID}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&providerModelCacheRow{ProviderID: providerID}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&providerValidationRow{ProviderID: providerID}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&providerHealthRow{ProviderID: providerID}).Error; err != nil {
			return err
		}
		return tx.Where("provider_id = ?", providerID).Delete(&providerAuthRow{}).Error
	})
}

func (s *Store) SaveProviderModelCache(ctx context.Context, cache domain.ProviderModelCache) error {
	now := domain.NowString(time.Now())
	if cache.UpdatedAt == "" {
		cache.UpdatedAt = now
	}
	if cache.RefreshedAt == "" && cache.Status == "ready" {
		cache.RefreshedAt = cache.UpdatedAt
	}
	if cache.Status == "" {
		cache.Status = "ready"
	}
	row := providerModelCacheRow{
		ProviderID:   cache.ProviderID,
		Models:       encodeModels(cache.Models),
		DefaultModel: cache.DefaultModel,
		Strategy:     cache.Strategy,
		ParserType:   cache.ParserType,
		Endpoint:     cache.Endpoint,
		CacheSource:  cache.CacheSource,
		Status:       cache.Status,
		Error:        cache.Error,
		RefreshedAt:  cache.RefreshedAt,
		UpdatedAt:    cache.UpdatedAt,
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
}

func (s *Store) LoadProviderModelCache(ctx context.Context, providerID string) (*domain.ProviderModelCache, error) {
	var row providerModelCacheRow
	err := s.db.WithContext(ctx).Where("provider_id = ?", providerID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	cache := providerModelCacheFromRow(row)
	return &cache, nil
}

func (s *Store) ListProviderModelCaches(ctx context.Context) ([]domain.ProviderModelCache, error) {
	var rows []providerModelCacheRow
	if err := s.db.WithContext(ctx).Order("updated_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	caches := make([]domain.ProviderModelCache, 0, len(rows))
	for _, row := range rows {
		caches = append(caches, providerModelCacheFromRow(row))
	}
	return caches, nil
}

func providerModelCacheFromRow(row providerModelCacheRow) domain.ProviderModelCache {
	return domain.ProviderModelCache{
		ProviderID: row.ProviderID, Models: decodeModels(row.Models), DefaultModel: row.DefaultModel,
		Strategy: row.Strategy, ParserType: row.ParserType, Endpoint: row.Endpoint, CacheSource: row.CacheSource,
		Status: row.Status, Error: row.Error, RefreshedAt: row.RefreshedAt, UpdatedAt: row.UpdatedAt,
	}
}

func (s *Store) SaveProviderValidation(ctx context.Context, result domain.ProviderValidationResult) error {
	if result.CheckedAt == "" {
		result.CheckedAt = domain.NowString(time.Now())
	}
	ready := 0
	if result.Ready {
		ready = 1
	}
	row := providerValidationRow{
		ProviderID: result.ProviderID, Ready: ready, Status: result.Status, Transport: result.Transport, AuthMode: result.AuthMode,
		Source: result.Source, Environment: result.Environment, BaseURL: result.BaseURL, DefaultModel: result.DefaultModel,
		ModelCount: result.ModelCount, Models: encodeModels(result.Models), Error: result.Error, CheckedAt: result.CheckedAt,
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
}

func (s *Store) LoadProviderValidation(ctx context.Context, providerID string) (*domain.ProviderValidationResult, error) {
	var row providerValidationRow
	err := s.db.WithContext(ctx).Where("provider_id = ?", providerID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	result := domain.ProviderValidationResult{
		ProviderID: row.ProviderID, Ready: row.Ready == 1, Status: row.Status, Transport: row.Transport, AuthMode: row.AuthMode,
		Source: row.Source, Environment: row.Environment, BaseURL: row.BaseURL, DefaultModel: row.DefaultModel,
		ModelCount: row.ModelCount, Models: decodeModels(row.Models), Error: row.Error, CheckedAt: row.CheckedAt,
	}
	return &result, nil
}

func (s *Store) SaveProviderHealth(ctx context.Context, health domain.ProviderHealth) error {
	if health.UpdatedAt == "" {
		health.UpdatedAt = domain.NowString(time.Now())
	}
	if health.Status == "" {
		health.Status = "unknown"
	}
	row := providerHealthRow{
		ProviderID:       health.ProviderID,
		Status:           health.Status,
		LastSuccessAt:    health.LastSuccessAt,
		LastFailureAt:    health.LastFailureAt,
		LastLatencyMs:    health.LastLatencyMs,
		LastErrorClass:   health.LastErrorClass,
		LastErrorMessage: health.LastErrorMessage,
		LastHTTPStatus:   health.LastHTTPStatus,
		FailureCount:     health.FailureCount,
		UpdatedAt:        health.UpdatedAt,
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
}

func (s *Store) LoadProviderHealth(ctx context.Context, providerID string) (*domain.ProviderHealth, error) {
	var row providerHealthRow
	err := s.db.WithContext(ctx).Where("provider_id = ?", providerID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	health := providerHealthFromRow(row)
	return &health, nil
}

func (s *Store) ListProviderHealth(ctx context.Context) ([]domain.ProviderHealth, error) {
	var rows []providerHealthRow
	if err := s.db.WithContext(ctx).Order("updated_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	health := make([]domain.ProviderHealth, 0, len(rows))
	for _, row := range rows {
		health = append(health, providerHealthFromRow(row))
	}
	return health, nil
}

func providerHealthFromRow(row providerHealthRow) domain.ProviderHealth {
	return domain.ProviderHealth{
		ProviderID:       row.ProviderID,
		Status:           row.Status,
		LastSuccessAt:    row.LastSuccessAt,
		LastFailureAt:    row.LastFailureAt,
		LastLatencyMs:    row.LastLatencyMs,
		LastErrorClass:   row.LastErrorClass,
		LastErrorMessage: row.LastErrorMessage,
		LastHTTPStatus:   row.LastHTTPStatus,
		FailureCount:     row.FailureCount,
		UpdatedAt:        row.UpdatedAt,
	}
}

func (s *Store) SaveProviderCallEvent(ctx context.Context, event domain.ProviderCallEvent) error {
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.CreatedAt == "" {
		event.CreatedAt = domain.NowString(time.Now())
	}
	streaming := 0
	if event.Streaming {
		streaming = 1
	}
	estimated := 0
	if event.Estimated {
		estimated = 1
	}
	row := providerCallEventRow{
		ID:            event.ID,
		ProviderID:    event.ProviderID,
		ModelID:       event.ModelID,
		Transport:     event.Transport,
		Status:        event.Status,
		ErrorClass:    event.ErrorClass,
		ErrorMessage:  event.ErrorMessage,
		HTTPStatus:    event.HTTPStatus,
		LatencyMs:     event.LatencyMs,
		InputTokens:   event.InputTokens,
		OutputTokens:  event.OutputTokens,
		TotalTokens:   event.TotalTokens,
		CostMicros:    event.CostMicros,
		Estimated:     estimated,
		Attempt:       event.Attempt,
		FallbackIndex: event.FallbackIndex,
		Streaming:     streaming,
		ToolCallCount: event.ToolCallCount,
		CreatedAt:     event.CreatedAt,
	}
	return s.db.WithContext(ctx).Create(&row).Error
}

func (s *Store) ListProviderCallEvents(ctx context.Context, providerID string, limit int) ([]domain.ProviderCallEvent, error) {
	if limit <= 0 || limit > 5000 {
		limit = 50
	}
	query := s.db.WithContext(ctx).Order("created_at DESC").Limit(limit)
	if strings.TrimSpace(providerID) != "" {
		query = query.Where("provider_id = ?", providerID)
	}
	var rows []providerCallEventRow
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	events := make([]domain.ProviderCallEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, providerCallEventFromRow(row))
	}
	return events, nil
}

func providerCallEventFromRow(row providerCallEventRow) domain.ProviderCallEvent {
	return domain.ProviderCallEvent{
		ID:            row.ID,
		ProviderID:    row.ProviderID,
		ModelID:       row.ModelID,
		Transport:     row.Transport,
		Status:        row.Status,
		ErrorClass:    row.ErrorClass,
		ErrorMessage:  row.ErrorMessage,
		HTTPStatus:    row.HTTPStatus,
		LatencyMs:     row.LatencyMs,
		InputTokens:   row.InputTokens,
		OutputTokens:  row.OutputTokens,
		TotalTokens:   row.TotalTokens,
		CostMicros:    row.CostMicros,
		Estimated:     row.Estimated == 1,
		Attempt:       row.Attempt,
		FallbackIndex: row.FallbackIndex,
		Streaming:     row.Streaming == 1,
		ToolCallCount: row.ToolCallCount,
		CreatedAt:     row.CreatedAt,
	}
}

func encodeHeaders(headers map[string]string) string {
	if len(headers) == 0 {
		return ""
	}
	raw, err := json.Marshal(headers)
	if err != nil {
		return ""
	}
	return string(raw)
}

func decodeHeaders(raw string) map[string]string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var headers map[string]string
	if err := json.Unmarshal([]byte(raw), &headers); err != nil {
		return nil
	}
	return headers
}

func encodeModelRefs(models []domain.ModelRef) string {
	if len(models) == 0 {
		return ""
	}
	raw, err := json.Marshal(models)
	if err != nil {
		return ""
	}
	return string(raw)
}

func encodeModelRef(model *domain.ModelRef) string {
	if model == nil || strings.TrimSpace(model.ProviderID) == "" || strings.TrimSpace(model.ModelID) == "" {
		return ""
	}
	return encodeModelRefs([]domain.ModelRef{*model})
}

func decodeModelRef(raw string) *domain.ModelRef {
	models := decodeModelRefs(raw)
	if len(models) == 0 {
		return nil
	}
	model := models[0]
	return &model
}

func decodeModelRefs(raw string) []domain.ModelRef {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var models []domain.ModelRef
	if err := json.Unmarshal([]byte(raw), &models); err != nil {
		return nil
	}
	return models
}

func encodeProviderRuntimePolicy(policy domain.ProviderRuntimePolicy) string {
	raw, err := json.Marshal(policy)
	if err != nil {
		return ""
	}
	return string(raw)
}

func decodeProviderRuntimePolicy(raw string) domain.ProviderRuntimePolicy {
	if strings.TrimSpace(raw) == "" {
		return defaultProviderRuntimePolicy()
	}
	var policy domain.ProviderRuntimePolicy
	if err := json.Unmarshal([]byte(raw), &policy); err != nil {
		return defaultProviderRuntimePolicy()
	}
	return normalizeProviderRuntimePolicy(policy)
}

func encodeWebSearchConfig(config domain.WebSearchConfig) string {
	config = normalizeWebSearchConfig(config)
	raw, err := json.Marshal(config)
	if err != nil {
		return ""
	}
	return string(raw)
}

func decodeWebSearchConfig(raw string) domain.WebSearchConfig {
	if strings.TrimSpace(raw) == "" {
		return defaultWebSearchConfig()
	}
	var config domain.WebSearchConfig
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return defaultWebSearchConfig()
	}
	return normalizeWebSearchConfig(config)
}

func encodeNativeToolsConfig(config domain.NativeToolsConfig) string {
	config = normalizeNativeToolsConfig(config)
	raw, err := json.Marshal(config)
	if err != nil {
		return ""
	}
	return string(raw)
}

func decodeNativeToolsConfig(raw string) domain.NativeToolsConfig {
	if strings.TrimSpace(raw) == "" {
		return domain.NativeToolsConfig{}
	}
	var config domain.NativeToolsConfig
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return domain.NativeToolsConfig{}
	}
	return normalizeNativeToolsConfig(config)
}

func normalizeNativeToolsConfig(config domain.NativeToolsConfig) domain.NativeToolsConfig {
	config.CodeExecution.FileIDs = normalizeIDList(config.CodeExecution.FileIDs)
	config.FileSearch.VectorStoreIDs = normalizeIDList(config.FileSearch.VectorStoreIDs)
	if len(config.RemoteMCP) > 0 {
		out := make([]domain.NativeMCPToolConfig, 0, len(config.RemoteMCP))
		seen := map[string]bool{}
		for _, server := range config.RemoteMCP {
			server.ServerURL = strings.TrimSpace(server.ServerURL)
			server.ServerLabel = strings.TrimSpace(server.ServerLabel)
			server.AllowedTools = normalizeIDList(server.AllowedTools)
			if !server.Enabled || server.ServerURL == "" || server.ServerLabel == "" {
				continue
			}
			key := server.ServerURL + "\x00" + server.ServerLabel
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, server)
		}
		config.RemoteMCP = out
	}
	return config
}

func normalizeIDList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func normalizeWebSearchConfig(config domain.WebSearchConfig) domain.WebSearchConfig {
	defaults := defaultWebSearchConfig()
	switch strings.TrimSpace(config.Mode) {
	case domain.WebSearchModeDisabled, domain.WebSearchModeLive:
	default:
		config.Mode = defaults.Mode
	}
	switch strings.TrimSpace(config.Route) {
	case domain.WebSearchRouteAuto, domain.WebSearchRouteLocal, domain.WebSearchRouteProvider:
	default:
		config.Route = defaults.Route
	}
	if strings.TrimSpace(config.LocalProvider) == "" {
		config.LocalProvider = defaults.LocalProvider
	}
	switch strings.TrimSpace(config.SearchContextSize) {
	case "", "low", "medium", "high":
	default:
		config.SearchContextSize = ""
	}
	config.AllowedDomains = normalizeDomainFilters(config.AllowedDomains)
	if config.UserLocation != nil && strings.TrimSpace(config.UserLocation.Type) == "" {
		config.UserLocation.Type = "approximate"
	}
	return config
}

func normalizeDomainFilters(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(strings.ToLower(value))
		value = strings.TrimPrefix(value, "http://")
		value = strings.TrimPrefix(value, "https://")
		value = strings.Trim(value, "/")
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func normalizeProviderRuntimePolicy(policy domain.ProviderRuntimePolicy) domain.ProviderRuntimePolicy {
	defaults := defaultProviderRuntimePolicy()
	if policy.EnableFallback == nil && policy.BufferStreamingFallback == nil && policy.MaxRetries == 0 && policy.RetryBaseDelayMs == 0 && policy.RateLimitCooldownSeconds == 0 {
		return defaults
	}
	if policy.EnableFallback == nil {
		policy.EnableFallback = defaults.EnableFallback
	}
	if policy.BufferStreamingFallback == nil {
		policy.BufferStreamingFallback = defaults.BufferStreamingFallback
	}
	if policy.MaxRetries < 0 {
		policy.MaxRetries = 0
	}
	if policy.MaxRetries > 5 {
		policy.MaxRetries = 5
	}
	if policy.RetryBaseDelayMs <= 0 {
		policy.RetryBaseDelayMs = defaults.RetryBaseDelayMs
	}
	if policy.RetryBaseDelayMs > 5000 {
		policy.RetryBaseDelayMs = 5000
	}
	if policy.RateLimitCooldownSeconds <= 0 {
		policy.RateLimitCooldownSeconds = defaults.RateLimitCooldownSeconds
	}
	if policy.RateLimitCooldownSeconds > 3600 {
		policy.RateLimitCooldownSeconds = 3600
	}
	return policy
}

func encodeModels(models []domain.ModelInfo) string {
	if len(models) == 0 {
		return "[]"
	}
	raw, err := json.Marshal(models)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func decodeModels(raw string) []domain.ModelInfo {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var models []domain.ModelInfo
	if err := json.Unmarshal([]byte(raw), &models); err != nil {
		return nil
	}
	return models
}

func (s *Store) UpsertProject(ctx context.Context, rootPath string) (domain.AssistantProject, error) {
	now := domain.NowString(time.Now())
	name := filepath.Base(strings.TrimRight(rootPath, string(os.PathSeparator)))
	row := projectRow{ID: uuid.NewString(), Name: name, RootPath: rootPath, TimeOpened: now, TimeUpdated: now}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "root_path"}},
		DoUpdates: clause.Assignments(map[string]any{"sidebar_hidden": 0, "time_updated": now}),
	}).Create(&row).Error
	if err != nil {
		return domain.AssistantProject{}, err
	}
	var saved projectRow
	if err := s.db.WithContext(ctx).Where("root_path = ?", rootPath).First(&saved).Error; err != nil {
		return domain.AssistantProject{}, err
	}
	return projectFromRow(saved), nil
}

func (s *Store) SetProjectSidebarHidden(ctx context.Context, rootPath string, hidden bool) (domain.AssistantProject, error) {
	rootPath = strings.TrimSpace(rootPath)
	if rootPath == "" {
		return domain.AssistantProject{}, errors.New("project path is required")
	}
	project, err := s.UpsertProject(ctx, rootPath)
	if err != nil {
		return domain.AssistantProject{}, err
	}
	now := domain.NowString(time.Now())
	hiddenValue := 0
	if hidden {
		hiddenValue = 1
	}
	if err := s.db.WithContext(ctx).
		Model(&projectRow{}).
		Where("id = ?", project.ID).
		Updates(map[string]any{"sidebar_hidden": hiddenValue, "time_updated": now}).
		Error; err != nil {
		return domain.AssistantProject{}, err
	}
	var saved projectRow
	if err := s.db.WithContext(ctx).Where("id = ?", project.ID).First(&saved).Error; err != nil {
		return domain.AssistantProject{}, err
	}
	return projectFromRow(saved), nil
}

func (s *Store) ListProjects(ctx context.Context, limit int) ([]domain.AssistantProject, error) {
	if limit <= 0 {
		limit = 20
	}
	var rows []projectRow
	if err := s.db.WithContext(ctx).
		Where("sidebar_hidden = ?", 0).
		Order("time_updated DESC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	projects := make([]domain.AssistantProject, 0, len(rows))
	for _, row := range rows {
		projects = append(projects, projectFromRow(row))
	}
	return projects, nil
}

func projectFromRow(row projectRow) domain.AssistantProject {
	return domain.AssistantProject{ID: row.ID, Name: row.Name, RootPath: row.RootPath, GitBranch: row.GitBranch, GitDirty: row.GitDirty == 1, GitAvailable: row.GitAvailable == 1, SidebarHidden: row.SidebarHidden == 1, TimeOpened: row.TimeOpened, TimeUpdated: row.TimeUpdated}
}
