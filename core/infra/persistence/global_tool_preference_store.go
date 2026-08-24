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

const globalDisabledToolLimit = 512

func (s *Store) ListGloballyDisabledToolNames(ctx context.Context) ([]string, error) {
	var rows []globalToolPreferenceRow
	if err := s.db.WithContext(ctx).Where("enabled = ?", 0).Order("name ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		if name := strings.TrimSpace(row.Name); name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}

func (s *Store) SetGlobalToolEnabled(ctx context.Context, name string, enabled bool) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("tool name is required")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if enabled {
			return tx.Where("name = ?", name).Delete(&globalToolPreferenceRow{}).Error
		}
		var existing int64
		if err := tx.Model(&globalToolPreferenceRow{}).Where("name = ? AND enabled = ?", name, 0).Count(&existing).Error; err != nil {
			return err
		}
		if existing == 0 {
			var count int64
			if err := tx.Model(&globalToolPreferenceRow{}).Where("enabled = ?", 0).Count(&count).Error; err != nil {
				return err
			}
			if count >= globalDisabledToolLimit {
				return errors.New("global disabled-tool limit reached")
			}
		}
		now := domain.NowString(time.Now())
		row := globalToolPreferenceRow{Name: name, Enabled: 0, TimeCreated: now, TimeUpdated: now}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoUpdates: clause.Assignments(map[string]any{"enabled": 0, "time_updated": now}),
		}).Create(&row).Error
	})
}
