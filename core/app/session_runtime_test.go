package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"aivo/core/domain"
	"aivo/core/infra/persistence"
)

func newSessionTestService(t *testing.T) (*Service, func()) {
	t.Helper()
	store, err := persistence.Open(filepath.Join(t.TempDir(), "aivo.db"))
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store)
	return service, func() { _ = store.Close() }
}

func TestSessionLifecycleVisibilityAndContextBuilder(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	service.now = func() time.Time { return time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC) }
	ctx := context.Background()
	service.now = func() time.Time { return time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC) }
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "Runtime", Goal: "Ship continuity", ProjectPath: t.TempDir(), SystemPromptSnapshot: "system"})
	if err != nil {
		t.Fatal(err)
	}
	normal, err := service.AppendEvent(ctx, domain.AppendEventRequest{SessionID: session.ID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser, Content: "visible"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AppendEvent(ctx, domain.AppendEventRequest{SessionID: session.ID, Type: domain.EventTypeSystemNote, Role: domain.EventRoleSystem, Visibility: domain.EventVisibilityInternal, Content: "secret"}); err != nil {
		t.Fatal(err)
	}
	turn, err := service.StartTurn(ctx, domain.StartTurnRequest{SessionID: session.ID, UserEventID: normal.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.FailTurn(ctx, domain.FailTurnRequest{TurnID: turn.ID, Error: "provider unavailable"}); err != nil {
		t.Fatal(err)
	}
	events, err := service.ListEvents(ctx, session.ID, false, 20)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Content == "secret" {
			t.Fatalf("internal event leaked in normal listing")
		}
	}
	result, err := service.BuildSessionContext(ctx, domain.BuildSessionContextRequest{SessionID: session.ID, CurrentInput: "next", CharacterBudget: 500})
	if err != nil {
		t.Fatal(err)
	}
	joined := ""
	for _, section := range result.Sections {
		joined += section.Content
	}
	if !contains(joined, "visible") || contains(joined, "secret") {
		t.Fatalf("context content = %q", joined)
	}
}

func TestSessionContextAndResumeDoNotInjectProjectInstructions(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	projectRoot := t.TempDir()
	writeSessionProjectFile(t, projectRoot, "AGENTS.md", "# Agent rules\nUse the project rules in context.\n")
	writeSessionProjectFile(t, projectRoot, ".env", "SECRET=value\n")
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: projectRoot})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.BuildSessionContext(ctx, domain.BuildSessionContextRequest{SessionID: session.ID, CharacterBudget: 4000})
	if err != nil {
		t.Fatal(err)
	}
	for _, section := range result.Sections {
		if section.Name == "project_instructions" {
			t.Fatalf("project instructions section should not be injected: %q", section.Content)
		}
	}
	joined := ""
	for _, section := range result.Sections {
		joined += section.Content
	}
	if contains(joined, "project rules") || contains(joined, "SECRET=value") {
		t.Fatalf("project file content leaked into context: %q", joined)
	}
	recap, err := service.ResumeRecap(ctx, domain.ResumeSessionRequest{SessionID: session.ID})
	if err != nil {
		t.Fatal(err)
	}
	if recap.ProjectPath != projectRoot {
		t.Fatalf("resume project path = %q, want %q", recap.ProjectPath, projectRoot)
	}
}

func TestCreateRuntimeSessionWithoutProjectPathCreatesManagedCodingContext(t *testing.T) {
	t.Setenv(managedWorkspaceRootEnv, t.TempDir())
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	service.now = func() time.Time { return time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC) }
	ctx := context.Background()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, Title: "https://github.com/example/repo"})
	if err != nil {
		t.Fatal(err)
	}
	cc, err := service.GetCodingContext(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertManagedWorkspace(t, cc.ProjectPath, filepath.Join("2026-06-27", workspaceSlug(session.ID)))
}

func TestCreateCodingSessionDefaultsToCodeAgent(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if session.AgentMode != domain.AgentModeCode {
		t.Fatalf("agent mode = %q, want %q", session.AgentMode, domain.AgentModeCode)
	}
}

