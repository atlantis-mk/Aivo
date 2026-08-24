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

func (s *Store) AppendSessionEvent(ctx context.Context, event domain.SessionEvent) error {
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.TimeCreated == "" {
		event.TimeCreated = domain.NowString(time.Now())
	}
	row := sessionEventRow{ID: event.ID, SessionID: event.SessionID, TurnID: event.TurnID, Type: event.Type, Role: event.Role, Visibility: event.Visibility, Content: event.Content, Payload: encodeAnyMap(event.Payload), TokenCount: event.TokenCount, TimeCreated: event.TimeCreated}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return tx.Model(&sessionRow{}).Where("id = ?", event.SessionID).Update("time_updated", event.TimeCreated).Error
	})
}

func (s *Store) GetSessionEvent(ctx context.Context, id string) (domain.SessionEvent, error) {
	var row sessionEventRow
	if err := s.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).First(&row).Error; err != nil {
		return domain.SessionEvent{}, err
	}
	return eventFromRow(row), nil
}

func (s *Store) ListSessionEvents(ctx context.Context, sessionID string, includeNonNormal bool, limit int) ([]domain.SessionEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := s.db.WithContext(ctx).Where("session_id = ?", sessionID)
	if !includeNonNormal {
		q = q.Where("visibility = ?", domain.EventVisibilityNormal)
	}
	var rows []sessionEventRow
	if err := q.Order("time_created DESC, id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	out := make([]domain.SessionEvent, 0, len(rows))
	for _, row := range rows {
		out = append(out, eventFromRow(row))
	}
	return out, nil
}

func (s *Store) UpdateSessionEvent(ctx context.Context, input domain.UpdateSessionEventRequest) (domain.SessionEvent, error) {
	eventID := strings.TrimSpace(input.EventID)
	if eventID == "" {
		return domain.SessionEvent{}, errors.New("eventId is required")
	}
	updates := map[string]any{"content": strings.TrimSpace(input.Content)}
	if input.Payload != nil {
		updates["payload"] = encodeAnyMap(input.Payload)
	}
	if err := s.db.WithContext(ctx).Model(&sessionEventRow{}).Where("id = ?", eventID).Updates(updates).Error; err != nil {
		return domain.SessionEvent{}, err
	}
	event, err := s.GetSessionEvent(ctx, eventID)
	if err != nil {
		return domain.SessionEvent{}, err
	}
	_ = s.db.WithContext(ctx).Model(&sessionRow{}).Where("id = ?", event.SessionID).Update("time_updated", domain.NowString(time.Now())).Error
	return event, nil
}

func (s *Store) SetSessionEventVisibility(ctx context.Context, id string, visibility string) (domain.SessionEvent, error) {
	nextVisibility, err := domain.NormalizeEventVisibility(visibility)
	if err != nil {
		return domain.SessionEvent{}, err
	}
	eventID := strings.TrimSpace(id)
	if eventID == "" {
		return domain.SessionEvent{}, errors.New("eventId is required")
	}
	if err := s.db.WithContext(ctx).Model(&sessionEventRow{}).Where("id = ?", eventID).Update("visibility", nextVisibility).Error; err != nil {
		return domain.SessionEvent{}, err
	}
	event, err := s.GetSessionEvent(ctx, eventID)
	if err != nil {
		return domain.SessionEvent{}, err
	}
	_ = s.db.WithContext(ctx).Model(&sessionRow{}).Where("id = ?", event.SessionID).Update("time_updated", domain.NowString(time.Now())).Error
	return event, nil
}

func (s *Store) HideSessionTurnEvents(ctx context.Context, turnID string) error {
	turnID = strings.TrimSpace(turnID)
	if turnID == "" {
		return errors.New("turnId is required")
	}
	return s.db.WithContext(ctx).Model(&sessionEventRow{}).Where("turn_id = ?", turnID).Update("visibility", domain.EventVisibilityHidden).Error
}

func eventFromRow(row sessionEventRow) domain.SessionEvent {
	return domain.SessionEvent{ID: row.ID, SessionID: row.SessionID, TurnID: row.TurnID, Type: row.Type, Role: row.Role, Visibility: row.Visibility, Content: row.Content, Payload: decodeAnyMap(row.Payload), TokenCount: row.TokenCount, TimeCreated: row.TimeCreated}
}

func (s *Store) StartTurn(ctx context.Context, turn domain.Turn) error {
	agentMode, err := domain.NormalizeAgentMode(turn.AgentMode)
	if err != nil {
		return err
	}
	turn.AgentMode = agentMode
	row := turnRow{ID: turn.ID, SessionID: turn.SessionID, Status: turn.Status, UserEventID: turn.UserEventID, Error: turn.Error, TimeCreated: turn.TimeCreated, TimeCompleted: turn.TimeCompleted, TimeUpdated: turn.TimeUpdated}
	row.AgentMode = turn.AgentMode
	return s.db.WithContext(ctx).Create(&row).Error
}

func (s *Store) GetTurn(ctx context.Context, turnID string) (domain.Turn, error) {
	var row turnRow
	if err := s.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(turnID)).First(&row).Error; err != nil {
		return domain.Turn{}, err
	}
	return turnFromRow(row), nil
}

