package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aivo/core/domain"
)

type fakeSandboxRunner struct {
	requests []SandboxRequest
	result   SandboxResult
	err      error
}

func (r *fakeSandboxRunner) Run(_ context.Context, request SandboxRequest) (SandboxResult, error) {
	r.requests = append(r.requests, request)
	result := r.result
	if result.Command == "" {
		result.Command = request.Command
	}
	if result.CWD == "" {
		result.CWD = request.CWD
	}
	if result.Backend == "" {
		result.Backend = "local"
	}
	if result.NetworkPolicy == "" {
		result.NetworkPolicy = request.NetworkPolicy
	}
	result.Duration = time.Millisecond
	return result, r.err
}

func TestCommandDetectorClassifiesAndBlocksCommands(t *testing.T) {
	root := t.TempDir()
	tests := []struct {
		name       string
		command    string
		category   string
		risk       string
		denyReason bool
	}{
		{name: "git status", command: "git status --short", category: CommandCategoryRead, risk: CommandRiskLow},
		{name: "core tests", command: "npm run test:core", category: CommandCategoryTest, risk: CommandRiskMedium},
		{name: "repository diagnostics", command: "npm run diagnostics", category: CommandCategoryBuild, risk: CommandRiskMedium},
		{name: "network", command: "curl https://example.com", category: CommandCategoryNetwork, risk: CommandRiskHigh},
		{name: "dependency write", command: "go mod tidy", category: CommandCategoryWrite, risk: CommandRiskHigh},
		{name: "rm root", command: "rm -rf /", category: CommandCategoryDangerous, risk: CommandRiskCritical, denyReason: true},
		{name: "external path", command: "ls /tmp", category: CommandCategoryDangerous, risk: CommandRiskCritical, denyReason: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detection := DetectCommand(tt.command, root, root, "bash")
			if detection.Category != tt.category || detection.RiskLevel != tt.risk {
				t.Fatalf("detection = %#v, want category=%s risk=%s", detection, tt.category, tt.risk)
			}
			if (detection.DenyReason != "") != tt.denyReason {
				t.Fatalf("denyReason = %q, want present=%t", detection.DenyReason, tt.denyReason)
			}
			if detection.ApprovalKey == "" {
				t.Fatal("approval key was not generated")
			}
		})
	}
}

func TestCommandPolicyStrictestDecisionAndMetacharacters(t *testing.T) {
	root := t.TempDir()
	detection := DetectCommand("git status --short | cat", root, root, "bash")
	policy := EvaluateCommandPolicy(detection, "bash")
	if policy.Decision != CommandDecisionAsk || policy.RiskLevel != CommandRiskHigh {
		t.Fatalf("policy = %#v, want ask/high because pipe raises risk", policy)
	}

	blocked := DetectCommand("sudo ls", root, root, "bash")
	policy = EvaluateCommandPolicy(blocked, "bash")
	if policy.Decision != CommandDecisionDeny || !policy.Hardline {
		t.Fatalf("policy = %#v, want hardline deny", policy)
	}
}

func TestApprovalKeyChangesWithCommandCWDBackendAndNetwork(t *testing.T) {
	root := t.TempDir()
	a := commandApprovalKey(root, root, "pwd", []string{"pwd"}, "bash", "local", "default", "deny", CommandCategoryRead, CommandRiskLow, []string{"shell.exec.foreground"})
	if a == commandApprovalKey(root, root, "ls", []string{"ls"}, "bash", "local", "default", "deny", CommandCategoryRead, CommandRiskLow, []string{"shell.exec.foreground"}) {
		t.Fatal("approval key did not change with command")
	}
	if a == commandApprovalKey(root, filepath.Join(root, "sub"), "pwd", []string{"pwd"}, "bash", "local", "default", "deny", CommandCategoryRead, CommandRiskLow, []string{"shell.exec.foreground"}) {
		t.Fatal("approval key did not change with cwd")
	}
	if a == commandApprovalKey(root, root, "pwd", []string{"pwd"}, "bash", "docker", "default", "deny", CommandCategoryRead, CommandRiskLow, []string{"shell.exec.foreground"}) {
		t.Fatal("approval key did not change with backend")
	}
	if a == commandApprovalKey(root, root, "pwd", []string{"pwd"}, "bash", "local", "default", "inherit", CommandCategoryRead, CommandRiskLow, []string{"shell.exec.foreground"}) {
		t.Fatal("approval key did not change with network policy")
	}
	if a == commandApprovalKey(root, root, "pwd", []string{"pwd"}, "bash", "local", "default", "deny", CommandCategoryRead, CommandRiskLow, []string{"shell.exec.background"}) {
		t.Fatal("approval key did not change with capability")
	}
}

