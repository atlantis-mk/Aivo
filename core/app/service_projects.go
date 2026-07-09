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
	return s.store.UpsertProject(ctx, abs)
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
