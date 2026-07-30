package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aivo/core/domain"
)

func (s *Store) SaveGitWorktree(ctx context.Context, worktree domain.GitWorktree) error {
	row := gitWorktreeRow{
		ID: worktree.ID, RepositoryRoot: worktree.RepositoryRoot, Path: worktree.Path, Branch: worktree.Branch, BaseRef: worktree.BaseRef,
		Head: worktree.Head, Status: worktree.Status, Managed: boolInt(worktree.Managed), OwnsBranch: boolInt(worktree.OwnsBranch), Detached: boolInt(worktree.Detached), Error: worktree.Error,
		TimeCreated: worktree.TimeCreated, TimeUpdated: worktree.TimeUpdated,
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"repository_root", "path", "branch", "base_ref", "head", "status", "managed", "owns_branch", "detached", "error", "time_updated"}),
	}).Create(&row).Error
}

func (s *Store) GetGitWorktree(ctx context.Context, id string) (domain.GitWorktree, error) {
	var row gitWorktreeRow
	if err := s.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).First(&row).Error; err != nil {
		return domain.GitWorktree{}, err
	}
	return gitWorktreeFromRow(row), nil
}

func (s *Store) ListGitWorktrees(ctx context.Context, repositoryRoot string, includeRemoved bool) ([]domain.GitWorktree, error) {
	query := s.db.WithContext(ctx)
	if strings.TrimSpace(repositoryRoot) != "" {
		query = query.Where("repository_root = ?", repositoryRoot)
	}
	if !includeRemoved {
		query = query.Where("status <> ?", domain.WorktreeStatusRemoved)
	}
	var rows []gitWorktreeRow
	if err := query.Order("time_updated DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.GitWorktree, 0, len(rows))
	for _, row := range rows {
		out = append(out, gitWorktreeFromRow(row))
	}
	return out, nil
}

func (s *Store) UpdateGitWorktreeStatus(ctx context.Context, id string, status string, head string, branch string, errText string) (domain.GitWorktree, error) {
	updates := map[string]any{"status": status, "head": head, "branch": branch, "error": errText, "time_updated": domain.NowString(time.Now())}
	result := s.db.WithContext(ctx).Model(&gitWorktreeRow{}).Where("id = ?", strings.TrimSpace(id)).Updates(updates)
	if result.Error != nil {
		return domain.GitWorktree{}, result.Error
	}
	if result.RowsAffected == 0 {
		return domain.GitWorktree{}, gorm.ErrRecordNotFound
	}
	return s.GetGitWorktree(ctx, id)
}

func (s *Store) ActiveSessionIDsForProjectPath(ctx context.Context, projectPath string) ([]string, error) {
	var ids []string
	err := s.db.WithContext(ctx).Table("coding_contexts").
		Select("coding_contexts.session_id").
		Joins("JOIN session_execution_states ON session_execution_states.session_id = coding_contexts.session_id").
		Where("coding_contexts.project_path = ? AND session_execution_states.status IN ?", projectPath, []string{domain.ExecutionStatusRunning, domain.ExecutionStatusCompacting}).
		Pluck("coding_contexts.session_id", &ids).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return ids, err
}

func gitWorktreeFromRow(row gitWorktreeRow) domain.GitWorktree {
	return domain.GitWorktree{
		ID: row.ID, RepositoryRoot: row.RepositoryRoot, Path: row.Path, Branch: row.Branch, BaseRef: row.BaseRef, Head: row.Head,
		Status: row.Status, Managed: row.Managed == 1, OwnsBranch: row.OwnsBranch == 1, Detached: row.Detached == 1, Error: row.Error, TimeCreated: row.TimeCreated, TimeUpdated: row.TimeUpdated,
	}
}
