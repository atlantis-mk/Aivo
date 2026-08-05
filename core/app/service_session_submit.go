package app

import (
	"context"
	"errors"
	"strings"

	"aivo/core/domain"
)

func (s *Service) SubmitSessionMessage(ctx context.Context, input domain.SubmitSessionMessageRequest) (domain.PreparedSessionTurn, error) {
	return s.submitSessionMessage(ctx, input, nil)
}

func (s *Service) SubmitSessionMessageStreaming(ctx context.Context, input domain.SubmitSessionMessageRequest) (domain.PreparedSessionTurn, error) {
	return s.submitSessionMessage(ctx, input, func(ctx context.Context, prepared domain.PreparedSessionTurn, work func(context.Context) (domain.PreparedSessionTurn, error)) (domain.PreparedSessionTurn, error) {
		go func() {
			turnCtx, cancel := context.WithCancel(context.Background())
			s.registerActiveTurn(prepared.Turn.ID, cancel)
			defer s.unregisterActiveTurn(prepared.Turn.ID)
			defer cancel()
			_, _ = work(turnCtx)
			if s.onSessionUpdated != nil {
				s.onSessionUpdated(prepared.Turn.SessionID, nil)
			}
		}()
		return prepared, nil
	})
}

func (s *Service) submitSessionMessage(
	ctx context.Context,
	input domain.SubmitSessionMessageRequest,
	async func(context.Context, domain.PreparedSessionTurn, func(context.Context) (domain.PreparedSessionTurn, error)) (domain.PreparedSessionTurn, error),
) (domain.PreparedSessionTurn, error) {
	text := strings.TrimSpace(input.Text)
	attachments := sanitizeSessionMessageAttachments(input.Attachments)
	input.Attachments = attachments
	if input.SessionID == "" {
		return domain.PreparedSessionTurn{}, errors.New("sessionId is required")
	}
	if text == "" && len(attachments) == 0 {
		return domain.PreparedSessionTurn{}, errors.New("message text or attachment is required")
	}
	eventText := sessionMessageEventText(text, attachments)
	reasoningEffort := normalizeReasoningEffort(input.ReasoningEffort)
	serviceTier := normalizeServiceTier(input.ServiceTier)
	delivery, err := domain.NormalizeInputDelivery(input.Delivery)
	if err != nil {
		return domain.PreparedSessionTurn{}, err
	}
	if delivery != domain.InputDeliveryImmediate {
		state, _ := s.store.GetSessionExecutionState(ctx, input.SessionID)
		if state.Status == domain.ExecutionStatusRunning || state.Status == domain.ExecutionStatusCompacting {
			pending, err := s.store.CreatePendingSessionInput(ctx, domain.PendingSessionInput{
				SessionID: input.SessionID, TurnID: state.TurnID, Text: eventText, Delivery: delivery, Status: domain.PendingInputStatusPending,
			})
			if err != nil {
				return domain.PreparedSessionTurn{}, err
			}
			state.PendingInputIDs = append(state.PendingInputIDs, pending.ID)
			state.Reason = "input queued for " + delivery + " boundary"
			_, _ = s.store.UpsertSessionExecutionState(ctx, state)
			_, _ = s.AppendEvent(ctx, domain.AppendEventRequest{
				SessionID: input.SessionID, TurnID: state.TurnID, Type: domain.EventTypeSystemNote, Role: domain.EventRoleSystem,
				Visibility: domain.EventVisibilityInternal, Content: "Queued session input",
				Payload: map[string]any{"kind": "pending_input", "pendingInputId": pending.ID, "delivery": delivery},
			})
			return domain.PreparedSessionTurn{
				Turn:      domain.Turn{SessionID: input.SessionID, Status: domain.TurnStatusRunning, TimeCreated: pending.TimeCreated, TimeUpdated: pending.TimeUpdated},
				UserEvent: domain.SessionEvent{SessionID: input.SessionID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser, Visibility: domain.EventVisibilityInternal, Content: eventText, TimeCreated: pending.TimeCreated},
			}, nil
		}
	}
	if input.Model != nil || strings.TrimSpace(input.ReasoningEffort) != "" || strings.TrimSpace(input.ServiceTier) != "" {
		_, _ = s.UpdateModelPreferences(ctx, domain.ModelPreferencesInput{Model: input.Model, ReasoningEffort: reasoningEffort, ServiceTier: serviceTier})
	}
	userEvent, err := s.AppendEvent(ctx, domain.AppendEventRequest{
		SessionID:  input.SessionID,
		Type:       domain.EventTypeUserMessage,
		Role:       domain.EventRoleUser,
		Visibility: domain.EventVisibilityNormal,
		Content:    eventText,
		Payload:    sessionMessageEventPayload(attachments),
	})
	if err != nil {
		return domain.PreparedSessionTurn{}, err
	}
	if _, err := s.maybeAutoCompactSessionContext(ctx, input.SessionID); err != nil {
		return domain.PreparedSessionTurn{}, err
	}
	modeDef, err := s.resolveAgentModeForRequest(ctx, input.SessionID, input.AgentMode)
	if err != nil {
		return domain.PreparedSessionTurn{}, err
	}
	turn, err := s.StartTurn(ctx, domain.StartTurnRequest{SessionID: input.SessionID, UserEventID: userEvent.ID, AgentMode: modeDef.ID})
	if err != nil {
		return domain.PreparedSessionTurn{}, err
	}
	_, _ = s.store.UpsertSessionExecutionState(ctx, domain.SessionExecutionState{SessionID: input.SessionID, TurnID: turn.ID, Status: domain.ExecutionStatusRunning, Reason: "turn running"})
	if s.pluginManager != nil {
		_ = s.pluginManager.InvokeHook(ctx, "on_session_start", map[string]any{"sessionId": input.SessionID, "turnId": turn.ID, "agentMode": modeDef.ID})
	}
	history, err := s.modelVisibleSessionHistory(ctx, input.SessionID)
	if err != nil {
		_, _ = s.FailTurn(ctx, domain.FailTurnRequest{TurnID: turn.ID, Error: err.Error()})
		return domain.PreparedSessionTurn{}, err
	}
	attachCurrentTurnFiles(history, userEvent.ID, eventText, attachments)
	prepared := domain.PreparedSessionTurn{Turn: turn, UserEvent: userEvent}
	work := func(workCtx context.Context) (domain.PreparedSessionTurn, error) {
		return s.completeSessionTurn(workCtx, input, text, history, turn, userEvent, reasoningEffort, serviceTier)
	}
	if async != nil {
		return async(ctx, prepared, work)
	}
	return work(ctx)
}

