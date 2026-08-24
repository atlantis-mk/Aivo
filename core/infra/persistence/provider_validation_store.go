package persistence

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aivo/core/domain"
)

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
