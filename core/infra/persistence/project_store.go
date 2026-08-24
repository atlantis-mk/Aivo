package persistence

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aivo/core/domain"
)

func (s *Store) UpsertProject(ctx context.Context, rootPath string) (domain.AssistantProject, error) {
	result, err := s.RegisterProject(ctx, rootPath)
	return result.Project, err
}

func (s *Store) RegisterProject(ctx context.Context, rootPath string) (domain.ProjectRegistrationResult, error) {
	now := domain.NowString(time.Now())
	name := filepath.Base(strings.TrimRight(rootPath, string(os.PathSeparator)))
	row := projectRow{ID: uuid.NewString(), Name: name, RootPath: rootPath, TimeOpened: now, TimeUpdated: now}
	var saved projectRow
	status := domain.ProjectRegistrationExisting
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		created := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
		if created.Error != nil {
			return created.Error
		}
		if created.RowsAffected == 1 {
			status = domain.ProjectRegistrationCreated
			saved = row
			return nil
		}
		if err := tx.Where("root_path = ?", rootPath).First(&saved).Error; err != nil {
			return err
		}
		if saved.SidebarHidden == 1 {
			status = domain.ProjectRegistrationRestored
		}
		if err := tx.Model(&projectRow{}).Where("id = ?", saved.ID).Updates(map[string]any{
			"sidebar_hidden": 0,
			"time_updated":   now,
		}).Error; err != nil {
			return err
		}
		saved.SidebarHidden = 0
		saved.TimeUpdated = now
		return nil
	})
	if err != nil {
		return domain.ProjectRegistrationResult{}, err
	}
	return domain.ProjectRegistrationResult{Project: projectFromRow(saved), Status: status}, nil
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

func (s *Store) GetProjectByID(ctx context.Context, id string) (domain.AssistantProject, bool, error) {
	var row projectRow
	if err := s.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.AssistantProject{}, false, nil
		}
		return domain.AssistantProject{}, false, err
	}
	return projectFromRow(row), true, nil
}

func (s *Store) GetProjectByPath(ctx context.Context, rootPath string) (domain.AssistantProject, bool, error) {
	var row projectRow
	if err := s.db.WithContext(ctx).Where("root_path = ?", normalizeStoredPath(rootPath)).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.AssistantProject{}, false, nil
		}
		return domain.AssistantProject{}, false, err
	}
	return projectFromRow(row), true, nil
}

type projectQueryCursor struct {
	TimeUpdated string `json:"timeUpdated"`
	ID          string `json:"id"`
}

func (s *Store) QueryProjects(ctx context.Context, input domain.ProjectQueryInput) (domain.ProjectQueryResult, error) {
	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	q := s.db.WithContext(ctx).Model(&projectRow{}).Where("sidebar_hidden = ?", 0)
	if projectID := strings.TrimSpace(input.ProjectID); projectID != "" {
		q = q.Where("id = ?", projectID)
		limit = 1
	} else {
		if query := strings.TrimSpace(input.Query); query != "" {
			like := "%" + escapeProjectLike(strings.ToLower(query)) + "%"
			q = q.Where("(LOWER(name) LIKE ? ESCAPE '\\' OR LOWER(description) LIKE ? ESCAPE '\\' OR LOWER(root_path) LIKE ? ESCAPE '\\')", like, like, like)
		}
		if cursor := strings.TrimSpace(input.Cursor); cursor != "" {
			decoded, err := decodeProjectQueryCursor(cursor)
			if err != nil {
				return domain.ProjectQueryResult{}, err
			}
			q = q.Where("(time_updated < ? OR (time_updated = ? AND id < ?))", decoded.TimeUpdated, decoded.TimeUpdated, decoded.ID)
		}
	}
	var rows []projectRow
	if err := q.Order("time_updated DESC").Order("id DESC").Limit(limit + 1).Find(&rows).Error; err != nil {
		return domain.ProjectQueryResult{}, err
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	projects := make([]domain.AssistantProject, 0, len(rows))
	for _, row := range rows {
		projects = append(projects, projectFromRow(row))
	}
	result := domain.ProjectQueryResult{Projects: projects}
	if hasMore && len(rows) > 0 {
		result.NextCursor = encodeProjectQueryCursor(projectQueryCursor{TimeUpdated: rows[len(rows)-1].TimeUpdated, ID: rows[len(rows)-1].ID})
	}
	return result, nil
}

func escapeProjectLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	return strings.ReplaceAll(value, `_`, `\_`)
}

