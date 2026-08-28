package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"aivo/core/domain"
)

const (
	defaultToolTimeout    = 10 * time.Second
	defaultToolMaxRetries = 2
	defaultMaxOutputChars = 16000
)

type ToolRuntime struct {
	Registry       *Registry
	WorkspaceRoot  string
	MaxOutputChars int
	Timeout        time.Duration
	Permissions    *PermissionEngine
	ExtensionHooks ToolHookRunner
}

type ToolHookRunner interface {
	InvokeHook(context.Context, string, map[string]any) []map[string]any
}

func NewToolRuntime(registry *Registry, workspaceRoot string) *ToolRuntime {
	return &ToolRuntime{
		Registry:       registry,
		WorkspaceRoot:  workspaceRoot,
		MaxOutputChars: defaultMaxOutputChars,
		Timeout:        defaultToolTimeout,
	}
}

func (r *ToolRuntime) Execute(ctx context.Context, call domain.ChatToolCall) domain.ToolResult {
	return r.ExecuteWithContext(ctx, call, domain.ToolExecutionContext{})
}

func (r *ToolRuntime) ExecuteWithContext(ctx context.Context, call domain.ChatToolCall, execCtx domain.ToolExecutionContext) domain.ToolResult {
	start := time.Now()
	name := strings.TrimSpace(call.Name)
	log.Printf("tool_call start name=%s call_id=%s argument_bytes=%d", name, call.ID, len(call.Arguments))
	if r == nil || r.Registry == nil {
		return r.finish(call, start, toolFailure(call.ID, name, "runtime_unconfigured", "tool runtime is not configured"), false)
	}
	if name == "" {
		return r.finish(call, start, toolFailure(call.ID, name, "invalid_tool_call", "tool name is required"), false)
	}
	if execCtx.ToolSnapshot != nil && !toolSnapshotContains(execCtx.ToolSnapshot, name) {
		return r.finish(call, start, toolFailure(call.ID, name, "tool_not_advertised", fmt.Sprintf("tool %s is not available in this turn", name)), false)
	}
	expected := execCtx.ExpectedRegistrations[name]
	tool, identity, ok := r.Registry.GetRegisteredForSnapshot(name, expected.RegistrationID)
	if !ok {
		if expected.RegistrationID != "" {
			return r.finish(call, start, toolFailure(call.ID, name, "stale_tool_registration", fmt.Sprintf("tool %s registration is no longer available", name)), false)
		}
		return r.finish(call, start, toolFailure(call.ID, name, "tool_not_found", fmt.Sprintf("unknown tool: %s", name)), false)
	}
	if expected.RegistrationID != "" && identity.RegistrationID != "" && expected.RegistrationID != identity.RegistrationID {
		return r.finish(call, start, toolFailure(call.ID, name, "stale_tool_registration", fmt.Sprintf("tool %s changed since it was advertised", name)), false)
	}
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = defaultToolTimeout
	}
	if strings.TrimSpace(execCtx.WorkspaceRoot) == "" {
		execCtx.WorkspaceRoot = r.WorkspaceRoot
	}
	if execCtx.OutputPolicy.MaxChars <= 0 {
		execCtx.OutputPolicy.MaxChars = r.MaxOutputChars
	}
	if strings.TrimSpace(execCtx.ToolCallID) == "" {
		execCtx.ToolCallID = call.ID
	}
	spec := tool.Spec()
	if namespace := strings.TrimSpace(call.Namespace); namespace != "" && namespace != strings.TrimSpace(spec.Namespace) {
		return r.finish(call, start, toolFailure(call.ID, name, "tool_namespace_mismatch", fmt.Sprintf("tool %s is not registered in namespace %s", name, namespace)), false)
	}
	if isLongRunningInteractionSpec(spec) {
		timeout = 24 * time.Hour
	}
	if len(execCtx.AllowedToolsets) > 0 && !toolSpecInToolsets(spec, execCtx.AllowedToolsets) {
		return r.finish(call, start, toolFailure(call.ID, name, "toolset_denied", "tool is not in the active agent toolset"), false)
	}
	if len(call.Arguments) == 0 {
		if spec.Kind == domain.ToolKindFreeform {
			call.Arguments = json.RawMessage("")
		} else {
			call.Arguments = json.RawMessage(`{}`)
		}
	}
	if spec.Kind != domain.ToolKindFreeform {
		var decoded any
		if err := json.Unmarshal(call.Arguments, &decoded); err != nil {
			return r.finish(call, start, toolFailure(call.ID, name, "invalid_arguments", "invalid JSON arguments: "+err.Error()), false)
		}
	}
	if name == "bash" {
		if _, err := parsePrimitiveBashArgs(call.Arguments); err != nil {
			return r.finish(call, start, toolFailure(call.ID, name, "invalid_arguments", err.Error()), false)
		}
	}
	if hookResult := r.runPreToolHooks(ctx, call, spec, execCtx); hookResult != nil {
		return r.finish(call, start, *hookResult, false)
	}
	if isShellPermissionSpec(spec) && r.Permissions == nil {
		return r.finish(call, start, toolFailure(call.ID, name, "permission_denied", "shell tools require a permission engine"), false)
	}
	if r.Permissions != nil {
		evaluation := r.Permissions.Evaluate(ctx, tool, call.Arguments, execCtx)
		switch evaluation.Decision {
		case domain.PermissionDecisionAllow:
		case domain.PermissionDecisionAsk:
			result := toolFailure(call.ID, name, "permission_required", evaluation.Reason)
			result.PendingApprovalID = evaluation.RequestID
			result.PermissionDecision = evaluation.Decision
			result.PermissionRequested = true
			return r.finish(call, start, result, false)
		case domain.PermissionDecisionDeny:
			result := toolFailure(call.ID, name, firstNonEmpty(evaluation.Code, "permission_denied"), firstNonEmpty(evaluation.Reason, "permission denied"))
			result.PermissionDecision = evaluation.Decision
			return r.finish(call, start, result, false)
		default:
			result := toolFailure(call.ID, name, "permission_denied", "permission denied")
			result.PermissionDecision = domain.PermissionDecisionDeny
			return r.finish(call, start, result, false)
		}
	}
	var result domain.ToolResult
	attempts := 0
	for attempt := 0; attempt <= defaultToolMaxRetries; attempt++ {
		attempts = attempt + 1
		result = r.executeToolAttempt(ctx, call, tool, name, timeout, execCtx)
		if result.ToolError == nil || result.ToolError.Code != "timeout" || attempt == defaultToolMaxRetries {
			break
		}
		log.Printf("tool_call retry name=%s call_id=%s attempt=%d max_attempts=%d reason=timeout", name, call.ID, attempts, defaultToolMaxRetries+1)
		time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
	}
	if attempts > 1 {
		if result.Structured == nil {
			result.Structured = map[string]any{}
		}
		result.Structured["attempts"] = attempts
		if result.ToolError != nil && result.ToolError.Code == "timeout" {
			result.Error = fmt.Sprintf("tool execution timed out after %d attempts", attempts)
			result.ToolError.Message = result.Error
		}
	}
	r.runPostToolHooks(ctx, call, spec, execCtx, &result)
	maxOutput := r.MaxOutputChars
	if execCtx.OutputPolicy.MaxChars > 0 {
		maxOutput = execCtx.OutputPolicy.MaxChars
	}
	if maxOutput <= 0 {
		maxOutput = defaultMaxOutputChars
	}
	truncated := r.retainAndBoundResult(call, execCtx, &result, maxOutput)
	return r.finish(call, start, result, truncated)
}

