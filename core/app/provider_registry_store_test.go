package app

import (
	"context"

	"github.com/google/uuid"

	"aivo/core/domain"
)

func (m *memoryProviderStore) LoadProviderAuth(_ context.Context, providerID string) (*domain.ProviderAuthRecord, error) {
	if m.auth == nil {
		return nil, nil
	}
	auth, ok := m.auth[providerID]
	if !ok {
		return nil, nil
	}
	return &auth, nil
}

func (m *memoryProviderStore) GetProviderAuth(_ context.Context, id string) (*domain.ProviderAuthRecord, error) {
	if m.authByID == nil {
		return nil, nil
	}
	auth, ok := m.authByID[id]
	if !ok {
		return nil, nil
	}
	return &auth, nil
}

func (m *memoryProviderStore) LoadConfig(context.Context) (domain.AppConfig, error) {
	if m.config != nil {
		return *m.config, nil
	}
	return domain.AppConfig{}, nil
}

func (m *memoryProviderStore) SaveConfig(_ context.Context, cfg domain.AppConfig) error {
	m.config = &cfg
	return nil
}

func (m *memoryProviderStore) SaveProvider(_ context.Context, provider domain.ProviderConfig) error {
	for i := range m.providers {
		if m.providers[i].ID == provider.ID {
			m.providers[i] = provider
			return nil
		}
	}
	m.providers = append(m.providers, provider)
	return nil
}

func (m *memoryProviderStore) ListProviders(context.Context) ([]domain.ProviderConfig, error) {
	return append([]domain.ProviderConfig(nil), m.providers...), nil
}

func (m *memoryProviderStore) DeleteProvider(_ context.Context, providerID string) error {
	next := m.providers[:0]
	for _, provider := range m.providers {
		if provider.ID != providerID {
			next = append(next, provider)
		}
	}
	m.providers = next
	delete(m.auth, providerID)
	delete(m.modelCaches, providerID)
	delete(m.health, providerID)
	return nil
}

func (m *memoryProviderStore) SaveProviderModelCache(_ context.Context, cache domain.ProviderModelCache) error {
	m.savedCache = &cache
	if m.modelCaches == nil {
		m.modelCaches = map[string]domain.ProviderModelCache{}
	}
	m.modelCaches[cache.ProviderID] = cache
	return nil
}

func (m *memoryProviderStore) LoadProviderModelCache(_ context.Context, providerID string) (*domain.ProviderModelCache, error) {
	if m.modelCaches == nil {
		return nil, nil
	}
	cache, ok := m.modelCaches[providerID]
	if !ok {
		return nil, nil
	}
	return &cache, nil
}

func (m *memoryProviderStore) ListProviderModelCaches(context.Context) ([]domain.ProviderModelCache, error) {
	if m.modelCaches == nil {
		return nil, nil
	}
	out := make([]domain.ProviderModelCache, 0, len(m.modelCaches))
	for _, cache := range m.modelCaches {
		out = append(out, cache)
	}
	return out, nil
}

func (m *memoryProviderStore) SaveProviderValidation(_ context.Context, result domain.ProviderValidationResult) error {
	m.savedValidation = &result
	return nil
}

func (m *memoryProviderStore) LoadProviderValidation(context.Context, string) (*domain.ProviderValidationResult, error) {
	return nil, nil
}

func (m *memoryProviderStore) SaveProviderHealth(_ context.Context, health domain.ProviderHealth) error {
	m.savedHealth = &health
	if m.health == nil {
		m.health = map[string]domain.ProviderHealth{}
	}
	m.health[health.ProviderID] = health
	return nil
}

func (m *memoryProviderStore) LoadProviderHealth(_ context.Context, providerID string) (*domain.ProviderHealth, error) {
	if m.health == nil {
		return nil, nil
	}
	health, ok := m.health[providerID]
	if !ok {
		return nil, nil
	}
	return &health, nil
}

