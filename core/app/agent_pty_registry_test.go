package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"aivo/core/domain"
)

func TestAgentPTYInteractivePromptAndMultiStepInput(t *testing.T) {
	registry := NewAgentPTYRegistry()
	defer registry.Shutdown()
	root := t.TempDir()
	events := make(chan ShellOutputEvent, 16)
	request := SandboxRequest{
		WorkspaceRoot: root, CWD: root, SessionID: "session-1", TurnID: "turn-1", ToolCallID: "call-1",
		Command:      `printf '%s\n' '{"v":1,"type":"input_request","id":"continue"}' >&3; printf 'Continue?'; read answer; printf '\nanswer=%s\n' "$answer"`,
		EnvAllowlist: defaultEnvAllowlist(), OutputSink: func(event ShellOutputEvent) { events <- event },
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	first, err := registry.Start(ctx, request, 24, 80, 3*time.Second, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != AgentPTYStatusWaitingInput || first.YieldReason != "input_request" || first.InputRequest == nil {
		t.Fatalf("first result = %#v, want waiting input prompt boundary", first)
	}
	if !strings.Contains(first.Output, "Continue?") {
		t.Fatalf("first output = %q", first.Output)
	}
	_, err = registry.Write(ctx, AgentPTYWriteInput{
		WorkspaceRoot: root, SessionID: "session-1", ProcessRef: first.ProcessRef,
		Chars: "no\r", Cursor: first.Cursor,
	})
	var decisionErr *AgentPTYDecisionRequiredError
	if !errors.As(err, &decisionErr) || decisionErr.RequestID != first.InputRequest.ID {
		t.Fatalf("unleased input error = %#v, want decision required", err)
	}
	if _, err = registry.ResolveInput(AgentPTYResolveInput{
		WorkspaceRoot: root, SessionID: "session-1", ProcessRef: first.ProcessRef,
		RequestID: first.InputRequest.ID, Mode: AgentPTYInputAgentOnce,
	}); err != nil {
		t.Fatal(err)
	}
	second, err := registry.Write(ctx, AgentPTYWriteInput{
		WorkspaceRoot: root, SessionID: "session-1", ProcessRef: first.ProcessRef,
		Chars: "yes\r", Cursor: first.Cursor, YieldTime: 3 * time.Second, MaxOutput: 4096,
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.Status != AgentPTYStatusExited || second.YieldReason != "exited" {
		t.Fatalf("second result = %#v, want exited", second)
	}
	if !strings.Contains(second.Output, "answer=yes") {
		t.Fatalf("second output = %q", second.Output)
	}
	if second.Cursor <= first.Cursor {
		t.Fatalf("cursor did not advance: first=%d second=%d", first.Cursor, second.Cursor)
	}

	seenCursor := false
	seenExited := false
	for len(events) > 0 {
		event := <-events
		if event.ProcessRef == first.ProcessRef && event.Cursor > 0 {
			seenCursor = true
		}
		if event.ProcessRef == first.ProcessRef && event.Status == AgentPTYStatusExited {
			seenExited = true
		}
	}
	if !seenCursor || !seenExited {
		t.Fatalf("events missing cursor/exited state: cursor=%v exited=%v", seenCursor, seenExited)
	}
}

func TestAgentPTYUserOnceAndAgentAlwaysLeases(t *testing.T) {
	registry := NewAgentPTYRegistry()
	defer registry.Shutdown()
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	first, err := registry.Start(ctx, SandboxRequest{
		WorkspaceRoot: root, CWD: root, SessionID: "owner",
		Command:      `printf '%s\n' '{"v":1,"type":"input_request","id":"one"}' >&3; printf 'One?'; read one; printf '%s\n' '{"v":1,"type":"input_request","id":"two"}' >&3; printf 'Two?'; read two; printf '\n%s/%s\n' "$one" "$two"`,
		EnvAllowlist: defaultEnvAllowlist(),
	}, 24, 80, 3*time.Second, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if first.InputRequest == nil {
		t.Fatal("first prompt has no input request")
	}
	if _, err = registry.ResolveInput(AgentPTYResolveInput{WorkspaceRoot: root, SessionID: "owner", ProcessRef: first.ProcessRef, RequestID: first.InputRequest.ID, Mode: AgentPTYInputUserOnce}); err != nil {
		t.Fatal(err)
	}
	second, err := registry.WriteUser(ctx, AgentPTYWriteInput{WorkspaceRoot: root, SessionID: "owner", ProcessRef: first.ProcessRef, Chars: "user\r", Cursor: first.Cursor, YieldTime: 3 * time.Second, MaxOutput: 4096})
	if err != nil {
		t.Fatal(err)
	}
	if second.InputMode != AgentPTYInputAsk || second.InputRequest == nil || second.Status != AgentPTYStatusWaitingInput {
		t.Fatalf("second = %#v", second)
	}
	if _, err = registry.ResolveInput(AgentPTYResolveInput{WorkspaceRoot: root, SessionID: "owner", ProcessRef: first.ProcessRef, RequestID: second.InputRequest.ID, Mode: AgentPTYInputAgentAlways}); err != nil {
		t.Fatal(err)
	}
	last, err := registry.Write(ctx, AgentPTYWriteInput{WorkspaceRoot: root, SessionID: "owner", ProcessRef: first.ProcessRef, Chars: "agent\r", Cursor: second.Cursor, YieldTime: 3 * time.Second, MaxOutput: 4096})
	if err != nil {
		t.Fatal(err)
	}
	if last.Status != AgentPTYStatusExited || !strings.Contains(last.Output, "user/agent") {
		t.Fatalf("last = %#v", last)
	}
}

func TestAgentPTYWriteDoesNotClearNextInputRequest(t *testing.T) {
	resolved := &AgentPTYInputRequest{ID: "one"}
	session := &agentPTYSession{inputRequest: &AgentPTYInputRequest{ID: "two"}}

	session.clearInputRequestLocked(resolved)

	if session.inputRequest == nil || session.inputRequest.ID != "two" {
		t.Fatalf("next input request = %#v, want request two preserved", session.inputRequest)
	}
	session.clearInputRequestLocked(session.inputRequest)
	if session.inputRequest != nil {
		t.Fatalf("matching input request was not cleared: %#v", session.inputRequest)
	}
}

func TestAgentPTYAttachmentReplaysAndStreamsWithoutOwningLifecycle(t *testing.T) {
	registry := NewAgentPTYRegistry()
	defer registry.Shutdown()
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	first, err := registry.Start(ctx, SandboxRequest{WorkspaceRoot: root, CWD: root, SessionID: "owner", Command: `printf ready; sleep .1; printf done`, EnvAllowlist: defaultEnvAllowlist()}, 24, 80, 50*time.Millisecond, 4096)
	if err != nil {
		t.Fatal(err)
	}
	attachment, err := registry.Attach(root, "owner", first.ProcessRef, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(attachment.Snapshot.Output, "ready") {
		t.Fatalf("replay = %q", attachment.Snapshot.Output)
	}
	attachment.Detach()
	session, _ := registry.owned(root, "owner", first.ProcessRef)
	select {
	case <-session.done:
	case <-ctx.Done():
		t.Fatal("detaching observer terminated or stalled process")
	}
}

func TestAgentPTYDetectsNextPromptAfterDirectUserWrite(t *testing.T) {
	registry := NewAgentPTYRegistry()
	defer registry.Shutdown()
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	first, err := registry.Start(ctx, SandboxRequest{WorkspaceRoot: root, CWD: root, SessionID: "owner", Command: `printf '%s\n' '{"v":1,"type":"input_request","id":"one"}' >&3; printf 'One?'; read one; printf '%s\n' '{"v":1,"type":"input_request","id":"two"}' >&3; printf 'Two?'; read two`, EnvAllowlist: defaultEnvAllowlist()}, 24, 80, 3*time.Second, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if first.InputRequest == nil {
		t.Fatal("first request missing")
	}
	if _, err = registry.ResolveInput(AgentPTYResolveInput{WorkspaceRoot: root, SessionID: "owner", ProcessRef: first.ProcessRef, RequestID: first.InputRequest.ID, Mode: AgentPTYInputUserOnce}); err != nil {
		t.Fatal(err)
	}
	attachment, err := registry.Attach(root, "owner", first.ProcessRef, first.Cursor)
	if err != nil {
		t.Fatal(err)
	}
	defer attachment.Detach()
	if _, err = registry.WriteUserNow(AgentPTYWriteInput{WorkspaceRoot: root, SessionID: "owner", ProcessRef: first.ProcessRef, Chars: "answer\r", Cursor: first.Cursor}); err != nil {
		t.Fatal(err)
	}
	for {
		select {
		case event := <-attachment.Events():
			if event.Type == "input_request" && event.Snapshot.InputRequest != nil && event.Snapshot.InputRequest.ID != first.InputRequest.ID {
				return
			}
		case <-ctx.Done():
			t.Fatal("next prompt did not create an independent input request")
		}
	}
}

func TestAgentPTYManualOwnershipChoiceWorksWithoutDetectedPrompt(t *testing.T) {
	for _, mode := range []string{AgentPTYInputUserOnce, AgentPTYInputAgentOnce, AgentPTYInputAgentAlways} {
		t.Run(mode, func(t *testing.T) {
			registry := NewAgentPTYRegistry()
			defer registry.Shutdown()
			root := t.TempDir()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			first, err := registry.Start(ctx, SandboxRequest{
				WorkspaceRoot: root, CWD: root, SessionID: "owner",
				Command: `read answer; printf '%s' "$answer"`, EnvAllowlist: defaultEnvAllowlist(),
			}, 24, 80, 100*time.Millisecond, 4096)
			if err != nil {
				t.Fatal(err)
			}
			if first.Status != AgentPTYStatusRunning || first.InputRequest != nil {
				t.Fatalf("first = %#v, want an undetected running input wait", first)
			}
			chosen, err := registry.ResolveInput(AgentPTYResolveInput{
				WorkspaceRoot: root, SessionID: "owner", ProcessRef: first.ProcessRef, Mode: mode,
			})
			if err != nil {
				t.Fatal(err)
			}
			if chosen.InputMode != mode || chosen.Status != AgentPTYStatusRunning {
				t.Fatalf("chosen = %#v", chosen)
			}
			write := registry.Write
			if mode == AgentPTYInputUserOnce {
				write = registry.WriteUser
			}
			last, err := write(ctx, AgentPTYWriteInput{
				WorkspaceRoot: root, SessionID: "owner", ProcessRef: first.ProcessRef,
				Chars: "manual\r", Cursor: first.Cursor, YieldTime: 3 * time.Second, MaxOutput: 4096,
			})
			if err != nil || last.Status != AgentPTYStatusExited || !strings.Contains(last.Output, "manual") {
				t.Fatalf("last = %#v err=%v", last, err)
			}
		})
	}
}

func TestAgentPTYPromptTextDoesNotCreateInputRequest(t *testing.T) {
	registry := NewAgentPTYRegistry()
	defer registry.Shutdown()
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := registry.Start(ctx, SandboxRequest{
		WorkspaceRoot: root, CWD: root, SessionID: "owner",
		Command: `printf 'Continue?'; sleep 1`, EnvAllowlist: defaultEnvAllowlist(),
	}, 24, 80, 400*time.Millisecond, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != AgentPTYStatusRunning || result.InputRequest != nil || result.YieldReason == "input_request" {
		t.Fatalf("result = %#v, prompt text must remain ordinary output", result)
	}
}

func TestAgentPTYIdleEmitsAdvisoryAttentionOnly(t *testing.T) {
	registry := NewAgentPTYRegistry()
	defer registry.Shutdown()
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	first, err := registry.Start(ctx, SandboxRequest{
		WorkspaceRoot: root, CWD: root, SessionID: "owner",
		Command: `printf ready; sleep 3`, EnvAllowlist: defaultEnvAllowlist(),
	}, 24, 80, 100*time.Millisecond, 4096)
	if err != nil {
		t.Fatal(err)
	}
	attachment, err := registry.Attach(root, "owner", first.ProcessRef, first.Cursor)
	if err != nil {
		t.Fatal(err)
	}
	defer attachment.Detach()
	for {
		select {
		case event := <-attachment.Events():
			if event.Type == "attention" {
				if event.Snapshot.InputRequest != nil || event.Snapshot.Status != AgentPTYStatusRunning || event.Snapshot.Attention == AgentPTYAttentionNone {
					t.Fatalf("attention = %#v", event.Snapshot)
				}
				return
			}
		case <-ctx.Done():
			t.Fatal("advisory attention was not emitted")
		}
	}
}

func TestAgentPTYListsMultipleIndependentSessionTerminals(t *testing.T) {
	registry := NewAgentPTYRegistry()
	defer registry.Shutdown()
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, command := range []string{`printf one; sleep 2`, `printf two; sleep 2`} {
		if _, err := registry.Start(ctx, SandboxRequest{WorkspaceRoot: root, CWD: root, SessionID: "owner", Command: command, EnvAllowlist: defaultEnvAllowlist()}, 24, 80, 100*time.Millisecond, 4096); err != nil {
			t.Fatal(err)
		}
	}
	terminals, err := registry.List(root, "owner")
	if err != nil {
		t.Fatal(err)
	}
	if len(terminals) != 2 || terminals[0].ProcessRef == terminals[1].ProcessRef {
		t.Fatalf("terminals = %#v", terminals)
	}
}

func TestAgentPTYRejectsStaleLeaseVersion(t *testing.T) {
	registry := NewAgentPTYRegistry()
	defer registry.Shutdown()
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	first, err := registry.Start(ctx, SandboxRequest{WorkspaceRoot: root, CWD: root, SessionID: "owner", Command: `read value`, EnvAllowlist: defaultEnvAllowlist()}, 24, 80, 100*time.Millisecond, 4096)
	if err != nil {
		t.Fatal(err)
	}
	chosen, err := registry.ResolveInput(AgentPTYResolveInput{WorkspaceRoot: root, SessionID: "owner", ProcessRef: first.ProcessRef, Mode: AgentPTYInputUserOnce})
	if err != nil {
		t.Fatal(err)
	}
	_, err = registry.WriteUserNow(AgentPTYWriteInput{WorkspaceRoot: root, SessionID: "owner", ProcessRef: first.ProcessRef, Chars: "stale\r", LeaseVersion: chosen.LeaseVersion + 1})
	if err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("stale write error = %v", err)
	}
}

func TestAgentPTYGlobalReplayBudgetUsesLRUTruncation(t *testing.T) {
	registry := NewAgentPTYRegistry()
	registry.globalBufferCap = 512
	defer registry.Shutdown()
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, char := range []string{"a", "b"} {
		command := fmt.Sprintf(`printf '%%0400d' 0 | tr 0 %s`, char)
		result, err := registry.Start(ctx, SandboxRequest{WorkspaceRoot: root, CWD: root, SessionID: "owner", Command: command, EnvAllowlist: defaultEnvAllowlist()}, 24, 80, 2*time.Second, 4096)
		if err != nil || result.Status != AgentPTYStatusExited {
			t.Fatalf("result = %#v err=%v", result, err)
		}
	}
	registry.enforceGlobalBufferCap()
	registry.mu.Lock()
	sessions := make([]*agentPTYSession, 0, len(registry.sessions))
	for _, session := range registry.sessions {
		sessions = append(sessions, session)
	}
	registry.mu.Unlock()
	total := 0
	truncated := false
	for _, session := range sessions {
		session.mu.Lock()
		total += len(session.buffer)
		truncated = truncated || session.baseCursor > 0
		session.mu.Unlock()
	}
	if total > registry.globalBufferCap || !truncated {
		t.Fatalf("buffer total=%d truncated=%v", total, truncated)
	}
}

func TestAgentPTYConcurrentInputDecisionsHaveOneWinner(t *testing.T) {
	registry := NewAgentPTYRegistry()
	defer registry.Shutdown()
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	first, err := registry.Start(ctx, SandboxRequest{WorkspaceRoot: root, CWD: root, SessionID: "owner", Command: `printf '%s\n' '{"v":1,"type":"input_request","id":"choose"}' >&3; printf 'Choose?'; read answer`, EnvAllowlist: defaultEnvAllowlist()}, 24, 80, 3*time.Second, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if first.InputRequest == nil {
		t.Fatal("input request missing")
	}
	var wait sync.WaitGroup
	wait.Add(2)
	results := make(chan error, 2)
	for _, mode := range []string{AgentPTYInputAgentOnce, AgentPTYInputUserOnce} {
		go func(mode string) {
			defer wait.Done()
			_, resolveErr := registry.ResolveInput(AgentPTYResolveInput{WorkspaceRoot: root, SessionID: "owner", ProcessRef: first.ProcessRef, RequestID: first.InputRequest.ID, Mode: mode})
			results <- resolveErr
		}(mode)
	}
	wait.Wait()
	close(results)
	succeeded := 0
	for resultErr := range results {
		if resultErr == nil {
			succeeded++
		}
	}
	if succeeded != 1 {
		t.Fatalf("successful decisions = %d, want 1", succeeded)
	}
}

func TestAgentPTYUserCanTakeBackAgentAlwaysLease(t *testing.T) {
	registry := NewAgentPTYRegistry()
	defer registry.Shutdown()
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	first, err := registry.Start(ctx, SandboxRequest{WorkspaceRoot: root, CWD: root, SessionID: "owner", Command: `printf '%s\n' '{"v":1,"type":"input_request","id":"choose"}' >&3; printf 'Choose?'; read answer; printf '%s' "$answer"`, EnvAllowlist: defaultEnvAllowlist()}, 24, 80, 3*time.Second, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = registry.ResolveInput(AgentPTYResolveInput{WorkspaceRoot: root, SessionID: "owner", ProcessRef: first.ProcessRef, RequestID: first.InputRequest.ID, Mode: AgentPTYInputAgentAlways}); err != nil {
		t.Fatal(err)
	}
	taken, err := registry.ResolveInput(AgentPTYResolveInput{WorkspaceRoot: root, SessionID: "owner", ProcessRef: first.ProcessRef, Mode: AgentPTYInputUserOnce})
	if err != nil {
		t.Fatal(err)
	}
	if taken.InputMode != AgentPTYInputUserOnce {
		t.Fatalf("taken = %#v", taken)
	}
	if _, err = registry.Write(ctx, AgentPTYWriteInput{WorkspaceRoot: root, SessionID: "owner", ProcessRef: first.ProcessRef, Chars: "agent\r", Cursor: first.Cursor}); err == nil {
		t.Fatal("agent write succeeded after user takeover")
	}
	last, err := registry.WriteUser(ctx, AgentPTYWriteInput{WorkspaceRoot: root, SessionID: "owner", ProcessRef: first.ProcessRef, Chars: "user\r", Cursor: first.Cursor, YieldTime: 3 * time.Second, MaxOutput: 4096})
	if err != nil || last.Status != AgentPTYStatusExited || !strings.Contains(last.Output, "user") {
		t.Fatalf("last = %#v err=%v", last, err)
	}
}

func TestAgentPTYCursorReplayReportsTruncation(t *testing.T) {
	registry := NewAgentPTYRegistry()
	defer registry.Shutdown()
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	first, err := registry.Start(ctx, SandboxRequest{
		WorkspaceRoot: root, CWD: root, SessionID: "session-1",
		Command: `yes x | head -c 400000`, EnvAllowlist: defaultEnvAllowlist(),
	}, 24, 80, 3*time.Second, 1024)
	if err != nil {
		t.Fatal(err)
	}
	session, err := registry.owned(root, "session-1", first.ProcessRef)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-session.done:
	case <-ctx.Done():
		t.Fatal("process did not exit")
	}
	replayed, err := registry.Write(ctx, AgentPTYWriteInput{
		WorkspaceRoot: root, SessionID: "session-1", ProcessRef: first.ProcessRef,
		Cursor: 0, YieldTime: time.Second, MaxOutput: 2048,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.OutputTruncated || replayed.BaseCursor == 0 {
		t.Fatalf("replay = %#v, want truncated replay after nonzero base cursor", replayed)
	}
	if replayed.Cursor <= replayed.BaseCursor {
		t.Fatalf("replay cursor = %d, base = %d", replayed.Cursor, replayed.BaseCursor)
	}
}

func TestAgentPTYRejectsOtherSessionAndTerminatesOwnedProcess(t *testing.T) {
	registry := NewAgentPTYRegistry()
	defer registry.Shutdown()
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	first, err := registry.Start(ctx, SandboxRequest{
		WorkspaceRoot: root, CWD: root, SessionID: "owner", Command: `printf ready; sleep 30`,
		EnvAllowlist: defaultEnvAllowlist(),
	}, 24, 80, 2*time.Second, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Write(ctx, AgentPTYWriteInput{
		WorkspaceRoot: root, SessionID: "other", ProcessRef: first.ProcessRef, Cursor: first.Cursor,
	}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("cross-session write error = %v", err)
	}
	terminated, err := registry.Write(ctx, AgentPTYWriteInput{
		WorkspaceRoot: root, SessionID: "owner", ProcessRef: first.ProcessRef,
		Cursor: first.Cursor, YieldTime: 2 * time.Second, Terminate: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if terminated.Status != AgentPTYStatusExited {
		t.Fatalf("terminated result = %#v", terminated)
	}
}

func TestAgentPTYCleanupSessionStopsProcess(t *testing.T) {
	registry := NewAgentPTYRegistry()
	defer registry.Shutdown()
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	first, err := registry.Start(ctx, SandboxRequest{
		WorkspaceRoot: root, CWD: root, SessionID: "owner", Command: `printf ready; sleep 30`,
		EnvAllowlist: defaultEnvAllowlist(),
	}, 24, 80, 2*time.Second, 4096)
	if err != nil {
		t.Fatal(err)
	}
	session, err := registry.owned(root, "owner", first.ProcessRef)
	if err != nil {
		t.Fatal(err)
	}
	registry.CleanupSession("owner")
	select {
	case <-session.done:
	case <-ctx.Done():
		t.Fatal("cleanup did not stop process")
	}
	if err := registry.ValidateOwner(root, "owner", first.ProcessRef); err == nil {
		t.Fatal("cleaned process should no longer be addressable")
	}
}

func TestAgentPTYCancelledStartKeepsProcessAlive(t *testing.T) {
	registry := NewAgentPTYRegistry()
	defer registry.Shutdown()
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)
	result, err := registry.Start(ctx, SandboxRequest{
		WorkspaceRoot: root, CWD: root, SessionID: "owner", Command: `sleep 30`,
		EnvAllowlist: defaultEnvAllowlist(),
	}, 24, 80, 10*time.Second, 4096)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("start error = %v, want context canceled", err)
	}
	if result.YieldReason != "cancelled" {
		t.Fatalf("result = %#v, want cancelled yield", result)
	}
	registry.mu.Lock()
	session := registry.sessions[result.ProcessRef]
	registry.mu.Unlock()
	if session == nil {
		t.Fatal("cancelled process session was not retained for review")
	}
	select {
	case <-session.done:
		t.Fatal("cancelling the tool wait terminated the PTY process")
	case <-time.After(250 * time.Millisecond):
	}
	if err := registry.ValidateOwner(root, "owner", result.ProcessRef); err != nil {
		t.Fatalf("cancelled wait lost process ownership: %v", err)
	}
	terminated, terminateErr := registry.Write(context.Background(), AgentPTYWriteInput{WorkspaceRoot: root, SessionID: "owner", ProcessRef: result.ProcessRef, Cursor: result.Cursor, Terminate: true, YieldTime: time.Second})
	if terminateErr != nil || terminated.Status != AgentPTYStatusExited {
		t.Fatalf("explicit termination = %#v err=%v", terminated, terminateErr)
	}
}

func TestAgentPTYCancelledWriteWaitKeepsProcessAlive(t *testing.T) {
	registry := NewAgentPTYRegistry()
	defer registry.Shutdown()
	root := t.TempDir()
	startCtx, startCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer startCancel()
	first, err := registry.Start(startCtx, SandboxRequest{WorkspaceRoot: root, CWD: root, SessionID: "owner", Command: `printf ready; sleep 30`, EnvAllowlist: defaultEnvAllowlist()}, 24, 80, 100*time.Millisecond, 4096)
	if err != nil {
		t.Fatal(err)
	}
	waitCtx, cancelWait := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancelWait)
	result, err := registry.Write(waitCtx, AgentPTYWriteInput{WorkspaceRoot: root, SessionID: "owner", ProcessRef: first.ProcessRef, Cursor: first.Cursor, YieldTime: 10 * time.Second})
	if !errors.Is(err, context.Canceled) || result.YieldReason != "cancelled" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if err := registry.ValidateOwner(root, "owner", first.ProcessRef); err != nil {
		t.Fatalf("cancelled poll lost process: %v", err)
	}
	session, _ := registry.owned(root, "owner", first.ProcessRef)
	select {
	case <-session.done:
		t.Fatal("cancelled poll terminated process")
	case <-time.After(250 * time.Millisecond):
	}
}

func TestSanitizeInteractivePermissionArgumentsRemovesInput(t *testing.T) {
	arguments := map[string]any{"process_ref": "agent-pty:1", "chars": "super-secret\r"}
	sanitizePermissionArguments(WriteStdinToolName, arguments)
	if _, ok := arguments["chars"]; ok {
		t.Fatal("permission arguments retained stdin value")
	}
	if arguments["stdinPresent"] != true {
		t.Fatalf("stdinPresent = %#v", arguments["stdinPresent"])
	}
	enterOnly := map[string]any{"process_ref": "agent-pty:1", "chars": "", "press_enter": true}
	sanitizePermissionArguments(WriteStdinToolName, enterOnly)
	if enterOnly["stdinPresent"] != true {
		t.Fatalf("enter-only stdinPresent = %#v", enterOnly["stdinPresent"])
	}
}

func TestCodingRegistryRegistersExecCommandTools(t *testing.T) {
	registry, err := NewCodingToolRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{ExecCommandToolName, WriteStdinToolName} {
		if _, ok := registry.Get(name); !ok {
			t.Fatalf("core command tool %s must be registered", name)
		}
	}
	if _, ok := registry.Get("bash"); ok {
		t.Fatal("bash must not be registered")
	}
}

func TestWriteStdinPressEnterSubmitsLine(t *testing.T) {
	registry := NewAgentPTYRegistry()
	defer registry.Shutdown()
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	first, err := registry.Start(ctx, SandboxRequest{
		WorkspaceRoot: root, CWD: root, SessionID: "owner",
		Command: `printf '>'; read answer; printf '<%s>' "$answer"`, EnvAllowlist: defaultEnvAllowlist(),
	}, 24, 80, time.Second, 4096)
	if err != nil {
		t.Fatal(err)
	}
	tool := NewWriteStdinTool(root, registry)
	result := tool.Execute(ctx, json.RawMessage(fmt.Sprintf(
		`{"process_ref":%q,"chars":"exit","press_enter":true,"cursor":%d,"yield_time_ms":3000}`,
		first.ProcessRef, first.Cursor,
	)), domain.ToolExecutionContext{WorkspaceRoot: root, SessionID: "owner"})
	if !result.OK || result.Structured["status"] != AgentPTYStatusExited || !strings.Contains(result.Content, "<exit>") {
		t.Fatalf("result = %#v", result)
	}
}

func TestInteractiveTerminalDecisionErrorIsTypedForModel(t *testing.T) {
	result := interactiveTerminalError(WriteStdinToolName, &AgentPTYDecisionRequiredError{ProcessRef: "agent-pty:1", RequestID: "input:1", Cursor: 42, InputMode: AgentPTYInputAsk})
	if result.OK || result.ToolError == nil || result.ToolError.Code != "input_decision_required" {
		t.Fatalf("result = %#v", result)
	}
	if result.Structured["requestId"] != "input:1" || strings.Contains(result.ModelContent, "secret") {
		t.Fatalf("structured result = %#v", result.Structured)
	}
}

func TestInteractiveTerminalRuntimeCancellationReturnsLiveProcessReference(t *testing.T) {
	registry := NewAgentPTYRegistry()
	defer registry.Shutdown()
	root := t.TempDir()
	tool := NewExecCommandTool(root, registry, nil)
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	runtime := NewToolRuntime(&Registry{tools: map[string][]registeredTool{ExecCommandToolName: {{tool: tool}}}}, root)
	result := runtime.executeToolAttempt(ctx, domain.ChatToolCall{ID: "call", Name: ExecCommandToolName, Arguments: json.RawMessage(`{"cmd":"printf ready; sleep 30","yield_time_ms":10000}`)}, tool, ExecCommandToolName, time.Hour, domain.ToolExecutionContext{WorkspaceRoot: root, SessionID: "owner"})
	processRef, _ := result.Structured["processRef"].(string)
	if processRef == "" || result.Structured["processAlive"] != true {
		t.Fatalf("result = %#v", result)
	}
	if err := registry.ValidateOwner(root, "owner", processRef); err != nil {
		t.Fatalf("runtime cancellation lost process: %v", err)
	}
}
