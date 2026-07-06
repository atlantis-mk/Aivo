package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aivo/core/domain"
)

func (s *Store) migrateSessionRuntime(ctx context.Context) error {
	if err := s.db.WithContext(ctx).AutoMigrate(&turnRow{}, &sessionEventRow{}, &toolCallRow{}, &sessionExecutionStateRow{}, &pendingSessionInputRow{}, &permissionRequestRow{}, &questionRequestRow{}, &sessionSummaryRow{}, &sessionCheckpointRow{}, &codingContextRow{}, &agentRunRow{}, &todoItemRow{}, &scheduledJobRow{}, &pluginInstallRow{}, &pluginDiagnosticRow{}, &mcpServerRow{}, &mcpToolRow{}, &mcpPromptRow{}, &mcpResourceRow{}, &toolRegistrationRow{}); err != nil {
		return err
	}
	return nil
}

func (s *Store) CreateRuntimeSession(ctx context.Context, input domain.CreateSessionRequest) (domain.Session, error) {
	now := domain.NowString(time.Now())
	sessionType, err := domain.NormalizeSessionType(input.Type)
	if err != nil {
		return domain.Session{}, err
	}
	source, err := domain.NormalizeSessionSource(input.Source)
	if err != nil {
		return domain.Session{}, err
	}
	agentModeInput := strings.TrimSpace(input.AgentMode)
	if agentModeInput == "" && sessionType == domain.SessionTypeCoding {
		agentModeInput = domain.AgentModeCode
	}
	agentMode, err := domain.NormalizeAgentMode(agentModeInput)
	if err != nil {
		return domain.Session{}, err
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = fallbackTitle(input.Goal)
	}
	projectPath := normalizeStoredPath(input.ProjectPath)
	session := domain.Session{
		ID: uuid.NewString(), Type: sessionType, Status: domain.SessionStatusActive, Source: source,
		Title: title, Goal: strings.TrimSpace(input.Goal), ProjectPath: projectPath, Model: input.Model,
		ModelSnapshot: strings.TrimSpace(input.ModelSnapshot), SystemPromptSnapshot: strings.TrimSpace(input.SystemPromptSnapshot),
		AgentMode: agentMode, Metadata: input.Metadata, TimeCreated: now, TimeUpdated: now,
	}
	var providerID, modelID string
	if input.Model != nil {
		providerID = input.Model.ProviderID
		modelID = input.Model.ModelID
	}
	projectID, err := s.projectIDForPath(ctx, projectPath)
	if err != nil {
		return domain.Session{}, err
	}
	row := sessionRow{
		ID: session.ID, Title: session.Title, ProjectID: projectID, ModelProviderID: providerID, ModelID: modelID,
		TimeCreated: now, TimeUpdated: now, Type: session.Type, Status: session.Status, Source: session.Source,
		Goal: session.Goal, Summary: "", ModelSnapshot: session.ModelSnapshot, SystemPromptSnapshot: session.SystemPromptSnapshot,
		AgentMode: session.AgentMode, Metadata: encodeStringMap(session.Metadata),
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return domain.Session{}, err
	}
	return session, nil
}

func (s *Store) GetRuntimeSession(ctx context.Context, id string) (domain.Session, error) {
	var row sessionRow
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if err != nil {
		return domain.Session{}, err
	}
	return s.sessionFromRow(ctx, row), nil
}

func (s *Store) ListRuntimeSessions(ctx context.Context, filter domain.ListSessionsRequest) ([]domain.Session, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := s.db.WithContext(ctx).Model(&sessionRow{})
	if !filter.IncludeDeleted {
		q = q.Where("COALESCE(status, ?) <> ?", domain.SessionStatusActive, domain.SessionStatusDeleted)
	}
	if filter.Type != "" {
		q = q.Where("COALESCE(type, ?) = ?", domain.SessionTypeCoding, filter.Type)
	}
	if filter.Status != "" {
		q = q.Where("COALESCE(status, ?) = ?", domain.SessionStatusActive, filter.Status)
	}
	if filter.Source != "" {
		q = q.Where("COALESCE(source, ?) = ?", domain.SessionSourceDesktop, filter.Source)
	}
	projectPath := normalizeStoredPath(filter.ProjectPath)
	if projectPath != "" {
		q = q.Where("EXISTS (SELECT 1 FROM coding_contexts cc WHERE cc.session_id = sessions.id AND cc.project_path = ?)", projectPath)
	}
	search := strings.TrimSpace(filter.Search)
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("(title LIKE ? OR COALESCE(summary, '') LIKE ? OR EXISTS (SELECT 1 FROM coding_contexts cc WHERE cc.session_id = sessions.id AND cc.project_path LIKE ?) OR EXISTS (SELECT 1 FROM session_events e WHERE e.session_id = sessions.id AND e.visibility = ? AND e.content LIKE ?))", like, like, like, domain.EventVisibilityNormal, like)
	}
	var rows []sessionRow
	if err := q.Order("time_updated DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Session, 0, len(rows))
	for _, row := range rows {
		out = append(out, s.sessionFromRow(ctx, row))
	}
	return out, nil
}

