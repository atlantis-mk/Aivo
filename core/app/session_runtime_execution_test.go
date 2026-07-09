package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"aivo/core/domain"
	"aivo/core/infra/persistence"
)

func TestSubmitSessionMessageStreamingFailureFinalizesTurn(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	updated := make(chan string, 2)
	service.SetSessionUpdatedHook(func(sessionID string, _ *domain.Session) {
		updated <- sessionID
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream stopped", http.StatusInternalServerError)
	}))
	defer server.Close()
	if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{ProviderID: "custom-api", Type: "openai-compatible", BaseURL: server.URL, ModelID: "test-model", APIKey: "test-key", Method: "api-key"}); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeGeneric, Source: domain.SessionSourceDesktop})
	if err != nil {
		t.Fatal(err)
	}
	run, err := service.SubmitSessionMessageStreaming(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "stream"})
	if err != nil {
		t.Fatal(err)
	}
	if run.Turn.Status != domain.TurnStatusRunning {
		t.Fatalf("initial run = %#v", run)
	}
	select {
	case got := <-updated:
		if got != session.ID {
			t.Fatalf("updated session = %q, want %q", got, session.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for async session update")
	}
	var turns []domain.Turn
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		turns, err = service.ListTurns(ctx, session.ID, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(turns) == 1 && turns[0].Status == domain.TurnStatusFailed && turns[0].TimeCompleted != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(turns) != 1 || turns[0].Status != domain.TurnStatusFailed || turns[0].TimeCompleted == "" {
		t.Fatalf("turns = %#v", turns)
	}
	events, err := service.ListEvents(ctx, session.ID, false, 20)
	if err != nil {
		t.Fatal(err)
	}
	var foundError bool
	for _, event := range events {
		foundError = foundError || event.Type == domain.EventTypeError
	}
	if !foundError {
		t.Fatalf("missing error event: %#v", events)
	}
}

func TestCancelSessionTurnStopsStreamingProviderRequest(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	requestStarted := make(chan struct{})
	requestCancelled := make(chan struct{})
	var closeStarted sync.Once
	var closeCancelled sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		closeStarted.Do(func() { close(requestStarted) })
		w.Header().Set("Content-Type", "text/event-stream")
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		select {
		case <-r.Context().Done():
			closeCancelled.Do(func() { close(requestCancelled) })
		case <-time.After(5 * time.Second):
			t.Errorf("provider request was not cancelled")
		}
	}))
	defer server.Close()
	if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{ProviderID: "openai", Type: "openai", BaseURL: server.URL, ModelID: "test-model", APIKey: "test-key", Method: "api-key"}); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeGeneric, Source: domain.SessionSourceDesktop})
	if err != nil {
		t.Fatal(err)
	}
	run, err := service.SubmitSessionMessageStreaming(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "stream until cancelled"})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-requestStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for provider request")
	}
	cancelled, err := service.CancelTurn(ctx, domain.CancelTurnRequest{TurnID: run.Turn.ID, Reason: "User stopped generation"})
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != domain.TurnStatusCancelled {
		t.Fatalf("cancelled turn = %#v", cancelled)
	}
	select {
	case <-requestCancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for provider request cancellation")
	}
	var turns []domain.Turn
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		turns, err = service.ListTurns(ctx, session.ID, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(turns) == 1 && turns[0].Status == domain.TurnStatusCancelled && turns[0].TimeCompleted != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(turns) != 1 || turns[0].Status != domain.TurnStatusCancelled || turns[0].TimeCompleted == "" {
		t.Fatalf("turns = %#v", turns)
	}
	events, err := service.ListEvents(ctx, session.ID, false, 20)
	if err != nil {
		t.Fatal(err)
	}
	var foundCancelNote bool
	for _, event := range events {
		foundCancelNote = foundCancelNote || event.Type == domain.EventTypeSystemNote && strings.Contains(event.Content, "User stopped generation")
		if event.Type == domain.EventTypeAssistantMessage {
			t.Fatalf("assistant event should not be persisted after cancellation: %#v", event)
		}
	}
	if !foundCancelNote {
		t.Fatalf("missing cancellation note: %#v", events)
	}
}

