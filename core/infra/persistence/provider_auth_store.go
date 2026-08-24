package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aivo/core/domain"
)

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