func (s *Store) UpdateRuntimeSession(ctx context.Context, input domain.UpdateSessionRequest) (domain.Session, error) {
	current, err := s.GetRuntimeSession(ctx, input.SessionID)
	if err != nil {
		return domain.Session{}, err
	}
	if strings.TrimSpace(input.Title) != "" {
		current.Title = strings.TrimSpace(input.Title)
	}
	if strings.TrimSpace(input.Goal) != "" {
		current.Goal = strings.TrimSpace(input.Goal)
	}
	if strings.TrimSpace(input.Summary) != "" {
		current.Summary = strings.TrimSpace(input.Summary)
	}
	if strings.TrimSpace(input.Status) != "" {
		status, err := domain.NormalizeSessionStatus(input.Status)
		if err != nil {
			return domain.Session{}, err
		}
		current.Status = status
	}
	current.TimeUpdated = domain.NowString(time.Now())
	err = s.db.WithContext(ctx).Model(&sessionRow{}).Where("id = ?", current.ID).Updates(map[string]any{
		"title": current.Title, "goal": current.Goal, "summary": current.Summary, "status": current.Status, "time_updated": current.TimeUpdated,
	}).Error
	if err != nil {
		return domain.Session{}, err
	}
	return current, nil
}

func (s *Store) SetRuntimeSessionStatus(ctx context.Context, id string, status string) (domain.Session, error) {
	nextStatus, err := domain.NormalizeSessionStatus(status)
	if err != nil {
		return domain.Session{}, err
	}
	now := domain.NowString(time.Now())
	updates := map[string]any{"status": nextStatus, "time_updated": now}
	if nextStatus == domain.SessionStatusArchived {
		updates["archived_at"] = now
	}
	if nextStatus == domain.SessionStatusDeleted {
		updates["deleted_at"] = now
	}
	if err := s.db.WithContext(ctx).Model(&sessionRow{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return domain.Session{}, err
	}
	return s.GetRuntimeSession(ctx, id)
}

func (s *Store) SetRuntimeSessionAgentMode(ctx context.Context, id string, mode string) (domain.Session, error) {
	agentMode, err := domain.NormalizeAgentMode(mode)
	if err != nil {
		return domain.Session{}, err
	}
	now := domain.NowString(time.Now())
	if err := s.db.WithContext(ctx).Model(&sessionRow{}).Where("id = ?", strings.TrimSpace(id)).Updates(map[string]any{
		"agent_mode":   agentMode,
		"time_updated": now,
	}).Error; err != nil {
		return domain.Session{}, err
	}
	return s.GetRuntimeSession(ctx, id)
}

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
	if err := s.db.WithContext(ctx).Model(&sessionEventRow{}).Where("id = ?", eventID).Update("content", strings.TrimSpace(input.Content)).Error; err != nil {
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

func (s *Store) CreatePermissionRequest(ctx context.Context, request domain.PermissionRequest) (domain.PermissionRequest, error) {
	now := domain.NowString(time.Now())
	if request.ID == "" {
		request.ID = uuid.NewString()
	}
	if request.Status == "" {
		request.Status = domain.PermissionRequestStatusPending
	}
	if request.TimeCreated == "" {
		request.TimeCreated = now
	}
	request.TimeUpdated = now
	row := permissionRequestRow{
		ID: request.ID, SessionID: request.SessionID, TurnID: request.TurnID, ToolCallID: request.ToolCallID,
		ToolName: request.ToolName, Action: request.Action, Paths: encodeStrings(request.Paths),
		Arguments: encodeAnyMap(request.Arguments), Status: request.Status, Remember: boolInt(request.Remember),
		Reason: request.Reason, TimeCreated: request.TimeCreated, TimeUpdated: request.TimeUpdated,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return domain.PermissionRequest{}, err
	}
	return permissionRequestFromRow(row), nil
}

func (s *Store) GetPermissionRequest(ctx context.Context, requestID string) (domain.PermissionRequest, error) {
	var row permissionRequestRow
	if err := s.db.WithContext(ctx).Where("id = ?", requestID).First(&row).Error; err != nil {
		return domain.PermissionRequest{}, err
	}
	return permissionRequestFromRow(row), nil
}

func (s *Store) ListPermissionRequests(ctx context.Context, sessionID string, status string) ([]domain.PermissionRequest, error) {
	q := s.db.WithContext(ctx).Model(&permissionRequestRow{})
	if strings.TrimSpace(sessionID) != "" {
		q = q.Where("session_id = ?", strings.TrimSpace(sessionID))
	}
	if strings.TrimSpace(status) != "" {
		q = q.Where("status = ?", strings.TrimSpace(status))
	}
	var rows []permissionRequestRow
	if err := q.Order("time_created DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.PermissionRequest, 0, len(rows))
	for _, row := range rows {
		out = append(out, permissionRequestFromRow(row))
	}
	return out, nil
}

func (s *Store) UpdatePermissionRequest(ctx context.Context, requestID string, status string, remember bool, reason string) (domain.PermissionRequest, error) {
	now := domain.NowString(time.Now())
	updates := map[string]any{"status": status, "remember": boolInt(remember), "reason": strings.TrimSpace(reason), "time_updated": now}
	if err := s.db.WithContext(ctx).Model(&permissionRequestRow{}).Where("id = ?", requestID).Updates(updates).Error; err != nil {
		return domain.PermissionRequest{}, err
	}
	return s.GetPermissionRequest(ctx, requestID)
}

func (s *Store) CreateQuestionRequest(ctx context.Context, request domain.QuestionRequest) (domain.QuestionRequest, error) {
	now := domain.NowString(time.Now())
	if request.ID == "" {
		request.ID = uuid.NewString()
	}
	if request.ToolName == "" {
		request.ToolName = "question"
	}
	if request.Status == "" {
		request.Status = domain.QuestionRequestStatusPending
	}
	if request.TimeCreated == "" {
		request.TimeCreated = now
	}
	request.TimeUpdated = now
	row := questionRequestRow{
		ID: request.ID, SessionID: request.SessionID, TurnID: request.TurnID, ToolCallID: request.ToolCallID,
		ToolName: request.ToolName, Questions: encodeQuestionPrompts(request.Questions), Answers: encodeStringMatrix(request.Answers),
		Status: request.Status, Reason: request.Reason, Arguments: encodeAnyMap(request.Arguments),
		TimeCreated: request.TimeCreated, TimeUpdated: request.TimeUpdated,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return domain.QuestionRequest{}, err
	}
	return questionRequestFromRow(row), nil
}

func (s *Store) GetQuestionRequest(ctx context.Context, requestID string) (domain.QuestionRequest, error) {
	var row questionRequestRow
	if err := s.db.WithContext(ctx).Where("id = ?", requestID).First(&row).Error; err != nil {
		return domain.QuestionRequest{}, err
	}
	return questionRequestFromRow(row), nil
}

func (s *Store) ListQuestionRequests(ctx context.Context, sessionID string, status string) ([]domain.QuestionRequest, error) {
	q := s.db.WithContext(ctx).Model(&questionRequestRow{})
	if strings.TrimSpace(sessionID) != "" {
		q = q.Where("session_id = ?", strings.TrimSpace(sessionID))
	}
	if strings.TrimSpace(status) != "" {
		q = q.Where("status = ?", strings.TrimSpace(status))
	}
	var rows []questionRequestRow
	if err := q.Order("time_created DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.QuestionRequest, 0, len(rows))
	for _, row := range rows {
		out = append(out, questionRequestFromRow(row))
	}
	return out, nil
}

func (s *Store) UpdateQuestionRequest(ctx context.Context, requestID string, status string, answers [][]string, reason string) (domain.QuestionRequest, error) {
	now := domain.NowString(time.Now())
	updates := map[string]any{"status": status, "answers": encodeStringMatrix(answers), "reason": strings.TrimSpace(reason), "time_updated": now}
	if err := s.db.WithContext(ctx).Model(&questionRequestRow{}).Where("id = ?", requestID).Updates(updates).Error; err != nil {
		return domain.QuestionRequest{}, err
	}
	return s.GetQuestionRequest(ctx, requestID)
}

func (s *Store) SavePermissionRule(ctx context.Context, rule domain.PermissionRule) (domain.PermissionRule, error) {
	now := domain.NowString(time.Now())
	if rule.ID == "" {
		rule.ID = uuid.NewString()
	}
	if rule.Scope == "" {
		rule.Scope = "workspace"
	}
	if rule.TimeCreated == "" {
		rule.TimeCreated = now
	}
	rule.TimeUpdated = now
	row := permissionRuleRow{
		ID: rule.ID, Scope: rule.Scope, SessionID: rule.SessionID, WorkspaceRoot: normalizeStoredPath(rule.WorkspaceRoot),
		ToolName: rule.ToolName, Action: rule.Action, Decision: rule.Decision, Paths: encodeStrings(rule.Paths),
		TimeCreated: rule.TimeCreated, TimeUpdated: rule.TimeUpdated,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return domain.PermissionRule{}, err
	}
	return permissionRuleFromRow(row), nil
}

func (s *Store) ListPermissionRules(ctx context.Context, workspaceRoot string, sessionID string) ([]domain.PermissionRule, error) {
	q := s.db.WithContext(ctx).Model(&permissionRuleRow{})
	workspaceRoot = normalizeStoredPath(workspaceRoot)
	if workspaceRoot != "" && strings.TrimSpace(sessionID) != "" {
		q = q.Where("workspace_root = ? OR session_id = ?", workspaceRoot, strings.TrimSpace(sessionID))
	} else if workspaceRoot != "" {
		q = q.Where("workspace_root = ?", workspaceRoot)
	} else if strings.TrimSpace(sessionID) != "" {
		q = q.Where("session_id = ?", strings.TrimSpace(sessionID))
	}
	var rows []permissionRuleRow
	if err := q.Order("time_created DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.PermissionRule, 0, len(rows))
	for _, row := range rows {
		out = append(out, permissionRuleFromRow(row))
	}
	return out, nil
}

func permissionRequestFromRow(row permissionRequestRow) domain.PermissionRequest {
	return domain.PermissionRequest{
		ID: row.ID, SessionID: row.SessionID, TurnID: row.TurnID, ToolCallID: row.ToolCallID,
		ToolName: row.ToolName, Action: row.Action, Paths: decodeStrings(row.Paths),
		Arguments: decodeAnyMap(row.Arguments), Status: row.Status, Remember: row.Remember != 0,
		Reason: row.Reason, TimeCreated: row.TimeCreated, TimeUpdated: row.TimeUpdated,
	}
}

func questionRequestFromRow(row questionRequestRow) domain.QuestionRequest {
	return domain.QuestionRequest{
		ID: row.ID, SessionID: row.SessionID, TurnID: row.TurnID, ToolCallID: row.ToolCallID,
		ToolName: row.ToolName, Questions: decodeQuestionPrompts(row.Questions), Answers: decodeStringMatrix(row.Answers),
		Status: row.Status, Reason: row.Reason, Arguments: decodeAnyMap(row.Arguments),
		TimeCreated: row.TimeCreated, TimeUpdated: row.TimeUpdated,
	}
}

func permissionRuleFromRow(row permissionRuleRow) domain.PermissionRule {
	return domain.PermissionRule{
		ID: row.ID, Scope: row.Scope, SessionID: row.SessionID, WorkspaceRoot: row.WorkspaceRoot,
		ToolName: row.ToolName, Action: row.Action, Decision: row.Decision, Paths: decodeStrings(row.Paths),
		TimeCreated: row.TimeCreated, TimeUpdated: row.TimeUpdated,
	}
}

func (s *Store) CreateSummary(ctx context.Context, summary domain.SessionSummary) error {
	if summary.ID == "" {
		summary.ID = uuid.NewString()
	}
	if summary.TimeCreated == "" {
		summary.TimeCreated = domain.NowString(time.Now())
	}
	row := summaryToRow(summary)
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return tx.Model(&sessionRow{}).Where("id = ?", summary.SessionID).Updates(map[string]any{"summary": summary.Summary, "time_updated": summary.TimeCreated}).Error
	})
}

func (s *Store) LatestSummary(ctx context.Context, sessionID string) (*domain.SessionSummary, error) {
	var row sessionSummaryRow
	err := s.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("time_created DESC").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	item := summaryFromRow(row)
	return &item, nil
}

func (s *Store) CreateCheckpoint(ctx context.Context, checkpoint domain.SessionCheckpoint) error {
	if checkpoint.ID == "" {
		checkpoint.ID = uuid.NewString()
	}
	if checkpoint.TimeCreated == "" {
		checkpoint.TimeCreated = domain.NowString(time.Now())
	}
	row := checkpointToRow(checkpoint)
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return tx.Model(&sessionRow{}).Where("id = ?", checkpoint.SessionID).Update("time_updated", checkpoint.TimeCreated).Error
	})
}

func (s *Store) ListCheckpoints(ctx context.Context, sessionID string, limit int) ([]domain.SessionCheckpoint, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []sessionCheckpointRow
	if err := s.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("time_created DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.SessionCheckpoint, 0, len(rows))
	for _, row := range rows {
		out = append(out, checkpointFromRow(row))
	}
	return out, nil
}

func (s *Store) LatestCheckpoint(ctx context.Context, sessionID string) (*domain.SessionCheckpoint, error) {
	items, err := s.ListCheckpoints(ctx, sessionID, 1)
	if err != nil || len(items) == 0 {
		return nil, err
	}
	return &items[0], nil
}

func (s *Store) UpsertCodingContext(ctx context.Context, cc domain.CodingContext) (domain.CodingContext, error) {
	now := domain.NowString(time.Now())
	if cc.ID == "" {
		cc.ID = uuid.NewString()
	}
	if cc.TimeCreated == "" {
		cc.TimeCreated = now
	}
	cc.TimeUpdated = now
	cc.ProjectPath = normalizeStoredPath(cc.ProjectPath)
	row := codingContextToRow(cc)
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "session_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"project_path": row.ProjectPath, "git_branch": row.GitBranch, "commit_sha": row.CommitSHA, "repo_url": row.RepoURL,
			"changed_files": row.ChangedFiles, "language_stack": row.LanguageStack, "package_manager": row.PackageManager,
			"cwd": row.CWD, "permissions": row.Permissions, "last_command": row.LastCommand, "time_updated": row.TimeUpdated,
		}),
	}).Create(&row).Error
	if err != nil {
		return domain.CodingContext{}, err
	}
	return s.GetCodingContext(ctx, cc.SessionID)
}