func (s *Service) completeSessionTurn(
	ctx context.Context,
	input domain.SubmitSessionMessageRequest,
	text string,
	history []domain.ChatMessage,
	turn domain.Turn,
	userEvent domain.SessionEvent,
	reasoningEffort string,
	serviceTier string,
) (domain.PreparedSessionTurn, error) {
	emittedText := ""
	reply, model, err := s.runAssistantAgentLoop(ctx, input, history, turn, reasoningEffort, serviceTier, func(delta string) {
		if s.onAssistantDelta != nil && delta != "" {
			emittedText += delta
			s.onAssistantDelta(input.SessionID, turn.ID, delta)
		}
	})
	if err != nil {
		if isModelExecutionUnavailable(err) {
			reply = deterministicAssistantFallback(text)
			model = input.Model
		} else if isContextCancelled(ctx, err) {
			if cancelled, cancelErr := s.store.UpdateTurnStatus(context.Background(), turn.ID, domain.TurnStatusCancelled, "Turn cancelled"); cancelErr == nil {
				_ = s.cleanupCancelledTurn(context.Background(), cancelled, "Turn cancelled")
				if s.onTurnUpdated != nil {
					s.onTurnUpdated(cancelled.SessionID, cancelled)
				}
			}
			_, _ = s.store.UpsertSessionExecutionState(context.Background(), domain.SessionExecutionState{SessionID: input.SessionID, TurnID: turn.ID, Status: domain.ExecutionStatusInterrupted, Reason: "turn cancelled"})
			return domain.PreparedSessionTurn{}, err
		} else {
			_, _ = s.FailTurn(ctx, domain.FailTurnRequest{TurnID: turn.ID, Error: err.Error()})
			_, _ = s.store.UpsertSessionExecutionState(ctx, domain.SessionExecutionState{SessionID: input.SessionID, TurnID: turn.ID, Status: domain.ExecutionStatusFailed, Reason: err.Error()})
			return domain.PreparedSessionTurn{}, err
		}
	}
	if reply == "" {
		replyErr := errors.New("assistant reply is empty")
		_, _ = s.FailTurn(ctx, domain.FailTurnRequest{TurnID: turn.ID, Error: replyErr.Error()})
		_, _ = s.store.UpsertSessionExecutionState(ctx, domain.SessionExecutionState{SessionID: input.SessionID, TurnID: turn.ID, Status: domain.ExecutionStatusFailed, Reason: replyErr.Error()})
		return domain.PreparedSessionTurn{}, replyErr
	}
	if !strings.Contains(emittedText, reply) && s.onAssistantDelta != nil {
		s.onAssistantDelta(input.SessionID, turn.ID, reply)
	}
	assistantEvent, err := s.AppendEvent(ctx, domain.AppendEventRequest{SessionID: input.SessionID, TurnID: turn.ID, Type: domain.EventTypeAssistantMessage, Role: domain.EventRoleAssistant, Visibility: domain.EventVisibilityNormal, Content: reply})
	if err != nil {
		_, _ = s.FailTurn(ctx, domain.FailTurnRequest{TurnID: turn.ID, Error: err.Error()})
		_, _ = s.store.UpsertSessionExecutionState(ctx, domain.SessionExecutionState{SessionID: input.SessionID, TurnID: turn.ID, Status: domain.ExecutionStatusFailed, Reason: err.Error()})
		return domain.PreparedSessionTurn{}, err
	}
	turn, err = s.CompleteTurn(ctx, domain.CompleteTurnRequest{TurnID: turn.ID})
	if err != nil {
		return domain.PreparedSessionTurn{}, err
	}
	_, _ = s.promotePendingInputsAtBoundary(ctx, input.SessionID, turn.ID, domain.InputDeliverySteer)
	_, _ = s.store.UpsertSessionExecutionState(ctx, domain.SessionExecutionState{SessionID: input.SessionID, TurnID: turn.ID, Status: domain.ExecutionStatusIdle, Reason: "turn complete"})
	if s.onSessionUpdated != nil {
		s.onSessionUpdated(input.SessionID, nil)
	}
	go s.ensureGeneratedSessionTitle(context.Background(), input.SessionID, model)
	return domain.PreparedSessionTurn{Turn: turn, Model: model, UserEvent: userEvent, AssistantEvent: &assistantEvent}, nil
}

