package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"aivo/core/domain"
)

const (
	managedWorkspaceRootEnv = "AIVO_WORKSPACES_DIR"
	managedWorkspaceRootDir = "Aivo Workspaces"
)

func (s *Service) CreateRuntimeSession(ctx context.Context, input domain.CreateSessionRequest) (domain.Session, error) {
	needsManagedWorkspace := input.Type == domain.SessionTypeCoding && strings.TrimSpace(input.ProjectPath) == ""
	session, err := s.store.CreateRuntimeSession(ctx, input)
	if err != nil {
		return domain.Session{}, err
	}
	if needsManagedWorkspace {
		projectPath, err := s.createManagedWorkspace(session.ID)
		if err != nil {
			return domain.Session{}, err
		}
		input.ProjectPath = projectPath
		session.ProjectPath = projectPath
	}
	if session.Type == domain.SessionTypeCoding && strings.TrimSpace(input.ProjectPath) != "" {
		if _, err := s.CreateOrUpdateCodingContext(ctx, session.ID, input.ProjectPath); err != nil {
			return domain.Session{}, err
		}
	}
	return session, nil
}

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
	return s.store.SetRuntimeSessionStatus(ctx, id, domain.SessionStatusDeleted)
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

func (s *Service) AppendEvent(ctx context.Context, input domain.AppendEventRequest) (domain.SessionEvent, error) {
	if strings.TrimSpace(input.SessionID) == "" {
		return domain.SessionEvent{}, errors.New("sessionId is required")
	}
	if err := domain.ValidateEventType(input.Type); err != nil {
		return domain.SessionEvent{}, err
	}
	role, err := domain.NormalizeEventRole(input.Role)
	if err != nil {
		return domain.SessionEvent{}, err
	}
	visibility, err := domain.NormalizeEventVisibility(input.Visibility)
	if err != nil {
		return domain.SessionEvent{}, err
	}
	event := domain.SessionEvent{
		ID: uuid.NewString(), SessionID: strings.TrimSpace(input.SessionID), TurnID: strings.TrimSpace(input.TurnID),
		Type: input.Type, Role: role, Visibility: visibility, Content: strings.TrimSpace(input.Content),
		Payload: input.Payload, TokenCount: input.TokenCount, TimeCreated: domain.NowString(s.now()),
	}
	if err := s.store.AppendSessionEvent(ctx, event); err != nil {
		return domain.SessionEvent{}, err
	}
	if event.Visibility == domain.EventVisibilityNormal && event.Type == domain.EventTypeUserMessage {
		_ = s.updateUntitledSession(ctx, event.SessionID, event.Content)
	}
	return event, nil
}

func (s *Service) ListEvents(ctx context.Context, sessionID string, includeNonNormal bool, limit int) ([]domain.SessionEvent, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("sessionId is required")
	}
	return s.store.ListSessionEvents(ctx, sessionID, includeNonNormal, limit)
}

func (s *Service) UpdateSessionEvent(ctx context.Context, input domain.UpdateSessionEventRequest) (domain.SessionEvent, error) {
	if strings.TrimSpace(input.EventID) == "" {
		return domain.SessionEvent{}, errors.New("eventId is required")
	}
	if strings.TrimSpace(input.Content) == "" {
		return domain.SessionEvent{}, errors.New("content is required")
	}
	event, err := s.store.GetSessionEvent(ctx, input.EventID)
	if err != nil {
		return domain.SessionEvent{}, err
	}
	if event.Type != domain.EventTypeUserMessage && event.Type != domain.EventTypeAssistantMessage {
		return domain.SessionEvent{}, errors.New("only user and assistant messages can be edited")
	}
	updated, err := s.store.UpdateSessionEvent(ctx, input)
	if err == nil && s.onSessionUpdated != nil {
		s.onSessionUpdated(updated.SessionID, nil)
	}
	return updated, err
}

func (s *Service) DeleteSessionEvent(ctx context.Context, input domain.DeleteSessionEventRequest) (domain.SessionEvent, error) {
	if strings.TrimSpace(input.EventID) == "" {
		return domain.SessionEvent{}, errors.New("eventId is required")
	}
	event, err := s.store.SetSessionEventVisibility(ctx, input.EventID, domain.EventVisibilityHidden)
	if err == nil && s.onSessionUpdated != nil {
		s.onSessionUpdated(event.SessionID, nil)
	}
	return event, err
}