func (s *Store) GetCodingContext(ctx context.Context, sessionID string) (domain.CodingContext, error) {
	var row codingContextRow
	if err := s.db.WithContext(ctx).Where("session_id = ?", sessionID).First(&row).Error; err != nil {
		return domain.CodingContext{}, err
	}
	return codingContextFromRow(row), nil
}

func (s *Store) LatestSessionByProject(ctx context.Context, projectPath string) (*domain.Session, error) {
	items, err := s.ListRuntimeSessions(ctx, domain.ListSessionsRequest{Type: domain.SessionTypeCoding, ProjectPath: projectPath, Status: domain.SessionStatusActive, Limit: 1})
	if err != nil || len(items) == 0 {
		return nil, err
	}
	return &items[0], nil
}

func (s *Store) ForkRuntimeSession(ctx context.Context, source domain.Session, input domain.ForkSessionRequest) (domain.Session, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = source.Title + " (fork)"
	}
	goal := strings.TrimSpace(input.Goal)
	if goal == "" {
		goal = source.Goal
	}
	fork, err := s.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: source.Type, Source: source.Source, Title: title, Goal: goal, ProjectPath: source.ProjectPath, Model: source.Model, ModelSnapshot: source.ModelSnapshot, SystemPromptSnapshot: source.SystemPromptSnapshot, AgentMode: source.AgentMode, Metadata: source.Metadata})
	if err != nil {
		return domain.Session{}, err
	}
	if err := s.db.WithContext(ctx).Model(&sessionRow{}).Where("id = ?", fork.ID).Updates(map[string]any{"parent_session_id": source.ID, "forked_from_session_id": source.ID}).Error; err != nil {
		return domain.Session{}, err
	}
	if cc, err := s.GetCodingContext(ctx, source.ID); err == nil {
		cc.ID = uuid.NewString()
		cc.SessionID = fork.ID
		cc.TimeCreated = ""
		_, _ = s.UpsertCodingContext(ctx, cc)
	}
	if latest, err := s.LatestSummary(ctx, source.ID); err == nil && latest != nil {
		latest.ID = uuid.NewString()
		latest.SessionID = fork.ID
		latest.TimeCreated = ""
		_ = s.CreateSummary(ctx, *latest)
	}
	return s.GetRuntimeSession(ctx, fork.ID)
}

