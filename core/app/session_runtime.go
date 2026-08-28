package app

import (
	"context"
	"errors"
	"strings"

	"aivo/core/domain"
)

func (s *Service) CreateRuntimeSession(ctx context.Context, input domain.CreateSessionRequest) (domain.Session, error) {
	if input.Type == domain.SessionTypeCoding && strings.TrimSpace(input.AgentMode) == "" {
		runtime := loadEffectiveRuntimeConfig(input.ProjectPath).Config
		if strings.TrimSpace(runtime.DefaultAgent) != "" {
			catalog, catalogErr := s.agentCatalogForProject(ctx, input.ProjectPath)
			if catalogErr != nil {
				return domain.Session{}, catalogErr
			}
			definition, err := catalog.Get(runtime.DefaultAgent)
			if err != nil || definition.Hidden || definition.Mode == "subagent" {
				return domain.Session{}, errors.New("configured defaultAgent is unavailable as a primary agent")
			}
			input.AgentMode = definition.ID
		}
	}
	isCodingSession := input.Type == domain.SessionTypeCoding
	var cfg domain.AppConfig
	if isCodingSession {
		var err error
		cfg, err = s.AppConfig(ctx)
		if err != nil {
			return domain.Session{}, err
		}
	}
	needsInitialWorkspace := isCodingSession && strings.TrimSpace(input.ProjectPath) == ""
	initialWorkspacePath := ""
	if needsInitialWorkspace {
		var err error
		initialWorkspacePath, err = ensureInitialWorkspaceDirectory(cfg.InitialWorkspacePath)
		if err != nil {
			return domain.Session{}, err
		}
	}
	session, err := s.store.CreateRuntimeSession(ctx, input)
	if err != nil {
		return domain.Session{}, err
	}
	if isCodingSession {
		if _, err := s.SetPermissionMode(ctx, domain.PermissionModeInput{SessionID: session.ID, Mode: cfg.DefaultPermissionMode}); err != nil {
			_, _ = s.store.SetRuntimeSessionStatus(ctx, session.ID, domain.SessionStatusDeleted)
			return domain.Session{}, err
		}
	}
	if needsInitialWorkspace {
		if _, err := s.CreateOrUpdateCodingContext(ctx, session.ID, initialWorkspacePath); err != nil {
			return domain.Session{}, err
		}
		return session, nil
	}
	if session.Type == domain.SessionTypeCoding && strings.TrimSpace(input.ProjectPath) != "" {
		if _, err := s.CreateOrUpdateCodingContext(ctx, session.ID, input.ProjectPath); err != nil {
			return domain.Session{}, err
		}
	}
	return session, nil
}

func (s *Service) ListRuntimeSessions(ctx context.Context, input domain.ListSessionsRequest) ([]domain.Session, error) {
	if input.Type != "" {
		if _, err := domain.NormalizeSessionType(input.Type); err != nil {
			return nil, err
		}
	}
	if input.Status != "" {
		if _, err := domain.NormalizeSessionStatus(input.Status); err != nil {
			return nil, err
		}
	}
	return s.store.ListRuntimeSessions(ctx, input)
}

func (s *Service) GetRuntimeSession(ctx context.Context, id string) (domain.Session, error) {
	if strings.TrimSpace(id) == "" {
		return domain.Session{}, errors.New("sessionId is required")
	}
	return s.store.GetRuntimeSession(ctx, id)
}

func (s *Service) UpdateRuntimeSession(ctx context.Context, input domain.UpdateSessionRequest) (domain.Session, error) {
	if strings.TrimSpace(input.SessionID) == "" {
		return domain.Session{}, errors.New("sessionId is required")
	}
	return s.store.UpdateRuntimeSession(ctx, input)
}

func (s *Service) ArchiveRuntimeSession(ctx context.Context, id string) (domain.Session, error) {
	if strings.TrimSpace(id) == "" {
		return domain.Session{}, errors.New("sessionId is required")
	}
	return s.store.SetRuntimeSessionStatus(ctx, id, domain.SessionStatusArchived)
}

func (s *Service) DeleteRuntimeSession(ctx context.Context, id string) (domain.Session, error) {
	if strings.TrimSpace(id) == "" {
		return domain.Session{}, errors.New("sessionId is required")
	}
	deleted, err := s.store.SetRuntimeSessionStatus(ctx, id, domain.SessionStatusDeleted)
	if err == nil {
		defaultAgentPTYRegistry.CleanupSession(strings.TrimSpace(id))
		cleanupRetainedOutputSession(strings.TrimSpace(id))
	}
	return deleted, err
}

func (s *Service) ContinueLastSession(ctx context.Context) (*domain.ResumeRecap, error) {
	sessions, err := s.store.ListRuntimeSessions(ctx, domain.ListSessionsRequest{Status: domain.SessionStatusActive, Limit: 1})
	if err != nil || len(sessions) == 0 {
		return nil, err
	}
	recap, err := s.ResumeRecap(ctx, domain.ResumeSessionRequest{SessionID: sessions[0].ID})
	return &recap, err
}

func (s *Service) ContinueProjectSession(ctx context.Context, projectPath string) (*domain.ResumeRecap, error) {
	latest, err := s.store.LatestSessionByProject(ctx, projectPath)
	if err != nil || latest == nil {
		return nil, err
	}
	recap, err := s.ResumeRecap(ctx, domain.ResumeSessionRequest{SessionID: latest.ID})
	return &recap, err
}
