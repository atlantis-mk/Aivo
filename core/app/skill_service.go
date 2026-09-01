package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"aivo/core/domain"
)

const skillSupportingFileLimit = 10

type loadedSkillResult struct {
	Event       domain.SessionEvent
	Skill       domain.SkillEntry
	ModelOutput string
}

func (s *Service) ensureSkillManager() *SkillManager {
	if s.skillManager == nil {
		s.skillManager = NewSkillManager(s.store)
	}
	return s.skillManager
}

func (s *Service) ScanGlobalSkills(ctx context.Context) (domain.SkillScanResult, error) {
	return s.ensureSkillManager().ScanGlobal(ctx)
}

func (s *Service) ScanProjectSkills(ctx context.Context, input domain.SkillScanInput) (domain.SkillScanResult, error) {
	return s.ensureSkillManager().ScanProject(ctx, input.WorkspaceRoot)
}

func (s *Service) ListSkills(ctx context.Context, input domain.SkillListInput) (domain.SkillListResult, error) {
	s.syncAivoSystemSkills(ctx)
	hasCodexAccount := s.codexOAuthConfigured(ctx)
	if hasCodexAccount {
		s.syncCodexSystemSkillsForAccount(ctx)
	}
	result, err := s.ensureSkillManager().List(ctx, input)
	if err != nil {
		return result, err
	}
	filtered := result.Entries[:0]
	for _, skill := range result.Entries {
		if skill.Source == domain.SkillSourceCodexSystem && !hasCodexAccount {
			continue
		}
		if isSystemSkillSource(skill.Source) && !s.skillAvailableForUse(ctx, skill) {
			continue
		}
		if skill.Source != domain.SkillSourceCodexSystem || hasCodexAccount {
			filtered = append(filtered, skill)
		}
	}
	result.Entries = filtered
	result.Entries = s.decorateSkillEntries(ctx, result.Entries)
	return result, nil
}

func (s *Service) ImportSkill(ctx context.Context, input domain.SkillImportInput) (domain.SkillEntry, error) {
	skill, err := s.ensureSkillManager().Import(ctx, input)
	if err != nil {
		return domain.SkillEntry{}, err
	}
	return s.decorateSkillEntry(ctx, skill), nil
}

func (s *Service) IgnoreSkillCandidatesByName(ctx context.Context, input domain.SkillIgnoreCandidatesInput) ([]domain.SkillImportCandidate, error) {
	return s.ensureSkillManager().IgnoreCandidatesByName(ctx, input)
}

func (s *Service) SetSkillEnabled(ctx context.Context, input domain.SkillEnabledInput) (domain.SkillEntry, error) {
	skill, err := s.ensureSkillManager().SetEnabled(ctx, input)
	if err != nil {
		return domain.SkillEntry{}, err
	}
	return s.decorateSkillEntry(ctx, skill), nil
}

func (s *Service) GetManagedSkillForEdit(ctx context.Context, skillID string) (domain.SkillEditResult, error) {
	result, err := s.ensureSkillManager().GetForEdit(ctx, skillID)
	if err != nil {
		return domain.SkillEditResult{}, err
	}
	result.Skill = s.decorateSkillEntry(ctx, result.Skill)
	return result, nil
}

func (s *Service) UpdateManagedSkill(ctx context.Context, input domain.SkillUpdateInput) (domain.SkillEditResult, error) {
	result, err := s.ensureSkillManager().Update(ctx, input)
	if err != nil {
		return domain.SkillEditResult{}, err
	}
	result.Skill = s.decorateSkillEntry(ctx, result.Skill)
	return result, nil
}

func (s *Service) DeleteManagedSkill(ctx context.Context, skillID string) error {
	return s.ensureSkillManager().Delete(ctx, skillID)
}

func (s *Service) LoadSkillIntoSession(ctx context.Context, input domain.LoadSkillIntoSessionInput) (domain.SessionEvent, error) {
	result, err := s.loadSkillIntoSession(ctx, input)
	return result.Event, err
}

func (s *Service) loadOrImportSkillIntoSession(ctx context.Context, input domain.LoadSkillIntoSessionInput) (loadedSkillResult, error) {
	manager := s.ensureSkillManager()
	skill, err := manager.Resolve(ctx, input.SkillID, input.Name, input.Scope)
	if err != nil && strings.TrimSpace(input.SkillID) == "" && strings.TrimSpace(input.Name) != "" {
		skill, err = manager.ImportByName(ctx, input.Name, input.Scope)
	}
	if err != nil {
		return loadedSkillResult{}, err
	}
	input.SkillID = skill.ID
	input.Name = ""
	input.Scope = ""
	return s.loadSkillIntoSession(ctx, input)
}