func (s *Store) projectIDForPath(ctx context.Context, projectPath string) (string, error) {
	if projectPath == "" {
		return "", nil
	}
	project, err := s.UpsertProject(ctx, projectPath)
	if err != nil {
		return "", err
	}
	return project.ID, nil
}

func (s *Store) sessionFromRow(ctx context.Context, row sessionRow) domain.Session {
	item := domain.Session{
		ID: row.ID, Type: defaultString(row.Type, domain.SessionTypeCoding), Status: defaultString(row.Status, domain.SessionStatusActive), Source: defaultString(row.Source, domain.SessionSourceDesktop),
		Title: row.Title, Goal: row.Goal, Summary: row.Summary, ParentSessionID: row.ParentSessionID, ForkedFromSessionID: row.ForkedFromSessionID,
		ModelSnapshot: row.ModelSnapshot, SystemPromptSnapshot: row.SystemPromptSnapshot, AgentMode: persistedAgentMode(row.AgentMode), TokenCount: row.TokenCount, CostMicros: row.CostMicros,
		Metadata: decodeStringMap(row.Metadata), ArchivedAt: row.ArchivedAt, DeletedAt: row.DeletedAt, TimeCreated: row.TimeCreated, TimeUpdated: row.TimeUpdated,
	}
	if row.ModelProviderID != "" && row.ModelID != "" {
		item.Model = &domain.ModelRef{ProviderID: row.ModelProviderID, ModelID: row.ModelID}
	}
	if row.ProjectID != "" {
		var project projectRow
		if err := s.db.WithContext(ctx).Where("id = ?", row.ProjectID).First(&project).Error; err == nil {
			item.ProjectPath = project.RootPath
		}
	}
	return item
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func persistedAgentMode(value string) string {
	mode, err := domain.NormalizeAgentMode(value)
	if err != nil {
		return domain.AgentModeAssistant
	}
	return mode
}

func optionalPersistedAgentMode(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return persistedAgentMode(value)
}

func persistedWorkerMode(value string) string {
	if strings.TrimSpace(value) == "" {
		return domain.AgentModeSchedulerWorker
	}
	mode, err := domain.NormalizeAgentMode(value)
	if err != nil {
		return domain.AgentModeSchedulerWorker
	}
	return mode
}

func summaryToRow(summary domain.SessionSummary) sessionSummaryRow {
	return sessionSummaryRow{ID: summary.ID, SessionID: summary.SessionID, FromEventID: summary.FromEventID, ToEventID: summary.ToEventID, Summary: summary.Summary, Facts: encodeStrings(summary.Facts), Decisions: encodeStrings(summary.Decisions), OpenTasks: encodeStrings(summary.OpenTasks), ChangedFiles: encodeStrings(summary.ChangedFiles), NextSuggestedAction: summary.NextSuggestedAction, TimeCreated: summary.TimeCreated}
}

func summaryFromRow(row sessionSummaryRow) domain.SessionSummary {
	return domain.SessionSummary{ID: row.ID, SessionID: row.SessionID, FromEventID: row.FromEventID, ToEventID: row.ToEventID, Summary: row.Summary, Facts: decodeStrings(row.Facts), Decisions: decodeStrings(row.Decisions), OpenTasks: decodeStrings(row.OpenTasks), ChangedFiles: decodeStrings(row.ChangedFiles), NextSuggestedAction: row.NextSuggestedAction, TimeCreated: row.TimeCreated}
}

func checkpointToRow(checkpoint domain.SessionCheckpoint) sessionCheckpointRow {
	return sessionCheckpointRow{ID: checkpoint.ID, SessionID: checkpoint.SessionID, Branch: checkpoint.Branch, CommitSHA: checkpoint.CommitSHA, ChangedFiles: encodeStrings(checkpoint.ChangedFiles), DiffSummary: checkpoint.DiffSummary, ConversationSummary: checkpoint.ConversationSummary, OpenTodos: encodeStrings(checkpoint.OpenTodos), KnownIssues: encodeStrings(checkpoint.KnownIssues), NextSuggestedAction: checkpoint.NextSuggestedAction, TimeCreated: checkpoint.TimeCreated}
}

func checkpointFromRow(row sessionCheckpointRow) domain.SessionCheckpoint {
	return domain.SessionCheckpoint{ID: row.ID, SessionID: row.SessionID, Branch: row.Branch, CommitSHA: row.CommitSHA, ChangedFiles: decodeStrings(row.ChangedFiles), DiffSummary: row.DiffSummary, ConversationSummary: row.ConversationSummary, OpenTodos: decodeStrings(row.OpenTodos), KnownIssues: decodeStrings(row.KnownIssues), NextSuggestedAction: row.NextSuggestedAction, TimeCreated: row.TimeCreated}
}

func codingContextToRow(cc domain.CodingContext) codingContextRow {
	return codingContextRow{ID: cc.ID, SessionID: cc.SessionID, ProjectPath: cc.ProjectPath, GitBranch: cc.GitBranch, CommitSHA: cc.CommitSHA, RepoURL: cc.RepoURL, ChangedFiles: encodeStrings(cc.ChangedFiles), LanguageStack: encodeStrings(cc.LanguageStack), PackageManager: cc.PackageManager, CWD: cc.CWD, Permissions: encodeStrings(cc.Permissions), LastCommand: cc.LastCommand, TimeCreated: cc.TimeCreated, TimeUpdated: cc.TimeUpdated}
}

func codingContextFromRow(row codingContextRow) domain.CodingContext {
	return domain.CodingContext{ID: row.ID, SessionID: row.SessionID, ProjectPath: row.ProjectPath, GitBranch: row.GitBranch, CommitSHA: row.CommitSHA, RepoURL: row.RepoURL, ChangedFiles: decodeStrings(row.ChangedFiles), LanguageStack: decodeStrings(row.LanguageStack), PackageManager: row.PackageManager, CWD: row.CWD, Permissions: decodeStrings(row.Permissions), LastCommand: row.LastCommand, TimeCreated: row.TimeCreated, TimeUpdated: row.TimeUpdated}
}

func (s *Store) SaveAgentRun(ctx context.Context, run domain.AgentRun) (domain.AgentRun, error) {
	now := domain.NowString(time.Now())
	if run.ID == "" {
		run.ID = uuid.NewString()
	}
	mode, err := domain.NormalizeAgentMode(run.Mode)
	if err != nil {
		return domain.AgentRun{}, err
	}
	status, err := domain.NormalizeAgentRunStatus(run.Status)
	if err != nil {
		return domain.AgentRun{}, err
	}
	if run.TimeCreated == "" {
		run.TimeCreated = now
	}
	run.Mode = mode
	run.Status = status
	run.TimeUpdated = now
	if status != domain.AgentRunStatusRunning && run.TimeCompleted == "" {
		run.TimeCompleted = now
	}
	row := agentRunToRow(run)
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error; err != nil {
		return domain.AgentRun{}, err
	}
	return agentRunFromRow(row), nil
}

func (s *Store) ListAgentRuns(ctx context.Context, input domain.AgentRunListRequest) ([]domain.AgentRun, error) {
	limit := input.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := s.db.WithContext(ctx).Model(&agentRunRow{})
	if strings.TrimSpace(input.SessionID) != "" {
		sessionID := strings.TrimSpace(input.SessionID)
		q = q.Where("session_id = ? OR parent_session_id = ?", sessionID, sessionID)
	}
	if strings.TrimSpace(input.Status) != "" {
		q = q.Where("status = ?", strings.TrimSpace(input.Status))
	}
	var rows []agentRunRow
	if err := q.Order("time_created DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.AgentRun, 0, len(rows))
	for _, row := range rows {
		out = append(out, agentRunFromRow(row))
	}
	return out, nil
}

func (s *Store) GetAgentRun(ctx context.Context, id string) (domain.AgentRun, error) {
	var row agentRunRow
	if err := s.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).First(&row).Error; err != nil {
		return domain.AgentRun{}, err
	}
	return agentRunFromRow(row), nil
}

func (s *Store) ReplaceTodoItems(ctx context.Context, input domain.TodoListInput, items []domain.TodoItem) ([]domain.TodoItem, error) {
	now := domain.NowString(time.Now())
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return nil, errors.New("session_id is required")
	}
	projectPath := normalizeStoredPath(input.ProjectPath)
	out := make([]domain.TodoItem, 0, len(items))
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q := tx.Where("session_id = ?", sessionID)
		if projectPath != "" {
			q = q.Where("project_path = ?", projectPath)
		}
		if err := q.Delete(&todoItemRow{}).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		rows := make([]todoItemRow, 0, len(items))
		for index, item := range items {
			if item.ID == "" {
				item.ID = uuid.NewString()
			}
			status, err := domain.NormalizeTodoStatus(item.Status)
			if err != nil {
				return err
			}
			item.SessionID = sessionID
			item.ProjectPath = projectPath
			item.Status = status
			item.Position = index
			if item.TimeCreated == "" {
				item.TimeCreated = now
			}
			item.TimeUpdated = now
			row := todoItemToRow(item)
			rows = append(rows, row)
			out = append(out, todoItemFromRow(row))
		}
		return tx.Create(&rows).Error
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) ListTodoItems(ctx context.Context, input domain.TodoListInput) ([]domain.TodoItem, error) {
	limit := input.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := s.db.WithContext(ctx).Model(&todoItemRow{})
	if strings.TrimSpace(input.SessionID) != "" {
		q = q.Where("session_id = ?", strings.TrimSpace(input.SessionID))
	}
	if projectPath := normalizeStoredPath(input.ProjectPath); projectPath != "" {
		q = q.Where("project_path = ?", projectPath)
	}
	if strings.TrimSpace(input.Status) != "" {
		q = q.Where("status = ?", strings.TrimSpace(input.Status))
	}
	var rows []todoItemRow
	if err := q.Order("position ASC, time_updated DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.TodoItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, todoItemFromRow(row))
	}
	return out, nil
}

