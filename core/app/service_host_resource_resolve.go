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
	hostInstructionSelectionLimit = 4
	hostToolSelectionLimit        = 8
	hostPreCallContextLimit       = 12000
	hostCatalogPreparationTimeout = 8 * time.Second
)

func (s *Service) resolveSessionToolReplacement(ctx context.Context, request ToolResolveRequest) (ToolResolveDecision, error) {
	candidates, err := s.filterGloballyVisibleToolCatalogEntries(ctx, request.Candidates)
	if err != nil {
		return ToolResolveDecision{}, err
	}
	manual := s.rememberedDeferredTools(ctx, request.SessionID)
	filtered := make([]domain.ToolCatalogEntry, 0, len(candidates))
	for _, candidate := range candidates {
		if !manual[candidate.Name] {
			filtered = append(filtered, candidate)
		}
	}
	request.Candidates = filtered
	if len(request.Candidates) == 0 {
		return ToolResolveDecision{Reason: "no globally visible eligible automatic tool candidates"}, nil
	}
	groups := hostToolGroupCandidates(request.Candidates)
	if len(groups) == 0 {
		return ToolResolveDecision{Reason: "no eligible grouped or individual tool candidates"}, nil
	}
	decision := s.resolveHostToolGroupsWithAuxiliaryModel(ctx, request.Intent, groups, false)
	return ToolResolveDecision{Names: normalizeDeferredToolNames(decision.ToolNames), Reason: decision.Reason}, nil
}

type hostToolGroupCandidate struct {
	Kind        string
	ID          string
	Name        string
	Description string
	ToolNames   []string
	Grouped     bool
}

type hostToolSelectionResource struct {
	Kind      string
	ID        string
	Name      string
	ToolCount int
}

type hostToolSelectionIntent string

const (
	hostToolSelectionInspect hostToolSelectionIntent = "inspect"
	hostToolSelectionUse     hostToolSelectionIntent = "use"
)

type hostToolGroupDecision struct {
	Intent    hostToolSelectionIntent
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
	Context     *extensionContextCandidate
}

type hostInstructionDecision struct {
	Keys                 []string
	SkillInstructionKeys []string
	Reason               string
}

type hostResourceDecision struct {
	ToolNames            []string
	ResourceKeys         []string
	SkillInstructionKeys []string
	Reason               string
}

