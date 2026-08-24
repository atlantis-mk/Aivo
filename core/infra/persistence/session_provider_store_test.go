package persistence

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"aivo/core/domain"
)

func TestAppConfigFallbackModelsPersistAcrossReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	ctx := context.Background()
	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg := domain.AppConfig{
		Initialized:     true,
		Provider:        &domain.ProviderConfig{ID: "openai", Type: "openai", Model: "gpt-5.5"},
		DefaultModel:    &domain.ModelRef{ProviderID: "openai", ModelID: "gpt-5.5"},
		AuxiliaryModel:  &domain.ModelRef{ProviderID: "openai", ModelID: "gpt-5.4-mini"},
		FallbackModels:  []domain.ModelRef{{ProviderID: "anthropic", ModelID: "claude-sonnet-4"}},
		ReasoningEffort: "medium",
		ServiceTier:     "default",
	}
	enableFallback := false
	bufferStreaming := false
	cfg.ProviderPolicy = domain.ProviderRuntimePolicy{
		EnableFallback:           &enableFallback,
		BufferStreamingFallback:  &bufferStreaming,
		MaxRetries:               0,
		RetryBaseDelayMs:         250,
		RateLimitCooldownSeconds: 45,
	}
	if err := store.SaveConfig(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	got, err := reopened.LoadConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.FallbackModels) != 1 || got.FallbackModels[0].ProviderID != "anthropic" || got.FallbackModels[0].ModelID != "claude-sonnet-4" {
		t.Fatalf("fallback models = %+v, want persisted anthropic fallback", got.FallbackModels)
	}
	if got.AuxiliaryModel == nil || got.AuxiliaryModel.ProviderID != "openai" || got.AuxiliaryModel.ModelID != "gpt-5.4-mini" {
		t.Fatalf("auxiliary model = %+v, want persisted openai gpt-5.4-mini", got.AuxiliaryModel)
	}
	if got.ProviderPolicy.EnableFallback == nil || *got.ProviderPolicy.EnableFallback || got.ProviderPolicy.BufferStreamingFallback == nil || *got.ProviderPolicy.BufferStreamingFallback {
		t.Fatalf("provider policy = %+v, want fallback/buffer disabled", got.ProviderPolicy)
	}
	if got.ProviderPolicy.MaxRetries != 0 || got.ProviderPolicy.RetryBaseDelayMs != 250 || got.ProviderPolicy.RateLimitCooldownSeconds != 45 {
		t.Fatalf("provider policy = %+v, want persisted retry/cooldown policy", got.ProviderPolicy)
	}
}

func TestProviderCallEventsPersistAcrossReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	ctx := context.Background()
	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	event := domain.ProviderCallEvent{
		ID:            "event-1",
		ProviderID:    "openai",
		ModelID:       "gpt-5.5",
		Transport:     "openai_responses",
		Status:        "failed",
		ErrorClass:    "rate_limit",
		ErrorMessage:  "too many requests",
		HTTPStatus:    429,
		LatencyMs:     123,
		InputTokens:   10,
		OutputTokens:  5,
		TotalTokens:   15,
		CostMicros:    100,
		Estimated:     true,
		Attempt:       2,
		FallbackIndex: 0,
		Streaming:     true,
		ToolCallCount: 1,
		CreatedAt:     "2026-01-01T00:00:00Z",
	}
	if err := store.SaveProviderCallEvent(ctx, event); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	events, err := reopened.ListProviderCallEvents(ctx, "openai", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %+v, want one event", events)
	}
	got := events[0]
	if got.ID != "event-1" || got.ErrorClass != "rate_limit" || got.HTTPStatus != 429 || !got.Streaming || got.ToolCallCount != 1 || got.TotalTokens != 15 || got.CostMicros != 100 || !got.Estimated {
		t.Fatalf("event = %+v, want persisted call event", got)
	}
}