func TestSanitizedEnvironmentDropsSecretsAndUnsafeCaches(t *testing.T) {
	root := t.TempDir()
	t.Setenv("OPENAI_API_KEY", "secret")
	t.Setenv("GITHUB_TOKEN", "secret")
	t.Setenv("PATH", "/usr/bin")
	t.Setenv("NPM_CONFIG_CACHE", "/tmp/aivo-outside-cache")
	t.Setenv("GOCACHE", filepath.Join(root, ".cache", "go"))

	env := SanitizedEnvironment(root, nil, map[string]string{"CUSTOM_TOKEN": "secret", "LANG": "C"}, nil)
	joined := "\n" + strings.Join(env, "\n") + "\n"
	for _, forbidden := range []string{"OPENAI_API_KEY=", "GITHUB_TOKEN=", "CUSTOM_TOKEN=", "NPM_CONFIG_CACHE=/tmp/aivo-outside-cache"} {
		if strings.Contains(joined, "\n"+forbidden) {
			t.Fatalf("sanitized env leaked %s in %q", forbidden, joined)
		}
	}
	for _, required := range []string{"PATH=/usr/bin", "GOCACHE=" + filepath.Join(root, ".cache", "go"), "AIVO_SANDBOX=local", "CI="} {
		if !strings.Contains(joined, "\n"+required) {
			t.Fatalf("sanitized env missing %s in %q", required, joined)
		}
	}
}

func TestLocalSandboxRunnerCapturesExitTimeoutAndRetainsOutput(t *testing.T) {
	root := t.TempDir()
	runner := NewLocalSandboxRunner()
	var outputEvents []ShellOutputEvent
	result, err := runner.Run(context.Background(), SandboxRequest{
		WorkspaceRoot: root,
		Command:       "printf stdout; printf stderr >&2; exit 7",
		OutputPolicy:  domain.OutputPolicy{MaxChars: 100},
		SessionID:     "s1",
		TurnID:        "t1",
		ToolCallID:    "c1",
		OutputSink: func(event ShellOutputEvent) {
			outputEvents = append(outputEvents, event)
		},
	})
	if err == nil || result.ExitCode != 7 || result.Stdout != "stdout" || result.Stderr != "stderr" {
		t.Fatalf("result=%#v err=%v, want captured non-zero exit", result, err)
	}
	if !hasShellOutputEvent(outputEvents, "stdout", "stdout", "s1", "t1", "c1") ||
		!hasShellOutputEvent(outputEvents, "stderr", "stderr", "s1", "t1", "c1") {
		t.Fatalf("output events = %#v, want stdout and stderr chunks", outputEvents)
	}

	result, err = runner.Run(context.Background(), SandboxRequest{
		WorkspaceRoot: root,
		Command:       "printf 123456789",
		OutputPolicy:  domain.OutputPolicy{MaxChars: 3},
		SessionID:     "s1",
		ToolCallID:    "c2",
	})
	if err != nil || !result.Truncated || result.StdoutRef == "" {
		t.Fatalf("result=%#v err=%v, want retained truncated stdout", result, err)
	}
	if raw, readErr := os.ReadFile(result.StdoutRef); readErr != nil || string(raw) != "123456789" {
		t.Fatalf("retained stdout = %q err=%v", raw, readErr)
	}

	result, err = runner.Run(context.Background(), SandboxRequest{
		WorkspaceRoot: root,
		Command:       "sleep 2",
		Timeout:       20 * time.Millisecond,
		OutputPolicy:  domain.OutputPolicy{MaxChars: 100},
	})
	if err == nil || !result.TimedOut {
		t.Fatalf("result=%#v err=%v, want timeout", result, err)
	}
}

