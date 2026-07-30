package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"aivo/core/domain"
)

const SkillToolName = "skill"

type SkillResolveCandidate struct {
	Name        string
	Description string
	Scope       string
	Source      string
	Status      string
}

type SkillResolveRequest struct {
	Intent     string
	MaxSkills  int
	SessionID  string
	TurnID     string
	AgentMode  string
	Candidates []SkillResolveCandidate
}

type SkillResolveDecision struct {
	Names  []string
	Reason string
}

type SkillResolveFunc func(context.Context, SkillResolveRequest) (SkillResolveDecision, error)

type SkillLoadTool struct {
	service *Service
	resolve SkillResolveFunc
}

func NewSkillLoadTool(service *Service, resolvers ...SkillResolveFunc) *SkillLoadTool {
	tool := &SkillLoadTool{service: service}
	if len(resolvers) > 0 {
		tool.resolve = resolvers[0]
	}
	return tool
}

func (t *SkillLoadTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name: SkillToolName,
		Description: strings.Join([]string{
			"Discover and activate imported, enabled Agent Skills using progressive disclosure.",
			"First use mode=discover with the user's intent. It privately filters the catalog and returns only candidate names and descriptions. Review those descriptions, then call mode=activate with the exact names that apply before continuing the task.",
			"Use mode=list only when the user asks which skills are available. Activation returns the full SKILL.md instructions and bundled resource paths; activated skills remain effective for the session.",
		}, "\n\n"),
		Capability: "skill.load", Category: "skill", RiskLevel: "low", Toolsets: []string{"safe", "coding"},
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{
			"mode":      map[string]any{"type": "string", "enum": []string{"discover", "activate", "list"}, "description": "Use discover before activate. Use list only for catalog questions."},
			"intent":    map[string]any{"type": "string", "description": "Concise current task used for private candidate filtering in discover mode."},
			"names":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Exact candidate skill names selected by the model for activate mode."},
			"maxSkills": map[string]any{"type": "integer", "minimum": 1, "maximum": 8, "description": "Maximum candidates returned by discover. Defaults to 3."},
		}, "additionalProperties": false},
	}
}

