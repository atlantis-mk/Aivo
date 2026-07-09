package persistence

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"aivo/core/domain"
)

func (s *Store) CreatePermissionRequest(ctx context.Context, request domain.PermissionRequest) (domain.PermissionRequest, error) {
	now := domain.NowString(time.Now())
	if request.ID == "" {
		request.ID = uuid.NewString()
	}
	if request.Status == "" {
		request.Status = domain.PermissionRequestStatusPending
	}
	if request.TimeCreated == "" {
		request.TimeCreated = now
	}
	request.TimeUpdated = now
	row := permissionRequestRow{
		ID: request.ID, SessionID: request.SessionID, TurnID: request.TurnID, ToolCallID: request.ToolCallID,
		ToolName: request.ToolName, Action: request.Action, Paths: encodeStrings(request.Paths),
		Arguments: encodeAnyMap(request.Arguments), Status: request.Status, Remember: boolInt(request.Remember),
		Reason: request.Reason, TimeCreated: request.TimeCreated, TimeUpdated: request.TimeUpdated,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return domain.PermissionRequest{}, err
	}
	return permissionRequestFromRow(row), nil
}

func (s *Store) GetPermissionRequest(ctx context.Context, requestID string) (domain.PermissionRequest, error) {
	var row permissionRequestRow
	if err := s.db.WithContext(ctx).Where("id = ?", requestID).First(&row).Error; err != nil {
		return domain.PermissionRequest{}, err
	}
	return permissionRequestFromRow(row), nil
}