func (s *Service) StartTurn(ctx context.Context, input domain.StartTurnRequest) (domain.Turn, error) {
	if strings.TrimSpace(input.SessionID) == "" {
		return domain.Turn{}, errors.New("sessionId is required")
	}
	agentMode := strings.TrimSpace(input.AgentMode)
	if agentMode == "" {
		if session, err := s.store.GetRuntimeSession(ctx, input.SessionID); err == nil {
			agentMode = session.AgentMode
		}
	}
	mode, err := domain.NormalizeAgentMode(agentMode)
	if err != nil {
		return domain.Turn{}, err
	}
	now := domain.NowString(s.now())
	turn := domain.Turn{ID: uuid.NewString(), SessionID: input.SessionID, UserEventID: input.UserEventID, AgentMode: mode, Status: domain.TurnStatusRunning, TimeCreated: now, TimeUpdated: now}
	if err := s.store.StartTurn(ctx, turn); err != nil {
		return domain.Turn{}, err
	}
	if s.onTurnUpdated != nil {
		s.onTurnUpdated(turn.SessionID, turn)
	}
	return turn, nil
}

func (s *Service) CompleteTurn(ctx context.Context, input domain.CompleteTurnRequest) (domain.Turn, error) {
	turn, err := s.store.UpdateTurnStatus(ctx, input.TurnID, domain.TurnStatusCompleted, "")
	if err == nil && s.onTurnUpdated != nil {
		s.onTurnUpdated(turn.SessionID, turn)
	}
	return turn, err
}

func (s *Service) FailTurn(ctx context.Context, input domain.FailTurnRequest) (domain.Turn, error) {
	turn, err := s.store.UpdateTurnStatus(ctx, input.TurnID, domain.TurnStatusFailed, strings.TrimSpace(input.Error))
	if err == nil {
		_, _ = s.AppendEvent(ctx, domain.AppendEventRequest{SessionID: turn.SessionID, TurnID: turn.ID, Type: domain.EventTypeError, Role: domain.EventRoleSystem, Visibility: domain.EventVisibilityNormal, Content: strings.TrimSpace(input.Error)})
		if s.onTurnUpdated != nil {
			s.onTurnUpdated(turn.SessionID, turn)
		}
	}
	return turn, err
}

func (s *Service) CancelTurn(ctx context.Context, input domain.CancelTurnRequest) (domain.Turn, error) {
	s.cancelActiveTurn(input.TurnID)
	turn, err := s.store.UpdateTurnStatus(ctx, input.TurnID, domain.TurnStatusCancelled, strings.TrimSpace(input.Reason))
	if err == nil {
		content := strings.TrimSpace(input.Reason)
		if content == "" {
			content = "Turn cancelled"
		}
		_, _ = s.AppendEvent(ctx, domain.AppendEventRequest{SessionID: turn.SessionID, TurnID: turn.ID, Type: domain.EventTypeSystemNote, Role: domain.EventRoleSystem, Visibility: domain.EventVisibilityNormal, Content: content})
		if s.onTurnUpdated != nil {
			s.onTurnUpdated(turn.SessionID, turn)
		}
	}
	return turn, err
}

func (s *Service) RetrySessionTurnStreaming(ctx context.Context, input domain.RetrySessionTurnRequest) (domain.PreparedSessionTurn, error) {
	turnID := strings.TrimSpace(input.TurnID)
	if turnID == "" {
		return domain.PreparedSessionTurn{}, errors.New("turnId is required")
	}
	turn, err := s.store.GetTurn(ctx, turnID)
	if err != nil {
		return domain.PreparedSessionTurn{}, err
	}
	if strings.TrimSpace(input.SessionID) != "" && strings.TrimSpace(input.SessionID) != turn.SessionID {
		return domain.PreparedSessionTurn{}, errors.New("turn does not belong to session")
	}
	if strings.TrimSpace(turn.UserEventID) == "" {
		return domain.PreparedSessionTurn{}, errors.New("turn has no user message to retry")
	}
	userEvent, err := s.store.GetSessionEvent(ctx, turn.UserEventID)
	if err != nil {
		return domain.PreparedSessionTurn{}, err
	}
	if userEvent.Type != domain.EventTypeUserMessage || strings.TrimSpace(userEvent.Content) == "" {
		return domain.PreparedSessionTurn{}, errors.New("turn user message is not retryable")
	}
	s.cancelActiveTurn(turn.ID)
	_, _ = s.store.SetSessionEventVisibility(ctx, userEvent.ID, domain.EventVisibilityHidden)
	_ = s.store.HideSessionTurnEvents(ctx, turn.ID)
	if turn.Status == domain.TurnStatusRunning {
		_, _ = s.store.UpdateTurnStatus(ctx, turn.ID, domain.TurnStatusCancelled, "Retried")
	}
	if s.onSessionUpdated != nil {
		s.onSessionUpdated(turn.SessionID, nil)
	}
	return s.SubmitSessionMessageStreaming(ctx, domain.SubmitSessionMessageRequest{
		SessionID:       turn.SessionID,
		Text:            userEvent.Content,
		Model:           input.Model,
		AgentMode:       firstNonEmpty(input.AgentMode, turn.AgentMode),
		Toolsets:        input.Toolsets,
		PermissionScope: input.PermissionScope,
		ReasoningEffort: input.ReasoningEffort,
		ServiceTier:     input.ServiceTier,
	})
}

