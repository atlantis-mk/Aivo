package persistence

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aivo/core/domain"
)

func (s *Store) migrate(ctx context.Context) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.AutoMigrate(&schemaVersionRow{}, &appConfigRow{}, &providerRow{}, &providerModelCacheRow{}, &providerValidationRow{}, &providerHealthRow{}, &providerCallEventRow{}, &projectRow{}, &sessionRow{}); err != nil {
			return err
		}
		if err := migrateProviderAuth(ctx, tx); err != nil {
			return err
		}
		if err := tx.AutoMigrate(&turnRow{}, &sessionEventRow{}, &toolCallRow{}, &sessionExecutionStateRow{}, &pendingSessionInputRow{}, &permissionRequestRow{}, &questionRequestRow{}, &permissionRuleRow{}, &sessionSummaryRow{}, &sessionCheckpointRow{}, &codingContextRow{}, &gitWorktreeRow{}, &agentRunRow{}, &todoItemRow{}, &scheduledJobRow{}, &pluginInstallRow{}, &pluginDiagnosticRow{}, &mcpServerRow{}, &mcpToolRow{}, &mcpPromptRow{}, &mcpResourceRow{}, &skillRow{}, &skillSourceRow{}, &skillImportCandidateRow{}, &toolRegistrationRow{}); err != nil {
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
