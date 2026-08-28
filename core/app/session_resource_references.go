package app

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"aivo/core/domain"
)

const (
	sessionResourceReferenceLimit     = 32
	sessionResourceReferenceIDLimit   = 512
	sessionResourceReferencePathLimit = 4096
)

type preparedSessionResourceReferences struct {
	references []domain.SessionResourceReference
	project    *preparedAgentProjectAssociation
	skillIDs   []string
	toolNames  []string
}

func (s *Service) prepareSessionResourceReferences(
	ctx context.Context,
	sessionID string,
	references []domain.SessionResourceReference,
) (preparedSessionResourceReferences, error) {
	normalized, err := normalizeSessionResourceReferences(references)
	if err != nil || len(normalized) == 0 {
		return preparedSessionResourceReferences{references: normalized}, err
	}
	if _, err := s.store.GetRuntimeSession(ctx, sessionID); err != nil {
		return preparedSessionResourceReferences{}, errors.New("resource reference session was not found")
	}

	prepared := preparedSessionResourceReferences{}
	workspaceRoot := ""
	if session, sessionErr := s.store.GetRuntimeSession(ctx, sessionID); sessionErr == nil {
		workspaceRoot = strings.TrimSpace(session.ProjectPath)
	}
	if workspaceRoot == "" {
		if codingContext, contextErr := s.store.GetCodingContext(ctx, sessionID); contextErr == nil {
			workspaceRoot = strings.TrimSpace(codingContext.ProjectPath)
		}
	}

	for _, reference := range normalized {
		if reference.Kind != domain.SessionResourceProject {
			continue
		}
		project, projectErr := s.prepareAgentProjectAssociation(ctx, sessionID, reference.ID, reference.RootPath)
		if projectErr != nil {
			return preparedSessionResourceReferences{}, projectErr
		}
		prepared.project = &project
		workspaceRoot = project.project.RootPath
	}

	needsExtensions := hasSessionResourceKind(normalized, domain.SessionResourceExtension)
	needsMCP := hasSessionResourceKind(normalized, domain.SessionResourceMCP)
	needsTools := hasSessionResourceKind(normalized, domain.SessionResourceTool) || needsExtensions || needsMCP
	extensions := map[string]domain.ExtensionInstall{}
	servers := map[string]domain.MCPServerConfig{}
	if needsExtensions {
		installs, listErr := s.ListExtensionInstalls(ctx)
		if listErr != nil {
			return preparedSessionResourceReferences{}, fmt.Errorf("list extension references: %w", listErr)
		}
		for _, install := range installs {
			if install.Enabled {
				extensions[install.ID] = install
			}
		}
	}
	if needsMCP {
		items, listErr := s.ListMCPServers(ctx, domain.MCPServerListInput{IncludeDisabled: false, IncludeTools: false})
		if listErr != nil {
			return preparedSessionResourceReferences{}, fmt.Errorf("list MCP references: %w", listErr)
		}
		for _, item := range items {
			if item.Server.Enabled {
				servers[item.Server.ID] = item.Server
			}
		}
	}

	failedSources := map[string]bool{}
	toolEntries := map[string]domain.ToolCatalogEntry{}
	if needsTools {
		failedSources = s.prepareEnabledToolCatalogs(ctx)
		catalog, listErr := s.ListToolCatalog(ctx, domain.ToolCatalogInput{WorkspaceRoot: workspaceRoot})
		if listErr != nil {
			return preparedSessionResourceReferences{}, fmt.Errorf("list tool references: %w", listErr)
		}
		for _, entry := range catalog {
			if !entry.Enabled || failedSources[toolSourceEligibilityKey(entry.Source, entry.SourceID)] {
				continue
			}
			toolEntries[entry.Name] = entry
		}
	}

	toolNames := map[string]bool{}
	skillIDs := map[string]bool{}
	canonical := make([]domain.SessionResourceReference, 0, len(normalized))
	for _, reference := range normalized {
		switch reference.Kind {
		case domain.SessionResourceProject:
			canonical = append(canonical, domain.SessionResourceReference{
				Kind: reference.Kind,
				ID:   prepared.project.project.ID,
				Name: prepared.project.project.Name,
			})
		case domain.SessionResourceSkill:
			skill, resolveErr := s.ensureSkillManager().Resolve(ctx, reference.ID, "", "")
			if resolveErr != nil || !skill.Enabled {
				return preparedSessionResourceReferences{}, fmt.Errorf("skill reference %q is unavailable", reference.ID)
			}
			skillIDs[skill.ID] = true
			canonical = append(canonical, domain.SessionResourceReference{Kind: reference.Kind, ID: skill.ID, Name: skill.Name})
		case domain.SessionResourceTool:
			entry, ok := toolEntries[reference.ID]
			if ok && isStandaloneToolCatalogEntry(entry) {
				toolNames[entry.Name] = true
				canonical = append(canonical, domain.SessionResourceReference{Kind: reference.Kind, ID: entry.Name, Name: entry.Name})
				break
			}
			groupNames, groupName := sessionResourceSelectionGroupToolNames(toolEntries, reference.ID, domain.SessionResourceTool)
			if len(groupNames) == 0 {
				return preparedSessionResourceReferences{}, fmt.Errorf("tool reference %q is unavailable", reference.ID)
			}
			for _, name := range groupNames {
				toolNames[name] = true
			}
			canonical = append(canonical, domain.SessionResourceReference{Kind: reference.Kind, ID: reference.ID, Name: groupName})
		case domain.SessionResourceExtension:
			install, ok := extensions[reference.ID]
			if !ok {
				return preparedSessionResourceReferences{}, fmt.Errorf("extension reference %q is unavailable", reference.ID)
			}
			for _, name := range sessionResourceSourceToolNames(toolEntries, domain.ToolSourceExtension, install.ID) {
				toolNames[name] = true
			}
			name := firstNonEmpty(install.Summary.Name, install.ID)
			canonical = append(canonical, domain.SessionResourceReference{Kind: reference.Kind, ID: install.ID, Name: name})
		case domain.SessionResourceMCP:
			server, ok := servers[reference.ID]
			if !ok || failedSources[toolSourceEligibilityKey(domain.ToolSourceMCP, reference.ID)] {
				return preparedSessionResourceReferences{}, fmt.Errorf("MCP reference %q is unavailable", reference.ID)
			}
			for _, name := range sessionResourceSourceToolNames(toolEntries, domain.ToolSourceMCP, server.ID) {
				toolNames[name] = true
			}
			name := firstNonEmpty(server.DisplayName, server.Name, server.ID)
			canonical = append(canonical, domain.SessionResourceReference{Kind: reference.Kind, ID: server.ID, Name: name})
		}
	}

	prepared.references = canonical
	prepared.skillIDs = sortedStringSet(skillIDs)
	prepared.toolNames = sortedStringSet(toolNames)
	return prepared, nil
}

