package app

import (
	"context"
	"errors"
	"strings"

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
	allowedToolsets := allowedToolsetsForRun(modeDef, input)
	requestedModel := s.modelForAgentMode(ctx, modeDef, input.Model)
	messages := append([]domain.ChatMessage(nil), history...)
	modePrompt := "Agent mode: " + modeDef.DisplayName + "\n\n" + modeDef.Prompt
	modePrompt += "\n\nIf a task matches an available skill, call the skill tool to load that skill before continuing. If current tools cannot perform a required action, call tool_resolve with a concise, specific missing capability. Do not use it for convenience, exploration, planning, or guessing tool names. If no allowed tool matches, stop with a local no_available_tool error."
	if len(messages) > 0 && messages[0].Role == domain.EventRoleSystem {
		messages[0].Text = messages[0].Text + "\n\n" + modePrompt
	} else {
		messages = append([]domain.ChatMessage{{Role: domain.EventRoleSystem, Text: modePrompt}}, messages...)
	}
	var model *domain.ModelRef
	for step := 0; step < defaultAgentMaxSteps; step++ {
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
		resp, activeModel, err := s.GenerateChatResponseStreamWithToolDelta(ctx, domain.ChatRequest{Messages: messages, Tools: specs}, requestedModel, reasoningEffort, serviceTier, onDelta, func(call domain.ChatToolCall) {
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
					PermissionScope:       firstNonEmpty(input.PermissionScope, defaultPermissionScopeForMode(modeDef.ID)),
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
		}
	}
	return "", model, ErrMaxStepsExceeded
}

func isModelExecutionUnavailable(err error) bool {
	return isProviderExecutionUnavailable(err)
}

func deterministicAssistantFallback(userText string) string {
	return deterministicModelUnavailableFallback(userText)
}
