package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"aivo/core/domain"
)

// ProjectSearchTool gives the agent a complete, compact project catalogue. The
// model performs semantic matching over descriptions; this avoids brittle local
// keyword filtering and lets it choose the correct project before acting.
type ProjectSearchTool struct{ service *Service }

func NewProjectSearchTool(service *Service) *ProjectSearchTool {
	return &ProjectSearchTool{service: service}
}

func (t *ProjectSearchTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name: "search_projects", Namespace: "projects", Capability: "projects.search", RiskLevel: "low", Category: "projects", Toolsets: []string{"safe", "coding"},
		Description: "List all available projects with their one-sentence descriptions so you can determine which project matches the user's request. When the user asks to read, modify, run, or otherwise operate on a named or inferred project, call this tool with activateProjectPath set to that exact returned rootPath before using workspace tools. Activating switches the current conversation to that project automatically.",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{
			"query":               map[string]any{"type": "string", "description": "The user's project-related request. This is returned as context; all projects are always listed for semantic matching."},
			"activateProjectPath": map[string]any{"type": "string", "description": "Exact rootPath from a previous result to make the current conversation's active project. Omit when only searching."},
		}},
	}
}

func (t *ProjectSearchTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	var input struct {
		Query               string `json:"query"`
		ActivateProjectPath string `json:"activateProjectPath"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolError("search_projects", err)
	}
	if t.service == nil {
		return toolError("search_projects", errors.New("project service is unavailable"))
	}
	if path := strings.TrimSpace(input.ActivateProjectPath); path != "" {
		if strings.TrimSpace(execCtx.SessionID) == "" {
			return toolError("search_projects", errors.New("a session is required to activate a project"))
		}
		if _, err := t.service.SwitchSessionProject(ctx, execCtx.SessionID, path); err != nil {
			return toolError("search_projects", err)
		}
	}
	projects, err := t.service.ListProjects(ctx, 200)
	if err != nil {
		return toolError("search_projects", err)
	}
	lines := make([]string, 0, len(projects)+1)
	if strings.TrimSpace(input.Query) != "" {
		lines = append(lines, "Request context: "+strings.TrimSpace(input.Query))
	}
	for _, project := range projects {
		lines = append(lines, fmt.Sprintf("- %s | %s | %s", project.Name, firstNonEmpty(project.Description, "No description available."), project.RootPath))
	}
	if len(projects) == 0 {
		lines = append(lines, "No projects are available.")
	}
	structured := map[string]any{"projects": projects}
	if strings.TrimSpace(input.ActivateProjectPath) != "" {
		structured["activeProjectPath"] = strings.TrimSpace(input.ActivateProjectPath)
	}
	return domain.ToolResult{Name: "search_projects", OK: true, Content: strings.Join(lines, "\n"), Structured: structured}
}