func (s *Store) ListPermissionRequests(ctx context.Context, sessionID string, status string) ([]domain.PermissionRequest, error) {
	q := s.db.WithContext(ctx).Model(&permissionRequestRow{})
	if strings.TrimSpace(sessionID) != "" {
		q = q.Where("session_id = ?", strings.TrimSpace(sessionID))
	}
	if strings.TrimSpace(status) != "" {
		q = q.Where("status = ?", strings.TrimSpace(status))
	}
	var rows []permissionRequestRow
	if err := q.Order("time_created DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.PermissionRequest, 0, len(rows))
	for _, row := range rows {
		out = append(out, permissionRequestFromRow(row))
	}
	return out, nil
}

func (s *Store) UpdatePermissionRequest(ctx context.Context, requestID string, status string, remember bool, reason string) (domain.PermissionRequest, error) {
	now := domain.NowString(time.Now())
	updates := map[string]any{"status": status, "remember": boolInt(remember), "reason": strings.TrimSpace(reason), "time_updated": now}
	if err := s.db.WithContext(ctx).Model(&permissionRequestRow{}).Where("id = ?", requestID).Updates(updates).Error; err != nil {
		return domain.PermissionRequest{}, err
	}
	return s.GetPermissionRequest(ctx, requestID)
}

func (s *Store) CreateQuestionRequest(ctx context.Context, request domain.QuestionRequest) (domain.QuestionRequest, error) {
	now := domain.NowString(time.Now())
	if request.ID == "" {
		request.ID = uuid.NewString()
	}
	if request.ToolName == "" {
		request.ToolName = "question"
	}
	if request.Status == "" {
		request.Status = domain.QuestionRequestStatusPending
	}
	if request.TimeCreated == "" {
		request.TimeCreated = now
	}
	request.TimeUpdated = now
	row := questionRequestRow{
		ID: request.ID, SessionID: request.SessionID, TurnID: request.TurnID, ToolCallID: request.ToolCallID,
		ToolName: request.ToolName, Questions: encodeQuestionPrompts(request.Questions), Answers: encodeStringMatrix(request.Answers),
		Status: request.Status, Reason: request.Reason, Arguments: encodeAnyMap(request.Arguments),
		TimeCreated: request.TimeCreated, TimeUpdated: request.TimeUpdated,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return domain.QuestionRequest{}, err
	}
	return questionRequestFromRow(row), nil
}

func (s *Store) GetQuestionRequest(ctx context.Context, requestID string) (domain.QuestionRequest, error) {
	var row questionRequestRow
	if err := s.db.WithContext(ctx).Where("id = ?", requestID).First(&row).Error; err != nil {
		return domain.QuestionRequest{}, err
	}
	return questionRequestFromRow(row), nil
}

func (s *Store) ListQuestionRequests(ctx context.Context, sessionID string, status string) ([]domain.QuestionRequest, error) {
	q := s.db.WithContext(ctx).Model(&questionRequestRow{})
	if strings.TrimSpace(sessionID) != "" {
		q = q.Where("session_id = ?", strings.TrimSpace(sessionID))
	}
	if strings.TrimSpace(status) != "" {
		q = q.Where("status = ?", strings.TrimSpace(status))
	}
	var rows []questionRequestRow
	if err := q.Order("time_created DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.QuestionRequest, 0, len(rows))
	for _, row := range rows {
		out = append(out, questionRequestFromRow(row))
	}
	return out, nil
}

func (s *Store) UpdateQuestionRequest(ctx context.Context, requestID string, status string, answers [][]string, reason string) (domain.QuestionRequest, error) {
	now := domain.NowString(time.Now())
	updates := map[string]any{"status": status, "answers": encodeStringMatrix(answers), "reason": strings.TrimSpace(reason), "time_updated": now}
	if err := s.db.WithContext(ctx).Model(&questionRequestRow{}).Where("id = ?", requestID).Updates(updates).Error; err != nil {
		return domain.QuestionRequest{}, err
	}
	return s.GetQuestionRequest(ctx, requestID)
}

func (s *Store) SavePermissionRule(ctx context.Context, rule domain.PermissionRule) (domain.PermissionRule, error) {
	now := domain.NowString(time.Now())
	if rule.ID == "" {
		rule.ID = uuid.NewString()
	}
	if rule.Scope == "" {
		rule.Scope = "workspace"
	}
	if rule.TimeCreated == "" {
		rule.TimeCreated = now
	}
	rule.TimeUpdated = now
	row := permissionRuleRow{
		ID: rule.ID, Scope: rule.Scope, SessionID: rule.SessionID, WorkspaceRoot: normalizeStoredPath(rule.WorkspaceRoot),
		ToolName: rule.ToolName, Action: rule.Action, Decision: rule.Decision, Paths: encodeStrings(rule.Paths),
		TimeCreated: rule.TimeCreated, TimeUpdated: rule.TimeUpdated,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return domain.PermissionRule{}, err
	}
	return permissionRuleFromRow(row), nil
}

func (s *Store) ListPermissionRules(ctx context.Context, workspaceRoot string, sessionID string) ([]domain.PermissionRule, error) {
	q := s.db.WithContext(ctx).Model(&permissionRuleRow{})
	workspaceRoot = normalizeStoredPath(workspaceRoot)
	if workspaceRoot != "" && strings.TrimSpace(sessionID) != "" {
		q = q.Where("workspace_root = ? OR session_id = ?", workspaceRoot, strings.TrimSpace(sessionID))
	} else if workspaceRoot != "" {
		q = q.Where("workspace_root = ?", workspaceRoot)
	} else if strings.TrimSpace(sessionID) != "" {
		q = q.Where("session_id = ?", strings.TrimSpace(sessionID))
	}
	var rows []permissionRuleRow
	if err := q.Order("time_created DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.PermissionRule, 0, len(rows))
	for _, row := range rows {
		out = append(out, permissionRuleFromRow(row))
	}
	return out, nil
}

func permissionRequestFromRow(row permissionRequestRow) domain.PermissionRequest {
	return domain.PermissionRequest{
		ID: row.ID, SessionID: row.SessionID, TurnID: row.TurnID, ToolCallID: row.ToolCallID,
		ToolName: row.ToolName, Action: row.Action, Paths: decodeStrings(row.Paths),
		Arguments: decodeAnyMap(row.Arguments), Status: row.Status, Remember: row.Remember != 0,
		Reason: row.Reason, TimeCreated: row.TimeCreated, TimeUpdated: row.TimeUpdated,
	}
}

func questionRequestFromRow(row questionRequestRow) domain.QuestionRequest {
	return domain.QuestionRequest{
		ID: row.ID, SessionID: row.SessionID, TurnID: row.TurnID, ToolCallID: row.ToolCallID,
		ToolName: row.ToolName, Questions: decodeQuestionPrompts(row.Questions), Answers: decodeStringMatrix(row.Answers),
		Status: row.Status, Reason: row.Reason, Arguments: decodeAnyMap(row.Arguments),
		TimeCreated: row.TimeCreated, TimeUpdated: row.TimeUpdated,
	}
}

func permissionRuleFromRow(row permissionRuleRow) domain.PermissionRule {
	return domain.PermissionRule{
		ID: row.ID, Scope: row.Scope, SessionID: row.SessionID, WorkspaceRoot: row.WorkspaceRoot,
		ToolName: row.ToolName, Action: row.Action, Decision: row.Decision, Paths: decodeStrings(row.Paths),
		TimeCreated: row.TimeCreated, TimeUpdated: row.TimeUpdated,
	}
}