func (t *SkillLoadTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	if t == nil || t.service == nil {
		return toolFailure(execCtx.ToolCallID, SkillToolName, "skill_service_unavailable", "skill service is unavailable")
	}
	var input struct {
		Mode      string   `json:"mode"`
		Intent    string   `json:"intent"`
		Names     []string `json:"names"`
		MaxSkills int      `json:"maxSkills"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolError(SkillToolName, errors.New("invalid skill arguments"))
	}
	input.Mode = strings.ToLower(strings.TrimSpace(input.Mode))
	input.Intent = strings.TrimSpace(input.Intent)
	input.Names = normalizeSkillNames(input.Names)
	if input.Mode == "" {
		if len(input.Names) > 0 && input.Intent == "" {
			input.Mode = "activate"
		} else {
			input.Mode = "discover"
		}
	}
	if input.MaxSkills <= 0 {
		input.MaxSkills = 3
	}
	if input.MaxSkills > 8 {
		input.MaxSkills = 8
	}
	candidates, err := t.service.skillResolveCandidates(ctx)
	if err != nil {
		return toolFailure(execCtx.ToolCallID, SkillToolName, "skill_catalog_failed", err.Error())
	}
	if input.Mode == "list" {
		return skillCatalogResult(execCtx.ToolCallID, "listed", candidates, "")
	}
	switch input.Mode {
	case "activate":
		if len(input.Names) == 0 {
			return toolError(SkillToolName, errors.New("names are required for activate mode"))
		}
		return t.activate(ctx, execCtx, candidates, input.Names)
	case "discover":
		if input.Intent == "" {
			return toolError(SkillToolName, errors.New("intent is required for discover mode"))
		}
		resolver := t.resolve
		if resolver == nil {
			resolver = localSkillResolve
		}
		decision, resolveErr := resolver(ctx, SkillResolveRequest{Intent: input.Intent, MaxSkills: input.MaxSkills, SessionID: execCtx.SessionID, TurnID: execCtx.TurnID, AgentMode: execCtx.AgentMode, Candidates: candidates})
		if resolveErr != nil {
			return toolFailure(execCtx.ToolCallID, SkillToolName, "skill_resolve_failed", resolveErr.Error())
		}
		names := validateSkillResolveSelection(candidates, decision.Names, input.MaxSkills)
		selected := skillCandidatesByName(candidates, names)
		return skillCatalogResult(execCtx.ToolCallID, "discovered", selected, strings.TrimSpace(decision.Reason))
	default:
		return toolError(SkillToolName, errors.New("mode must be discover, activate, or list"))
	}
}

func (t *SkillLoadTool) activate(ctx context.Context, execCtx domain.ToolExecutionContext, candidates []SkillResolveCandidate, requested []string) domain.ToolResult {
	names := validateSkillResolveSelection(candidates, requested, 8)
	if len(names) == 0 {
		structured := map[string]any{"status": "no_match", "skills": []string{}, "reason": "no requested imported and enabled skill exists"}
		raw, _ := json.MarshalIndent(structured, "", "  ")
		return domain.ToolResult{Name: SkillToolName, CallID: execCtx.ToolCallID, OK: true, Content: string(raw), ModelContent: string(raw), Structured: structured}
	}
	blocks := make([]string, 0, len(names))
	loaded := make([]map[string]any, 0, len(names))
	for _, name := range names {
		skill, resolveErr := t.service.ensureSkillManager().Resolve(ctx, "", name, "")
		if resolveErr != nil {
			return toolFailure(execCtx.ToolCallID, SkillToolName, "skill_load_failed", resolveErr.Error())
		}
		already, _ := t.service.sessionSkillLoaded(ctx, execCtx.SessionID, skill)
		if already {
			loaded = append(loaded, map[string]any{"name": skill.Name, "skillId": skill.ID, "status": "already_active"})
			continue
		}
		result, loadErr := t.service.loadSkillIntoSession(ctx, domain.LoadSkillIntoSessionInput{SessionID: execCtx.SessionID, SkillID: skill.ID, Reason: "model activated discovered skill"})
		if loadErr != nil {
			return toolFailure(execCtx.ToolCallID, SkillToolName, "skill_load_failed", loadErr.Error())
		}
		blocks = append(blocks, result.ModelOutput)
		loaded = append(loaded, map[string]any{"name": result.Skill.Name, "skillId": result.Skill.ID, "directory": result.Skill.RootPath, "contentHash": result.Skill.ContentHash, "status": "activated"})
	}
	content := strings.Join(blocks, "\n\n")
	structured := map[string]any{"status": "activated", "skills": loaded, "count": len(loaded)}
	if content == "" {
		raw, _ := json.MarshalIndent(structured, "", "  ")
		content = string(raw)
	}
	return domain.ToolResult{Name: SkillToolName, CallID: execCtx.ToolCallID, OK: true, Content: content, ModelContent: content, Structured: structured}
}

func skillCatalogResult(callID string, status string, candidates []SkillResolveCandidate, reason string) domain.ToolResult {
	items := make([]map[string]any, 0, len(candidates))
	lines := []string{"<available_skills>"}
	for _, candidate := range candidates {
		items = append(items, map[string]any{"name": candidate.Name, "description": candidate.Description, "scope": candidate.Scope, "source": candidate.Source})
		lines = append(lines, "  <skill>", "    <name>"+xmlEscape(candidate.Name)+"</name>", "    <description>"+xmlEscape(candidate.Description)+"</description>", "  </skill>")
	}
	lines = append(lines, "</available_skills>")
	if status == "discovered" && len(candidates) > 0 {
		lines = append(lines, "Review the candidate descriptions and call the skill tool with mode=activate and the exact applicable names before continuing.")
	}
	structured := map[string]any{"status": status, "skills": items, "count": len(items), "reason": strings.TrimSpace(reason)}
	content := strings.Join(lines, "\n")
	return domain.ToolResult{Name: SkillToolName, CallID: callID, OK: true, Content: content, ModelContent: content, Structured: structured}
}

func skillCandidatesByName(candidates []SkillResolveCandidate, names []string) []SkillResolveCandidate {
	wanted := map[string]bool{}
	for _, name := range names {
		wanted[normalizeSkillName(name)] = true
	}
	out := make([]SkillResolveCandidate, 0, len(names))
	for _, candidate := range candidates {
		if wanted[normalizeSkillName(candidate.Name)] {
			out = append(out, candidate)
		}
	}
	return out
}

func normalizeSkillNames(names []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(names))
	for _, name := range names {
		name = normalizeSkillName(name)
		if name != "" && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	return out
}
