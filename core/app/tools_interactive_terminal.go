package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"

	"aivo/core/domain"
)

const (
	ExecCommandToolName = "exec_command"
	WriteStdinToolName  = "write_stdin"
)

type execCommandInput struct {
	Cmd                string   `json:"cmd"`
	Workdir            string   `json:"workdir"`
	Shell              string   `json:"shell"`
	Login              *bool    `json:"login"`
	TTY                *bool    `json:"tty"`
	YieldTimeMS        int      `json:"yield_time_ms"`
	MaxOutputTokens    int      `json:"max_output_tokens"`
	SandboxPermissions string   `json:"sandbox_permissions"`
	Justification      string   `json:"justification"`
	PrefixRule         []string `json:"prefix_rule"`
}

type writeStdinInput struct {
	ProcessRef     string `json:"process_ref"`
	Chars          string `json:"chars"`
	Cursor         *int64 `json:"cursor"`
	YieldTimeMS    int    `json:"yield_time_ms"`
	MaxOutputChars int    `json:"max_output_chars"`
	Rows           int    `json:"rows"`
	Cols           int    `json:"cols"`
	PressEnter     bool   `json:"press_enter"`
	Terminate      bool   `json:"terminate"`
	LeaseVersion   int64  `json:"lease_version"`
}

type ExecCommandTool struct {
	workspaceRoot string
	registry      *AgentPTYRegistry
	outputSink    ShellOutputSink
	environment   ExecutionEnvironment
}

type WriteStdinTool struct {
	workspaceRoot string
	registry      *AgentPTYRegistry
	environment   ExecutionEnvironment
}

func NewExecCommandTool(workspaceRoot string, registry *AgentPTYRegistry, outputSink ShellOutputSink) *ExecCommandTool {
	if registry == nil {
		registry = NewAgentPTYRegistry()
	}
	return &ExecCommandTool{workspaceRoot: workspaceRoot, registry: registry, outputSink: outputSink}
}

func NewWriteStdinTool(workspaceRoot string, registry *AgentPTYRegistry) *WriteStdinTool {
	if registry == nil {
		registry = NewAgentPTYRegistry()
	}
	return &WriteStdinTool{workspaceRoot: workspaceRoot, registry: registry}
}

func (t *ExecCommandTool) Spec() domain.ToolSpec {
	yieldTimeMSDescription := "Wait before yielding output. Defaults to 10000 ms; effective range is 250-30000 ms."
	if runtime.GOOS == "windows" {
		yieldTimeMSDescription = "Maximum time to wait before returning a session ID for a still-running command. Commands that finish sooner return immediately. For ordinary commands, omit this parameter to use the 10000 ms default. Effective range on Windows is 10000-30000 ms."
	}
	return domain.ToolSpec{
		Name:        ExecCommandToolName,
		Description: execCommandDescription(),
		Namespace:   filesystemNamespace, NamespaceDescription: shellNamespaceDescription,
		Capability: "shell.exec", RiskLevel: "critical", Category: "shell", Toolsets: []string{"shell", "coding"}, RequiresWorkspace: true,
		InputSchema: map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"cmd":               map[string]any{"type": "string", "description": "Shell command to execute."},
				"workdir":           map[string]any{"type": "string", "description": "Working directory for the command. Defaults to the turn cwd."},
				"shell":             map[string]any{"type": "string", "description": "Shell binary to launch. Defaults to the user's default shell."},
				"login":             map[string]any{"type": "boolean", "description": "True runs the shell with -l/-i semantics; false disables them. Defaults to true."},
				"tty":               map[string]any{"type": "boolean", "description": "True allocates a PTY for the command; false or omitted uses plain pipes."},
				"yield_time_ms":     map[string]any{"type": "number", "description": yieldTimeMSDescription},
				"max_output_tokens": map[string]any{"type": "number", "description": "Output token budget. Defaults to 10000 tokens; larger requests may be capped by policy."},
				"sandbox_permissions": map[string]any{
					"type":        "string",
					"enum":        []string{"use_default", "require_escalated"},
					"description": "Per-command sandbox override. Defaults to `use_default`; use `require_escalated` for unsandboxed execution.",
				},
				"justification": map[string]any{"type": "string", "description": "User-facing approval question for `require_escalated`; omit otherwise."},
				"prefix_rule":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Reusable approval prefix for `cmd`, only with `sandbox_permissions: \"require_escalated\"`; for example [\"git\", \"pull\"]."},
			},
			"required": []string{"cmd"},
		},
	}
}

