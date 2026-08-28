package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"

	"aivo/core/domain"
)

type serviceTool struct {
	spec    domain.ToolSpec
	handler func(context.Context, json.RawMessage, domain.ToolExecutionContext) domain.ToolResult
}

func (t serviceTool) Spec() domain.ToolSpec {
	return t.spec
}

func (t serviceTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	return t.handler(ctx, args, execCtx)
}

func newAgentRuntimeTools(service *Service) []domain.Tool {
	tools := []domain.Tool{
		serviceTool{spec: jsonToolSpec("agent_mode_list", "List available agent modes.", "agent.read", "agent", "safe", "admin"), handler: service.agentModeListTool},
		serviceTool{spec: jsonToolSpec("agent_mode_set", "Set the current session agent mode.", "agent.write", "agent", "admin"), handler: service.agentModeSetTool},
		serviceTool{spec: jsonToolSpec("agent_delegate_task", "Delegate a bounded task to a child agent session.", "agent.delegate", "agent", "safe", "coding", "personal"), handler: service.agentDelegateTaskTool},
		serviceTool{spec: jsonToolSpec("agent_run_list", "List subagent run records for the current session.", "agent.read", "agent", "safe", "personal"), handler: service.agentRunListTool},
		serviceTool{spec: jsonToolSpec("agent_run_cancel", "Cancel a subagent run record.", "agent.write", "agent", "personal"), handler: service.agentRunCancelTool},
	}
	tools = append(tools, newDefaultHostControlTools(service)...)
	tools = append(tools,
		serviceTool{spec: jsonToolSpec("automation_create", "Create an automation or scheduled job.", "scheduler.write", "automation", "personal"), handler: service.automationCreateTool},
		serviceTool{spec: jsonToolSpec("automation_list", "List automations and scheduled jobs.", "scheduler.read", "automation", "safe", "personal"), handler: service.automationListTool},
		serviceTool{spec: jsonToolSpec("automation_update", "Update an automation or scheduled job.", "scheduler.write", "automation", "personal"), handler: service.automationUpdateTool},
		serviceTool{spec: jsonToolSpec("automation_cancel", "Cancel an automation or scheduled job.", "scheduler.write", "automation", "personal"), handler: service.automationCancelTool},
	)
	return tools
}

func newDefaultHostControlTools(service *Service) []domain.Tool {
	return []domain.Tool{
		serviceTool{spec: updatePlanToolSpec(), handler: service.updatePlanTool},
		serviceTool{spec: questionToolSpec("ask_user", ""), handler: service.askUserTool},
	}
}

func registerDefaultHostControlTools(registry *Registry, service *Service) error {
	for _, tool := range newDefaultHostControlTools(service) {
		if err := registry.Register(tool); err != nil {
			return err
		}
	}
	return nil
}

func jsonToolSpec(name, description, capability, category string, toolsets ...string) domain.ToolSpec {
	return domain.ToolSpec{
		Name:        name,
		Description: description,
		Kind:        domain.ToolKindJSON,
		InputSchema: map[string]any{"type": "object", "additionalProperties": true},
		Capability:  capability,
		RiskLevel:   "low",
		Category:    category,
		Toolsets:    toolsets,
	}
}

func updatePlanToolSpec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:        "update_plan",
		Description: "Update the visible agent execution plan for the current session. Use this for non-trivial work with 3+ distinct steps, multiple user tasks, or work that benefits from visible progress. Submit the full current plan every time. Keep at most one step in_progress, mark steps completed immediately after they are actually done, and do not use this for simple informational or single-step requests.",
		Kind:        domain.ToolKindJSON,
		InputSchema: map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"explanation": map[string]any{
					"type":        "string",
					"description": "Optional short note about why the plan changed.",
				},
				"plan": map[string]any{
					"type":        "array",
					"description": "The full current ordered plan.",
					"items": map[string]any{
						"type":                 "object",
						"additionalProperties": false,
						"properties": map[string]any{
							"id": map[string]any{
								"type":        "string",
								"description": "Optional stable step id.",
							},
							"step": map[string]any{
								"type":        "string",
								"description": "Task step text.",
							},
							"status": map[string]any{
								"type":        "string",
								"enum":        []string{"pending", "in_progress", "completed", "cancelled"},
								"description": "Current step status.",
							},
						},
						"required": []string{"step", "status"},
					},
				},
			},
			"required": []string{"plan"},
		},
		Capability: "plan.write",
		RiskLevel:  "low",
		Category:   "plan",
		Toolsets:   []string{"safe", "personal"},
	}
}

