package app

import (
	"context"

	"aivo/core/domain"
)

func (s *Service) recordToolCallStarted(ctx context.Context, sessionID string, turnID string, call domain.ChatToolCall, identity domain.ToolRegistrationIdentity) error {
	args := toolCallArgumentsMap(call)
	var result map[string]any
	if s.extensionSupervisor != nil && identity.Source == domain.ToolSourceExtension {
		if view := s.extensionSupervisor.ToolViewRef(identity.SourceID, identity.ImplementationHash, call.Name); view != nil {
			result = map[string]any{"details": map[string]any{"view": view}}
		}
	}
	_, err := s.SaveToolCall(ctx, domain.CreateToolCallRequest{
		ID:        call.ID,
		SessionID: sessionID,
		TurnID:    turnID,
		Name:      call.Name,
		Arguments: args,
		Status:    domain.ToolCallStatusRunning,
		Result:    result,
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
	registry, err := newCodingToolRegistry(workspaceRoot, func(event ShellOutputEvent) {
		if s.onShellOutput != nil {
			s.onShellOutput(event)
		}
	}, nil, s.ptyManager)
	if err != nil {
		return nil, nil
	}
	if err := registerDefaultHostControlTools(registry, s); err != nil {
		return nil, nil
	}
	if err := registry.RegisterScoped(NewResourceResolveTool(registry, s.resolveSessionResources), domain.ToolSourceBridge, "tool_selection", "v1"); err != nil {
		return nil, nil
	}
	if err := registry.Register(NewSkillsListTool(s)); err != nil {
		return nil, nil
	}
	if err := registry.Register(NewSkillsReadTool(s)); err != nil {
		return nil, nil
	}
	if err := registry.Register(NewCodexWebSearchTool(s)); err != nil {
		return nil, nil
	}
	if err := registry.Register(NewCodexImageGenerationTool(s)); err != nil {
		return nil, nil
	}
	if s.extensionSupervisor != nil {
		_ = s.extensionSupervisor.RegisterAllReadyTools(registry)
	}
	if s.mcpManager != nil {
		s.mcpManager.RegisterCachedEnabledTools(context.Background(), registry)
	}
	runtime := NewToolRuntime(registry, workspaceRoot)
	runtime.ExtensionHooks = s.extensionSupervisor
	runtime.Permissions = NewPermissionEngine(s.store)
	runtime.Permissions.ProjectPreflight = s.prepareProjectPermission
	runtime.Permissions.MCPRegistrationPreflight = s.prepareToolRegistrationPermission
	runtime.Permissions.PTYRegistry = s.ptyManager
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
