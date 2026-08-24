package app

import (
	"testing"
	"time"

	"aivo/core/domain"
)

func TestSessionRuntimeMetricsRecordsSuccessfulSteps(t *testing.T) {
	startedAt := time.Unix(0, 0)
	metrics := sessionRuntimeMetrics{}
	metrics.recordSuccessfulStep(
		startedAt,
		startedAt.Add(700*time.Millisecond),
		startedAt.Add(1100*time.Millisecond),
		&domain.TokenUsage{
			InputTokens: 9400, OutputTokens: 50, CacheReadTokens: 0,
			InputTokensAvailable: true, OutputTokensAvailable: true, CacheReadTokensAvailable: true,
		},
	)
	metrics.recordSuccessfulStep(
		startedAt,
		time.Time{},
		startedAt.Add(500*time.Millisecond),
		nil,
	)

	payload := metrics.payload()
	want := map[string]int64{
		"steps":           2,
		"llmMs":           1600,
		"ttftMs":          700,
		"ttftSteps":       1,
		"decodeMs":        400,
		"decodeTokens":    50,
		"inputTokens":     9400,
		"outputTokens":    50,
		"cacheReadTokens": 0,
		"usageSteps":      1,
		"inputUsageSteps": 1,
		"cacheUsageSteps": 1,
	}
	for key, expected := range want {
		if got, ok := payload[key].(int64); !ok || got != expected {
			t.Fatalf("payload[%q] = %#v, want %d", key, payload[key], expected)
		}
	}
	if available, ok := payload["cacheReadAvailable"].(bool); !ok || !available {
		t.Fatalf("cacheReadAvailable = %#v, want true", payload["cacheReadAvailable"])
	}
}

func TestSessionRuntimeMetricsOmitsUnavailableGroups(t *testing.T) {
	startedAt := time.Unix(0, 0)
	metrics := sessionRuntimeMetrics{}
	metrics.recordSuccessfulStep(startedAt, time.Time{}, startedAt.Add(time.Second), nil)

	payload := metrics.payload()
	for _, key := range []string{"ttftMs", "ttftSteps", "decodeMs", "decodeTokens", "inputTokens", "outputTokens", "cacheReadTokens"} {
		if _, exists := payload[key]; exists {
			t.Fatalf("payload unexpectedly contains %q: %#v", key, payload)
		}
	}
}

func TestFoldSessionRuntimeStatsCountsEachTurnOnce(t *testing.T) {
	stats := domain.SessionRuntimeStats{}
	countedTurns := map[string]bool{}
	foldSessionRuntimeStats(&stats, countedTurns, []domain.SessionEvent{
		{
			ID: "assistant-1", TurnID: "turn-1", Type: domain.EventTypeAssistantMessage, Visibility: domain.EventVisibilityNormal,
			Payload: map[string]any{"runtimeMetrics": map[string]any{
				"steps": float64(2), "llmMs": float64(2500), "ttftMs": float64(1200), "ttftSteps": float64(2),
				"decodeMs": float64(3000), "decodeTokens": float64(40), "inputTokens": float64(100),
				"outputTokens": float64(40), "cacheReadTokens": float64(120),
			}},
		},
		{
			ID: "assistant-duplicate", TurnID: "turn-1", Type: domain.EventTypeAssistantMessage, Visibility: domain.EventVisibilityNormal,
			Payload: map[string]any{"runtimeMetrics": map[string]any{"steps": float64(1), "llmMs": float64(500)}},
		},
		{
			ID: "malformed", TurnID: "turn-2", Type: domain.EventTypeAssistantMessage, Visibility: domain.EventVisibilityNormal,
			Payload: map[string]any{"runtimeMetrics": map[string]any{"steps": float64(-1), "llmMs": float64(1)}},
		},
	})

	if stats.Turns != 1 || stats.Steps != 3 || stats.LLMMs != 3000 {
		t.Fatalf("stats = %+v, want one turn, three steps, and 3000ms", stats)
	}
	if stats.TTFTMs != 1200 || stats.TTFTSteps != 2 || stats.DecodeMs != 3000 || stats.DecodeTokens != 40 {
		t.Fatalf("timing stats = %+v, want persisted timing totals", stats)
	}
	if stats.InputTokens != 100 || stats.OutputTokens != 40 || stats.CacheReadTokens != 100 {
		t.Fatalf("usage stats = %+v, want cache reads clamped to input", stats)
	}
	if !stats.CacheReadAvailable {
		t.Fatalf("usage stats = %+v, want cache availability inferred from positive legacy cache usage", stats)
	}
}