func (s *Service) loadSkillIntoSession(ctx context.Context, input domain.LoadSkillIntoSessionInput) (loadedSkillResult, error) {
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return loadedSkillResult{}, errors.New("sessionId is required")
	}
	s.syncAivoSystemSkills(ctx)
	manager := s.ensureSkillManager()
	skill, err := manager.Resolve(ctx, input.SkillID, input.Name, input.Scope)
	if err != nil {
		return loadedSkillResult{}, err
	}
	if skill.Source == domain.SkillSourceCodexSystem && !s.codexOAuthConfigured(ctx) {
		return loadedSkillResult{}, errors.New("Codex system skill requires a connected Codex OAuth account")
	}
	if !skill.Enabled {
		return loadedSkillResult{}, errors.New("skill is disabled")
	}
	if !s.skillAvailableForUse(ctx, skill) {
		return loadedSkillResult{}, errors.New("skill is not available")
	}
	already, _ := s.sessionSkillLoaded(ctx, sessionID, skill)
	if already && !input.Reload {
		return loadedSkillResult{}, errors.New("skill is already loaded in this session")
	}
	content, err := manager.ReadContent(skill)
	if err != nil {
		return loadedSkillResult{}, err
	}
	files, err := manager.SupportingFiles(skill, skillSupportingFileLimit)
	if err != nil {
		return loadedSkillResult{}, err
	}
	output := renderSkillModelOutputWithSnapshot(s.currentPromptSnapshot(), skill, content, files)
	if _, err := s.rememberActiveSkill(ctx, sessionID, skill); err != nil {
		return loadedSkillResult{}, err
	}
	event, err := s.AppendEvent(ctx, domain.AppendEventRequest{
		SessionID: sessionID, Type: domain.EventTypeSystemNote, Role: domain.EventRoleSystem, Visibility: domain.EventVisibilityInternal,
		Content: output,
		Payload: map[string]any{"kind": "skill", "skillId": skill.ID, "skillName": skill.Name, "contentHash": skill.ContentHash, "reason": strings.TrimSpace(input.Reason)},
	})
	if err == nil && s.onSessionUpdated != nil {
		s.onSessionUpdated(sessionID, nil)
	}
	return loadedSkillResult{Event: event, Skill: skill, ModelOutput: output}, err
}

func (s *Service) GetSessionActiveSkills(ctx context.Context, sessionID string) (domain.SessionActiveSkillsResult, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return domain.SessionActiveSkillsResult{}, errors.New("sessionId is required")
	}
	ids, skills := s.activeSkills(ctx, sessionID)
	return domain.SessionActiveSkillsResult{SessionID: sessionID, SkillIDs: ids, Skills: skills}, nil
}

func (s *Service) SetSessionActiveSkills(ctx context.Context, input domain.SessionActiveSkillsInput) (domain.SessionActiveSkillsResult, error) {
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return domain.SessionActiveSkillsResult{}, errors.New("sessionId is required")
	}
	workspaceRoot := s.sessionSkillWorkspaceRoot(ctx, sessionID)
	seen := map[string]bool{}
	ids := make([]string, 0, len(input.SkillIDs))
	visibleSkillIDs := make([]string, 0, len(input.SkillIDs))
	for _, id := range input.SkillIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		skills := s.resolveAvailableSkillReference(ctx, workspaceRoot, id)
		if len(skills) == 0 {
			continue
		}
		for _, skill := range skills {
			if seen[skill.ID] {
				continue
			}
			seen[skill.ID] = true
			ids = append(ids, skill.ID)
			visibleSkillIDs = append(visibleSkillIDs, skill.ID)
		}
	}
	sort.Strings(ids)
	state, err := s.store.GetSessionExecutionState(ctx, sessionID)
	if err != nil {
		return domain.SessionActiveSkillsResult{}, err
	}
	if state.Metadata == nil {
		state.Metadata = map[string]any{}
	}
	state.Metadata[sessionMetadataActiveSkills] = ids
	state.Metadata[sessionMetadataVisibleSkills] = mergeSessionMetadataStringSet(state.Metadata[sessionMetadataVisibleSkills], visibleSkillIDs)
	markAutoSelectedToolsInitialized(state.Metadata)
	if _, err := s.store.UpsertSessionExecutionState(ctx, state); err != nil {
		return domain.SessionActiveSkillsResult{}, err
	}
	if s.onSessionUpdated != nil {
		s.onSessionUpdated(sessionID, nil)
	}
	return s.GetSessionActiveSkills(ctx, sessionID)
}

