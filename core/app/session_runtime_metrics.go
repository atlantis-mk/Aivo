package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"aivo/core/domain"
)

const sessionRuntimeStatsPageSize = 500

type sessionRuntimeMetrics struct {
	steps           int64
	llmMs           int64
	ttftMs          int64
	ttftSteps       int64
	decodeMs        int64
	decodeTokens    int64
	decodeSteps     int64
	inputTokens     int64
	outputTokens    int64
	cacheReadTokens int64
	usageSteps      int64
	inputUsageSteps int64
	cacheUsageSteps int64
}

func (metrics *sessionRuntimeMetrics) recordSuccessfulStep(
	startedAt time.Time,
	firstTokenAt time.Time,
	completedAt time.Time,
	usage *domain.TokenUsage,
) {
	if metrics == nil {
		return
	}
	metrics.steps++
	metrics.llmMs += nonNegativeMilliseconds(completedAt.Sub(startedAt))
	if !firstTokenAt.IsZero() {
		metrics.ttftMs += nonNegativeMilliseconds(firstTokenAt.Sub(startedAt))
		metrics.ttftSteps++
	}
	if usage == nil {
		return
	}
	inputAvailable := usage.InputTokensAvailable || usage.InputTokens > 0
	outputAvailable := usage.OutputTokensAvailable || usage.OutputTokens > 0
	if !inputAvailable && !outputAvailable {
		return
	}
	inputTokens := max(0, usage.InputTokens)
	outputTokens := max(0, usage.OutputTokens)
	cacheReadTokens := min(max(0, usage.CacheReadTokens), inputTokens)
	metrics.inputTokens += int64(inputTokens)
	metrics.outputTokens += int64(outputTokens)
	metrics.usageSteps++
	if inputAvailable {
		metrics.inputUsageSteps++
		if usage.CacheReadTokensAvailable {
			metrics.cacheReadTokens += int64(cacheReadTokens)
			metrics.cacheUsageSteps++
		}
	}
	if !firstTokenAt.IsZero() {
		metrics.decodeMs += nonNegativeMilliseconds(completedAt.Sub(firstTokenAt))
		metrics.decodeTokens += int64(outputTokens)
		metrics.decodeSteps++
	}
}

func nonNegativeMilliseconds(duration time.Duration) int64 {
	if duration <= 0 {
		return 0
	}
	return duration.Milliseconds()
}

func (metrics sessionRuntimeMetrics) payload() map[string]any {
	if metrics.steps == 0 {
		return nil
	}
	payload := map[string]any{
		"steps": metrics.steps,
		"llmMs": metrics.llmMs,
	}
	if metrics.ttftSteps > 0 {
		payload["ttftMs"] = metrics.ttftMs
		payload["ttftSteps"] = metrics.ttftSteps
	}
	if metrics.decodeSteps > 0 {
		payload["decodeMs"] = metrics.decodeMs
		payload["decodeTokens"] = metrics.decodeTokens
	}
	if metrics.usageSteps > 0 {
		payload["inputTokens"] = metrics.inputTokens
		payload["outputTokens"] = metrics.outputTokens
		payload["usageSteps"] = metrics.usageSteps
		payload["inputUsageSteps"] = metrics.inputUsageSteps
		payload["cacheUsageSteps"] = metrics.cacheUsageSteps
		cacheReadAvailable := metrics.inputUsageSteps > 0 && metrics.cacheUsageSteps == metrics.inputUsageSteps
		payload["cacheReadAvailable"] = cacheReadAvailable
		if cacheReadAvailable {
			payload["cacheReadTokens"] = metrics.cacheReadTokens
		}
	}
	return payload
}