func (s *Service) registerActiveTurn(id string, cancel context.CancelFunc) {
	id = strings.TrimSpace(id)
	if s == nil || id == "" || cancel == nil {
		return
	}
	s.activeTurnMu.Lock()
	defer s.activeTurnMu.Unlock()
	if s.activeTurnCancel == nil {
		s.activeTurnCancel = map[string]context.CancelFunc{}
	}
	s.activeTurnCancel[id] = cancel
}

func (s *Service) unregisterActiveTurn(id string) {
	id = strings.TrimSpace(id)
	if s == nil || id == "" {
		return
	}
	s.activeTurnMu.Lock()
	defer s.activeTurnMu.Unlock()
	delete(s.activeTurnCancel, id)
}

func (s *Service) cancelActiveTurn(id string) {
	id = strings.TrimSpace(id)
	if s == nil || id == "" {
		return
	}
	s.activeTurnMu.Lock()
	cancel := s.activeTurnCancel[id]
	s.activeTurnMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *Service) SaveToolCall(ctx context.Context, input domain.CreateToolCallRequest) (domain.ToolCall, error) {
	status, err := domain.NormalizeToolCallStatus(input.Status)
	if err != nil {
		return domain.ToolCall{}, err
	}
	if strings.TrimSpace(input.SessionID) == "" || strings.TrimSpace(input.Name) == "" {
		return domain.ToolCall{}, errors.New("sessionId and name are required")
	}
	now := domain.NowString(s.now())
	visibility := domain.EventVisibilityInternal
	if status == domain.ToolCallStatusFailed {
		visibility = domain.EventVisibilityNormal
	}
	event, err := s.AppendEvent(ctx, domain.AppendEventRequest{SessionID: input.SessionID, TurnID: input.TurnID, Type: domain.EventTypeToolCall, Role: domain.EventRoleTool, Visibility: visibility, Content: input.ResultSummary, Payload: map[string]any{"name": input.Name}})
	if err != nil {
		return domain.ToolCall{}, err
	}
	callID := uuid.NewString()
	if strings.TrimSpace(input.ID) != "" {
		callID = strings.TrimSpace(input.ID)
	}
	call := domain.ToolCall{ID: callID, SessionID: input.SessionID, TurnID: input.TurnID, EventID: event.ID, Name: input.Name, Arguments: input.Arguments, Status: status, ResultSummary: bounded(input.ResultSummary, 2000), Result: input.Result, Error: input.Error, TimeCreated: now, TimeUpdated: now}
	if err := s.store.SaveToolCall(ctx, call); err != nil {
		return domain.ToolCall{}, err
	}
	if s.onToolCallUpdated != nil {
		s.onToolCallUpdated(call.SessionID, call.TurnID, call, status == domain.ToolCallStatusRunning)
	}
	return call, nil
}

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

func (s *Service) ListTurns(ctx context.Context, sessionID string, limit int) ([]domain.Turn, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("sessionId is required")
	}
	return s.store.ListTurns(ctx, sessionID, limit)
}

func (s *Service) ListToolCalls(ctx context.Context, sessionID string) ([]domain.ToolCall, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("sessionId is required")
	}
	return s.store.ListToolCalls(ctx, sessionID)
}

