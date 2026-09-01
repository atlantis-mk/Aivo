package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strings"
	"time"
	"unicode"

	"aivo/core/domain"
)

const (
	hostCatalogPreparationTimeout = 8 * time.Second
)

func (s *Service) resolveSessionResources(ctx context.Context, request ResourceResolveRequest) (ResourceResolveDecision, error) {
	resolved, err := s.resolveHostResources(ctx, hostResourceResolveRequest{
		SessionID:      request.SessionID,
		TurnID:         request.TurnID,
		Intent:         request.Intent,
		Mode:           request.Mode,
		AgentMode:      request.AgentMode,
		WorkspaceRoot:  request.WorkspaceRoot,
		ToolCandidates: request.Candidates,
		Explicit:       true,
		Required:       request.Required,
	})
	if err != nil {
		return ResourceResolveDecision{}, err
	}
	return ResourceResolveDecision{
		Names:           resolved.ToolNames,
		ResourceKeys:    resolved.ResourceKeys,
		ResourceContext: resolved.ResourceContext,
		Resources:       resolved.Resources,
		Reason:          resolved.Reason,
	}, nil
}

type hostToolGroupCandidate struct {
	Kind        string
	ID          string
	Name        string
	Description string
	ToolNames   []string
	Grouped     bool
}

type hostResourceSelectionResource struct {
	Kind      string
	ID        string
	Name      string
	ToolCount int
}

type hostResourceSelectionIntent string

const (
	hostResourceSelectionInspect hostResourceSelectionIntent = "inspect"
	hostResourceSelectionUse     hostResourceSelectionIntent = "use"
)

type hostToolGroupDecision struct {
	Intent    hostResourceSelectionIntent
	Groups    []hostToolGroupCandidate
	ToolNames []string
	Reason    string
}

type hostInstructionCandidate struct {
	Key         string
	Kind        string
	Name        string
	Description string
	Source      string
	Skill       *SkillResolveCandidate
	SkillGroup  []SkillResolveCandidate
	Context     *extensionContextCandidate
}

type hostInstructionDecision struct {
	Keys   []string
	Reason string
}

type hostResourceDecision struct {
	ToolNames    []string
	ResourceKeys []string
	Reason       string
}

type hostResourceResolution struct {
	ToolActivations map[string]string
	ToolNames       []string
	ResourceKeys    []string
	Resources       []hostResourceSelectionResource
	ResourceContext string
	Reason          string
	Context         string
}

type hostResourceResolveRequest struct {
	SessionID                string
	TurnID                   string
	Intent                   string
	Mode                     string
	AgentMode                string
	WorkspaceRoot            string
	Registry                 *Registry
	Specs                    []domain.ToolSpec
	ToolCandidates           []domain.ToolCatalogEntry
	Explicit                 bool
	Required                 bool
	SkipInstructionSelection bool
}

// prepareEnabledToolCatalogs restores readiness/catalog state for sources the
// user has already enabled. It does not enable, trust, select, or execute a
// tool. Failures remain isolated to the source and are recorded by its manager.
func (s *Service) prepareEnabledToolCatalogs(ctx context.Context) map[string]bool {
	prepareCtx, cancel := context.WithTimeout(ctx, hostCatalogPreparationTimeout)
	defer cancel()
	failed := map[string]bool{}
	if s.mcpManager != nil {
		for key := range s.mcpManager.PrepareEnabledCatalogs(prepareCtx) {
			failed[key] = true
		}
	}
	return failed
}

func toolSourceEligibilityKey(source, sourceID string) string {
	return strings.TrimSpace(source) + "\x00" + strings.TrimSpace(sourceID)
}

func filterEligibleToolSpecs(registry *Registry, specs []domain.ToolSpec, failed map[string]bool) []domain.ToolSpec {
	if registry == nil || len(failed) == 0 {
		return specs
	}
	out := make([]domain.ToolSpec, 0, len(specs))
	for _, spec := range specs {
		identity, ok := registry.IdentityFor(spec.Name)
		if ok && failed[toolSourceEligibilityKey(identity.Source, identity.SourceID)] {
			continue
		}
		out = append(out, spec)
	}
	return out
}

