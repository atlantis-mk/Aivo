package app

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"aivo/core/domain"
)

func (s *Service) preCallToolCandidates(ctx context.Context, sessionID, turnID string, registry *Registry, specs []domain.ToolSpec) (map[string]string, []domain.ToolCatalogEntry) {
	activated := map[string]string{}
	for name := range s.rememberedDeferredTools(ctx, sessionID) {
		activated[name] = "pinned"
	}
	for name := range s.warmDeferredToolsForTurn(ctx, sessionID, turnID) {
		if activated[name] == "" {
			activated[name] = "warm"
		}
	}
	if registry == nil {
		return activated, nil
	}
	allowed := map[string]bool{}
	for _, spec := range specs {
		allowed[spec.Name] = true
	}
	candidates := make([]domain.ToolCatalogEntry, 0)
	for _, entry := range deferrableCatalogEntries(registry) {
		if !allowed[entry.Name] {
			continue
		}
		if entry.ActivationPolicy == "default" && activated[entry.Name] == "" {
			activated[entry.Name] = "mode"
			continue
		}
		if firstNonEmpty(entry.ActivationPolicy, "auto") == "auto" && activated[entry.Name] == "" {
			candidates = append(candidates, entry)
		}
	}
	return activated, candidates
}

func (s *Service) warmDeferredToolsForTurn(ctx context.Context, sessionID, turnID string) map[string]bool {
	result := map[string]bool{}
	if s == nil || s.store == nil || strings.TrimSpace(sessionID) == "" {
		return result
	}
	state, err := s.store.GetSessionExecutionState(ctx, sessionID)
	if err != nil {
		return result
	}
	leases := intMapFromAny(state.Metadata[sessionMetadataWarmDeferredTools])
	lastTurn, _ := state.Metadata[sessionMetadataWarmDeferredTurn].(string)
	if strings.TrimSpace(turnID) != "" && lastTurn != turnID {
		for name, remaining := range leases {
			remaining--
			if remaining <= 0 {
				delete(leases, name)
			} else {
				leases[name] = remaining
			}
		}
		if state.Metadata == nil {
			state.Metadata = map[string]any{}
		}
		state.Metadata[sessionMetadataWarmDeferredTools] = leases
		state.Metadata[sessionMetadataWarmDeferredTurn] = turnID
		_, _ = s.store.UpsertSessionExecutionState(ctx, state)
	}
	for name, remaining := range leases {
		if remaining > 0 {
			result[name] = true
		}
	}
	return result
}

func (s *Service) rememberWarmDeferredTool(ctx context.Context, sessionID, toolName string) error {
	toolName = strings.TrimSpace(toolName)
	if s == nil || s.store == nil || sessionID == "" || toolName == "" || isReservedCoreToolName(toolName) || isBridgeToolName(toolName) {
		return nil
	}
	state, err := s.store.GetSessionExecutionState(ctx, sessionID)
	if err != nil {
		return err
	}
	if state.Metadata == nil {
		state.Metadata = map[string]any{}
	}
	leases := intMapFromAny(state.Metadata[sessionMetadataWarmDeferredTools])
	leases[toolName] = 3
	order := stringSliceFromAny(state.Metadata[sessionMetadataWarmDeferredOrder])
	next := make([]string, 0, len(order)+1)
	for _, name := range order {
		if name != toolName && leases[name] > 0 {
			next = append(next, name)
		}
	}
	next = append(next, toolName)
	for len(next) > 8 {
		delete(leases, next[0])
		next = next[1:]
	}
	state.Metadata[sessionMetadataWarmDeferredTools] = leases
	state.Metadata[sessionMetadataWarmDeferredOrder] = next
	_, err = s.store.UpsertSessionExecutionState(ctx, state)
	return err
}

func intMapFromAny(value any) map[string]int {
	out := map[string]int{}
	if typed, ok := value.(map[string]int); ok {
		for key, item := range typed {
			out[key] = item
		}
		return out
	}
	if typed, ok := value.(map[string]any); ok {
		for key, item := range typed {
			switch number := item.(type) {
			case float64:
				out[key] = int(number)
			case int:
				out[key] = number
			case json.Number:
				parsed, _ := number.Int64()
				out[key] = int(parsed)
			}
		}
	}
	return out
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
		if name != "" && !isReservedCoreToolName(name) && !isBridgeToolName(name) {
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
	coreNames := make([]string, 0, len(coreToolNames()))
	for _, name := range coreToolNames() {
		if !s.disabledCoreTools(ctx, sessionID)[name] {
			coreNames = append(coreNames, name)
		}
	}
	return domain.SessionActiveToolsResult{SessionID: sessionID, ToolNames: names, CoreToolNames: coreNames}, nil
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
	selected := map[string]bool{}
	for _, name := range input.ToolNames {
		selected[strings.TrimSpace(name)] = true
	}
	disabledCore := make([]string, 0, len(coreToolNames()))
	for _, name := range coreToolNames() {
		if !selected[name] {
			disabledCore = append(disabledCore, name)
		}
	}
	if state.Metadata == nil {
		state.Metadata = map[string]any{}
	}
	state.Metadata[sessionMetadataRememberedDeferredTools] = names
	state.Metadata[sessionMetadataDisabledCoreTools] = disabledCore
	if _, err := s.store.UpsertSessionExecutionState(ctx, state); err != nil {
		return domain.SessionActiveToolsResult{}, err
	}
	if s.onSessionUpdated != nil {
		s.onSessionUpdated(sessionID, nil)
	}
	return s.GetSessionActiveTools(ctx, sessionID)
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
		if name != "" && !isReservedCoreToolName(name) && !isBridgeToolName(name) {
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
		if name == "" || isReservedCoreToolName(name) || isBridgeToolName(name) || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (s *Service) disabledCoreTools(ctx context.Context, sessionID string) map[string]bool {
	disabled := map[string]bool{}
	if s == nil || s.store == nil || strings.TrimSpace(sessionID) == "" {
		return disabled
	}
	state, err := s.store.GetSessionExecutionState(ctx, sessionID)
	if err != nil || state.Metadata == nil {
		return disabled
	}
	for _, name := range stringSliceFromAny(state.Metadata[sessionMetadataDisabledCoreTools]) {
		name = strings.TrimSpace(name)
		if isReservedCoreToolName(name) {
			disabled[name] = true
		}
	}
	return disabled
}

func coreToolNames() []string {
	return []string{"read", "bash", "edit", "write"}
}
