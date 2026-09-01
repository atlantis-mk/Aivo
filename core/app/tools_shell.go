package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"aivo/core/domain"
)

const shellNamespaceDescription = "Foreground non-interactive shell execution in the active execution environment; Windows uses PowerShell first and Command Prompt as fallback."

func resolveCommandShell(workspaceRoot string, requested string) (string, error) {
	if requested = strings.TrimSpace(requested); requested != "" {
		return resolveShellExecutable(requested)
	}
	if runtime.GOOS == "windows" {
		for _, fallback := range windowsShellCandidates() {
			if path, err := exec.LookPath(fallback); err == nil {
				return path, nil
			}
		}
		return "", errors.New("no supported Windows shell is available; install PowerShell or pass shell explicitly")
	}
	if shell := strings.TrimSpace(os.Getenv("SHELL")); shell != "" {
		if path, err := resolveShellExecutable(shell); err == nil {
			return path, nil
		}
	}
	for _, fallback := range []string{"/bin/zsh", "/bin/bash", "/bin/sh"} {
		if path, err := resolveShellExecutable(fallback); err == nil {
			return path, nil
		}
	}
	if path, err := exec.LookPath("sh"); err == nil {
		return path, nil
	}
	return "", errors.New("no supported command shell is available; pass shell explicitly")
}

func resolveShellExecutable(shell string) (string, error) {
	shell = strings.TrimSpace(shell)
	if shell == "" {
		return "", errors.New("shell is required")
	}
	if !supportedCommandShellName(shellExecutableName(shell)) {
		return "", errors.New("unsupported shell executable")
	}
	if filepath.IsAbs(shell) || strings.ContainsAny(shell, `/\`) {
		info, err := os.Stat(shell)
		if err != nil || info.IsDir() {
			return "", errors.New("shell executable is unavailable")
		}
		return shell, nil
	}
	path, err := exec.LookPath(shell)
	if err != nil {
		return "", errors.New("shell executable is unavailable")
	}
	return path, nil
}

func supportedCommandShellName(name string) bool {
	switch name {
	case "zsh", "zsh.exe", "bash", "bash.exe", "sh", "sh.exe", "powershell", "powershell.exe", "pwsh", "pwsh.exe", "cmd", "cmd.exe":
		return true
	default:
		return false
	}
}

func windowsShellCandidates() []string {
	return []string{"powershell.exe", "pwsh.exe", "cmd.exe"}
}

func shellRuntimeInstruction(workspaceRoot string) string {
	shell, err := resolveCommandShell(workspaceRoot, "")
	if err != nil {
		return "The `exec_command` tool has no available command shell in this environment. Do not call it until the environment is configured."
	}
	return shellRuntimeInstructionForExecutable(shell)
}

func shellRuntimeInstructionForExecutable(shell string) string {
	switch shellExecutableName(shell) {
	case "powershell", "powershell.exe", "pwsh", "pwsh.exe":
		return "The `exec_command` tool executes through PowerShell in this environment. Write PowerShell syntax only; do not use Bash-specific syntax. The command is non-interactive."
	case "cmd", "cmd.exe":
		return "The `exec_command` tool executes through Windows Command Prompt in this environment. Write cmd.exe syntax only; do not use Bash or PowerShell syntax. The command is non-interactive."
	case "zsh", "zsh.exe":
		return "The `exec_command` tool executes through zsh in this environment. Write zsh-compatible shell syntax. The command is non-interactive. In zsh, unmatched globs fail before the command runs; prefer find or rg for file iteration, or enable scoped null_glob intentionally."
	case "bash", "bash.exe":
		return "The `exec_command` tool executes through Bash in this environment. Write Bash syntax. The command is non-interactive."
	case "sh", "sh.exe":
		return "The `exec_command` tool executes through sh in this environment. Write POSIX sh syntax. The command is non-interactive."
	default:
		return "The `exec_command` tool executes through a POSIX-compatible shell in this environment. Write portable shell syntax. The command is non-interactive."
	}
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
		Description:          "Preferred tool for declared test, lint, or build commands in this workspace. Use this instead of exec_command when target and kind can express the operation. This tool never accepts arbitrary shell text; it maps target and kind to known commands in code and runs through the same command policy, sandbox, timeout, and permission flow as exec_command.",
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
	workspaceRoot := toolWorkspaceRoot(t.workspaceRoot, execCtx)
	shell, err := resolveCommandShell(workspaceRoot, "")
	if err != nil {
		return primitiveError("run_tests", "bash_unavailable", err)
	}
	for index, command := range commands {
		prepared, err := prepareShellCommand(workspaceRoot, execCtx, "run_tests", command, "", input.TimeoutSeconds, "deny", "foreground", "", shell, false, nil)
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