func TestSubmitSessionMessageRecreatesDeletedManagedWorkspace(t *testing.T) {
	t.Setenv(managedWorkspaceRootEnv, t.TempDir())
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	service.now = func() time.Time { return time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC) }
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer server.Close()
	if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{ProviderID: "custom-api", Type: "openai-compatible", BaseURL: server.URL, ModelID: "test-model", APIKey: "test-key", Method: "api-key"}); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop})
	if err != nil {
		t.Fatal(err)
	}
	before, err := service.GetCodingContext(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	wantSuffix := filepath.Join("2026-06-27", workspaceSlug(session.ID))
	assertManagedWorkspace(t, before.ProjectPath, wantSuffix)
	if err := os.RemoveAll(before.ProjectPath); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "hello"}); err != nil {
		t.Fatal(err)
	}
	after, err := service.GetCodingContext(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertManagedWorkspace(t, after.ProjectPath, wantSuffix)
	if after.ProjectPath != before.ProjectPath {
		t.Fatalf("managed workspace path changed: before=%q after=%q", before.ProjectPath, after.ProjectPath)
	}
}

func assertManagedWorkspace(t *testing.T, path string, wantSuffix string) {
	t.Helper()
	if strings.TrimSpace(path) == "" {
		t.Fatal("managed workspace path is empty")
	}
	root, err := managedWorkspaceRoot()
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		t.Fatalf("path = %q, want inside managed root %q", path, root)
	}
	if rel != wantSuffix {
		t.Fatalf("path suffix = %q, want %q", rel, wantSuffix)
	}
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		t.Fatalf("workspace dir missing: %q, %v", path, err)
	}
}

func TestSummaryCheckpointForkAndResume(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	projectPath := t.TempDir()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "Parent", Goal: "Goal", ProjectPath: projectPath})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateSummary(ctx, domain.CreateSummaryRequest{SessionID: session.ID, Summary: "Summary", OpenTasks: []string{"todo"}, NextSuggestedAction: "continue"}); err != nil {
		t.Fatal(err)
	}
	checkpoint, err := service.CreateCheckpoint(ctx, domain.CreateCheckpointRequest{SessionID: session.ID, ConversationSummary: "Checkpoint", OpenTodos: []string{"todo"}, NextSuggestedAction: "resume"})
	if err != nil {
		t.Fatal(err)
	}
	if len(checkpoint.KnownIssues) == 0 {
		t.Fatalf("checkpoint without git metadata should record known issue")
	}
	fork, err := service.ForkSession(ctx, domain.ForkSessionRequest{SessionID: session.ID, Title: "Fork"})
	if err != nil {
		t.Fatal(err)
	}
	if fork.ForkedFromSessionID != session.ID || fork.Status != domain.SessionStatusActive {
		t.Fatalf("fork = %#v", fork)
	}
	if _, err := service.AppendEvent(ctx, domain.AppendEventRequest{SessionID: fork.ID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser, Content: "fork only"}); err != nil {
		t.Fatal(err)
	}
	sourceEvents, err := service.ListEvents(ctx, session.ID, false, 20)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range sourceEvents {
		if event.Content == "fork only" {
			t.Fatalf("fork event leaked to source session")
		}
	}
	recap, err := service.ResumeRecap(ctx, domain.ResumeSessionRequest{SessionID: session.ID})
	if err != nil {
		t.Fatal(err)
	}
	if recap.LatestSummary == nil || recap.LatestCheckpoint == nil || recap.ProjectPath == "" || recap.NextSuggestedAction == "" {
		t.Fatalf("recap = %#v", recap)
	}
	latest, err := service.ContinueProjectSession(ctx, projectPath)
	if err != nil || latest == nil || latest.SessionID == "" {
		t.Fatalf("continue project = %#v, %v", latest, err)
	}
}

