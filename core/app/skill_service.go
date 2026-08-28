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
	hasCodexAccount := s.codexOAuthConfigured(ctx)
	if hasCodexAccount {
		s.syncCodexSystemSkillsForAccount(ctx)
	}
	result, err := s.ensureSkillManager().List(ctx, input)
	if err != nil || hasCodexAccount {
		return result, err
	}
	filtered := result.Entries[:0]
	for _, skill := range result.Entries {
		if skill.Source != domain.SkillSourceCodexSystem {
			filtered = append(filtered, skill)
		}
	}
	result.Entries = filtered
	return result, nil
}

func (s *Service) ImportSkill(ctx context.Context, input domain.SkillImportInput) (domain.SkillEntry, error) {
	return s.ensureSkillManager().Import(ctx, input)
}

func (s *Service) IgnoreSkillCandidatesByName(ctx context.Context, input domain.SkillIgnoreCandidatesInput) ([]domain.SkillImportCandidate, error) {
	return s.ensureSkillManager().IgnoreCandidatesByName(ctx, input)
}

func (s *Service) SetSkillEnabled(ctx context.Context, input domain.SkillEnabledInput) (domain.SkillEntry, error) {
	return s.ensureSkillManager().SetEnabled(ctx, input)
}

func (s *Service) GetManagedSkillForEdit(ctx context.Context, skillID string) (domain.SkillEditResult, error) {
	return s.ensureSkillManager().GetForEdit(ctx, skillID)
}

func (s *Service) UpdateManagedSkill(ctx context.Context, input domain.SkillUpdateInput) (domain.SkillEditResult, error) {
	return s.ensureSkillManager().Update(ctx, input)
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
	manager := s.ensureSkillManager()
	seen := map[string]bool{}
	ids := make([]string, 0, len(input.SkillIDs))
	for _, id := range input.SkillIDs {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		skill, err := manager.Resolve(ctx, id, "", "")
		if err != nil || !skill.Enabled || (skill.Source == domain.SkillSourceCodexSystem && !s.codexOAuthConfigured(ctx)) {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
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
	if _, err := s.store.UpsertSessionExecutionState(ctx, state); err != nil {
		return domain.SessionActiveSkillsResult{}, err
	}
	if s.onSessionUpdated != nil {
		s.onSessionUpdated(sessionID, nil)
	}
	return s.GetSessionActiveSkills(ctx, sessionID)
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
		if err != nil || !skill.Enabled || (skill.Source == domain.SkillSourceCodexSystem && !s.codexOAuthConfigured(ctx)) {
			continue
		}
		seen[id] = true
		outIDs = append(outIDs, id)
		skills = append(skills, skill)
	}
	sort.Strings(outIDs)
	return outIDs, skills
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
		"Use the skill tool to load a skill when a task matches its description. Pending import candidates can be loaded by name; ignored candidates are not listed.",
	}
	lines = append(lines, "<available_skills>")
	for _, skill := range skills {
		lines = append(lines,
			"  <skill>",
			fmt.Sprintf("    <name>%s</name>", xmlEscape(skill.Name)),
			fmt.Sprintf("    <description>%s</description>", xmlEscape(skill.Description)),
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
