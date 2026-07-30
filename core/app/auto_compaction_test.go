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
	triggered, err := service.maybeAutoCompactSessionContext(ctx, session.ID)
	if err != nil || !triggered {
		t.Fatalf("triggered = %v err = %v", triggered, err)
	}
	summary, err := service.LatestSummary(ctx, session.ID)
	if err != nil || summary == nil || summary.ToEventID == "" {
		t.Fatalf("summary = %#v err = %v", summary, err)
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
	triggered, err := service.maybeAutoCompactSessionContext(ctx, session.ID)
	if err != nil || triggered {
		t.Fatalf("triggered = %v err = %v", triggered, err)
	}
}
