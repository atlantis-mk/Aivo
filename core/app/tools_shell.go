package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"aivo/core/domain"
)

const shellNamespaceDescription = "Guarded command execution tools. Prefer run_tests for declared test, lint, and build commands. Use bash only as an escape hatch for short non-interactive workspace commands that no safer dedicated tool can represent."

type BashTool struct {
	workspaceRoot string
	runner        SandboxRunner
	processes     *ShellProcessRegistry
	agentShells   *AgentShellRegistry
	loadSavedCWD  func(sessionID string, workspaceRoot string) string
	saveCWD       func(sessionID string, workspaceRoot string, cwd string)
	outputSink    ShellOutputSink
}

func NewBashTool(workspaceRoot string, runner SandboxRunner, outputSink ...ShellOutputSink) *BashTool {
	if runner == nil {
		runner = NewLocalSandboxRunner()
	}
	var sink ShellOutputSink
	if len(outputSink) > 0 {
		sink = outputSink[0]
	}
	return &BashTool{workspaceRoot: workspaceRoot, runner: runner, processes: defaultShellProcessRegistry, agentShells: defaultAgentShellRegistry, outputSink: sink}
}

func (t *BashTool) SetPersistentCWDHooks(load func(sessionID string, workspaceRoot string) string, save func(sessionID string, workspaceRoot string, cwd string)) {
	t.loadSavedCWD = load
	t.saveCWD = save
}

func (t *BashTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:                 "bash",
		Description:          "Escape-hatch shell execution after command policy and permission approval. Prefer run_tests for test/lint/build, read_diagnostics for diagnostics, and format_code for formatter-backed rewrites. Use bash only for short non-interactive workspace commands that no safer dedicated tool can represent. Arguments must be JSON. Foreground mode is bounded; background mode returns a managed processRef. PTY, stdin, env overrides, external cwd, sudo, and network are separate approval dimensions.",
		Namespace:            filesystemNamespace,
		NamespaceDescription: shellNamespaceDescription,
		Capability:           "shell.exec",
		RiskLevel:            "critical",
		Category:             "shell",
		Toolsets:             []string{"shell", "coding"},
		RequiresWorkspace:    true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command":        map[string]any{"type": "string", "description": "Shell command to execute non-interactively inside the workspace. Do not use bash for test, lint, build, diagnostics, or formatting when run_tests, read_diagnostics, or format_code can express the operation."},
				"cwd":            map[string]any{"type": "string", "description": "Optional workspace-relative working directory. Defaults to workspace root."},
				"timeoutSeconds": map[string]any{"type": "integer", "minimum": 1, "maximum": int(maxCommandTimeout.Seconds())},
				"network":        map[string]any{"type": "string", "enum": []string{"deny", "inherit"}, "description": "Requested network policy. Local sandbox can only enforce this by policy checks."},
				"mode":           map[string]any{"type": "string", "enum": []string{"foreground", "background", "pty"}, "description": "Execution mode. Defaults to foreground."},
				"stdin":          map[string]any{"type": "string", "description": "Optional stdin for the command. Requires explicit shell.stdin approval."},
				"env":            map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}, "description": "Optional safe env overrides. Secret-like keys are denied."},
				"justification":  map[string]any{"type": "string", "description": "Short user-visible reason for the permission prompt."},
			},
			"required": []string{"command"},
		},
	}
}

