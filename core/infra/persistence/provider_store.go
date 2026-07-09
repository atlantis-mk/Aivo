package persistence

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aivo/core/domain"
)

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