func execCommandDescription() string {
	description := "Runs a command in a PTY, returning output or a session ID for ongoing interaction."
	if runtime.GOOS == "windows" {
		return description + "\n\n" + windowsShellGuidance()
	}
	return description
}

func windowsShellGuidance() string {
	return "Windows safety rules:\n- Do not compose destructive filesystem commands across shells. Do not enumerate paths in PowerShell and then pass them to `cmd /c`, batch builtins, or another shell for deletion or moving. Use one shell end-to-end, prefer native PowerShell cmdlets such as `Remove-Item` / `Move-Item` with `-LiteralPath`, and avoid string-built shell commands for file operations.\n- Before any recursive delete or move on Windows, verify the resolved absolute target paths stay within the intended workspace or explicitly named target directory. Never issue a recursive delete or move against a computed path if the final target has not been checked.\n- When using `Start-Process` to launch a background helper or service, pass `-WindowStyle Hidden` unless the user explicitly asked for a visible interactive window. Use visible windows only for interactive tools the user needs to see or control."
}

func (t *ExecCommandTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	if t.environment != nil {
		return t.environment.ExecutePrimitive(ctx, ExecCommandToolName, args, execCtx)
	}
	input, err := parseExecCommandArgs(args)
	if err != nil {
		return toolError(ExecCommandToolName, err)
	}
	workspaceRoot := toolWorkspaceRoot(t.workspaceRoot, execCtx)
	shell, err := resolveCommandShell(workspaceRoot, input.Shell)
	if err != nil {
		return toolError(ExecCommandToolName, err)
	}
	login := resolveLoginShell(input.Login, shell)
	prepared, err := prepareShellCommand(workspaceRoot, execCtx, ExecCommandToolName, input.Cmd, input.Workdir, 0, "", "pty", "", shell, login, nil)
	if err != nil {
		return commandToolError(ExecCommandToolName, prepared, err)
	}
	prepared.request.OutputSink = t.outputSink
	result, err := t.registry.Start(ctx, prepared.request, 0, 0, time.Duration(input.YieldTimeMS)*time.Millisecond, outputTokenBudgetToChars(input.MaxOutputTokens))
	if err != nil {
		if result.ProcessRef != "" {
			return interactiveTerminalWaitError(ExecCommandToolName, result, err)
		}
		return interactiveTerminalError(ExecCommandToolName, err)
	}
	return interactiveTerminalResult(ExecCommandToolName, result)
}

func (t *WriteStdinTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:        WriteStdinToolName,
		Description: "Write characters to, poll, resize, or terminate an existing interactive process returned by exec_command or listed in live_terminals. Use this for follow-up requests about a previous or ongoing terminal; do not restart the command. For normal line input, pass plain text in chars and set press_enter=true in the same call. Do not append escaped \\r, \\n, or \\u000a text to chars. Pass the last cursor to receive only new output. Empty chars with press_enter=false waits without writing.",
		Namespace:   filesystemNamespace, NamespaceDescription: shellNamespaceDescription,
		Capability: "shell.exec", RiskLevel: "critical", Category: "shell", Toolsets: []string{"shell", "coding"}, RequiresWorkspace: true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"process_ref":      map[string]any{"type": "string"},
				"chars":            map[string]any{"type": "string", "description": "Exact text or raw characters to write. For normal line input, do not append an escaped newline; use press_enter=true."},
				"press_enter":      map[string]any{"type": "boolean", "description": "Press Enter after chars by appending one real carriage-return byte. For example, use chars=exit and press_enter=true."},
				"cursor":           map[string]any{"type": "integer", "minimum": 0},
				"yield_time_ms":    map[string]any{"type": "integer", "minimum": 100, "maximum": int(agentPTYMaxYield.Milliseconds())},
				"max_output_chars": map[string]any{"type": "integer", "minimum": 256, "maximum": agentPTYBufferCap},
				"rows":             map[string]any{"type": "integer", "minimum": 4, "maximum": 200},
				"cols":             map[string]any{"type": "integer", "minimum": 20, "maximum": 400},
				"terminate":        map[string]any{"type": "boolean"},
				"lease_version":    map[string]any{"type": "integer", "minimum": 0, "description": "Input lease version returned by the previous terminal result."},
			},
			"required": []string{"process_ref"},
		},
	}
}