func (s *Service) sessionSkillWorkspaceRoot(ctx context.Context, sessionID string) string {
	if s == nil || s.store == nil || strings.TrimSpace(sessionID) == "" {
		return ""
	}
	if session, err := s.store.GetRuntimeSession(ctx, sessionID); err == nil {
		if root := strings.TrimSpace(session.ProjectPath); root != "" {
			return root
		}
	}
	if codingContext, err := s.store.GetCodingContext(ctx, sessionID); err == nil {
		return strings.TrimSpace(codingContext.ProjectPath)
	}
	return ""
}

func (s *Service) resolveAvailableSkillReference(ctx context.Context, workspaceRoot string, id string) []domain.SkillEntry {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	if strings.HasPrefix(id, skillGroupResourceKeyPrefix) {
		return s.availableSkillGroupMembers(ctx, workspaceRoot, strings.TrimPrefix(id, skillGroupResourceKeyPrefix))
	}
	manager := s.ensureSkillManager()
	skill, err := manager.Resolve(ctx, id, "", "")
	if err == nil && skill.Enabled && s.skillAvailableForUse(ctx, skill) {
		return []domain.SkillEntry{s.decorateSkillEntry(ctx, skill)}
	}
	return s.availableSkillGroupMembers(ctx, workspaceRoot, id)
}

func (s *Service) availableSkillGroupMembers(ctx context.Context, workspaceRoot string, groupID string) []domain.SkillEntry {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return nil
	}
	result, err := s.ListSkills(ctx, domain.SkillListInput{WorkspaceRoot: strings.TrimSpace(workspaceRoot)})
	if err != nil {
		return nil
	}
	members := make([]domain.SkillEntry, 0)
	for _, skill := range result.Entries {
		if !skill.Enabled || !s.skillAvailableForUse(ctx, skill) || skill.SelectionGroup == nil || skill.SelectionGroup.ID != groupID {
			continue
		}
		members = append(members, skill)
	}
	sort.Slice(members, func(i, j int) bool { return members[i].Name < members[j].Name })
	return members
}

func skillGroupDisplayName(skills []domain.SkillEntry, groupID string) string {
	for _, skill := range skills {
		if skill.SelectionGroup != nil && skill.SelectionGroup.ID == groupID && strings.TrimSpace(skill.SelectionGroup.Name) != "" {
			return skill.SelectionGroup.Name
		}
	}
	return firstNonEmpty(groupID, "Skill group")
}

