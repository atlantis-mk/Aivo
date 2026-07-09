package app

import (
	"context"
	"strings"

	"aivo/core/domain"
)

func (s *Service) emitApplyPatchDraft(sessionID string, turnID string, workspaceRoot string, call domain.ChatToolCall) {
	if s.onToolCallUpdated == nil || call.Name != "apply_patch" || strings.TrimSpace(call.ID) == "" {
		return
	}
	patchText, files := applyPatchDraftFiles(call.Arguments, workspaceRoot)
	if strings.TrimSpace(patchText) == "" && len(files) == 0 {
		return
	}
	now := domain.NowString(s.now())
	result := map[string]any{"draft": true}
	if len(files) > 0 {
		result["files"] = files
	}
	if strings.TrimSpace(patchText) != "" {
		result["patchTextPreview"] = bounded(patchText, 4000)
	}
	s.onToolCallUpdated(sessionID, turnID, domain.ToolCall{
		ID:          call.ID,
		SessionID:   sessionID,
		TurnID:      turnID,
		Name:        call.Name,
		Status:      domain.ToolCallStatusRunning,
		Result:      result,
		TimeCreated: now,
		TimeUpdated: now,
	}, false)
}

func (s *Service) recordToolCallStarted(ctx context.Context, sessionID string, turnID string, call domain.ChatToolCall) error {
	args := toolCallArgumentsMap(call)
	_, err := s.SaveToolCall(ctx, domain.CreateToolCallRequest{
		ID:        call.ID,
		SessionID: sessionID,
		TurnID:    turnID,
		Name:      call.Name,
		Arguments: args,
		Status:    domain.ToolCallStatusRunning,
	})
	if err == nil && s.onSessionUpdated != nil {
		s.onSessionUpdated(sessionID, nil)
	}
	return err
}

func (s *Service) sessionChatHistory(ctx context.Context, sessionID string, limit int) ([]domain.ChatMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	events, err := s.store.ListSessionEvents(ctx, sessionID, false, 500)
	if err != nil {
		return nil, err
	}
	messages := make([]domain.ChatMessage, 0, len(events))
	for _, event := range events {
		if event.Type != domain.EventTypeUserMessage && event.Type != domain.EventTypeAssistantMessage {
			continue
		}
		role := event.Role
		if role == "" {
			if event.Type == domain.EventTypeUserMessage {
				role = domain.EventRoleUser
			} else {
				role = domain.EventRoleAssistant
			}
		}
		messages = append(messages, domain.ChatMessage{Role: role, Text: event.Content})
	}
	if len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}
	return messages, nil
}

func (s *Service) toolsForWorkspace(workspaceRoot string) (*Registry, *ToolRuntime) {
	if workspaceRoot == "" {
		return nil, nil
	}
	registry, err := NewCodingToolRegistryWithShellOutputSink(workspaceRoot, func(event ShellOutputEvent) {
		if s.onShellOutput != nil {
			s.onShellOutput(event)
		}
	})
	if err != nil {
		return nil, nil
	}
	if bash, ok := registry.Get("bash"); ok {
		if bashTool, ok := bash.(*BashTool); ok {
			bashTool.SetPersistentCWDHooks(s.loadAgentShellCWD, s.saveAgentShellCWD)
		}
	}
	for _, tool := range newAgentRuntimeTools(s) {
		_ = registry.Register(tool)
	}
	_ = registry.Register(NewSkillLoadTool(s))
	if s.pluginManager == nil {
		s.pluginManager = NewPluginManager(s.store)
	}
	if s.mcpManager == nil {
		s.mcpManager = NewMCPManager(s.store, s.secrets)
	}
	s.pluginManager.RegisterEnabledTools(context.Background(), registry)
	s.mcpManager.RegisterEnabledTools(context.Background(), registry)
	_ = registry.RegisterScoped(NewToolResolveTool(registry, s.resolveToolsWithAuxiliaryModel, s.rememberDeferredToolUsed), domain.ToolSourceBridge, "tool_discovery", "")
	runtime := NewToolRuntime(registry, workspaceRoot)
	runtime.PluginHooks = s.pluginManager
	runtime.Permissions = NewPermissionEngine(s.store)
	runtime.Permissions.notifier = s.permissionNotifier
	runtime.Permissions.onRequest = func(request domain.PermissionRequest) {
		if request.SessionID != "" && request.ToolCallID != "" {
			_, _ = s.SaveToolCall(context.Background(), domain.CreateToolCallRequest{
				ID:            request.ToolCallID,
				SessionID:     request.SessionID,
				TurnID:        request.TurnID,
				Name:          request.ToolName,
				Arguments:     request.Arguments,
				Status:        domain.ToolCallStatusPending,
				ResultSummary: "Waiting for permission approval",
				Result: map[string]any{
					"ok":                 false,
					"call_id":            request.ToolCallID,
					"name":               request.ToolName,
					"pendingApprovalId":  request.ID,
					"permissionDecision": domain.PermissionDecisionAsk,
				},
				Error: "permission approval is required",
			})
		}
		if s.onPermissionRequested != nil {
			s.onPermissionRequested(request)
		}
		if s.onSessionUpdated != nil && request.SessionID != "" {
			s.onSessionUpdated(request.SessionID, nil)
		}
	}
	return registry, runtime
}

func (s *Service) loadAgentShellCWD(sessionID string, workspaceRoot string) string {
	return s.loadAgentShellWorkingDirectory(sessionID, workspaceRoot)
}

func (s *Service) saveAgentShellCWD(sessionID string, workspaceRoot string, cwd string) {
	s.saveAgentShellWorkingDirectory(sessionID, workspaceRoot, cwd)
}

func workspaceInternalCWD(workspaceRoot string, cwd string) string {
	return normalizeWorkspaceInternalCWD(workspaceRoot, cwd)
}

func logToolCalls(calls []domain.ChatToolCall) {
	logModelToolCalls(calls)
}

func (s *Service) recordToolResult(ctx context.Context, sessionID string, turnID string, call domain.ChatToolCall, result domain.ToolResult) error {
	return s.saveToolResult(ctx, sessionID, turnID, call, result)
}

func (s *Service) recordToolResultWithMetadata(ctx context.Context, sessionID string, turnID string, call domain.ChatToolCall, result domain.ToolResult, metadata map[string]any) error {
	return s.saveToolResultWithMetadata(ctx, sessionID, turnID, call, result, metadata)
}

func (s *Service) appendToolResultEvent(ctx context.Context, sessionID string, turnID string, result domain.ToolResult) error {
	_, err := s.AppendEvent(ctx, domain.AppendEventRequest{
		SessionID:  sessionID,
		TurnID:     turnID,
		Type:       domain.EventTypeToolResult,
		Role:       domain.EventRoleTool,
		Visibility: domain.EventVisibilityInternal,
		Content:    toolResultSummary(result),
		Payload:    map[string]any{"callId": result.CallID, "name": result.Name, "ok": result.OK},
	})
	return err
}

func encodeToolResultForModel(result domain.ToolResult) string {
	return encodeToolResult(result)
}

func toolCallArgumentsMap(call domain.ChatToolCall) map[string]any {
	return decodeToolCallArguments(call)
}

func toolResultSummary(result domain.ToolResult) string {
	return summarizeToolResult(result)
}

func toolCallsPayload(calls []domain.ChatToolCall) []map[string]any {
	return buildToolCallsPayload(calls)
}

func providerAuthMethods(id string, env string) []domain.ProviderAuthMethod {
	return buildProviderAuthMethods(id, env)
}
