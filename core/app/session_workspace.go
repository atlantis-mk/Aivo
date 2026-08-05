package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	managedWorkspaceRootEnv = "AIVO_WORKSPACES_DIR"
	managedWorkspaceRootDir = "Aivo Workspaces"
)

func managedWorkspaceRoot() (string, error) {
	if configured := strings.TrimSpace(os.Getenv(managedWorkspaceRootEnv)); configured != "" {
		return filepath.Abs(configured)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Documents", managedWorkspaceRootDir), nil
}

func isManagedWorkspace(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	root, err := managedWorkspaceRoot()
	if err != nil {
		return false
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(root, absPath)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}

func ensureInitialWorkspaceDirectory(path string) (string, error) {
	clean := strings.TrimSpace(path)
	if clean == "" {
		return "", errors.New("initial workspace is not configured; complete setup first")
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return "", fmt.Errorf("resolve initial workspace: %w", err)
	}
	info, err := os.Stat(abs)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(abs, 0o700); err != nil {
			return "", fmt.Errorf("create initial workspace: %w", err)
		}
		info, err = os.Stat(abs)
	}
	if err != nil {
		return "", fmt.Errorf("inspect initial workspace: %w", err)
	}
	if !info.IsDir() {
		return "", errors.New("initial workspace path is not a directory")
	}
	return abs, nil
}

func (s *Service) ensureUnscopedWorkspace(ctx context.Context, _ string) (string, error) {
	cfg, err := s.AppConfig(ctx)
	if err != nil {
		return "", err
	}
	return ensureInitialWorkspaceDirectory(cfg.InitialWorkspacePath)
}