func (s *Service) sessionSkillLoaded(ctx context.Context, sessionID string, skill domain.SkillEntry) (bool, error) {
	state, err := s.store.GetSessionExecutionState(ctx, sessionID)
	if err != nil {
		return false, err
	}
	for _, item := range stringSliceFromAny(state.Metadata[sessionMetadataActiveSkills]) {
		if strings.TrimSpace(item) == skill.ID {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) rememberActiveSkill(ctx context.Context, sessionID string, skill domain.SkillEntry) ([]string, error) {
	state, err := s.store.GetSessionExecutionState(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var ids []string
	for _, id := range stringSliceFromAny(state.Metadata[sessionMetadataActiveSkills]) {
		id = strings.TrimSpace(id)
		if id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if !seen[skill.ID] {
		ids = append(ids, skill.ID)
	}
	sort.Strings(ids)
	if state.Metadata == nil {
		state.Metadata = map[string]any{}
	}
	state.Metadata[sessionMetadataActiveSkills] = ids
	state.Metadata[sessionMetadataVisibleSkills] = mergeSessionMetadataStringSet(state.Metadata[sessionMetadataVisibleSkills], []string{skill.ID})
	markAutoSelectedToolsInitialized(state.Metadata)
	_, err = s.store.UpsertSessionExecutionState(ctx, state)
	return ids, err
}

func (s *Service) activeSkills(ctx context.Context, sessionID string) ([]string, []domain.SkillEntry) {
	if s == nil || s.store == nil {
		return nil, nil
	}
	state, err := s.store.GetSessionExecutionState(ctx, sessionID)
	if err != nil {
		return nil, nil
	}
	manager := s.ensureSkillManager()
	ids := stringSliceFromAny(state.Metadata[sessionMetadataActiveSkills])
	outIDs := make([]string, 0, len(ids))
	skills := make([]domain.SkillEntry, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		skill, err := manager.Resolve(ctx, id, "", "")
		if err != nil || !s.skillAvailableForUse(ctx, skill) {
			continue
		}
		seen[id] = true
		outIDs = append(outIDs, id)
		skills = append(skills, s.decorateSkillEntry(ctx, skill))
	}
	sort.Strings(outIDs)
	return outIDs, skills
}

func (s *Service) rememberVisibleSkills(ctx context.Context, sessionID string, skills []domain.SkillEntry) ([]string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, errors.New("sessionId is required")
	}
	state, err := s.store.GetSessionExecutionState(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	ids := make([]string, 0, len(skills))
	for _, skill := range skills {
		if strings.TrimSpace(skill.ID) == "" || seen[skill.ID] {
			continue
		}
		seen[skill.ID] = true
		ids = append(ids, skill.ID)
	}
	sort.Strings(ids)
	if state.Metadata == nil {
		state.Metadata = map[string]any{}
	}
	state.Metadata[sessionMetadataVisibleSkills] = ids
	_, err = s.store.UpsertSessionExecutionState(ctx, state)
	return ids, err
}

func (s *Service) visibleSkills(ctx context.Context, sessionID string) ([]string, []domain.SkillEntry) {
	if s == nil || s.store == nil || strings.TrimSpace(sessionID) == "" {
		return nil, nil
	}
	state, err := s.store.GetSessionExecutionState(ctx, sessionID)
	if err != nil || state.Metadata == nil {
		return nil, nil
	}
	manager := s.ensureSkillManager()
	ids := stringSliceFromAny(state.Metadata[sessionMetadataVisibleSkills])
	outIDs := make([]string, 0, len(ids))
	skills := make([]domain.SkillEntry, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		skill, err := manager.Resolve(ctx, id, "", "")
		if err != nil || !skill.Enabled || !s.skillAvailableForUse(ctx, skill) {
			continue
		}
		seen[id] = true
		outIDs = append(outIDs, id)
		skills = append(skills, s.decorateSkillEntry(ctx, skill))
	}
	sort.Strings(outIDs)
	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })
	return outIDs, skills
}

func (s *Service) visibleSkillsContext(ctx context.Context, sessionID string) string {
	_, skills := s.visibleSkills(ctx, sessionID)
	if len(skills) == 0 {
		return ""
	}
	return renderAvailableSkills(skills, nil)
}

func mergeSessionMetadataStringSet(existing any, added []string) []string {
	seen := map[string]bool{}
	values := make([]string, 0)
	for _, value := range stringSliceFromAny(existing) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		values = append(values, value)
	}
	for _, value := range added {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func (s *Service) activeSkillsContext(ctx context.Context, sessionID string) string {
	_, skills := s.activeSkills(ctx, sessionID)
	if len(skills) == 0 {
		return ""
	}
	manager := s.ensureSkillManager()
	blocks := make([]string, 0, len(skills))
	for _, skill := range skills {
		content, err := manager.ReadContent(skill)
		if err != nil {
			continue
		}
		files, _ := manager.SupportingFiles(skill, skillSupportingFileLimit)
		blocks = append(blocks, renderSkillModelOutputWithSnapshot(s.currentPromptSnapshot(), skill, content, files))
	}
	return strings.Join(blocks, "\n\n")
}

func (s *Service) activeExtensionContextKeys(ctx context.Context, sessionID string) []string {
	if s == nil || s.store == nil || strings.TrimSpace(sessionID) == "" {
		return nil
	}
	state, err := s.store.GetSessionExecutionState(ctx, sessionID)
	if err != nil || state.Metadata == nil {
		return nil
	}
	keys := normalizeHostResourceKeys(stringSliceFromAny(state.Metadata[sessionMetadataActiveExtensionContexts]))
	return keys
}

func (s *Service) rememberActiveExtensionContexts(ctx context.Context, sessionID string, keys []string) ([]string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, errors.New("sessionId is required")
	}
	state, err := s.store.GetSessionExecutionState(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	merged := make([]string, 0)
	for _, key := range stringSliceFromAny(state.Metadata[sessionMetadataActiveExtensionContexts]) {
		key = strings.TrimSpace(key)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		merged = append(merged, key)
	}
	for _, key := range normalizeHostResourceKeys(keys) {
		if seen[key] {
			continue
		}
		seen[key] = true
		merged = append(merged, key)
	}
	sort.Strings(merged)
	if state.Metadata == nil {
		state.Metadata = map[string]any{}
	}
	state.Metadata[sessionMetadataActiveExtensionContexts] = merged
	_, err = s.store.UpsertSessionExecutionState(ctx, state)
	return merged, err
}

func (s *Service) activeExtensionContextsContext(ctx context.Context, sessionID string) string {
	keys := s.activeExtensionContextKeys(ctx, sessionID)
	if len(keys) == 0 || s == nil || s.extensionSupervisor == nil {
		return ""
	}
	allowed := map[string]bool{}
	for _, key := range keys {
		allowed[key] = true
	}
	candidates := s.extensionSupervisor.ContextCatalog()
	blocks := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if !allowed[candidate.Key] {
			continue
		}
		resource, ok := s.extensionContextResource(ctx, candidate.ExtensionID, candidate.ContextID)
		if ok {
			blocks = append(blocks, renderExtensionContextResource(resource))
		}
	}
	return strings.Join(blocks, "\n\n")
}

func (s *Service) availableSkillsContext(ctx context.Context, workspaceRoot string) string {
	if s == nil || s.store == nil {
		return ""
	}
	result, err := s.ListSkills(ctx, domain.SkillListInput{WorkspaceRoot: workspaceRoot, IncludeCandidates: true})
	if err != nil {
		return ""
	}
	skills := make([]domain.SkillEntry, 0, len(result.Entries))
	for _, skill := range result.Entries {
		if skill.Enabled && strings.TrimSpace(skill.Description) != "" {
			skills = append(skills, skill)
		}
	}
	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })
	candidates := make([]domain.SkillImportCandidate, 0, len(result.Candidates))
	for _, candidate := range result.Candidates {
		if candidate.Status == domain.SkillCandidateStatusPending && strings.TrimSpace(candidate.Description) != "" {
			candidates = append(candidates, candidate)
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Name < candidates[j].Name })
	return renderAvailableSkills(skills, candidates)
}