func (s *Service) resolveHostResources(
	ctx context.Context,
	request hostResourceResolveRequest,
) (hostResourceResolution, error) {
	sessionID := strings.TrimSpace(request.SessionID)
	intent := strings.TrimSpace(request.Intent)
	mode := string(hostResourceSelectionUse)
	if request.Explicit || strings.TrimSpace(request.Mode) != "" {
		normalized, err := normalizeResourceResolveMode(request.Mode)
		if err != nil {
			return hostResourceResolution{}, err
		}
		mode = normalized
	}
	toolActivations := map[string]string{}
	toolCandidates := request.ToolCandidates
	if !request.Explicit {
		toolActivations, toolCandidates = s.snapshotToolCandidates(ctx, sessionID, request.TurnID, request.Registry, request.Specs)
	}
	for name, source := range s.visibleSkillToolActivations(ctx, sessionID) {
		if toolActivations[name] == "" {
			toolActivations[name] = source
		}
	}
	if !request.Explicit {
		return hostResourceResolution{ToolActivations: toolActivations}, nil
	}
	filteredToolCandidates, eligibilityErr := s.filterGloballyVisibleToolCatalogEntries(ctx, toolCandidates)
	if eligibilityErr == nil {
		toolCandidates = filteredToolCandidates
	} else {
		toolCandidates = nil
	}
	if request.Explicit {
		manual := s.rememberedDeferredTools(ctx, sessionID)
		filtered := make([]domain.ToolCatalogEntry, 0, len(toolCandidates))
		for _, candidate := range toolCandidates {
			if !manual[candidate.Name] {
				filtered = append(filtered, candidate)
			}
		}
		toolCandidates = filtered
	}
	if intent == "" {
		return hostResourceResolution{ToolActivations: toolActivations}, nil
	}
	instructionCandidates := []hostInstructionCandidate{}
	if !request.SkipInstructionSelection {
		_, instructionCandidates = s.hostInstructionCandidates(ctx, request.WorkspaceRoot)
	}
	if request.Explicit {
		if len(toolCandidates) == 0 && len(instructionCandidates) == 0 {
			if !request.Required && mode == string(hostResourceSelectionUse) {
				if err := s.replaceAutoSelectedTools(ctx, sessionID, nil); err != nil {
					return hostResourceResolution{}, err
				}
			}
			return hostResourceResolution{ToolActivations: toolActivations, Reason: "no globally visible eligible automatic tool or instruction resources"}, nil
		}
	}
	toolGroups := hostToolGroupCandidates(toolCandidates)
	toolDecision := hostToolGroupDecision{Intent: hostResourceSelectionUse, Reason: "no eligible tool resources"}
	if mode == string(hostResourceSelectionInspect) && isToolInventoryIntent(intent) {
		groups, names, err := expandHostToolGroups(toolGroups, nil, false)
		if err != nil {
			toolDecision = hostToolGroupDecision{Intent: hostResourceSelectionInspect, Reason: err.Error()}
		} else {
			toolDecision = hostToolGroupDecision{Intent: hostResourceSelectionInspect, Groups: groups, ToolNames: names, Reason: "matched by local catalog inspection"}
		}
	} else if !(isSkillInventoryIntent(intent) && !isToolInventoryIntent(intent)) {
		toolDecision = s.resolveHostToolGroupsWithAuxiliaryModel(ctx, intent, toolGroups)
	}
	if toolDecision.Intent == hostResourceSelectionInspect {
		toolDecision.ToolNames = make([]string, 0, len(toolCandidates))
		for _, entry := range toolCandidates {
			toolDecision.ToolNames = append(toolDecision.ToolNames, entry.Name)
		}
	}
	instructionDecision := hostInstructionDecision{}
	if len(instructionCandidates) > 0 {
		localInstructionPreview := localHostInstructionResolve(intent, instructionCandidates)
		if len(localInstructionPreview.Keys) > 0 {
			instructionDecision = s.resolveHostInstructionsWithAuxiliaryModel(ctx, intent, request.AgentMode, instructionCandidates)
		}
	}
	if request.Explicit && len(instructionDecision.Keys) > 0 && strings.HasPrefix(toolDecision.Reason, "matched by local Host resource-group search") {
		toolDecision.Groups = nil
		toolDecision.ToolNames = nil
	}
	selectedTools := validateResourceResolveToolSelection(toolCandidates, toolDecision.ToolNames)
	if request.Explicit && mode == string(hostResourceSelectionInspect) {
		resources := hostResourceSelectionResources(toolDecision.Groups, normalizeDeferredToolNames(selectedToolNames(selectedTools)))
		if len(instructionDecision.Keys) > 0 {
			selectedInstructions := validateHostInstructionSelection(instructionCandidates, instructionDecision.Keys)
			resources = append(resources, s.hostInstructionSelectionResources(ctx, sessionID, selectedInstructions)...)
		}
		reason := strings.TrimSpace(toolDecision.Reason)
		if strings.TrimSpace(instructionDecision.Reason) != "" {
			reason = strings.TrimSpace(reason + "; " + strings.TrimSpace(instructionDecision.Reason))
		}
		if reason == "" {
			reason = "no eligible grouped, individual, or instruction resources"
		}
		return hostResourceResolution{
			ToolActivations: toolActivations,
			Resources:       normalizeHostResourceSelectionResources(resources),
			Reason:          reason,
		}, nil
	}
	if request.Explicit {
		if len(selectedTools) > 0 {
			if err := s.replaceAutoSelectedTools(ctx, sessionID, normalizeDeferredToolNames(selectedToolNames(selectedTools))); err != nil {
				return hostResourceResolution{}, err
			}
		}
	}
	selectedResources := hostResourceSelectionResources(toolDecision.Groups, normalizeDeferredToolNames(selectedToolNames(selectedTools)))
	resourceContext := ""
	resourceSummaries := []hostResourceSelectionResource{}
	if request.Explicit && len(instructionDecision.Keys) > 0 {
		var err error
		resourceContext, resourceSummaries, err = s.applySessionInstructionResources(ctx, sessionID, request.WorkspaceRoot, instructionDecision.Keys)
		if err != nil {
			return hostResourceResolution{}, err
		}
		if hasHostResourceSelectionKind(resourceSummaries, domain.SessionResourceSkill) {
			toolActivations[SkillsListToolName] = "skillCatalog"
			toolActivations[SkillsReadToolName] = "skillCatalog"
		}
	}
	resources := append([]hostResourceSelectionResource{}, selectedResources...)
	resources = append(resources, resourceSummaries...)
	if request.Explicit && len(selectedTools) == 0 && strings.TrimSpace(resourceContext) == "" && !request.Required {
		if err := s.replaceAutoSelectedTools(ctx, sessionID, nil); err != nil {
			return hostResourceResolution{}, err
		}
	}
	reason := strings.TrimSpace(toolDecision.Reason)
	if strings.TrimSpace(instructionDecision.Reason) != "" {
		reason = strings.TrimSpace(reason + "; " + strings.TrimSpace(instructionDecision.Reason))
	}
	if reason == "" && request.Explicit {
		reason = "no eligible grouped, individual, or instruction resources"
	}
	return hostResourceResolution{
		ToolActivations: toolActivations,
		ToolNames:       normalizeDeferredToolNames(selectedToolNames(selectedTools)),
		ResourceKeys:    instructionDecision.Keys,
		Resources:       normalizeHostResourceSelectionResources(resources),
		ResourceContext: resourceContext,
		Reason:          reason,
		Context:         resourceContext,
	}, nil
}