func (t *BashTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	input, err := parseBashArgs(args)
	if err != nil {
		return toolError("bash", err)
	}
	workspaceRoot := toolWorkspaceRoot(t.workspaceRoot, execCtx)
	cwd := input.CWD
	if shouldUsePersistentAgentShell(input, execCtx) && cwd == "" && t.agentShells != nil {
		cwd = t.agentShells.CurrentCWD(execCtx.SessionID, workspaceRoot)
		if cwd == "" && t.loadSavedCWD != nil {
			cwd = t.loadSavedCWD(execCtx.SessionID, workspaceRoot)
		}
	}
	prepared, err := prepareShellCommand(workspaceRoot, execCtx, "bash", input.Command, cwd, input.TimeoutSeconds, input.Network, input.Mode, input.Stdin, input.Env)
	if err != nil {
		return commandToolError("bash", prepared, err)
	}
	prepared.request.OutputSink = t.outputSink
	if prepared.request.Mode == "background" {
		result, runErr := t.processes.Start(ctx, prepared.request)
		return commandToolResult("bash", prepared, result, runErr)
	}
	if prepared.request.Mode == "pty" {
		return commandToolError("bash", prepared, errors.New("model-facing PTY mode is tracked through the dedicated terminal service"))
	}
	if shouldUsePersistentAgentShell(input, execCtx) && t.agentShells != nil {
		result, runErr := t.agentShells.Run(ctx, prepared.request)
		if result.CWD != "" && t.saveCWD != nil {
			t.saveCWD(execCtx.SessionID, workspaceRoot, result.CWD)
		}
		return commandToolResult("bash", prepared, result, runErr)
	}
	result, runErr := t.runner.Run(ctx, prepared.request)
	return commandToolResult("bash", prepared, result, runErr)
}

type RunTestsTool struct {
	workspaceRoot string
	runner        SandboxRunner
	outputSink    ShellOutputSink
}

func NewRunTestsTool(workspaceRoot string, runner SandboxRunner, outputSink ...ShellOutputSink) *RunTestsTool {
	if runner == nil {
		runner = NewLocalSandboxRunner()
	}
	var sink ShellOutputSink
	if len(outputSink) > 0 {
		sink = outputSink[0]
	}
	return &RunTestsTool{workspaceRoot: workspaceRoot, runner: runner, outputSink: sink}
}

func (t *RunTestsTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:                 "run_tests",
		Description:          "Preferred tool for declared test, lint, or build commands in this workspace. Use this instead of bash when target and kind can express the operation. This tool never accepts arbitrary shell text; it maps target and kind to known commands in code and runs through the same command policy, sandbox, timeout, and permission flow as bash.",
		Namespace:            filesystemNamespace,
		NamespaceDescription: shellNamespaceDescription,
		Capability:           "shell.test",
		RiskLevel:            "medium",
		Category:             "shell",
		Toolsets:             []string{"coding"},
		RequiresWorkspace:    true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"target":         map[string]any{"type": "string", "enum": []string{"core", "desktop", "all"}, "description": "Workspace target. Defaults to all."},
				"kind":           map[string]any{"type": "string", "enum": []string{"test", "lint", "build", "auto"}, "description": "Command kind. auto chooses a conservative default."},
				"filter":         map[string]any{"type": "string", "description": "Optional test filter where supported. Unsupported combinations are rejected."},
				"timeoutSeconds": map[string]any{"type": "integer", "minimum": 1, "maximum": int(maxCommandTimeout.Seconds())},
			},
		},
	}
}

func (t *RunTestsTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	input, err := parseRunTestsArgs(args)
	if err != nil {
		return toolError("run_tests", err)
	}
	commands, err := runTestsCommands(input)
	if err != nil {
		return toolError("run_tests", err)
	}
	results := make([]map[string]any, 0, len(commands))
	var content strings.Builder
	ok := true
	var firstErr error
	var retained []string
	for index, command := range commands {
		prepared, err := prepareShellCommand(toolWorkspaceRoot(t.workspaceRoot, execCtx), execCtx, "run_tests", command, "", input.TimeoutSeconds, "deny", "foreground", "", nil)
		if err != nil {
			return commandToolError("run_tests", prepared, err)
		}
		prepared.request.OutputSink = t.outputSink
		result, runErr := t.runner.Run(ctx, prepared.request)
		toolResult := commandToolResult("run_tests", prepared, result, runErr)
		results = append(results, toolResult.Structured)
		retained = append(retained, toolResult.RetainedOutputRefs...)
		if index > 0 {
			content.WriteString("\n\n")
		}
		content.WriteString(toolResult.Content)
		if !toolResult.OK {
			ok = false
			if firstErr == nil {
				firstErr = runErr
			}
			break
		}
	}
	structured := map[string]any{"commands": results}
	return domain.ToolResult{
		Name:               "run_tests",
		OK:                 ok,
		Content:            content.String(),
		ModelContent:       content.String(),
		Structured:         structured,
		RetainedOutputRefs: retained,
		Error:              errorString(firstErr),
		ToolError:          toolErrorFromErr(firstErr),
	}
}

