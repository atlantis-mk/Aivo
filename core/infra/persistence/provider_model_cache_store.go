package persistence

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aivo/core/domain"
)

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
