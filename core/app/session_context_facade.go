package app

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"aivo/core/domain"
)

func (s *Service) CreateSummary(ctx context.Context, input domain.CreateSummaryRequest) (domain.SessionSummary, error) {
	if strings.TrimSpace(input.SessionID) == "" {
		return domain.SessionSummary{}, errors.New("sessionId is required")
	}
	summaryText := strings.TrimSpace(input.Summary)
	if summaryText == "" {
		summaryText = s.generatedSummary(ctx, input.SessionID)
		if summaryText == "" {
			summaryText = s.fallbackSummary(ctx, input.SessionID)
		}
	}
	summary := domain.SessionSummary{ID: uuid.NewString(), SessionID: input.SessionID, FromEventID: input.FromEventID, ToEventID: input.ToEventID, Summary: summaryText, Facts: input.Facts, Decisions: input.Decisions, OpenTasks: input.OpenTasks, ChangedFiles: input.ChangedFiles, NextSuggestedAction: input.NextSuggestedAction, TimeCreated: domain.NowString(s.now())}
	if err := s.store.CreateSummary(ctx, summary); err != nil {
		return domain.SessionSummary{}, err
	}
	_, _ = s.AppendEvent(ctx, domain.AppendEventRequest{SessionID: input.SessionID, Type: domain.EventTypeSummary, Role: domain.EventRoleSystem, Visibility: domain.EventVisibilityNormal, Content: summary.Summary})
	return summary, nil
}

func (s *Service) LatestSummary(ctx context.Context, sessionID string) (*domain.SessionSummary, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("sessionId is required")
	}
	return s.store.LatestSummary(ctx, sessionID)
}

func (s *Service) ListCheckpoints(ctx context.Context, sessionID string, limit int) ([]domain.SessionCheckpoint, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("sessionId is required")
	}
	return s.store.ListCheckpoints(ctx, sessionID, limit)
}

func (s *Service) LatestCheckpoint(ctx context.Context, sessionID string) (*domain.SessionCheckpoint, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("sessionId is required")
	}
	return s.store.LatestCheckpoint(ctx, sessionID)
}

func (s *Service) GetCodingContext(ctx context.Context, sessionID string) (domain.CodingContext, error) {
	if strings.TrimSpace(sessionID) == "" {
		return domain.CodingContext{}, errors.New("sessionId is required")
	}
	return s.store.GetCodingContext(ctx, sessionID)
}

func (s *Service) CompactSession(ctx context.Context, sessionID string) (domain.SessionSummary, error) {
	return s.CreateSummary(ctx, domain.CreateSummaryRequest{SessionID: sessionID})
}

func (s *Service) CompactSessionContext(ctx context.Context, input domain.CompactSessionContextInput) (domain.CompactSessionContextResult, error) {
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return domain.CompactSessionContextResult{}, errors.New("sessionId is required")
	}
	compactionKind := "manual"
	if input.Automatic {
		compactionKind = "automatic"
	}
	compacting, err := s.store.UpsertSessionExecutionState(ctx, domain.SessionExecutionState{SessionID: sessionID, Status: domain.ExecutionStatusCompacting, Reason: compactionKind + " compaction requested"})
	if err != nil {
		return domain.CompactSessionContextResult{}, err
	}
	eventsBefore, _ := s.store.ListSessionEvents(ctx, sessionID, false, 500)
	fromEventID := ""
	if latest, _ := s.store.LatestSummary(ctx, sessionID); latest != nil {
		fromEventID = latest.ToEventID
	}
	toEventID := ""
	if len(eventsBefore) > 0 {
		toEventID = eventsBefore[len(eventsBefore)-1].ID
	}
	summary, err := s.CreateSummary(ctx, domain.CreateSummaryRequest{SessionID: sessionID, FromEventID: fromEventID, ToEventID: toEventID})
	if err != nil {
		_, _ = s.store.UpsertSessionExecutionState(ctx, domain.SessionExecutionState{SessionID: sessionID, Status: domain.ExecutionStatusFailed, Reason: err.Error()})
		return domain.CompactSessionContextResult{}, err
	}
	budget := input.CharacterBudget
	if budget <= 0 {
		budget = 6000
	}
	contextResult, err := s.BuildSessionContext(ctx, domain.BuildSessionContextRequest{SessionID: sessionID, CharacterBudget: budget})
	if err != nil {
		_, _ = s.store.UpsertSessionExecutionState(ctx, domain.SessionExecutionState{SessionID: sessionID, Status: domain.ExecutionStatusFailed, Reason: err.Error()})
		return domain.CompactSessionContextResult{}, err
	}
	event, _ := s.AppendEvent(ctx, domain.AppendEventRequest{
		SessionID: sessionID, Type: domain.EventTypeSystemNote, Role: domain.EventRoleSystem,
		Visibility: domain.EventVisibilityNormal, Content: "Session context compacted",
		Payload: map[string]any{"kind": "context_compacted", "mode": compactionKind, "summaryId": summary.ID, "sectionCount": len(contextResult.Sections)},
	})
	state, err := s.store.UpsertSessionExecutionState(ctx, domain.SessionExecutionState{SessionID: sessionID, Status: domain.ExecutionStatusIdle, Reason: "compaction complete", LastEventID: event.ID})
	if err != nil {
		return domain.CompactSessionContextResult{}, err
	}
	if s.onSessionUpdated != nil {
		s.onSessionUpdated(sessionID, nil)
	}
	if compacting.Status == "" {
		compacting = state
	}
	return domain.CompactSessionContextResult{State: state, Summary: summary, Context: contextResult, CompactedEventID: event.ID}, nil
}

