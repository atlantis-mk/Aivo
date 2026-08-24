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

func (s *Store) UpsertSessionExecutionState(ctx context.Context, state domain.SessionExecutionState) (domain.SessionExecutionState, error) {
	now := domain.NowString(time.Now())
	if state.ID == "" {
		state.ID = uuid.NewString()
	}
	if state.TimeCreated == "" {
		state.TimeCreated = now
	}
	state.TimeUpdated = now
	status, err := domain.NormalizeExecutionStatus(state.Status)
	if err != nil {
		return domain.SessionExecutionState{}, err
	}
	state.Status = status
	if existing, err := s.GetSessionExecutionState(ctx, state.SessionID); err == nil && existing.ID != "" {
		state.Metadata = mergeAnyMetadata(existing.Metadata, state.Metadata)
	}
	row := sessionExecutionStateRow{
		ID: state.ID, SessionID: strings.TrimSpace(state.SessionID), TurnID: strings.TrimSpace(state.TurnID),
		Status: state.Status, Reason: strings.TrimSpace(state.Reason), LastEventID: strings.TrimSpace(state.LastEventID),
		PendingInputIDs: encodeStrings(state.PendingInputIDs), Metadata: encodeAnyMap(state.Metadata),
		TimeCreated: state.TimeCreated, TimeUpdated: state.TimeUpdated,
	}
	err = s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "session_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"turn_id": row.TurnID, "status": row.Status, "reason": row.Reason, "last_event_id": row.LastEventID,
			"pending_input_ids": row.PendingInputIDs, "metadata": row.Metadata, "time_updated": row.TimeUpdated,
		}),
	}).Create(&row).Error
	if err != nil {
		return domain.SessionExecutionState{}, err
	}
	return s.GetSessionExecutionState(ctx, state.SessionID)
}

func (s *Store) GetSessionExecutionState(ctx context.Context, sessionID string) (domain.SessionExecutionState, error) {
	var row sessionExecutionStateRow
	if err := s.db.WithContext(ctx).Where("session_id = ?", strings.TrimSpace(sessionID)).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			now := domain.NowString(time.Now())
			return domain.SessionExecutionState{ID: uuid.NewString(), SessionID: strings.TrimSpace(sessionID), Status: domain.ExecutionStatusIdle, TimeCreated: now, TimeUpdated: now}, nil
		}
		return domain.SessionExecutionState{}, err
	}
	return executionStateFromRow(row), nil
}

func (s *Store) CreatePendingSessionInput(ctx context.Context, input domain.PendingSessionInput) (domain.PendingSessionInput, error) {
	now := domain.NowString(time.Now())
	if input.ID == "" {
		input.ID = uuid.NewString()
	}
	if input.TimeCreated == "" {
		input.TimeCreated = now
	}
	input.TimeUpdated = now
	delivery, err := domain.NormalizeInputDelivery(input.Delivery)
	if err != nil {
		return domain.PendingSessionInput{}, err
	}
	input.Delivery = delivery
	if input.Status == "" {
		input.Status = domain.PendingInputStatusPending
	}
	row := pendingSessionInputRow{
		ID: input.ID, SessionID: strings.TrimSpace(input.SessionID), TurnID: strings.TrimSpace(input.TurnID),
		Text: strings.TrimSpace(input.Text), Delivery: input.Delivery, Status: input.Status,
		PromotedTurnID: strings.TrimSpace(input.PromotedTurnID), TimeCreated: input.TimeCreated, TimeUpdated: input.TimeUpdated,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return domain.PendingSessionInput{}, err
	}
	return pendingInputFromRow(row), nil
}

