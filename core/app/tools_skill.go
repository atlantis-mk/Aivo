package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"aivo/core/domain"
)

const SkillToolName = "skill"

type SkillLoadTool struct {
	service *Service
}

func NewSkillLoadTool(service *Service) *SkillLoadTool {
	return &SkillLoadTool{service: service}
}

func (t *SkillLoadTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name: SkillToolName, Description: strings.Join([]string{
			"Load a specialized skill when the task at hand matches one of the available skills in the system context.",
			"",
			"Use this tool to inject the skill's instructions and resources into the current conversation. The output may contain detailed workflow guidance as well as references to scripts, files, etc. in the same directory as the skill.",
			"",
			"The skill name must match one of the available skills in the system context.",
		}, "\n"),
		Capability: "skill.load", Category: "skill", RiskLevel: "low", Toolsets: []string{"safe", "coding"},
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{
			"name": map[string]any{"type": "string", "description": "The name of the skill from the available skills list."},
		}, "required": []string{"name"}, "additionalProperties": false},
	}
}

func (t *SkillLoadTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	if t == nil || t.service == nil {
		return toolFailure(execCtx.ToolCallID, SkillToolName, "skill_service_unavailable", "skill service is unavailable")
	}
	var input struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolError(SkillToolName, errors.New("invalid skill arguments"))
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return toolError(SkillToolName, errors.New("name is required"))
	}
	result, err := t.service.loadSkillIntoSession(ctx, domain.LoadSkillIntoSessionInput{SessionID: execCtx.SessionID, Name: name})
	if err != nil {
		if strings.Contains(err.Error(), "already loaded") {
			structured := map[string]any{"status": "already_loaded", "name": name}
			return domain.ToolResult{Name: SkillToolName, CallID: execCtx.ToolCallID, OK: true, Content: "Skill is already loaded in this session.", Structured: structured}
		}
		return toolFailure(execCtx.ToolCallID, SkillToolName, "skill_load_failed", err.Error())
	}
	structured := map[string]any{
		"status":      "loaded",
		"name":        result.Skill.Name,
		"directory":   result.Skill.RootPath,
		"skillId":     result.Skill.ID,
		"contentHash": result.Skill.ContentHash,
		"eventId":     result.Event.ID,
	}
	return domain.ToolResult{Name: SkillToolName, CallID: execCtx.ToolCallID, OK: true, Content: result.ModelOutput, ModelContent: result.ModelOutput, Structured: structured}
}