func (t *WriteStdinTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	if t.environment != nil {
		return t.environment.ExecutePrimitive(ctx, WriteStdinToolName, args, execCtx)
	}
	input, err := parseWriteStdinArgs(args)
	if err != nil {
		return toolError(WriteStdinToolName, err)
	}
	cursor := int64(-1)
	if input.Cursor != nil {
		cursor = *input.Cursor
	}
	chars := input.Chars
	if input.PressEnter {
		chars += "\r"
	}
	result, err := t.registry.Write(ctx, AgentPTYWriteInput{
		WorkspaceRoot: toolWorkspaceRoot(t.workspaceRoot, execCtx), SessionID: execCtx.SessionID,
		ProcessRef: input.ProcessRef, Chars: chars, Cursor: cursor,
		YieldTime: time.Duration(input.YieldTimeMS) * time.Millisecond, MaxOutput: input.MaxOutputChars,
		Rows: input.Rows, Cols: input.Cols, Terminate: input.Terminate,
		LeaseVersion: input.LeaseVersion,
	})
	if err != nil {
		if result.ProcessRef != "" {
			return interactiveTerminalWaitError(WriteStdinToolName, result, err)
		}
		return interactiveTerminalError(WriteStdinToolName, err)
	}
	return interactiveTerminalResult(WriteStdinToolName, result)
}

func parseExecCommandArgs(args json.RawMessage) (execCommandInput, error) {
	var input execCommandInput
	if err := decodeStrictToolArgs(args, &input); err != nil {
		return input, errors.New("invalid exec_command arguments")
	}
	input.Cmd = strings.TrimSpace(input.Cmd)
	input.Workdir = strings.TrimSpace(input.Workdir)
	input.Shell = strings.TrimSpace(input.Shell)
	input.SandboxPermissions = strings.TrimSpace(input.SandboxPermissions)
	input.Justification = strings.TrimSpace(input.Justification)
	if input.Cmd == "" {
		return input, errors.New("cmd is required")
	}
	if input.SandboxPermissions != "" && input.SandboxPermissions != "use_default" && input.SandboxPermissions != "require_escalated" {
		return input, errors.New("sandbox_permissions must be use_default or require_escalated")
	}
	if input.YieldTimeMS < 0 || input.YieldTimeMS > int(agentPTYMaxYield.Milliseconds()) {
		return input, fmt.Errorf("yield_time_ms must be between 0 and %d", agentPTYMaxYield.Milliseconds())
	}
	if input.MaxOutputTokens < 0 {
		return input, errors.New("max_output_tokens must be non-negative")
	}
	return input, nil
}

func outputTokenBudgetToChars(tokens int) int {
	if tokens <= 0 {
		return 0
	}
	chars := tokens * 4
	if chars < 256 {
		return 256
	}
	if chars > agentPTYBufferCap {
		return agentPTYBufferCap
	}
	return chars
}