func (s *Service) applySessionResourceReferences(ctx context.Context, sessionID string, prepared preparedSessionResourceReferences) error {
	if prepared.project != nil {
		if _, err := s.AssociateAgentProject(ctx, sessionID, prepared.project.project.ID, prepared.project.project.RootPath); err != nil {
			return err
		}
	}
	if len(prepared.skillIDs) > 0 {
		active, err := s.GetSessionActiveSkills(ctx, sessionID)
		if err != nil {
			return err
		}
		ids := append(append([]string(nil), active.SkillIDs...), prepared.skillIDs...)
		if _, err := s.SetSessionActiveSkills(ctx, domain.SessionActiveSkillsInput{SessionID: sessionID, SkillIDs: ids}); err != nil {
			return err
		}
	}
	if len(prepared.toolNames) > 0 {
		active, err := s.GetSessionActiveTools(ctx, sessionID)
		if err != nil {
			return err
		}
		names := append(append(append([]string(nil), active.CoreToolNames...), active.ToolNames...), prepared.toolNames...)
		if _, err := s.SetSessionActiveTools(ctx, domain.SessionActiveToolsInput{SessionID: sessionID, ToolNames: names}); err != nil {
			return err
		}
	}
	return nil
}

func normalizeSessionResourceReferences(references []domain.SessionResourceReference) ([]domain.SessionResourceReference, error) {
	if len(references) > sessionResourceReferenceLimit {
		return nil, fmt.Errorf("resourceReferences exceeds the limit of %d", sessionResourceReferenceLimit)
	}
	seen := map[string]bool{}
	projects := map[string]bool{}
	normalized := make([]domain.SessionResourceReference, 0, len(references))
	for _, reference := range references {
		kind := strings.ToLower(strings.TrimSpace(reference.Kind))
		id := strings.TrimSpace(reference.ID)
		rootPath := strings.TrimSpace(reference.RootPath)
		switch kind {
		case domain.SessionResourceProject, domain.SessionResourceSkill, domain.SessionResourceTool, domain.SessionResourceExtension, domain.SessionResourceMCP:
		default:
			return nil, fmt.Errorf("unsupported resource reference kind %q", reference.Kind)
		}
		if id == "" {
			return nil, fmt.Errorf("resource reference %q requires an id", kind)
		}
		if len(id) > sessionResourceReferenceIDLimit {
			return nil, fmt.Errorf("resource reference %q id exceeds the limit of %d bytes", kind, sessionResourceReferenceIDLimit)
		}
		if kind == domain.SessionResourceProject {
			if rootPath == "" {
				return nil, errors.New("project resource reference requires rootPath")
			}
			if len(rootPath) > sessionResourceReferencePathLimit {
				return nil, fmt.Errorf("project resource reference rootPath exceeds the limit of %d bytes", sessionResourceReferencePathLimit)
			}
			projects[id+"\x00"+rootPath] = true
			if len(projects) > 1 {
				return nil, errors.New("only one project resource reference is allowed")
			}
		}
		key := kind + "\x00" + id
		if seen[key] {
			continue
		}
		seen[key] = true
		normalized = append(normalized, domain.SessionResourceReference{Kind: kind, ID: id, RootPath: rootPath})
	}
	return normalized, nil
}