func toolSnapshotContains(snapshot *domain.ToolSnapshot, name string) bool {
	if snapshot == nil {
		return true
	}
	for _, entry := range snapshot.Tools {
		if entry.Name == name {
			return true
		}
	}
	return false
}

func (r *ToolRuntime) executeToolAttempt(ctx context.Context, call domain.ChatToolCall, tool domain.Tool, name string, timeout time.Duration, execCtx domain.ToolExecutionContext) domain.ToolResult {
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	resultCh := make(chan domain.ToolResult, 1)
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				resultCh <- toolFailure(call.ID, name, "tool_panic", fmt.Sprintf("tool panic: %v", recovered))
			}
		}()
		result := tool.Execute(callCtx, call.Arguments, execCtx)
		result.CallID = call.ID
		if result.Name == "" {
			result.Name = name
		}
		resultCh <- result
	}()
	select {
	case <-callCtx.Done():
		if isLongRunningInteractionSpec(tool.Spec()) {
			select {
			case result := <-resultCh:
				return result
			case <-time.After(100 * time.Millisecond):
			}
		}
		if errors.Is(ctx.Err(), context.Canceled) || errors.Is(callCtx.Err(), context.Canceled) {
			return toolFailure(call.ID, name, "cancelled", "tool execution was cancelled")
		}
		return toolFailure(call.ID, name, "timeout", "tool execution timed out")
	case result := <-resultCh:
		return result
	}
}

func (r *ToolRuntime) retainAndBoundResult(call domain.ChatToolCall, execCtx domain.ToolExecutionContext, result *domain.ToolResult, maxOutput int) bool {
	if result == nil {
		return false
	}
	truncated := false
	originalContent := result.Content
	originalModelContent := result.ModelContent
	contentRef := ""
	if originalContent != "" && len(originalContent) > maxOutput {
		contentRef = r.retainToolOutput(call, execCtx, "content", originalContent)
		result.OriginalSize = maxInt(result.OriginalSize, len(originalContent))
		result.Content = truncationMarker("content", originalContent, maxOutput, contentRef)
		result.RetainedOutputRefs = appendRetainedOutputRef(result.RetainedOutputRefs, contentRef)
		result.Truncated = true
		truncated = true
	}
	if originalModelContent == "" {
		return truncated
	}
	if originalModelContent == originalContent {
		if truncated {
			result.ModelContent = result.Content
		}
		return truncated
	}
	if len(originalModelContent) > maxOutput {
		modelRef := r.retainToolOutput(call, execCtx, "model", originalModelContent)
		result.OriginalSize = maxInt(result.OriginalSize, len(originalModelContent))
		result.ModelContent = truncationMarker("model content", originalModelContent, maxOutput, modelRef)
		result.RetainedOutputRefs = appendRetainedOutputRef(result.RetainedOutputRefs, modelRef)
		result.Truncated = true
		truncated = true
	}
	return truncated
}

