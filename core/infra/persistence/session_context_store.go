package persistence

import (
	"context"
	"errors"
	"fmt"
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
	if err := s.copyForkHistory(ctx, source.ID, fork.ID, strings.TrimSpace(input.AtEventID)); err != nil {
		_ = s.db.WithContext(ctx).Delete(&sessionRow{}, "id = ?", fork.ID).Error
		return domain.Session{}, err
	}
	return s.GetRuntimeSession(ctx, fork.ID)
}

func (s *Store) copyForkHistory(ctx context.Context, sourceID string, forkID string, atEventID string) error {
	var allEvents []sessionEventRow
	if err := s.db.WithContext(ctx).Where("session_id = ?", sourceID).Order("time_created ASC, id ASC").Find(&allEvents).Error; err != nil {
		return err
	}
	boundaryTime := ""
	boundaryIndex := len(allEvents) - 1
	if atEventID != "" {
		boundaryIndex = -1
		for index, event := range allEvents {
			if event.ID == atEventID {
				boundaryIndex = index
				boundaryTime = event.TimeCreated
				break
			}
		}
		if boundaryIndex < 0 {
			return fmt.Errorf("fork event boundary %q does not belong to session", atEventID)
		}
	} else if len(allEvents) > 0 {
		boundaryTime = allEvents[len(allEvents)-1].TimeCreated
	}

	selectedEvents := make([]sessionEventRow, 0, boundaryIndex+1)
	turnIDs := map[string]bool{}
	for index, event := range allEvents {
		if index > boundaryIndex {
			break
		}
		if event.Visibility != domain.EventVisibilityNormal {
			continue
		}
		selectedEvents = append(selectedEvents, event)
		if event.TurnID != "" {
			turnIDs[event.TurnID] = true
		}
	}

	var turns []turnRow
	if len(turnIDs) > 0 {
		ids := stringSetKeys(turnIDs)
		if err := s.db.WithContext(ctx).Where("session_id = ? AND id IN ?", sourceID, ids).Order("time_created ASC").Find(&turns).Error; err != nil {
			return err
		}
	}
	eventIDs := map[string]bool{}
	for _, event := range selectedEvents {
		eventIDs[event.ID] = true
	}
	var tools []toolCallRow
	toolQuery := s.db.WithContext(ctx).Where("session_id = ?", sourceID)
	switch {
	case len(turnIDs) > 0 && len(eventIDs) > 0:
		toolQuery = toolQuery.Where("turn_id IN ? OR event_id IN ?", stringSetKeys(turnIDs), stringSetKeys(eventIDs))
	case len(turnIDs) > 0:
		toolQuery = toolQuery.Where("turn_id IN ?", stringSetKeys(turnIDs))
	case len(eventIDs) > 0:
		toolQuery = toolQuery.Where("event_id IN ?", stringSetKeys(eventIDs))
	default:
		toolQuery = toolQuery.Where("1 = 0")
	}
	if err := toolQuery.Order("time_created ASC").Find(&tools).Error; err != nil {
		return err
	}

	var summaries []sessionSummaryRow
	var checkpoints []sessionCheckpointRow
	summaryQuery := s.db.WithContext(ctx).Where("session_id = ?", sourceID)
	checkpointQuery := s.db.WithContext(ctx).Where("session_id = ?", sourceID)
	if boundaryTime != "" {
		summaryQuery = summaryQuery.Where("time_created <= ?", boundaryTime)
		checkpointQuery = checkpointQuery.Where("time_created <= ?", boundaryTime)
	}
	if err := summaryQuery.Order("time_created ASC").Find(&summaries).Error; err != nil {
		return err
	}
	if err := checkpointQuery.Order("time_created ASC").Find(&checkpoints).Error; err != nil {
		return err
	}
	var codingContexts []codingContextRow
	if err := s.db.WithContext(ctx).Where("session_id = ?", sourceID).Find(&codingContexts).Error; err != nil {
		return err
	}

	turnMap := map[string]string{}
	eventMap := map[string]string{}
	toolMap := map[string]string{}
	summaryMap := map[string]string{}
	checkpointMap := map[string]string{}
	for _, row := range turns {
		turnMap[row.ID] = uuid.NewString()
	}
	for _, row := range selectedEvents {
		eventMap[row.ID] = uuid.NewString()
	}
	for _, row := range tools {
		toolMap[row.ID] = uuid.NewString()
	}
	for _, row := range summaries {
		summaryMap[row.ID] = uuid.NewString()
	}
	for _, row := range checkpoints {
		checkpointMap[row.ID] = uuid.NewString()
	}
	idMap := mergeIDMaps(turnMap, eventMap, toolMap, summaryMap, checkpointMap)

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, row := range turns {
			row.ID = turnMap[row.ID]
			row.SessionID = forkID
			row.UserEventID = eventMap[row.UserEventID]
			if row.Status == domain.TurnStatusRunning {
				row.Status = domain.TurnStatusCancelled
				row.Error = "forked without active execution ownership"
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		for _, row := range selectedEvents {
			row.ID = eventMap[row.ID]
			row.SessionID = forkID
			row.TurnID = turnMap[row.TurnID]
			row.Payload = remapEncodedIDs(row.Payload, idMap)
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		for _, row := range tools {
			row.ID = toolMap[row.ID]
			row.SessionID = forkID
			row.TurnID = turnMap[row.TurnID]
			row.EventID = eventMap[row.EventID]
			if row.Status == domain.ToolCallStatusRunning || row.Status == domain.ToolCallStatusPending {
				row.Status = domain.ToolCallStatusInterrupted
				row.Error = "forked without active execution ownership"
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		for _, row := range summaries {
			row.ID = summaryMap[row.ID]
			row.SessionID = forkID
			row.FromEventID = eventMap[row.FromEventID]
			row.ToEventID = eventMap[row.ToEventID]
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		for _, row := range checkpoints {
			row.ID = checkpointMap[row.ID]
			row.SessionID = forkID
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		for _, row := range codingContexts {
			row.ID = uuid.NewString()
			row.SessionID = forkID
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		lastEventID := ""
		if len(selectedEvents) > 0 {
			lastEventID = eventMap[selectedEvents[len(selectedEvents)-1].ID]
		}
		state := sessionExecutionStateRow{
			ID: uuid.NewString(), SessionID: forkID, Status: domain.ExecutionStatusIdle,
			Reason: "forked history ready", LastEventID: lastEventID,
			TimeCreated: domain.NowString(time.Now()), TimeUpdated: domain.NowString(time.Now()),
		}
		if err := tx.Create(&state).Error; err != nil {
			return err
		}
		updates := map[string]any{"time_updated": domain.NowString(time.Now())}
		if len(summaries) > 0 {
			updates["summary"] = summaries[len(summaries)-1].Summary
		}
		return tx.Model(&sessionRow{}).Where("id = ?", forkID).Updates(updates).Error
	})
}

func stringSetKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func mergeIDMaps(maps ...map[string]string) map[string]string {
	out := map[string]string{}
	for _, values := range maps {
		for key, value := range values {
			out[key] = value
		}
	}
	return out
}

func remapEncodedIDs(raw string, ids map[string]string) string {
	value := decodeAnyMap(raw)
	var remap func(any) any
	remap = func(input any) any {
		switch typed := input.(type) {
		case string:
			if replacement := ids[typed]; replacement != "" {
				return replacement
			}
			return typed
		case []any:
			for index := range typed {
				typed[index] = remap(typed[index])
			}
			return typed
		case map[string]any:
			for key := range typed {
				typed[key] = remap(typed[key])
			}
			return typed
		default:
			return input
		}
	}
	mapped, _ := remap(value).(map[string]any)
	return encodeAnyMap(mapped)
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
