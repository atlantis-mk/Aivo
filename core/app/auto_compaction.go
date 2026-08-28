package app

import (
	"context"
	"fmt"
	"strings"

	"aivo/core/domain"
)

const (
	defaultCompactionThresholdPercent = 80
	defaultCompactionRetainPercent    = 16
	fallbackCompactionContextTokens   = (modelContextSectionBudget + modelTailMessageBudget) / 4
)

type compactionPressure struct {
	ContextWindowTokens   int
	AutoCompactTokenLimit int
	TriggerTokens         int
	RetainTokens          int
	ThresholdPercent      int
	CapacitySource        string
}

type compactionPlan struct {
	FromEventID string
	ToEventID   string
	Pressure    compactionPressure
}

func (s *Service) maybeAutoCompactSessionContext(ctx context.Context, sessionID string, requestedModel *domain.ModelRef) (bool, error) {
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
	pressure := s.resolveCompactionPressure(ctx, session, requestedModel, config)
	if pressure.TriggerTokens <= 0 || uncompactedEventTokens(events, latest) < pressure.TriggerTokens {
		return false, nil
	}
	plan, ok := automaticCompactionPlan(events, latest, pressure)
	if !ok {
		return false, nil
	}
	if _, err := s.compactSessionContext(ctx, domain.CompactSessionContextInput{
		SessionID: sessionID, CharacterBudget: modelContextSectionBudget, Automatic: true,
	}, &plan); err != nil {
		return false, fmt.Errorf("automatic session compaction failed: %w", err)
	}
	return true, nil
}

func (s *Service) resolveCompactionPressure(ctx context.Context, session domain.Session, requestedModel *domain.ModelRef, config domain.CompactionRuntimeConfig) compactionPressure {
	selectedModel := requestedModel
	if selectedModel == nil || strings.TrimSpace(selectedModel.ModelID) == "" {
		selectedModel = session.Model
	}
	if selectedModel == nil || strings.TrimSpace(selectedModel.ModelID) == "" {
		if appConfig, err := s.AppConfig(ctx); err == nil {
			selectedModel = appConfig.DefaultModel
		}
	}
	contextTokens, autoCompactTokenLimit, source := s.modelCompactionLimits(ctx, selectedModel)
	if contextTokens <= 0 {
		contextTokens = fallbackCompactionContextTokens
		source = "safe_estimate"
	}
	threshold := config.ThresholdPercent
	if threshold <= 0 || threshold > 100 {
		threshold = defaultCompactionThresholdPercent
	}
	trigger := contextTokens * threshold / 100
	if autoCompactTokenLimit > 0 && autoCompactTokenLimit < trigger {
		trigger = autoCompactTokenLimit
	}
	if config.ReserveTokens > 0 {
		reservedLimit := contextTokens - config.ReserveTokens
		if reservedLimit > 0 && reservedLimit < trigger {
			trigger = reservedLimit
		}
	}
	retain := contextTokens * defaultCompactionRetainPercent / 100
	if trigger > 1 && retain >= trigger {
		retain = trigger / 2
	}
	return compactionPressure{
		ContextWindowTokens:   contextTokens,
		AutoCompactTokenLimit: autoCompactTokenLimit,
		TriggerTokens:         trigger,
		RetainTokens:          retain,
		ThresholdPercent:      threshold,
		CapacitySource:        source,
	}
}

func (s *Service) modelContextLength(ctx context.Context, model *domain.ModelRef) (int, string) {
	contextTokens, _, source := s.modelCompactionLimits(ctx, model)
	return contextTokens, source
}

func (s *Service) modelCompactionLimits(ctx context.Context, model *domain.ModelRef) (int, int, string) {
	if model == nil || strings.TrimSpace(model.ModelID) == "" {
		return 0, 0, ""
	}
	providerID := s.normalizeProviderID(model.ProviderID)
	if cache, err := s.store.LoadProviderModelCache(ctx, providerID); err == nil && cache != nil {
		if info, found := findModelInfo(cache.Models, model.ModelID); found && info.ContextLength > 0 {
			return info.ContextLength, info.AutoCompactTokenLimit, "provider_cache"
		}
	}
	if definition, ok := s.providerDefinition(providerID); ok {
		if info, found := findModelInfo(definition.Models, model.ModelID); found && info.ContextLength > 0 {
			return info.ContextLength, info.AutoCompactTokenLimit, "provider_catalog"
		}
	}
	return 0, 0, ""
}

func automaticCompactionPlan(events []domain.SessionEvent, latest *domain.SessionSummary, pressure compactionPressure) (compactionPlan, bool) {
	start := firstEventAfterSummary(events, latest)
	if start >= len(events) {
		return compactionPlan{}, false
	}
	retainStart := len(events)
	retainedTokens := 0
	for i := len(events) - 1; i >= start; i-- {
		retainedTokens += sessionEventTokens(events[i])
		retainStart = i
		if retainedTokens >= pressure.RetainTokens {
			break
		}
	}
	// Automatic compaction runs after the new user event is persisted but before
	// its turn starts. That newest unanswered input must remain outside the summary.
	for i := len(events) - 1; i >= start; i-- {
		if events[i].Type == domain.EventTypeUserMessage {
			if i < retainStart {
				retainStart = i
			}
			break
		}
	}
	toIndex := retainStart - 1
	for toIndex >= start && !compactableSessionEvent(events[toIndex]) {
		toIndex--
	}
	if toIndex < start {
		return compactionPlan{}, false
	}
	fromEventID := ""
	if latest != nil {
		fromEventID = latest.ToEventID
	}
	return compactionPlan{FromEventID: fromEventID, ToEventID: events[toIndex].ID, Pressure: pressure}, true
}

func uncompactedEventTokens(events []domain.SessionEvent, latest *domain.SessionSummary) int {
	start := firstEventAfterSummary(events, latest)
	used := 0
	for _, event := range events[start:] {
		if event.Type != domain.EventTypeSummary {
			used += sessionEventTokens(event)
		}
	}
	return used
}

func firstEventAfterSummary(events []domain.SessionEvent, latest *domain.SessionSummary) int {
	if latest == nil {
		return 0
	}
	if latest.ToEventID != "" {
		for i, event := range events {
			if event.ID == latest.ToEventID {
				return i + 1
			}
		}
	}
	for i, event := range events {
		if event.TimeCreated > latest.TimeCreated {
			return i
		}
	}
	return len(events)
}

func sessionEventTokens(event domain.SessionEvent) int {
	if event.TokenCount > 0 {
		return event.TokenCount
	}
	return estimateTokens(event.Role + " " + strings.TrimSpace(event.Content))
}

func compactableSessionEvent(event domain.SessionEvent) bool {
	return event.Type == domain.EventTypeUserMessage || event.Type == domain.EventTypeAssistantMessage
}