func TestBashToolReusesPersistentShellPerSession(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o700); err != nil {
		t.Fatal(err)
	}
	tool := NewBashTool(root, nil)
	sessionID := "persistent-shell-session"
	defer defaultAgentShellRegistry.CleanupSession(sessionID)

	first := tool.Execute(context.Background(), json.RawMessage(`{"command":"export AIVO_PERSISTED_FLAG=kept; cd sub"}`), domain.ToolExecutionContext{WorkspaceRoot: root, SessionID: sessionID, TurnID: "t1", ToolCallID: "c1"})
	if !first.OK {
		t.Fatalf("first = %#v, want successful command", first)
	}
	second := tool.Execute(context.Background(), json.RawMessage(`{"command":"printf '%s:%s' \"$AIVO_PERSISTED_FLAG\" \"$(basename \"$PWD\")\""}`), domain.ToolExecutionContext{WorkspaceRoot: root, SessionID: sessionID, TurnID: "t1", ToolCallID: "c2"})
	if !second.OK || strings.TrimSpace(second.Structured["stdout"].(string)) != "kept:sub" {
		t.Fatalf("second = %#v, want persisted env and cwd", second)
	}
}

func TestBashToolPersistsAndRestoresLastWorkspaceCWD(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o700); err != nil {
		t.Fatal(err)
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, ProjectPath: root})
	if err != nil {
		t.Fatal(err)
	}
	defer defaultAgentShellRegistry.CleanupSession(session.ID)

	tool := NewBashTool(root, nil)
	tool.SetPersistentCWDHooks(service.loadAgentShellCWD, service.saveAgentShellCWD)
	first := tool.Execute(ctx, json.RawMessage(`{"command":"cd sub"}`), domain.ToolExecutionContext{WorkspaceRoot: root, SessionID: session.ID, TurnID: "t1", ToolCallID: "c1"})
	if !first.OK {
		t.Fatalf("first = %#v, want cd command to succeed", first)
	}
	cc, err := service.GetCodingContext(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cc.CWD != filepath.ToSlash(filepath.Join(realRoot, "sub")) {
		t.Fatalf("persisted cwd = %q, want subdir", cc.CWD)
	}

	defaultAgentShellRegistry.CleanupSession(session.ID)
	restoredTool := NewBashTool(root, nil)
	restoredTool.SetPersistentCWDHooks(service.loadAgentShellCWD, service.saveAgentShellCWD)
	second := restoredTool.Execute(ctx, json.RawMessage(`{"command":"printf '%s' \"$(basename \"$PWD\")\""}`), domain.ToolExecutionContext{WorkspaceRoot: root, SessionID: session.ID, TurnID: "t2", ToolCallID: "c2"})
	if !second.OK || strings.TrimSpace(second.Structured["stdout"].(string)) != "sub" {
		t.Fatalf("second = %#v, want restored cwd only", second)
	}
}

func TestBashRequiresApprovalThenSavedApprovalIsExact(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o700); err != nil {
		t.Fatal(err)
	}
	_, runtime := service.toolsForWorkspace(root)
	call := domain.ChatToolCall{ID: "call_pwd", Name: "bash", Arguments: json.RawMessage(`{"command":"pwd"}`)}
	execCtx := domain.ToolExecutionContext{WorkspaceRoot: root, SessionID: "s1", TurnID: "t1", ToolCallID: call.ID}

	resultCh := make(chan domain.ToolResult, 1)
	go func() {
		resultCh <- runtime.ExecuteWithContext(ctx, call, execCtx)
	}()
	request := waitForPermissionRequest(t, service, "s1")
	if request.Action != permissionActionShell || request.Arguments["approvalKey"] == "" {
		t.Fatalf("request = %#v, want shell approval key", request)
	}
	if _, err := service.ApprovePermissionRequest(ctx, domain.ApprovePermissionRequestInput{RequestID: request.ID, Remember: true}); err != nil {
		t.Fatal(err)
	}
	result := waitForToolResult(t, resultCh)
	if !result.OK || !strings.Contains(result.Content, "Exit code: 0") {
		t.Fatalf("result = %#v, want successful bash", result)
	}

	second := runtime.ExecuteWithContext(ctx, domain.ChatToolCall{ID: "call_pwd_2", Name: "bash", Arguments: json.RawMessage(`{"command":"pwd"}`)}, domain.ToolExecutionContext{WorkspaceRoot: root, SessionID: "s1", TurnID: "t1", ToolCallID: "call_pwd_2"})
	if !second.OK || second.PermissionRequested {
		t.Fatalf("second = %#v, want saved exact approval", second)
	}

	approvalCtx, cancel := context.WithTimeout(ctx, 30*time.Millisecond)
	defer cancel()
	third := runtime.ExecuteWithContext(approvalCtx, domain.ChatToolCall{ID: "call_pwd_3", Name: "bash", Arguments: json.RawMessage(`{"command":"pwd","cwd":"sub"}`)}, domain.ToolExecutionContext{WorkspaceRoot: root, SessionID: "s1", TurnID: "t1", ToolCallID: "call_pwd_3"})
	if third.OK || !third.PermissionRequested {
		t.Fatalf("third = %#v, want different cwd to require approval", third)
	}
}