func (s *Service) agentModeListTool(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	return s.agentListModesToolNamed(ctx, args, execCtx, "agent_mode_list")
}

func (s *Service) agentListModesToolNamed(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext, toolName string) domain.ToolResult {
	var input struct {
		IncludeHidden bool `json:"includeHidden"`
	}
	_ = json.Unmarshal(args, &input)
	modes, err := s.ListAgentModesForProject(ctx, execCtx.WorkspaceRoot, input.IncludeHidden)
	return structuredToolResult(toolName, modes, err)
}

func (s *Service) agentModeSetTool(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	return s.agentSetModeToolNamed(ctx, args, execCtx, "agent_mode_set")
}

func (s *Service) agentSetModeToolNamed(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext, toolName string) domain.ToolResult {
	var input domain.SetSessionAgentModeInput
	if err := json.Unmarshal(args, &input); err != nil {
		return errorToolResult(toolName, err)
	}
	if input.SessionID == "" {
		input.SessionID = execCtx.SessionID
	}
	session, err := s.SetSessionAgentMode(ctx, input)
	return structuredToolResult(toolName, session, err)
}

func (s *Service) agentDelegateTaskTool(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	return s.delegateTaskToolNamed(ctx, args, execCtx, "agent_delegate_task")
}

func (s *Service) delegateTaskToolNamed(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext, toolName string) domain.ToolResult {
	var input struct {
		Mode   string `json:"mode"`
		Prompt string `json:"prompt"`
		Title  string `json:"title"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return errorToolResult(toolName, err)
	}
	if strings.TrimSpace(execCtx.SessionID) == "" {
		return errorToolResult(toolName, errors.New("sessionId is required"))
	}
	prompt := strings.TrimSpace(input.Prompt)
	if prompt == "" {
		return errorToolResult(toolName, errors.New("prompt is required"))
	}
	mode, err := domain.NormalizeAgentMode(input.Mode)
	if err != nil {
		return errorToolResult(toolName, err)
	}
	if mode == domain.AgentModeSchedulerWorker {
		return errorToolResult(toolName, errors.New("hidden worker modes cannot be delegated directly"))
	}
	if err := s.validateDelegateLimits(ctx, execCtx.SessionID); err != nil {
		return errorToolResult(toolName, err)
	}
	parent, err := s.store.GetRuntimeSession(ctx, execCtx.SessionID)
	if err != nil {
		return errorToolResult(toolName, err)
	}
	catalog, err := s.agentCatalogForProject(ctx, parent.ProjectPath)
	if err != nil {
		return errorToolResult(toolName, err)
	}
	parentMode, err := catalog.Get(firstNonEmpty(execCtx.AgentMode, parent.AgentMode))
	if err != nil {
		return errorToolResult(toolName, err)
	}
	if !agentModeAllowsSubagent(parentMode, mode) {
		return errorToolResult(toolName, errors.New("agent mode "+parentMode.ID+" is not associated with subagent "+mode))
	}
	modeDefinition, err := catalog.Get(mode)
	if err != nil {
		return errorToolResult(toolName, err)
	}
	if modeDefinition.Mode == "primary" || modeDefinition.ID == domain.AgentModeSummary || modeDefinition.ID == domain.AgentModeTitle || modeDefinition.ID == domain.AgentModeSchedulerWorker {
		return errorToolResult(toolName, errors.New("agent is not available for delegated tasks"))
	}
	// Built-in agents deliberately describe the full production toolset. Tests and
	// reduced embeddings may expose only a subset, so availability validation is
	// reserved for user-defined agents whose contract must be checked strictly.
	if modeDefinition.Revision != "" && !modeDefinition.BuiltIn {
		if err := s.validateAgentToolsets(parent.ProjectPath, modeDefinition); err != nil {
			return errorToolResult(toolName, err)
		}
	}
	child, err := s.store.ForkRuntimeSession(ctx, parent, domain.ForkSessionRequest{Title: firstNonEmpty(strings.TrimSpace(input.Title), "Subagent: "+bounded(prompt, 48)), Goal: prompt})
	if err != nil {
		return errorToolResult(toolName, err)
	}
	child, _ = s.store.SetRuntimeSessionAgentMode(ctx, child.ID, mode)
	runMetadata := map[string]string{"depth": delegateDepthString(parent)}
	if strings.TrimSpace(execCtx.ToolCallID) != "" {
		runMetadata["toolCallId"] = strings.TrimSpace(execCtx.ToolCallID)
	}
	run, err := s.store.SaveAgentRun(ctx, domain.AgentRun{ParentSessionID: execCtx.SessionID, SessionID: child.ID, Mode: mode, Status: domain.AgentRunStatusRunning, Prompt: prompt, Metadata: runMetadata})
	if err != nil {
		return errorToolResult(toolName, err)
	}
	s.emitDelegateTaskToolCallUpdate(execCtx, run, toolName)
	runCtx, cancel := context.WithCancel(ctx)
	s.registerActiveAgentRun(run.ID, cancel)
	prepared, runErr := s.SubmitSessionMessage(runCtx, domain.SubmitSessionMessageRequest{SessionID: child.ID, Text: prompt, AgentMode: mode})
	runCtxErr := runCtx.Err()
	s.unregisterActiveAgentRun(run.ID)
	cancel()
	run.Status = domain.AgentRunStatusCompleted
	if runCtxErr != nil {
		run.Status = domain.AgentRunStatusCancelled
		run.Error = runCtxErr.Error()
	} else if runErr != nil {
		run.Status = domain.AgentRunStatusFailed
		run.Error = runErr.Error()
	} else if prepared.AssistantEvent != nil {
		run.Result = prepared.AssistantEvent.Content
	}
	run, _ = s.store.SaveAgentRun(ctx, run)
	_, _ = s.AppendEvent(ctx, domain.AppendEventRequest{
		SessionID:  execCtx.SessionID,
		TurnID:     execCtx.TurnID,
		Type:       domain.EventTypeSystemNote,
		Role:       domain.EventRoleSystem,
		Visibility: domain.EventVisibilityInternal,
		Content:    bounded(run.Result, 2000),
		Payload:    map[string]any{"agentRunId": run.ID, "childSessionId": child.ID, "mode": mode},
	})
	if runErr != nil {
		return structuredToolResult(toolName, run, runErr)
	}
	return structuredToolResult(toolName, run, nil)
}

func agentModeAllowsSubagent(parent domain.AgentModeDefinition, child string) bool {
	for _, candidate := range parent.Subagents {
		if candidate == child {
			return true
		}
	}
	return false
}

func (s *Service) emitDelegateTaskToolCallUpdate(execCtx domain.ToolExecutionContext, run domain.AgentRun, toolName string) {
	if s.onToolCallUpdated == nil || strings.TrimSpace(execCtx.ToolCallID) == "" || strings.TrimSpace(execCtx.SessionID) == "" {
		return
	}
	now := domain.NowString(s.now())
	s.onToolCallUpdated(execCtx.SessionID, execCtx.TurnID, domain.ToolCall{
		ID:        execCtx.ToolCallID,
		SessionID: execCtx.SessionID,
		TurnID:    execCtx.TurnID,
		Name:      toolName,
		Status:    domain.ToolCallStatusRunning,
		Result: map[string]any{
			"structured": map[string]any{"result": run},
		},
		TimeCreated: now,
		TimeUpdated: now,
	}, false)
}

func (s *Service) validateDelegateLimits(ctx context.Context, sessionID string) error {
	session, err := s.store.GetRuntimeSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if session.ParentSessionID != "" {
		parent, err := s.store.GetRuntimeSession(ctx, session.ParentSessionID)
		if err == nil && parent.ParentSessionID != "" {
			return errors.New("agent_delegate_task depth limit exceeded")
		}
	}
	runs, err := s.store.ListAgentRuns(ctx, domain.AgentRunListRequest{SessionID: sessionID, Status: domain.AgentRunStatusRunning, Limit: 10})
	if err != nil {
		return err
	}
	limit := loadEffectiveRuntimeConfig(session.ProjectPath).Config.MaxParallelChildren
	if limit <= 0 {
		limit = 4
	}
	if len(runs) >= limit {
		return errors.New("agent_delegate_task concurrent child limit exceeded")
	}
	return nil
}

func delegateDepthString(session domain.Session) string {
	if session.ParentSessionID == "" {
		return "1"
	}
	return "2"
}

func (s *Service) agentRunListTool(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	return s.agentRunListToolNamed(ctx, args, execCtx, "agent_run_list")
}

func (s *Service) agentRunListToolNamed(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext, toolName string) domain.ToolResult {
	var input domain.AgentRunListRequest
	_ = json.Unmarshal(args, &input)
	if input.SessionID == "" {
		input.SessionID = execCtx.SessionID
	}
	runs, err := s.ListAgentRuns(ctx, input)
	return structuredToolResult(toolName, runs, err)
}

func (s *Service) agentRunCancelTool(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	return s.agentRunCancelToolNamed(ctx, args, execCtx, "agent_run_cancel")
}

func (s *Service) agentRunCancelToolNamed(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext, toolName string) domain.ToolResult {
	var input struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return errorToolResult(toolName, err)
	}
	run, err := s.CancelAgentRun(ctx, input.ID)
	return structuredToolResult(toolName, run, err)
}

func (s *Service) updatePlanTool(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	var input domain.UpdatePlanInput
	if err := json.Unmarshal(args, &input); err != nil {
		return errorToolResult("update_plan", err)
	}
	if input.SessionID == "" {
		input.SessionID = execCtx.SessionID
	}
	if input.ProjectPath == "" {
		input.ProjectPath = execCtx.WorkspaceRoot
	}
	if input.OwnerMode == "" {
		input.OwnerMode = execCtx.AgentMode
	}
	items, err := s.UpdatePlan(ctx, input)
	return structuredToolResult("update_plan", map[string]any{
		"items": items,
		"summary": map[string]int{
			"pending":    countTodoStatus(items, domain.TodoStatusPending),
			"inProgress": countTodoStatus(items, domain.TodoStatusInProgress),
			"completed":  countTodoStatus(items, domain.TodoStatusCompleted),
			"cancelled":  countTodoStatus(items, domain.TodoStatusCancelled),
			"total":      len(items),
		},
	}, err)
}

func countTodoStatus(items []domain.TodoItem, status string) int {
	count := 0
	for _, item := range items {
		if item.Status == status {
			count++
		}
	}
	return count
}

func (s *Service) automationCreateTool(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	return s.scheduledJobSaveToolNamed(ctx, args, execCtx, "automation_create")
}

func (s *Service) automationUpdateTool(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	return s.scheduledJobSaveToolNamed(ctx, args, execCtx, "automation_update")
}

func (s *Service) scheduledJobSaveToolNamed(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext, toolName string) domain.ToolResult {
	var input domain.ScheduledJobInput
	if err := json.Unmarshal(args, &input); err != nil {
		return errorToolResult(toolName, err)
	}
	if input.SessionID == "" {
		input.SessionID = execCtx.SessionID
	}
	job, err := s.SaveScheduledJob(ctx, input)
	return structuredToolResult(toolName, job, err)
}

func (s *Service) automationListTool(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	return s.scheduledJobListToolNamed(ctx, args, execCtx, "automation_list")
}

func (s *Service) scheduledJobListToolNamed(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext, toolName string) domain.ToolResult {
	var input domain.ScheduledJobListInput
	_ = json.Unmarshal(args, &input)
	if input.SessionID == "" {
		input.SessionID = execCtx.SessionID
	}
	jobs, err := s.ListScheduledJobs(ctx, input)
	return structuredToolResult(toolName, jobs, err)
}

func (s *Service) automationCancelTool(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	return s.scheduledJobCancelToolNamed(ctx, args, execCtx, "automation_cancel")
}

func (s *Service) scheduledJobCancelToolNamed(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext, toolName string) domain.ToolResult {
	var input struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return errorToolResult(toolName, err)
	}
	if strings.TrimSpace(input.ID) == "" {
		return errorToolResult(toolName, errors.New("id is required"))
	}
	err := s.DeleteScheduledJob(ctx, input.ID)
	return structuredToolResult(toolName, map[string]string{"id": input.ID, "status": domain.ScheduledJobStatusCancelled}, err)
}

func structuredToolResult(name string, value any, err error) domain.ToolResult {
	if err != nil {
		return errorToolResult(name, err)
	}
	content := ""
	if raw, marshalErr := json.MarshalIndent(value, "", "  "); marshalErr == nil {
		content = string(raw)
	}
	structured := map[string]any{"result": value}
	return domain.ToolResult{CallID: uuid.NewString(), Name: name, OK: true, Content: content, Structured: structured}
}

func errorToolResult(name string, err error) domain.ToolResult {
	message := "unknown error"
	if err != nil {
		message = err.Error()
	}
	return domain.ToolResult{CallID: uuid.NewString(), Name: name, OK: false, Error: message, ToolError: &domain.ToolError{Code: "tool_error", Message: message}}
}
