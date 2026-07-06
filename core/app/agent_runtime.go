package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"aivo/core/domain"
)

func (s *Service) resolveAgentModeForRequest(ctx context.Context, sessionID string, requested string) (domain.AgentModeDefinition, error) {
	if s.agentCatalog == nil {
		s.agentCatalog = NewAgentCatalog()
	}
	mode := strings.TrimSpace(requested)
	if mode == "" && strings.TrimSpace(sessionID) != "" {
		session, err := s.store.GetRuntimeSession(ctx, sessionID)
		if err == nil {
			mode = session.AgentMode
		}
	}
	def, err := s.agentCatalog.Get(mode)
	if err != nil {
		return domain.AgentModeDefinition{}, err
	}
	if strings.TrimSpace(requested) != "" && strings.TrimSpace(sessionID) != "" && !def.Hidden {
		if session, err := s.store.GetRuntimeSession(ctx, sessionID); err == nil && session.AgentMode != def.ID {
			_, _ = s.store.SetRuntimeSessionAgentMode(ctx, sessionID, def.ID)
		}
	}
	return def, nil
}

func allowedToolsetsForRun(modeDef domain.AgentModeDefinition, input domain.SubmitSessionMessageRequest) []string {
	if modeDef.ID == domain.AgentModeSchedulerWorker {
		return normalizeScheduledToolsets(input.Toolsets)
	}
	return modeDef.Toolsets
}

func visibleToolSpecsForMode(mode string, specs []domain.ToolSpec) []domain.ToolSpec {
	out := make([]domain.ToolSpec, 0, len(specs))
	for _, spec := range specs {
		action := permissionActionForSpec(spec)
		switch mode {
		case domain.AgentModeExplore, domain.AgentModePlan, domain.AgentModePlanner, domain.AgentModeReview, domain.AgentModeSummary, domain.AgentModeTitle:
			if action == permissionActionRead && !isSchedulerTool(spec) {
				out = append(out, spec)
			}
		case domain.AgentModeDebug:
			if action != permissionActionWrite && !isSchedulerTool(spec) {
				out = append(out, spec)
			}
		default:
			out = append(out, spec)
		}
	}
	return out
}

func (s *Service) modelForAgentMode(ctx context.Context, modeDef domain.AgentModeDefinition, requested *domain.ModelRef) *domain.ModelRef {
	if modeDef.Model != nil {
		return modeDef.Model
	}
	switch modeDef.ID {
	case domain.AgentModeSummary, domain.AgentModeTitle:
		cfg, err := s.AppConfig(ctx)
		if err == nil && cfg.AuxiliaryModel != nil && strings.TrimSpace(cfg.AuxiliaryModel.ModelID) != "" {
			return cfg.AuxiliaryModel
		}
	}
	return requested
}

func defaultPermissionScopeForMode(mode string) string {
	switch mode {
	case domain.AgentModeExplore, domain.AgentModePlan, domain.AgentModePlanner, domain.AgentModeReview, domain.AgentModeSummary, domain.AgentModeTitle, domain.AgentModeSchedulerWorker:
		return "read_only"
	default:
		return ""
	}
}

func (s *Service) ListAgentModes(ctx context.Context, includeHidden bool) ([]domain.AgentModeDefinition, error) {
	if s.agentCatalog == nil {
		s.agentCatalog = NewAgentCatalog()
	}
	return s.agentCatalog.List(includeHidden), nil
}

func (s *Service) SetSessionAgentMode(ctx context.Context, input domain.SetSessionAgentModeInput) (domain.Session, error) {
	if strings.TrimSpace(input.SessionID) == "" {
		return domain.Session{}, errors.New("sessionId is required")
	}
	mode, err := domain.NormalizeAgentMode(input.Mode)
	if err != nil {
		return domain.Session{}, err
	}
	session, err := s.store.SetRuntimeSessionAgentMode(ctx, input.SessionID, mode)
	if err == nil && s.onSessionUpdated != nil {
		s.onSessionUpdated(session.ID, &session)
	}
	return session, err
}

