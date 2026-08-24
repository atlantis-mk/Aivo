package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aivo/core/domain"
)

func (s *Store) SaveAgentRun(ctx context.Context, run domain.AgentRun) (domain.AgentRun, error) {
	now := domain.NowString(time.Now())
	if run.ID == "" {
		run.ID = uuid.NewString()
	}
	mode, err := domain.NormalizeAgentMode(run.Mode)
	if err != nil {
		return domain.AgentRun{}, err
	}
	status, err := domain.NormalizeAgentRunStatus(run.Status)
	if err != nil {
		return domain.AgentRun{}, err
	}
	if run.TimeCreated == "" {
		run.TimeCreated = now
	}
	run.Mode = mode
	run.Status = status
	run.TimeUpdated = now
	if status != domain.AgentRunStatusRunning && run.TimeCompleted == "" {
		run.TimeCompleted = now
	}
	row := agentRunToRow(run)
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error; err != nil {
		return domain.AgentRun{}, err
	}
	return agentRunFromRow(row), nil
}

func (s *Store) ListAgentRuns(ctx context.Context, input domain.AgentRunListRequest) ([]domain.AgentRun, error) {
	limit := input.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := s.db.WithContext(ctx).Model(&agentRunRow{})
	if strings.TrimSpace(input.SessionID) != "" {
		sessionID := strings.TrimSpace(input.SessionID)
		q = q.Where("session_id = ? OR parent_session_id = ?", sessionID, sessionID)
	}
	if strings.TrimSpace(input.Status) != "" {
		q = q.Where("status = ?", strings.TrimSpace(input.Status))
	}
	var rows []agentRunRow
	if err := q.Order("time_created DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.AgentRun, 0, len(rows))
	for _, row := range rows {
		out = append(out, agentRunFromRow(row))
	}
	return out, nil
}

func (s *Store) GetAgentRun(ctx context.Context, id string) (domain.AgentRun, error) {
	var row agentRunRow
	if err := s.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).First(&row).Error; err != nil {
		return domain.AgentRun{}, err
	}
	return agentRunFromRow(row), nil
}

func (s *Store) ReplaceTodoItems(ctx context.Context, input domain.TodoListInput, items []domain.TodoItem) ([]domain.TodoItem, error) {
	now := domain.NowString(time.Now())
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return nil, errors.New("session_id is required")
	}
	projectPath := normalizeStoredPath(input.ProjectPath)
	out := make([]domain.TodoItem, 0, len(items))
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q := tx.Where("session_id = ?", sessionID)
		if projectPath != "" {
			q = q.Where("project_path = ?", projectPath)
		}
		if err := q.Delete(&todoItemRow{}).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		rows := make([]todoItemRow, 0, len(items))
		for index, item := range items {
			if item.ID == "" {
				item.ID = uuid.NewString()
			}
			status, err := domain.NormalizeTodoStatus(item.Status)
			if err != nil {
				return err
			}
			item.SessionID = sessionID
			item.ProjectPath = projectPath
			item.Status = status
			item.Position = index
			if item.TimeCreated == "" {
				item.TimeCreated = now
			}
			item.TimeUpdated = now
			row := todoItemToRow(item)
			rows = append(rows, row)
			out = append(out, todoItemFromRow(row))
		}
		return tx.Create(&rows).Error
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) ListTodoItems(ctx context.Context, input domain.TodoListInput) ([]domain.TodoItem, error) {
	limit := input.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := s.db.WithContext(ctx).Model(&todoItemRow{})
	if strings.TrimSpace(input.SessionID) != "" {
		q = q.Where("session_id = ?", strings.TrimSpace(input.SessionID))
	}
	if projectPath := normalizeStoredPath(input.ProjectPath); projectPath != "" {
		q = q.Where("project_path = ?", projectPath)
	}
	if strings.TrimSpace(input.Status) != "" {
		q = q.Where("status = ?", strings.TrimSpace(input.Status))
	}
	var rows []todoItemRow
	if err := q.Order("position ASC, time_updated DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.TodoItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, todoItemFromRow(row))
	}
	return out, nil
}

func (s *Store) SaveScheduledJob(ctx context.Context, job domain.ScheduledJob) (domain.ScheduledJob, error) {
	now := domain.NowString(time.Now())
	if job.ID == "" {
		job.ID = uuid.NewString()
	}
	status, err := domain.NormalizeScheduledJobStatus(job.Status)
	if err != nil {
		return domain.ScheduledJob{}, err
	}
	workerMode := strings.TrimSpace(job.WorkerMode)
	if workerMode == "" {
		workerMode = domain.AgentModeSchedulerWorker
	}
	mode, err := domain.NormalizeAgentMode(workerMode)
	if err != nil {
		return domain.ScheduledJob{}, err
	}
	job.Status = status
	job.WorkerMode = mode
	if job.TimeCreated == "" {
		job.TimeCreated = now
	}
	job.TimeUpdated = now
	row := scheduledJobToRow(job)
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error; err != nil {
		return domain.ScheduledJob{}, err
	}
	return scheduledJobFromRow(row), nil
}

