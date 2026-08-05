package app

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"aivo/core/domain"
)

const (
	projectExtensionID       = "aivo.projects"
	projectQueryToolName     = "aivo_projects_query"
	projectAddToolName       = "aivo_projects_add"
	projectAssociateToolName = "aivo_projects_associate"
)

//go:embed builtin_extensions/aivo.projects.json
var projectExtensionManifest []byte

type projectBuiltinExtensionClient struct{ service *Service }

func (c *projectBuiltinExtensionClient) Initialize(_ context.Context, manifest domain.ExtensionManifest) error {
	if c == nil || c.service == nil {
		return errors.New("project service is unavailable")
	}
	if manifest.ID != projectExtensionID || manifest.Runtime.Type != domain.ExtensionRuntimeBuiltin {
		return errors.New("invalid project extension manifest")
	}
	return nil
}

func (c *projectBuiltinExtensionClient) Execute(ctx context.Context, name string, args json.RawMessage, execCtx domain.ToolExecutionContext) (domain.ToolResult, error) {
	switch name {
	case projectQueryToolName:
		var input domain.ProjectQueryInput
		if err := decodeStrictToolArgs(args, &input); err != nil {
			return projectToolFailure(name, "invalid_arguments", err), nil
		}
		result, err := c.service.QueryAgentProjects(ctx, execCtx.SessionID, input)
		if err != nil {
			return projectToolFailure(name, projectErrorCode(err, "project_update_failed"), err), nil
		}
		lines := make([]string, 0, len(result.Projects)+2)
		if result.CurrentProject != nil {
			lines = append(lines, fmt.Sprintf("Current project: %s | %s", result.CurrentProject.Name, result.CurrentProject.RootPath))
		} else {
			lines = append(lines, "Current project: unbound")
		}
		for _, project := range result.Projects {
			lines = append(lines, fmt.Sprintf("- %s | %s | %s | %s", project.ID, project.Name, firstNonEmpty(project.Description, "No description available."), project.RootPath))
		}
		if len(result.Projects) == 0 {
			lines = append(lines, "No matching projects.")
		}
		return domain.ToolResult{Name: name, OK: true, Content: strings.Join(lines, "\n"), Structured: projectStructuredResult(result)}, nil
	case projectAddToolName:
		var input struct {
			RootPath string `json:"rootPath"`
		}
		if err := decodeStrictToolArgs(args, &input); err != nil {
			return projectToolFailure(name, "invalid_arguments", err), nil
		}
		result, err := c.service.RegisterAgentProject(ctx, input.RootPath)
		if err != nil {
			return projectToolFailure(name, projectErrorCode(err, "project_update_failed"), err), nil
		}
		content := fmt.Sprintf("Project %s: %s | %s | %s", result.Status, result.Project.ID, result.Project.Name, result.Project.RootPath)
		return domain.ToolResult{Name: name, OK: true, Content: content, Structured: projectStructuredResult(result)}, nil
	case projectAssociateToolName:
		var input struct {
			ProjectID string `json:"projectId"`
			RootPath  string `json:"rootPath"`
		}
		if err := decodeStrictToolArgs(args, &input); err != nil {
			return projectToolFailure(name, "invalid_arguments", err), nil
		}
		result, err := c.service.AssociateAgentProject(ctx, execCtx.SessionID, input.ProjectID, input.RootPath)
		if err != nil {
			return projectToolFailure(name, projectErrorCode(err, "project_update_failed"), err), nil
		}
		content := "Session is already associated with this project."
		if result.Changed {
			content = "Session permanently associated with project " + result.Session.ProjectPath + ". Future workspace operations use this project."
		}
		return domain.ToolResult{Name: name, OK: true, Content: content, Structured: projectStructuredResult(result)}, nil
	default:
		return projectToolFailure(name, "invalid_arguments", errors.New("unknown project tool")), nil
	}
}

func (*projectBuiltinExtensionClient) UIEvent(context.Context, string, string, any) (any, error) {
	return nil, errors.New("project extension does not expose a view")
}

func (*projectBuiltinExtensionClient) Shutdown(context.Context) error { return nil }

func projectToolFailure(name, code string, err error) domain.ToolResult {
	return primitiveError(name, code, err)
}

func projectStructuredResult(value any) map[string]any {
	raw, err := json.Marshal(value)
	if err != nil {
		return map[string]any{}
	}
	var result map[string]any
	if json.Unmarshal(raw, &result) != nil || result == nil {
		return map[string]any{}
	}
	return result
}
