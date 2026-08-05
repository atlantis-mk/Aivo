package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"aivo/core/domain"
)

type agentProjectStore interface {
	RegisterProject(context.Context, string) (domain.ProjectRegistrationResult, error)
	GetProjectByID(context.Context, string) (domain.AssistantProject, bool, error)
	GetProjectByPath(context.Context, string) (domain.AssistantProject, bool, error)
	QueryProjects(context.Context, domain.ProjectQueryInput) (domain.ProjectQueryResult, error)
	BindSessionProject(context.Context, string, string, domain.CodingContext) (domain.SessionProjectBindingResult, error)
}

type projectOperationError struct {
	Code string
	Err  error
}

func (e *projectOperationError) Error() string {
	if e == nil || e.Err == nil {
		return "project operation failed"
	}
	return e.Err.Error()
}

func (e *projectOperationError) Unwrap() error { return e.Err }

func projectError(code, message string) error {
	return &projectOperationError{Code: code, Err: errors.New(message)}
}

func projectErrorCode(err error, fallback string) string {
	var operationErr *projectOperationError
	if errors.As(err, &operationErr) && strings.TrimSpace(operationErr.Code) != "" {
		return operationErr.Code
	}
	return fallback
}

func (s *Service) selectProjectDirectory(path string) (string, error) {
	clean := strings.TrimSpace(path)
	if clean == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		clean = wd
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("selected path is not a directory")
	}
	return abs, nil
}

func (s *Service) UpsertProject(ctx context.Context, path string) (domain.AssistantProject, error) {
	abs, err := s.SelectProjectDirectory(path)
	if err != nil {
		return domain.AssistantProject{}, err
	}
	project, err := s.store.UpsertProject(ctx, abs)
	if err != nil || strings.TrimSpace(project.Description) != "" {
		return project, err
	}
	// Always provide a useful searchable description immediately. A configured
	// auxiliary model refines it asynchronously without delaying project selection.
	description := fallbackProjectDescription(abs)
	project, _ = s.store.UpdateProjectDescription(ctx, abs, description)
	go s.refineProjectDescription(abs)
	return project, nil
}

func (s *Service) SetProjectSidebarHidden(ctx context.Context, path string, hidden bool) (domain.AssistantProject, error) {
	abs, err := s.SelectProjectDirectory(path)
	if err != nil {
		return domain.AssistantProject{}, err
	}
	return s.store.SetProjectSidebarHidden(ctx, abs, hidden)
}

func (s *Service) QueryAgentProjects(ctx context.Context, sessionID string, input domain.ProjectQueryInput) (domain.ProjectQueryResult, error) {
	store, ok := s.store.(agentProjectStore)
	if !ok {
		return domain.ProjectQueryResult{}, projectError("project_update_failed", "project catalog is unavailable")
	}
	input.ProjectID = strings.TrimSpace(input.ProjectID)
	input.Query = strings.TrimSpace(input.Query)
	input.Cursor = strings.TrimSpace(input.Cursor)
	if input.ProjectID != "" && (input.Query != "" || input.Cursor != "" || input.Limit != 0) {
		return domain.ProjectQueryResult{}, projectError("invalid_arguments", "projectId cannot be combined with query, limit, or cursor")
	}
	if input.Limit < 0 {
		return domain.ProjectQueryResult{}, projectError("invalid_arguments", "limit cannot be negative")
	}
	if input.Limit == 0 {
		input.Limit = 20
	}
	if input.Limit > 50 {
		input.Limit = 50
	}
	requestedLimit := input.Limit
	result := domain.ProjectQueryResult{Projects: []domain.AssistantProject{}}
	if input.ProjectID != "" {
		page, err := store.QueryProjects(ctx, input)
		if err != nil {
			return domain.ProjectQueryResult{}, err
		}
		for _, project := range page.Projects {
			if !isManagedWorkspace(project.RootPath) {
				result.Projects = append(result.Projects, project)
			}
		}
	} else {
		cursor := input.Cursor
		for len(result.Projects) < requestedLimit {
			pageInput := input
			pageInput.Cursor = cursor
			pageInput.Limit = requestedLimit - len(result.Projects)
			page, err := store.QueryProjects(ctx, pageInput)
			if err != nil {
				if strings.Contains(strings.ToLower(err.Error()), "invalid project cursor") {
					return domain.ProjectQueryResult{}, projectError("invalid_cursor", "project cursor is invalid")
				}
				return domain.ProjectQueryResult{}, err
			}
			for _, project := range page.Projects {
				if strings.TrimSpace(project.RootPath) != "" && !isManagedWorkspace(project.RootPath) {
					result.Projects = append(result.Projects, project)
				}
			}
			cursor = page.NextCursor
			if cursor == "" || len(page.Projects) == 0 {
				break
			}
		}
		result.NextCursor = cursor
	}
	if sessionID := strings.TrimSpace(sessionID); sessionID != "" {
		if session, err := s.store.GetRuntimeSession(ctx, sessionID); err == nil && strings.TrimSpace(session.ProjectPath) != "" {
			if current, found, findErr := store.GetProjectByPath(ctx, session.ProjectPath); findErr == nil && found {
				result.CurrentProject = &current
			}
		}
	}
	return result, nil
}

