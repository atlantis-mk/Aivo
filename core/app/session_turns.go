package app

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"aivo/core/domain"
)

func (s *Service) StartTurn(ctx context.Context, input domain.StartTurnRequest) (domain.Turn, error) {
	if strings.TrimSpace(input.SessionID) == "" {
		return domain.Turn{}, errors.New("sessionId is required")
	}
	agentMode := strings.TrimSpace(input.AgentMode)
	if agentMode == "" {
		if session, err := s.store.GetRuntimeSession(ctx, input.SessionID); err == nil {
			agentMode = session.AgentMode
		}
	}
	mode, err := domain.NormalizeAgentMode(agentMode)
	if err != nil {
		return domain.Turn{}, err
	}
	now := domain.NowString(s.now())
	turn := domain.Turn{ID: uuid.NewString(), SessionID: input.SessionID, UserEventID: input.UserEventID, AgentMode: mode, Status: domain.TurnStatusRunning, TimeCreated: now, TimeUpdated: now}
	if err := s.store.StartTurn(ctx, turn); err != nil {
		return domain.Turn{}, err
	}
	if s.onTurnUpdated != nil {
		s.onTurnUpdated(turn.SessionID, turn)
	}
	return turn, nil
}

func (s *Service) CompleteTurn(ctx context.Context, input domain.CompleteTurnRequest) (domain.Turn, error) {
	turn, err := s.store.UpdateTurnStatus(ctx, input.TurnID, domain.TurnStatusCompleted, "")
	if err == nil && s.onTurnUpdated != nil {
		s.onTurnUpdated(turn.SessionID, turn)
	}
	return turn, err
}

func (s *Service) FailTurn(ctx context.Context, input domain.FailTurnRequest) (domain.Turn, error) {
	turn, err := s.store.UpdateTurnStatus(ctx, input.TurnID, domain.TurnStatusFailed, strings.TrimSpace(input.Error))
	if err == nil {
		_, _ = s.AppendEvent(ctx, domain.AppendEventRequest{SessionID: turn.SessionID, TurnID: turn.ID, Type: domain.EventTypeError, Role: domain.EventRoleSystem, Visibility: domain.EventVisibilityNormal, Content: strings.TrimSpace(input.Error)})
		if s.onTurnUpdated != nil {
			s.onTurnUpdated(turn.SessionID, turn)
		}
	}
	return turn, err
}

func (s *Service) CancelTurn(ctx context.Context, input domain.CancelTurnRequest) (domain.Turn, error) {
	s.cancelActiveTurn(input.TurnID)
	turn, err := s.store.UpdateTurnStatus(ctx, input.TurnID, domain.TurnStatusCancelled, strings.TrimSpace(input.Reason))
	if err == nil {
		content := strings.TrimSpace(input.Reason)
		if content == "" {
			content = "Turn cancelled"
		}
		cleanupErr := s.cleanupCancelledTurn(ctx, turn, content)
		_, _ = s.AppendEvent(ctx, domain.AppendEventRequest{SessionID: turn.SessionID, TurnID: turn.ID, Type: domain.EventTypeSystemNote, Role: domain.EventRoleSystem, Visibility: domain.EventVisibilityNormal, Content: content})
		if s.onTurnUpdated != nil {
			s.onTurnUpdated(turn.SessionID, turn)
		}
		if cleanupErr != nil {
			return turn, cleanupErr
		}
	}
	return turn, err
}

func (s *Service) cleanupCancelledTurn(ctx context.Context, turn domain.Turn, reason string) error {
	if s == nil || s.store == nil || strings.TrimSpace(turn.ID) == "" || strings.TrimSpace(turn.SessionID) == "" {
		return nil
	}
	if ctx == nil || ctx.Err() != nil {
		ctx = context.Background()
	}
	reason = firstNonEmpty(strings.TrimSpace(reason), "Turn cancelled")
	var cleanupErr error
	permissions, err := s.store.ListPermissionRequests(ctx, turn.SessionID, domain.PermissionRequestStatusPending)
	if err != nil {
		cleanupErr = errors.Join(cleanupErr, err)
	} else {
		for _, request := range permissions {
			if request.TurnID != turn.ID {
				continue
			}
			_, err := s.DenyPermissionRequest(ctx, domain.DenyPermissionRequestInput{RequestID: request.ID, Reason: reason})
			cleanupErr = errors.Join(cleanupErr, err)
		}
	}
	questions, err := s.store.ListQuestionRequests(ctx, turn.SessionID, domain.QuestionRequestStatusPending)
	if err != nil {
		cleanupErr = errors.Join(cleanupErr, err)
	} else {
		for _, request := range questions {
			if request.TurnID != turn.ID {
				continue
			}
			_, err := s.RejectQuestionRequest(ctx, domain.RejectQuestionRequestInput{RequestID: request.ID, Reason: reason})
			cleanupErr = errors.Join(cleanupErr, err)
		}
	}
	calls, err := s.store.ListToolCalls(ctx, turn.SessionID)
	if err != nil {
		cleanupErr = errors.Join(cleanupErr, err)
	} else {
		for _, call := range calls {
			if call.TurnID != turn.ID || (call.Status != domain.ToolCallStatusRunning && call.Status != domain.ToolCallStatusPending) {
				continue
			}
			call.Status = domain.ToolCallStatusInterrupted
			call.Error = reason
			call.ResultSummary = reason
			if call.Result == nil {
				call.Result = map[string]any{}
			}
			call.Result["ok"] = false
			call.Result["cancelled"] = true
			call.Result["error"] = reason
			if err := s.store.SaveToolCall(ctx, call); err != nil {
				cleanupErr = errors.Join(cleanupErr, err)
				continue
			}
			if s.onToolCallUpdated != nil {
				s.onToolCallUpdated(call.SessionID, call.TurnID, call, false)
			}
		}
	}
	if state, err := s.store.GetSessionExecutionState(ctx, turn.SessionID); err == nil && state.TurnID == turn.ID {
		_, err = s.store.UpsertSessionExecutionState(ctx, domain.SessionExecutionState{
			SessionID: turn.SessionID, TurnID: turn.ID, Status: domain.ExecutionStatusInterrupted, Reason: reason,
		})
		cleanupErr = errors.Join(cleanupErr, err)
	}
	return cleanupErr
}

