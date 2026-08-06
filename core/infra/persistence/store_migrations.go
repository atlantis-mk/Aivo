package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aivo/core/domain"
)

const latestSchemaVersion = 4

func (s *Store) migrate(ctx context.Context) error {
	version, hasVersionTable, err := s.currentSchemaVersion(ctx)
	if err != nil {
		return err
	}
	if version > latestSchemaVersion {
		return fmt.Errorf("database schema version %d is newer than supported version %d", version, latestSchemaVersion)
	}
	if version == latestSchemaVersion {
		if !s.db.WithContext(ctx).Migrator().HasColumn(&appConfigRow{}, "initial_workspace_path") {
			return errors.New("database schema version 2 is missing app_config.initial_workspace_path")
		}
		if !s.db.WithContext(ctx).Migrator().HasTable(&extensionInstallRow{}) {
			return errors.New("database schema version 4 is missing extension_installs")
		}
		if !s.db.WithContext(ctx).Migrator().HasColumn(&extensionInstallRow{}, "install_mode") {
			return errors.New("database schema version 4 is missing extension_installs.install_mode")
		}
		return nil
	}

	hasLegacyData := hasVersionTable || s.hasApplicationSchema()
	if hasLegacyData {
		backupVersion := version
		if backupVersion == 0 {
			backupVersion = 1
		}
		if err := s.ensureMigrationBackup(ctx, backupVersion); err != nil {
			return err
		}
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.AutoMigrate(&schemaVersionRow{}, &appConfigRow{}, &providerRow{}, &providerModelCacheRow{}, &providerValidationRow{}, &providerHealthRow{}, &providerCallEventRow{}, &projectRow{}, &sessionRow{}); err != nil {
			return err
		}
		if err := migrateProviderAuth(ctx, tx); err != nil {
			return err
		}
		if err := tx.AutoMigrate(&turnRow{}, &sessionEventRow{}, &toolCallRow{}, &sessionExecutionStateRow{}, &pendingSessionInputRow{}, &permissionRequestRow{}, &questionRequestRow{}, &permissionRuleRow{}, &sessionSummaryRow{}, &sessionCheckpointRow{}, &codingContextRow{}, &gitWorktreeRow{}, &agentRunRow{}, &todoItemRow{}, &scheduledJobRow{}, &pluginInstallRow{}, &pluginDiagnosticRow{}, &mcpServerRow{}, &mcpToolRow{}, &mcpPromptRow{}, &mcpResourceRow{}, &skillRow{}, &skillSourceRow{}, &skillImportCandidateRow{}, &toolRegistrationRow{}, &extensionInstallRow{}); err != nil {
			return err
		}
		if err := migrateLegacyMessages(ctx, tx); err != nil {
			return err
		}
		now := domain.NowString(time.Now())
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&schemaVersionRow{Version: latestSchemaVersion, AppliedAt: now}).Error
	})
}

func (s *Store) currentSchemaVersion(ctx context.Context) (int, bool, error) {
	migrator := s.db.WithContext(ctx).Migrator()
	if !migrator.HasTable(&schemaVersionRow{}) {
		return 0, false, nil
	}
	var version int
	if err := s.db.WithContext(ctx).Model(&schemaVersionRow{}).Select("COALESCE(MAX(version), 0)").Scan(&version).Error; err != nil {
		return 0, true, err
	}
	return version, true, nil
}

func (s *Store) hasApplicationSchema() bool {
	migrator := s.db.Migrator()
	for _, table := range []string{"app_config", "providers", "projects", "sessions", "messages"} {
		if migrator.HasTable(table) {
			return true
		}
	}
	return false
}

func (s *Store) ensureMigrationBackup(ctx context.Context, version int) error {
	if strings.TrimSpace(s.path) == "" || s.path == ":memory:" {
		return errors.New("cannot migrate an existing database without a backup path")
	}
	backupPath := migrationBackupPath(s.path, version)
	if _, err := os.Stat(backupPath); err == nil {
		if err := verifySQLiteBackup(backupPath); err != nil {
			return fmt.Errorf("verify schema v%d migration backup: %w", version, err)
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect schema v%d migration backup: %w", version, err)
	}

	temporary, err := os.CreateTemp(filepath.Dir(backupPath), filepath.Base(backupPath)+".tmp-*")
	if err != nil {
		return fmt.Errorf("prepare schema v%d migration backup: %w", version, err)
	}
	temporaryPath := temporary.Name()
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("prepare schema v%d migration backup: %w", version, err)
	}
	if err := os.Remove(temporaryPath); err != nil {
		return fmt.Errorf("prepare schema v%d migration backup: %w", version, err)
	}
	defer os.Remove(temporaryPath)

	quotedPath := strings.ReplaceAll(temporaryPath, "'", "''")
	if err := s.db.WithContext(ctx).Exec("VACUUM INTO '" + quotedPath + "'").Error; err != nil {
		return fmt.Errorf("create schema v%d migration backup: %w", version, err)
	}
	if err := verifySQLiteBackup(temporaryPath); err != nil {
		return fmt.Errorf("verify schema v%d migration backup: %w", version, err)
	}
	if err := os.Chmod(temporaryPath, 0o600); err != nil {
		return fmt.Errorf("secure schema v%d migration backup: %w", version, err)
	}
	if err := os.Rename(temporaryPath, backupPath); err != nil {
		return fmt.Errorf("publish schema v%d migration backup: %w", version, err)
	}
	return nil
}

func migrationBackupPath(databasePath string, version int) string {
	return fmt.Sprintf("%s.v%d.bak", databasePath, version)
}

func verifySQLiteBackup(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return errors.New("backup is not a non-empty regular file")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()
	var result string
	if err := db.QueryRow("PRAGMA integrity_check").Scan(&result); err != nil {
		return err
	}
	if result != "ok" {
		return fmt.Errorf("integrity check returned %q", result)
	}
	return nil
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