func (s *Service) RegisterAgentProject(ctx context.Context, rootPath string) (domain.ProjectRegistrationResult, error) {
	store, ok := s.store.(agentProjectStore)
	if !ok {
		return domain.ProjectRegistrationResult{}, projectError("project_update_failed", "project catalog is unavailable")
	}
	abs, err := validateAgentProjectRoot(rootPath)
	if err != nil {
		return domain.ProjectRegistrationResult{}, err
	}
	result, err := store.RegisterProject(ctx, abs)
	if err != nil {
		return domain.ProjectRegistrationResult{}, projectError("project_update_failed", "register project: "+err.Error())
	}
	if strings.TrimSpace(result.Project.Description) == "" {
		description := fallbackProjectDescription(abs)
		if updated, updateErr := s.store.UpdateProjectDescription(ctx, abs, description); updateErr == nil {
			result.Project = updated
		}
		go s.refineProjectDescription(abs)
	}
	return result, nil
}

func (s *Service) AssociateAgentProject(ctx context.Context, sessionID, projectID, rootPath string) (domain.SessionProjectBindingResult, error) {
	prepared, err := s.prepareAgentProjectAssociation(ctx, sessionID, projectID, rootPath)
	if err != nil {
		return domain.SessionProjectBindingResult{}, err
	}
	if prepared.idempotent {
		return domain.SessionProjectBindingResult{Session: prepared.session, Changed: false}, nil
	}
	if err := ctx.Err(); err != nil {
		return domain.SessionProjectBindingResult{}, projectError("cancelled", "project association was cancelled")
	}
	targetContext := s.buildCodingContext(ctx, prepared.session.ID, prepared.project.RootPath, false)
	if err := ctx.Err(); err != nil {
		return domain.SessionProjectBindingResult{}, projectError("cancelled", "project association was cancelled")
	}
	result, err := prepared.store.BindSessionProject(ctx, prepared.session.ID, prepared.project.ID, targetContext)
	if err != nil {
		return domain.SessionProjectBindingResult{}, projectError("project_update_failed", "associate project: "+err.Error())
	}
	if result.Conflict {
		return domain.SessionProjectBindingResult{}, projectError("project_already_bound", "this session was bound to another project and cannot switch")
	}
	if s.onSessionUpdated != nil {
		s.onSessionUpdated(prepared.session.ID, &result.Session)
	}
	return result, nil
}

type preparedAgentProjectAssociation struct {
	store      agentProjectStore
	session    domain.Session
	project    domain.AssistantProject
	idempotent bool
}

