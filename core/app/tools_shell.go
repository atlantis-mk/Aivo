package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

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