func (s *Service) promotePendingInputsAtBoundary(ctx context.Context, sessionID string, turnID string, delivery string) ([]domain.PendingSessionInput, error) {
	items, err := s.store.ListPendingSessionInputs(ctx, sessionID, domain.PendingInputStatusPending)
	if err != nil {
		return nil, err
	}
	promoted := []domain.PendingSessionInput{}
	for _, item := range items {
		if item.Delivery != delivery {
			continue
		}
		updated, err := s.store.UpdatePendingSessionInputStatus(ctx, item.ID, domain.PendingInputStatusPromoted, turnID)
		if err != nil {
			return promoted, err
		}
		promoted = append(promoted, updated)
		_, _ = s.AppendEvent(ctx, domain.AppendEventRequest{
			SessionID: sessionID, TurnID: turnID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser,
			Visibility: domain.EventVisibilityNormal, Content: item.Text,
			Payload: map[string]any{"kind": "promoted_input", "delivery": item.Delivery, "pendingInputId": item.ID},
		})
	}
	return promoted, nil
}

func sanitizeSessionMessageAttachments(attachments []domain.MessageAttachment) []domain.MessageAttachment {
	return normalizeSessionMessageAttachments(attachments)
}

func sessionMessageEventText(text string, attachments []domain.MessageAttachment) string {
	return buildSessionMessageEventText(text, attachments)
}

func sessionMessageEventPayload(attachments []domain.MessageAttachment) map[string]any {
	return buildSessionMessageEventPayload(attachments)
}

func attachCurrentTurnFiles(messages []domain.ChatMessage, userEventID string, eventText string, attachments []domain.MessageAttachment) {
	attachFilesToCurrentTurn(messages, userEventID, eventText, attachments)
}

func isContextCancelled(ctx context.Context, err error) bool {
	if ctx != nil && errors.Is(ctx.Err(), context.Canceled) {
		return true
	}
	return errors.Is(err, context.Canceled)
}