func (s *Service) prepareAgentProjectAssociation(ctx context.Context, sessionID, projectID, rootPath string) (preparedAgentProjectAssociation, error) {
	if ctx.Err() != nil {
		return preparedAgentProjectAssociation{}, projectError("cancelled", "project association was cancelled")
	}
	store, ok := s.store.(agentProjectStore)
	if !ok {
		return preparedAgentProjectAssociation{}, projectError("project_update_failed", "project association is unavailable")
	}
	sessionID = strings.TrimSpace(sessionID)
	projectID = strings.TrimSpace(projectID)
	rootPath = strings.TrimSpace(rootPath)
	if sessionID == "" {
		return preparedAgentProjectAssociation{}, projectError("session_required", "the current session is required")
	}
	if projectID == "" || rootPath == "" {
		return preparedAgentProjectAssociation{}, projectError("invalid_arguments", "projectId and rootPath are required")
	}
	project, found, err := store.GetProjectByID(ctx, projectID)
	if err != nil {
		return preparedAgentProjectAssociation{}, projectError("project_update_failed", "load project: "+err.Error())
	}
	if !found {
		return preparedAgentProjectAssociation{}, projectError("project_not_found", "project is not available")
	}
	if !sameProjectPath(project.RootPath, rootPath) {
		return preparedAgentProjectAssociation{}, projectError("project_reference_mismatch", "projectId and rootPath do not identify the same project")
	}
	session, err := s.store.GetRuntimeSession(ctx, sessionID)
	if err != nil {
		return preparedAgentProjectAssociation{}, projectError("session_required", "current session was not found")
	}
	if session.Type != domain.SessionTypeCoding {
		return preparedAgentProjectAssociation{}, projectError("coding_session_required", "only a coding session can be associated with a project")
	}
	if strings.TrimSpace(session.ProjectPath) != "" {
		if sameProjectPath(session.ProjectPath, project.RootPath) {
			return preparedAgentProjectAssociation{store: store, session: session, project: project, idempotent: true}, nil
		}
		return preparedAgentProjectAssociation{}, projectError("project_already_bound", "this session is already bound to another project and cannot switch")
	}
	if project.SidebarHidden {
		return preparedAgentProjectAssociation{}, projectError("project_not_found", "project is not available")
	}
	cc, err := s.store.GetCodingContext(ctx, sessionID)
	if err != nil {
		return preparedAgentProjectAssociation{}, projectError("workspace_specialized", "the unscoped session workspace is unavailable")
	}
	cfg, err := s.AppConfig(ctx)
	if err != nil || strings.TrimSpace(cfg.InitialWorkspacePath) == "" || !sameProjectPath(cc.ProjectPath, cfg.InitialWorkspacePath) {
		return preparedAgentProjectAssociation{}, projectError("workspace_specialized", "the session already uses a specialized workspace and cannot bind a project")
	}
	if busy, busyErr := s.sessionHasLiveTerminal(ctx, cc.ProjectPath, sessionID); busyErr != nil {
		return preparedAgentProjectAssociation{}, projectError("project_update_failed", "inspect session terminals: "+busyErr.Error())
	} else if busy {
		return preparedAgentProjectAssociation{}, projectError("workspace_busy", "close the session's interactive terminals before binding a project")
	}
	if err := ctx.Err(); err != nil {
		return preparedAgentProjectAssociation{}, projectError("cancelled", "project association was cancelled")
	}
	return preparedAgentProjectAssociation{store: store, session: session, project: project}, nil
}