func (s *Service) RetrySessionTurnStreaming(ctx context.Context, input domain.RetrySessionTurnRequest) (domain.PreparedSessionTurn, error) {
	turnID := strings.TrimSpace(input.TurnID)
	if turnID == "" {
		return domain.PreparedSessionTurn{}, errors.New("turnId is required")
	}
	turn, err := s.store.GetTurn(ctx, turnID)
	if err != nil {
		return domain.PreparedSessionTurn{}, err
	}
	if strings.TrimSpace(input.SessionID) != "" && strings.TrimSpace(input.SessionID) != turn.SessionID {
		return domain.PreparedSessionTurn{}, errors.New("turn does not belong to session")
	}
	if strings.TrimSpace(turn.UserEventID) == "" {
		return domain.PreparedSessionTurn{}, errors.New("turn has no user message to retry")
	}
	userEvent, err := s.store.GetSessionEvent(ctx, turn.UserEventID)
	if err != nil {
		return domain.PreparedSessionTurn{}, err
	}
	if userEvent.Type != domain.EventTypeUserMessage || strings.TrimSpace(userEvent.Content) == "" {
		return domain.PreparedSessionTurn{}, errors.New("turn user message is not retryable")
	}
	s.cancelActiveTurn(turn.ID)
	_, _ = s.store.SetSessionEventVisibility(ctx, userEvent.ID, domain.EventVisibilityHidden)
	_ = s.store.HideSessionTurnEvents(ctx, turn.ID)
	if turn.Status == domain.TurnStatusRunning {
		turn, _ = s.store.UpdateTurnStatus(ctx, turn.ID, domain.TurnStatusCancelled, "Retried")
	}
	_ = s.cleanupCancelledTurn(ctx, turn, "Retried")
	if s.onSessionUpdated != nil {
		s.onSessionUpdated(turn.SessionID, nil)
	}
	return s.SubmitSessionMessageStreaming(ctx, domain.SubmitSessionMessageRequest{
		SessionID:       turn.SessionID,
		Text:            userEvent.Content,
		Model:           input.Model,
		AgentMode:       firstNonEmpty(input.AgentMode, turn.AgentMode),
		Toolsets:        input.Toolsets,
		PermissionScope: input.PermissionScope,
		ReasoningEffort: input.ReasoningEffort,
		ServiceTier:     input.ServiceTier,
	})
}

func (s *Service) registerActiveTurn(id string, cancel context.CancelFunc) {
	id = strings.TrimSpace(id)
	if s == nil || id == "" || cancel == nil {
		return
	}
	s.activeTurnMu.Lock()
	defer s.activeTurnMu.Unlock()
	if s.activeTurnCancel == nil {
		s.activeTurnCancel = map[string]context.CancelFunc{}
	}
	s.activeTurnCancel[id] = cancel
}

func (s *Service) unregisterActiveTurn(id string) {
	id = strings.TrimSpace(id)
	if s == nil || id == "" {
		return
	}
	s.activeTurnMu.Lock()
	defer s.activeTurnMu.Unlock()
	delete(s.activeTurnCancel, id)
}

func (s *Service) cancelActiveTurn(id string) {
	id = strings.TrimSpace(id)
	if s == nil || id == "" {
		return
	}
	s.activeTurnMu.Lock()
	cancel := s.activeTurnCancel[id]
	s.activeTurnMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *Service) ListTurns(ctx context.Context, sessionID string, limit int) ([]domain.Turn, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("sessionId is required")
	}
	return s.store.ListTurns(ctx, sessionID, limit)
}