func TestSubmitMessageUsesDeterministicFallbackWithoutProvider(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	run, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "Add session runtime"})
	if err != nil {
		t.Fatal(err)
	}
	if run.Turn.Status != domain.TurnStatusCompleted || run.AssistantEvent == nil || !contains(run.AssistantEvent.Content, "I recorded your request") {
		t.Fatalf("run = %#v", run)
	}
	events, err := service.ListEvents(ctx, session.ID, false, 20)
	if err != nil {
		t.Fatal(err)
	}
	var user, assistant bool
	for _, event := range events {
		user = user || event.Type == domain.EventTypeUserMessage
		assistant = assistant || event.Type == domain.EventTypeAssistantMessage
	}
	if !user || !assistant {
		t.Fatalf("events = %#v", events)
	}
	updated, err := service.GetRuntimeSession(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Add session runtime" {
		t.Fatalf("title = %q", updated.Title)
	}
}

func TestUpdateAndDeleteSessionEventAffectVisibleContext(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeGeneric, Source: domain.SessionSourceDesktop, Title: "Manual title"})
	if err != nil {
		t.Fatal(err)
	}
	user, err := service.AppendEvent(ctx, domain.AppendEventRequest{SessionID: session.ID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser, Content: "old request"})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.UpdateSessionEvent(ctx, domain.UpdateSessionEventRequest{EventID: user.ID, Content: "new request"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Content != "new request" {
		t.Fatalf("updated event = %#v", updated)
	}
	visible, err := service.ListEvents(ctx, session.ID, false, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(visible) != 1 || visible[0].Content != "new request" {
		t.Fatalf("visible events after edit = %#v", visible)
	}
	contextResult, err := service.BuildSessionContext(ctx, domain.BuildSessionContextRequest{SessionID: session.ID, CharacterBudget: 500})
	if err != nil {
		t.Fatal(err)
	}
	joined := ""
	for _, section := range contextResult.Sections {
		joined += section.Content
	}
	if !contains(joined, "new request") || contains(joined, "old request") {
		t.Fatalf("context after edit = %q", joined)
	}
	deleted, err := service.DeleteSessionEvent(ctx, domain.DeleteSessionEventRequest{EventID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	if deleted.Visibility != domain.EventVisibilityHidden {
		t.Fatalf("deleted event = %#v", deleted)
	}
	visible, err = service.ListEvents(ctx, session.ID, false, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(visible) != 0 {
		t.Fatalf("visible events after delete = %#v", visible)
	}
	all, err := service.ListEvents(ctx, session.ID, true, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].Visibility != domain.EventVisibilityHidden {
		t.Fatalf("all events after delete = %#v", all)
	}
}

func TestRetrySessionTurnHidesOriginalEventsAndCreatesNewTurn(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeGeneric, Source: domain.SessionSourceDesktop})
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "retry this"})
	if err != nil {
		t.Fatal(err)
	}
	if first.AssistantEvent == nil || first.Turn.Status != domain.TurnStatusCompleted {
		t.Fatalf("first run = %#v", first)
	}
	retry, err := service.RetrySessionTurnStreaming(ctx, domain.RetrySessionTurnRequest{TurnID: first.Turn.ID})
	if err != nil {
		t.Fatal(err)
	}
	if retry.Turn.ID == first.Turn.ID || retry.UserEvent.ID == first.UserEvent.ID {
		t.Fatalf("retry reused original ids: first=%#v retry=%#v", first, retry)
	}
	deadline := time.Now().Add(2 * time.Second)
	var visible []domain.SessionEvent
	for time.Now().Before(deadline) {
		visible, err = service.ListEvents(ctx, session.ID, false, 20)
		if err != nil {
			t.Fatal(err)
		}
		foundNewAssistant := false
		for _, event := range visible {
			if event.Type == domain.EventTypeAssistantMessage && event.TurnID == retry.Turn.ID {
				foundNewAssistant = true
			}
		}
		if foundNewAssistant {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	var visibleUsers, visibleAssistants int
	for _, event := range visible {
		switch event.Type {
		case domain.EventTypeUserMessage:
			visibleUsers++
			if event.ID == first.UserEvent.ID {
				t.Fatalf("original user event is still visible: %#v", visible)
			}
		case domain.EventTypeAssistantMessage:
			visibleAssistants++
			if event.ID == first.AssistantEvent.ID {
				t.Fatalf("original assistant event is still visible: %#v", visible)
			}
		}
	}
	if visibleUsers != 1 || visibleAssistants != 1 {
		t.Fatalf("visible events after retry = %#v", visible)
	}
	all, err := service.ListEvents(ctx, session.ID, true, 20)
	if err != nil {
		t.Fatal(err)
	}
	var originalUserHidden, originalAssistantHidden bool
	for _, event := range all {
		if event.ID == first.UserEvent.ID && event.Visibility == domain.EventVisibilityHidden {
			originalUserHidden = true
		}
		if event.ID == first.AssistantEvent.ID && event.Visibility == domain.EventVisibilityHidden {
			originalAssistantHidden = true
		}
	}
	if !originalUserHidden || !originalAssistantHidden {
		t.Fatalf("original events were not hidden: %#v", all)
	}
}

func TestSessionTurnDiffRevertAndUnrevert(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	projectPath := t.TempDir()
	targetPath := filepath.Join(projectPath, "README.md")
	if err := os.WriteFile(targetPath, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, ProjectPath: projectPath})
	if err != nil {
		t.Fatal(err)
	}
	user, err := service.AppendEvent(ctx, domain.AppendEventRequest{SessionID: session.ID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser, Content: "edit readme"})
	if err != nil {
		t.Fatal(err)
	}
	turn, err := service.StartTurn(ctx, domain.StartTurnRequest{SessionID: session.ID, UserEventID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	call := domain.ChatToolCall{ID: "call_edit", Name: "edit_file", Arguments: json.RawMessage(`{"path":"README.md","oldString":"old","newString":"new"}`)}
	result := NewEditFileTool(projectPath).Execute(ctx, call.Arguments, domain.ToolExecutionContext{WorkspaceRoot: projectPath, SessionID: session.ID, TurnID: turn.ID, ToolCallID: call.ID})
	if !result.OK {
		t.Fatalf("edit result = %#v", result)
	}
	if err := service.recordToolResult(ctx, session.ID, turn.ID, call, result); err != nil {
		t.Fatal(err)
	}
	diff, err := service.GetSessionTurnDiff(ctx, domain.GetSessionTurnDiffRequest{SessionID: session.ID, TurnID: turn.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Files) != 1 || !diff.Files[0].Revertible || diff.Files[0].Unrevertible {
		t.Fatalf("diff after edit = %#v", diff)
	}
	if _, err := service.ApplySessionTurnFileState(ctx, domain.ApplySessionTurnFileStateRequest{SessionID: session.ID, TurnID: turn.ID, TargetState: "before"}); err != nil {
		t.Fatal(err)
	}
	events, err := service.ListEvents(ctx, session.ID, false, 20)
	if err != nil {
		t.Fatal(err)
	}
	if !sessionEventContains(events, domain.EventTypeSystemNote, "File changes reverted", "README.md") {
		t.Fatalf("events after revert = %#v, want revert audit event", events)
	}
	raw, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "old\n" {
		t.Fatalf("file after revert = %q", raw)
	}
	diff, err = service.GetSessionTurnDiff(ctx, domain.GetSessionTurnDiffRequest{SessionID: session.ID, TurnID: turn.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Files) != 1 || diff.Files[0].Revertible || !diff.Files[0].Unrevertible {
		t.Fatalf("diff after revert = %#v", diff)
	}
	if _, err := service.ApplySessionTurnFileState(ctx, domain.ApplySessionTurnFileStateRequest{SessionID: session.ID, TurnID: turn.ID, TargetState: "after"}); err != nil {
		t.Fatal(err)
	}
	events, err = service.ListEvents(ctx, session.ID, false, 20)
	if err != nil {
		t.Fatal(err)
	}
	if !sessionEventContains(events, domain.EventTypeSystemNote, "File changes restored", "README.md") {
		t.Fatalf("events after unrevert = %#v, want restore audit event", events)
	}
	raw, err = os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "new\n" {
		t.Fatalf("file after unrevert = %q", raw)
	}
	if err := os.WriteFile(targetPath, []byte("manual\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ApplySessionTurnFileState(ctx, domain.ApplySessionTurnFileStateRequest{SessionID: session.ID, TurnID: turn.ID, TargetState: "before"}); err == nil {
		t.Fatal("expected stale file to block revert")
	}
	raw, err = os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "manual\n" {
		t.Fatalf("stale revert changed file = %q", raw)
	}
	eventsAfterFailedRevert, err := service.ListEvents(ctx, session.ID, false, 20)
	if err != nil {
		t.Fatal(err)
	}
	if countSessionEventsContaining(eventsAfterFailedRevert, domain.EventTypeSystemNote, "File changes reverted") != 1 {
		t.Fatalf("events after failed revert = %#v, want no extra revert audit event", eventsAfterFailedRevert)
	}
}

func sessionEventContains(events []domain.SessionEvent, eventType string, parts ...string) bool {
	for _, event := range events {
		if event.Type != eventType {
			continue
		}
		matches := true
		for _, part := range parts {
			if !strings.Contains(event.Content, part) {
				matches = false
				break
			}
		}
		if matches {
			return true
		}
	}
	return false
}

func countSessionEventsContaining(events []domain.SessionEvent, eventType string, part string) int {
	count := 0
	for _, event := range events {
		if event.Type == eventType && strings.Contains(event.Content, part) {
			count++
		}
	}
	return count
}

func TestSubmitSessionMessageStreamsDeltas(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	var deltas []string
	service.SetAssistantDeltaHook(func(sessionID string, turnID string, delta string) {
		deltas = append(deltas, delta)
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"hello\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\" world\"}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()
	if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{ProviderID: "openai", Type: "openai", BaseURL: server.URL, ModelID: "test-model", APIKey: "test-key", Method: "api-key"}); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeGeneric, Source: domain.SessionSourceDesktop})
	if err != nil {
		t.Fatal(err)
	}
	run, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "stream"})
	if err != nil {
		t.Fatal(err)
	}
	if run.AssistantEvent == nil || run.AssistantEvent.Content != "hello world" {
		t.Fatalf("run = %#v", run)
	}
	if strings.Join(deltas, "") != "hello world" || len(deltas) != 2 {
		t.Fatalf("deltas = %#v", deltas)
	}
}

func TestSaveToolCallEmitsCreatedAndUpdatedHooks(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	var events []struct {
		created bool
		call    domain.ToolCall
	}
	service.SetToolCallUpdatedHook(func(sessionID string, turnID string, call domain.ToolCall, created bool) {
		events = append(events, struct {
			created bool
			call    domain.ToolCall
		}{created: created, call: call})
	})
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeGeneric, Source: domain.SessionSourceDesktop})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveToolCall(ctx, domain.CreateToolCallRequest{ID: "call_1", SessionID: session.ID, Name: "edit_file", Status: domain.ToolCallStatusRunning}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveToolCall(ctx, domain.CreateToolCallRequest{ID: "call_1", SessionID: session.ID, Name: "edit_file", Status: domain.ToolCallStatusSuccess, ResultSummary: "done"}); err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("events = %#v, want created and updated", events)
	}
	if !events[0].created || events[0].call.ID != "call_1" {
		t.Fatalf("first event = %#v, want created call_1", events[0])
	}
	if events[1].created || events[1].call.Status != domain.ToolCallStatusSuccess {
		t.Fatalf("second event = %#v, want success update", events[1])
	}
}

func TestReplaySessionToolCallCreatesFreshToolCall(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	root := t.TempDir()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, ProjectPath: root})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetPermissionMode(ctx, domain.PermissionModeInput{SessionID: session.ID, Mode: domain.PermissionModeFullAccess}); err != nil {
		t.Fatal(err)
	}
	original, err := service.SaveToolCall(ctx, domain.CreateToolCallRequest{
		ID:        "call_failed",
		SessionID: session.ID,
		Name:      "bash",
		Arguments: map[string]any{"command": "pwd"},
		Status:    domain.ToolCallStatusFailed,
		Error:     "previous run failed",
	})
	if err != nil {
		t.Fatal(err)
	}

	replayed, err := service.ReplaySessionToolCall(ctx, domain.ReplaySessionToolCallRequest{SessionID: session.ID, ToolCallID: original.ID})
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ID == original.ID || !strings.HasPrefix(replayed.ID, "replay_") {
		t.Fatalf("replayed id = %q, original = %q", replayed.ID, original.ID)
	}
	if replayed.Name != "bash" || replayed.Status != domain.ToolCallStatusSuccess {
		t.Fatalf("replayed call = %#v, want successful bash call", replayed)
	}
	if replayed.Result["replayOfToolCallId"] != original.ID || replayed.Result["replayOfToolName"] != original.Name {
		t.Fatalf("replayed metadata = %#v, want replay source", replayed.Result)
	}
	if !strings.Contains(replayed.ResultSummary, filepath.Base(root)) && !strings.Contains(fmt.Sprint(replayed.Result), root) {
		t.Fatalf("replayed result = %#v, want workspace pwd output", replayed)
	}
	calls, err := service.ListToolCalls(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	var originalStillFailed, replayFound bool
	for _, call := range calls {
		if call.ID == original.ID && call.Status == domain.ToolCallStatusFailed {
			originalStillFailed = true
		}
		if call.ID == replayed.ID && call.Status == domain.ToolCallStatusSuccess {
			replayFound = true
		}
	}
	if !originalStillFailed || !replayFound {
		t.Fatalf("tool calls after replay = %#v", calls)
	}
	events, err := service.ListEvents(ctx, session.ID, false, 20)
	if err != nil {
		t.Fatal(err)
	}
	if !sessionEventContains(events, domain.EventTypeSystemNote, "Tool call replay succeeded", original.ID) {
		t.Fatalf("events after replay = %#v, want replay audit event", events)
	}
}

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

func TestSubmitSessionMessageBuildsModelVisibleContext(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time {
		now = now.Add(time.Millisecond)
		return now
	}
	var captured []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		captured = body.Messages
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"context ok"}}]}`))
	}))
	defer server.Close()
	if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{ProviderID: "custom-api", Type: "openai-compatible", BaseURL: server.URL, ModelID: "test-model", APIKey: "test-key", Method: "api-key"}); err != nil {
		t.Fatal(err)
	}
	projectRoot := t.TempDir()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{
		Type:                 domain.SessionTypeCoding,
		Source:               domain.SessionSourceDesktop,
		Goal:                 "Keep the conversation coherent",
		ProjectPath:          projectRoot,
		SystemPromptSnapshot: "Prefer concise implementation notes",
	})
	if err != nil {
		t.Fatal(err)
	}
	writeSessionProjectFile(t, projectRoot, "AGENTS.md", "# Agent rules\nAlways run focused tests before final response.\n")
	writeSessionProjectFile(t, projectRoot, ".env", "SECRET=value\n")
	for i := 0; i < 35; i++ {
		if _, err := service.AppendEvent(ctx, domain.AppendEventRequest{SessionID: session.ID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser, Content: "old user message " + strconv.Itoa(i)}); err != nil {
			t.Fatal(err)
		}
		if _, err := service.AppendEvent(ctx, domain.AppendEventRequest{SessionID: session.ID, Type: domain.EventTypeAssistantMessage, Role: domain.EventRoleAssistant, Content: "old assistant message " + strconv.Itoa(i)}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := service.CreateSummary(ctx, domain.CreateSummaryRequest{SessionID: session.ID, Summary: "Durable architecture summary", OpenTasks: []string{"preserve current user request"}, NextSuggestedAction: "continue from summary"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "current request"}); err != nil {
		t.Fatal(err)
	}
	if len(captured) == 0 {
		t.Fatal("provider did not receive messages")
	}
	if captured[0].Role != "system" || !contains(captured[0].Content, "Aivo") {
		t.Fatalf("first message = %#v", captured[0])
	}
	joined := ""
	for _, message := range captured {
		joined += "\n" + message.Role + ": " + message.Content
	}
	if !contains(joined, "Durable architecture summary") ||
		!contains(joined, "Keep the conversation coherent") ||
		!contains(joined, "current request") {
		t.Fatalf("context missing expected content: %s", joined)
	}
	if contains(joined, "Always run focused tests") {
		t.Fatalf("project instructions leaked into context: %s", joined)
	}
	if contains(joined, "SECRET=value") {
		t.Fatalf("sensitive project file leaked into context: %s", joined)
	}
	if contains(joined, "old user message 0") {
		t.Fatalf("oldest tail leaked into bounded context: %s", joined)
	}
	if len(captured) >= 72 {
		t.Fatalf("captured %d messages, want bounded context", len(captured))
	}
}

func TestEnsureGeneratedSessionTitleUpdatesDefaultTitle(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	var notified string
	service.SetSessionUpdatedHook(func(sessionID string, _ *domain.Session) {
		notified = sessionID
	})
	service.titleGenerator = func(context.Context, string, *domain.ModelRef) (string, error) {
		return "\"Redis 缓存方案\"\nextra text", nil
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{
		Type:  domain.SessionTypeCoding,
		Title: "帮我写一个 Redis 缓存方案",
		Model: &domain.ModelRef{ProviderID: "openai", ModelID: "gpt-5.5"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AppendEvent(ctx, domain.AppendEventRequest{SessionID: session.ID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser, Content: "帮我写一个 Redis 缓存方案"}); err != nil {
		t.Fatal(err)
	}
	service.ensureGeneratedSessionTitle(ctx, session.ID, &domain.ModelRef{ProviderID: "openai", ModelID: "gpt-5.5"})
	updated, err := service.GetRuntimeSession(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Redis 缓存方案" {
		t.Fatalf("title = %q", updated.Title)
	}
	if notified != session.ID {
		t.Fatalf("notified session = %q", notified)
	}
}

func TestCleanGeneratedSessionTitle(t *testing.T) {
	got := cleanGeneratedSessionTitle("<think>reasoning</think>\n`Fix React TS2345 error`\nignored")
	if got != "Fix React TS2345 error" {
		t.Fatalf("title = %q", got)
	}
}

func TestResolveTitleModelsPrefersSmallConnectedProviderModel(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{
		ProviderID: "openai",
		Type:       "openai",
		ModelID:    "gpt-5.5",
		Method:     "env",
	}); err != nil {
		t.Fatal(err)
	}
	models := service.resolveTitleModels(ctx, &domain.ModelRef{ProviderID: "openai", ModelID: "gpt-5.5"})
	if len(models) != 2 {
		t.Fatalf("models = %#v", models)
	}
	if models[0].ModelID != "gpt-5.4-mini" || models[1].ModelID != "gpt-5.5" {
		t.Fatalf("models = %#v", models)
	}
}

func TestUpdateModelPreferencesPersistsDefaultModelAndReasoningEffort(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{
		ProviderID: "openai",
		Type:       "openai",
		ModelID:    "gpt-5.5",
		Method:     "env",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateModelPreferences(ctx, domain.ModelPreferencesInput{
		Model:           &domain.ModelRef{ProviderID: "openai", ModelID: "gpt-5.4-mini"},
		AuxiliaryModel:  &domain.ModelRef{ProviderID: "openai", ModelID: "gpt-5-mini"},
		ReasoningEffort: "high",
		ServiceTier:     "priority",
	}); err != nil {
		t.Fatal(err)
	}
	cfg, err := service.AppConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultModel == nil || cfg.DefaultModel.ModelID != "gpt-5.4-mini" {
		t.Fatalf("default model = %#v, want gpt-5.4-mini", cfg.DefaultModel)
	}
	if cfg.Provider == nil || cfg.Provider.Model != "gpt-5.4-mini" {
		t.Fatalf("provider = %#v, want model gpt-5.4-mini", cfg.Provider)
	}
	if cfg.AuxiliaryModel == nil || cfg.AuxiliaryModel.ModelID != "gpt-5-mini" {
		t.Fatalf("auxiliary model = %#v, want gpt-5-mini", cfg.AuxiliaryModel)
	}
	if cfg.ReasoningEffort != "high" {
		t.Fatalf("reasoningEffort = %q, want high", cfg.ReasoningEffort)
	}
	if cfg.ServiceTier != "priority" {
		t.Fatalf("serviceTier = %q, want priority", cfg.ServiceTier)
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

func contains(value string, needle string) bool {
	return strings.Contains(value, needle)
}

func writeSessionProjectFile(t *testing.T, root string, rel string, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
