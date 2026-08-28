package app

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"aivo/core/domain"
)

func logModelToolCalls(calls []domain.ChatToolCall) {
	names := make([]string, 0, len(calls))
	for _, call := range calls {
		names = append(names, call.Name)
	}
	log.Printf("model returned tool_calls count=%d names=%s", len(calls), strings.Join(names, ","))
}

func (s *Service) saveToolResult(ctx context.Context, sessionID string, turnID string, call domain.ChatToolCall, result domain.ToolResult) error {
	return s.saveToolResultWithMetadata(ctx, sessionID, turnID, call, result, nil)
}

func (s *Service) saveToolResultWithMetadata(ctx context.Context, sessionID string, turnID string, call domain.ChatToolCall, result domain.ToolResult, metadata map[string]any) error {
	args := decodeToolCallArguments(call)
	status := domain.ToolCallStatusSuccess
	if !result.OK {
		status = domain.ToolCallStatusFailed
	}
	if result.PermissionRequested {
		status = domain.ToolCallStatusPending
	}
	resultMap := map[string]any{"ok": result.OK, "call_id": result.CallID, "name": result.Name}
	if result.Content != "" {
		resultMap["content"] = bounded(result.Content, 2000)
	}
	if result.ModelContent != "" {
		resultMap["modelContent"] = bounded(result.ModelContent, 2000)
	}
	if result.Structured != nil {
		resultMap["structured"] = result.Structured
	}
	if result.Details != nil {
		resultMap["details"] = result.Details
	}
	if len(result.RetainedOutputRefs) > 0 {
		resultMap["retainedOutputRefs"] = result.RetainedOutputRefs
	}
	if len(result.Files) > 0 {
		resultMap["files"] = result.Files
	}
	if result.Error != "" {
		resultMap["error"] = result.Error
	}
	if result.PendingApprovalID != "" {
		resultMap["pendingApprovalId"] = result.PendingApprovalID
	}
	if result.PermissionDecision != "" {
		resultMap["permissionDecision"] = result.PermissionDecision
	}
	for key, value := range metadata {
		if strings.TrimSpace(key) != "" && value != nil {
			resultMap[key] = value
		}
	}
	_, err := s.SaveToolCall(ctx, domain.CreateToolCallRequest{
		ID:            call.ID,
		SessionID:     sessionID,
		TurnID:        turnID,
		Name:          call.Name,
		Arguments:     args,
		Status:        status,
		ResultSummary: summarizeToolResult(result),
		Result:        resultMap,
		Error:         result.Error,
	})
	if appendErr := s.appendToolResultEvent(ctx, sessionID, turnID, result); appendErr != nil && err == nil {
		err = appendErr
	}
	return err
}

func encodeToolResult(result domain.ToolResult) string {
	if strings.TrimSpace(result.ModelContent) != "" {
		modelResult := result
		modelResult.Content = result.ModelContent
		raw, err := json.Marshal(modelResult)
		if err == nil {
			return string(raw)
		}
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return result.Error
	}
	return string(raw)
}

func decodeToolCallArguments(call domain.ChatToolCall) map[string]any {
	args := map[string]any{}
	if len(call.Arguments) == 0 {
		return args
	}
	if err := json.Unmarshal(call.Arguments, &args); err == nil {
		return args
	}
	return args
}

func summarizeToolResult(result domain.ToolResult) string {
	if result.OK {
		return bounded(result.Content, 500)
	}
	return bounded(result.Error, 500)
}

func buildToolCallsPayload(calls []domain.ChatToolCall) []map[string]any {
	out := make([]map[string]any, 0, len(calls))
	for _, call := range calls {
		item := map[string]any{"id": call.ID, "name": call.Name, "arguments": string(call.Arguments)}
		if namespace := strings.TrimSpace(call.Namespace); namespace != "" {
			item["namespace"] = namespace
		}
		out = append(out, item)
	}
	return out
}