func (s *Store) UpdateTurnStatus(ctx context.Context, turnID string, status string, errText string) (domain.Turn, error) {
	if err := domain.ValidateTurnStatus(status); err != nil {
		return domain.Turn{}, err
	}
	now := domain.NowString(time.Now())
	updates := map[string]any{"status": status, "error": errText, "time_updated": now}
	if status != domain.TurnStatusRunning {
		updates["time_completed"] = now
	}
	if err := s.db.WithContext(ctx).Model(&turnRow{}).Where("id = ?", turnID).Updates(updates).Error; err != nil {
		return domain.Turn{}, err
	}
	var row turnRow
	if err := s.db.WithContext(ctx).Where("id = ?", turnID).First(&row).Error; err != nil {
		return domain.Turn{}, err
	}
	return turnFromRow(row), nil
}

func (s *Store) ListTurns(ctx context.Context, sessionID string, limit int) ([]domain.Turn, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []turnRow
	if err := s.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("time_created DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Turn, 0, len(rows))
	for _, row := range rows {
		out = append(out, turnFromRow(row))
	}
	return out, nil
}

func turnFromRow(row turnRow) domain.Turn {
	return domain.Turn{ID: row.ID, SessionID: row.SessionID, AgentMode: persistedAgentMode(row.AgentMode), Status: row.Status, UserEventID: row.UserEventID, Error: row.Error, TimeCreated: row.TimeCreated, TimeCompleted: row.TimeCompleted, TimeUpdated: row.TimeUpdated}
}

func (s *Store) SaveToolCall(ctx context.Context, call domain.ToolCall) error {
	now := domain.NowString(time.Now())
	if call.ID == "" {
		call.ID = uuid.NewString()
	}
	if call.TimeCreated == "" {
		call.TimeCreated = now
	}
	call.TimeUpdated = now
	row := toolCallRow{ID: call.ID, SessionID: call.SessionID, TurnID: call.TurnID, EventID: call.EventID, Name: call.Name, Arguments: encodeAnyMap(call.Arguments), Status: call.Status, ResultSummary: call.ResultSummary, Result: encodeAnyMap(call.Result), Error: call.Error, TimeCreated: call.TimeCreated, TimeUpdated: call.TimeUpdated}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"session_id",
			"turn_id",
			"event_id",
			"name",
			"arguments",
			"status",
			"result_summary",
			"result",
			"error",
			"time_updated",
		}),
	}).Create(&row).Error
}

func (s *Store) ListToolCalls(ctx context.Context, sessionID string) ([]domain.ToolCall, error) {
	var rows []toolCallRow
	if err := s.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("time_created DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ToolCall, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.ToolCall{ID: row.ID, SessionID: row.SessionID, TurnID: row.TurnID, EventID: row.EventID, Name: row.Name, Arguments: decodeAnyMap(row.Arguments), Status: row.Status, ResultSummary: row.ResultSummary, Result: decodeAnyMap(row.Result), Error: row.Error, TimeCreated: row.TimeCreated, TimeUpdated: row.TimeUpdated})
	}
	return out, nil
}