func (s *Service) prepareProjectPermission(ctx context.Context, name string, args json.RawMessage, execCtx domain.ToolExecutionContext) ([]string, map[string]any, bool, error) {
	switch name {
	case projectAddToolName:
		var input struct {
			RootPath string `json:"rootPath"`
		}
		if err := decodeStrictToolArgs(args, &input); err != nil {
			return nil, nil, false, &projectOperationError{Code: "invalid_arguments", Err: err}
		}
		rootPath, err := validateAgentProjectRoot(input.RootPath)
		if err != nil {
			return nil, nil, false, err
		}
		return []string{projectPermissionKey("add", rootPath)}, map[string]any{
			"projectOperation": "add", "projectRoot": rootPath, "rememberScope": "exact_project",
		}, false, nil
	case projectAssociateToolName:
		var input struct {
			ProjectID string `json:"projectId"`
			RootPath  string `json:"rootPath"`
		}
		if err := decodeStrictToolArgs(args, &input); err != nil {
			return nil, nil, false, &projectOperationError{Code: "invalid_arguments", Err: err}
		}
		prepared, err := s.prepareAgentProjectAssociation(ctx, execCtx.SessionID, input.ProjectID, input.RootPath)
		if err != nil {
			return nil, nil, false, err
		}
		return []string{projectPermissionKey("associate", prepared.project.ID)}, map[string]any{
			"projectOperation": "associate", "projectRoot": prepared.project.RootPath, "projectName": prepared.project.Name,
			"immutableAssociation": true, "rememberScope": "exact_project",
		}, prepared.idempotent, nil
	default:
		return nil, nil, false, projectError("invalid_arguments", "unsupported project permission operation")
	}
}

func validateAgentProjectRoot(rootPath string) (string, error) {
	rootPath = strings.TrimSpace(rootPath)
	if rootPath == "" || !filepath.IsAbs(rootPath) {
		return "", projectError("absolute_path_required", "rootPath must be an absolute path")
	}
	abs := filepath.Clean(rootPath)
	info, err := os.Stat(abs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", projectError("project_path_not_found", "project path does not exist")
		}
		return "", projectError("project_update_failed", "project path is not accessible")
	}
	if !info.IsDir() {
		return "", projectError("project_not_directory", "project path is not a directory")
	}
	return abs, nil
}

func projectPermissionKey(operation, target string) string {
	return "project:" + operation + ":" + base64.RawURLEncoding.EncodeToString([]byte(strings.TrimSpace(target)))
}

func (s *Service) sessionHasLiveTerminal(ctx context.Context, workspaceRoot, sessionID string) (bool, error) {
	agentTerminals, err := s.ListSessionTerminals(ctx, workspaceRoot, sessionID)
	if err != nil {
		return false, err
	}
	for _, terminal := range agentTerminals {
		if terminal.Status != AgentPTYStatusExited {
			return true, nil
		}
	}
	if s.terminals == nil {
		return false, nil
	}
	terminals, err := s.terminals.List(ctx, workspaceRoot)
	if err != nil {
		return false, err
	}
	for _, terminal := range terminals {
		if terminal.SessionID == sessionID && terminal.Status == TerminalStatusRunning {
			return true, nil
		}
	}
	return false, nil
}

func sameProjectPath(left, right string) bool {
	left = filepath.Clean(strings.TrimSpace(left))
	right = filepath.Clean(strings.TrimSpace(right))
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func (s *Service) ListProjects(ctx context.Context, limit int) ([]domain.AssistantProject, error) {
	requestedLimit := limit
	if requestedLimit <= 0 {
		requestedLimit = 20
	}
	fetchLimit := requestedLimit * 3
	if fetchLimit < 50 {
		fetchLimit = 50
	}
	if fetchLimit > 200 {
		fetchLimit = 200
	}
	projects, err := s.store.ListProjects(ctx, fetchLimit)
	if err != nil {
		return nil, err
	}
	filtered := make([]domain.AssistantProject, 0, len(projects))
	for _, project := range projects {
		if strings.TrimSpace(project.RootPath) == "" || isManagedWorkspace(project.RootPath) {
			continue
		}
		filtered = append(filtered, project)
		if len(filtered) >= requestedLimit {
			break
		}
	}
	return filtered, nil
}
