package app

import (
	"context"
	"path/filepath"
	"strings"

	"aivo/core/domain"
)

func (s *Service) loadAgentShellWorkingDirectory(sessionID string, workspaceRoot string) string {
	if s == nil || s.store == nil {
		return ""
	}
	cc, err := s.store.GetCodingContext(context.Background(), strings.TrimSpace(sessionID))
	if err != nil {
		return ""
	}
	cwd := normalizeWorkspaceInternalCWD(workspaceRoot, cc.CWD)
	if cwd != "" {
		return cwd
	}
	if strings.TrimSpace(cc.CWD) != "" {
		s.saveAgentShellWorkingDirectory(sessionID, workspaceRoot, workspaceRoot)
	}
	return ""
}

func (s *Service) saveAgentShellWorkingDirectory(sessionID string, workspaceRoot string, cwd string) {
	if s == nil || s.store == nil || strings.TrimSpace(sessionID) == "" {
		return
	}
	normalized := normalizeWorkspaceInternalCWD(workspaceRoot, cwd)
	if normalized == "" {
		normalized = normalizeWorkspaceInternalCWD(workspaceRoot, workspaceRoot)
	}
	if normalized == "" {
		return
	}
	ctx := context.Background()
	cc, err := s.store.GetCodingContext(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		cc = domain.CodingContext{
			SessionID:   strings.TrimSpace(sessionID),
			ProjectPath: normalizeWorkspaceInternalCWD(workspaceRoot, workspaceRoot),
			Permissions: []string{"local-filesystem"},
		}
	}
	if strings.TrimSpace(cc.ProjectPath) == "" {
		cc.ProjectPath = normalizeWorkspaceInternalCWD(workspaceRoot, workspaceRoot)
	}
	cc.CWD = normalized
	_, _ = s.store.UpsertCodingContext(ctx, cc)
}

func normalizeWorkspaceInternalCWD(workspaceRoot string, cwd string) string {
	if strings.TrimSpace(workspaceRoot) == "" {
		return ""
	}
	_, normalized, err := normalizeSandboxCWD(workspaceRoot, cwd, false)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(normalized)
}