func (s *Service) SaveAgentRun(ctx context.Context, input domain.AgentRunRequest) (domain.AgentRun, error) {
	mode, err := domain.NormalizeAgentMode(input.Mode)
	if err != nil {
		return domain.AgentRun{}, err
	}
	status, err := domain.NormalizeAgentRunStatus(input.Status)
	if err != nil {
		return domain.AgentRun{}, err
	}
	return s.store.SaveAgentRun(ctx, domain.AgentRun{
		ParentSessionID: strings.TrimSpace(input.ParentSessionID),
		SessionID:       strings.TrimSpace(input.SessionID),
		Mode:            mode,
		Status:          status,
		Prompt:          strings.TrimSpace(input.Prompt),
		Result:          strings.TrimSpace(input.Result),
		Error:           strings.TrimSpace(input.Error),
		Metadata:        input.Metadata,
	})
}

func (s *Service) ListAgentRuns(ctx context.Context, input domain.AgentRunListRequest) ([]domain.AgentRun, error) {
	return s.store.ListAgentRuns(ctx, input)
}

func (s *Service) CancelAgentRun(ctx context.Context, id string) (domain.AgentRun, error) {
	run, err := s.store.GetAgentRun(ctx, id)
	if err != nil {
		return domain.AgentRun{}, err
	}
	s.cancelActiveAgentRun(run.ID)
	run.Status = domain.AgentRunStatusCancelled
	return s.store.SaveAgentRun(ctx, run)
}

func (s *Service) registerActiveAgentRun(id string, cancel context.CancelFunc) {
	id = strings.TrimSpace(id)
	if s == nil || id == "" || cancel == nil {
		return
	}
	s.activeAgentRunMu.Lock()
	defer s.activeAgentRunMu.Unlock()
	if s.activeAgentRunCancel == nil {
		s.activeAgentRunCancel = map[string]context.CancelFunc{}
	}
	s.activeAgentRunCancel[id] = cancel
}

func (s *Service) unregisterActiveAgentRun(id string) {
	id = strings.TrimSpace(id)
	if s == nil || id == "" {
		return
	}
	s.activeAgentRunMu.Lock()
	defer s.activeAgentRunMu.Unlock()
	delete(s.activeAgentRunCancel, id)
}

func (s *Service) cancelActiveAgentRun(id string) {
	id = strings.TrimSpace(id)
	if s == nil || id == "" {
		return
	}
	s.activeAgentRunMu.Lock()
	cancel := s.activeAgentRunCancel[id]
	s.activeAgentRunMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *Service) ListTodoItems(ctx context.Context, input domain.TodoListInput) ([]domain.TodoItem, error) {
	return s.store.ListTodoItems(ctx, input)
}

func (s *Service) UpdatePlan(ctx context.Context, input domain.UpdatePlanInput) ([]domain.TodoItem, error) {
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return nil, errors.New("sessionId is required")
	}
	mode := strings.TrimSpace(input.OwnerMode)
	if mode != "" {
		if _, err := domain.NormalizeAgentMode(mode); err != nil {
			return nil, err
		}
	}
	items := make([]domain.TodoItem, 0, len(input.Plan))
	inProgressCount := 0
	for index, step := range input.Plan {
		title := strings.TrimSpace(step.Step)
		if title == "" {
			return nil, errors.New("plan step is required")
		}
		status, err := domain.NormalizeTodoStatus(step.Status)
		if err != nil {
			return nil, err
		}
		if status == domain.TodoStatusInProgress {
			inProgressCount++
		}
		metadata := mergeStringMetadata(input.Metadata, nil)
		if strings.TrimSpace(input.Explanation) != "" {
			metadata = mergeStringMetadata(metadata, map[string]string{"explanation": strings.TrimSpace(input.Explanation)})
		}
		items = append(items, domain.TodoItem{
			ID:            strings.TrimSpace(step.ID),
			SessionID:     sessionID,
			ProjectPath:   strings.TrimSpace(input.ProjectPath),
			Title:         title,
			Status:        status,
			Position:      index,
			OwnerMode:     mode,
			SourceEventID: strings.TrimSpace(input.SourceEventID),
			Metadata:      metadata,
		})
	}
	if inProgressCount > 1 {
		return nil, errors.New("at most one plan step can be in_progress")
	}
	savedItems, err := s.store.ReplaceTodoItems(ctx, domain.TodoListInput{
		SessionID:   sessionID,
		ProjectPath: input.ProjectPath,
	}, items)
	if err != nil {
		return nil, err
	}
	if s.onTodoItemsUpdated != nil {
		s.onTodoItemsUpdated(sessionID, strings.TrimSpace(input.ProjectPath), savedItems)
	}
	return savedItems, nil
}

