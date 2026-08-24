package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"aivo/core/domain"
)

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
