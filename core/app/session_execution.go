package app

import (
	"context"
	"errors"
	"strings"

	"aivo/core/domain"
)

func (s *Service) GetSessionExecutionState(ctx context.Context, sessionID string) (domain.SessionExecutionState, error) {
	if strings.TrimSpace(sessionID) == "" {
		return domain.SessionExecutionState{}, errors.New("sessionId is required")
	}
	return s.store.GetSessionExecutionState(ctx, sessionID)
}

func (s *Service) InterruptSessionExecution(ctx context.Context, input domain.InterruptSessionExecutionInput) (domain.SessionExecutionState, error) {
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return domain.SessionExecutionState{}, errors.New("sessionId is required")
	}
	turns, _ := s.store.ListTurns(ctx, sessionID, 20)
	var runningTurnID string
	for _, turn := range turns {
		if turn.Status == domain.TurnStatusRunning {
			runningTurnID = turn.ID
			s.cancelActiveTurn(turn.ID)
			_, _ = s.store.UpdateTurnStatus(ctx, turn.ID, domain.TurnStatusCancelled, firstNonEmpty(input.Reason, "Interrupted by user"))
			break
		}
	}
	_, _ = s.store.MarkRunningToolCallsInterrupted(ctx, sessionID, firstNonEmpty(input.Reason, "Interrupted by user"))
	event, _ := s.AppendEvent(ctx, domain.AppendEventRequest{
		SessionID: sessionID, TurnID: runningTurnID, Type: domain.EventTypeSystemNote, Role: domain.EventRoleSystem,
		Visibility: domain.EventVisibilityNormal, Content: firstNonEmpty(input.Reason, "Session execution interrupted"),
		Payload: map[string]any{"kind": "execution_interrupted"},
	})
	state := domain.SessionExecutionState{
		SessionID: sessionID, TurnID: runningTurnID, Status: domain.ExecutionStatusInterrupted,
		Reason: firstNonEmpty(input.Reason, "interrupted"), LastEventID: event.ID,
		Metadata: map[string]any{"interruptedToolCalls": true},
	}
	if updated, err := s.store.UpsertSessionExecutionState(ctx, state); err == nil {
		if s.onSessionUpdated != nil {
			s.onSessionUpdated(sessionID, nil)
		}
		return updated, nil
	}
	return state, nil
}

func (s *Service) ResumeSessionExecution(ctx context.Context, input domain.ResumeSessionExecutionInput) (domain.SessionExecutionState, error) {
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return domain.SessionExecutionState{}, errors.New("sessionId is required")
	}
	_, _ = s.store.MarkRunningToolCallsInterrupted(ctx, sessionID, "Interrupted before resume; not replayed automatically")
	pending, _ := s.store.ListPendingSessionInputs(ctx, sessionID, domain.PendingInputStatusPending)
	pendingIDs := make([]string, 0, len(pending))
	for _, item := range pending {
		pendingIDs = append(pendingIDs, item.ID)
	}
	status := domain.ExecutionStatusIdle
	reason := "ready"
	if len(pending) > 0 {
		status = domain.ExecutionStatusRunning
		reason = "pending queued input is ready to continue"
	}
	event, _ := s.AppendEvent(ctx, domain.AppendEventRequest{
		SessionID: sessionID, Type: domain.EventTypeSystemNote, Role: domain.EventRoleSystem,
		Visibility: domain.EventVisibilityNormal, Content: "Session execution resumed",
		Payload: map[string]any{"kind": "execution_resumed", "pendingInputCount": len(pending)},
	})
	state, err := s.store.UpsertSessionExecutionState(ctx, domain.SessionExecutionState{
		SessionID: sessionID, Status: status, Reason: reason, LastEventID: event.ID, PendingInputIDs: pendingIDs,
	})
	if err == nil && s.onSessionUpdated != nil {
		s.onSessionUpdated(sessionID, nil)
	}
	return state, err
}
