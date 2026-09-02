package app

import (
	"context"
	"errors"
	"sort"
	"strings"

	"aivo/core/domain"
)

func (s *Service) snapshotToolCandidates(ctx context.Context, sessionID, turnID string, registry *Registry, specs []domain.ToolSpec) (map[string]string, []domain.ToolCatalogEntry) {
	_ = turnID
	activated := map[string]string{}
	for name := range s.rememberedDeferredTools(ctx, sessionID) {
		activated[name] = "manual"
	}
	automatic, initialized := s.autoSelectedTools(ctx, sessionID)
	for name := range automatic {
		if activated[name] == "" {
			activated[name] = "automatic"
		}
	}
	if registry == nil || initialized {
		return activated, nil
	}
	allowed := map[string]bool{}
	for _, spec := range specs {
		allowed[spec.Name] = true
	}
	candidates := make([]domain.ToolCatalogEntry, 0)
	for _, entry := range deferrableCatalogEntries(registry) {
		if !allowed[entry.Name] || activated[entry.Name] != "" {
			continue
		}
		policy := firstNonEmpty(entry.ActivationPolicy, "auto")
		if policy == "auto" || policy == "default" {
			candidates = append(candidates, entry)
		}
	}
	return activated, candidates
}

func (s *Service) autoSelectedTools(ctx context.Context, sessionID string) (map[string]bool, bool) {
	selected := map[string]bool{}
	if s == nil || s.store == nil || strings.TrimSpace(sessionID) == "" {
		return selected, false
	}
	state, err := s.store.GetSessionExecutionState(ctx, sessionID)
	if err != nil || state.Metadata == nil {
		return selected, false
	}
	for _, name := range stringSliceFromAny(state.Metadata[sessionMetadataAutoSelectedTools]) {
		name = strings.TrimSpace(name)
		if name != "" && !isReservedCoreToolName(name) && !isBridgeToolName(name) {
			selected[name] = true
		}
	}
	initialized, _ := state.Metadata[sessionMetadataAutoToolsInitialized].(bool)
	return selected, initialized
}

func (s *Service) replaceAutoSelectedTools(ctx context.Context, sessionID string, toolNames []string) error {
	sessionID = strings.TrimSpace(sessionID)
	if s == nil || s.store == nil || sessionID == "" {
		return errors.New("sessionId is required")
	}
	state, err := s.store.GetSessionExecutionState(ctx, sessionID)
	if err != nil {
		return err
	}
	if state.Metadata == nil {
		state.Metadata = map[string]any{}
	}
	state.Metadata[sessionMetadataAutoSelectedTools] = normalizeDeferredToolNames(toolNames)
	state.Metadata[sessionMetadataAutoToolsInitialized] = true
	_, err = s.store.UpsertSessionExecutionState(ctx, state)
	return err
}

func markAutoSelectedToolsInitialized(metadata map[string]any) {
	if metadata == nil {
		return
	}
	if _, ok := metadata[sessionMetadataAutoSelectedTools]; !ok {
		metadata[sessionMetadataAutoSelectedTools] = []string{}
	}
	metadata[sessionMetadataAutoToolsInitialized] = true
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
	automatic, _ := s.autoSelectedTools(ctx, sessionID)
	automaticNames := make([]string, 0, len(automatic))
	for name := range automatic {
		automaticNames = append(automaticNames, name)
	}
	sort.Strings(automaticNames)
	coreNames := coreToolNames()
	return domain.SessionActiveToolsResult{
		SessionID:          sessionID,
		ToolNames:          names,
		CoreToolNames:      coreNames,
		AutomaticToolNames: automaticNames,
	}, nil
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
	current := s.rememberedDeferredTools(ctx, sessionID)
	disabled, err := s.globallyDisabledToolNames(ctx)
	if err != nil {
		return domain.SessionActiveToolsResult{}, err
	}
	for _, name := range names {
		if disabled[name] && !current[name] {
			return domain.SessionActiveToolsResult{}, errors.New("tool is hidden from new conversation activation: " + name)
		}
	}
	if state.Metadata == nil {
		state.Metadata = map[string]any{}
	}
	state.Metadata[sessionMetadataRememberedDeferredTools] = names
	state.Metadata[sessionMetadataDisabledCoreTools] = []string{}
	markAutoSelectedToolsInitialized(state.Metadata)
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
	_ = ctx
	_ = sessionID
	return map[string]bool{}
}

func coreToolNames() []string {
	return []string{"read", ExecCommandToolName, WriteStdinToolName, "edit", "write", "update_plan", "ask_user"}
}