func (s *Service) GetSessionRuntimeStats(ctx context.Context, sessionID string) (domain.SessionRuntimeStats, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return domain.SessionRuntimeStats{}, errors.New("sessionId is required")
	}
	stats := domain.SessionRuntimeStats{}
	countedTurns := map[string]bool{}
	cursor := ""
	for {
		events, nextCursor, err := s.store.ListSessionEventsAfterCursor(ctx, sessionID, cursor, false, sessionRuntimeStatsPageSize)
		if err != nil {
			return domain.SessionRuntimeStats{}, err
		}
		foldSessionRuntimeStats(&stats, countedTurns, events)
		if len(events) < sessionRuntimeStatsPageSize || nextCursor == "" || nextCursor == cursor {
			break
		}
		cursor = nextCursor
	}
	return stats, nil
}

func foldSessionRuntimeStats(stats *domain.SessionRuntimeStats, countedTurns map[string]bool, events []domain.SessionEvent) {
	for _, event := range events {
		if event.Type != domain.EventTypeAssistantMessage || event.Visibility != domain.EventVisibilityNormal {
			continue
		}
		rawMetrics, _ := event.Payload["runtimeMetrics"].(map[string]any)
		steps, stepsOK := runtimeMetricInt64(rawMetrics["steps"])
		llmMs, llmOK := runtimeMetricInt64(rawMetrics["llmMs"])
		if !stepsOK || steps < 1 || !llmOK {
			continue
		}
		turnKey := strings.TrimSpace(event.TurnID)
		if turnKey == "" {
			turnKey = event.ID
		}
		if !countedTurns[turnKey] {
			stats.Turns++
			countedTurns[turnKey] = true
		}
		stats.Steps += steps
		stats.LLMMs += llmMs
		stats.TTFTMs += optionalRuntimeMetric(rawMetrics, "ttftMs")
		stats.TTFTSteps += optionalRuntimeMetric(rawMetrics, "ttftSteps")
		stats.DecodeMs += optionalRuntimeMetric(rawMetrics, "decodeMs")
		stats.DecodeTokens += optionalRuntimeMetric(rawMetrics, "decodeTokens")
		inputTokens, inputOK := runtimeMetricInt64(rawMetrics["inputTokens"])
		outputTokens, outputOK := runtimeMetricInt64(rawMetrics["outputTokens"])
		if inputOK && outputOK {
			stats.InputTokens += inputTokens
			stats.OutputTokens += outputTokens
			inputUsageSteps, inputStepsOK := runtimeMetricInt64(rawMetrics["inputUsageSteps"])
			if !inputStepsOK {
				inputUsageSteps = steps
			}
			stats.InputUsageSteps += inputUsageSteps
			cacheUsageSteps, cacheStepsOK := runtimeMetricInt64(rawMetrics["cacheUsageSteps"])
			cacheAvailable, availabilityOK := rawMetrics["cacheReadAvailable"].(bool)
			if !cacheStepsOK && ((availabilityOK && cacheAvailable) || optionalRuntimeMetric(rawMetrics, "cacheReadTokens") > 0) {
				cacheUsageSteps = inputUsageSteps
			}
			stats.CacheUsageSteps += cacheUsageSteps
			if cacheUsageSteps > 0 {
				stats.CacheReadTokens += optionalRuntimeMetric(rawMetrics, "cacheReadTokens")
			}
		}
	}
	stats.CacheReadTokens = min(stats.CacheReadTokens, stats.InputTokens)
	stats.CacheReadAvailable = stats.InputUsageSteps > 0 && stats.CacheUsageSteps == stats.InputUsageSteps
}

func optionalRuntimeMetric(metrics map[string]any, key string) int64 {
	value, ok := runtimeMetricInt64(metrics[key])
	if !ok {
		return 0
	}
	return value
}

func runtimeMetricInt64(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		if typed >= 0 {
			return int64(typed), true
		}
	case int64:
		if typed >= 0 {
			return typed, true
		}
	case float64:
		converted := int64(typed)
		if typed >= 0 && float64(converted) == typed {
			return converted, true
		}
	}
	return 0, false
}