func parseWriteStdinArgs(args json.RawMessage) (writeStdinInput, error) {
	var input writeStdinInput
	if err := json.Unmarshal(args, &input); err != nil {
		return input, errors.New("invalid write_stdin arguments")
	}
	input.ProcessRef = strings.TrimSpace(input.ProcessRef)
	if input.ProcessRef == "" {
		return input, errors.New("process_ref is required")
	}
	if (input.Rows == 0) != (input.Cols == 0) {
		return input, errors.New("rows and cols must be provided together")
	}
	if input.YieldTimeMS < 0 || input.YieldTimeMS > int(agentPTYMaxYield.Milliseconds()) {
		return input, fmt.Errorf("yield_time_ms must be between 0 and %d", agentPTYMaxYield.Milliseconds())
	}
	return input, nil
}

func interactiveTerminalResult(name string, result AgentPTYResult) domain.ToolResult {
	structured := map[string]any{
		"processRef": result.ProcessRef, "status": result.Status, "pid": result.PID,
		"cwd": result.CWD, "rows": result.Rows, "cols": result.Cols,
		"cursor": result.Cursor, "processCursor": result.ProcessCursor, "baseCursor": result.BaseCursor,
		"output": result.Output, "outputTruncated": result.OutputTruncated, "yieldReason": result.YieldReason,
		"inputMode": result.InputMode,
		"attention": result.Attention, "inputOwner": result.InputOwner, "leaseMode": result.LeaseMode, "leaseVersion": result.LeaseVersion,
		"title": result.Title, "command": result.Command, "sessionId": result.SessionID, "origin": result.Origin,
	}
	if result.InputRequest != nil {
		request := *result.InputRequest
		request.Prompt = ""
		structured["inputRequest"] = &request
	}
	if result.ExitCode != nil {
		structured["exitCode"] = *result.ExitCode
	}
	content := fmt.Sprintf("Process: %s\nStatus: %s\nCursor: %d\nYield reason: %s", result.ProcessRef, result.Status, result.Cursor, result.YieldReason)
	if result.ExitCode != nil {
		content += fmt.Sprintf("\nExit code: %d", *result.ExitCode)
	}
	if result.OutputTruncated {
		content += "\nOutput truncated: true; continue from the returned cursor"
	}
	if result.Output != "" {
		content += "\nOUTPUT:\n" + result.Output
	}
	return domain.ToolResult{Name: name, OK: true, Content: content, ModelContent: content, Structured: structured, Truncated: result.OutputTruncated, OriginalSize: int(result.ProcessCursor - result.BaseCursor)}
}

func interactiveTerminalError(name string, err error) domain.ToolResult {
	var decisionErr *AgentPTYDecisionRequiredError
	if errors.As(err, &decisionErr) {
		structured := map[string]any{
			"code": "input_decision_required", "processRef": decisionErr.ProcessRef,
			"requestId": decisionErr.RequestID, "cursor": decisionErr.Cursor, "inputMode": decisionErr.InputMode,
		}
		content := fmt.Sprintf("Terminal input decision required for %s (request %s, cursor %d). Ask the user to choose agent once, user once, or agent always before writing.", decisionErr.ProcessRef, decisionErr.RequestID, decisionErr.Cursor)
		return domain.ToolResult{Name: name, OK: false, Content: content, ModelContent: content, Error: err.Error(), Structured: structured, ToolError: &domain.ToolError{Code: "input_decision_required", Message: err.Error()}}
	}
	return domain.ToolResult{Name: name, OK: false, Content: err.Error(), ModelContent: err.Error(), Error: err.Error(), ToolError: toolErrorFromErr(err)}
}

func interactiveTerminalWaitError(name string, result AgentPTYResult, err error) domain.ToolResult {
	toolResult := interactiveTerminalResult(name, result)
	toolResult.OK = false
	toolResult.Error = err.Error()
	toolResult.ToolError = toolErrorFromErr(err)
	toolResult.Content += "\nThe current wait was cancelled, but the PTY process remains alive and can be reattached or continued with its processRef."
	toolResult.ModelContent = toolResult.Content
	toolResult.Structured["waitError"] = err.Error()
	toolResult.Structured["processAlive"] = result.Status != AgentPTYStatusExited
	return toolResult
}
