package app

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"
	"unicode"

	"aivo/core/domain"
)

const (
	hostInstructionSelectionLimit = 4
	hostPreCallContextLimit       = 12000
	hostCatalogPreparationTimeout = 8 * time.Second
)

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
	if s.pluginManager != nil {
		for key := range s.pluginManager.PrepareEnabled(prepareCtx) {
			failed[key] = true
		}
	}
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
) hostPreCallResolution {
	toolActivations, toolCandidates := s.preCallToolCandidates(ctx, sessionID, turnID, registry, specs)
	skillCandidates, instructionCandidates := s.hostInstructionCandidates(ctx, workspaceRoot)
	if isSkillInventoryIntent(intent) {
		return hostPreCallResolution{ToolActivations: toolActivations, Context: renderHostSkillInventory(skillCandidates)}
	}
	if strings.TrimSpace(intent) == "" || (len(toolCandidates) == 0 && len(instructionCandidates) == 0) {
		return hostPreCallResolution{ToolActivations: toolActivations}
	}
	decision := s.resolveHostResourcesWithAuxiliaryModel(ctx, intent, agentMode, toolCandidates, instructionCandidates)
	for _, entry := range validateToolResolveSelection(toolCandidates, decision.ToolNames, 4) {
		toolActivations[entry.Name] = "auxiliary"
	}
	selected := validateHostInstructionSelection(instructionCandidates, decision.ResourceKeys, hostInstructionSelectionLimit)
	skillInstructionKeys := validateHostSkillInstructionSelection(selected, decision.SkillInstructionKeys)
	return hostPreCallResolution{
		ToolActivations: toolActivations,
		Context:         s.renderSelectedHostInstructions(ctx, sessionID, workspaceRoot, selected, skillInstructionKeys),
	}
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
		"intent": intent, "agentMode": agentMode, "maxTools": 4, "maxResources": hostInstructionSelectionLimit,
		"tools": toolCatalog, "resources": resourceCatalog,
	}, "", "  ")
	messages := []domain.ChatMessage{
		{Role: "system", Text: "Act as the Host pre-call resource resolver. Select only tools and instruction/context resources that directly help the user's current request. Return strict JSON: {\"tools\":[\"exact_tool_name\"],\"resources\":[\"exact_resource_key\"],\"skillInstructions\":[\"exact_selected_skill_key\"],\"reason\":\"short reason\"}. resources selects what the Host will materialize. skillInstructions must be an exact subset of the selected skill: resource keys. Include a Skill key there only when the primary model must follow that Skill to perform the task; omit it when a canonical summary is sufficient for listing or explanation. Never invent names or keys. Respect both maxima and return empty arrays when no clear match exists. Selection grants no authority and performs no action."},
		{Role: "user", Text: string(payload)},
	}
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
	toolDecision, _ := localToolResolve(ctx, ToolResolveRequest{Intent: intent, MaxTools: 4, AgentMode: agentMode, Candidates: toolCandidates})
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
			blocks = append(blocks, renderSkillModelOutput(skill, content, files))
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
