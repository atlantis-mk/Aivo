package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"aivo/core/domain"
)

func (s *Service) SaveToolCall(ctx context.Context, input domain.CreateToolCallRequest) (domain.ToolCall, error) {
	status, err := domain.NormalizeToolCallStatus(input.Status)
	if err != nil {
		return domain.ToolCall{}, err
	}
	if strings.TrimSpace(input.SessionID) == "" || strings.TrimSpace(input.Name) == "" {
		return domain.ToolCall{}, errors.New("sessionId and name are required")
	}
	now := domain.NowString(s.now())
	visibility := domain.EventVisibilityInternal
	if status == domain.ToolCallStatusFailed {
		visibility = domain.EventVisibilityNormal
	}
	event, err := s.AppendEvent(ctx, domain.AppendEventRequest{SessionID: input.SessionID, TurnID: input.TurnID, Type: domain.EventTypeToolCall, Role: domain.EventRoleTool, Visibility: visibility, Content: input.ResultSummary, Payload: map[string]any{"name": input.Name}})
	if err != nil {
		return domain.ToolCall{}, err
	}
	callID := uuid.NewString()
	if strings.TrimSpace(input.ID) != "" {
		callID = strings.TrimSpace(input.ID)
	}
	call := domain.ToolCall{ID: callID, SessionID: input.SessionID, TurnID: input.TurnID, EventID: event.ID, Name: input.Name, Arguments: input.Arguments, Status: status, ResultSummary: bounded(input.ResultSummary, 2000), Result: input.Result, Error: input.Error, TimeCreated: now, TimeUpdated: now}
	if err := s.store.SaveToolCall(ctx, call); err != nil {
		return domain.ToolCall{}, err
	}
	if s.onToolCallUpdated != nil {
		s.onToolCallUpdated(call.SessionID, call.TurnID, call, status == domain.ToolCallStatusRunning)
	}
	return call, nil
}

func (s *Service) ListToolCalls(ctx context.Context, sessionID string) ([]domain.ToolCall, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("sessionId is required")
	}
	return s.store.ListToolCalls(ctx, sessionID)
}