func (s *Service) SaveScheduledJob(ctx context.Context, input domain.ScheduledJobInput) (domain.ScheduledJob, error) {
	current := domain.ScheduledJob{}
	if strings.TrimSpace(input.ID) != "" {
		if existing, err := s.store.GetScheduledJob(ctx, input.ID); err == nil {
			current = existing
		}
	}
	status, err := domain.NormalizeScheduledJobStatus(firstNonEmpty(input.Status, current.Status))
	if err != nil {
		return domain.ScheduledJob{}, err
	}
	mode, err := domain.NormalizeAgentMode(firstNonEmpty(input.WorkerMode, current.WorkerMode, domain.AgentModeSchedulerWorker))
	if err != nil {
		return domain.ScheduledJob{}, err
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = current.Title
	}
	prompt := strings.TrimSpace(input.Prompt)
	if prompt == "" {
		prompt = current.Prompt
	}
	schedule := strings.TrimSpace(input.Schedule)
	if schedule == "" {
		schedule = current.Schedule
	}
	if title == "" {
		return domain.ScheduledJob{}, errors.New("title is required")
	}
	if prompt == "" {
		return domain.ScheduledJob{}, errors.New("prompt is required")
	}
	nextRunAt := strings.TrimSpace(input.NextRunAt)
	if nextRunAt == "" {
		nextRunAt = current.NextRunAt
	}
	if nextRunAt == "" {
		nextRunAt = nextRunFromSchedule(schedule, s.now())
	}
	toolsets := input.Toolsets
	if len(toolsets) == 0 {
		toolsets = current.Toolsets
	}
	toolsets = normalizeScheduledToolsets(toolsets)
	return s.store.SaveScheduledJob(ctx, domain.ScheduledJob{
		ID:              strings.TrimSpace(input.ID),
		SessionID:       firstNonEmpty(strings.TrimSpace(input.SessionID), current.SessionID),
		Title:           title,
		Prompt:          prompt,
		Schedule:        schedule,
		WorkerMode:      mode,
		Toolsets:        toolsets,
		PermissionScope: firstNonEmpty(strings.TrimSpace(input.PermissionScope), current.PermissionScope),
		Status:          status,
		NextRunAt:       nextRunAt,
		LastRunAt:       current.LastRunAt,
		LastResult:      current.LastResult,
		LastError:       current.LastError,
		Metadata:        mergeStringMetadata(current.Metadata, input.Metadata),
	})
}

func mergeStringMetadata(base map[string]string, overlay map[string]string) map[string]string {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	out := map[string]string{}
	for key, value := range base {
		out[key] = value
	}
	for key, value := range overlay {
		out[key] = value
	}
	return out
}

func (s *Service) ListScheduledJobs(ctx context.Context, input domain.ScheduledJobListInput) ([]domain.ScheduledJob, error) {
	return s.store.ListScheduledJobs(ctx, input)
}

func (s *Service) DeleteScheduledJob(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("id is required")
	}
	return s.store.DeleteScheduledJob(ctx, id)
}

func (s *Service) RunDueScheduledJobs(ctx context.Context, limit int) ([]domain.ScheduledJob, error) {
	jobs, err := s.store.ListDueScheduledJobs(ctx, domain.NowString(s.now()), limit)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ScheduledJob, 0, len(jobs))
	for _, job := range jobs {
		job.Status = domain.ScheduledJobStatusRunning
		job.LastRunAt = domain.NowString(s.now())
		running, err := s.store.SaveScheduledJob(ctx, job)
		if err != nil {
			return out, err
		}
		result, runErr := s.runScheduledJob(ctx, running)
		running.LastResult = result
		running.LastError = ""
		running.Status = domain.ScheduledJobStatusCompleted
		if runErr != nil {
			running.LastError = runErr.Error()
			running.Status = domain.ScheduledJobStatusFailed
		}
		if next := nextRunAfterScheduleCompletion(running.Schedule, s.now()); next != "" && runErr == nil {
			running.Status = domain.ScheduledJobStatusActive
			running.NextRunAt = next
		} else {
			running.NextRunAt = ""
		}
		saved, saveErr := s.store.SaveScheduledJob(ctx, running)
		if saveErr != nil {
			return out, saveErr
		}
		out = append(out, saved)
	}
	return out, nil
}

