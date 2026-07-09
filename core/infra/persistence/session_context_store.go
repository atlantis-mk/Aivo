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

func (s *Store) CreateSummary(ctx context.Context, summary domain.SessionSummary) error {
	if summary.ID == "" {
		summary.ID = uuid.NewString()
	}
	if summary.TimeCreated == "" {
		summary.TimeCreated = domain.NowString(time.Now())
	}
	row := summaryToRow(summary)
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return tx.Model(&sessionRow{}).Where("id = ?", summary.SessionID).Updates(map[string]any{"summary": summary.Summary, "time_updated": summary.TimeCreated}).Error
	})
}

func (s *Store) LatestSummary(ctx context.Context, sessionID string) (*domain.SessionSummary, error) {
	var row sessionSummaryRow
	err := s.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("time_created DESC").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	item := summaryFromRow(row)
	return &item, nil
}

func (s *Store) CreateCheckpoint(ctx context.Context, checkpoint domain.SessionCheckpoint) error {
	if checkpoint.ID == "" {
		checkpoint.ID = uuid.NewString()
	}
	if checkpoint.TimeCreated == "" {
		checkpoint.TimeCreated = domain.NowString(time.Now())
	}
	row := checkpointToRow(checkpoint)
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return tx.Model(&sessionRow{}).Where("id = ?", checkpoint.SessionID).Update("time_updated", checkpoint.TimeCreated).Error
	})
}

func (s *Store) ListCheckpoints(ctx context.Context, sessionID string, limit int) ([]domain.SessionCheckpoint, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []sessionCheckpointRow
	if err := s.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("time_created DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.SessionCheckpoint, 0, len(rows))
	for _, row := range rows {
		out = append(out, checkpointFromRow(row))
	}
	return out, nil
}

func (s *Store) LatestCheckpoint(ctx context.Context, sessionID string) (*domain.SessionCheckpoint, error) {
	items, err := s.ListCheckpoints(ctx, sessionID, 1)
	if err != nil || len(items) == 0 {
		return nil, err
	}
	return &items[0], nil
}

func (s *Store) UpsertCodingContext(ctx context.Context, cc domain.CodingContext) (domain.CodingContext, error) {
	now := domain.NowString(time.Now())
	if cc.ID == "" {
		cc.ID = uuid.NewString()
	}
	if cc.TimeCreated == "" {
		cc.TimeCreated = now
	}
	cc.TimeUpdated = now
	cc.ProjectPath = normalizeStoredPath(cc.ProjectPath)
	row := codingContextToRow(cc)
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "session_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"project_path": row.ProjectPath, "git_branch": row.GitBranch, "commit_sha": row.CommitSHA, "repo_url": row.RepoURL,
			"changed_files": row.ChangedFiles, "language_stack": row.LanguageStack, "package_manager": row.PackageManager,
			"cwd": row.CWD, "permissions": row.Permissions, "last_command": row.LastCommand, "time_updated": row.TimeUpdated,
		}),
	}).Create(&row).Error
	if err != nil {
		return domain.CodingContext{}, err
	}
	return s.GetCodingContext(ctx, cc.SessionID)
}

func (s *Store) GetCodingContext(ctx context.Context, sessionID string) (domain.CodingContext, error) {
	var row codingContextRow
	if err := s.db.WithContext(ctx).Where("session_id = ?", sessionID).First(&row).Error; err != nil {
		return domain.CodingContext{}, err
	}
	return codingContextFromRow(row), nil
}

func (s *Store) LatestSessionByProject(ctx context.Context, projectPath string) (*domain.Session, error) {
	items, err := s.ListRuntimeSessions(ctx, domain.ListSessionsRequest{Type: domain.SessionTypeCoding, ProjectPath: projectPath, Status: domain.SessionStatusActive, Limit: 1})
	if err != nil || len(items) == 0 {
		return nil, err
	}
	return &items[0], nil
}

func (s *Store) ForkRuntimeSession(ctx context.Context, source domain.Session, input domain.ForkSessionRequest) (domain.Session, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = source.Title + " (fork)"
	}
	goal := strings.TrimSpace(input.Goal)
	if goal == "" {
		goal = source.Goal
	}
	fork, err := s.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: source.Type, Source: source.Source, Title: title, Goal: goal, ProjectPath: source.ProjectPath, Model: source.Model, ModelSnapshot: source.ModelSnapshot, SystemPromptSnapshot: source.SystemPromptSnapshot, AgentMode: source.AgentMode, Metadata: source.Metadata})
	if err != nil {
		return domain.Session{}, err
	}
	if err := s.db.WithContext(ctx).Model(&sessionRow{}).Where("id = ?", fork.ID).Updates(map[string]any{"parent_session_id": source.ID, "forked_from_session_id": source.ID}).Error; err != nil {
		return domain.Session{}, err
	}
	if cc, err := s.GetCodingContext(ctx, source.ID); err == nil {
		cc.ID = uuid.NewString()
		cc.SessionID = fork.ID
		cc.TimeCreated = ""
		_, _ = s.UpsertCodingContext(ctx, cc)
	}
	if latest, err := s.LatestSummary(ctx, source.ID); err == nil && latest != nil {
		latest.ID = uuid.NewString()
		latest.SessionID = fork.ID
		latest.TimeCreated = ""
		_ = s.CreateSummary(ctx, *latest)
	}
	return s.GetRuntimeSession(ctx, fork.ID)
}

func (s *Store) projectIDForPath(ctx context.Context, projectPath string) (string, error) {
	if projectPath == "" {
		return "", nil
	}
	project, err := s.UpsertProject(ctx, projectPath)
	if err != nil {
		return "", err
	}
	return project.ID, nil
}

