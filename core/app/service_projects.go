package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"aivo/core/domain"
)

func (s *Service) selectProjectDirectory(path string) (string, error) {
	clean := strings.TrimSpace(path)
	if clean == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		clean = wd
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("selected path is not a directory")
	}
	return abs, nil
}

func (s *Service) UpsertProject(ctx context.Context, path string) (domain.AssistantProject, error) {
	abs, err := s.SelectProjectDirectory(path)
	if err != nil {
		return domain.AssistantProject{}, err
	}
	project, err := s.store.UpsertProject(ctx, abs)
	if err != nil || strings.TrimSpace(project.Description) != "" {
		return project, err
	}
	// Always provide a useful searchable description immediately. A configured
	// auxiliary model refines it asynchronously without delaying project selection.
	description := fallbackProjectDescription(abs)
	project, _ = s.store.UpdateProjectDescription(ctx, abs, description)
	go s.refineProjectDescription(abs)
	return project, nil
}

func (s *Service) SwitchSessionProject(ctx context.Context, sessionID string, projectPath string) (domain.Session, error) {
	abs, err := s.SelectProjectDirectory(projectPath)
	if err != nil {
		return domain.Session{}, err
	}
	if _, err := s.UpsertProject(ctx, abs); err != nil {
		return domain.Session{}, err
	}
	session, err := s.store.SetRuntimeSessionProject(ctx, sessionID, abs)
	if err != nil {
		return domain.Session{}, err
	}
	if session.Type == domain.SessionTypeCoding {
		if _, err := s.CreateOrUpdateCodingContext(ctx, sessionID, abs); err != nil {
			return domain.Session{}, err
		}
	}
	if s.onSessionUpdated != nil {
		s.onSessionUpdated(sessionID, &session)
	}
	return session, nil
}

func (s *Service) SetProjectSidebarHidden(ctx context.Context, path string, hidden bool) (domain.AssistantProject, error) {
	abs, err := s.SelectProjectDirectory(path)
	if err != nil {
		return domain.AssistantProject{}, err
	}
	return s.store.SetProjectSidebarHidden(ctx, abs, hidden)
}

func (s *Service) ListProjects(ctx context.Context, limit int) ([]domain.AssistantProject, error) {
	requestedLimit := limit
	if requestedLimit <= 0 {
		requestedLimit = 20
	}
	fetchLimit := requestedLimit * 3
	if fetchLimit < 50 {
		fetchLimit = 50
	}
	if fetchLimit > 200 {
		fetchLimit = 200
	}
	projects, err := s.store.ListProjects(ctx, fetchLimit)
	if err != nil {
		return nil, err
	}
	filtered := make([]domain.AssistantProject, 0, len(projects))
	for _, project := range projects {
		if strings.TrimSpace(project.RootPath) == "" || isManagedWorkspace(project.RootPath) {
			continue
		}
		filtered = append(filtered, project)
		if len(filtered) >= requestedLimit {
			break
		}
	}
	return filtered, nil
}