func selectedToolNames(entries []domain.ToolCatalogEntry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name)
	}
	return names
}

func hostToolGroupCandidates(entries []domain.ToolCatalogEntry) []hostToolGroupCandidate {
	type groupIdentity struct {
		key string
	}
	identities := make([]groupIdentity, 0, len(entries))
	byKey := map[string]*hostToolGroupCandidate{}
	toolDescriptions := map[string][]string{}
	for _, entry := range entries {
		toolName := strings.TrimSpace(entry.Name)
		if toolName == "" {
			continue
		}
		kind := domain.SessionResourceTool
		candidateID := toolName
		candidateName := toolName
		description := sanitizeHostToolGroupText(entry.Description, 4000)
		grouped := false
		if entry.SelectionGroup != nil {
			kind = hostResourceSelectionToolKind(entry)
			candidateID = strings.TrimSpace(entry.SelectionGroup.ID)
			candidateName = sanitizeHostToolGroupText(entry.SelectionGroup.Name, 100)
			description = sanitizeHostToolGroupText(entry.SelectionGroup.Description, 4000)
			grouped = true
		}
		if candidateID == "" || sanitizeHostToolGroupText(candidateID, 512) != candidateID || candidateName == "" {
			continue
		}
		key := kind + "\x00" + candidateID
		group := byKey[key]
		if group == nil {
			group = &hostToolGroupCandidate{
				Kind: kind, ID: candidateID, Name: candidateName,
				Description: description, Grouped: grouped,
			}
			byKey[key] = group
			identities = append(identities, groupIdentity{key: key})
		} else if group.Name != candidateName || group.Grouped != grouped {
			continue
		} else if group.Description == "" {
			group.Description = description
		}
		if !containsString(group.ToolNames, toolName) {
			group.ToolNames = append(group.ToolNames, toolName)
			if description := sanitizeHostToolGroupText(entry.Description, 400); description != "" {
				toolDescriptions[key] = append(toolDescriptions[key], toolName+": "+description)
			}
		}
	}
	out := make([]hostToolGroupCandidate, 0, len(identities))
	for _, identity := range identities {
		group := byKey[identity.key]
		if group != nil && len(group.ToolNames) > 0 {
			if group.Description == "" && group.Kind == domain.SessionResourceMCP {
				group.Description = sanitizeHostToolGroupText(strings.Join(toolDescriptions[identity.key], "; "), 4000)
			}
			out = append(out, *group)
		}
	}
	return out
}

func hostStandaloneToolNames(entries []domain.ToolCatalogEntry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.SelectionGroup == nil {
			names = append(names, entry.Name)
		}
	}
	return normalizeDeferredToolNames(names)
}

func hostResourceSelectionToolKind(entry domain.ToolCatalogEntry) string {
	if entry.Source == domain.ToolSourceMCP || entry.Category == "mcp" {
		return domain.SessionResourceMCP
	}
	if entry.Source == domain.ToolSourceExtension && !strings.HasPrefix(strings.TrimSpace(entry.SourceID), "aivo.") {
		return domain.SessionResourceExtension
	}
	return domain.SessionResourceTool
}

func hostResourceSelectionResources(groups []hostToolGroupCandidate, selectedToolNames []string) []hostResourceSelectionResource {
	selected := make(map[string]bool, len(selectedToolNames))
	for _, name := range selectedToolNames {
		selected[name] = true
	}
	resources := make([]hostResourceSelectionResource, 0, len(groups))
	for _, group := range groups {
		count := 0
		for _, name := range group.ToolNames {
			if selected[name] {
				count++
			}
		}
		if count == 0 {
			continue
		}
		resources = append(resources, hostResourceSelectionResource{
			Kind: group.Kind, ID: group.ID, Name: group.Name, ToolCount: count,
		})
	}
	return resources
}