func TestHardlineCommandDeniedBeforeApproval(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	root := t.TempDir()
	_, runtime := service.toolsForWorkspace(root)
	result := runtime.ExecuteWithContext(ctx, domain.ChatToolCall{ID: "call_rm", Name: "bash", Arguments: json.RawMessage(`{"command":"rm -rf /"}`)}, domain.ToolExecutionContext{WorkspaceRoot: root, SessionID: "s1", TurnID: "t1", ToolCallID: "call_rm"})
	if result.OK || result.PermissionRequested || result.ToolError == nil || result.ToolError.Code != "permission_denied" {
		t.Fatalf("result = %#v, want deterministic denial without approval", result)
	}
	requests, err := service.ListPermissionRequests(ctx, "s1", domain.PermissionRequestStatusPending)
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 0 {
		t.Fatalf("requests = %#v, want no approval request", requests)
	}
}

func TestFullAccessAllowsKnownReadBashButStillAsksForUnknownBash(t *testing.T) {
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
	_, runtime := service.toolsForWorkspace(root)
	readResult := runtime.ExecuteWithContext(ctx, domain.ChatToolCall{ID: "call_pwd", Name: "bash", Arguments: json.RawMessage(`{"command":"pwd"}`)}, domain.ToolExecutionContext{WorkspaceRoot: root, SessionID: session.ID, TurnID: "t1", ToolCallID: "call_pwd"})
	if !readResult.OK || readResult.PermissionRequested {
		t.Fatalf("readResult = %#v, want full-access known read command allowed", readResult)
	}
	approvalCtx, cancel := context.WithTimeout(ctx, 30*time.Millisecond)
	defer cancel()
	unknownResult := runtime.ExecuteWithContext(approvalCtx, domain.ChatToolCall{ID: "call_echo", Name: "bash", Arguments: json.RawMessage(`{"command":"echo hi"}`)}, domain.ToolExecutionContext{WorkspaceRoot: root, SessionID: session.ID, TurnID: "t1", ToolCallID: "call_echo"})
	if unknownResult.OK || !unknownResult.PermissionRequested {
		t.Fatalf("unknownResult = %#v, want arbitrary bash to still ask", unknownResult)
	}
}

func TestRunTestsMappingRejectsUnsupportedFilterAndUsesDeclaredCommand(t *testing.T) {
	if _, err := runTestsCommands(runTestsInput{Target: "core", Kind: "test", Filter: "TestName"}); err == nil {
		t.Fatal("filter should be rejected in initial mapping")
	}
	commands, err := runTestsCommands(runTestsInput{Target: "desktop", Kind: "lint"})
	if err != nil || len(commands) != 1 || commands[0] != "npm run lint" {
		t.Fatalf("commands=%v err=%v, want npm run lint", commands, err)
	}
}

