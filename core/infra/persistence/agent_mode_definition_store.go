package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aivo/core/domain"
)

const agentModeDefinitionLimit = 128

func (s *Store) ListAgentModeDefinitions(ctx context.Context) ([]domain.AgentModeDefinition, error) {
	var rows []agentModeDefinitionRow
	if err := s.db.WithContext(ctx).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	definitions := make([]domain.AgentModeDefinition, 0, len(rows))
	for _, row := range rows {
		var definition domain.AgentModeDefinition
		if err := json.Unmarshal([]byte(row.Definition), &definition); err != nil {
			return nil, errors.New("decode persisted agent mode " + row.ID)
		}
		definition.ID = row.ID
		definitions = append(definitions, definition)
	}
	return definitions, nil
}

func (s *Store) SaveAgentModeDefinition(ctx context.Context, definition domain.AgentModeDefinition) error {
	id := strings.TrimSpace(definition.ID)
	if id == "" {
		return errors.New("agent mode id is required")
	}
	definition.ID = id
	definition.Source = ""
	definition.BuiltIn = false
	definition.Overridden = false
	definition.Revision = ""
	raw, err := json.Marshal(definition)
	if err != nil {
		return err
	}
	if len(raw) > 64*1024 {
		return errors.New("agent mode definition is too large")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing int64
		if err := tx.Model(&agentModeDefinitionRow{}).Where("id = ?", id).Count(&existing).Error; err != nil {
			return err
		}
		if existing == 0 {
			var count int64
			if err := tx.Model(&agentModeDefinitionRow{}).Count(&count).Error; err != nil {
				return err
			}
			if count >= agentModeDefinitionLimit {
				return errors.New("agent mode definition limit reached")
			}
		}
		now := domain.NowString(time.Now())
		row := agentModeDefinitionRow{ID: id, Definition: string(raw), TimeCreated: now, TimeUpdated: now}
		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"definition": string(raw), "time_updated": now,
			}),
		}).Create(&row).Error
	})
}

func (s *Store) DeleteAgentModeDefinition(ctx context.Context, id string, requireUnreferenced bool) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("agent mode id is required")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if requireUnreferenced {
			var count int64
			if err := tx.Model(&sessionRow{}).Where("agent_mode = ?", id).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return errors.New("agent mode is referenced by an existing session")
			}
			var definitions []agentModeDefinitionRow
			if err := tx.Find(&definitions).Error; err != nil {
				return err
			}
			for _, row := range definitions {
				var definition domain.AgentModeDefinition
				if err := json.Unmarshal([]byte(row.Definition), &definition); err != nil {
					return errors.New("decode persisted agent mode " + row.ID)
				}
				for _, subagent := range definition.Subagents {
					if strings.TrimSpace(subagent) == id {
						return errors.New("agent mode is referenced by another agent mode")
					}
				}
			}
		}
		result := tx.Where("id = ?", id).Delete(&agentModeDefinitionRow{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 && requireUnreferenced {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}