func hasHostResourceSelectionKind(resources []hostResourceSelectionResource, kind string) bool {
	for _, resource := range resources {
		if resource.Kind == kind {
			return true
		}
	}
	return false
}

func (s *Service) visibleSkillToolActivations(ctx context.Context, sessionID string) map[string]string {
	activations := map[string]string{}
	if strings.TrimSpace(sessionID) == "" {
		return activations
	}
	ids, _ := s.visibleSkills(ctx, sessionID)
	if len(ids) == 0 {
		return activations
	}
	activations[SkillsListToolName] = "skillCatalog"
	activations[SkillsReadToolName] = "skillCatalog"
	return activations
}

func (s *Service) hostInstructionSelectionResources(ctx context.Context, sessionID string, selected []hostInstructionCandidate) []hostResourceSelectionResource {
	activeNames := map[string]bool{}
	activeContextKeys := map[string]bool{}
	if strings.TrimSpace(sessionID) != "" {
		_, activeSkills := s.activeSkills(ctx, sessionID)
		for _, skill := range activeSkills {
			activeNames[skill.Name] = true
		}
		for _, key := range s.activeExtensionContextKeys(ctx, sessionID) {
			activeContextKeys[key] = true
		}
	}
	resources := make([]hostResourceSelectionResource, 0, len(selected))
	manager := s.ensureSkillManager()
	for _, candidate := range selected {
		if candidate.Skill != nil {
			if activeNames[candidate.Skill.Name] {
				continue
			}
			skill, err := manager.Resolve(ctx, "", candidate.Skill.Name, candidate.Skill.Scope)
			if err != nil || !skill.Enabled {
				continue
			}
			resources = append(resources, hostResourceSelectionResource{
				Kind: domain.SessionResourceSkill, ID: skill.Name, Name: skill.Name, ToolCount: 0,
			})
			continue
		}
		if len(candidate.SkillGroup) > 0 {
			count := 0
			for _, member := range candidate.SkillGroup {
				if activeNames[member.Name] {
					continue
				}
				skill, err := manager.Resolve(ctx, "", member.Name, member.Scope)
				if err != nil || !skill.Enabled {
					continue
				}
				count++
			}
			if count > 0 {
				resources = append(resources, hostResourceSelectionResource{
					Kind: domain.SessionResourceSkill, ID: strings.TrimPrefix(candidate.Key, skillGroupResourceKeyPrefix),
					Name: candidate.Name, ToolCount: 0,
				})
			}
			continue
		}
		if candidate.Context != nil {
			if activeContextKeys[candidate.Context.Key] {
				continue
			}
			resources = append(resources, hostResourceSelectionResource{
				Kind: domain.SessionResourceExtension, ID: candidate.Context.Key, Name: candidate.Context.Name, ToolCount: 0,
			})
		}
	}
	return resources
}

func sanitizeHostToolGroupText(value string, limit int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	return boundedHostToolGroupText(value, limit)
}

func boundedHostToolGroupText(value string, limit int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if limit <= 0 || len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func renderHostToolGroupSelectionPrompt(intent string, candidates []hostToolGroupCandidate) string {
	lines := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		lines = append(lines, renderHostToolSelectionCandidate(candidate))
	}
	prompt, err := renderPromptTemplate(builtinPromptBody("auxiliary.host_resource_groups.user"), map[string]string{
		"intent": boundedHostToolGroupText(intent, 4000), "candidates": strings.Join(lines, "\n"),
	})
	if err != nil {
		return ""
	}
	return prompt
}

func (s *Service) resolveHostToolGroupsWithAuxiliaryModel(ctx context.Context, intent string, candidates []hostToolGroupCandidate) hostToolGroupDecision {
	if strings.TrimSpace(intent) == "" || len(candidates) == 0 {
		return hostToolGroupDecision{Intent: hostResourceSelectionUse, Reason: "no eligible tool resources"}
	}
	systemPrompt, systemErr := s.renderManagedPrompt("auxiliary.host_resource_groups.system", nil)
	candidateLines := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidateLines = append(candidateLines, renderHostToolSelectionCandidate(candidate))
	}
	userPrompt, userErr := s.renderManagedPrompt("auxiliary.host_resource_groups.user", map[string]string{"intent": boundedHostToolGroupText(intent, 4000), "candidates": strings.Join(candidateLines, "\n")})
	if systemErr != nil || userErr != nil {
		decision, _ := localHostToolGroupSelection(intent, candidates)
		decision.Reason = "matched by local Host resource-group search"
		return decision
	}
	messages := []domain.ChatMessage{{Role: "system", Text: systemPrompt}, {Role: "user", Text: userPrompt}}
	for _, model := range s.resolveAuxiliaryModels(ctx, nil) {
		reply, _, err := s.GenerateChatReply(ctx, messages, &model, "low", "default")
		if err != nil {
			continue
		}
		decision, err := parseAndExpandHostToolGroupSelection(reply, candidates)
		if err == nil {
			decision.Reason = "selected by auxiliary Host resource-group resolver"
			return decision
		}
	}
	decision, err := localHostToolGroupSelection(intent, candidates)
	if err != nil {
		return hostToolGroupDecision{Intent: hostResourceSelectionUse, Reason: err.Error()}
	}
	decision.Reason = "matched by local Host resource-group search"
	return decision
}