func TestBashBackgroundModeReturnsManagedProcessRef(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	root := t.TempDir()
	outputCh := make(chan ShellOutputEvent, 4)
	service.SetShellOutputHook(func(event ShellOutputEvent) {
		outputCh <- event
	})
	_, runtime := service.toolsForWorkspace(root)
	registered, _, ok := runtime.Registry.GetRegistered("bash")
	if !ok {
		t.Fatal("bash tool was not registered")
	}
	if bashTool, ok := registered.(*BashTool); !ok || bashTool.outputSink == nil {
		t.Fatalf("bash tool = %#v, want output sink", registered)
	}
	call := domain.ChatToolCall{ID: "call_bg", Name: "bash", Arguments: json.RawMessage(`{"command":"printf ready","mode":"background"}`)}
	execCtx := domain.ToolExecutionContext{WorkspaceRoot: root, SessionID: "s-bg", TurnID: "t1", ToolCallID: call.ID}
	resultCh := make(chan domain.ToolResult, 1)
	go func() {
		resultCh <- runtime.ExecuteWithContext(ctx, call, execCtx)
	}()
	request := waitForPermissionRequest(t, service, "s-bg")
	if caps, ok := request.Arguments["capabilities"].([]any); !ok || !containsAnyString(caps, "shell.exec.background") {
		t.Fatalf("request capabilities = %#v, want shell.exec.background", request.Arguments["capabilities"])
	}
	if _, err := service.ApprovePermissionRequest(ctx, domain.ApprovePermissionRequestInput{RequestID: request.ID}); err != nil {
		t.Fatal(err)
	}
	result := waitForToolResult(t, resultCh)
	ref, _ := result.Structured["processRef"].(string)
	if !result.OK || ref == "" {
		t.Fatalf("result = %#v, want processRef", result)
	}
	info, err := service.WaitShellProcess(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(info.Stdout, "ready") {
		t.Fatalf("process info = %#v, want stdout ready", info)
	}
	event := waitForShellOutputEvent(t, outputCh, "stdout", "ready", "s-bg", "t1", call.ID)
	if event.ProcessRef != ref {
		t.Fatalf("event processRef = %q, want %q", event.ProcessRef, ref)
	}
}

func TestBashCapabilityPoliciesForEnvExternalCWDAndSudo(t *testing.T) {
	root := t.TempDir()
	if _, err := prepareShellCommand(root, domain.ToolExecutionContext{WorkspaceRoot: root}, "bash", "pwd", "", 1, "deny", "foreground", "", map[string]string{"OPENAI_API_KEY": "x"}); err == nil {
		t.Fatal("secret env override should be denied")
	}
	if _, err := prepareShellCommand(root, domain.ToolExecutionContext{WorkspaceRoot: root}, "bash", "sudo -S ls", "", 1, "deny", "foreground", "", nil); err == nil {
		t.Fatal("sudo password piping should be denied")
	}
	prepared, err := prepareShellCommand(root, domain.ToolExecutionContext{WorkspaceRoot: root}, "bash", "pwd", filepath.Dir(root), 1, "deny", "foreground", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !containsCapability(prepared.detect.Capabilities, "shell.cwd.external") {
		t.Fatalf("capabilities = %#v, want external cwd", prepared.detect.Capabilities)
	}
	if prepared.policy.Decision != CommandDecisionAsk {
		t.Fatalf("policy = %#v, want ask for external cwd", prepared.policy)
	}
}

func containsAnyString(values []any, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func containsCapability(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func hasShellOutputEvent(events []ShellOutputEvent, stream string, chunk string, sessionID string, turnID string, toolCallID string) bool {
	for _, event := range events {
		if event.Stream == stream &&
			strings.Contains(event.Chunk, chunk) &&
			event.SessionID == sessionID &&
			event.TurnID == turnID &&
			event.ToolCallID == toolCallID &&
			event.Sequence > 0 {
			return true
		}
	}
	return false
}

func waitForShellOutputEvent(t *testing.T, events <-chan ShellOutputEvent, stream string, chunk string, sessionID string, turnID string, toolCallID string) ShellOutputEvent {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case event := <-events:
			if event.Stream == stream &&
				strings.Contains(event.Chunk, chunk) &&
				event.SessionID == sessionID &&
				event.TurnID == turnID &&
				event.ToolCallID == toolCallID &&
				event.Sequence > 0 {
				return event
			}
		case <-deadline:
			t.Fatalf("shell output event timed out for %s %q", stream, chunk)
			return ShellOutputEvent{}
		}
	}
}

func waitForPermissionRequest(t *testing.T, service *Service, sessionID string) domain.PermissionRequest {
	t.Helper()
	for i := 0; i < 40; i++ {
		requests, err := service.ListPermissionRequests(context.Background(), sessionID, domain.PermissionRequestStatusPending)
		if err != nil {
			t.Fatal(err)
		}
		if len(requests) > 0 {
			return requests[0]
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("permission request was not created")
	return domain.PermissionRequest{}
}

func waitForToolResult(t *testing.T, resultCh <-chan domain.ToolResult) domain.ToolResult {
	t.Helper()
	select {
	case result := <-resultCh:
		return result
	case <-time.After(2 * time.Second):
		t.Fatal("tool result timed out")
		return domain.ToolResult{}
	}
}
