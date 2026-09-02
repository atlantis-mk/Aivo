package app

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"aivo/core/domain"
)

func (s *Service) runAssistantAgentLoop(
	ctx context.Context,
	input domain.SubmitSessionMessageRequest,
	history []domain.ChatMessage,
	turn domain.Turn,
	reasoningEffort string,
	serviceTier string,
	runtimeMetrics *sessionRuntimeMetrics,
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
		} else if strings.TrimSpace(session.ProjectPath) == "" && strings.TrimSpace(cc.ProjectPath) != "" {
			projectPath, workspaceErr := s.ensureUnscopedWorkspace(ctx, cc.ProjectPath)
			if workspaceErr != nil {
				return "", nil, workspaceErr
			}
			if projectPath != cc.ProjectPath {
				cc, _ = s.CreateOrUpdateCodingContext(ctx, input.SessionID, projectPath)
			}
		}
	}
	var registry *Registry
	var runtime *ToolRuntime
	ctx = withProviderRegistry(ctx, s.providerRegistryForProject(strings.TrimSpace(cc.ProjectPath)))
	allowedToolsets := allowedToolsetsForRun(modeDef, input)
	requestedModel := s.modelForAgentMode(ctx, modeDef, input.Model)
	promptSnapshot := PromptSnapshot{}
	if s.prompts != nil {
		promptSnapshot = s.prompts.Snapshot()
	}
	messages := prependAgentSystemPromptWithSnapshotAndShell(history, modeDef, promptSnapshot, shellRuntimeInstruction(strings.TrimSpace(cc.ProjectPath)))
	var model *domain.ModelRef
	for {
		failedSources := s.prepareEnabledToolCatalogs(ctx)
		registry, runtime = s.toolsForWorkspace(strings.TrimSpace(cc.ProjectPath))
		var specs []domain.ToolSpec
		expectedRegistrations := map[string]domain.ToolRegistrationIdentity{}
		var toolSnapshot domain.ToolSnapshot
		if registry != nil {
			specs = visibleToolSpecsForMode(modeDef.ID, registry.SpecsForToolsets(allowedToolsets))
			specs = configureAgentDelegateToolSpecs(modeDef, specs)
			specs = filterEligibleToolSpecs(registry, specs, failedSources)
		}
		resolved, err := s.resolveHostResources(ctx, hostResourceResolveRequest{
			SessionID:     input.SessionID,
			TurnID:        turn.ID,
			Intent:        input.Text,
			AgentMode:     modeDef.ID,
			WorkspaceRoot: strings.TrimSpace(cc.ProjectPath),
			Registry:      registry,
			Specs:         specs,
			Required:      true,
			SkipInstructionSelection: hasSessionResourceKind(input.ResourceReferences, domain.SessionResourceSkill) ||
				hasSessionResourceKind(input.ResourceReferences, domain.SessionResourceExtension),
		})
		if err != nil {
			return "", model, err
		}
		explicitResourceContext := renderSessionResourceReferenceContext(input.ResourceReferences)
		if explicitResourceContext != "" {
			resolved.Context = strings.TrimSpace(explicitResourceContext + "\n\n" + resolved.Context)
		}
		if registry != nil {
			for name, source := range s.providerDeclaredLocalToolActivations(ctx, requestedModel) {
				if resolved.ToolActivations[name] == "" {
					resolved.ToolActivations[name] = source
				}
			}
			if len(modeDef.Subagents) > 0 {
				resolved.ToolActivations["agent_delegate_task"] = "modeAssociation"
			}
			for name := range s.disabledCoreTools(ctx, input.SessionID) {
				resolved.ToolActivations[name] = "disabled"
			}
			assembly := AssembleToolSpecsWithSources(registry, specs, resolved.ToolActivations)
			specs = assembly.Specs
			expectedRegistrations = assembly.ExpectedRegistrations
			toolSnapshot = assembly.Snapshot
		}
		requestMessages := appendHostPreSnapshotContext(messages, resolved.Context)
		requestStartedAt := time.Now()
		var firstTokenAt time.Time
		markFirstToken := func() {
			if firstTokenAt.IsZero() {
				firstTokenAt = time.Now()
			}
		}
		resp, activeModel, err := s.GenerateChatResponseStreamWithToolDelta(ctx, domain.ChatRequest{Messages: requestMessages, Tools: specs, Temperature: modeDef.Temperature, TopP: modeDef.TopP, Options: modeDef.Options}, requestedModel, reasoningEffort, serviceTier, func(delta string) {
			if delta != "" {
				markFirstToken()
			}
			if onDelta != nil {
				onDelta(delta)
			}
		}, func(call domain.ChatToolCall) {
			if call.ID != "" || call.Name != "" || len(call.Arguments) > 0 {
				markFirstToken()
			}
		})
		requestCompletedAt := time.Now()
		if err != nil {
			return "", activeModel, err
		}
		runtimeMetrics.recordSuccessfulStep(requestStartedAt, firstTokenAt, requestCompletedAt, resp.Usage)
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
				_ = s.recordToolCallStarted(ctx, input.SessionID, turn.ID, call, expectedRegistrations[call.Name])
			}
			limit := loadEffectiveRuntimeConfig(strings.TrimSpace(cc.ProjectPath)).Config.MaxParallelChildren
			results := executeBoundedParallelToolCalls(ctx, resp.ToolCalls, limit, func(call domain.ChatToolCall) domain.ToolResult {
				return runtime.ExecuteWithContext(ctx, call, domain.ToolExecutionContext{
					WorkspaceRoot: strings.TrimSpace(cc.ProjectPath), SessionID: input.SessionID, TurnID: turn.ID,
					ActiveModel: model,
					AgentMode:   modeDef.ID, AllowedToolsets: allowedToolsets,
					PermissionScope:       firstNonEmpty(input.PermissionScope, permissionScopeForAgent(modeDef)),
					ExpectedRegistrations: expectedRegistrations,
					ToolSnapshot:          &toolSnapshot,
					RecentImages:          recentImageAttachments(messages, 5),
				})
			})
			for index, call := range resp.ToolCalls {
				result := results[index]
				_ = s.recordToolResult(ctx, input.SessionID, turn.ID, call, result)
				messages = appendToolResultMessages(messages, call, result)
			}
			continue
		}
		for _, call := range resp.ToolCalls {
			_ = s.recordToolCallStarted(ctx, input.SessionID, turn.ID, call, expectedRegistrations[call.Name])
			var result domain.ToolResult
			if runtime == nil {
				result = domain.ToolResult{CallID: call.ID, Name: call.Name, OK: false, Error: "tool runtime unavailable: this session has no workspace root"}
			} else {
				result = runtime.ExecuteWithContext(ctx, call, domain.ToolExecutionContext{
					WorkspaceRoot:         strings.TrimSpace(cc.ProjectPath),
					ActiveModel:           model,
					SessionID:             input.SessionID,
					TurnID:                turn.ID,
					AgentMode:             modeDef.ID,
					AllowedToolsets:       allowedToolsets,
					PermissionScope:       firstNonEmpty(input.PermissionScope, permissionScopeForAgent(modeDef)),
					ExpectedRegistrations: expectedRegistrations,
					ToolSnapshot:          &toolSnapshot,
					RecentImages:          recentImageAttachments(messages, 5),
				})
			}
			_ = s.recordToolResult(ctx, input.SessionID, turn.ID, call, result)
			if result.PermissionRequested {
				return "等待你批准工具权限后，我可以继续执行这次修改。", model, nil
			}
			messages = appendToolResultMessages(messages, call, result)
			if call.Name == projectAssociateToolName && result.OK {
				cc, _ = s.store.GetCodingContext(ctx, input.SessionID)
				ctx = withProviderRegistry(ctx, s.providerRegistryForProject(strings.TrimSpace(cc.ProjectPath)))
			}
		}
	}
}