func (s *Store) SaveScheduledJob(ctx context.Context, job domain.ScheduledJob) (domain.ScheduledJob, error) {
	now := domain.NowString(time.Now())
	if job.ID == "" {
		job.ID = uuid.NewString()
	}
	status, err := domain.NormalizeScheduledJobStatus(job.Status)
	if err != nil {
		return domain.ScheduledJob{}, err
	}
	workerMode := strings.TrimSpace(job.WorkerMode)
	if workerMode == "" {
		workerMode = domain.AgentModeSchedulerWorker
	}
	mode, err := domain.NormalizeAgentMode(workerMode)
	if err != nil {
		return domain.ScheduledJob{}, err
	}
	job.Status = status
	job.WorkerMode = mode
	if job.TimeCreated == "" {
		job.TimeCreated = now
	}
	job.TimeUpdated = now
	row := scheduledJobToRow(job)
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error; err != nil {
		return domain.ScheduledJob{}, err
	}
	return scheduledJobFromRow(row), nil
}

func (s *Store) GetScheduledJob(ctx context.Context, id string) (domain.ScheduledJob, error) {
	var row scheduledJobRow
	if err := s.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).First(&row).Error; err != nil {
		return domain.ScheduledJob{}, err
	}
	return scheduledJobFromRow(row), nil
}