type bashInput struct {
	Command        string            `json:"command"`
	CWD            string            `json:"cwd"`
	TimeoutSeconds int               `json:"timeoutSeconds"`
	Network        string            `json:"network"`
	Mode           string            `json:"mode"`
	Stdin          string            `json:"stdin"`
	Env            map[string]string `json:"env"`
	Justification  string            `json:"justification"`
}

func parseBashArgs(args json.RawMessage) (bashInput, error) {
	var input bashInput
	if err := json.Unmarshal(args, &input); err != nil {
		return input, errors.New("invalid bash arguments")
	}
	input.Command = strings.TrimSpace(input.Command)
	input.CWD = strings.TrimSpace(input.CWD)
	input.Network = strings.TrimSpace(input.Network)
	input.Mode = normalizeSandboxMode(input.Mode)
	input.Justification = strings.TrimSpace(input.Justification)
	if input.Command == "" {
		return input, errors.New("command is required")
	}
	if input.Network != "" && input.Network != "deny" && input.Network != "inherit" {
		return input, errors.New("network must be deny or inherit")
	}
	if input.Mode != "foreground" && input.Mode != "background" && input.Mode != "pty" {
		return input, errors.New("mode must be foreground, background, or pty")
	}
	for key := range input.Env {
		if isSecretEnvName(key) {
			return input, fmt.Errorf("env override %s is denied because it looks secret-bearing", key)
		}
	}
	return input, nil
}

func shouldUsePersistentAgentShell(input bashInput, execCtx domain.ToolExecutionContext) bool {
	return strings.TrimSpace(execCtx.SessionID) != "" &&
		input.Mode == "foreground" &&
		input.Stdin == "" &&
		len(input.Env) == 0
}