func (r *ToolRuntime) retainToolOutput(call domain.ChatToolCall, execCtx domain.ToolExecutionContext, stream string, content string) string {
	request := SandboxRequest{
		SessionID:     execCtx.SessionID,
		TurnID:        execCtx.TurnID,
		ToolCallID:    firstNonEmpty(execCtx.ToolCallID, call.ID),
		ToolName:      call.Name,
		WorkspaceRoot: execCtx.WorkspaceRoot,
	}
	ref, err := retainSandboxOutput(request, stream, content)
	if err != nil {
		log.Printf("tool_call retain_output failed name=%s call_id=%s stream=%s err=%v", call.Name, call.ID, stream, err)
		return ""
	}
	return ref
}

func truncationMarker(label string, content string, maxOutput int, ref string) string {
	tail := content[len(content)-maxOutput:]
	if ref != "" {
		return fmt.Sprintf("[truncated: %s exceeded %d characters; full output retained at %s; showing tail]\n\n", label, maxOutput, ref) + tail
	}
	return fmt.Sprintf("[truncated: %s exceeded %d characters; showing tail]\n\n", label, maxOutput) + tail
}

func appendRetainedOutputRef(refs []string, ref string) []string {
	if ref == "" {
		return refs
	}
	for _, existing := range refs {
		if existing == ref {
			return refs
		}
	}
	return append(refs, ref)
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func (r *ToolRuntime) runPreToolHooks(ctx context.Context, call domain.ChatToolCall, spec domain.ToolSpec, execCtx domain.ToolExecutionContext) *domain.ToolResult {
	if r == nil || r.ExtensionHooks == nil || execCtx.BridgeCallDepth > 0 {
		return nil
	}
	payload := map[string]any{"tool": spec.Name, "arguments": rawMessageToAny(call.Arguments), "sessionId": execCtx.SessionID, "turnId": execCtx.TurnID, "toolCallId": call.ID, "capability": spec.Capability, "category": spec.Category}
	for _, result := range r.ExtensionHooks.InvokeHook(ctx, "pre_tool_call", payload) {
		block, _ := result["block"].(bool)
		if !block {
			continue
		}
		message, _ := result["message"].(string)
		if message == "" {
			message = "policy blocked tool call"
		}
		blocked := toolFailure(call.ID, spec.Name, "policy_denied", message)
		return &blocked
	}
	return nil
}

func (r *ToolRuntime) runPostToolHooks(ctx context.Context, call domain.ChatToolCall, spec domain.ToolSpec, execCtx domain.ToolExecutionContext, result *domain.ToolResult) {
	if r == nil || r.ExtensionHooks == nil || result == nil || execCtx.BridgeCallDepth > 0 {
		return
	}
	payload := map[string]any{"tool": spec.Name, "arguments": rawMessageToAny(call.Arguments), "result": result.Structured, "content": result.Content, "ok": result.OK, "sessionId": execCtx.SessionID, "turnId": execCtx.TurnID, "toolCallId": call.ID}
	_ = r.ExtensionHooks.InvokeHook(ctx, "post_tool_call", payload)
	for _, hookResult := range r.ExtensionHooks.InvokeHook(ctx, "transform_tool_result", payload) {
		if content, _ := hookResult["content"].(string); strings.TrimSpace(content) != "" {
			result.Content = content
			result.ModelContent = content
		}
		if structured, _ := hookResult["structured"].(map[string]any); structured != nil {
			result.Structured = structured
		}
	}
}

func rawMessageToAny(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return string(raw)
	}
	return out
}

func isShellPermissionSpec(spec domain.ToolSpec) bool {
	return spec.Capability == "shell.exec" || spec.Capability == "shell.test"
}

func isLongRunningInteractionSpec(spec domain.ToolSpec) bool {
	return (spec.Category == "interaction" && spec.Capability == "user.question") ||
		spec.Name == ExecCommandToolName || spec.Name == WriteStdinToolName || spec.Name == CodexImagegenToolName
}

func (r *ToolRuntime) finish(call domain.ChatToolCall, start time.Time, result domain.ToolResult, truncated bool) domain.ToolResult {
	errorCode := ""
	if result.ToolError != nil {
		errorCode = result.ToolError.Code
	}
	log.Printf("tool_call finish name=%s call_id=%s ok=%t duration_ms=%d truncated=%t error_code=%s", result.Name, call.ID, result.OK, time.Since(start).Milliseconds(), truncated, errorCode)
	return result
}

func toolFailure(callID string, name string, code string, message string) domain.ToolResult {
	return domain.ToolResult{
		CallID: callID,
		Name:   name,
		OK:     false,
		Error:  message,
		ToolError: &domain.ToolError{
			Code:    code,
			Message: message,
		},
	}
}