func renderHostToolSelectionCandidate(candidate hostToolGroupCandidate) string {
	return candidate.Kind + ":" + candidate.ID + "：" + sanitizeHostToolGroupText(candidate.Name, 100) + "｜" + sanitizeHostToolGroupText(candidate.Description, 4000)
}

func parseAndExpandHostToolGroupSelection(raw string, candidates []hostToolGroupCandidate) (hostToolGroupDecision, error) {
	text := strings.TrimSpace(raw)
	if text == "" || !strings.HasPrefix(text, "{") || !strings.HasSuffix(text, "}") {
		return hostToolGroupDecision{}, errors.New("tool-group selection must be a strict JSON object")
	}
	var payload struct {
		Resources *[]struct {
			Kind string `json:"kind"`
			ID   string `json:"id"`
		} `json:"resources"`
	}
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return hostToolGroupDecision{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return hostToolGroupDecision{}, errors.New("tool-group selection contains trailing content")
	}
	if payload.Resources == nil {
		return hostToolGroupDecision{}, errors.New("resource selection is missing resources")
	}
	resources := *payload.Resources
	keys := make([]string, 0, len(resources))
	for _, resource := range resources {
		if resource.Kind != domain.SessionResourceMCP && resource.Kind != domain.SessionResourceExtension && resource.Kind != domain.SessionResourceTool {
			return hostToolGroupDecision{}, errors.New("resource selection contains an invalid resource kind")
		}
		if resource.ID == "" || strings.TrimSpace(resource.ID) != resource.ID || sanitizeHostToolGroupText(resource.ID, 512) != resource.ID {
			return hostToolGroupDecision{}, errors.New("resource selection contains an invalid resource id")
		}
		keys = append(keys, resource.Kind+"\x00"+resource.ID)
	}
	groups, toolNames, err := expandHostToolGroups(candidates, keys, true)
	if err != nil {
		return hostToolGroupDecision{}, err
	}
	return hostToolGroupDecision{Intent: hostResourceSelectionUse, Groups: groups, ToolNames: toolNames}, nil
}

func expandHostToolGroups(candidates []hostToolGroupCandidate, keys []string, validateKeys bool) ([]hostToolGroupCandidate, []string, error) {
	byName := make(map[string]hostToolGroupCandidate, len(candidates))
	for _, candidate := range candidates {
		byName[candidate.Kind+"\x00"+candidate.ID] = candidate
	}
	if !validateKeys {
		keys = make([]string, 0, len(candidates))
		for _, candidate := range candidates {
			keys = append(keys, candidate.Kind+"\x00"+candidate.ID)
		}
	}
	seenGroups := map[string]bool{}
	seenTools := map[string]bool{}
	groups := make([]hostToolGroupCandidate, 0, len(keys))
	expanded := make([]string, 0)
	for _, key := range keys {
		if key == "" || seenGroups[key] {
			return nil, nil, errors.New("tool-group selection contains an invalid or duplicate source")
		}
		candidate, ok := byName[key]
		if !ok {
			return nil, nil, errors.New("tool-group selection contains an unknown source")
		}
		seenGroups[key] = true
		groups = append(groups, candidate)
		for _, toolName := range candidate.ToolNames {
			if seenTools[toolName] {
				continue
			}
			seenTools[toolName] = true
			expanded = append(expanded, toolName)
		}
	}
	return groups, expanded, nil
}

func localHostToolGroupSelection(intent string, candidates []hostToolGroupCandidate) (hostToolGroupDecision, error) {
	entries := make([]domain.ToolCatalogEntry, 0, len(candidates))
	for _, candidate := range candidates {
		entries = append(entries, domain.ToolCatalogEntry{Name: candidate.Kind + ":" + candidate.ID, Description: candidate.Name + " " + candidate.Description})
	}
	matches := searchToolCatalog(entries, intent, 0)
	selectedGroups := make([]string, 0, len(matches))
	for _, match := range matches {
		kind, id, ok := strings.Cut(match.Name, ":")
		if ok {
			selectedGroups = append(selectedGroups, kind+"\x00"+id)
		}
	}
	groups, expanded, err := expandHostToolGroups(candidates, selectedGroups, true)
	return hostToolGroupDecision{Intent: hostResourceSelectionUse, Groups: groups, ToolNames: expanded}, err
}

func isToolInventoryIntent(intent string) bool {
	lower := strings.ToLower(intent)
	hasTool := strings.Contains(lower, "tool") || strings.Contains(lower, "capabilit") || strings.Contains(lower, "工具") || strings.Contains(lower, "能力")
	hasInventory := strings.Contains(lower, "available") || strings.Contains(lower, "list") || strings.Contains(lower, "inspect") || strings.Contains(lower, "what") ||
		strings.Contains(lower, "哪些") || strings.Contains(lower, "有什么") || strings.Contains(lower, "列出") || strings.Contains(lower, "列表") || strings.Contains(lower, "当前有") || strings.Contains(lower, "可调用")
	return hasTool && hasInventory
}

