package persistence

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
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

const latestSchemaVersion = 11

func (s *Store) migrate(ctx context.Context) error {
	version, hasVersionTable, err := s.currentSchemaVersion(ctx)
	if err != nil {
		return err
	}
	if version > latestSchemaVersion {
		return fmt.Errorf("database schema version %d is newer than supported version %d", version, latestSchemaVersion)
	}
	if version == latestSchemaVersion {
		if !s.db.WithContext(ctx).Migrator().HasColumn(&appConfigRow{}, "default_permission_mode") {
			return errors.New("database schema version 11 is missing app_config.default_permission_mode")
		}
		if !s.db.WithContext(ctx).Migrator().HasColumn(&appConfigRow{}, "app_name") {
			return errors.New("database schema version 10 is missing app_config.app_name")
		}
		if !s.db.WithContext(ctx).Migrator().HasColumn(&appConfigRow{}, "initial_workspace_path") {
			return errors.New("database schema version 2 is missing app_config.initial_workspace_path")
		}
		if !s.db.WithContext(ctx).Migrator().HasTable(&extensionInstallRow{}) {
			return errors.New("database schema version 4 is missing extension_installs")
		}
		if !s.db.WithContext(ctx).Migrator().HasColumn(&extensionInstallRow{}, "install_mode") {
			return errors.New("database schema version 4 is missing extension_installs.install_mode")
		}
		if !s.db.WithContext(ctx).Migrator().HasTable(&globalToolPreferenceRow{}) {
			return errors.New("database schema version 5 is missing global_tool_preferences")
		}
		if !s.db.WithContext(ctx).Migrator().HasTable(&agentModeDefinitionRow{}) {
			return errors.New("database schema version 6 is missing agent_mode_definitions")
		}
		promptRoot, rootErr := s.ManagedPromptRoot()
		if rootErr != nil {
			return rootErr
		}
		return publishPendingAgentPromptMigration(promptRoot)
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

	migrationErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.AutoMigrate(&schemaVersionRow{}, &appConfigRow{}, &providerRow{}, &providerModelCacheRow{}, &providerValidationRow{}, &providerHealthRow{}, &providerCallEventRow{}, &projectRow{}, &sessionRow{}); err != nil {
			return err
		}
		if err := migrateProviderAuth(ctx, tx); err != nil {
			return err
		}
		if err := tx.AutoMigrate(&turnRow{}, &sessionEventRow{}, &toolCallRow{}, &sessionExecutionStateRow{}, &pendingSessionInputRow{}, &permissionRequestRow{}, &questionRequestRow{}, &permissionRuleRow{}, &sessionSummaryRow{}, &sessionCheckpointRow{}, &codingContextRow{}, &gitWorktreeRow{}, &agentRunRow{}, &todoItemRow{}, &scheduledJobRow{}, &pluginInstallRow{}, &pluginDiagnosticRow{}, &mcpServerRow{}, &mcpToolRow{}, &mcpPromptRow{}, &mcpResourceRow{}, &skillRow{}, &skillSourceRow{}, &skillImportCandidateRow{}, &toolRegistrationRow{}, &extensionInstallRow{}, &globalToolPreferenceRow{}, &agentModeDefinitionRow{}); err != nil {
			return err
		}
		if err := migrateLegacyMessages(ctx, tx); err != nil {
			return err
		}
		if version < 7 {
			if err := migrateAgentModeToolsets(ctx, tx); err != nil {
				return err
			}
		}
		if version < 9 {
			promptRoot, err := s.ManagedPromptRoot()
			if err != nil {
				return err
			}
			if err := migrateAgentModePrompts(ctx, tx, promptRoot); err != nil {
				return err
			}
		}
		now := domain.NowString(time.Now())
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&schemaVersionRow{Version: latestSchemaVersion, AppliedAt: now}).Error
	})
	if migrationErr != nil {
		return migrationErr
	}
	if version < 9 {
		promptRoot, err := s.ManagedPromptRoot()
		if err != nil {
			return err
		}
		return publishPendingAgentPromptMigration(promptRoot)
	}
	return nil
}