type hostPreCallResolution struct {
	ToolActivations map[string]string
	Context         string
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

func (s *Service) resolveHostPreCallResources(
	ctx context.Context,
	sessionID string,
	turnID string,
	intent string,
	agentMode string,
	workspaceRoot string,
	registry *Registry,
	specs []domain.ToolSpec,
) (hostPreCallResolution, error) {
	toolActivations, toolCandidates := s.preCallToolCandidates(ctx, sessionID, turnID, registry, specs)
	filteredToolCandidates, eligibilityErr := s.filterGloballyVisibleToolCatalogEntries(ctx, toolCandidates)
	if eligibilityErr == nil {
		toolCandidates = filteredToolCandidates
	} else {
		toolCandidates = nil
	}
	_, autoInitialized := s.autoSelectedTools(ctx, sessionID)
	skillCandidates, instructionCandidates := s.hostInstructionCandidates(ctx, workspaceRoot)
	if isSkillInventoryIntent(intent) {
		if !autoInitialized && eligibilityErr == nil {
			_ = s.replaceAutoSelectedTools(ctx, sessionID, nil)
		}
		return hostPreCallResolution{ToolActivations: toolActivations, Context: renderHostSkillInventory(skillCandidates)}, nil
	}
	if strings.TrimSpace(intent) == "" || (autoInitialized && len(toolCandidates) == 0 && len(instructionCandidates) == 0) {
		return hostPreCallResolution{ToolActivations: toolActivations}, nil
	}
	toolGroups := hostToolGroupCandidates(toolCandidates)
	var progressEventID string
	if !autoInitialized && eligibilityErr == nil && len(toolGroups) > 0 && len(s.resolveAuxiliaryModels(ctx, nil)) > 0 {
		event, err := s.startInitialToolSelection(ctx, sessionID, turnID)
		if err != nil {
			return hostPreCallResolution{}, err
		}
		progressEventID = event.ID
	}
	toolDecision := s.resolveHostToolGroupsWithAuxiliaryModel(ctx, intent, toolGroups, true)
	if toolDecision.Intent == hostToolSelectionInspect {
		toolDecision.ToolNames = make([]string, 0, len(toolCandidates))
		for _, entry := range toolCandidates {
			toolDecision.ToolNames = append(toolDecision.ToolNames, entry.Name)
		}
	}
	selectedTools := validateToolResolveSelection(toolCandidates, toolDecision.ToolNames)
	if !autoInitialized && eligibilityErr == nil {
		names := make([]string, 0, len(selectedTools))
		for _, entry := range selectedTools {
			names = append(names, entry.Name)
			activationSource := "automatic"
			if toolDecision.Intent == hostToolSelectionInspect {
				activationSource = "request"
			}
			toolActivations[entry.Name] = activationSource
		}
		committedNames := normalizeDeferredToolNames(names)
		selectedResources := hostToolSelectionResources(toolDecision.Groups, committedNames)
		persistedNames := committedNames
		if toolDecision.Intent == hostToolSelectionInspect {
			persistedNames = nil
		}
		if err := s.replaceAutoSelectedTools(ctx, sessionID, persistedNames); err != nil {
			for _, name := range names {
				delete(toolActivations, name)
			}
			if progressEventID != "" {
				if _, updateErr := s.failInitialToolSelection(ctx, progressEventID); updateErr != nil {
					return hostPreCallResolution{}, updateErr
				}
			}
		} else {
			lifetime := "conversation"
			if toolDecision.Intent == hostToolSelectionInspect {
				lifetime = "request"
			}
			if progressEventID != "" {
				if _, err := s.completeInitialToolSelection(ctx, progressEventID, selectedResources, lifetime); err != nil {
					return hostPreCallResolution{}, err
				}
			} else if err := s.recordInitialToolSelection(ctx, sessionID, turnID, selectedResources, lifetime); err != nil {
				return hostPreCallResolution{}, err
			}
		}
	}
	instructionDecision := hostInstructionDecision{}
	if len(instructionCandidates) > 0 {
		instructionDecision = s.resolveHostInstructionsWithAuxiliaryModel(ctx, intent, agentMode, instructionCandidates)
	}
	selected := validateHostInstructionSelection(instructionCandidates, instructionDecision.Keys, hostInstructionSelectionLimit)
	skillInstructionKeys := validateHostSkillInstructionSelection(selected, instructionDecision.SkillInstructionKeys)
	return hostPreCallResolution{
		ToolActivations: toolActivations,
		Context:         s.renderSelectedHostInstructions(ctx, sessionID, workspaceRoot, selected, skillInstructionKeys),
	}, nil
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
			kind = hostToolSelectionResourceKind(entry)
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

func hostToolSelectionResourceKind(entry domain.ToolCatalogEntry) string {
	if entry.Source == domain.ToolSourceMCP || entry.Category == "mcp" {
		return domain.SessionResourceMCP
	}
	if entry.Source == domain.ToolSourceExtension && !strings.HasPrefix(strings.TrimSpace(entry.SourceID), "aivo.") {
		return domain.SessionResourceExtension
	}
	return domain.SessionResourceTool
}

func hostToolSelectionResources(groups []hostToolGroupCandidate, selectedToolNames []string) []hostToolSelectionResource {
	selected := make(map[string]bool, len(selectedToolNames))
	for _, name := range selectedToolNames {
		selected[name] = true
	}
	resources := make([]hostToolSelectionResource, 0, len(groups))
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
		resources = append(resources, hostToolSelectionResource{
			Kind: group.Kind, ID: group.ID, Name: group.Name, ToolCount: count,
		})
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
	prompt, err := renderPromptTemplate(builtinPromptBody("auxiliary.host_tool_groups.user"), map[string]string{
		"intent": boundedHostToolGroupText(intent, 4000), "candidates": strings.Join(lines, "\n"),
	})
	if err != nil {
		return ""
	}
	return prompt
}

func (s *Service) resolveHostToolGroupsWithAuxiliaryModel(ctx context.Context, intent string, candidates []hostToolGroupCandidate, allowInspect bool) hostToolGroupDecision {
	if strings.TrimSpace(intent) == "" || len(candidates) == 0 {
		return hostToolGroupDecision{Intent: hostToolSelectionUse, Reason: "no eligible tool resources"}
	}
	selectionRule := "Classify intent as use and select only grouped or individual tool resources directly needed to perform the requested action."
	intentShape := "\"use\""
	if allowInspect {
		selectionRule = "Classify intent as inspect only when the user asks to list, inspect, explain, or compare available tools/capabilities without asking to perform an action; inspect requires resources:[] and the Host will expose the complete eligible catalog once. Otherwise classify as use and select only grouped or individual resources directly needed to perform the requested action."
		intentShape = "\"inspect\"|\"use\""
	}
	systemPrompt, systemErr := s.renderManagedPrompt("auxiliary.host_tool_groups.system", map[string]string{"selection_rule": selectionRule, "intent_shape": intentShape})
	candidateLines := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidateLines = append(candidateLines, renderHostToolSelectionCandidate(candidate))
	}
	userPrompt, userErr := s.renderManagedPrompt("auxiliary.host_tool_groups.user", map[string]string{"intent": boundedHostToolGroupText(intent, 4000), "candidates": strings.Join(candidateLines, "\n")})
	if systemErr != nil || userErr != nil {
		decision, _ := localHostToolGroupSelection(intent, candidates, allowInspect)
		decision.Reason = "matched by bounded local Host tool-group search"
		return decision
	}
	messages := []domain.ChatMessage{{Role: "system", Text: systemPrompt}, {Role: "user", Text: userPrompt}}
	for _, model := range s.resolveAuxiliaryModels(ctx, nil) {
		reply, _, err := s.GenerateChatReply(ctx, messages, &model, "low", "default")
		if err != nil {
			continue
		}
		decision, err := parseAndExpandHostToolGroupSelection(reply, candidates, hostToolSelectionLimit, allowInspect)
		if err == nil {
			decision.Reason = "selected by auxiliary Host tool-group resolver"
			return decision
		}
	}
	decision, err := localHostToolGroupSelection(intent, candidates, allowInspect)
	if err != nil {
		if allowInspect && isToolInventoryIntent(intent) {
			return hostToolGroupDecision{Intent: hostToolSelectionInspect, Reason: err.Error()}
		}
		return hostToolGroupDecision{Intent: hostToolSelectionUse, Reason: err.Error()}
	}
	decision.Reason = "matched by bounded local Host tool-group search"
	return decision
}

func renderHostToolSelectionCandidate(candidate hostToolGroupCandidate) string {
	return candidate.Kind + ":" + candidate.ID + "：" + sanitizeHostToolGroupText(candidate.Name, 100) + "｜" + sanitizeHostToolGroupText(candidate.Description, 4000)
}

func parseAndExpandHostToolGroupSelection(raw string, candidates []hostToolGroupCandidate, limit int, allowInspect bool) (hostToolGroupDecision, error) {
	text := strings.TrimSpace(raw)
	if text == "" || !strings.HasPrefix(text, "{") || !strings.HasSuffix(text, "}") {
		return hostToolGroupDecision{}, errors.New("tool-group selection must be a strict JSON object")
	}
	var payload struct {
		Intent    string `json:"intent"`
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
	selectionIntent := hostToolSelectionIntent(payload.Intent)
	if selectionIntent != hostToolSelectionInspect && selectionIntent != hostToolSelectionUse {
		return hostToolGroupDecision{}, errors.New("tool-group selection contains an invalid intent")
	}
	if payload.Resources == nil {
		return hostToolGroupDecision{}, errors.New("tool selection is missing resources")
	}
	resources := *payload.Resources
	if selectionIntent == hostToolSelectionInspect {
		if !allowInspect {
			return hostToolGroupDecision{}, errors.New("tool inspection is unavailable for persistent replacement")
		}
		if len(resources) != 0 {
			return hostToolGroupDecision{}, errors.New("tool inspection must not select partial groups")
		}
		groups, toolNames, err := expandHostToolGroups(candidates, nil, false)
		if err != nil {
			return hostToolGroupDecision{}, err
		}
		return hostToolGroupDecision{Intent: selectionIntent, Groups: groups, ToolNames: toolNames}, nil
	}
	if limit <= 0 || limit > hostToolSelectionLimit {
		limit = hostToolSelectionLimit
	}
	if len(resources) > limit {
		return hostToolGroupDecision{}, errors.New("tool selection exceeds the resource limit")
	}
	keys := make([]string, 0, len(resources))
	for _, resource := range resources {
		if resource.Kind != domain.SessionResourceMCP && resource.Kind != domain.SessionResourceExtension && resource.Kind != domain.SessionResourceTool {
			return hostToolGroupDecision{}, errors.New("tool selection contains an invalid resource kind")
		}
		if resource.ID == "" || strings.TrimSpace(resource.ID) != resource.ID || sanitizeHostToolGroupText(resource.ID, 512) != resource.ID {
			return hostToolGroupDecision{}, errors.New("tool selection contains an invalid resource id")
		}
		keys = append(keys, resource.Kind+"\x00"+resource.ID)
	}
	groups, toolNames, err := expandHostToolGroups(candidates, keys, true)
	if err != nil {
		return hostToolGroupDecision{}, err
	}
	return hostToolGroupDecision{Intent: selectionIntent, Groups: groups, ToolNames: toolNames}, nil
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

func localHostToolGroupSelection(intent string, candidates []hostToolGroupCandidate, allowInspect bool) (hostToolGroupDecision, error) {
	if allowInspect && isToolInventoryIntent(intent) {
		groups, toolNames, err := expandHostToolGroups(candidates, nil, false)
		return hostToolGroupDecision{Intent: hostToolSelectionInspect, Groups: groups, ToolNames: toolNames}, err
	}
	entries := make([]domain.ToolCatalogEntry, 0, len(candidates))
	for _, candidate := range candidates {
		entries = append(entries, domain.ToolCatalogEntry{Name: candidate.Kind + ":" + candidate.ID, Description: candidate.Name + " " + candidate.Description})
	}
	matches := searchToolCatalog(entries, intent, hostToolSelectionLimit)
	selectedGroups := make([]string, 0, len(matches))
	for _, match := range matches {
		kind, id, ok := strings.Cut(match.Name, ":")
		if ok {
			selectedGroups = append(selectedGroups, kind+"\x00"+id)
		}
	}
	groups, expanded, err := expandHostToolGroups(candidates, selectedGroups, true)
	return hostToolGroupDecision{Intent: hostToolSelectionUse, Groups: groups, ToolNames: expanded}, err
}

func isToolInventoryIntent(intent string) bool {
	lower := strings.ToLower(intent)
	hasTool := strings.Contains(lower, "tool") || strings.Contains(lower, "capabilit") || strings.Contains(lower, "工具") || strings.Contains(lower, "能力")
	hasInventory := strings.Contains(lower, "available") || strings.Contains(lower, "list") || strings.Contains(lower, "inspect") || strings.Contains(lower, "what") ||
		strings.Contains(lower, "哪些") || strings.Contains(lower, "有什么") || strings.Contains(lower, "列出") || strings.Contains(lower, "列表") || strings.Contains(lower, "当前有") || strings.Contains(lower, "可调用")
	return hasTool && hasInventory
}

func (s *Service) preCallInstructionContext(ctx context.Context, sessionID, intent, agentMode, workspaceRoot string) string {
	skillCandidates, candidates := s.hostInstructionCandidates(ctx, workspaceRoot)
	if isSkillInventoryIntent(intent) {
		return renderHostSkillInventory(skillCandidates)
	}
	if strings.TrimSpace(intent) == "" || len(candidates) == 0 {
		return ""
	}
	decision := s.resolveHostInstructionsWithAuxiliaryModel(ctx, intent, agentMode, candidates)
	selected := validateHostInstructionSelection(candidates, decision.Keys, hostInstructionSelectionLimit)
	if len(selected) == 0 {
		return ""
	}
	skillInstructionKeys := validateHostSkillInstructionSelection(selected, decision.SkillInstructionKeys)
	return s.renderSelectedHostInstructions(ctx, sessionID, workspaceRoot, selected, skillInstructionKeys)
}

func (s *Service) hostInstructionCandidates(ctx context.Context, workspaceRoot string) ([]SkillResolveCandidate, []hostInstructionCandidate) {
	skillCandidates, _ := s.hostSkillCandidates(ctx, workspaceRoot)
	candidates := make([]hostInstructionCandidate, 0, len(skillCandidates))
	for index := range skillCandidates {
		skill := skillCandidates[index]
		candidates = append(candidates, hostInstructionCandidate{
			Key: "skill:" + skill.Name, Kind: "skill", Name: skill.Name,
			Description: skill.Description, Source: skill.Source, Skill: &skill,
		})
	}
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

func (s *Service) hostSkillCandidates(ctx context.Context, workspaceRoot string) ([]SkillResolveCandidate, error) {
	result, err := s.ListSkills(ctx, domain.SkillListInput{WorkspaceRoot: strings.TrimSpace(workspaceRoot)})
	if err != nil {
		return nil, err
	}
	byName := map[string]SkillResolveCandidate{}
	for _, skill := range result.Entries {
		if !skill.Enabled || strings.TrimSpace(skill.Description) == "" {
			continue
		}
		byName[skill.Name] = SkillResolveCandidate{Name: skill.Name, Description: skill.Description, Scope: skill.Scope, Source: skill.Source, Status: "imported"}
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
	return hostInstructionDecision{Keys: decision.ResourceKeys, SkillInstructionKeys: decision.SkillInstructionKeys, Reason: decision.Reason}
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
		"intent": intent, "agentMode": agentMode, "maxTools": hostToolSelectionLimit, "maxResources": hostInstructionSelectionLimit,
		"tools": toolCatalog, "resources": resourceCatalog,
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
	toolDecision, _ := localToolResolve(ctx, ToolResolveRequest{Intent: intent, MaxTools: hostToolSelectionLimit, AgentMode: agentMode, Candidates: toolCandidates})
	resourceDecision := localHostInstructionResolve(intent, instructionCandidates, hostInstructionSelectionLimit)
	return hostResourceDecision{
		ToolNames:            toolDecision.Names,
		ResourceKeys:         resourceDecision.Keys,
		SkillInstructionKeys: skillInstructionKeysForSelection(instructionCandidates, resourceDecision.Keys),
		Reason:               "matched by bounded local Host resource catalog search",
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
		Tools             []string `json:"tools"`
		Resources         []string `json:"resources"`
		SkillInstructions []string `json:"skillInstructions"`
		Reason            string   `json:"reason"`
	}
	if err := json.Unmarshal([]byte(text), &decoded); err != nil {
		return hostResourceDecision{}, err
	}
	return hostResourceDecision{
		ToolNames:            normalizeHostResourceKeys(decoded.Tools),
		ResourceKeys:         normalizeHostResourceKeys(decoded.Resources),
		SkillInstructionKeys: normalizeHostResourceKeys(decoded.SkillInstructions),
		Reason:               strings.TrimSpace(decoded.Reason),
	}, nil
}

func localHostInstructionResolve(intent string, candidates []hostInstructionCandidate, limit int) hostInstructionDecision {
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
		for _, token := range intentTokens {
			if len([]rune(token)) > 1 && strings.Contains(candidateText, token) {
				score++
			}
		}
		for _, token := range tokenizeHostResourceText(candidateText) {
			if len([]rune(token)) > 1 && strings.Contains(intentText, token) {
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
	keys := make([]string, 0, limit)
	for _, candidate := range scored {
		keys = append(keys, candidate.key)
		if len(keys) >= limit {
			break
		}
	}
	return hostInstructionDecision{Keys: keys, Reason: "matched by bounded local resource catalog search"}
}

func validateHostInstructionSelection(candidates []hostInstructionCandidate, keys []string, limit int) []hostInstructionCandidate {
	byKey := make(map[string]hostInstructionCandidate, len(candidates))
	for _, candidate := range candidates {
		byKey[candidate.Key] = candidate
	}
	seen := map[string]bool{}
	selected := make([]hostInstructionCandidate, 0, limit)
	for _, key := range normalizeHostResourceKeys(keys) {
		candidate, ok := byKey[key]
		if !ok || seen[key] {
			continue
		}
		seen[key] = true
		selected = append(selected, candidate)
		if len(selected) >= limit {
			break
		}
	}
	return selected
}

func skillInstructionKeysForSelection(candidates []hostInstructionCandidate, selectedKeys []string) []string {
	byKey := make(map[string]hostInstructionCandidate, len(candidates))
	for _, candidate := range candidates {
		byKey[candidate.Key] = candidate
	}
	keys := make([]string, 0, len(selectedKeys))
	for _, key := range normalizeHostResourceKeys(selectedKeys) {
		if candidate, ok := byKey[key]; ok && candidate.Skill != nil {
			keys = append(keys, key)
		}
	}
	return keys
}

func validateHostSkillInstructionSelection(selected []hostInstructionCandidate, keys []string) map[string]bool {
	selectedSkills := make(map[string]bool, len(selected))
	for _, candidate := range selected {
		if candidate.Skill != nil {
			selectedSkills[candidate.Key] = true
		}
	}
	validated := map[string]bool{}
	for _, key := range normalizeHostResourceKeys(keys) {
		if selectedSkills[key] {
			validated[key] = true
		}
	}
	return validated
}

func (s *Service) renderSelectedHostInstructions(ctx context.Context, sessionID, workspaceRoot string, selected []hostInstructionCandidate, skillInstructionKeys map[string]bool) string {
	activeNames := map[string]bool{}
	if strings.TrimSpace(sessionID) != "" {
		_, activeSkills := s.activeSkills(ctx, sessionID)
		for _, skill := range activeSkills {
			activeNames[skill.Name] = true
		}
	}
	blocks := make([]string, 0, len(selected))
	selectedContexts := map[string]map[string]bool{}
	manager := s.ensureSkillManager()
	for _, candidate := range selected {
		if candidate.Skill != nil {
			if activeNames[candidate.Skill.Name] {
				continue
			}
			skill, err := manager.Resolve(ctx, "", candidate.Skill.Name, workspaceRoot)
			if err != nil || !skill.Enabled {
				continue
			}
			blocks = append(blocks, renderHostSkillSummary(skill))
			if !skillInstructionKeys[candidate.Key] {
				continue
			}
			content, err := manager.ReadContent(skill)
			if err != nil {
				continue
			}
			files, _ := manager.SupportingFiles(skill, skillSupportingFileLimit)
			blocks = append(blocks, renderSkillModelOutputWithSnapshot(s.currentPromptSnapshot(), skill, content, files))
			continue
		}
		if candidate.Context != nil {
			if selectedContexts[candidate.Context.ExtensionID] == nil {
				selectedContexts[candidate.Context.ExtensionID] = map[string]bool{}
			}
			selectedContexts[candidate.Context.ExtensionID][candidate.Context.ContextID] = true
		}
	}
	extensionIDs := make([]string, 0, len(selectedContexts))
	for extensionID := range selectedContexts {
		extensionIDs = append(extensionIDs, extensionID)
	}
	sort.Strings(extensionIDs)
	for _, extensionID := range extensionIDs {
		resources, err := s.extensionSupervisor.ContextResources(extensionID)
		if err != nil {
			continue
		}
		for _, resource := range resources {
			if !selectedContexts[extensionID][resource.ID] {
				continue
			}
			blocks = append(blocks, renderExtensionContextResource(resource))
		}
	}
	return bounded(strings.Join(blocks, "\n\n"), hostPreCallContextLimit)
}

func renderHostSkillSummary(skill domain.SkillEntry) string {
	return `<skill_summary name="` + xmlEscape(skill.Name) + `" scope="` + xmlEscape(skill.Scope) + `" source="` + xmlEscape(skill.Source) + `">` +
		"\n" + xmlEscape(strings.TrimSpace(skill.Description)) + "\n</skill_summary>"
}

func renderExtensionContextResource(resource domain.ExtensionContextResource) string {
	return `<extension_context extension="` + xmlEscape(resource.ExtensionID) + `" id="` + xmlEscape(resource.ID) + `" kind="` + xmlEscape(resource.Kind) + `" sha256="` + xmlEscape(resource.SHA256) + `">` +
		"\n" + strings.TrimSpace(resource.Content) + "\n</extension_context>"
}

func renderHostSkillInventory(skills []SkillResolveCandidate) string {
	if len(skills) == 0 {
		return "<available_skills>\n</available_skills>"
	}
	lines := []string{"The Host found these imported and enabled Skills. Pending imports and disabled Skills are excluded.", "<available_skills>"}
	for _, skill := range skills {
		lines = append(lines,
			"  <skill>",
			"    <name>"+xmlEscape(skill.Name)+"</name>",
			"    <description>"+xmlEscape(skill.Description)+"</description>",
			"  </skill>",
		)
	}
	lines = append(lines, "</available_skills>")
	return bounded(strings.Join(lines, "\n"), hostPreCallContextLimit)
}

func appendHostPreCallContext(messages []domain.ChatMessage, content string) []domain.ChatMessage {
	content = strings.TrimSpace(content)
	if content == "" {
		return append([]domain.ChatMessage(nil), messages...)
	}
	message := domain.ChatMessage{Role: domain.EventRoleSystem, Text: "<host_preactivated_resources>\n" + content + "\n</host_preactivated_resources>"}
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
	return strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
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
