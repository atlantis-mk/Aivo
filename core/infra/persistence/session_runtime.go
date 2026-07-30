package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"aivo/core/domain"
)

func (s *Store) migrateSessionRuntime(ctx context.Context) error {
	if err := s.db.WithContext(ctx).AutoMigrate(&turnRow{}, &sessionEventRow{}, &toolCallRow{}, &sessionExecutionStateRow{}, &pendingSessionInputRow{}, &permissionRequestRow{}, &questionRequestRow{}, &sessionSummaryRow{}, &sessionCheckpointRow{}, &codingContextRow{}, &gitWorktreeRow{}, &agentRunRow{}, &todoItemRow{}, &scheduledJobRow{}, &pluginInstallRow{}, &pluginDiagnosticRow{}, &mcpServerRow{}, &mcpToolRow{}, &mcpPromptRow{}, &mcpResourceRow{}, &skillRow{}, &skillSourceRow{}, &skillImportCandidateRow{}, &toolRegistrationRow{}); err != nil {
		return err
	}
	return nil
}

func (s *Store) CreateRuntimeSession(ctx context.Context, input domain.CreateSessionRequest) (domain.Session, error) {
	now := domain.NowString(time.Now())
	sessionType, err := domain.NormalizeSessionType(input.Type)
	if err != nil {
		return domain.Session{}, err
	}
	source, err := domain.NormalizeSessionSource(input.Source)
	if err != nil {
		return domain.Session{}, err
	}
	agentModeInput := strings.TrimSpace(input.AgentMode)
	if agentModeInput == "" && sessionType == domain.SessionTypeCoding {
		agentModeInput = domain.AgentModeCode
	}
	agentMode, err := domain.NormalizeAgentMode(agentModeInput)
	if err != nil {
		return domain.Session{}, err
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = fallbackTitle(input.Goal)
	}
	projectPath := normalizeStoredPath(input.ProjectPath)
	session := domain.Session{
		ID: uuid.NewString(), Type: sessionType, Status: domain.SessionStatusActive, Source: source,
		Title: title, Goal: strings.TrimSpace(input.Goal), ProjectPath: projectPath, Model: input.Model,
		ModelSnapshot: strings.TrimSpace(input.ModelSnapshot), SystemPromptSnapshot: strings.TrimSpace(input.SystemPromptSnapshot),
		AgentMode: agentMode, Metadata: input.Metadata, TimeCreated: now, TimeUpdated: now,
	}
	var providerID, modelID string
	if input.Model != nil {
		providerID = input.Model.ProviderID
		modelID = input.Model.ModelID
	}
	projectID, err := s.projectIDForPath(ctx, projectPath)
	if err != nil {
		return domain.Session{}, err
	}
	row := sessionRow{
		ID: session.ID, Title: session.Title, ProjectID: projectID, ModelProviderID: providerID, ModelID: modelID,
		TimeCreated: now, TimeUpdated: now, Type: session.Type, Status: session.Status, Source: session.Source,
		Goal: session.Goal, Summary: "", ModelSnapshot: session.ModelSnapshot, SystemPromptSnapshot: session.SystemPromptSnapshot,
		AgentMode: session.AgentMode, Metadata: encodeStringMap(session.Metadata),
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return domain.Session{}, err
	}
	return session, nil
}

func (s *Store) GetRuntimeSession(ctx context.Context, id string) (domain.Session, error) {
	var row sessionRow
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if err != nil {
		return domain.Session{}, err
	}
	return s.sessionFromRow(ctx, row), nil
}

func (s *Store) ListRuntimeSessions(ctx context.Context, filter domain.ListSessionsRequest) ([]domain.Session, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := s.db.WithContext(ctx).Model(&sessionRow{})
	if !filter.IncludeDeleted {
		q = q.Where("COALESCE(status, ?) <> ?", domain.SessionStatusActive, domain.SessionStatusDeleted)
	}
	if filter.Type != "" {
		q = q.Where("COALESCE(type, ?) = ?", domain.SessionTypeCoding, filter.Type)
	}
	if filter.Status != "" {
		q = q.Where("COALESCE(status, ?) = ?", domain.SessionStatusActive, filter.Status)
	}
	if filter.Source != "" {
		q = q.Where("COALESCE(source, ?) = ?", domain.SessionSourceDesktop, filter.Source)
	}
	projectPath := normalizeStoredPath(filter.ProjectPath)
	if projectPath != "" {
		q = q.Where("EXISTS (SELECT 1 FROM coding_contexts cc WHERE cc.session_id = sessions.id AND cc.project_path = ?)", projectPath)
	}
	search := strings.TrimSpace(filter.Search)
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("(title LIKE ? OR COALESCE(summary, '') LIKE ? OR EXISTS (SELECT 1 FROM coding_contexts cc WHERE cc.session_id = sessions.id AND cc.project_path LIKE ?) OR EXISTS (SELECT 1 FROM session_events e WHERE e.session_id = sessions.id AND e.visibility = ? AND e.content LIKE ?))", like, like, like, domain.EventVisibilityNormal, like)
	}
	var rows []sessionRow
	if err := q.Order("time_updated DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Session, 0, len(rows))
	for _, row := range rows {
		out = append(out, s.sessionFromRow(ctx, row))
	}
	return out, nil
}

