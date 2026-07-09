package persistence

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"

	"aivo/core/domain"
)

func (s *Store) UpsertProject(ctx context.Context, rootPath string) (domain.AssistantProject, error) {
	now := domain.NowString(time.Now())
	name := filepath.Base(strings.TrimRight(rootPath, string(os.PathSeparator)))
	row := projectRow{ID: uuid.NewString(), Name: name, RootPath: rootPath, TimeOpened: now, TimeUpdated: now}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "root_path"}},
		DoUpdates: clause.Assignments(map[string]any{"sidebar_hidden": 0, "time_updated": now}),
	}).Create(&row).Error
	if err != nil {
		return domain.AssistantProject{}, err
	}
	var saved projectRow
	if err := s.db.WithContext(ctx).Where("root_path = ?", rootPath).First(&saved).Error; err != nil {
		return domain.AssistantProject{}, err
	}
	return projectFromRow(saved), nil
}

func (s *Store) SetProjectSidebarHidden(ctx context.Context, rootPath string, hidden bool) (domain.AssistantProject, error) {
	rootPath = strings.TrimSpace(rootPath)
	if rootPath == "" {
		return domain.AssistantProject{}, errors.New("project path is required")
	}
	project, err := s.UpsertProject(ctx, rootPath)
	if err != nil {
		return domain.AssistantProject{}, err
	}
	now := domain.NowString(time.Now())
	hiddenValue := 0
	if hidden {
		hiddenValue = 1
	}
	if err := s.db.WithContext(ctx).
		Model(&projectRow{}).
		Where("id = ?", project.ID).
		Updates(map[string]any{"sidebar_hidden": hiddenValue, "time_updated": now}).
		Error; err != nil {
		return domain.AssistantProject{}, err
	}
	var saved projectRow
	if err := s.db.WithContext(ctx).Where("id = ?", project.ID).First(&saved).Error; err != nil {
		return domain.AssistantProject{}, err
	}
	return projectFromRow(saved), nil
}

func (s *Store) ListProjects(ctx context.Context, limit int) ([]domain.AssistantProject, error) {
	if limit <= 0 {
		limit = 20
	}
	var rows []projectRow
	if err := s.db.WithContext(ctx).
		Where("sidebar_hidden = ?", 0).
		Order("time_updated DESC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	projects := make([]domain.AssistantProject, 0, len(rows))
	for _, row := range rows {
		projects = append(projects, projectFromRow(row))
	}
	return projects, nil
}

func projectFromRow(row projectRow) domain.AssistantProject {
	return domain.AssistantProject{ID: row.ID, Name: row.Name, RootPath: row.RootPath, GitBranch: row.GitBranch, GitDirty: row.GitDirty == 1, GitAvailable: row.GitAvailable == 1, SidebarHidden: row.SidebarHidden == 1, TimeOpened: row.TimeOpened, TimeUpdated: row.TimeUpdated}
}