func (s *Store) sessionFromRow(ctx context.Context, row sessionRow) domain.Session {
	item := domain.Session{
		ID: row.ID, Type: defaultString(row.Type, domain.SessionTypeCoding), Status: defaultString(row.Status, domain.SessionStatusActive), Source: defaultString(row.Source, domain.SessionSourceDesktop),
		Title: row.Title, Goal: row.Goal, Summary: row.Summary, ParentSessionID: row.ParentSessionID, ForkedFromSessionID: row.ForkedFromSessionID,
		ModelSnapshot: row.ModelSnapshot, SystemPromptSnapshot: row.SystemPromptSnapshot, AgentMode: persistedAgentMode(row.AgentMode), TokenCount: row.TokenCount, CostMicros: row.CostMicros,
		Metadata: decodeStringMap(row.Metadata), ArchivedAt: row.ArchivedAt, DeletedAt: row.DeletedAt, TimeCreated: row.TimeCreated, TimeUpdated: row.TimeUpdated,
	}
	if row.ModelProviderID != "" && row.ModelID != "" {
		item.Model = &domain.ModelRef{ProviderID: row.ModelProviderID, ModelID: row.ModelID}
	}
	if row.ProjectID != "" {
		var project projectRow
		if err := s.db.WithContext(ctx).Where("id = ?", row.ProjectID).First(&project).Error; err == nil {
			item.ProjectPath = project.RootPath
		}
	}
	return item
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func persistedAgentMode(value string) string {
	mode, err := domain.NormalizeAgentMode(value)
	if err != nil {
		return domain.AgentModeAssistant
	}
	return mode
}

func optionalPersistedAgentMode(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return persistedAgentMode(value)
}

func persistedWorkerMode(value string) string {
	if strings.TrimSpace(value) == "" {
		return domain.AgentModeSchedulerWorker
	}
	mode, err := domain.NormalizeAgentMode(value)
	if err != nil {
		return domain.AgentModeSchedulerWorker
	}
	return mode
}

func summaryToRow(summary domain.SessionSummary) sessionSummaryRow {
	return sessionSummaryRow{ID: summary.ID, SessionID: summary.SessionID, FromEventID: summary.FromEventID, ToEventID: summary.ToEventID, Summary: summary.Summary, Facts: encodeStrings(summary.Facts), Decisions: encodeStrings(summary.Decisions), OpenTasks: encodeStrings(summary.OpenTasks), ChangedFiles: encodeStrings(summary.ChangedFiles), NextSuggestedAction: summary.NextSuggestedAction, TimeCreated: summary.TimeCreated}
}

func summaryFromRow(row sessionSummaryRow) domain.SessionSummary {
	return domain.SessionSummary{ID: row.ID, SessionID: row.SessionID, FromEventID: row.FromEventID, ToEventID: row.ToEventID, Summary: row.Summary, Facts: decodeStrings(row.Facts), Decisions: decodeStrings(row.Decisions), OpenTasks: decodeStrings(row.OpenTasks), ChangedFiles: decodeStrings(row.ChangedFiles), NextSuggestedAction: row.NextSuggestedAction, TimeCreated: row.TimeCreated}
}

func checkpointToRow(checkpoint domain.SessionCheckpoint) sessionCheckpointRow {
	return sessionCheckpointRow{ID: checkpoint.ID, SessionID: checkpoint.SessionID, Branch: checkpoint.Branch, CommitSHA: checkpoint.CommitSHA, ChangedFiles: encodeStrings(checkpoint.ChangedFiles), DiffSummary: checkpoint.DiffSummary, ConversationSummary: checkpoint.ConversationSummary, OpenTodos: encodeStrings(checkpoint.OpenTodos), KnownIssues: encodeStrings(checkpoint.KnownIssues), NextSuggestedAction: checkpoint.NextSuggestedAction, TimeCreated: checkpoint.TimeCreated}
}

func checkpointFromRow(row sessionCheckpointRow) domain.SessionCheckpoint {
	return domain.SessionCheckpoint{ID: row.ID, SessionID: row.SessionID, Branch: row.Branch, CommitSHA: row.CommitSHA, ChangedFiles: decodeStrings(row.ChangedFiles), DiffSummary: row.DiffSummary, ConversationSummary: row.ConversationSummary, OpenTodos: decodeStrings(row.OpenTodos), KnownIssues: decodeStrings(row.KnownIssues), NextSuggestedAction: row.NextSuggestedAction, TimeCreated: row.TimeCreated}
}

func codingContextToRow(cc domain.CodingContext) codingContextRow {
	return codingContextRow{ID: cc.ID, SessionID: cc.SessionID, ProjectPath: cc.ProjectPath, GitBranch: cc.GitBranch, CommitSHA: cc.CommitSHA, RepoURL: cc.RepoURL, ChangedFiles: encodeStrings(cc.ChangedFiles), LanguageStack: encodeStrings(cc.LanguageStack), PackageManager: cc.PackageManager, CWD: cc.CWD, Permissions: encodeStrings(cc.Permissions), LastCommand: cc.LastCommand, TimeCreated: cc.TimeCreated, TimeUpdated: cc.TimeUpdated}
}

func codingContextFromRow(row codingContextRow) domain.CodingContext {
	return domain.CodingContext{ID: row.ID, SessionID: row.SessionID, ProjectPath: row.ProjectPath, GitBranch: row.GitBranch, CommitSHA: row.CommitSHA, RepoURL: row.RepoURL, ChangedFiles: decodeStrings(row.ChangedFiles), LanguageStack: decodeStrings(row.LanguageStack), PackageManager: row.PackageManager, CWD: row.CWD, Permissions: decodeStrings(row.Permissions), LastCommand: row.LastCommand, TimeCreated: row.TimeCreated, TimeUpdated: row.TimeUpdated}
}
