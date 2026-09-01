package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aivo/core/domain"
)

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

func TestSubmitSessionMessageRejectsUnreadableAttachmentBeforePersistence(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{
		SessionID: session.ID,
		Text:      "inspect this",
		Attachments: []domain.MessageAttachment{{
			Name: "broken.pdf", MIMEType: "application/pdf", Kind: "file", Data: "not-base64",
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "invalid base64 attachment data") {
		t.Fatalf("error = %v, want invalid attachment refusal", err)
	}
	events, listErr := service.ListEvents(ctx, session.ID, true, 20)
	if listErr != nil {
		t.Fatal(listErr)
	}
	if len(events) != 0 {
		t.Fatalf("events = %#v, want no persisted event", events)
	}

	_, err = service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{
		SessionID: session.ID,
		Text:      "inspect this archive",
		Attachments: []domain.MessageAttachment{{
			Name: "archive.zip", MIMEType: "application/octet-stream", Kind: "file", Data: "UEsDBA==",
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported binary attachment MIME type") {
		t.Fatalf("error = %v, want unsupported MIME refusal", err)
	}
	events, listErr = service.ListEvents(ctx, session.ID, true, 20)
	if listErr != nil {
		t.Fatal(listErr)
	}
	if len(events) != 0 {
		t.Fatalf("events = %#v, want unsupported attachment rejected before persistence", events)
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
		Name:      ExecCommandToolName,
		Arguments: map[string]any{"cmd": "pwd"},
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
	if replayed.Name != ExecCommandToolName || replayed.Status != domain.ToolCallStatusSuccess {
		t.Fatalf("replayed call = %#v, want successful exec_command call", replayed)
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

func TestReplaySessionCoreToolCallIgnoresLegacyGlobalPreference(t *testing.T) {
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
		ID:        "call_globally_disabled",
		SessionID: session.ID,
		Name:      ExecCommandToolName,
		Arguments: map[string]any{"cmd": "touch replay-must-not-exist"},
		Status:    domain.ToolCallStatusFailed,
		Error:     "previous run failed",
	})
	if err != nil {
		t.Fatal(err)
	}
	preferences, ok := service.store.(globalToolPreferenceStore)
	if !ok {
		t.Fatal("global tool preference store unavailable")
	}
	if err := preferences.SetGlobalToolEnabled(ctx, ExecCommandToolName, false); err != nil {
		t.Fatal(err)
	}

	replayed, err := service.ReplaySessionToolCall(ctx, domain.ReplaySessionToolCallRequest{SessionID: session.ID, ToolCallID: original.ID})
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Status != domain.ToolCallStatusSuccess {
		t.Fatalf("replayed call = %#v, want explicit replay to remain available", replayed)
	}
	if _, err := os.Stat(filepath.Join(root, "replay-must-not-exist")); err != nil {
		t.Fatalf("explicit replay did not execute the globally hidden tool: %v", err)
	}
}
