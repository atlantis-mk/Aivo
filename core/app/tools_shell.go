package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"aivo/core/domain"
)

const shellNamespaceDescription = "Foreground non-interactive Bash execution in the active execution environment."

type BashTool struct {
	workspaceRoot string
	runner        SandboxRunner
	processes     *ShellProcessRegistry
	agentShells   *AgentShellRegistry
	loadSavedCWD  func(sessionID string, workspaceRoot string) string
	saveCWD       func(sessionID string, workspaceRoot string, cwd string)
	outputSink    ShellOutputSink
	environment   ExecutionEnvironment
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
		Description:          "Run one foreground, non-interactive Bash command in the active execution environment. Each call has independent shell state and bounded stdout/stderr.",
		Namespace:            filesystemNamespace,
		NamespaceDescription: shellNamespaceDescription,
		Capability:           "shell.exec",
		RiskLevel:            "critical",
		Category:             "shell",
		Toolsets:             []string{"shell", "coding"},
		RequiresWorkspace:    true,
		ImplementationHash:   executionEnvironmentHash(t.environment),
		InputSchema: map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"command": map[string]any{"type": "string", "minLength": 1, "description": "One foreground, non-interactive Bash command. Shell state does not persist between calls."},
				"timeout": map[string]any{"type": "integer", "minimum": 1, "maximum": int(maxCommandTimeout.Seconds()), "description": "Timeout in seconds. Defaults to 30 and caps at 300."},
			},
			"required": []string{"command"},
		},
	}
}

func (t *BashTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	if t.environment != nil {
		return t.environment.ExecutePrimitive(ctx, "bash", args, execCtx)
	}
	input, err := parsePrimitiveBashArgs(args)
	if err != nil {
		return primitiveError("bash", "invalid_arguments", err)
	}
	workspaceRoot := toolWorkspaceRoot(t.workspaceRoot, execCtx)
	prepared, err := prepareShellCommand(workspaceRoot, execCtx, "bash", input.Command, "", input.TimeoutSeconds, "", "foreground", "", nil)
	if err != nil {
		return commandToolError("bash", prepared, err)
	}
	bashPath, err := resolveBashExecutable(workspaceRoot)
	if err != nil {
		return primitiveError("bash", "bash_unavailable", err)
	}
	prepared.request.Shell = bashPath
	prepared.request.OutputPolicy.MaxChars = defaultStreamMaxChars
	prepared.request.OutputSink = t.outputSink
	result, runErr := t.runner.Run(ctx, prepared.request)
	return commandToolResult("bash", prepared, result, runErr)
}

func resolveBashExecutable(workspaceRoot string) (string, error) {
	if configured := strings.TrimSpace(loadEffectiveRuntimeConfig(workspaceRoot).Config.ExecutionEnvironment.BashPath); configured != "" {
		info, err := os.Stat(configured)
		if err != nil || info.IsDir() {
			return "", errors.New("configured Bash executable is unavailable")
		}
		return configured, nil
	}
	if configured := strings.TrimSpace(os.Getenv("AIVO_BASH_PATH")); configured != "" {
		info, err := os.Stat(configured)
		if err != nil || info.IsDir() {
			return "", errors.New("configured Bash executable is unavailable")
		}
		return configured, nil
	}
	name := "bash"
	if runtime.GOOS == "windows" {
		name = "bash.exe"
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return "", errors.New("Bash is unavailable; configure a Bash-compatible executable")
	}
	return path, nil
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