func TestExecutionControlInterruptCompactCursorAndQueuedInput(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	user, err := service.AppendEvent(ctx, domain.AppendEventRequest{SessionID: session.ID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser, Content: "start"})
	if err != nil {
		t.Fatal(err)
	}
	turn, err := service.StartTurn(ctx, domain.StartTurnRequest{SessionID: session.ID, UserEventID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveToolCall(ctx, domain.CreateToolCallRequest{ID: "call_running", SessionID: session.ID, TurnID: turn.ID, Name: "bash", Status: domain.ToolCallStatusRunning}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.store.UpsertSessionExecutionState(ctx, domain.SessionExecutionState{SessionID: session.ID, TurnID: turn.ID, Status: domain.ExecutionStatusRunning}); err != nil {
		t.Fatal(err)
	}
	queued, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "steer this run", Delivery: domain.InputDeliverySteer})
	if err != nil {
		t.Fatal(err)
	}
	if queued.UserEvent.Visibility != domain.EventVisibilityInternal {
		t.Fatalf("queued input event = %#v, want internal placeholder", queued.UserEvent)
	}
	pending, err := service.store.ListPendingSessionInputs(ctx, session.ID, domain.PendingInputStatusPending)
	if err != nil || len(pending) != 1 || pending[0].Delivery != domain.InputDeliverySteer {
		t.Fatalf("pending inputs = %#v err=%v", pending, err)
	}
	state, err := service.InterruptSessionExecution(ctx, domain.InterruptSessionExecutionInput{SessionID: session.ID, Reason: "stop"})
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != domain.ExecutionStatusInterrupted {
		t.Fatalf("state = %#v", state)
	}
	calls, err := service.ListToolCalls(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].Status != domain.ToolCallStatusInterrupted {
		t.Fatalf("tool calls after interrupt = %#v", calls)
	}
	firstPage, err := service.ListSessionEventsAfterCursor(ctx, domain.ListSessionEventsAfterCursorInput{SessionID: session.ID, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(firstPage.Events) != 1 || firstPage.NextCursor == "" {
		t.Fatalf("first cursor page = %#v", firstPage)
	}
	secondPage, err := service.ListSessionEventsAfterCursor(ctx, domain.ListSessionEventsAfterCursorInput{SessionID: session.ID, Cursor: firstPage.NextCursor, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(secondPage.Events) == 0 {
		t.Fatalf("second cursor page empty after cursor %q", firstPage.NextCursor)
	}
	compacted, err := service.CompactSessionContext(ctx, domain.CompactSessionContextInput{SessionID: session.ID, CharacterBudget: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if compacted.State.Status != domain.ExecutionStatusIdle || compacted.Summary.ID == "" || compacted.CompactedEventID == "" {
		t.Fatalf("compacted = %#v", compacted)
	}
	events, err := service.ListEvents(ctx, session.ID, false, 100)
	if err != nil {
		t.Fatal(err)
	}
	if !sessionEventContains(events, domain.EventTypeSummary, compacted.Summary.Summary) {
		t.Fatalf("summary event missing from events: %#v", events)
	}
}

func TestStartupRecoveryMarksRunningToolCallsInterrupted(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	store, err := persistence.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store)
	ctx := context.Background()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveToolCall(ctx, domain.CreateToolCallRequest{ID: "call_startup", SessionID: session.ID, Name: "bash", Status: domain.ToolCallStatusRunning}); err != nil {
		t.Fatal(err)
	}
	service.Shutdown()
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := persistence.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	recovered := NewService(reopened)
	defer recovered.Shutdown()
	calls, err := recovered.ListToolCalls(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].Status != domain.ToolCallStatusInterrupted {
		t.Fatalf("calls after service startup = %#v", calls)
	}
}