func (s *Service) ReplaySessionToolCall(ctx context.Context, input domain.ReplaySessionToolCallRequest) (domain.ToolCall, error) {
	toolCallID := strings.TrimSpace(input.ToolCallID)
	if toolCallID == "" {
		return domain.ToolCall{}, errors.New("toolCallId is required")
	}
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return domain.ToolCall{}, errors.New("sessionId is required")
	}
	toolCalls, err := s.store.ListToolCalls(ctx, sessionID)
	if err != nil {
		return domain.ToolCall{}, err
	}
	var original domain.ToolCall
	for _, call := range toolCalls {
		if call.ID == toolCallID {
			original = call
			break
		}
	}
	if original.ID == "" {
		return domain.ToolCall{}, errors.New("tool call not found")
	}
	if original.SessionID != sessionID {
		return domain.ToolCall{}, errors.New("tool call does not belong to session")
	}
	session, err := s.store.GetRuntimeSession(ctx, sessionID)
	if err != nil {
		return domain.ToolCall{}, err
	}
	cc, _ := s.store.GetCodingContext(ctx, sessionID)
	workspaceRoot := strings.TrimSpace(cc.ProjectPath)
	if workspaceRoot == "" {
		workspaceRoot = strings.TrimSpace(session.ProjectPath)
	}
	if workspaceRoot == "" {
		return domain.ToolCall{}, errors.New("tool replay requires a workspace root")
	}
	var turn domain.Turn
	if strings.TrimSpace(original.TurnID) != "" {
		turn, err = s.store.GetTurn(ctx, original.TurnID)
		if err != nil {
			return domain.ToolCall{}, err
		}
		if turn.SessionID != sessionID {
			return domain.ToolCall{}, errors.New("tool call turn does not belong to session")
		}
	}
	registry, runtime := s.toolsForWorkspace(workspaceRoot)
	if runtime == nil || registry == nil {
		return domain.ToolCall{}, errors.New("tool runtime unavailable for workspace")
	}
	replayID := "replay_" + uuid.NewString()
	rawArgs, err := replayToolCallArguments(original)
	if err != nil {
		return domain.ToolCall{}, err
	}
	replayCall := domain.ChatToolCall{ID: replayID, Name: original.Name, Arguments: rawArgs}
	if err := s.recordToolCallStarted(ctx, sessionID, original.TurnID, replayCall); err != nil {
		return domain.ToolCall{}, err
	}
	mode := firstNonEmpty(turn.AgentMode, session.AgentMode, domain.AgentModeAssistant)
	modeDef, err := s.resolveAgentModeForRequest(ctx, sessionID, mode)
	if err != nil {
		return domain.ToolCall{}, err
	}
	allowedToolsets := allowedToolsetsForRun(modeDef, domain.SubmitSessionMessageRequest{})
	specs := visibleToolSpecsForMode(modeDef.ID, registry.SpecsForToolsets(allowedToolsets))
	assembly := AssembleToolSpecs(registry, specs)
	result := runtime.ExecuteWithContext(ctx, replayCall, domain.ToolExecutionContext{
		WorkspaceRoot:         workspaceRoot,
		SessionID:             sessionID,
		TurnID:                original.TurnID,
		ToolCallID:            replayID,
		AgentMode:             modeDef.ID,
		AllowedToolsets:       allowedToolsets,
		PermissionScope:       firstNonEmpty(input.PermissionScope, defaultPermissionScopeForMode(modeDef.ID)),
		ExpectedRegistrations: assembly.ExpectedRegistrations,
	})
	if err := s.recordToolResultWithMetadata(ctx, sessionID, original.TurnID, replayCall, result, map[string]any{
		"replayOfToolCallId": original.ID,
		"replayOfToolName":   original.Name,
	}); err != nil {
		return domain.ToolCall{}, err
	}
	_ = s.appendToolReplayEvent(ctx, sessionID, original.TurnID, original, replayID, result)
	toolCalls, err = s.store.ListToolCalls(ctx, sessionID)
	if err != nil {
		return domain.ToolCall{}, err
	}
	for _, call := range toolCalls {
		if call.ID == replayID {
			return call, nil
		}
	}
	return domain.ToolCall{}, errors.New("replayed tool call was not saved")
}

func (s *Service) GetSessionExecutionState(ctx context.Context, sessionID string) (domain.SessionExecutionState, error) {
	if strings.TrimSpace(sessionID) == "" {
		return domain.SessionExecutionState{}, errors.New("sessionId is required")
	}
	return s.store.GetSessionExecutionState(ctx, sessionID)
}

func (s *Service) InterruptSessionExecution(ctx context.Context, input domain.InterruptSessionExecutionInput) (domain.SessionExecutionState, error) {
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return domain.SessionExecutionState{}, errors.New("sessionId is required")
	}
	turns, _ := s.store.ListTurns(ctx, sessionID, 20)
	var runningTurnID string
	for _, turn := range turns {
		if turn.Status == domain.TurnStatusRunning {
			runningTurnID = turn.ID
			s.cancelActiveTurn(turn.ID)
			_, _ = s.store.UpdateTurnStatus(ctx, turn.ID, domain.TurnStatusCancelled, firstNonEmpty(input.Reason, "Interrupted by user"))
			break
		}
	}
	_, _ = s.store.MarkRunningToolCallsInterrupted(ctx, sessionID, firstNonEmpty(input.Reason, "Interrupted by user"))
	event, _ := s.AppendEvent(ctx, domain.AppendEventRequest{
		SessionID: sessionID, TurnID: runningTurnID, Type: domain.EventTypeSystemNote, Role: domain.EventRoleSystem,
		Visibility: domain.EventVisibilityNormal, Content: firstNonEmpty(input.Reason, "Session execution interrupted"),
		Payload: map[string]any{"kind": "execution_interrupted"},
	})
	state := domain.SessionExecutionState{
		SessionID: sessionID, TurnID: runningTurnID, Status: domain.ExecutionStatusInterrupted,
		Reason: firstNonEmpty(input.Reason, "interrupted"), LastEventID: event.ID,
		Metadata: map[string]any{"interruptedToolCalls": true},
	}
	if updated, err := s.store.UpsertSessionExecutionState(ctx, state); err == nil {
		if s.onSessionUpdated != nil {
			s.onSessionUpdated(sessionID, nil)
		}
		return updated, nil
	}
	return state, nil
}