func TestDeleteProviderRemovesProviderStateButKeepsCallEvents(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	ctx := context.Background()
	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProvider(ctx, domain.ProviderConfig{ID: "custom-api", Type: "openai_compatible", Model: "local-model"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProviderModelCache(ctx, domain.ProviderModelCache{ProviderID: "custom-api", Models: []domain.ModelInfo{{ID: "local-model", ProviderID: "custom-api", Name: "Local"}}, Status: "ready"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProviderValidation(ctx, domain.ProviderValidationResult{ProviderID: "custom-api", Status: "ready", Ready: true, CheckedAt: "2026-01-01T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProviderHealth(ctx, domain.ProviderHealth{ProviderID: "custom-api", Status: "ready", UpdatedAt: "2026-01-01T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProviderAuth(ctx, domain.ProviderAuthRecord{ProviderID: "custom-api", Method: "api-key", APIKeyRef: "ref", UpdatedAt: "2026-01-01T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProviderCallEvent(ctx, domain.ProviderCallEvent{ID: "event-1", ProviderID: "custom-api", ModelID: "local-model", Status: "success", CreatedAt: "2026-01-01T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteProvider(ctx, "custom-api"); err != nil {
		t.Fatal(err)
	}
	providers, err := store.ListProviders(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(providers) != 0 {
		t.Fatalf("providers = %+v, want deleted", providers)
	}
	cache, err := store.LoadProviderModelCache(ctx, "custom-api")
	if err != nil || cache != nil {
		t.Fatalf("cache = %+v, err = %v, want deleted", cache, err)
	}
	health, err := store.LoadProviderHealth(ctx, "custom-api")
	if err != nil || health != nil {
		t.Fatalf("health = %+v, err = %v, want deleted", health, err)
	}
	auth, err := store.LoadProviderAuth(ctx, "custom-api")
	if err != nil || auth != nil {
		t.Fatalf("auth = %+v, err = %v, want deleted", auth, err)
	}
	events, err := store.ListProviderCallEvents(ctx, "custom-api", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %+v, want retained audit event", events)
	}
}

func TestSessionRuntimeJSONRoundTripAndFilters(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "aivo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	session, err := store.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceWeb, Title: "Build runtime", ProjectPath: t.TempDir(), Metadata: map[string]string{"k": "v"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.UpsertCodingContext(ctx, domain.CodingContext{SessionID: session.ID, ProjectPath: session.ProjectPath, ChangedFiles: []string{"main.go"}, LanguageStack: []string{"go"}, Permissions: []string{"local-filesystem"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AppendSessionEvent(ctx, domain.SessionEvent{ID: "e1", SessionID: session.ID, Type: domain.EventTypeToolCall, Role: domain.EventRoleTool, Visibility: domain.EventVisibilityNormal, Content: "ran test", Payload: map[string]any{"command": "go test"}, TimeCreated: "2026-01-01T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	events, err := store.ListSessionEvents(ctx, session.ID, false, 10)
	if err != nil || len(events) != 1 || events[0].Payload["command"] != "go test" {
		t.Fatalf("events = %#v, %v", events, err)
	}
	results, err := store.ListRuntimeSessions(ctx, domain.ListSessionsRequest{Type: domain.SessionTypeCoding, Search: "runtime", Limit: 10})
	if err != nil || len(results) != 1 || results[0].Metadata["k"] != "v" {
		t.Fatalf("results = %#v, %v", results, err)
	}
}

func TestListSessionEventsReturnsLatestLimitInChronologicalOrder(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "aivo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	session, err := store.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeGeneric, Source: domain.SessionSourceDesktop, Title: "limited"})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if err := store.AppendSessionEvent(ctx, domain.SessionEvent{
			ID:          fmt.Sprintf("e%d", i),
			SessionID:   session.ID,
			Type:        domain.EventTypeUserMessage,
			Role:        domain.EventRoleUser,
			Visibility:  domain.EventVisibilityNormal,
			Content:     fmt.Sprintf("event %d", i),
			TimeCreated: fmt.Sprintf("2026-01-01T00:00:0%dZ", i),
		}); err != nil {
			t.Fatal(err)
		}
	}
	events, err := store.ListSessionEvents(ctx, session.ID, false, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 || events[0].Content != "event 2" || events[1].Content != "event 3" || events[2].Content != "event 4" {
		t.Fatalf("events = %#v", events)
	}
}