func (s *Store) GetScheduledJob(ctx context.Context, id string) (domain.ScheduledJob, error) {
	var row scheduledJobRow
	if err := s.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).First(&row).Error; err != nil {
		return domain.ScheduledJob{}, err
	}
	return scheduledJobFromRow(row), nil
}

func (s *Store) ListScheduledJobs(ctx context.Context, input domain.ScheduledJobListInput) ([]domain.ScheduledJob, error) {
	limit := input.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := s.db.WithContext(ctx).Model(&scheduledJobRow{})
	if strings.TrimSpace(input.SessionID) != "" {
		q = q.Where("session_id = ?", strings.TrimSpace(input.SessionID))
	}
	if strings.TrimSpace(input.Status) != "" {
		q = q.Where("status = ?", strings.TrimSpace(input.Status))
	}
	var rows []scheduledJobRow
	if err := q.Order("time_updated DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ScheduledJob, 0, len(rows))
	for _, row := range rows {
		out = append(out, scheduledJobFromRow(row))
	}
	return out, nil
}

func (s *Store) ListDueScheduledJobs(ctx context.Context, now string, limit int) ([]domain.ScheduledJob, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	var rows []scheduledJobRow
	err := s.db.WithContext(ctx).
		Where("status = ? AND next_run_at <> '' AND next_run_at <= ?", domain.ScheduledJobStatusActive, now).
		Order("next_run_at ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.ScheduledJob, 0, len(rows))
	for _, row := range rows {
		out = append(out, scheduledJobFromRow(row))
	}
	return out, nil
}

func (s *Store) DeleteScheduledJob(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&scheduledJobRow{ID: strings.TrimSpace(id)}).Error
}

func agentRunToRow(run domain.AgentRun) agentRunRow {
	return agentRunRow{ID: run.ID, ParentSessionID: run.ParentSessionID, SessionID: run.SessionID, Mode: run.Mode, Status: run.Status, Prompt: run.Prompt, Result: run.Result, Error: run.Error, Metadata: encodeStringMap(run.Metadata), TimeCreated: run.TimeCreated, TimeUpdated: run.TimeUpdated, TimeCompleted: run.TimeCompleted}
}

func agentRunFromRow(row agentRunRow) domain.AgentRun {
	return domain.AgentRun{ID: row.ID, ParentSessionID: row.ParentSessionID, SessionID: row.SessionID, Mode: persistedAgentMode(row.Mode), Status: row.Status, Prompt: row.Prompt, Result: row.Result, Error: row.Error, Metadata: decodeStringMap(row.Metadata), TimeCreated: row.TimeCreated, TimeUpdated: row.TimeUpdated, TimeCompleted: row.TimeCompleted}
}

func todoItemToRow(item domain.TodoItem) todoItemRow {
	return todoItemRow{ID: item.ID, SessionID: item.SessionID, ProjectPath: item.ProjectPath, Title: item.Title, Status: item.Status, Position: item.Position, OwnerMode: item.OwnerMode, SourceEventID: item.SourceEventID, Metadata: encodeStringMap(item.Metadata), TimeCreated: item.TimeCreated, TimeUpdated: item.TimeUpdated}
}

func todoItemFromRow(row todoItemRow) domain.TodoItem {
	return domain.TodoItem{ID: row.ID, SessionID: row.SessionID, ProjectPath: row.ProjectPath, Title: row.Title, Status: row.Status, Position: row.Position, OwnerMode: optionalPersistedAgentMode(row.OwnerMode), SourceEventID: row.SourceEventID, Metadata: decodeStringMap(row.Metadata), TimeCreated: row.TimeCreated, TimeUpdated: row.TimeUpdated}
}

func scheduledJobToRow(job domain.ScheduledJob) scheduledJobRow {
	return scheduledJobRow{ID: job.ID, SessionID: job.SessionID, Title: job.Title, Prompt: job.Prompt, Schedule: job.Schedule, WorkerMode: job.WorkerMode, Toolsets: encodeStrings(job.Toolsets), PermissionScope: job.PermissionScope, Status: job.Status, NextRunAt: job.NextRunAt, LastRunAt: job.LastRunAt, LastResult: job.LastResult, LastError: job.LastError, Metadata: encodeStringMap(job.Metadata), TimeCreated: job.TimeCreated, TimeUpdated: job.TimeUpdated}
}

func scheduledJobFromRow(row scheduledJobRow) domain.ScheduledJob {
	return domain.ScheduledJob{ID: row.ID, SessionID: row.SessionID, Title: row.Title, Prompt: row.Prompt, Schedule: row.Schedule, WorkerMode: persistedWorkerMode(row.WorkerMode), Toolsets: decodeStrings(row.Toolsets), PermissionScope: row.PermissionScope, Status: row.Status, NextRunAt: row.NextRunAt, LastRunAt: row.LastRunAt, LastResult: row.LastResult, LastError: row.LastError, Metadata: decodeStringMap(row.Metadata), TimeCreated: row.TimeCreated, TimeUpdated: row.TimeUpdated}
}