func (s *Service) ResumeSessionExecution(ctx context.Context, input domain.ResumeSessionExecutionInput) (domain.SessionExecutionState, error) {
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return domain.SessionExecutionState{}, errors.New("sessionId is required")
	}
	_, _ = s.store.MarkRunningToolCallsInterrupted(ctx, sessionID, "Interrupted before resume; not replayed automatically")
	pending, _ := s.store.ListPendingSessionInputs(ctx, sessionID, domain.PendingInputStatusPending)
	pendingIDs := make([]string, 0, len(pending))
	for _, item := range pending {
		pendingIDs = append(pendingIDs, item.ID)
	}
	status := domain.ExecutionStatusIdle
	reason := "ready"
	if len(pending) > 0 {
		status = domain.ExecutionStatusRunning
		reason = "pending queued input is ready to continue"
	}
	event, _ := s.AppendEvent(ctx, domain.AppendEventRequest{
		SessionID: sessionID, Type: domain.EventTypeSystemNote, Role: domain.EventRoleSystem,
		Visibility: domain.EventVisibilityNormal, Content: "Session execution resumed",
		Payload: map[string]any{"kind": "execution_resumed", "pendingInputCount": len(pending)},
	})
	state, err := s.store.UpsertSessionExecutionState(ctx, domain.SessionExecutionState{
		SessionID: sessionID, Status: status, Reason: reason, LastEventID: event.ID, PendingInputIDs: pendingIDs,
	})
	if err == nil && s.onSessionUpdated != nil {
		s.onSessionUpdated(sessionID, nil)
	}
	return state, err
}