func (s *Service) hostInstructionCandidates(ctx context.Context, workspaceRoot string) ([]SkillResolveCandidate, []hostInstructionCandidate) {
	skillCandidates, _ := s.hostSkillCandidates(ctx, workspaceRoot)
	candidates := hostSkillInstructionCandidates(skillCandidates)
	if s.extensionSupervisor != nil {
		for _, resource := range s.extensionSupervisor.ContextCatalog() {
			resource := resource
			candidates = append(candidates, hostInstructionCandidate{
				Key: resource.Key, Kind: "context", Name: resource.Name,
				Description: resource.Description, Source: resource.ExtensionID, Context: &resource,
			})
		}
	}
	return skillCandidates, candidates
}

const skillGroupResourceKeyPrefix = "skill-group:"

func hostSkillInstructionCandidates(skillCandidates []SkillResolveCandidate) []hostInstructionCandidate {
	type groupState struct {
		index   int
		group   domain.ToolSelectionGroup
		members []SkillResolveCandidate
	}
	groups := map[string]*groupState{}
	candidates := make([]hostInstructionCandidate, 0, len(skillCandidates))
	for index := range skillCandidates {
		skill := skillCandidates[index]
		if skill.SelectionGroup == nil {
			skill := skill
			candidates = append(candidates, hostInstructionCandidate{
				Key: "skill:" + skill.Name, Kind: "skill", Name: skill.Name,
				Description: skill.Description, Source: skill.Source, Skill: &skill,
			})
			continue
		}
		group := *skill.SelectionGroup
		state := groups[group.ID]
		if state == nil {
			state = &groupState{index: len(candidates), group: group}
			groups[group.ID] = state
			candidates = append(candidates, hostInstructionCandidate{
				Key: skillGroupResourceKeyPrefix + group.ID, Kind: "skill", Name: group.Name,
				Description: group.Description, Source: "skill-group",
			})
		}
		state.members = append(state.members, skill)
	}
	for _, state := range groups {
		sort.Slice(state.members, func(i, j int) bool { return state.members[i].Name < state.members[j].Name })
		candidate := candidates[state.index]
		candidate.SkillGroup = state.members
		candidates[state.index] = candidate
	}
	return candidates
}

