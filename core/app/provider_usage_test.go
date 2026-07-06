package app

import (
	"context"
	"testing"

	"aivo/core/domain"
)

func TestRecordProviderCallEventEstimatesTokensAndCost(t *testing.T) {
	store := &memoryProviderStore{}
	service := NewService(store)
	route := ResolvedModelRoute{
		Provider:  domain.ProviderConfig{ID: "priced-provider"},
		Model:     domain.ModelRef{ProviderID: "priced-provider", ModelID: "priced-model"},
		Transport: TransportOpenAICompatible,
		Definition: ProviderDefinition{
			ID: "priced-provider",
			Models: []domain.ModelInfo{{
				ID: "priced-model", ProviderID: "priced-provider", Name: "Priced Model",
				Pricing: map[string]float64{"input": 2, "output": 10},
			}},
		},
	}
	req := modelRequirement{InputTokens: 100, ToolCallCount: 1}

	service.recordProviderCallEvent(context.Background(), route, req, 0, 1, 0, domain.ChatResponse{Text: "hello world"}, nil)

	if len(store.callEvents) != 1 {
		t.Fatalf("callEvents = %+v, want one", store.callEvents)
	}
	event := store.callEvents[0]
	if event.InputTokens != 100 || event.OutputTokens == 0 || event.TotalTokens != event.InputTokens+event.OutputTokens || !event.Estimated {
		t.Fatalf("event = %+v, want estimated token fields", event)
	}
	if event.CostMicros <= 0 {
		t.Fatalf("costMicros = %d, want positive", event.CostMicros)
	}
}

func TestRecordProviderCallEventUsesResponseUsage(t *testing.T) {
	store := &memoryProviderStore{}
	service := NewService(store)
	route := ResolvedModelRoute{
		Provider:  domain.ProviderConfig{ID: "priced-provider"},
		Model:     domain.ModelRef{ProviderID: "priced-provider", ModelID: "priced-model"},
		Transport: TransportOpenAICompatible,
		Definition: ProviderDefinition{
			ID: "priced-provider",
			Models: []domain.ModelInfo{{
				ID: "priced-model", ProviderID: "priced-provider", Name: "Priced Model",
				Pricing: map[string]float64{"input": 2, "output": 10},
			}},
		},
	}

	service.recordProviderCallEvent(context.Background(), route, modelRequirement{InputTokens: 999}, 0, 1, 0, domain.ChatResponse{
		Text:  "hello",
		Usage: &domain.TokenUsage{InputTokens: 12, OutputTokens: 8, TotalTokens: 20},
	}, nil)

	event := store.callEvents[0]
	if event.InputTokens != 12 || event.OutputTokens != 8 || event.TotalTokens != 20 {
		t.Fatalf("event = %+v, want response usage", event)
	}
	if event.Estimated {
		t.Fatalf("event.Estimated = true, want false for provider usage")
	}
}

func TestGetProviderUsageAggregatesEvents(t *testing.T) {
	store := &memoryProviderStore{callEvents: []domain.ProviderCallEvent{
		{ProviderID: "openai", Status: "success", InputTokens: 10, OutputTokens: 5, TotalTokens: 15, CostMicros: 100, Estimated: true, CreatedAt: "2026-01-01T00:00:00Z"},
		{ProviderID: "openai", Status: "failed", ErrorClass: "rate_limit", ErrorMessage: "too many", CreatedAt: "2026-01-01T00:01:00Z"},
		{ProviderID: "anthropic", Status: "success", InputTokens: 99, CreatedAt: "2026-01-01T00:02:00Z"},
	}}
	service := NewService(store)

	stats, err := service.GetProviderUsage(context.Background(), domain.ProviderUsageInput{ProviderID: "openai", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if stats.CallCount != 2 || stats.SuccessCount != 1 || stats.FailureCount != 1 {
		t.Fatalf("stats = %+v, want 2 calls with one success/failure", stats)
	}
	if stats.InputTokens != 10 || stats.OutputTokens != 5 || stats.TotalTokens != 15 || stats.CostMicros != 100 || !stats.Estimated {
		t.Fatalf("stats = %+v, want aggregated usage", stats)
	}
	if stats.LastErrorClass != "rate_limit" || stats.LastErrorProvider != "openai" {
		t.Fatalf("stats = %+v, want last rate_limit error", stats)
	}
}
