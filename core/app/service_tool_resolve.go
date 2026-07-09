package app

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"aivo/core/domain"
)

func (s *Service) resolveToolsWithAuxiliaryModel(ctx context.Context, request ToolResolveRequest) (ToolResolveDecision, error) {
	if len(request.Candidates) == 0 {
		return ToolResolveDecision{}, nil
	}
	maxTools := request.MaxTools
	if maxTools <= 0 {
		maxTools = 8
	}
	catalog := make([]map[string]any, 0, len(request.Candidates))
	for _, entry := range request.Candidates {
		catalog = append(catalog, map[string]any{
			"name":        entry.Name,
			"description": bounded(entry.Description, 400),
			"source":      entry.Source,
			"sourceId":    entry.SourceID,
			"category":    entry.Category,
			"namespace":   entry.Namespace,
			"capability":  entry.Capability,
			"riskLevel":   entry.RiskLevel,
		})
	}
	payload := map[string]any{
		"intent":    request.Intent,
		"maxTools":  maxTools,
		"agentMode": request.AgentMode,
		"catalog":   catalog,
	}
	rawPayload, _ := json.MarshalIndent(payload, "", "  ")
	messages := []domain.ChatMessage{
		{Role: "system", Text: "Select tools only from the provided catalog for the requested missing capability. Return strict JSON: {\"tools\":[\"exact_tool_name\"],\"reason\":\"short reason\"}. Select only clear matches. Do not invent names, infer hidden tools, or choose adjacent tools. If uncertain or no clear match exists, return {\"tools\":[],\"reason\":\"no matching allowed tool\"}."},
		{Role: "user", Text: string(rawPayload)},
	}
	models := s.resolveAuxiliaryModels(ctx, nil)
	var lastErr error
	for _, model := range models {
		reply, _, err := s.GenerateChatReply(ctx, messages, &model, "low", "default")
		if err != nil {
			lastErr = err
			continue
		}
		decision, err := parseToolResolveDecision(reply)
		if err != nil {
			lastErr = err
			continue
		}
		if len(decision.Names) > maxTools {
			decision.Names = decision.Names[:maxTools]
		}
		return decision, nil
	}
	if lastErr != nil {
		return ToolResolveDecision{}, lastErr
	}
	return ToolResolveDecision{}, errors.New("auxiliary model is not configured")
}

func parseToolResolveDecision(raw string) (ToolResolveDecision, error) {
	text := strings.TrimSpace(stripThinkBlocks(raw))
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end >= start {
		text = text[start : end+1]
	}
	var decoded struct {
		Tools  []string `json:"tools"`
		Names  []string `json:"names"`
		Reason string   `json:"reason"`
	}
	if err := json.Unmarshal([]byte(text), &decoded); err != nil {
		return ToolResolveDecision{}, err
	}
	names := decoded.Tools
	if len(names) == 0 {
		names = decoded.Names
	}
	out := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" {
			out = append(out, name)
		}
	}
	return ToolResolveDecision{Names: out, Reason: strings.TrimSpace(decoded.Reason)}, nil
}

func (s *Service) rememberedDeferredTools(ctx context.Context, sessionID string) map[string]bool {
	remembered := map[string]bool{}
	if s == nil || s.store == nil || strings.TrimSpace(sessionID) == "" {
		return remembered
	}
	state, err := s.store.GetSessionExecutionState(ctx, sessionID)
	if err != nil || state.Metadata == nil {
		return remembered
	}
	for _, name := range stringSliceFromAny(state.Metadata[sessionMetadataRememberedDeferredTools]) {
		name = strings.TrimSpace(name)
		if name != "" && !isBridgeToolName(name) {
			remembered[name] = true
		}
	}
	return remembered
}

func (s *Service) GetSessionActiveTools(ctx context.Context, sessionID string) (domain.SessionActiveToolsResult, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return domain.SessionActiveToolsResult{}, errors.New("sessionId is required")
	}
	remembered := s.rememberedDeferredTools(ctx, sessionID)
	names := make([]string, 0, len(remembered))
	for name := range remembered {
		names = append(names, name)
	}
	sort.Strings(names)
	return domain.SessionActiveToolsResult{SessionID: sessionID, ToolNames: names}, nil
}

func (s *Service) SetSessionActiveTools(ctx context.Context, input domain.SessionActiveToolsInput) (domain.SessionActiveToolsResult, error) {
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return domain.SessionActiveToolsResult{}, errors.New("sessionId is required")
	}
	state, err := s.store.GetSessionExecutionState(ctx, sessionID)
	if err != nil {
		return domain.SessionActiveToolsResult{}, err
	}
	names := normalizeDeferredToolNames(input.ToolNames)
	if state.Metadata == nil {
		state.Metadata = map[string]any{}
	}
	state.Metadata[sessionMetadataRememberedDeferredTools] = names
	if _, err := s.store.UpsertSessionExecutionState(ctx, state); err != nil {
		return domain.SessionActiveToolsResult{}, err
	}
	if s.onSessionUpdated != nil {
		s.onSessionUpdated(sessionID, nil)
	}
	return domain.SessionActiveToolsResult{SessionID: sessionID, ToolNames: names}, nil
}

func (s *Service) rememberDeferredToolUsed(ctx context.Context, sessionID string, toolName string) error {
	toolName = strings.TrimSpace(toolName)
	if s == nil || s.store == nil || strings.TrimSpace(sessionID) == "" || toolName == "" || isBridgeToolName(toolName) {
		return nil
	}
	state, err := s.store.GetSessionExecutionState(ctx, sessionID)
	if err != nil {
		return err
	}
	remembered := map[string]bool{}
	for _, name := range stringSliceFromAny(state.Metadata[sessionMetadataRememberedDeferredTools]) {
		name = strings.TrimSpace(name)
		if name != "" && !isBridgeToolName(name) {
			remembered[name] = true
		}
	}
	if remembered[toolName] {
		return nil
	}
	remembered[toolName] = true
	names := make([]string, 0, len(remembered))
	for name := range remembered {
		names = append(names, name)
	}
	sort.Strings(names)
	if state.Metadata == nil {
		state.Metadata = map[string]any{}
	}
	state.Metadata[sessionMetadataRememberedDeferredTools] = names
	_, err = s.store.UpsertSessionExecutionState(ctx, state)
	return err
}

func normalizeDeferredToolNames(toolNames []string) []string {
	seen := map[string]bool{}
	names := make([]string, 0, len(toolNames))
	for _, name := range toolNames {
		name = strings.TrimSpace(name)
		if name == "" || isBridgeToolName(name) || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