func encodeProjectQueryCursor(cursor projectQueryCursor) string {
	raw, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeProjectQueryCursor(value string) (projectQueryCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return projectQueryCursor{}, errors.New("invalid project cursor")
	}
	var cursor projectQueryCursor
	if err := json.Unmarshal(raw, &cursor); err != nil || strings.TrimSpace(cursor.TimeUpdated) == "" || strings.TrimSpace(cursor.ID) == "" {
		return projectQueryCursor{}, errors.New("invalid project cursor")
	}
	return cursor, nil
}

func (s *Store) BindSessionProject(ctx context.Context, sessionID, projectID string, codingContext domain.CodingContext) (domain.SessionProjectBindingResult, error) {
	lockValue, _ := s.projectBindingMu.LoadOrStore(sessionID, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	var changed bool
	var conflict bool
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row sessionRow
		if err := tx.Where("id = ?", strings.TrimSpace(sessionID)).First(&row).Error; err != nil {
			return err
		}
		if row.ProjectID != "" {
			conflict = row.ProjectID != projectID
			return nil
		}
		result := tx.Model(&sessionRow{}).
			Where("id = ? AND COALESCE(project_id, '') = ''", sessionID).
			Updates(map[string]any{"project_id": projectID, "time_updated": codingContext.TimeUpdated})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			conflict = true
			return nil
		}
		if _, err := upsertCodingContextWithDB(tx, codingContext); err != nil {
			return err
		}
		changed = true
		return nil
	})
	if err != nil {
		return domain.SessionProjectBindingResult{}, err
	}
	session, err := s.GetRuntimeSession(ctx, sessionID)
	if err != nil {
		return domain.SessionProjectBindingResult{}, err
	}
	if !changed && session.ProjectPath != "" {
		project, found, projectErr := s.GetProjectByID(ctx, projectID)
		if projectErr == nil && found && normalizeStoredPath(session.ProjectPath) == normalizeStoredPath(project.RootPath) {
			conflict = false
		}
	}
	return domain.SessionProjectBindingResult{Session: session, Changed: changed, Conflict: conflict}, nil
}

func (s *Store) UpdateProjectDescription(ctx context.Context, rootPath string, description string) (domain.AssistantProject, error) {
	rootPath = strings.TrimSpace(rootPath)
	description = strings.TrimSpace(description)
	if rootPath == "" {
		return domain.AssistantProject{}, errors.New("project path is required")
	}
	if err := s.db.WithContext(ctx).Model(&projectRow{}).Where("root_path = ?", rootPath).Updates(map[string]any{
		"description":  description,
		"time_updated": domain.NowString(time.Now()),
	}).Error; err != nil {
		return domain.AssistantProject{}, err
	}
	var saved projectRow
	if err := s.db.WithContext(ctx).Where("root_path = ?", rootPath).First(&saved).Error; err != nil {
		return domain.AssistantProject{}, err
	}
	return projectFromRow(saved), nil
}

func projectFromRow(row projectRow) domain.AssistantProject {
	return domain.AssistantProject{ID: row.ID, Name: row.Name, Description: row.Description, RootPath: row.RootPath, GitBranch: row.GitBranch, GitDirty: row.GitDirty == 1, GitAvailable: row.GitAvailable == 1, SidebarHidden: row.SidebarHidden == 1, TimeOpened: row.TimeOpened, TimeUpdated: row.TimeUpdated}
}