type runTestsInput struct {
	Target         string `json:"target"`
	Kind           string `json:"kind"`
	Filter         string `json:"filter"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
}

func parseRunTestsArgs(args json.RawMessage) (runTestsInput, error) {
	var input runTestsInput
	if len(args) > 0 {
		if err := json.Unmarshal(args, &input); err != nil {
			return input, errors.New("invalid run_tests arguments")
		}
	}
	input.Target = firstNonEmpty(strings.TrimSpace(input.Target), "all")
	input.Kind = firstNonEmpty(strings.TrimSpace(input.Kind), "auto")
	input.Filter = strings.TrimSpace(input.Filter)
	switch input.Target {
	case "core", "desktop", "all":
	default:
		return input, fmt.Errorf("unsupported run_tests target %q", input.Target)
	}
	switch input.Kind {
	case "test", "lint", "build", "auto":
	default:
		return input, fmt.Errorf("unsupported run_tests kind %q", input.Kind)
	}
	return input, nil
}

func runTestsCommands(input runTestsInput) ([]string, error) {
	target := input.Target
	kind := input.Kind
	if kind == "auto" {
		switch target {
		case "desktop":
			kind = "lint"
		default:
			kind = "test"
		}
	}
	if strings.TrimSpace(input.Filter) != "" {
		return nil, errors.New("run_tests filter is not supported by the initial command mapping")
	}
	switch target + ":" + kind {
	case "core:test", "all:test":
		return []string{"npm run test:core"}, nil
	case "desktop:lint":
		return []string{"npm run lint"}, nil
	case "desktop:build", "all:build":
		return []string{"npm run build"}, nil
	}
	return nil, fmt.Errorf("unsupported run_tests combination target=%s kind=%s", target, kind)
}

type preparedShellCommand struct {
	request   SandboxRequest
	detect    CommandDetection
	policy    CommandPolicyEvaluation
	cwd       string
	timeout   time.Duration
	timeoutS  int
	metadata  map[string]any
	toolName  string
	workspace string
}

func prepareShellCommand(workspaceRoot string, execCtx domain.ToolExecutionContext, toolName string, command string, cwdArg string, timeoutSeconds int, network string, mode string, stdin string, env map[string]string) (preparedShellCommand, error) {
	prepared := preparedShellCommand{toolName: toolName, workspace: workspaceRoot}
	mode = normalizeSandboxMode(mode)
	externalCWD, cwd, err := cwdIsExternal(workspaceRoot, cwdArg)
	if err != nil {
		return prepared, err
	}
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeoutSeconds <= 0 {
		timeout = defaultCommandTimeout
		timeoutSeconds = int(defaultCommandTimeout.Seconds())
	}
	timeout = clampCommandTimeout(timeout)
	timeoutSeconds = int(timeout.Seconds())
	detection := DetectCommand(command, cwd, workspaceRoot, toolName)
	capabilities := []string{"shell.exec." + mode}
	if stdin != "" {
		capabilities = append(capabilities, "shell.stdin")
	}
	if len(env) > 0 {
		for key := range env {
			if isSecretEnvName(key) {
				prepared.detect = detection
				return prepared, fmt.Errorf("env override %s is denied because it looks secret-bearing", key)
			}
			if !envOverrideKeyAllowed(key) {
				prepared.detect = detection
				return prepared, fmt.Errorf("env override %s is not allowed by Phase 3 env policy", key)
			}
		}
		capabilities = append(capabilities, "shell.env.override")
	}
	if externalCWD {
		capabilities = append(capabilities, "shell.cwd.external")
		detection.ExternalPaths = appendUniqueStrings(detection.ExternalPaths, cwd)
	}
	if network == "inherit" || detection.Category == CommandCategoryNetwork {
		capabilities = append(capabilities, "shell.network")
	}
	detection.Capabilities = appendUniqueStrings(detection.Capabilities, capabilities...)
	detection.ApprovalKey = commandApprovalKey(workspaceRoot, cwd, detection.NormalizedCommand, detection.Argv, toolName, "local", "default", firstNonEmpty(network, detection.NetworkHint), detection.Category, detection.RiskLevel, detection.Capabilities)
	policy := EvaluateCommandPolicy(detection, toolName)
	if network == "" {
		network = policy.NetworkPolicy
	}
	if network == "" {
		network = "deny"
	}
	if network == "deny" && policy.Category == CommandCategoryNetwork {
		policy.Decision = CommandDecisionAsk
		policy.NetworkPolicy = "deny"
		policy.Justification = firstNonEmpty(policy.Justification, "network command requested with denied network policy")
	}
	prepared.detect = detection
	prepared.policy = policy
	prepared.cwd = cwd
	prepared.timeout = timeout
	prepared.timeoutS = timeoutSeconds
	prepared.metadata = commandPolicyMetadata(detection, policy, timeoutSeconds, cwd)
	prepared.request = SandboxRequest{
		WorkspaceRoot:    workspaceRoot,
		CWD:              cwd,
		Command:          detection.NormalizedCommand,
		Argv:             detection.Argv,
		Mode:             mode,
		Timeout:          timeout,
		Stdin:            stdin,
		NetworkPolicy:    network,
		Backend:          "local",
		OutputPolicy:     execCtx.OutputPolicy,
		SessionID:        execCtx.SessionID,
		TurnID:           execCtx.TurnID,
		ToolCallID:       execCtx.ToolCallID,
		ToolName:         toolName,
		ApprovalKey:      detection.ApprovalKey,
		EnvAllowlist:     defaultEnvAllowlist(),
		Env:              nil,
		EnvOverrides:     env,
		Shell:            "",
		AllowExternalCWD: externalCWD,
	}
	if policy.Decision == CommandDecisionDeny {
		return prepared, commandPolicyDeniedError(policy)
	}
	if toolName == "run_tests" && policy.Decision != CommandDecisionAllow {
		return prepared, errors.New("run_tests command is not an allowed declared command")
	}
	return prepared, nil
}

func commandToolResult(toolName string, prepared preparedShellCommand, result SandboxResult, runErr error) domain.ToolResult {
	ok := runErr == nil && result.ExitCode == 0 && !result.TimedOut && !result.Cancelled
	content := commandResultContent(result, runErr)
	structured := commandResultStructured(prepared, result)
	retained := []string{}
	if result.StdoutRef != "" {
		retained = append(retained, result.StdoutRef)
	}
	if result.StderrRef != "" {
		retained = append(retained, result.StderrRef)
	}
	return domain.ToolResult{
		Name:               toolName,
		OK:                 ok,
		Content:            content,
		ModelContent:       content,
		Structured:         structured,
		RetainedOutputRefs: retained,
		Error:              errorString(runErr),
		ToolError:          toolErrorFromErr(runErr),
		Truncated:          result.Truncated,
		OriginalSize:       result.OriginalSize,
	}
}

func commandToolError(toolName string, prepared preparedShellCommand, err error) domain.ToolResult {
	structured := map[string]any{}
	for key, value := range prepared.metadata {
		structured[key] = value
	}
	structured["ok"] = false
	content := err.Error()
	return domain.ToolResult{
		Name:         toolName,
		OK:           false,
		Content:      content,
		ModelContent: content,
		Structured:   structured,
		Error:        err.Error(),
		ToolError:    toolErrorFromErr(err),
	}
}

func commandResultContent(result SandboxResult, runErr error) string {
	lines := []string{
		fmt.Sprintf("Command: %s", result.Command),
		fmt.Sprintf("CWD: %s", filepath.ToSlash(result.CWD)),
		fmt.Sprintf("Exit code: %d", result.ExitCode),
		fmt.Sprintf("Duration: %dms", result.Duration.Milliseconds()),
	}
	if result.TimedOut {
		lines = append(lines, "Timed out: true")
	}
	if result.Cancelled {
		lines = append(lines, "Cancelled: true")
	}
	if runErr != nil && !result.TimedOut && !result.Cancelled {
		lines = append(lines, "Error: "+runErr.Error())
	}
	if strings.TrimSpace(result.Stdout) != "" {
		lines = append(lines, "STDOUT:\n"+result.Stdout)
	}
	if strings.TrimSpace(result.Stderr) != "" {
		lines = append(lines, "STDERR:\n"+result.Stderr)
	}
	if result.Truncated {
		lines = append(lines, "Output truncated: true")
	}
	return strings.Join(lines, "\n")
}

func commandResultStructured(prepared preparedShellCommand, result SandboxResult) map[string]any {
	out := map[string]any{
		"command":        result.Command,
		"argv":           result.Argv,
		"mode":           result.Mode,
		"cwd":            filepath.ToSlash(result.CWD),
		"exitCode":       result.ExitCode,
		"stdout":         result.Stdout,
		"stderr":         result.Stderr,
		"timedOut":       result.TimedOut,
		"cancelled":      result.Cancelled,
		"durationMs":     result.Duration.Milliseconds(),
		"backend":        result.Backend,
		"networkPolicy":  result.NetworkPolicy,
		"approvalKey":    prepared.detect.ApprovalKey,
		"policyDecision": prepared.policy.Decision,
		"riskLevel":      prepared.policy.RiskLevel,
		"category":       prepared.policy.Category,
		"truncated":      result.Truncated,
		"originalSize":   result.OriginalSize,
		"stdoutRef":      result.StdoutRef,
		"stderrRef":      result.StderrRef,
		"processId":      result.ProcessID,
		"processRef":     result.ProcessRef,
	}
	if result.ProcessRef != "" && strings.HasPrefix(result.ProcessRef, "agent-shell:") {
		out["persistentShell"] = true
		out["restoredState"] = "cwd_only"
	}
	for key, value := range prepared.metadata {
		if _, exists := out[key]; !exists {
			out[key] = value
		}
	}
	return out
}

func toolErrorFromErr(err error) *domain.ToolError {
	if err == nil {
		return nil
	}
	code := "tool_error"
	var sandboxErr *SandboxError
	if errors.As(err, &sandboxErr) && sandboxErr.Code != "" {
		code = sandboxErr.Code
	}
	if strings.HasPrefix(err.Error(), "command denied:") {
		code = "command_denied"
	}
	return &domain.ToolError{Code: code, Message: err.Error()}
}