func recentImageAttachments(messages []domain.ChatMessage, limit int) []domain.MessageAttachment {
	if limit <= 0 {
		return nil
	}
	out := make([]domain.MessageAttachment, 0, limit)
	for i := len(messages) - 1; i >= 0 && len(out) < limit; i-- {
		attachments := messages[i].Attachments
		for j := len(attachments) - 1; j >= 0 && len(out) < limit; j-- {
			attachment := attachments[j]
			if strings.EqualFold(strings.TrimSpace(attachment.Kind), "image") && isImageAttachmentMIME(attachment.MIMEType) && strings.TrimSpace(attachment.Data) != "" {
				out = append(out, attachment)
			}
		}
	}
	for left, right := 0, len(out)-1; left < right; left, right = left+1, right-1 {
		out[left], out[right] = out[right], out[left]
	}
	return out
}

func normalizeHostResourceSelectionResources(resources []hostResourceSelectionResource) []hostResourceSelectionResource {
	seen := map[string]bool{}
	out := make([]hostResourceSelectionResource, 0, len(resources))
	for _, resource := range resources {
		resource.Kind = strings.TrimSpace(resource.Kind)
		resource.ID = strings.TrimSpace(resource.ID)
		resource.Name = strings.TrimSpace(resource.Name)
		key := resource.Kind + "\x00" + resource.ID
		if (resource.Kind != domain.SessionResourceExtension && resource.Kind != domain.SessionResourceMCP && resource.Kind != domain.SessionResourceTool && resource.Kind != domain.SessionResourceSkill) || resource.ID == "" || resource.ToolCount < 0 || seen[key] {
			continue
		}
		if resource.Kind == domain.SessionResourceTool && resource.ToolCount == 0 {
			continue
		}
		if resource.Name == "" {
			resource.Name = resource.ID
		}
		seen[key] = true
		out = append(out, resource)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind == out[j].Kind {
			return out[i].ID < out[j].ID
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}

func hostResourceSelectionPayload(resources []hostResourceSelectionResource) []map[string]any {
	payload := make([]map[string]any, 0, len(resources))
	for _, resource := range resources {
		payload = append(payload, map[string]any{
			"kind": resource.Kind, "id": resource.ID, "name": resource.Name, "toolCount": resource.ToolCount,
		})
	}
	return payload
}

func appendToolResultMessages(messages []domain.ChatMessage, call domain.ChatToolCall, result domain.ToolResult) []domain.ChatMessage {
	messages = append(messages, domain.ChatMessage{Role: "tool", Text: encodeToolResultForModel(result), ToolCallID: call.ID, Name: call.Name})
	if len(result.ModelAttachments) > 0 {
		messages = append(messages, domain.ChatMessage{Role: "user", Text: "Image content returned by tool " + call.Name + ".", Attachments: result.ModelAttachments})
	}
	return messages
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