func (s *Store) UpdateRuntimeSession(ctx context.Context, input domain.UpdateSessionRequest) (domain.Session, error) {
	current, err := s.GetRuntimeSession(ctx, input.SessionID)
	if err != nil {
		return domain.Session{}, err
	}
	if strings.TrimSpace(input.Title) != "" {
		current.Title = strings.TrimSpace(input.Title)
	}
	if strings.TrimSpace(input.Goal) != "" {
		current.Goal = strings.TrimSpace(input.Goal)
	}
	if strings.TrimSpace(input.Summary) != "" {
		current.Summary = strings.TrimSpace(input.Summary)
	}
	if strings.TrimSpace(input.Status) != "" {
		status, err := domain.NormalizeSessionStatus(input.Status)
		if err != nil {
			return domain.Session{}, err
		}
		current.Status = status
	}
	current.TimeUpdated = domain.NowString(time.Now())
	err = s.db.WithContext(ctx).Model(&sessionRow{}).Where("id = ?", current.ID).Updates(map[string]any{
		"title": current.Title, "goal": current.Goal, "summary": current.Summary, "status": current.Status, "time_updated": current.TimeUpdated,
	}).Error
	if err != nil {
		return domain.Session{}, err
	}
	return current, nil
}

func (s *Store) SetRuntimeSessionStatus(ctx context.Context, id string, status string) (domain.Session, error) {
	nextStatus, err := domain.NormalizeSessionStatus(status)
	if err != nil {
		return domain.Session{}, err
	}
	now := domain.NowString(time.Now())
	updates := map[string]any{"status": nextStatus, "time_updated": now}
	if nextStatus == domain.SessionStatusArchived {
		updates["archived_at"] = now
	}
	if nextStatus == domain.SessionStatusDeleted {
		updates["deleted_at"] = now
	}
	if err := s.db.WithContext(ctx).Model(&sessionRow{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return domain.Session{}, err
	}
	return s.GetRuntimeSession(ctx, id)
}

func (s *Store) SetRuntimeSessionAgentMode(ctx context.Context, id string, mode string) (domain.Session, error) {
	agentMode, err := domain.NormalizeAgentMode(mode)
	if err != nil {
		return domain.Session{}, err
	}
	now := domain.NowString(time.Now())
	if err := s.db.WithContext(ctx).Model(&sessionRow{}).Where("id = ?", strings.TrimSpace(id)).Updates(map[string]any{
		"agent_mode":   agentMode,
		"time_updated": now,
	}).Error; err != nil {
		return domain.Session{}, err
	}
	return s.GetRuntimeSession(ctx, id)
}

func (s *Store) SetRuntimeSessionProject(ctx context.Context, sessionID string, projectPath string) (domain.Session, error) {
	projectPath = normalizeStoredPath(projectPath)
	if strings.TrimSpace(sessionID) == "" || projectPath == "" {
		return domain.Session{}, errors.New("sessionId and project path are required")
	}
	projectID, err := s.projectIDForPath(ctx, projectPath)
	if err != nil {
		return domain.Session{}, err
	}
	now := domain.NowString(time.Now())
	if err := s.db.WithContext(ctx).Model(&sessionRow{}).Where("id = ?", sessionID).Updates(map[string]any{
		"project_id": projectID, "time_updated": now,
	}).Error; err != nil {
		return domain.Session{}, err
	}
	return s.GetRuntimeSession(ctx, sessionID)
}

func normalizeStoredPath(path string) string {
	clean := strings.TrimSpace(path)
	if clean == "" {
		return ""
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return clean
	}
	return abs
}

func fallbackTitle(text string) string {
	clean := strings.TrimSpace(text)
	if clean == "" {
		return "Untitled session"
	}
	if len(clean) > 80 {
		return clean[:80]
	}
	return clean
}

func encodeStrings(values []string) string {
	if len(values) == 0 {
		return ""
	}
	raw, _ := json.Marshal(values)
	return string(raw)
}

func decodeStrings(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func encodeStringMatrix(values [][]string) string {
	if len(values) == 0 {
		return ""
	}
	raw, _ := json.Marshal(values)
	return string(raw)
}

func decodeStringMatrix(raw string) [][]string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out [][]string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func encodeQuestionPrompts(values []domain.QuestionPrompt) string {
	if len(values) == 0 {
		return "[]"
	}
	raw, _ := json.Marshal(values)
	return string(raw)
}

func decodeQuestionPrompts(raw string) []domain.QuestionPrompt {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []domain.QuestionPrompt
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func encodeAnyMap(values map[string]any) string {
	if len(values) == 0 {
		return ""
	}
	raw, _ := json.Marshal(values)
	return string(raw)
}

func decodeAnyMap(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func mergeAnyMetadata(base map[string]any, overlay map[string]any) map[string]any {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	merged := make(map[string]any, len(base)+len(overlay))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range overlay {
		merged[key] = value
	}
	return merged
}

func encodeStringMap(values map[string]string) string {
	if len(values) == 0 {
		return ""
	}
	raw, _ := json.Marshal(values)
	return string(raw)
}

func decodeStringMap(raw string) map[string]string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out map[string]string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