func replayToolCallArguments(call domain.ToolCall) (json.RawMessage, error) {
	if call.Arguments == nil {
		return json.RawMessage(`{}`), nil
	}
	if freeform, _ := call.Arguments["freeform"].(bool); freeform {
		if patchText, _ := call.Arguments["patchText"].(string); patchText != "" {
			return json.RawMessage(patchText), nil
		}
	}
	raw, err := json.Marshal(call.Arguments)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func (s *Service) appendToolReplayEvent(ctx context.Context, sessionID string, turnID string, original domain.ToolCall, replayID string, result domain.ToolResult) error {
	status := "failed"
	if result.OK {
		status = "succeeded"
	}
	if result.PermissionRequested {
		status = "waiting for permission"
	}
	content := fmt.Sprintf("Tool call replay %s: %s from %s", status, firstNonEmpty(result.Name, original.Name), original.ID)
	_, err := s.AppendEvent(ctx, domain.AppendEventRequest{
		SessionID:  sessionID,
		TurnID:     turnID,
		Type:       domain.EventTypeSystemNote,
		Role:       domain.EventRoleSystem,
		Visibility: domain.EventVisibilityNormal,
		Content:    content,
		Payload: map[string]any{
			"kind":               "tool_call_replay",
			"status":             status,
			"originalToolCallId": original.ID,
			"replayToolCallId":   replayID,
			"toolName":           firstNonEmpty(result.Name, original.Name),
		},
	})
	return err
}

func (s *Service) CompactSession(ctx context.Context, sessionID string) (domain.SessionSummary, error) {
	return s.CreateSummary(ctx, domain.CreateSummaryRequest{SessionID: sessionID})
}

func (s *Service) CompactSessionContext(ctx context.Context, input domain.CompactSessionContextInput) (domain.CompactSessionContextResult, error) {
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return domain.CompactSessionContextResult{}, errors.New("sessionId is required")
	}
	compacting, err := s.store.UpsertSessionExecutionState(ctx, domain.SessionExecutionState{SessionID: sessionID, Status: domain.ExecutionStatusCompacting, Reason: "manual compaction requested"})
	if err != nil {
		return domain.CompactSessionContextResult{}, err
	}
	summary, err := s.CreateSummary(ctx, domain.CreateSummaryRequest{SessionID: sessionID})
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
		Payload: map[string]any{"kind": "context_compacted", "summaryId": summary.ID, "sectionCount": len(contextResult.Sections)},
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

func (s *Service) ListSessionEventsAfterCursor(ctx context.Context, input domain.ListSessionEventsAfterCursorInput) (domain.ListSessionEventsAfterCursorResult, error) {
	if strings.TrimSpace(input.SessionID) == "" {
		return domain.ListSessionEventsAfterCursorResult{}, errors.New("sessionId is required")
	}
	events, next, err := s.store.ListSessionEventsAfterCursor(ctx, input.SessionID, input.Cursor, input.IncludeNonNormal, input.Limit)
	if err != nil {
		return domain.ListSessionEventsAfterCursorResult{}, err
	}
	return domain.ListSessionEventsAfterCursorResult{Events: events, NextCursor: next}, nil
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

func (s *Service) updateUntitledSession(ctx context.Context, sessionID, content string) error {
	session, err := s.store.GetRuntimeSession(ctx, sessionID)
	if err != nil || !isLegacyUntitledSessionTitle(session.Title, content) {
		return err
	}
	_, err = s.updateSessionTitle(ctx, sessionID, fallbackSessionTitle(content), false)
	return err
}

func (s *Service) ensureGeneratedSessionTitle(ctx context.Context, sessionID string, model *domain.ModelRef) {
	session, err := s.store.GetRuntimeSession(ctx, sessionID)
	if err != nil {
		return
	}
	events, err := s.store.ListSessionEvents(ctx, sessionID, false, 20)
	if err != nil {
		return
	}
	var userText string
	var userCount int
	for _, event := range events {
		if event.Type != domain.EventTypeUserMessage || strings.TrimSpace(event.Content) == "" {
			continue
		}
		userCount++
		userText = event.Content
	}
	if userCount != 1 || !isDefaultSessionTitle(session.Title, userText) {
		return
	}
	title := ""
	if s.titleGenerator != nil && model != nil && strings.TrimSpace(model.ModelID) != "" {
		generated, err := s.titleGenerator(ctx, userText, model)
		if err != nil {
			fmt.Printf("session title generation failed for %s/%s: %v\n", model.ProviderID, model.ModelID, err)
		}
		title = cleanGeneratedSessionTitle(generated)
	}
	if title == "" {
		return
	}
	_, _ = s.updateSessionTitle(context.Background(), sessionID, title, true)
}

func (s *Service) generateSessionTitle(ctx context.Context, userText string, model *domain.ModelRef) (string, error) {
	messages := []domain.ChatMessage{
		{Role: "system", Text: sessionTitleSystemPrompt},
		{Role: "user", Text: "Generate a title for this conversation:\n\n" + strings.TrimSpace(userText)},
	}
	var failures []string
	for _, titleModel := range s.resolveAuxiliaryModels(ctx, model) {
		title, _, err := s.GenerateChatReply(ctx, messages, &titleModel, "medium", "default")
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s/%s: %v", titleModel.ProviderID, titleModel.ModelID, err))
			continue
		}
		title = cleanGeneratedSessionTitle(title)
		if title != "" {
			return title, nil
		}
		failures = append(failures, fmt.Sprintf("%s/%s: empty title", titleModel.ProviderID, titleModel.ModelID))
	}
	if len(failures) > 0 {
		return "", errors.New(strings.Join(failures, "; "))
	}
	return "", nil
}

const sessionSummarySystemPrompt = `You are a conversation summarizer. Output ONLY a concise durable summary.

Rules:
- Keep the summary factual and compact.
- Preserve user goals, decisions, constraints, open tasks, files, commands, errors, and important technical terms.
- Do not include markdown headings, bullets, preambles, or commentary.
- Use the same primary language as the conversation.`

func (s *Service) generatedSummary(ctx context.Context, sessionID string) string {
	events, err := s.store.ListSessionEvents(ctx, sessionID, false, 80)
	if err != nil || len(events) == 0 {
		return ""
	}
	transcript := renderEventsForSummary(events)
	if strings.TrimSpace(transcript) == "" {
		return ""
	}
	for _, model := range s.configuredAuxiliaryModels(ctx) {
		summary, _, err := s.GenerateChatReply(ctx, []domain.ChatMessage{
			{Role: "system", Text: sessionSummarySystemPrompt},
			{Role: "user", Text: "Summarize this conversation for future context:\n\n" + transcript},
		}, &model, "medium", "default")
		if err == nil && strings.TrimSpace(summary) != "" {
			return bounded(strings.TrimSpace(stripThinkBlocks(summary)), 4000)
		}
	}
	return ""
}

func renderEventsForSummary(events []domain.SessionEvent) string {
	parts := make([]string, 0, len(events))
	for _, event := range events {
		content := strings.TrimSpace(event.Content)
		if content == "" {
			continue
		}
		role := strings.TrimSpace(event.Role)
		if role == "" {
			role = event.Type
		}
		parts = append(parts, role+": "+bounded(content, 1200))
	}
	return bounded(strings.Join(parts, "\n\n"), 16000)
}

func (s *Service) configuredAuxiliaryModels(ctx context.Context) []domain.ModelRef {
	cfg, err := s.AppConfig(ctx)
	if err != nil || cfg.AuxiliaryModel == nil || strings.TrimSpace(cfg.AuxiliaryModel.ModelID) == "" {
		return nil
	}
	return s.resolveAuxiliaryModels(ctx, nil)
}

func (s *Service) resolveAuxiliaryModels(ctx context.Context, fallback *domain.ModelRef) []domain.ModelRef {
	cfg, err := s.AppConfig(ctx)
	if err == nil && cfg.AuxiliaryModel != nil && strings.TrimSpace(cfg.AuxiliaryModel.ModelID) != "" {
		auxiliary := *cfg.AuxiliaryModel
		models := []domain.ModelRef{auxiliary}
		for _, model := range s.resolveTitleModels(ctx, fallback) {
			if model != auxiliary {
				models = append(models, model)
			}
		}
		return models
	}
	return s.resolveTitleModels(ctx, fallback)
}

func (s *Service) resolveTitleModels(ctx context.Context, fallback *domain.ModelRef) []domain.ModelRef {
	if fallback == nil || strings.TrimSpace(fallback.ModelID) == "" {
		cfg, err := s.AppConfig(ctx)
		if err != nil || cfg.DefaultModel == nil || strings.TrimSpace(cfg.DefaultModel.ModelID) == "" {
			return nil
		}
		fallback = cfg.DefaultModel
	}
	fallbackModel := domain.ModelRef{
		ProviderID: strings.TrimSpace(fallback.ProviderID),
		ModelID:    strings.TrimSpace(fallback.ModelID),
	}
	catalog, err := s.Catalog(ctx)
	if err != nil {
		return []domain.ModelRef{fallbackModel}
	}
	for _, provider := range catalog.Providers {
		if provider.ID != fallbackModel.ProviderID || !provider.Connected {
			continue
		}
		if modelID := smallModelIDForTitleProvider(provider); modelID != "" {
			titleModel := domain.ModelRef{ProviderID: provider.ID, ModelID: modelID}
			if titleModel == fallbackModel {
				return []domain.ModelRef{fallbackModel}
			}
			return []domain.ModelRef{titleModel, fallbackModel}
		}
		break
	}
	return []domain.ModelRef{fallbackModel}
}

func smallModelIDForTitleProvider(provider domain.ProviderInfo) string {
	priority := []string{
		"claude-haiku-4-5",
		"claude-haiku-4.5",
		"3-5-haiku",
		"3.5-haiku",
		"gemini-3-flash",
		"gemini-2.5-flash",
		"gpt-5.4-mini",
		"gpt-5-mini",
	}
	if strings.HasPrefix(provider.ID, "opencode") {
		priority = []string{"gpt-5.4-mini", "gpt-5-mini"}
	}
	if strings.HasPrefix(provider.ID, "github-copilot") {
		priority = append([]string{"gpt-5-mini", "claude-haiku-4.5"}, priority...)
	}
	for _, item := range priority {
		if provider.ID == "amazon-bedrock" {
			if match := smallBedrockTitleModelID(provider.Models, item); match != "" {
				return match
			}
			continue
		}
		for _, model := range provider.Models {
			if strings.Contains(model.ID, item) {
				return model.ID
			}
		}
	}
	return ""
}

func smallBedrockTitleModelID(models []domain.ModelInfo, item string) string {
	var candidates []string
	for _, model := range models {
		if strings.Contains(model.ID, item) {
			candidates = append(candidates, model.ID)
		}
	}
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, "global.") {
			return candidate
		}
	}
	for _, candidate := range candidates {
		if !strings.HasPrefix(candidate, "global.") && !strings.HasPrefix(candidate, "us.") && !strings.HasPrefix(candidate, "eu.") {
			return candidate
		}
	}
	if len(candidates) > 0 {
		return candidates[0]
	}
	return ""
}