func sessionResourceReferencePayload(references []domain.SessionResourceReference) []map[string]string {
	payload := make([]map[string]string, 0, len(references))
	for _, reference := range references {
		item := map[string]string{"kind": reference.Kind, "id": reference.ID}
		if strings.TrimSpace(reference.Name) != "" {
			item["name"] = reference.Name
		}
		payload = append(payload, item)
	}
	return payload
}

func renderSessionResourceReferenceContext(references []domain.SessionResourceReference) string {
	if len(references) == 0 {
		return ""
	}
	lines := []string{
		"The user explicitly selected these resources in the composer. Core already validated and attached them to this conversation.",
		"<explicit_user_resource_references>",
	}
	for _, reference := range references {
		lines = append(lines, fmt.Sprintf(
			`  <resource kind="%s" id="%s" name="%s" />`,
			xmlEscape(reference.Kind),
			xmlEscape(reference.ID),
			xmlEscape(reference.Name),
		))
	}
	lines = append(lines, "</explicit_user_resource_references>")
	return bounded(strings.Join(lines, "\n"), 4000)
}

func hasSessionResourceKind(references []domain.SessionResourceReference, kind string) bool {
	for _, reference := range references {
		if reference.Kind == kind {
			return true
		}
	}
	return false
}

func sortedStringSet(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func sessionResourceSourceToolNames(entries map[string]domain.ToolCatalogEntry, source, sourceID string) []string {
	names := make([]string, 0)
	for _, entry := range entries {
		if entry.Source == source && entry.SourceID == sourceID {
			names = append(names, entry.Name)
		}
	}
	sort.Strings(names)
	return names
}

func sessionResourceSelectionGroupToolNames(entries map[string]domain.ToolCatalogEntry, groupID, kind string) ([]string, string) {
	names := make([]string, 0)
	groupName := ""
	for _, entry := range entries {
		if entry.SelectionGroup == nil || entry.SelectionGroup.ID != groupID || hostToolSelectionResourceKind(entry) != kind {
			continue
		}
		names = append(names, entry.Name)
		if groupName == "" {
			groupName = entry.SelectionGroup.Name
		}
	}
	sort.Strings(names)
	return names, firstNonEmpty(groupName, groupID)
}