func (s *Service) ReplaySessionToolCall(ctx context.Context, input domain.ReplaySessionToolCallRequest) (domain.ToolCall, error) {
	toolCallID := strings.TrimSpace(input.ToolCallID)
	if toolCallID == "" {
		return domain.ToolCall{}, errors.New("toolCallId is required")
	}
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return domain.ToolCall{}, errors.New("sessionId is required")
	}
	toolCalls, err := s.store.ListToolCalls(ctx, sessionID)
	if err != nil {
		return domain.ToolCall{}, err
	}
	var original domain.ToolCall
	for _, call := range toolCalls {
		if call.ID == toolCallID {
			original = call
			break
		}
	}
	if original.ID == "" {
		return domain.ToolCall{}, errors.New("tool call not found")
	}
	if original.SessionID != sessionID {
		return domain.ToolCall{}, errors.New("tool call does not belong to session")
	}
	session, err := s.store.GetRuntimeSession(ctx, sessionID)
	if err != nil {
		return domain.ToolCall{}, err
	}
	cc, _ := s.store.GetCodingContext(ctx, sessionID)
	workspaceRoot := strings.TrimSpace(cc.ProjectPath)
	if workspaceRoot == "" {
		workspaceRoot = strings.TrimSpace(session.ProjectPath)
	}
	if workspaceRoot == "" {
		return domain.ToolCall{}, errors.New("tool replay requires a workspace root")
	}
	var turn domain.Turn
	if strings.TrimSpace(original.TurnID) != "" {
		turn, err = s.store.GetTurn(ctx, original.TurnID)
		if err != nil {
			return domain.ToolCall{}, err
		}
		if turn.SessionID != sessionID {
			return domain.ToolCall{}, errors.New("tool call turn does not belong to session")
		}
	}
	registry, runtime := s.toolsForWorkspace(workspaceRoot)
	if runtime == nil || registry == nil {
		return domain.ToolCall{}, errors.New("tool runtime unavailable for workspace")
	}
	replayID := "replay_" + uuid.NewString()
	rawArgs, err := replayToolCallArguments(original)
	if err != nil {
		return domain.ToolCall{}, err
	}
	replayCall := domain.ChatToolCall{ID: replayID, Name: original.Name, Arguments: rawArgs}
	identity, _ := registry.IdentityFor(replayCall.Name)
	if err := s.recordToolCallStarted(ctx, sessionID, original.TurnID, replayCall, identity); err != nil {
		return domain.ToolCall{}, err
	}
	mode := firstNonEmpty(turn.AgentMode, session.AgentMode, domain.AgentModeAssistant)
	modeDef, err := s.resolveAgentModeForRequest(ctx, sessionID, mode)
	if err != nil {
		return domain.ToolCall{}, err
	}
	allowedToolsets := allowedToolsetsForRun(modeDef, domain.SubmitSessionMessageRequest{})
	specs := visibleToolSpecsForMode(modeDef.ID, registry.SpecsForToolsets(allowedToolsets))
	specs = configureAgentDelegateToolSpecs(modeDef, specs)
	assembly := AssembleToolSpecsWithSources(registry, specs, map[string]string{replayCall.Name: "replay"})
	result := runtime.ExecuteWithContext(ctx, replayCall, domain.ToolExecutionContext{
		WorkspaceRoot:         workspaceRoot,
		SessionID:             sessionID,
		TurnID:                original.TurnID,
		ToolCallID:            replayID,
		AgentMode:             modeDef.ID,
		AllowedToolsets:       allowedToolsets,
		PermissionScope:       firstNonEmpty(input.PermissionScope, defaultPermissionScopeForMode(modeDef.ID)),
		ExpectedRegistrations: assembly.ExpectedRegistrations,
		ToolSnapshot:          &assembly.Snapshot,
	})
	if err := s.recordToolResultWithMetadata(ctx, sessionID, original.TurnID, replayCall, result, map[string]any{
		"replayOfToolCallId": original.ID,
		"replayOfToolName":   original.Name,
	}); err != nil {
		return domain.ToolCall{}, err
	}
	_ = s.appendToolReplayEvent(ctx, sessionID, original.TurnID, original, replayID, result)
	toolCalls, err = s.store.ListToolCalls(ctx, sessionID)
	if err != nil {
		return domain.ToolCall{}, err
	}
	for _, call := range toolCalls {
		if call.ID == replayID {
			return call, nil
		}
	}
	return domain.ToolCall{}, errors.New("replayed tool call was not saved")
}

func replayToolCallArguments(call domain.ToolCall) (json.RawMessage, error) {
	if call.Arguments == nil {
		return json.RawMessage(`{}`), nil
	}
	raw, err := json.Marshal(call.Arguments)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func (s *Service) appendToolReplayEvent(ctx context.Context, sessionID string, turnID string, original domain.ToolCall, replayID string, result domain.ToolResult) error {
	status := "failed"
	if result.OK {
		status = "succeeded"
	}
	if result.PermissionRequested {
		status = "waiting for permission"
	}
	content := fmt.Sprintf("Tool call replay %s: %s from %s", status, firstNonEmpty(result.Name, original.Name), original.ID)
	_, err := s.AppendEvent(ctx, domain.AppendEventRequest{
		SessionID:  sessionID,
		TurnID:     turnID,
		Type:       domain.EventTypeSystemNote,
		Role:       domain.EventRoleSystem,
		Visibility: domain.EventVisibilityNormal,
		Content:    content,
		Payload: map[string]any{
			"kind":               "tool_call_replay",
			"status":             status,
			"originalToolCallId": original.ID,
			"replayToolCallId":   replayID,
			"toolName":           firstNonEmpty(result.Name, original.Name),
		},
	})
	return err
}
