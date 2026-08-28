package app

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestMaybeAutoCompactSessionContextTriggersAndRecordsMode(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	t.Setenv("HOME", t.TempDir())
	ctx := context.Background()
	root := t.TempDir()
	writeRuntimeConfigTestFile(t, filepath.Join(root, ".aivo", "config.json"), `{"compaction":{"auto":true,"thresholdPercent":1,"reserveTokens":1}}`)
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: root})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AppendEvent(ctx, domain.AppendEventRequest{
		SessionID: session.ID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser,
		Visibility: domain.EventVisibilityNormal, Content: strings.Repeat("context pressure ", 80),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AppendEvent(ctx, domain.AppendEventRequest{
		SessionID: session.ID, Type: domain.EventTypeAssistantMessage, Role: domain.EventRoleAssistant,
		Visibility: domain.EventVisibilityNormal, Content: strings.Repeat("settled response ", 80),
	}); err != nil {
		t.Fatal(err)
	}
	latestUser, err := service.AppendEvent(ctx, domain.AppendEventRequest{
		SessionID: session.ID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser,
		Visibility: domain.EventVisibilityNormal, Content: "new unanswered request",
	})
	if err != nil {
		t.Fatal(err)
	}
	triggered, err := service.maybeAutoCompactSessionContext(ctx, session.ID, nil)
	if err != nil || !triggered {
		t.Fatalf("triggered = %v err = %v", triggered, err)
	}
	summary, err := service.LatestSummary(ctx, session.ID)
	if err != nil || summary == nil || summary.ToEventID == "" {
		t.Fatalf("summary = %#v err = %v", summary, err)
	}
	if summary.ToEventID == latestUser.ID {
		t.Fatal("automatic compaction included the newest unanswered user message")
	}
	events, err := service.ListEvents(ctx, session.ID, false, 20)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, event := range events {
		if event.Payload["kind"] == "context_compacted" && event.Payload["mode"] == "automatic" {
			found = true
		}
	}
	if !found {
		t.Fatalf("automatic compaction event missing: %#v", events)
	}
}

func TestMaybeAutoCompactSessionContextHonorsDisabledConfig(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	t.Setenv("HOME", t.TempDir())
	ctx := context.Background()
	root := t.TempDir()
	writeRuntimeConfigTestFile(t, filepath.Join(root, "aivo.json"), `{"compaction":{"auto":false,"thresholdPercent":1}}`)
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: root})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AppendEvent(ctx, domain.AppendEventRequest{SessionID: session.ID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser, Visibility: domain.EventVisibilityNormal, Content: strings.Repeat("x", 5000)}); err != nil {
		t.Fatal(err)
	}
	triggered, err := service.maybeAutoCompactSessionContext(ctx, session.ID, nil)
	if err != nil || triggered {
		t.Fatalf("triggered = %v err = %v", triggered, err)
	}
}

func TestResolveCompactionPressureUsesProviderContextAtDefaultEightyPercent(t *testing.T) {
	t.Setenv("AIVO_MODELS_CACHE", filepath.Join(t.TempDir(), "models-dev.json"))
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	pressure := service.resolveCompactionPressure(
		context.Background(),
		domain.Session{},
		&domain.ModelRef{ProviderID: "openai", ModelID: "gpt-5.4"},
		domain.CompactionRuntimeConfig{},
	)
	if pressure.ContextWindowTokens != 400000 || pressure.TriggerTokens != 320000 {
		t.Fatalf("pressure = %#v", pressure)
	}
	if pressure.ThresholdPercent != 80 || pressure.CapacitySource != "provider_catalog" {
		t.Fatalf("pressure metadata = %#v", pressure)
	}
}

func TestResolveCompactionPressureHonorsDeclaredAutoCompactLimit(t *testing.T) {
	service := NewService(&memoryProviderStore{modelCaches: map[string]domain.ProviderModelCache{
		"openai": {ProviderID: "openai", Models: []domain.ModelInfo{{
			ID: "gpt-codex", ProviderID: "openai", ContextLength: 272000, AutoCompactTokenLimit: 240000,
		}}},
	}})
	defer service.Shutdown()
	pressure := service.resolveCompactionPressure(
		context.Background(), domain.Session{},
		&domain.ModelRef{ProviderID: "openai", ModelID: "gpt-codex"},
		domain.CompactionRuntimeConfig{ThresholdPercent: 100},
	)
	if pressure.ContextWindowTokens != 272000 || pressure.AutoCompactTokenLimit != 240000 || pressure.TriggerTokens != 240000 {
		t.Fatalf("pressure = %#v", pressure)
	}
	if pressure.CapacitySource != "provider_cache" {
		t.Fatalf("capacity source = %q", pressure.CapacitySource)
	}
}

func TestModelVisibleHistoryUsesSummaryAndOnlyPostBoundaryMessages(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "fixed test session", ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	oldEvent, err := service.AppendEvent(ctx, domain.AppendEventRequest{
		SessionID: session.ID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser,
		Visibility: domain.EventVisibilityNormal, Content: "old message that must not be projected",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateSummary(ctx, domain.CreateSummaryRequest{
		SessionID: session.ID, ToEventID: oldEvent.ID, Summary: "durable compacted summary",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AppendEvent(ctx, domain.AppendEventRequest{
		SessionID: session.ID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser,
		Visibility: domain.EventVisibilityNormal, Content: "new post-boundary message",
	}); err != nil {
		t.Fatal(err)
	}
	messages, err := service.modelVisibleSessionHistory(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	var projected strings.Builder
	for _, message := range messages {
		projected.WriteString(message.Text)
		projected.WriteByte('\n')
	}
	if strings.Contains(projected.String(), oldEvent.Content) {
		t.Fatalf("old event leaked past summary boundary: %s", projected.String())
	}
	if !strings.Contains(projected.String(), "durable compacted summary") || !strings.Contains(projected.String(), "new post-boundary message") {
		t.Fatalf("summary or post-boundary message missing: %s", projected.String())
	}
}

func TestManualCompactionRejectsRunningSession(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AppendEvent(ctx, domain.AppendEventRequest{
		SessionID: session.ID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser,
		Visibility: domain.EventVisibilityNormal, Content: "settled history",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.store.UpsertSessionExecutionState(ctx, domain.SessionExecutionState{SessionID: session.ID, Status: domain.ExecutionStatusRunning}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CompactSessionContext(ctx, domain.CompactSessionContextInput{SessionID: session.ID}); err == nil {
		t.Fatal("expected running session compaction to be rejected")
	}
}