func (s *Store) ListScheduledJobs(ctx context.Context, input domain.ScheduledJobListInput) ([]domain.ScheduledJob, error) {
	limit := input.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := s.db.WithContext(ctx).Model(&scheduledJobRow{})
	if strings.TrimSpace(input.SessionID) != "" {
		q = q.Where("session_id = ?", strings.TrimSpace(input.SessionID))
	}
	if strings.TrimSpace(input.Status) != "" {
		q = q.Where("status = ?", strings.TrimSpace(input.Status))
	}
	var rows []scheduledJobRow
	if err := q.Order("time_updated DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ScheduledJob, 0, len(rows))
	for _, row := range rows {
		out = append(out, scheduledJobFromRow(row))
	}
	return out, nil
}

func (s *Store) ListDueScheduledJobs(ctx context.Context, now string, limit int) ([]domain.ScheduledJob, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	var rows []scheduledJobRow
	err := s.db.WithContext(ctx).
		Where("status = ? AND next_run_at <> '' AND next_run_at <= ?", domain.ScheduledJobStatusActive, now).
		Order("next_run_at ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.ScheduledJob, 0, len(rows))
	for _, row := range rows {
		out = append(out, scheduledJobFromRow(row))
	}
	return out, nil
}

func (s *Store) DeleteScheduledJob(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&scheduledJobRow{ID: strings.TrimSpace(id)}).Error
}

func agentRunToRow(run domain.AgentRun) agentRunRow {
	return agentRunRow{ID: run.ID, ParentSessionID: run.ParentSessionID, SessionID: run.SessionID, Mode: run.Mode, Status: run.Status, Prompt: run.Prompt, Result: run.Result, Error: run.Error, Metadata: encodeStringMap(run.Metadata), TimeCreated: run.TimeCreated, TimeUpdated: run.TimeUpdated, TimeCompleted: run.TimeCompleted}
}

func agentRunFromRow(row agentRunRow) domain.AgentRun {
	return domain.AgentRun{ID: row.ID, ParentSessionID: row.ParentSessionID, SessionID: row.SessionID, Mode: persistedAgentMode(row.Mode), Status: row.Status, Prompt: row.Prompt, Result: row.Result, Error: row.Error, Metadata: decodeStringMap(row.Metadata), TimeCreated: row.TimeCreated, TimeUpdated: row.TimeUpdated, TimeCompleted: row.TimeCompleted}
}

func todoItemToRow(item domain.TodoItem) todoItemRow {
	return todoItemRow{ID: item.ID, SessionID: item.SessionID, ProjectPath: item.ProjectPath, Title: item.Title, Status: item.Status, Position: item.Position, OwnerMode: item.OwnerMode, SourceEventID: item.SourceEventID, Metadata: encodeStringMap(item.Metadata), TimeCreated: item.TimeCreated, TimeUpdated: item.TimeUpdated}
}

func todoItemFromRow(row todoItemRow) domain.TodoItem {
	return domain.TodoItem{ID: row.ID, SessionID: row.SessionID, ProjectPath: row.ProjectPath, Title: row.Title, Status: row.Status, Position: row.Position, OwnerMode: optionalPersistedAgentMode(row.OwnerMode), SourceEventID: row.SourceEventID, Metadata: decodeStringMap(row.Metadata), TimeCreated: row.TimeCreated, TimeUpdated: row.TimeUpdated}
}

func scheduledJobToRow(job domain.ScheduledJob) scheduledJobRow {
	return scheduledJobRow{ID: job.ID, SessionID: job.SessionID, Title: job.Title, Prompt: job.Prompt, Schedule: job.Schedule, WorkerMode: job.WorkerMode, Toolsets: encodeStrings(job.Toolsets), PermissionScope: job.PermissionScope, Status: job.Status, NextRunAt: job.NextRunAt, LastRunAt: job.LastRunAt, LastResult: job.LastResult, LastError: job.LastError, Metadata: encodeStringMap(job.Metadata), TimeCreated: job.TimeCreated, TimeUpdated: job.TimeUpdated}
}

func scheduledJobFromRow(row scheduledJobRow) domain.ScheduledJob {
	return domain.ScheduledJob{ID: row.ID, SessionID: row.SessionID, Title: row.Title, Prompt: row.Prompt, Schedule: row.Schedule, WorkerMode: persistedWorkerMode(row.WorkerMode), Toolsets: decodeStrings(row.Toolsets), PermissionScope: row.PermissionScope, Status: row.Status, NextRunAt: row.NextRunAt, LastRunAt: row.LastRunAt, LastResult: row.LastResult, LastError: row.LastError, Metadata: decodeStringMap(row.Metadata), TimeCreated: row.TimeCreated, TimeUpdated: row.TimeUpdated}
}

func normalizeStoredPath(path string) string {
	clean := strings.TrimSpace(path)
	if clean == "" {
		return ""
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return clean
	}
	return abs
}

func fallbackTitle(text string) string {
	clean := strings.TrimSpace(text)
	if clean == "" {
		return "Untitled session"
	}
	if len(clean) > 80 {
		return clean[:80]
	}
	return clean
}

func encodeStrings(values []string) string {
	if len(values) == 0 {
		return ""
	}
	raw, _ := json.Marshal(values)
	return string(raw)
}

func decodeStrings(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func encodeStringMatrix(values [][]string) string {
	if len(values) == 0 {
		return ""
	}
	raw, _ := json.Marshal(values)
	return string(raw)
}

func decodeStringMatrix(raw string) [][]string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out [][]string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func encodeQuestionPrompts(values []domain.QuestionPrompt) string {
	if len(values) == 0 {
		return "[]"
	}
	raw, _ := json.Marshal(values)
	return string(raw)
}

func decodeQuestionPrompts(raw string) []domain.QuestionPrompt {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []domain.QuestionPrompt
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func encodeAnyMap(values map[string]any) string {
	if len(values) == 0 {
		return ""
	}
	raw, _ := json.Marshal(values)
	return string(raw)
}

func decodeAnyMap(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func mergeAnyMetadata(base map[string]any, overlay map[string]any) map[string]any {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	merged := make(map[string]any, len(base)+len(overlay))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range overlay {
		merged[key] = value
	}
	return merged
}

func encodeStringMap(values map[string]string) string {
	if len(values) == 0 {
		return ""
	}
	raw, _ := json.Marshal(values)
	return string(raw)
}

func decodeStringMap(raw string) map[string]string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out map[string]string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
