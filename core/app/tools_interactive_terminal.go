package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"aivo/core/domain"
)

const (
	ExecCommandToolName = "exec_command"
	WriteStdinToolName  = "write_stdin"
)

type execCommandInput struct {
	Command        string            `json:"command"`
	CWD            string            `json:"cwd"`
	YieldTimeMS    int               `json:"yield_time_ms"`
	MaxOutputChars int               `json:"max_output_chars"`
	Network        string            `json:"network"`
	Env            map[string]string `json:"env"`
	Rows           int               `json:"rows"`
	Cols           int               `json:"cols"`
	Justification  string            `json:"justification"`
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
}

type WriteStdinTool struct {
	workspaceRoot string
	registry      *AgentPTYRegistry
}

func NewExecCommandTool(workspaceRoot string, registry *AgentPTYRegistry, outputSink ShellOutputSink) *ExecCommandTool {
	if registry == nil {
		registry = defaultAgentPTYRegistry
	}
	return &ExecCommandTool{workspaceRoot: workspaceRoot, registry: registry, outputSink: outputSink}
}

func NewWriteStdinTool(workspaceRoot string, registry *AgentPTYRegistry) *WriteStdinTool {
	if registry == nil {
		registry = defaultAgentPTYRegistry
	}
	return &WriteStdinTool{workspaceRoot: workspaceRoot, registry: registry}
}

func (t *ExecCommandTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:        ExecCommandToolName,
		Description: "Start one new interactive command in an owned PTY. Before starting, reuse a suitable live terminal from the live_terminals context when the user refers to a previous or ongoing process; continue it with write_stdin instead of launching a duplicate. Returns after output becomes idle, the output bound is reached, the process exits, or yield_time_ms elapses. If status is running, continue with write_stdin using processRef and cursor.",
		Namespace:   filesystemNamespace, NamespaceDescription: shellNamespaceDescription,
		Capability: "shell.exec", RiskLevel: "critical", Category: "shell", Toolsets: []string{"shell", "coding"}, RequiresWorkspace: true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command":          map[string]any{"type": "string", "description": "Command to run inside a real PTY."},
				"cwd":              map[string]any{"type": "string", "description": "Optional workspace-relative working directory."},
				"yield_time_ms":    map[string]any{"type": "integer", "minimum": 100, "maximum": int(agentPTYMaxYield.Milliseconds())},
				"max_output_chars": map[string]any{"type": "integer", "minimum": 256, "maximum": agentPTYBufferCap},
				"network":          map[string]any{"type": "string", "enum": []string{"deny", "inherit"}},
				"env":              map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}},
				"rows":             map[string]any{"type": "integer", "minimum": 4, "maximum": 200},
				"cols":             map[string]any{"type": "integer", "minimum": 20, "maximum": 400},
				"justification":    map[string]any{"type": "string"},
			},
			"required": []string{"command"},
		},
	}
}

func (t *ExecCommandTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	input, err := parseExecCommandArgs(args)
	if err != nil {
		return toolError(ExecCommandToolName, err)
	}
	workspaceRoot := toolWorkspaceRoot(t.workspaceRoot, execCtx)
	prepared, err := prepareShellCommand(workspaceRoot, execCtx, ExecCommandToolName, input.Command, input.CWD, 0, input.Network, "pty", "", input.Env)
	if err != nil {
		return commandToolError(ExecCommandToolName, prepared, err)
	}
	prepared.request.OutputSink = t.outputSink
	result, err := t.registry.Start(ctx, prepared.request, input.Rows, input.Cols, time.Duration(input.YieldTimeMS)*time.Millisecond, input.MaxOutputChars)
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
	if err := json.Unmarshal(args, &input); err != nil {
		return input, errors.New("invalid exec_command arguments")
	}
	input.Command = strings.TrimSpace(input.Command)
	input.CWD = strings.TrimSpace(input.CWD)
	input.Network = strings.TrimSpace(input.Network)
	input.Justification = strings.TrimSpace(input.Justification)
	if input.Command == "" {
		return input, errors.New("command is required")
	}
	if input.Network != "" && input.Network != "deny" && input.Network != "inherit" {
		return input, errors.New("network must be deny or inherit")
	}
	if (input.Rows == 0) != (input.Cols == 0) {
		return input, errors.New("rows and cols must be provided together")
	}
	if input.YieldTimeMS < 0 || input.YieldTimeMS > int(agentPTYMaxYield.Milliseconds()) {
		return input, fmt.Errorf("yield_time_ms must be between 0 and %d", agentPTYMaxYield.Milliseconds())
	}
	for key := range input.Env {
		if isSecretEnvName(key) {
			return input, fmt.Errorf("env override %s is denied because it looks secret-bearing", key)
		}
	}
	return input, nil
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