func (s *Service) hostSkillCandidates(ctx context.Context, workspaceRoot string) ([]SkillResolveCandidate, error) {
	result, err := s.ListSkills(ctx, domain.SkillListInput{WorkspaceRoot: strings.TrimSpace(workspaceRoot)})
	if err != nil {
		return nil, err
	}
	byName := map[string]SkillResolveCandidate{}
	for _, skill := range result.Entries {
		if !isModelResolvableSkillEntry(skill) {
			continue
		}
		byName[skill.Name] = SkillResolveCandidate{Name: skill.Name, Description: skill.Description, Scope: skill.Scope, Source: skill.Source, Status: "imported", SelectionGroup: skill.SelectionGroup}
	}
	out := make([]SkillResolveCandidate, 0, len(byName))
	for _, candidate := range byName {
		out = append(out, candidate)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *Service) resolveHostInstructionsWithAuxiliaryModel(ctx context.Context, intent, agentMode string, candidates []hostInstructionCandidate) hostInstructionDecision {
	decision := s.resolveHostResourcesWithAuxiliaryModel(ctx, intent, agentMode, nil, candidates)
	return hostInstructionDecision{Keys: decision.ResourceKeys, Reason: decision.Reason}
}

func (s *Service) resolveHostResourcesWithAuxiliaryModel(ctx context.Context, intent, agentMode string, toolCandidates []domain.ToolCatalogEntry, instructionCandidates []hostInstructionCandidate) hostResourceDecision {
	toolCatalog := make([]map[string]any, 0, len(toolCandidates))
	for _, entry := range toolCandidates {
		toolCatalog = append(toolCatalog, map[string]any{
			"name": entry.Name, "description": bounded(entry.Description, 400),
			"source": entry.Source, "sourceId": entry.SourceID, "category": entry.Category,
			"namespace": entry.Namespace, "capability": entry.Capability, "riskLevel": entry.RiskLevel,
		})
	}
	resourceCatalog := make([]map[string]any, 0, len(instructionCandidates))
	for _, candidate := range instructionCandidates {
		resourceCatalog = append(resourceCatalog, map[string]any{
			"key": candidate.Key, "kind": candidate.Kind, "name": candidate.Name,
			"description": bounded(candidate.Description, 600), "source": candidate.Source,
		})
	}
	payload, _ := json.MarshalIndent(map[string]any{
		"intent": intent, "agentMode": agentMode, "tools": toolCatalog, "resources": resourceCatalog,
	}, "", "  ")
	systemPrompt, promptErr := s.renderManagedPrompt("auxiliary.host_resources.system", nil)
	if promptErr != nil {
		systemPrompt = builtinPromptBody("auxiliary.host_resources.system")
	}
	messages := []domain.ChatMessage{{Role: "system", Text: systemPrompt}, {Role: "user", Text: string(payload)}}
	for _, model := range s.resolveAuxiliaryModels(ctx, nil) {
		reply, _, err := s.GenerateChatReply(ctx, messages, &model, "low", "default")
		if err != nil {
			continue
		}
		decision, err := parseHostResourceDecision(reply)
		if err == nil {
			return decision
		}
	}
	toolDecision, _ := localResourceResolve(ctx, ResourceResolveRequest{Intent: intent, AgentMode: agentMode, Candidates: toolCandidates})
	resourceDecision := localHostInstructionResolve(intent, instructionCandidates)
	return hostResourceDecision{
		ToolNames:    toolDecision.Names,
		ResourceKeys: resourceDecision.Keys,
		Reason:       "matched by local Host resource catalog search",
	}
}

func parseHostResourceDecision(raw string) (hostResourceDecision, error) {
	text := strings.TrimSpace(stripThinkBlocks(raw))
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	if start, end := strings.Index(text, "{"), strings.LastIndex(text, "}"); start >= 0 && end >= start {
		text = text[start : end+1]
	}
	var decoded struct {
		Tools     []string `json:"tools"`
		Resources []string `json:"resources"`
		Reason    string   `json:"reason"`
	}
	if err := json.Unmarshal([]byte(text), &decoded); err != nil {
		return hostResourceDecision{}, err
	}
	return hostResourceDecision{
		ToolNames:    normalizeHostResourceKeys(decoded.Tools),
		ResourceKeys: normalizeHostResourceKeys(decoded.Resources),
		Reason:       strings.TrimSpace(decoded.Reason),
	}, nil
}

func localHostInstructionResolve(intent string, candidates []hostInstructionCandidate) hostInstructionDecision {
	intentText := strings.ToLower(strings.TrimSpace(intent))
	intentTokens := tokenizeHostResourceText(intentText)
	type scoredCandidate struct {
		key   string
		score int
	}
	scored := make([]scoredCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		candidateText := strings.ToLower(candidate.Name + " " + candidate.Description + " " + candidate.Source)
		score := 0
		name := strings.ToLower(strings.TrimSpace(candidate.Name))
		if len([]rune(name)) > 1 && strings.Contains(intentText, name) {
			score += 4
		}
		candidateTokens := map[string]bool{}
		for _, token := range tokenizeHostResourceText(candidateText) {
			candidateTokens[token] = true
		}
		intentTokenSet := map[string]bool{}
		for _, token := range intentTokens {
			intentTokenSet[token] = true
			if candidateTokens[token] {
				score++
			}
		}
		for token := range candidateTokens {
			if intentTokenSet[token] {
				score++
			}
		}
		if score > 0 {
			scored = append(scored, scoredCandidate{key: candidate.Key, score: score})
		}
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].key < scored[j].key
		}
		return scored[i].score > scored[j].score
	})
	keys := make([]string, 0, len(scored))
	for _, candidate := range scored {
		keys = append(keys, candidate.key)
	}
	if len(keys) == 0 && isSkillInventoryIntent(intent) {
		for _, candidate := range candidates {
			if candidate.Skill == nil && len(candidate.SkillGroup) == 0 {
				continue
			}
			keys = append(keys, candidate.Key)
		}
	}
	return hostInstructionDecision{Keys: keys, Reason: "matched by local resource catalog search"}
}

func validateHostInstructionSelection(candidates []hostInstructionCandidate, keys []string) []hostInstructionCandidate {
	byKey := make(map[string]hostInstructionCandidate, len(candidates))
	for _, candidate := range candidates {
		byKey[candidate.Key] = candidate
	}
	seen := map[string]bool{}
	selected := make([]hostInstructionCandidate, 0, len(keys))
	for _, key := range normalizeHostResourceKeys(keys) {
		candidate, ok := byKey[key]
		if !ok || seen[key] {
			continue
		}
		seen[key] = true
		selected = append(selected, candidate)
	}
	return selected
}

