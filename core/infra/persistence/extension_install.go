package persistence

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"gorm.io/gorm/clause"

	"aivo/core/domain"
)

func (s *Store) SaveExtensionInstall(ctx context.Context, install domain.ExtensionInstall) (domain.ExtensionInstall, error) {
	now := domain.NowString(time.Now())
	if install.TimeCreated == "" {
		if current, err := s.GetExtensionInstall(ctx, install.ID); err == nil {
			install.TimeCreated = current.TimeCreated
		}
	}
	if install.TimeCreated == "" {
		install.TimeCreated = now
	}
	install.TimeUpdated = now
	raw, err := json.Marshal(install.Manifest)
	if err != nil {
		return domain.ExtensionInstall{}, err
	}
	installMode := strings.TrimSpace(install.InstallMode)
	if installMode == "" {
		installMode = domain.ExtensionInstallModeLinked
	}
	row := extensionInstallRow{
		ID: strings.TrimSpace(install.ID), Manifest: string(raw), RootPath: normalizeStoredPath(install.RootPath),
		InstallMode:  installMode,
		ManifestPath: normalizeStoredPath(install.ManifestPath), Integrity: strings.TrimSpace(install.Integrity),
		Enabled: boolInt(install.Enabled), Status: strings.TrimSpace(install.Status), Error: strings.TrimSpace(install.Error),
		TimeCreated: install.TimeCreated, TimeUpdated: install.TimeUpdated,
	}
	err = s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"manifest", "install_mode", "root_path", "manifest_path", "integrity", "enabled", "status", "error", "time_updated",
		}),
	}).Create(&row).Error
	if err != nil {
		return domain.ExtensionInstall{}, err
	}
	return extensionInstallFromRow(row), nil
}

func (s *Store) GetExtensionInstall(ctx context.Context, id string) (domain.ExtensionInstall, error) {
	var row extensionInstallRow
	if err := s.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).First(&row).Error; err != nil {
		return domain.ExtensionInstall{}, err
	}
	return extensionInstallFromRow(row), nil
}

func (s *Store) ListExtensionInstalls(ctx context.Context) ([]domain.ExtensionInstall, error) {
	var rows []extensionInstallRow
	if err := s.db.WithContext(ctx).Order("time_updated DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ExtensionInstall, 0, len(rows))
	for _, row := range rows {
		out = append(out, extensionInstallFromRow(row))
	}
	return out, nil
}

func (s *Store) SetExtensionInstallState(ctx context.Context, id string, enabled bool, status string, errText string) (domain.ExtensionInstall, error) {
	err := s.db.WithContext(ctx).Model(&extensionInstallRow{}).Where("id = ?", strings.TrimSpace(id)).Updates(map[string]any{
		"enabled": boolInt(enabled), "status": strings.TrimSpace(status), "error": strings.TrimSpace(errText),
		"time_updated": domain.NowString(time.Now()),
	}).Error
	if err != nil {
		return domain.ExtensionInstall{}, err
	}
	return s.GetExtensionInstall(ctx, id)
}

func (s *Store) DeleteExtensionInstall(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&extensionInstallRow{}, "id = ?", strings.TrimSpace(id)).Error
}

func extensionInstallFromRow(row extensionInstallRow) domain.ExtensionInstall {
	var manifest domain.ExtensionManifest
	_ = json.Unmarshal([]byte(row.Manifest), &manifest)
	return domain.ExtensionInstall{
		ID: row.ID, Manifest: manifest, Summary: domain.ExtensionSummary(manifest), InstallMode: row.InstallMode, RootPath: row.RootPath, ManifestPath: row.ManifestPath, Integrity: row.Integrity,
		Enabled: row.Enabled != 0, Status: row.Status, Error: row.Error, TimeCreated: row.TimeCreated, TimeUpdated: row.TimeUpdated,
	}
}
