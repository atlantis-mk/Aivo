package app

import (
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

func (s *Service) createManagedWorkspace(sessionID string) (string, error) {
	root, err := managedWorkspaceRoot()
	if err != nil {
		return "", err
	}
	dateDir := s.now().Format("2006-01-02")
	baseName := workspaceSlug(sessionID)
	parent := filepath.Join(root, dateDir)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return "", err
	}
	for i := 0; i < 1000; i++ {
		name := baseName
		if i > 0 {
			name = fmt.Sprintf("%s-%d", baseName, i+1)
		}
		path := filepath.Join(parent, name)
		err := os.Mkdir(path, 0o700)
		if err == nil {
			return path, nil
		}
		if errors.Is(err, os.ErrExist) {
			continue
		}
		return "", err
	}
	return "", fmt.Errorf("could not allocate managed workspace under %s", parent)
}

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

func workspaceSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	slug := strings.Trim(builder.String(), "-")
	if slug == "" {
		return "session"
	}
	if len(slug) > 80 {
		slug = strings.Trim(slug[:80], "-")
	}
	if slug == "" {
		return "session"
	}
	return slug
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

func ensureManagedWorkspace(path string) (string, bool, error) {
	if strings.TrimSpace(path) == "" {
		return path, false, nil
	}
	if !isManagedWorkspace(path) {
		return path, false, nil
	}
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		return path, false, nil
	}
	if err == nil && !info.IsDir() {
		return "", false, fmt.Errorf("temporary workspace path is not a directory: %s", path)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", false, err
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return "", false, err
	}
	return path, true, nil
}