func (m *memoryProviderStore) ListProviderHealth(context.Context) ([]domain.ProviderHealth, error) {
	if m.health == nil {
		return nil, nil
	}
	out := make([]domain.ProviderHealth, 0, len(m.health))
	for _, health := range m.health {
		out = append(out, health)
	}
	return out, nil
}

func (m *memoryProviderStore) SaveProviderCallEvent(_ context.Context, event domain.ProviderCallEvent) error {
	m.callEvents = append(m.callEvents, event)
	return nil
}

func (m *memoryProviderStore) ListProviderCallEvents(_ context.Context, providerID string, limit int) ([]domain.ProviderCallEvent, error) {
	var out []domain.ProviderCallEvent
	for _, event := range m.callEvents {
		if providerID == "" || event.ProviderID == providerID {
			out = append(out, event)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *memoryProviderStore) SaveProviderAuth(_ context.Context, auth domain.ProviderAuthRecord) error {
	m.savedAuth = &auth
	if m.auth == nil {
		m.auth = map[string]domain.ProviderAuthRecord{}
	}
	m.auth[auth.ProviderID] = auth
	if m.authByID == nil {
		m.authByID = map[string]domain.ProviderAuthRecord{}
	}
	if auth.ID != "" {
		m.authByID[auth.ID] = auth
	}
	return nil
}

func (m *memoryProviderStore) ListProviderAuths(_ context.Context, providerID string) ([]domain.ProviderAuthRecord, error) {
	var out []domain.ProviderAuthRecord
	if m.auth != nil {
		if auth, ok := m.auth[providerID]; ok {
			out = append(out, auth)
		}
	}
	if m.authByID != nil {
		seen := map[string]bool{}
		for _, auth := range out {
			seen[auth.ID] = true
		}
		for _, auth := range m.authByID {
			if auth.ProviderID == providerID && !seen[auth.ID] {
				out = append(out, auth)
			}
		}
	}
	return out, nil
}

func (m *memoryProviderStore) DeleteProviderAuth(_ context.Context, id string) error {
	delete(m.authByID, id)
	return nil
}

func (m *memoryProviderStore) UpsertProject(context.Context, string) (domain.AssistantProject, error) {
	return domain.AssistantProject{}, nil
}

func (m *memoryProviderStore) SetProjectSidebarHidden(context.Context, string, bool) (domain.AssistantProject, error) {
	return domain.AssistantProject{}, nil
}

func (m *memoryProviderStore) ListProjects(context.Context, int) ([]domain.AssistantProject, error) {
	return nil, nil
}

func (m *memoryProviderStore) UpdateProjectDescription(context.Context, string, string) (domain.AssistantProject, error) {
	return domain.AssistantProject{}, nil
}

func (m *memoryProviderStore) CreateRuntimeSession(_ context.Context, input domain.CreateSessionRequest) (domain.Session, error) {
	return domain.Session{
		ID:          uuid.NewString(),
		Type:        input.Type,
		Source:      input.Source,
		Title:       input.Title,
		ProjectPath: input.ProjectPath,
		AgentMode:   input.AgentMode,
	}, nil
}

func (m *memoryProviderStore) GetRuntimeSession(context.Context, string) (domain.Session, error) {
	return domain.Session{}, nil
}

func (m *memoryProviderStore) ListRuntimeSessions(context.Context, domain.ListSessionsRequest) ([]domain.Session, error) {
	return nil, nil
}

func (m *memoryProviderStore) UpdateRuntimeSession(context.Context, domain.UpdateSessionRequest) (domain.Session, error) {
	return domain.Session{}, nil
}

func (m *memoryProviderStore) SetRuntimeSessionStatus(context.Context, string, string) (domain.Session, error) {
	return domain.Session{}, nil
}

func (m *memoryProviderStore) SetRuntimeSessionAgentMode(context.Context, string, string) (domain.Session, error) {
	return domain.Session{}, nil
}

func (m *memoryProviderStore) SetRuntimeSessionProject(context.Context, string, string) (domain.Session, error) {
	return domain.Session{}, nil
}

func (m *memoryProviderStore) AppendSessionEvent(context.Context, domain.SessionEvent) error {
	return nil
}

func (m *memoryProviderStore) GetSessionEvent(context.Context, string) (domain.SessionEvent, error) {
	return domain.SessionEvent{}, nil
}

func (m *memoryProviderStore) ListSessionEvents(context.Context, string, bool, int) ([]domain.SessionEvent, error) {
	return nil, nil
}

func (m *memoryProviderStore) UpdateSessionEvent(context.Context, domain.UpdateSessionEventRequest) (domain.SessionEvent, error) {
	return domain.SessionEvent{}, nil
}

func (m *memoryProviderStore) SetSessionEventVisibility(context.Context, string, string) (domain.SessionEvent, error) {
	return domain.SessionEvent{}, nil
}

func (m *memoryProviderStore) HideSessionTurnEvents(context.Context, string) error {
	return nil
}

func (m *memoryProviderStore) StartTurn(context.Context, domain.Turn) error { return nil }

func (m *memoryProviderStore) GetTurn(context.Context, string) (domain.Turn, error) {
	return domain.Turn{}, nil
}

func (m *memoryProviderStore) UpdateTurnStatus(context.Context, string, string, string) (domain.Turn, error) {
	return domain.Turn{}, nil
}

func (m *memoryProviderStore) ListTurns(context.Context, string, int) ([]domain.Turn, error) {
	return nil, nil
}

func (m *memoryProviderStore) SaveToolCall(context.Context, domain.ToolCall) error { return nil }

func (m *memoryProviderStore) ListToolCalls(context.Context, string) ([]domain.ToolCall, error) {
	return nil, nil
}

func (m *memoryProviderStore) UpsertSessionExecutionState(_ context.Context, state domain.SessionExecutionState) (domain.SessionExecutionState, error) {
	return state, nil
}

func (m *memoryProviderStore) GetSessionExecutionState(_ context.Context, sessionID string) (domain.SessionExecutionState, error) {
	return domain.SessionExecutionState{SessionID: sessionID, Status: domain.ExecutionStatusIdle}, nil
}

func (m *memoryProviderStore) CreatePendingSessionInput(_ context.Context, input domain.PendingSessionInput) (domain.PendingSessionInput, error) {
	return input, nil
}

func (m *memoryProviderStore) ListPendingSessionInputs(context.Context, string, string) ([]domain.PendingSessionInput, error) {
	return nil, nil
}

func (m *memoryProviderStore) UpdatePendingSessionInputStatus(_ context.Context, _ string, status string, promotedTurnID string) (domain.PendingSessionInput, error) {
	return domain.PendingSessionInput{Status: status, PromotedTurnID: promotedTurnID}, nil
}

func (m *memoryProviderStore) ListSessionEventsAfterCursor(context.Context, string, string, bool, int) ([]domain.SessionEvent, string, error) {
	return nil, "", nil
}

func (m *memoryProviderStore) MarkRunningToolCallsInterrupted(context.Context, string, string) (int, error) {
	return 0, nil
}

func (m *memoryProviderStore) CreatePermissionRequest(context.Context, domain.PermissionRequest) (domain.PermissionRequest, error) {
	return domain.PermissionRequest{}, nil
}

func (m *memoryProviderStore) GetPermissionRequest(context.Context, string) (domain.PermissionRequest, error) {
	return domain.PermissionRequest{}, nil
}

func (m *memoryProviderStore) ListPermissionRequests(context.Context, string, string) ([]domain.PermissionRequest, error) {
	return nil, nil
}

func (m *memoryProviderStore) UpdatePermissionRequest(context.Context, string, string, bool, string) (domain.PermissionRequest, error) {
	return domain.PermissionRequest{}, nil
}

func (m *memoryProviderStore) SavePermissionRule(context.Context, domain.PermissionRule) (domain.PermissionRule, error) {
	return domain.PermissionRule{}, nil
}

func (m *memoryProviderStore) ListPermissionRules(context.Context, string, string) ([]domain.PermissionRule, error) {
	return nil, nil
}

func (m *memoryProviderStore) CreateQuestionRequest(context.Context, domain.QuestionRequest) (domain.QuestionRequest, error) {
	return domain.QuestionRequest{}, nil
}

func (m *memoryProviderStore) GetQuestionRequest(context.Context, string) (domain.QuestionRequest, error) {
	return domain.QuestionRequest{}, nil
}

func (m *memoryProviderStore) ListQuestionRequests(context.Context, string, string) ([]domain.QuestionRequest, error) {
	return nil, nil
}

func (m *memoryProviderStore) UpdateQuestionRequest(context.Context, string, string, [][]string, string) (domain.QuestionRequest, error) {
	return domain.QuestionRequest{}, nil
}

func (m *memoryProviderStore) CreateSummary(context.Context, domain.SessionSummary) error { return nil }

func (m *memoryProviderStore) LatestSummary(context.Context, string) (*domain.SessionSummary, error) {
	return nil, nil
}

func (m *memoryProviderStore) CreateCheckpoint(context.Context, domain.SessionCheckpoint) error {
	return nil
}

func (m *memoryProviderStore) ListCheckpoints(context.Context, string, int) ([]domain.SessionCheckpoint, error) {
	return nil, nil
}

func (m *memoryProviderStore) LatestCheckpoint(context.Context, string) (*domain.SessionCheckpoint, error) {
	return nil, nil
}

func (m *memoryProviderStore) UpsertCodingContext(context.Context, domain.CodingContext) (domain.CodingContext, error) {
	return domain.CodingContext{}, nil
}

func (m *memoryProviderStore) GetCodingContext(context.Context, string) (domain.CodingContext, error) {
	return domain.CodingContext{}, nil
}

func (m *memoryProviderStore) LatestSessionByProject(context.Context, string) (*domain.Session, error) {
	return nil, nil
}

func (m *memoryProviderStore) ForkRuntimeSession(context.Context, domain.Session, domain.ForkSessionRequest) (domain.Session, error) {
	return domain.Session{}, nil
}

func (m *memoryProviderStore) SaveAgentRun(context.Context, domain.AgentRun) (domain.AgentRun, error) {
	return domain.AgentRun{}, nil
}

func (m *memoryProviderStore) ListAgentRuns(context.Context, domain.AgentRunListRequest) ([]domain.AgentRun, error) {
	return nil, nil
}

func (m *memoryProviderStore) GetAgentRun(context.Context, string) (domain.AgentRun, error) {
	return domain.AgentRun{}, nil
}

func (m *memoryProviderStore) ReplaceTodoItems(context.Context, domain.TodoListInput, []domain.TodoItem) ([]domain.TodoItem, error) {
	return nil, nil
}

func (m *memoryProviderStore) ListTodoItems(context.Context, domain.TodoListInput) ([]domain.TodoItem, error) {
	return nil, nil
}

func (m *memoryProviderStore) SaveScheduledJob(context.Context, domain.ScheduledJob) (domain.ScheduledJob, error) {
	return domain.ScheduledJob{}, nil
}

func (m *memoryProviderStore) GetScheduledJob(context.Context, string) (domain.ScheduledJob, error) {
	return domain.ScheduledJob{}, nil
}

func (m *memoryProviderStore) ListScheduledJobs(context.Context, domain.ScheduledJobListInput) ([]domain.ScheduledJob, error) {
	return nil, nil
}

func (m *memoryProviderStore) ListDueScheduledJobs(context.Context, string, int) ([]domain.ScheduledJob, error) {
	return nil, nil
}

func (m *memoryProviderStore) DeleteScheduledJob(context.Context, string) error {
	return nil
}
