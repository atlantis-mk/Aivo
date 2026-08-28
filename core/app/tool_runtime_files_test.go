package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"aivo/core/domain"
)

func TestEditRequiresApprovalThenAppliesAfterSavedRule(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	registry, runtime := service.toolsForWorkspace(root)
	if registry == nil || runtime == nil {
		t.Fatal("tool runtime was not created")
	}
	resolvedCh := make(chan domain.PermissionRequest, 1)
	service.SetPermissionResolvedHook(func(request domain.PermissionRequest) {
		resolvedCh <- request
	})
	call := domain.ChatToolCall{ID: "call_edit", Name: "edit", Arguments: json.RawMessage(`{"path":"README.md","edits":[{"oldText":"old","newText":"new"}]}`)}
	execCtx := domain.ToolExecutionContext{WorkspaceRoot: root, SessionID: "s1", TurnID: "t1"}
	resultCh := make(chan domain.ToolResult, 1)
	go func() {
		resultCh <- runtime.ExecuteWithContext(ctx, call, execCtx)
	}()
	var request domain.PermissionRequest
	for i := 0; i < 40; i++ {
		requests, err := service.ListPermissionRequests(ctx, "s1", domain.PermissionRequestStatusPending)
		if err != nil {
			t.Fatal(err)
		}
		if len(requests) > 0 {
			request = requests[0]
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if request.ID == "" {
		t.Fatal("permission request was not created")
	}
	if content, _ := os.ReadFile(filepath.Join(root, "README.md")); string(content) != "old\n" {
		t.Fatalf("file changed before approval: %q", content)
	}
	if _, err := service.ApprovePermissionRequest(ctx, domain.ApprovePermissionRequestInput{RequestID: request.ID, Remember: true}); err != nil {
		t.Fatal(err)
	}
	select {
	case resolved := <-resolvedCh:
		if resolved.ID != request.ID || resolved.Status != domain.PermissionRequestStatusApproved {
			t.Fatalf("resolved = %#v, want approved %s", resolved, request.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("permission resolved hook was not called")
	}
	var result domain.ToolResult
	select {
	case result = <-resultCh:
	case <-time.After(time.Second):
		t.Fatal("tool execution was not woken after approval")
	}
	if !result.OK {
		t.Fatalf("approved edit failed: %#v", result)
	}
	if len(result.Files) != 1 || result.Files[0].FullPath != filepath.ToSlash(filepath.Join(root, "README.md")) {
		t.Fatalf("files = %#v, want full path for edited file", result.Files)
	}
	if content, _ := os.ReadFile(filepath.Join(root, "README.md")); string(content) != "new\n" {
		t.Fatalf("file content = %q, want new", content)
	}
}

func TestAgentLoopPersistsToolCallWhilePermissionPending(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Tools []any `json:"tools"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		if len(body.Tools) == 0 {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"tools\":[],\"reason\":\"core edit is sufficient\"}"}}]}`))
			return
		}
		requestCount++
		if requestCount == 1 {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"tool_calls":[{"id":"call_edit","type":"function","function":{"name":"edit","arguments":"{\"path\":\"README.md\",\"edits\":[{\"oldText\":\"old\",\"newText\":\"new\"}]}"}}]}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"done"}}]}`))
	}))
	defer server.Close()
	if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{ProviderID: "custom-api", Type: "openai-compatible", BaseURL: server.URL, ModelID: "test-model", APIKey: "test-key", Method: "api-key"}); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, ProjectPath: root})
	if err != nil {
		t.Fatal(err)
	}
	run, err := service.SubmitSessionMessageStreaming(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "edit README"})
	if err != nil {
		t.Fatal(err)
	}

	var request domain.PermissionRequest
	var toolCalls []domain.ToolCall
	for i := 0; i < 40; i++ {
		requests, err := service.ListPermissionRequests(ctx, session.ID, domain.PermissionRequestStatusPending)
		if err != nil {
			t.Fatal(err)
		}
		toolCalls, err = service.ListToolCalls(ctx, session.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(requests) > 0 && len(toolCalls) > 0 {
			request = requests[0]
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if request.ID == "" {
		t.Fatal("permission request was not created")
	}
	if len(toolCalls) != 1 {
		t.Fatalf("toolCalls = %#v, want one pending visible tool call", toolCalls)
	}
	if toolCalls[0].ID != "call_edit" || toolCalls[0].TurnID != run.Turn.ID || toolCalls[0].Status != domain.ToolCallStatusPending {
		t.Fatalf("toolCalls[0] = %#v, want pending approval call_edit for prepared turn", toolCalls[0])
	}
	if pendingID, _ := toolCalls[0].Result["pendingApprovalId"].(string); pendingID != request.ID {
		t.Fatalf("toolCalls[0].Result[pendingApprovalId] = %q, want %q", pendingID, request.ID)
	}
	if request.ToolCallID != toolCalls[0].ID {
		t.Fatalf("request.ToolCallID = %q, want %q", request.ToolCallID, toolCalls[0].ID)
	}
	if _, err := service.ApprovePermissionRequest(ctx, domain.ApprovePermissionRequestInput{RequestID: request.ID}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 40; i++ {
		toolCalls, err = service.ListToolCalls(ctx, session.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(toolCalls) == 1 && toolCalls[0].Status == domain.ToolCallStatusSuccess {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("toolCalls = %#v, want call_edit to complete after approval", toolCalls)
}

func TestWriteRejectsSensitivePathDeterministically(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	root := t.TempDir()
	registry, runtime := service.toolsForWorkspace(root)
	if registry == nil || runtime == nil {
		t.Fatal("tool runtime was not created")
	}
	result := runtime.ExecuteWithContext(context.Background(), domain.ChatToolCall{
		ID: "call_secret", Name: "write", Arguments: json.RawMessage(`{"path":".env","content":"TOKEN=x"}`),
	}, domain.ToolExecutionContext{WorkspaceRoot: root, SessionID: "s1", TurnID: "t1"})
	if result.OK || result.PermissionRequested || result.ToolError == nil || result.ToolError.Code != "permission_denied" {
		t.Fatalf("result = %#v, want deterministic permission denial", result)
	}
}

func TestPermissionModeFullAccessAllowsWorkspaceWrite(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, ProjectPath: root})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetPermissionMode(ctx, domain.PermissionModeInput{SessionID: session.ID, Mode: domain.PermissionModeFullAccess}); err != nil {
		t.Fatal(err)
	}
	_, runtime := service.toolsForWorkspace(root)
	result := runtime.ExecuteWithContext(ctx, domain.ChatToolCall{
		ID: "call_edit", Name: "edit", Arguments: json.RawMessage(`{"path":"README.md","edits":[{"oldText":"old","newText":"new"}]}`),
	}, domain.ToolExecutionContext{WorkspaceRoot: root, SessionID: session.ID, TurnID: "t1"})
	if !result.OK || result.PermissionRequested {
		t.Fatalf("result = %#v, want full access to allow workspace patch", result)
	}
	if content, _ := os.ReadFile(filepath.Join(root, "README.md")); string(content) != "new\n" {
		t.Fatalf("file content = %q, want new", content)
	}
}

func TestPermissionModeRequestApprovalOverridesFullAccess(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, ProjectPath: root})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetPermissionMode(ctx, domain.PermissionModeInput{SessionID: session.ID, Mode: domain.PermissionModeFullAccess}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetPermissionMode(ctx, domain.PermissionModeInput{SessionID: session.ID, Mode: domain.PermissionModeRequestApproval}); err != nil {
		t.Fatal(err)
	}
	_, runtime := service.toolsForWorkspace(root)
	resultCh := make(chan domain.ToolResult, 1)
	go func() {
		resultCh <- runtime.ExecuteWithContext(ctx, domain.ChatToolCall{
			ID: "call_edit", Name: "edit", Arguments: json.RawMessage(`{"path":"README.md","edits":[{"oldText":"old","newText":"new"}]}`),
		}, domain.ToolExecutionContext{WorkspaceRoot: root, SessionID: session.ID, TurnID: "t1"})
	}()
	request := waitForPermissionRequest(t, service, session.ID)
	if request.ToolName != "edit" || request.Status != domain.PermissionRequestStatusPending {
		t.Fatalf("request = %#v, want pending edit approval", request)
	}
	if _, err := service.DenyPermissionRequest(ctx, domain.DenyPermissionRequestInput{RequestID: request.ID, Reason: "test denial"}); err != nil {
		t.Fatal(err)
	}
	result := waitForToolResult(t, resultCh)
	if result.OK || result.PermissionRequested || result.ToolError == nil || result.ToolError.Code != "permission_denied" {
		t.Fatalf("result = %#v, want denied edit after explicit approval request", result)
	}
	if content, _ := os.ReadFile(filepath.Join(root, "README.md")); string(content) != "old\n" {
		t.Fatalf("file changed before approval: %q", content)
	}
}

func TestRememberedPermissionAppliesToNewSessionInWorkspace(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	firstSession, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, ProjectPath: root})
	if err != nil {
		t.Fatal(err)
	}
	_, runtime := service.toolsForWorkspace(root)
	firstResult := make(chan domain.ToolResult, 1)
	go func() {
		firstResult <- runtime.ExecuteWithContext(ctx, domain.ChatToolCall{ID: "first_edit", Name: "edit", Arguments: json.RawMessage(`{"path":"README.md","edits":[{"oldText":"old","newText":"new"}]}`)}, domain.ToolExecutionContext{WorkspaceRoot: root, SessionID: firstSession.ID, TurnID: "t1"})
	}()
	request := waitForPermissionRequest(t, service, firstSession.ID)
	if _, err := service.ApprovePermissionRequest(ctx, domain.ApprovePermissionRequestInput{RequestID: request.ID, Remember: true}); err != nil {
		t.Fatal(err)
	}
	if result := waitForToolResult(t, firstResult); !result.OK {
		t.Fatalf("first edit result = %#v, want success", result)
	}

	secondSession, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, ProjectPath: root})
	if err != nil {
		t.Fatal(err)
	}
	second := runtime.ExecuteWithContext(ctx, domain.ChatToolCall{ID: "second_edit", Name: "edit", Arguments: json.RawMessage(`{"path":"README.md","edits":[{"oldText":"new","newText":"newer"}]}`)}, domain.ToolExecutionContext{WorkspaceRoot: root, SessionID: secondSession.ID, TurnID: "t2"})
	if !second.OK || second.PermissionRequested {
		t.Fatalf("second edit result = %#v, want remembered workspace permission", second)
	}
}

func TestWriteFileToolCreatesFile(t *testing.T) {
	root := t.TempDir()
	tool := NewWriteFileTool(root)
	result := tool.Execute(context.Background(), json.RawMessage(`{"path":"docs/summary.md","content":"hello\n"}`), domain.ToolExecutionContext{WorkspaceRoot: root})
	if !result.OK {
		t.Fatalf("write_file failed: %#v", result)
	}
	if len(result.Files) != 1 || result.Files[0].Type != "add" || result.Files[0].Additions != 1 || result.Files[0].Deletions != 0 {
		t.Fatalf("files = %#v, want created file with +1 -0", result.Files)
	}
	if result.Files[0].FullPath != filepath.ToSlash(filepath.Join(root, "docs", "summary.md")) {
		t.Fatalf("fullPath = %q, want absolute file path", result.Files[0].FullPath)
	}
	content, err := os.ReadFile(filepath.Join(root, "docs", "summary.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "hello\n" {
		t.Fatalf("content = %q, want hello", content)
	}
}

func TestWriteFileToolRejectsContentOverLineLimit(t *testing.T) {
	root := t.TempDir()
	tool := NewWriteFileTool(root)
	content := strings.Repeat("line\n", maxDirectWriteLines+1)
	result := tool.Execute(context.Background(), json.RawMessage(`{"path":"docs/long.md","content":`+strconv.Quote(content)+`}`), domain.ToolExecutionContext{WorkspaceRoot: root})
	if result.OK || result.ToolError == nil || !strings.Contains(result.Error, "exceeds 150 lines") {
		t.Fatalf("result = %#v, want line limit failure", result)
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "long.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("long file should not be created, stat err = %v", err)
	}
}

func TestEditFileToolReplacesExactText(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "README.md")
	if err := os.WriteFile(target, []byte("one\ntwo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tool := NewEditFileTool(root)
	result := tool.Execute(context.Background(), json.RawMessage(`{"path":"README.md","oldString":"two","newString":"three"}`), domain.ToolExecutionContext{WorkspaceRoot: root})
	if !result.OK {
		t.Fatalf("edit_file failed: %#v", result)
	}
	if len(result.Files) != 1 || result.Files[0].Type != "edit" || result.Files[0].Additions != 1 || result.Files[0].Deletions != 1 {
		t.Fatalf("files = %#v, want edited file with +1 -1", result.Files)
	}
	if result.Files[0].FullPath != filepath.ToSlash(target) {
		t.Fatalf("fullPath = %q, want absolute file path", result.Files[0].FullPath)
	}
	content, _ := os.ReadFile(target)
	if string(content) != "one\nthree\n" {
		t.Fatalf("content = %q, want edited content", content)
	}
}

func TestReadFileToolReturnsSnapshotMetadata(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "README.md")
	if err := os.WriteFile(target, []byte("one\ntwo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tool := NewReadFileTool(root)
	result := tool.Execute(context.Background(), json.RawMessage(`{"path":"README.md"}`), domain.ToolExecutionContext{WorkspaceRoot: root})
	if !result.OK {
		t.Fatalf("read_file failed: %#v", result)
	}
	snapshot, _ := result.Structured["snapshot"].(fileSnapshot)
	if snapshot.Path != "README.md" || snapshot.SHA256 == "" || snapshot.ID == "" || snapshot.LineRange != "all" {
		t.Fatalf("snapshot = %#v, want path/hash/id/line range", result.Structured["snapshot"])
	}
}

func TestEditFileToolRejectsStaleExpectedHash(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "README.md")
	if err := os.WriteFile(target, []byte("one\ntwo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot, raw, err := readFileSnapshot("README.md", target, "all", false)
	if err != nil || string(raw) == "" {
		t.Fatalf("snapshot failed: %#v %v", snapshot, err)
	}
	if err := os.WriteFile(target, []byte("external\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tool := NewEditFileTool(root)
	result := tool.Execute(context.Background(), json.RawMessage(`{"path":"README.md","oldString":"one","newString":"three","expectedHash":"`+snapshot.SHA256+`"}`), domain.ToolExecutionContext{WorkspaceRoot: root})
	if result.OK || result.ToolError == nil || result.ToolError.Code != "stale_file" || !result.ToolError.Retry {
		t.Fatalf("result = %#v, want retryable stale_file", result)
	}
	content, _ := os.ReadFile(target)
	if string(content) != "external\n" {
		t.Fatalf("content = %q, want external edit preserved", content)
	}
}

func TestEditFileToolRejectsReplacementOverLineLimit(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "README.md")
	if err := os.WriteFile(target, []byte("start\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tool := NewEditFileTool(root)
	replacement := strings.Repeat("line\n", maxDirectEditArgLines+1)
	result := tool.Execute(context.Background(), json.RawMessage(`{"path":"README.md","oldString":"start\n","newString":`+strconv.Quote(replacement)+`}`), domain.ToolExecutionContext{WorkspaceRoot: root})
	if result.OK || result.ToolError == nil || !strings.Contains(result.Error, "exceeds 150 lines") {
		t.Fatalf("result = %#v, want line limit failure", result)
	}
	content, _ := os.ReadFile(target)
	if string(content) != "start\n" {
		t.Fatalf("content = %q, want original content", content)
	}
}

func TestEditFileToolRejectsAmbiguousMatch(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "README.md")
	if err := os.WriteFile(target, []byte("same\nsame\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tool := NewEditFileTool(root)
	result := tool.Execute(context.Background(), json.RawMessage(`{"path":"README.md","oldString":"same","newString":"next"}`), domain.ToolExecutionContext{WorkspaceRoot: root})
	if result.OK || result.ToolError == nil || !strings.Contains(result.Error, "multiple times") {
		t.Fatalf("result = %#v, want ambiguous match failure", result)
	}
}

func TestSafeJoinRejectsTraversalAndSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	if _, err := safeJoin(root, "../outside"); err == nil {
		t.Fatal("safeJoin allowed traversal")
	}
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := safeJoin(root, "link/secret.txt"); err == nil {
		t.Fatal("safeJoin allowed symlink escape")
	}
}
