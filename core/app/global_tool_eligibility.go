package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"aivo/core/domain"
)

type globalToolPreferenceStore interface {
	ListGloballyDisabledToolNames(context.Context) ([]string, error)
	SetGlobalToolEnabled(context.Context, string, bool) error
}

func (s *Service) globallyDisabledToolNames(ctx context.Context) (map[string]bool, error) {
	store, ok := s.store.(globalToolPreferenceStore)
	if !ok {
		return map[string]bool{}, nil
	}
	names, err := store.ListGloballyDisabledToolNames(ctx)
	if err != nil {
		return nil, err
	}
	disabled := make(map[string]bool, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" && !isReservedCoreToolName(name) {
			disabled[name] = true
		}
	}
	return disabled, nil
}

func (s *Service) applyGlobalToolEligibility(ctx context.Context, entries []domain.ToolCatalogEntry) ([]domain.ToolCatalogEntry, error) {
	disabled, err := s.globallyDisabledToolNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("load global tool preferences: %w", err)
	}
	out := append([]domain.ToolCatalogEntry(nil), entries...)
	for index := range out {
		if disabled[out[index].Name] {
			out[index].Enabled = false
		}
	}
	return out, nil
}

func (s *Service) filterGloballyVisibleToolCatalogEntries(ctx context.Context, entries []domain.ToolCatalogEntry) ([]domain.ToolCatalogEntry, error) {
	disabled, err := s.globallyDisabledToolNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("load global tool preferences: %w", err)
	}
	if len(disabled) == 0 {
		return entries, nil
	}
	out := make([]domain.ToolCatalogEntry, 0, len(entries))
	for _, entry := range entries {
		if !disabled[entry.Name] {
			out = append(out, entry)
		}
	}
	return out, nil
}

func (s *Service) SetGlobalToolEnabled(ctx context.Context, input domain.GlobalToolEnabledInput) (domain.ToolCatalogEntry, error) {
	name := strings.TrimSpace(input.Name)
	if err := validateCanonicalToolName(name); err != nil {
		return domain.ToolCatalogEntry{}, err
	}
	if isReservedCoreToolName(name) && !input.Enabled {
		return domain.ToolCatalogEntry{}, errors.New("required core tools cannot be disabled")
	}
	entries, err := s.ListToolCatalog(ctx, domain.ToolCatalogInput{WorkspaceRoot: strings.TrimSpace(input.WorkspaceRoot)})
	if err != nil {
		return domain.ToolCatalogEntry{}, err
	}
	var selected domain.ToolCatalogEntry
	found := false
	for _, entry := range entries {
		if entry.Name == name && !isBridgeToolName(entry.Name) {
			selected = entry
			found = true
			break
		}
	}
	if !found {
		return domain.ToolCatalogEntry{}, errors.New("tool is not available in the global catalog")
	}
	store, ok := s.store.(globalToolPreferenceStore)
	if !ok {
		return domain.ToolCatalogEntry{}, errors.New("global tool preferences are unavailable")
	}
	if err := store.SetGlobalToolEnabled(ctx, name, input.Enabled); err != nil {
		return domain.ToolCatalogEntry{}, err
	}
	selected.Enabled = input.Enabled
	return selected, nil
}