func (s *Service) updateSessionTitle(ctx context.Context, sessionID string, title string, verifyDefault bool) (domain.Session, error) {
	title = cleanGeneratedSessionTitle(title)
	if title == "" {
		return domain.Session{}, errors.New("title is empty")
	}
	if verifyDefault {
		current, err := s.store.GetRuntimeSession(ctx, sessionID)
		if err != nil {
			return domain.Session{}, err
		}
		events, err := s.store.ListSessionEvents(ctx, sessionID, false, 20)
		if err != nil {
			return domain.Session{}, err
		}
		firstUser := ""
		for _, event := range events {
			if event.Type == domain.EventTypeUserMessage && strings.TrimSpace(event.Content) != "" {
				firstUser = event.Content
				break
			}
		}
		if !isDefaultSessionTitle(current.Title, firstUser) {
			return current, nil
		}
	}
	updated, err := s.store.UpdateRuntimeSession(ctx, domain.UpdateSessionRequest{SessionID: sessionID, Title: title})
	if err == nil && s.onSessionUpdated != nil {
		s.onSessionUpdated(sessionID, &updated)
	}
	return updated, err
}

const sessionTitleSystemPrompt = `You are a title generator. You output ONLY a thread title. Nothing else.

Rules:
- The title must be a single line.
- The title must be <=50 characters.
- Use the same language as the user's message.
- Do not include quotes, punctuation wrappers, prefixes, explanations, markdown, bullets, or emojis.
- Do not mention tool names, model names, or implementation details unless they are the topic.
- Focus on the main task or question.
- Preserve important technical terms, filenames, frameworks, languages, and errors.
- Prefer concise noun phrases over full sentences.

Examples:
User: How do I fix TypeScript error TS2345 in my React app?
Title: Fix React TS2345 error

User: 帮我写一个 Redis 缓存方案
Title: Redis 缓存方案

User: What's the difference between OAuth and SAML?
Title: OAuth and SAML comparison`