func (s *Service) runScheduledJob(ctx context.Context, job domain.ScheduledJob) (string, error) {
	if strings.TrimSpace(job.SessionID) == "" {
		return "", errors.New("scheduled job has no session")
	}
	run, err := s.store.SaveAgentRun(ctx, domain.AgentRun{
		ParentSessionID: job.SessionID,
		SessionID:       job.SessionID,
		Mode:            domain.AgentModeSchedulerWorker,
		Status:          domain.AgentRunStatusRunning,
		Prompt:          job.Prompt,
		Metadata: map[string]string{
			"scheduledJobId":  job.ID,
			"permissionScope": job.PermissionScope,
			"toolsets":        strings.Join(normalizeScheduledToolsets(job.Toolsets), ","),
		},
	})
	if err != nil {
		return "", err
	}
	runCtx, cancel := context.WithCancel(ctx)
	s.registerActiveAgentRun(run.ID, cancel)
	defer s.unregisterActiveAgentRun(run.ID)
	defer cancel()
	prepared, err := s.SubmitSessionMessage(runCtx, domain.SubmitSessionMessageRequest{
		SessionID:       job.SessionID,
		Text:            job.Prompt,
		AgentMode:       domain.AgentModeSchedulerWorker,
		Toolsets:        normalizeScheduledToolsets(job.Toolsets),
		PermissionScope: firstNonEmpty(job.PermissionScope, "read_only"),
	})
	if err != nil {
		run.Status = domain.AgentRunStatusFailed
		run.Error = err.Error()
		_, _ = s.store.SaveAgentRun(context.Background(), run)
		return "", err
	}
	if runCtx.Err() != nil {
		run.Status = domain.AgentRunStatusCancelled
		run.Error = runCtx.Err().Error()
		_, _ = s.store.SaveAgentRun(context.Background(), run)
		return "", runCtx.Err()
	}
	run.Status = domain.AgentRunStatusCompleted
	if prepared.AssistantEvent == nil {
		_, _ = s.store.SaveAgentRun(context.Background(), run)
		return "", nil
	}
	run.Result = prepared.AssistantEvent.Content
	_, _ = s.store.SaveAgentRun(context.Background(), run)
	return prepared.AssistantEvent.Content, nil
}

func normalizeScheduledToolsets(toolsets []string) []string {
	out := make([]string, 0, len(toolsets))
	seen := map[string]bool{}
	for _, toolset := range toolsets {
		clean := strings.TrimSpace(toolset)
		if clean == "" || clean == "admin" || clean == "mcp" || seen[clean] {
			continue
		}
		seen[clean] = true
		out = append(out, clean)
	}
	if len(out) == 0 {
		return []string{"safe"}
	}
	return out
}

func nextRunFromSchedule(schedule string, now time.Time) string {
	clean := strings.TrimSpace(schedule)
	if clean == "" || clean == "once" || clean == "one_time" {
		return domain.NowString(now)
	}
	if strings.HasPrefix(clean, "every:") {
		d, err := time.ParseDuration(strings.TrimPrefix(clean, "every:"))
		if err == nil && d > 0 {
			return domain.NowString(now.Add(d))
		}
	}
	if t, err := time.Parse(time.RFC3339, clean); err == nil {
		return domain.NowString(t)
	}
	return domain.NowString(now)
}

func nextRunAfterScheduleCompletion(schedule string, now time.Time) string {
	clean := strings.TrimSpace(schedule)
	if strings.HasPrefix(clean, "every:") {
		d, err := time.ParseDuration(strings.TrimPrefix(clean, "every:"))
		if err == nil && d > 0 {
			return domain.NowString(now.Add(d))
		}
	}
	return ""
}

func (s *Service) startSchedulerLoop() {
	if s == nil || s.store == nil || s.schedulerCancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.schedulerCancel = cancel
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _ = s.RunDueScheduledJobs(ctx, 5)
			}
		}
	}()
}
