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
		_, _ = s.AppendEvent(ctx, domain.AppendEventRequest{SessionID: turn.SessionID, TurnID: turn.ID, Type: domain.EventTypeSystemNote, Role: domain.EventRoleSystem, Visibility: domain.EventVisibilityNormal, Content: content})
		if s.onTurnUpdated != nil {
			s.onTurnUpdated(turn.SessionID, turn)
		}
	}
	return turn, err
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
		_, _ = s.store.UpdateTurnStatus(ctx, turn.ID, domain.TurnStatusCancelled, "Retried")
	}
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