func isLegacyUntitledSessionTitle(title string, firstUserText string) bool {
	title = strings.TrimSpace(title)
	switch title {
	case "", "Untitled session", "生成第一版MVP":
		return true
	}
	first := fallbackSessionTitle(firstUserText)
	return first != "" && title == first
}

func isDefaultSessionTitle(title string, firstUserText string) bool {
	title = strings.TrimSpace(title)
	switch title {
	case "", "Untitled session", "New chat", "生成第一版MVP":
		return true
	}
	if strings.HasPrefix(title, "New session - ") {
		return true
	}
	first := fallbackSessionTitle(firstUserText)
	return first != "" && title == first
}

func fallbackSessionTitle(text string) string {
	return cleanGeneratedSessionTitle(bounded(strings.TrimSpace(text), 80))
}

func cleanGeneratedSessionTitle(text string) string {
	text = strings.TrimSpace(stripThinkBlocks(text))
	text = strings.Trim(text, "\"'`“”‘’")
	for _, line := range strings.Split(text, "\n") {
		title := strings.TrimSpace(line)
		title = strings.Trim(title, "\"'`“”‘’")
		if title == "" {
			continue
		}
		runes := []rune(title)
		if len(runes) > 50 {
			title = string(runes[:50])
		}
		return title
	}
	return ""
}

func stripThinkBlocks(text string) string {
	for {
		lower := strings.ToLower(text)
		start := strings.Index(lower, "<think>")
		if start < 0 {
			return text
		}
		rest := text[start+len("<think>"):]
		end := strings.Index(strings.ToLower(rest), "</think>")
		if end < 0 {
			return strings.TrimSpace(text[:start])
		}
		text = text[:start] + rest[end+len("</think>"):]
	}
}

func (s *Service) fallbackSummary(ctx context.Context, sessionID string) string {
	events, err := s.store.ListSessionEvents(ctx, sessionID, false, 5)
	if err != nil || len(events) == 0 {
		return "No visible events have been recorded yet."
	}
	parts := make([]string, 0, len(events))
	for _, event := range events {
		if event.Content != "" {
			parts = append(parts, bounded(event.Content, 160))
		}
	}
	if len(parts) == 0 {
		return "Visible activity was recorded without message content."
	}
	return strings.Join(parts, "\n")
}

func gitOutput(ctx context.Context, dir string, args ...string) string {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func detectLanguageStack(dir string) []string {
	var stack []string
	if fileExists(filepath.Join(dir, "go.mod")) {
		stack = append(stack, "go")
	}
	if fileExists(filepath.Join(dir, "package.json")) {
		stack = append(stack, "typescript", "node")
	}
	return stack
}

func detectPackageManager(dir string) string {
	for _, item := range []struct{ file, name string }{{"pnpm-lock.yaml", "pnpm"}, {"yarn.lock", "yarn"}, {"package-lock.json", "npm"}, {"go.mod", "go"}} {
		if fileExists(filepath.Join(dir, item.file)) {
			return item.name
		}
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func lines(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, "\n")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if clean := strings.TrimSpace(part); clean != "" {
			out = append(out, clean)
		}
	}
	return out
}

func renderEvents(events []domain.SessionEvent) string {
	var out []string
	for _, event := range events {
		out = append(out, event.Type+": "+event.Content)
	}
	return strings.Join(out, "\n")
}

func renderTools(tools []domain.ToolCall) string {
	var out []string
	for _, tool := range tools {
		if tool.ResultSummary != "" {
			out = append(out, tool.Name+": "+tool.ResultSummary)
		}
	}
	return strings.Join(out, "\n")
}

func applyContextBudget(sessionID string, sections []domain.ContextSection, charBudget int, maxTokens int) domain.BuildSessionContextResult {
	if charBudget <= 0 && maxTokens > 0 {
		charBudget = maxTokens * 4
	}
	if charBudget <= 0 {
		charBudget = 12000
	}
	used := 0
	var truncated []string
	for i := range sections {
		content := sections[i].Content
		remaining := charBudget - used
		if remaining <= 0 {
			if content != "" {
				truncated = append(truncated, sections[i].Name)
			}
			sections[i].Content = ""
			sections[i].Truncated = true
			continue
		}
		if len(content) > remaining {
			sections[i].Content = content[:remaining]
			sections[i].Truncated = true
			truncated = append(truncated, sections[i].Name)
		}
		used += len(sections[i].Content)
	}
	return domain.BuildSessionContextResult{SessionID: sessionID, Sections: sections, EstimatedTokens: used / 4, CharacterBudget: charBudget, TruncatedSections: truncated}
}

func bounded(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit]
}
