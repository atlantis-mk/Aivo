package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"

	"aivo/core/domain"
)

func (s *Service) runAssistantAgentLoop(
	ctx context.Context,
	input domain.SubmitSessionMessageRequest,
	history []domain.ChatMessage,
	turn domain.Turn,
	reasoningEffort string,
	serviceTier string,
	onDelta func(string),
) (string, *domain.ModelRef, error) {
	cc, _ := s.store.GetCodingContext(ctx, input.SessionID)
	modeDef, err := s.resolveAgentModeForRequest(ctx, input.SessionID, firstNonEmpty(input.AgentMode, turn.AgentMode))
	if err != nil {
		return "", nil, err
	}
	if session, err := s.store.GetRuntimeSession(ctx, input.SessionID); err == nil && session.Type == domain.SessionTypeCoding {
		if strings.TrimSpace(cc.ProjectPath) == "" && strings.TrimSpace(session.ProjectPath) != "" {
			cc, _ = s.CreateOrUpdateCodingContext(ctx, input.SessionID, session.ProjectPath)
		} else if projectPath, changed, err := ensureManagedWorkspace(cc.ProjectPath); err == nil && changed {
			cc, _ = s.CreateOrUpdateCodingContext(ctx, input.SessionID, projectPath)
		}
	}
	registry, runtime := s.toolsForWorkspace(strings.TrimSpace(cc.ProjectPath))
	ctx = withProviderRegistry(ctx, s.providerRegistryForProject(strings.TrimSpace(cc.ProjectPath)))
	allowedToolsets := allowedToolsetsForRun(modeDef, input)
	requestedModel := s.modelForAgentMode(ctx, modeDef, input.Model)
	messages := prependAgentSystemPrompt(history, modeDef)
	var model *domain.ModelRef
	for {
		var specs []domain.ToolSpec
		expectedRegistrations := map[string]domain.ToolRegistrationIdentity{}
		if registry != nil {
			specs = visibleToolSpecsForMode(modeDef.ID, registry.SpecsForToolsets(allowedToolsets))
			assembly := AssembleToolSpecsWithActivated(registry, specs, s.rememberedDeferredTools(ctx, input.SessionID))
			specs = assembly.Specs
			expectedRegistrations = assembly.ExpectedRegistrations
		}
		if s.pluginManager != nil {
			_ = s.pluginManager.InvokeHook(ctx, "pre_llm_call", map[string]any{"sessionId": input.SessionID, "turnId": turn.ID, "toolCount": len(specs), "messageCount": len(messages), "agentMode": modeDef.ID})
		}
		resp, activeModel, err := s.GenerateChatResponseStreamWithToolDelta(ctx, domain.ChatRequest{Messages: messages, Tools: specs, Temperature: modeDef.Temperature, TopP: modeDef.TopP, Options: modeDef.Options}, requestedModel, reasoningEffort, serviceTier, onDelta, func(call domain.ChatToolCall) {
			s.emitApplyPatchDraft(input.SessionID, turn.ID, strings.TrimSpace(cc.ProjectPath), call)
		})
		if s.pluginManager != nil {
			_ = s.pluginManager.InvokeHook(ctx, "post_llm_call", map[string]any{"sessionId": input.SessionID, "turnId": turn.ID, "toolCallCount": len(resp.ToolCalls), "textLength": len(resp.Text), "agentMode": modeDef.ID})
		}
		if err != nil {
			return "", activeModel, err
		}
		model = activeModel
		if len(resp.ToolCalls) == 0 {
			return resp.Text, model, nil
		}
		logToolCalls(resp.ToolCalls)
		if strings.TrimSpace(resp.Text) != "" {
			_, _ = s.AppendEvent(ctx, domain.AppendEventRequest{
				SessionID:  input.SessionID,
				TurnID:     turn.ID,
				Type:       domain.EventTypeAssistantMessage,
				Role:       domain.EventRoleAssistant,
				Visibility: domain.EventVisibilityNormal,
				Content:    resp.Text,
				Payload:    map[string]any{"phase": "before_tool"},
			})
		}
		messages = append(messages, domain.ChatMessage{Role: "assistant", Text: resp.Text, ToolCalls: resp.ToolCalls})
		_, _ = s.AppendEvent(ctx, domain.AppendEventRequest{
			SessionID:  input.SessionID,
			TurnID:     turn.ID,
			Type:       domain.EventTypeToolCall,
			Role:       domain.EventRoleAssistant,
			Visibility: domain.EventVisibilityInternal,
			Content:    bounded(resp.Text, 1000),
			Payload:    map[string]any{"toolCalls": toolCallsPayload(resp.ToolCalls)},
		})
		if isParallelDelegateBatch(resp.ToolCalls) && runtime != nil {
			for _, call := range resp.ToolCalls {
				_ = s.recordToolCallStarted(ctx, input.SessionID, turn.ID, call)
			}
			limit := loadEffectiveRuntimeConfig(strings.TrimSpace(cc.ProjectPath)).Config.MaxParallelChildren
			results := executeBoundedParallelToolCalls(ctx, resp.ToolCalls, limit, func(call domain.ChatToolCall) domain.ToolResult {
				return runtime.ExecuteWithContext(ctx, call, domain.ToolExecutionContext{
					WorkspaceRoot: strings.TrimSpace(cc.ProjectPath), SessionID: input.SessionID, TurnID: turn.ID,
					AgentMode: modeDef.ID, AllowedToolsets: allowedToolsets,
					PermissionScope:       firstNonEmpty(input.PermissionScope, permissionScopeForAgent(modeDef)),
					ExpectedRegistrations: expectedRegistrations,
				})
			})
			for index, call := range resp.ToolCalls {
				result := results[index]
				_ = s.recordToolResult(ctx, input.SessionID, turn.ID, call, result)
				messages = append(messages, domain.ChatMessage{Role: "tool", Text: encodeToolResultForModel(result), ToolCallID: call.ID, Name: call.Name})
			}
			continue
		}
		for _, call := range resp.ToolCalls {
			_ = s.recordToolCallStarted(ctx, input.SessionID, turn.ID, call)
			var result domain.ToolResult
			if runtime == nil {
				result = domain.ToolResult{CallID: call.ID, Name: call.Name, OK: false, Error: "tool runtime unavailable: this session has no workspace root"}
			} else {
				result = runtime.ExecuteWithContext(ctx, call, domain.ToolExecutionContext{
					WorkspaceRoot:         strings.TrimSpace(cc.ProjectPath),
					SessionID:             input.SessionID,
					TurnID:                turn.ID,
					AgentMode:             modeDef.ID,
					AllowedToolsets:       allowedToolsets,
					PermissionScope:       firstNonEmpty(input.PermissionScope, permissionScopeForAgent(modeDef)),
					ExpectedRegistrations: expectedRegistrations,
				})
			}
			_ = s.recordToolResult(ctx, input.SessionID, turn.ID, call, result)
			if result.PermissionRequested {
				return "等待你批准工具权限后，我可以继续执行这次修改。", model, nil
			}
			if call.Name == ToolResolveName && result.ToolError != nil && result.ToolError.Code == "no_available_tool" {
				return "", model, errors.New(result.ToolError.Message)
			}
			messages = append(messages, domain.ChatMessage{Role: "tool", Text: encodeToolResultForModel(result), ToolCallID: call.ID, Name: call.Name})
			if call.Name == "search_projects" && result.OK && projectSearchActivates(call.Arguments) {
				cc, _ = s.store.GetCodingContext(ctx, input.SessionID)
				registry, runtime = s.toolsForWorkspace(strings.TrimSpace(cc.ProjectPath))
				ctx = withProviderRegistry(ctx, s.providerRegistryForProject(strings.TrimSpace(cc.ProjectPath)))
			}
		}
	}
}

