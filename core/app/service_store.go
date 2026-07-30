package app

import (
	"context"

	"aivo/core/domain"
)

type Store interface {
	configStore
	providerStore
	providerAuthStore
	projectStore
	sessionStore
	permissionStore
	questionStore
	agentRunStore
	todoStore
	scheduledJobStore
}

type configStore interface {
	LoadConfig(context.Context) (domain.AppConfig, error)
	SaveConfig(context.Context, domain.AppConfig) error
}

type providerStore interface {
	SaveProvider(context.Context, domain.ProviderConfig) error
	ListProviders(context.Context) ([]domain.ProviderConfig, error)
	DeleteProvider(context.Context, string) error
	SaveProviderModelCache(context.Context, domain.ProviderModelCache) error
	LoadProviderModelCache(context.Context, string) (*domain.ProviderModelCache, error)
	ListProviderModelCaches(context.Context) ([]domain.ProviderModelCache, error)
	SaveProviderValidation(context.Context, domain.ProviderValidationResult) error
	LoadProviderValidation(context.Context, string) (*domain.ProviderValidationResult, error)
	SaveProviderHealth(context.Context, domain.ProviderHealth) error
	LoadProviderHealth(context.Context, string) (*domain.ProviderHealth, error)
	ListProviderHealth(context.Context) ([]domain.ProviderHealth, error)
	SaveProviderCallEvent(context.Context, domain.ProviderCallEvent) error
	ListProviderCallEvents(context.Context, string, int) ([]domain.ProviderCallEvent, error)
}

type providerAuthStore interface {
	SaveProviderAuth(context.Context, domain.ProviderAuthRecord) error
	GetProviderAuth(context.Context, string) (*domain.ProviderAuthRecord, error)
	LoadProviderAuth(context.Context, string) (*domain.ProviderAuthRecord, error)
	ListProviderAuths(context.Context, string) ([]domain.ProviderAuthRecord, error)
	DeleteProviderAuth(context.Context, string) error
}

type projectStore interface {
	UpsertProject(context.Context, string) (domain.AssistantProject, error)
	SetProjectSidebarHidden(context.Context, string, bool) (domain.AssistantProject, error)
	ListProjects(context.Context, int) ([]domain.AssistantProject, error)
	UpdateProjectDescription(context.Context, string, string) (domain.AssistantProject, error)
}

type sessionStore interface {
	CreateRuntimeSession(context.Context, domain.CreateSessionRequest) (domain.Session, error)
	GetRuntimeSession(context.Context, string) (domain.Session, error)
	ListRuntimeSessions(context.Context, domain.ListSessionsRequest) ([]domain.Session, error)
	UpdateRuntimeSession(context.Context, domain.UpdateSessionRequest) (domain.Session, error)
	SetRuntimeSessionStatus(context.Context, string, string) (domain.Session, error)
	SetRuntimeSessionAgentMode(context.Context, string, string) (domain.Session, error)
	SetRuntimeSessionProject(context.Context, string, string) (domain.Session, error)
	AppendSessionEvent(context.Context, domain.SessionEvent) error
	GetSessionEvent(context.Context, string) (domain.SessionEvent, error)
	ListSessionEvents(context.Context, string, bool, int) ([]domain.SessionEvent, error)
	UpdateSessionEvent(context.Context, domain.UpdateSessionEventRequest) (domain.SessionEvent, error)
	SetSessionEventVisibility(context.Context, string, string) (domain.SessionEvent, error)
	HideSessionTurnEvents(context.Context, string) error
	StartTurn(context.Context, domain.Turn) error
	GetTurn(context.Context, string) (domain.Turn, error)
	UpdateTurnStatus(context.Context, string, string, string) (domain.Turn, error)
	ListTurns(context.Context, string, int) ([]domain.Turn, error)
	SaveToolCall(context.Context, domain.ToolCall) error
	ListToolCalls(context.Context, string) ([]domain.ToolCall, error)
	UpsertSessionExecutionState(context.Context, domain.SessionExecutionState) (domain.SessionExecutionState, error)
	GetSessionExecutionState(context.Context, string) (domain.SessionExecutionState, error)
	CreatePendingSessionInput(context.Context, domain.PendingSessionInput) (domain.PendingSessionInput, error)
	ListPendingSessionInputs(context.Context, string, string) ([]domain.PendingSessionInput, error)
	UpdatePendingSessionInputStatus(context.Context, string, string, string) (domain.PendingSessionInput, error)
	ListSessionEventsAfterCursor(context.Context, string, string, bool, int) ([]domain.SessionEvent, string, error)
	MarkRunningToolCallsInterrupted(context.Context, string, string) (int, error)
	CreateSummary(context.Context, domain.SessionSummary) error
	LatestSummary(context.Context, string) (*domain.SessionSummary, error)
	CreateCheckpoint(context.Context, domain.SessionCheckpoint) error
	ListCheckpoints(context.Context, string, int) ([]domain.SessionCheckpoint, error)
	LatestCheckpoint(context.Context, string) (*domain.SessionCheckpoint, error)
	UpsertCodingContext(context.Context, domain.CodingContext) (domain.CodingContext, error)
	GetCodingContext(context.Context, string) (domain.CodingContext, error)
	LatestSessionByProject(context.Context, string) (*domain.Session, error)
	ForkRuntimeSession(context.Context, domain.Session, domain.ForkSessionRequest) (domain.Session, error)
}

type permissionStore interface {
	CreatePermissionRequest(context.Context, domain.PermissionRequest) (domain.PermissionRequest, error)
	GetPermissionRequest(context.Context, string) (domain.PermissionRequest, error)
	ListPermissionRequests(context.Context, string, string) ([]domain.PermissionRequest, error)
	UpdatePermissionRequest(context.Context, string, string, bool, string) (domain.PermissionRequest, error)
	SavePermissionRule(context.Context, domain.PermissionRule) (domain.PermissionRule, error)
	ListPermissionRules(context.Context, string, string) ([]domain.PermissionRule, error)
}

type questionStore interface {
	CreateQuestionRequest(context.Context, domain.QuestionRequest) (domain.QuestionRequest, error)
	GetQuestionRequest(context.Context, string) (domain.QuestionRequest, error)
	ListQuestionRequests(context.Context, string, string) ([]domain.QuestionRequest, error)
	UpdateQuestionRequest(context.Context, string, string, [][]string, string) (domain.QuestionRequest, error)
}

type agentRunStore interface {
	SaveAgentRun(context.Context, domain.AgentRun) (domain.AgentRun, error)
	ListAgentRuns(context.Context, domain.AgentRunListRequest) ([]domain.AgentRun, error)
	GetAgentRun(context.Context, string) (domain.AgentRun, error)
}

type todoStore interface {
	ReplaceTodoItems(context.Context, domain.TodoListInput, []domain.TodoItem) ([]domain.TodoItem, error)
	ListTodoItems(context.Context, domain.TodoListInput) ([]domain.TodoItem, error)
}

type scheduledJobStore interface {
	SaveScheduledJob(context.Context, domain.ScheduledJob) (domain.ScheduledJob, error)
	GetScheduledJob(context.Context, string) (domain.ScheduledJob, error)
	ListScheduledJobs(context.Context, domain.ScheduledJobListInput) ([]domain.ScheduledJob, error)
	ListDueScheduledJobs(context.Context, string, int) ([]domain.ScheduledJob, error)
	DeleteScheduledJob(context.Context, string) error
}
