package app

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"aivo/core/domain"
)

func (s *Service) AppendEvent(ctx context.Context, input domain.AppendEventRequest) (domain.SessionEvent, error) {
	if strings.TrimSpace(input.SessionID) == "" {
		return domain.SessionEvent{}, errors.New("sessionId is required")
	}
	if err := domain.ValidateEventType(input.Type); err != nil {
		return domain.SessionEvent{}, err
	}
	role, err := domain.NormalizeEventRole(input.Role)
	if err != nil {
		return domain.SessionEvent{}, err
	}
	visibility, err := domain.NormalizeEventVisibility(input.Visibility)
	if err != nil {
		return domain.SessionEvent{}, err
	}
	event := domain.SessionEvent{
		ID: uuid.NewString(), SessionID: strings.TrimSpace(input.SessionID), TurnID: strings.TrimSpace(input.TurnID),
		Type: input.Type, Role: role, Visibility: visibility, Content: strings.TrimSpace(input.Content),
		Payload: input.Payload, TokenCount: input.TokenCount, TimeCreated: domain.NowString(s.now()),
	}
	if err := s.store.AppendSessionEvent(ctx, event); err != nil {
		return domain.SessionEvent{}, err
	}
	if event.Visibility == domain.EventVisibilityNormal && event.Type == domain.EventTypeUserMessage {
		_ = s.updateUntitledSession(ctx, event.SessionID, event.Content)
	}
	return event, nil
}

func (s *Service) ListEvents(ctx context.Context, sessionID string, includeNonNormal bool, limit int) ([]domain.SessionEvent, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("sessionId is required")
	}
	return s.store.ListSessionEvents(ctx, sessionID, includeNonNormal, limit)
}

func (s *Service) UpdateSessionEvent(ctx context.Context, input domain.UpdateSessionEventRequest) (domain.SessionEvent, error) {
	if strings.TrimSpace(input.EventID) == "" {
		return domain.SessionEvent{}, errors.New("eventId is required")
	}
	if strings.TrimSpace(input.Content) == "" {
		return domain.SessionEvent{}, errors.New("content is required")
	}
	event, err := s.store.GetSessionEvent(ctx, input.EventID)
	if err != nil {
		return domain.SessionEvent{}, err
	}
	if event.Type != domain.EventTypeUserMessage && event.Type != domain.EventTypeAssistantMessage {
		return domain.SessionEvent{}, errors.New("only user and assistant messages can be edited")
	}
	updated, err := s.store.UpdateSessionEvent(ctx, input)
	if err == nil && s.onSessionUpdated != nil {
		s.onSessionUpdated(updated.SessionID, nil)
	}
	return updated, err
}

func (s *Service) DeleteSessionEvent(ctx context.Context, input domain.DeleteSessionEventRequest) (domain.SessionEvent, error) {
	if strings.TrimSpace(input.EventID) == "" {
		return domain.SessionEvent{}, errors.New("eventId is required")
	}
	event, err := s.store.SetSessionEventVisibility(ctx, input.EventID, domain.EventVisibilityHidden)
	if err == nil && s.onSessionUpdated != nil {
		s.onSessionUpdated(event.SessionID, nil)
	}
	return event, err
}

func (s *Service) ListSessionEventsAfterCursor(ctx context.Context, input domain.ListSessionEventsAfterCursorInput) (domain.ListSessionEventsAfterCursorResult, error) {
	if strings.TrimSpace(input.SessionID) == "" {
		return domain.ListSessionEventsAfterCursorResult{}, errors.New("sessionId is required")
	}
	events, next, err := s.store.ListSessionEventsAfterCursor(ctx, input.SessionID, input.Cursor, input.IncludeNonNormal, input.Limit)
	if err != nil {
		return domain.ListSessionEventsAfterCursorResult{}, err
	}
	return domain.ListSessionEventsAfterCursorResult{Events: events, NextCursor: next}, nil
}
