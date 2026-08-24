package app

import (
	"context"

	"aivo/core/domain"
)

func (s *Service) AppConfig(ctx context.Context) (domain.AppConfig, error) {
	return s.appConfig(ctx)
}

func (s *Service) SelectProjectDirectory(path string) (string, error) {
	return s.selectProjectDirectory(path)
}