func renderAvailableSkills(skills []domain.SkillEntry, candidates []domain.SkillImportCandidate) string {
	if len(skills) == 0 && len(candidates) == 0 {
		return ""
	}
	lines := []string{
		"Skills provide specialized instructions and workflows for specific tasks.",
		"Only the Skills selected for this session are listed. Use skills_read with the package locator to read SKILL.md before following a Skill. Pending import candidates are not activatable; ignored candidates are not listed.",
	}
	lines = append(lines, "<available_skills>")
	for _, skill := range skills {
		lines = append(lines,
			"  <skill>",
			fmt.Sprintf("    <name>%s</name>", xmlEscape(skill.Name)),
			fmt.Sprintf("    <description>%s</description>", xmlEscape(skill.Description)),
			fmt.Sprintf("    <package>%s</package>", xmlEscape(skillPackageLocator(skill))),
			fmt.Sprintf("    <main_resource>%s</main_resource>", xmlEscape(skillMainResourceLocator(skill))),
			"    <status>imported</status>",
			"  </skill>",
		)
	}
	for _, candidate := range candidates {
		lines = append(lines,
			"  <skill>",
			fmt.Sprintf("    <name>%s</name>", xmlEscape(candidate.Name)),
			fmt.Sprintf("    <description>%s</description>", xmlEscape(candidate.Description)),
			"    <status>pending_import</status>",
			"  </skill>",
		)
	}
	lines = append(lines, "</available_skills>")
	return strings.Join(lines, "\n")
}

func renderSkillModelOutput(skill domain.SkillEntry, content string, files []string) string {
	return renderSkillModelOutputWithSnapshot(PromptSnapshot{}, skill, content, files)
}

func renderSkillModelOutputWithSnapshot(snapshot PromptSnapshot, skill domain.SkillEntry, content string, files []string) string {
	directory := strings.TrimSpace(skill.RootPath)
	if directory == "" && strings.TrimSpace(skill.SkillPath) != "" {
		directory = filepath.Dir(skill.SkillPath)
	}
	footer, err := snapshot.Render("dynamic.skill_content_footer", map[string]string{"directory": directory})
	if err != nil {
		footer, _ = renderPromptTemplate(builtinPromptBody("dynamic.skill_content_footer"), map[string]string{"directory": directory})
	}
	lines := []string{
		fmt.Sprintf(`<skill_content name="%s">`, xmlEscape(skill.Name)),
		"# Skill: " + skill.Name,
		"",
		strings.TrimSpace(content),
		"",
		footer,
		"",
		"<skill_resources>",
	}
	for _, file := range files {
		relative := file
		if rel, err := filepath.Rel(directory, file); err == nil {
			relative = filepath.ToSlash(rel)
		}
		lines = append(lines, fmt.Sprintf("<file>%s</file>", xmlEscape(relative)))
	}
	lines = append(lines, "</skill_resources>", "</skill_content>")
	return strings.Join(lines, "\n")
}

func xmlEscape(value string) string {
	value = strings.ReplaceAll(value, "&", "&amp;")
	value = strings.ReplaceAll(value, "<", "&lt;")
	value = strings.ReplaceAll(value, ">", "&gt;")
	value = strings.ReplaceAll(value, `"`, "&quot;")
	return value
}