func (s *Service) applySessionInstructionResources(ctx context.Context, sessionID string, workspaceRoot string, keys []string) (string, []hostResourceSelectionResource, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", nil, errors.New("sessionId is required")
	}
	_, candidates := s.hostInstructionCandidates(ctx, workspaceRoot)
	selected := validateHostInstructionSelection(candidates, keys)
	if len(selected) == 0 {
		return "", nil, nil
	}
	blocks := make([]string, 0, len(selected))
	summaries := make([]hostResourceSelectionResource, 0, len(selected))
	manager := s.ensureSkillManager()
	visibleSkills := make([]domain.SkillEntry, 0, len(selected))
	contextKeys := make([]string, 0, len(selected))
	for _, candidate := range selected {
		if candidate.Skill != nil {
			skill, err := manager.Resolve(ctx, "", candidate.Skill.Name, candidate.Skill.Scope)
			if err != nil || !skill.Enabled || !s.skillAvailableForUse(ctx, skill) {
				continue
			}
			visibleSkills = append(visibleSkills, s.decorateSkillEntry(ctx, skill))
			summaries = append(summaries, hostResourceSelectionResource{Kind: domain.SessionResourceSkill, ID: skill.Name, Name: skill.Name, ToolCount: 0})
			continue
		}
		if len(candidate.SkillGroup) > 0 {
			count := 0
			for _, member := range candidate.SkillGroup {
				skill, err := manager.Resolve(ctx, "", member.Name, member.Scope)
				if err != nil || !skill.Enabled || !s.skillAvailableForUse(ctx, skill) {
					continue
				}
				visibleSkills = append(visibleSkills, s.decorateSkillEntry(ctx, skill))
				count++
			}
			if count > 0 {
				summaries = append(summaries, hostResourceSelectionResource{
					Kind: domain.SessionResourceSkill, ID: strings.TrimPrefix(candidate.Key, skillGroupResourceKeyPrefix),
					Name: candidate.Name, ToolCount: 0,
				})
			}
			continue
		}
		if candidate.Context != nil {
			resource, ok := s.extensionContextResource(ctx, candidate.Context.ExtensionID, candidate.Context.ContextID)
			if !ok {
				continue
			}
			contextKeys = append(contextKeys, candidate.Context.Key)
			blocks = append(blocks, renderExtensionContextResource(resource))
			summaries = append(summaries, hostResourceSelectionResource{Kind: domain.SessionResourceExtension, ID: candidate.Context.Key, Name: candidate.Context.Name, ToolCount: 0})
		}
	}
	if len(visibleSkills) > 0 {
		sort.Slice(visibleSkills, func(i, j int) bool { return visibleSkills[i].Name < visibleSkills[j].Name })
		if _, err := s.rememberVisibleSkills(ctx, sessionID, visibleSkills); err != nil {
			return "", nil, err
		}
		blocks = append([]string{renderAvailableSkills(visibleSkills, nil)}, blocks...)
	}
	if len(contextKeys) > 0 {
		if _, err := s.rememberActiveExtensionContexts(ctx, sessionID, contextKeys); err != nil {
			return "", nil, err
		}
	}
	return strings.Join(blocks, "\n\n"), normalizeHostResourceSelectionResources(summaries), nil
}

func (s *Service) extensionContextResource(ctx context.Context, extensionID string, contextID string) (domain.ExtensionContextResource, bool) {
	_ = ctx
	if s == nil || s.extensionSupervisor == nil {
		return domain.ExtensionContextResource{}, false
	}
	resources, err := s.extensionSupervisor.ContextResources(extensionID)
	if err != nil {
		return domain.ExtensionContextResource{}, false
	}
	for _, resource := range resources {
		if resource.ID == contextID {
			return resource, true
		}
	}
	return domain.ExtensionContextResource{}, false
}

func renderExtensionContextResource(resource domain.ExtensionContextResource) string {
	return `<extension_context extension="` + xmlEscape(resource.ExtensionID) + `" id="` + xmlEscape(resource.ID) + `" kind="` + xmlEscape(resource.Kind) + `" sha256="` + xmlEscape(resource.SHA256) + `">` +
		"\n" + strings.TrimSpace(resource.Content) + "\n</extension_context>"
}

func appendHostPreSnapshotContext(messages []domain.ChatMessage, content string) []domain.ChatMessage {
	content = strings.TrimSpace(content)
	if content == "" {
		return append([]domain.ChatMessage(nil), messages...)
	}
	message := domain.ChatMessage{Role: domain.EventRoleSystem, Text: "<host_selected_resources>\n" + content + "\n</host_selected_resources>"}
	out := make([]domain.ChatMessage, 0, len(messages)+1)
	if len(messages) > 0 && messages[0].Role == domain.EventRoleSystem {
		out = append(out, messages[0], message)
		out = append(out, messages[1:]...)
		return out
	}
	out = append(out, message)
	return append(out, messages...)
}

func isSkillInventoryIntent(intent string) bool {
	lower := strings.ToLower(intent)
	hasSkill := strings.Contains(lower, "skill") || strings.Contains(lower, "技能")
	hasInventory := strings.Contains(lower, "available") || strings.Contains(lower, "installed") || strings.Contains(lower, "list") ||
		strings.Contains(lower, "哪些") || strings.Contains(lower, "有什么") || strings.Contains(lower, "列出") || strings.Contains(lower, "列表") || strings.Contains(lower, "当前有")
	return hasSkill && hasInventory
}

func tokenizeHostResourceText(value string) []string {
	fields := strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	seen := map[string]bool{}
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" || seen[field] || isHostResourceStopword(field) {
			continue
		}
		if len([]rune(field)) <= 2 && !isShortHostResourceToken(field) {
			continue
		}
		seen[field] = true
		out = append(out, field)
	}
	return out
}

func isShortHostResourceToken(token string) bool {
	switch token {
	case "ai", "go", "js", "ts", "ui", "ux":
		return true
	default:
		return false
	}
}

func isHostResourceStopword(token string) bool {
	switch token {
	case "a", "an", "and", "are", "as", "at", "be", "by", "for", "from", "how", "in", "into", "is", "it", "of", "on", "or", "the", "this", "to", "use", "what", "when", "which", "with", "当前", "哪些", "有什么", "列出", "列表", "技能":
		return true
	default:
		return false
	}
}

func normalizeHostResourceKeys(keys []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	return out
}