func migrateAgentModePrompts(ctx context.Context, tx *gorm.DB, promptRoot string) error {
	if !tx.WithContext(ctx).Migrator().HasTable(&agentModeDefinitionRow{}) {
		return nil
	}
	var rows []agentModeDefinitionRow
	if err := tx.WithContext(ctx).Find(&rows).Error; err != nil {
		return err
	}
	pending := map[string]string{}
	for _, row := range rows {
		var definition map[string]any
		if err := json.Unmarshal([]byte(row.Definition), &definition); err != nil {
			return fmt.Errorf("decode agent mode %s for schema v9: %w", row.ID, err)
		}
		modeID, err := domain.NormalizeAgentMode(row.ID)
		if err != nil {
			return fmt.Errorf("normalize agent mode %s for schema v9: %w", row.ID, err)
		}
		promptID := "agent." + modeID
		if configured, _ := definition["promptId"].(string); strings.TrimSpace(configured) != "" {
			promptID = strings.TrimSpace(configured)
		}
		body, _ := definition["prompt"].(string)
		body = strings.TrimSpace(body)
		if body != "" {
			title, _ := definition["displayName"].(string)
			if strings.TrimSpace(title) == "" {
				title = modeID
			}
			revision, err := stageMigratedAgentPrompt(promptRoot, promptID, title, body)
			if err != nil {
				return fmt.Errorf("publish agent prompt %s for schema v9: %w", promptID, err)
			}
			pending[promptID] = revision
		}
		definition["promptId"] = promptID
		delete(definition, "prompt")
		raw, err := json.Marshal(definition)
		if err != nil {
			return fmt.Errorf("encode agent mode %s for schema v9: %w", row.ID, err)
		}
		if err := tx.WithContext(ctx).Model(&agentModeDefinitionRow{}).Where("id = ?", row.ID).Update("definition", string(raw)).Error; err != nil {
			return err
		}
	}
	if len(pending) == 0 {
		return nil
	}
	raw, err := json.Marshal(promptMigrationManifest{Version: 1, Prompts: pending})
	if err != nil {
		return err
	}
	return atomicWriteMigrationFile(filepath.Join(promptRoot, ".state", "migration", "pending.json"), raw)
}

type promptMigrationManifest struct {
	Version int               `json:"version"`
	Prompts map[string]string `json:"prompts"`
}

func stageMigratedAgentPrompt(promptRoot, promptID, title, body string) (string, error) {
	titleJSON, _ := json.Marshal(strings.TrimSpace(title))
	raw := []byte(fmt.Sprintf("---\nschema: aivo.prompt/v1\nid: %s\ncategory: agent\ntitle: %s\nenabled: true\n---\n\n%s\n", promptID, titleJSON, strings.TrimSpace(body)))
	if len(raw) > 32*1024 {
		return "", errors.New("agent prompt exceeds 32768 bytes")
	}
	hash := sha256.Sum256(raw)
	revision := hex.EncodeToString(hash[:])
	validatedPath := filepath.Join(promptRoot, ".state", "migration", revision+".md")
	if err := atomicWriteMigrationFile(validatedPath, raw); err != nil {
		return "", err
	}
	return revision, nil
}

func publishPendingAgentPromptMigration(promptRoot string) error {
	manifestPath := filepath.Join(promptRoot, ".state", "migration", "pending.json")
	raw, err := os.ReadFile(manifestPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var manifest promptMigrationManifest
	if err := json.Unmarshal(raw, &manifest); err != nil || manifest.Version != 1 {
		return errors.New("invalid pending prompt migration manifest")
	}
	for promptID, revision := range manifest.Prompts {
		if !strings.HasPrefix(promptID, "agent.") || strings.ContainsAny(promptID, `/\\`) || len(revision) != 64 {
			return errors.New("invalid pending prompt migration entry")
		}
		staged, err := os.ReadFile(filepath.Join(promptRoot, ".state", "migration", revision+".md"))
		if err != nil {
			return err
		}
		if !strings.Contains(string(staged), "\nid: "+promptID+"\n") {
			return errors.New("pending prompt migration content does not match its id")
		}
		if err := atomicWriteMigrationFile(filepath.Join(promptRoot, "overrides", "agent", promptID+".md"), staged); err != nil {
			return err
		}
	}
	return os.Remove(manifestPath)
}

func atomicWriteMigrationFile(path string, raw []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(dir, ".migration-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(raw); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	directory, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func migrateAgentModeToolsets(ctx context.Context, tx *gorm.DB) error {
	if !tx.WithContext(ctx).Migrator().HasTable(&agentModeDefinitionRow{}) {
		return nil
	}
	var rows []agentModeDefinitionRow
	if err := tx.WithContext(ctx).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		var definition map[string]any
		if err := json.Unmarshal([]byte(row.Definition), &definition); err != nil {
			return fmt.Errorf("decode agent mode %s for schema v7: %w", row.ID, err)
		}
		if _, exists := definition["toolsets"]; !exists {
			continue
		}
		delete(definition, "toolsets")
		raw, err := json.Marshal(definition)
		if err != nil {
			return fmt.Errorf("encode agent mode %s for schema v7: %w", row.ID, err)
		}
		if err := tx.WithContext(ctx).Model(&agentModeDefinitionRow{}).Where("id = ?", row.ID).Update("definition", string(raw)).Error; err != nil {
			return err
		}
	}
	return nil
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