func (s *Store) ListPendingSessionInputs(ctx context.Context, sessionID string, status string) ([]domain.PendingSessionInput, error) {
	q := s.db.WithContext(ctx).Where("session_id = ?", strings.TrimSpace(sessionID))
	if strings.TrimSpace(status) != "" {
		q = q.Where("status = ?", strings.TrimSpace(status))
	}
	var rows []pendingSessionInputRow
	if err := q.Order("time_created ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.PendingSessionInput, 0, len(rows))
	for _, row := range rows {
		out = append(out, pendingInputFromRow(row))
	}
	return out, nil
}

func (s *Store) UpdatePendingSessionInputStatus(ctx context.Context, inputID string, status string, promotedTurnID string) (domain.PendingSessionInput, error) {
	now := domain.NowString(time.Now())
	if err := s.db.WithContext(ctx).Model(&pendingSessionInputRow{}).Where("id = ?", strings.TrimSpace(inputID)).Updates(map[string]any{
		"status": status, "promoted_turn_id": strings.TrimSpace(promotedTurnID), "time_updated": now,
	}).Error; err != nil {
		return domain.PendingSessionInput{}, err
	}
	var row pendingSessionInputRow
	if err := s.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(inputID)).First(&row).Error; err != nil {
		return domain.PendingSessionInput{}, err
	}
	return pendingInputFromRow(row), nil
}

func (s *Store) ListSessionEventsAfterCursor(ctx context.Context, sessionID string, cursor string, includeNonNormal bool, limit int) ([]domain.SessionEvent, string, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := s.db.WithContext(ctx).Where("session_id = ?", strings.TrimSpace(sessionID))
	if strings.TrimSpace(cursor) != "" {
		var cursorRow sessionEventRow
		if err := s.db.WithContext(ctx).Where("id = ? AND session_id = ?", strings.TrimSpace(cursor), strings.TrimSpace(sessionID)).First(&cursorRow).Error; err == nil {
			q = q.Where("(time_created > ? OR (time_created = ? AND id > ?))", cursorRow.TimeCreated, cursorRow.TimeCreated, cursorRow.ID)
		} else {
			q = q.Where("id <> ?", strings.TrimSpace(cursor))
		}
	}
	if !includeNonNormal {
		q = q.Where("visibility = ?", domain.EventVisibilityNormal)
	}
	var rows []sessionEventRow
	if err := q.Order("time_created ASC, id ASC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, "", err
	}
	out := make([]domain.SessionEvent, 0, len(rows))
	nextCursor := strings.TrimSpace(cursor)
	for _, row := range rows {
		out = append(out, eventFromRow(row))
		nextCursor = row.ID
	}
	return out, nextCursor, nil
}

func (s *Store) MarkRunningToolCallsInterrupted(ctx context.Context, sessionID string, reason string) (int, error) {
	q := s.db.WithContext(ctx).Model(&toolCallRow{}).Where("status = ?", domain.ToolCallStatusRunning)
	if strings.TrimSpace(sessionID) != "" {
		q = q.Where("session_id = ?", strings.TrimSpace(sessionID))
	}
	now := domain.NowString(time.Now())
	result := q.Updates(map[string]any{
		"status": domain.ToolCallStatusInterrupted, "error": firstNonEmptyString(strings.TrimSpace(reason), "interrupted during recovery"), "time_updated": now,
	})
	return int(result.RowsAffected), result.Error
}

func executionStateFromRow(row sessionExecutionStateRow) domain.SessionExecutionState {
	return domain.SessionExecutionState{
		ID: row.ID, SessionID: row.SessionID, TurnID: row.TurnID, Status: row.Status, Reason: row.Reason,
		LastEventID: row.LastEventID, PendingInputIDs: decodeStrings(row.PendingInputIDs), Metadata: decodeAnyMap(row.Metadata),
		TimeCreated: row.TimeCreated, TimeUpdated: row.TimeUpdated,
	}
}

func pendingInputFromRow(row pendingSessionInputRow) domain.PendingSessionInput {
	return domain.PendingSessionInput{
		ID: row.ID, SessionID: row.SessionID, TurnID: row.TurnID, Text: row.Text, Delivery: row.Delivery,
		Status: row.Status, PromotedTurnID: row.PromotedTurnID, TimeCreated: row.TimeCreated, TimeUpdated: row.TimeUpdated,
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
