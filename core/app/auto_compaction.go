package app

import (
	"context"
	"fmt"
	"strings"

	"aivo/core/domain"
)

func (s *Service) maybeAutoCompactSessionContext(ctx context.Context, sessionID string) (bool, error) {
	session, err := s.store.GetRuntimeSession(ctx, sessionID)
	if err != nil {
		return false, err
	}
	config := loadEffectiveRuntimeConfig(session.ProjectPath).Config.Compaction
	if config.Auto != nil && !*config.Auto {
		return false, nil
	}
	events, err := s.store.ListSessionEvents(ctx, sessionID, false, 500)
	if err != nil {
		return false, err
	}
	latest, _ := s.store.LatestSummary(ctx, sessionID)
	usedChars := uncompactedEventChars(events, latest)
	capacityChars := modelContextSectionBudget + modelTailMessageBudget
	threshold := config.ThresholdPercent
	if threshold <= 0 {
		threshold = 80
	}
	triggerChars := capacityChars * threshold / 100
	if config.ReserveTokens > 0 {
		reservedLimit := capacityChars - config.ReserveTokens*4
		if reservedLimit > 0 && reservedLimit < triggerChars {
			triggerChars = reservedLimit
		}
	}
	if triggerChars <= 0 || usedChars < triggerChars {
		return false, nil
	}
	if _, err := s.CompactSessionContext(ctx, domain.CompactSessionContextInput{
		SessionID: sessionID, CharacterBudget: modelContextSectionBudget, Automatic: true,
	}); err != nil {
		return false, fmt.Errorf("automatic session compaction failed: %w", err)
	}
	return true, nil
}

func uncompactedEventChars(events []domain.SessionEvent, latest *domain.SessionSummary) int {
	afterSummary := latest == nil
	used := 0
	for _, event := range events {
		if !afterSummary {
			if latest.ToEventID != "" {
				if event.ID == latest.ToEventID {
					afterSummary = true
				}
				continue
			}
			if event.TimeCreated <= latest.TimeCreated {
				continue
			}
			afterSummary = true
		}
		if event.Type == domain.EventTypeSummary {
			continue
		}
		used += len(strings.TrimSpace(event.Content))
	}
	return used
}