func TestSessionRuntimeMetricsDistinguishesMissingCacheUsageFromZeroHits(t *testing.T) {
	startedAt := time.Unix(0, 0)
	unknown := sessionRuntimeMetrics{}
	unknown.recordSuccessfulStep(startedAt, time.Time{}, startedAt.Add(time.Second), &domain.TokenUsage{
		InputTokens: 10, OutputTokens: 2, InputTokensAvailable: true, OutputTokensAvailable: true,
	})
	unknownPayload := unknown.payload()
	if available, ok := unknownPayload["cacheReadAvailable"].(bool); !ok || available {
		t.Fatalf("unknown cache availability = %#v, want false", unknownPayload["cacheReadAvailable"])
	}
	if _, exists := unknownPayload["cacheReadTokens"]; exists {
		t.Fatalf("unknown cache payload unexpectedly contains cacheReadTokens: %#v", unknownPayload)
	}

	knownZero := sessionRuntimeMetrics{}
	knownZero.recordSuccessfulStep(startedAt, time.Time{}, startedAt.Add(time.Second), &domain.TokenUsage{
		InputTokens: 10, OutputTokens: 2, InputTokensAvailable: true, OutputTokensAvailable: true,
		CacheReadTokensAvailable: true,
	})
	knownPayload := knownZero.payload()
	if available, ok := knownPayload["cacheReadAvailable"].(bool); !ok || !available {
		t.Fatalf("known cache availability = %#v, want true", knownPayload["cacheReadAvailable"])
	}
	if value, ok := knownPayload["cacheReadTokens"].(int64); !ok || value != 0 {
		t.Fatalf("known cache read tokens = %#v, want explicit zero", knownPayload["cacheReadTokens"])
	}
}

func TestFoldSessionRuntimeStatsOmitsCacheWhenAnyUsageStepDoesNotReportIt(t *testing.T) {
	stats := domain.SessionRuntimeStats{}
	foldSessionRuntimeStats(&stats, map[string]bool{}, []domain.SessionEvent{
		{
			ID: "known", TurnID: "turn-1", Type: domain.EventTypeAssistantMessage, Visibility: domain.EventVisibilityNormal,
			Payload: map[string]any{"runtimeMetrics": map[string]any{
				"steps": int64(1), "llmMs": int64(10), "inputTokens": int64(100), "outputTokens": int64(10),
				"inputUsageSteps": int64(1), "cacheUsageSteps": int64(1), "cacheReadAvailable": true, "cacheReadTokens": int64(80),
			}},
		},
		{
			ID: "unknown", TurnID: "turn-2", Type: domain.EventTypeAssistantMessage, Visibility: domain.EventVisibilityNormal,
			Payload: map[string]any{"runtimeMetrics": map[string]any{
				"steps": int64(1), "llmMs": int64(10), "inputTokens": int64(50), "outputTokens": int64(5),
				"inputUsageSteps": int64(1), "cacheUsageSteps": int64(0), "cacheReadAvailable": false,
			}},
		},
	})

	if stats.InputTokens != 150 || stats.OutputTokens != 15 {
		t.Fatalf("usage stats = %+v, want complete token totals", stats)
	}
	if stats.CacheReadAvailable {
		t.Fatalf("usage stats = %+v, cache percentage must be unavailable", stats)
	}
}