func (s *Service) CreateCheckpoint(ctx context.Context, input domain.CreateCheckpointRequest) (domain.SessionCheckpoint, error) {
	if strings.TrimSpace(input.SessionID) == "" {
		return domain.SessionCheckpoint{}, errors.New("sessionId is required")
	}
	cc, _ := s.store.GetCodingContext(ctx, input.SessionID)
	knownIssues := append([]string{}, input.KnownIssues...)
	diffSummary := strings.TrimSpace(input.DiffSummary)
	if diffSummary == "" && cc.ProjectPath != "" {
		diffSummary = gitOutput(ctx, cc.ProjectPath, "diff", "--stat")
		if diffSummary == "" {
			knownIssues = append(knownIssues, "Git diff unavailable for this checkpoint.")
		}
	}
	checkpoint := domain.SessionCheckpoint{ID: uuid.NewString(), SessionID: input.SessionID, Branch: cc.GitBranch, CommitSHA: cc.CommitSHA, ChangedFiles: cc.ChangedFiles, DiffSummary: diffSummary, ConversationSummary: strings.TrimSpace(input.ConversationSummary), OpenTodos: input.OpenTodos, KnownIssues: knownIssues, NextSuggestedAction: strings.TrimSpace(input.NextSuggestedAction), TimeCreated: domain.NowString(s.now())}
	if err := s.store.CreateCheckpoint(ctx, checkpoint); err != nil {
		return domain.SessionCheckpoint{}, err
	}
	_, _ = s.AppendEvent(ctx, domain.AppendEventRequest{SessionID: input.SessionID, Type: domain.EventTypeCheckpoint, Role: domain.EventRoleSystem, Visibility: domain.EventVisibilityNormal, Content: checkpoint.ConversationSummary, Payload: map[string]any{"checkpointId": checkpoint.ID}})
	return checkpoint, nil
}

func (s *Service) ForkSession(ctx context.Context, input domain.ForkSessionRequest) (domain.Session, error) {
	source, err := s.store.GetRuntimeSession(ctx, input.SessionID)
	if err != nil {
		return domain.Session{}, err
	}
	return s.store.ForkRuntimeSession(ctx, source, input)
}

func (s *Service) CreateOrUpdateCodingContext(ctx context.Context, sessionID string, projectPath string) (domain.CodingContext, error) {
	abs, err := filepath.Abs(strings.TrimSpace(projectPath))
	if err != nil {
		abs = strings.TrimSpace(projectPath)
	}
	current, _ := s.store.GetCodingContext(ctx, sessionID)
	cwd := abs
	if restored := workspaceInternalCWD(abs, current.CWD); restored != "" {
		cwd = restored
	}
	changed := lines(gitOutput(ctx, abs, "status", "--short"))
	cc := domain.CodingContext{
		SessionID: sessionID, ProjectPath: abs, GitBranch: strings.TrimSpace(gitOutput(ctx, abs, "branch", "--show-current")),
		CommitSHA: strings.TrimSpace(gitOutput(ctx, abs, "rev-parse", "HEAD")), RepoURL: strings.TrimSpace(gitOutput(ctx, abs, "config", "--get", "remote.origin.url")),
		ChangedFiles: changed, LanguageStack: detectLanguageStack(abs), PackageManager: detectPackageManager(abs), CWD: cwd, Permissions: []string{"local-filesystem"}, TimeCreated: domain.NowString(s.now()), TimeUpdated: domain.NowString(s.now()),
	}
	return s.store.UpsertCodingContext(ctx, cc)
}

func (s *Service) ResumeRecap(ctx context.Context, input domain.ResumeSessionRequest) (domain.ResumeRecap, error) {
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" && strings.TrimSpace(input.ProjectPath) != "" {
		latest, err := s.store.LatestSessionByProject(ctx, input.ProjectPath)
		if err != nil || latest == nil {
			return domain.ResumeRecap{}, err
		}
		sessionID = latest.ID
	}
	session, err := s.store.GetRuntimeSession(ctx, sessionID)
	if err != nil {
		return domain.ResumeRecap{}, err
	}
	events, err := s.store.ListSessionEvents(ctx, session.ID, false, 20)
	if err != nil {
		return domain.ResumeRecap{}, err
	}
	latestSummary, _ := s.store.LatestSummary(ctx, session.ID)
	latestCheckpoint, _ := s.store.LatestCheckpoint(ctx, session.ID)
	cc, _ := s.store.GetCodingContext(ctx, session.ID)
	recap := domain.ResumeRecap{SessionID: session.ID, Title: session.Title, Goal: session.Goal, LatestSummary: latestSummary, ProjectPath: cc.ProjectPath, Branch: cc.GitBranch, ChangedFiles: cc.ChangedFiles, LastCommand: cc.LastCommand, UpdatedTime: session.TimeUpdated, LatestCheckpoint: latestCheckpoint, RecentEvents: events}
	if latestCheckpoint != nil {
		recap.OpenTodos = latestCheckpoint.OpenTodos
		recap.NextSuggestedAction = latestCheckpoint.NextSuggestedAction
	}
	if latestSummary != nil && recap.NextSuggestedAction == "" {
		recap.OpenTodos = latestSummary.OpenTasks
		recap.NextSuggestedAction = latestSummary.NextSuggestedAction
	}
	return recap, nil
}

func (s *Service) BuildSessionContext(ctx context.Context, input domain.BuildSessionContextRequest) (domain.BuildSessionContextResult, error) {
	return s.buildSessionContextSections(ctx, input)
}