func projectSearchActivates(arguments []byte) bool {
	var input struct {
		ActivateProjectPath string `json:"activateProjectPath"`
	}
	return json.Unmarshal(arguments, &input) == nil && strings.TrimSpace(input.ActivateProjectPath) != ""
}

func isParallelDelegateBatch(calls []domain.ChatToolCall) bool {
	if len(calls) < 2 {
		return false
	}
	for _, call := range calls {
		if call.Name != "agent_delegate_task" && call.Name != "task" {
			return false
		}
	}
	return true
}

func executeBoundedParallelToolCalls(
	ctx context.Context,
	calls []domain.ChatToolCall,
	limit int,
	execute func(domain.ChatToolCall) domain.ToolResult,
) []domain.ToolResult {
	if limit <= 0 {
		limit = 1
	}
	if limit > 32 {
		limit = 32
	}
	results := make([]domain.ToolResult, len(calls))
	semaphore := make(chan struct{}, limit)
	var wait sync.WaitGroup
	for index, call := range calls {
		wait.Add(1)
		go func(index int, call domain.ChatToolCall) {
			defer wait.Done()
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				results[index] = domain.ToolResult{CallID: call.ID, Name: call.Name, OK: false, Error: ctx.Err().Error()}
				return
			}
			if err := ctx.Err(); err != nil {
				results[index] = domain.ToolResult{CallID: call.ID, Name: call.Name, OK: false, Error: err.Error()}
				return
			}
			results[index] = execute(call)
		}(index, call)
	}
	wait.Wait()
	return results
}

func isModelExecutionUnavailable(err error) bool {
	return isProviderExecutionUnavailable(err)
}

func deterministicAssistantFallback(userText string) string {
	return deterministicModelUnavailableFallback(userText)
}
