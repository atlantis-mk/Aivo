package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"aivo/core/domain"
)

var ErrMaxStepsExceeded = errors.New("maximum tool calling steps exceeded")

const sessionMetadataRememberedDeferredTools = "rememberedDeferredTools"

type Store interface {
	LoadConfig(context.Context) (domain.AppConfig, error)
	SaveConfig(context.Context, domain.AppConfig) error
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
	SaveProviderAuth(context.Context, domain.ProviderAuthRecord) error
	GetProviderAuth(context.Context, string) (*domain.ProviderAuthRecord, error)
	LoadProviderAuth(context.Context, string) (*domain.ProviderAuthRecord, error)
	ListProviderAuths(context.Context, string) ([]domain.ProviderAuthRecord, error)
	DeleteProviderAuth(context.Context, string) error
	UpsertProject(context.Context, string) (domain.AssistantProject, error)
	SetProjectSidebarHidden(context.Context, string, bool) (domain.AssistantProject, error)
	ListProjects(context.Context, int) ([]domain.AssistantProject, error)
	CreateRuntimeSession(context.Context, domain.CreateSessionRequest) (domain.Session, error)
	GetRuntimeSession(context.Context, string) (domain.Session, error)
	ListRuntimeSessions(context.Context, domain.ListSessionsRequest) ([]domain.Session, error)
	UpdateRuntimeSession(context.Context, domain.UpdateSessionRequest) (domain.Session, error)
	SetRuntimeSessionStatus(context.Context, string, string) (domain.Session, error)
	SetRuntimeSessionAgentMode(context.Context, string, string) (domain.Session, error)
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
	CreatePermissionRequest(context.Context, domain.PermissionRequest) (domain.PermissionRequest, error)
	GetPermissionRequest(context.Context, string) (domain.PermissionRequest, error)
	ListPermissionRequests(context.Context, string, string) ([]domain.PermissionRequest, error)
	UpdatePermissionRequest(context.Context, string, string, bool, string) (domain.PermissionRequest, error)
	SavePermissionRule(context.Context, domain.PermissionRule) (domain.PermissionRule, error)
	ListPermissionRules(context.Context, string, string) ([]domain.PermissionRule, error)
	CreateQuestionRequest(context.Context, domain.QuestionRequest) (domain.QuestionRequest, error)
	GetQuestionRequest(context.Context, string) (domain.QuestionRequest, error)
	ListQuestionRequests(context.Context, string, string) ([]domain.QuestionRequest, error)
	UpdateQuestionRequest(context.Context, string, string, [][]string, string) (domain.QuestionRequest, error)
	CreateSummary(context.Context, domain.SessionSummary) error
	LatestSummary(context.Context, string) (*domain.SessionSummary, error)
	CreateCheckpoint(context.Context, domain.SessionCheckpoint) error
	ListCheckpoints(context.Context, string, int) ([]domain.SessionCheckpoint, error)
	LatestCheckpoint(context.Context, string) (*domain.SessionCheckpoint, error)
	UpsertCodingContext(context.Context, domain.CodingContext) (domain.CodingContext, error)
	GetCodingContext(context.Context, string) (domain.CodingContext, error)
	LatestSessionByProject(context.Context, string) (*domain.Session, error)
	ForkRuntimeSession(context.Context, domain.Session, domain.ForkSessionRequest) (domain.Session, error)
	SaveAgentRun(context.Context, domain.AgentRun) (domain.AgentRun, error)
	ListAgentRuns(context.Context, domain.AgentRunListRequest) ([]domain.AgentRun, error)
	GetAgentRun(context.Context, string) (domain.AgentRun, error)
	ReplaceTodoItems(context.Context, domain.TodoListInput, []domain.TodoItem) ([]domain.TodoItem, error)
	ListTodoItems(context.Context, domain.TodoListInput) ([]domain.TodoItem, error)
	SaveScheduledJob(context.Context, domain.ScheduledJob) (domain.ScheduledJob, error)
	GetScheduledJob(context.Context, string) (domain.ScheduledJob, error)
	ListScheduledJobs(context.Context, domain.ScheduledJobListInput) ([]domain.ScheduledJob, error)
	ListDueScheduledJobs(context.Context, string, int) ([]domain.ScheduledJob, error)
	DeleteScheduledJob(context.Context, string) error
}

type Service struct {
	store                 Store
	now                   func() time.Time
	authFlows             *ProviderAuthManager
	onAuthSuccess         func()
	onProviderAuthUpdated func(domain.ProviderAuthStatus)
	onSessionUpdated      func(string, *domain.Session)
	onTurnUpdated         func(string, domain.Turn)
	onAssistantDelta      func(sessionID string, turnID string, delta string)
	onToolCallUpdated     func(string, string, domain.ToolCall, bool)
	onShellOutput         func(ShellOutputEvent)
	onTodoItemsUpdated    func(sessionID string, projectPath string, items []domain.TodoItem)
	onPermissionRequested func(domain.PermissionRequest)
	onPermissionResolved  func(domain.PermissionRequest)
	onQuestionRequested   func(domain.QuestionRequest)
	onQuestionResolved    func(domain.QuestionRequest)
	onTerminalEvent       func(string, TerminalInfo)
	permissionNotifier    *permissionNotifier
	questionNotifier      *permissionNotifier
	terminals             *DefaultTerminalService
	titleGenerator        func(context.Context, string, *domain.ModelRef) (string, error)
	secrets               SecretStore
	providers             *ProviderRegistry
	agentCatalog          *AgentCatalog
	rateLimiter           *providerRateLimiter
	modelRefreshMu        sync.Mutex
	refreshedModels       map[string][]domain.ModelInfo
	refreshedDefault      map[string]string
	refreshedInfo         map[string]domain.ProviderInfo
	schedulerCancel       context.CancelFunc
	activeAgentRunMu      sync.Mutex
	activeAgentRunCancel  map[string]context.CancelFunc
	activeTurnMu          sync.Mutex
	activeTurnCancel      map[string]context.CancelFunc
	pluginManager         *PluginManager
	mcpManager            *MCPManager
}

func NewService(store Store) *Service {
	service := &Service{
		store:                store,
		now:                  time.Now,
		secrets:              NewDefaultSecretStore(),
		providers:            NewDefaultProviderRegistry(),
		agentCatalog:         NewAgentCatalog(),
		rateLimiter:          newProviderRateLimiter(),
		permissionNotifier:   newPermissionNotifier(),
		questionNotifier:     newPermissionNotifier(),
		terminals:            NewTerminalService(),
		refreshedModels:      map[string][]domain.ModelInfo{},
		refreshedDefault:     map[string]string{},
		refreshedInfo:        map[string]domain.ProviderInfo{},
		activeAgentRunCancel: map[string]context.CancelFunc{},
		activeTurnCancel:     map[string]context.CancelFunc{},
	}
	service.pluginManager = NewPluginManager(store)
	service.mcpManager = NewMCPManager(store, service.secrets)
	service.authFlows = NewProviderAuthManager(service)
	service.startSchedulerLoop()
	service.terminals.SetEventHook(func(name string, info TerminalInfo) {
		if service.onTerminalEvent != nil {
			service.onTerminalEvent(name, info)
		}
	})
	service.titleGenerator = service.generateSessionTitle
	_, _ = store.MarkRunningToolCallsInterrupted(context.Background(), "", "Interrupted during startup recovery; not replayed automatically")
	return service
}

func (s *Service) RegisterProviderDefinition(def ProviderDefinition) error {
	if s.providers == nil {
		s.providers = NewDefaultProviderRegistry()
	}
	return s.providers.RegisterDefinition(def)
}

func (s *Service) RegisterProviderAuthDriver(driver ProviderAuthDriver) {
	if s.authFlows == nil {
		s.authFlows = NewProviderAuthManager(s)
	}
	s.authFlows.RegisterDriver(driver)
}

func (s *Service) SetSecretStore(store SecretStore) {
	if store == nil {
		store = NewMemorySecretStore()
	}
	s.secrets = store
	if s.mcpManager != nil {
		s.mcpManager.SetSecretStore(store)
	}
}

func (s *Service) SetAuthSuccessHook(hook func()) {
	s.onAuthSuccess = hook
}

func (s *Service) SetProviderAuthUpdatedHook(hook func(domain.ProviderAuthStatus)) {
	s.onProviderAuthUpdated = hook
}

func (s *Service) SetSessionUpdatedHook(hook func(string, *domain.Session)) {
	s.onSessionUpdated = hook
}

func (s *Service) SetTurnUpdatedHook(hook func(string, domain.Turn)) {
	s.onTurnUpdated = hook
}

func (s *Service) SetAssistantDeltaHook(hook func(sessionID string, turnID string, delta string)) {
	s.onAssistantDelta = hook
}

func (s *Service) SetToolCallUpdatedHook(hook func(sessionID string, turnID string, call domain.ToolCall, created bool)) {
	s.onToolCallUpdated = hook
}

func (s *Service) SetShellOutputHook(hook func(ShellOutputEvent)) {
	s.onShellOutput = hook
}

func (s *Service) SetTodoItemsUpdatedHook(hook func(sessionID string, projectPath string, items []domain.TodoItem)) {
	s.onTodoItemsUpdated = hook
}

func (s *Service) SetPermissionRequestedHook(hook func(domain.PermissionRequest)) {
	s.onPermissionRequested = hook
}

func (s *Service) SetPermissionResolvedHook(hook func(domain.PermissionRequest)) {
	s.onPermissionResolved = hook
}

func (s *Service) SetQuestionRequestedHook(hook func(domain.QuestionRequest)) {
	s.onQuestionRequested = hook
}

func (s *Service) SetQuestionResolvedHook(hook func(domain.QuestionRequest)) {
	s.onQuestionResolved = hook
}

func (s *Service) SetTerminalEventHook(hook func(string, TerminalInfo)) {
	s.onTerminalEvent = hook
}

func (s *Service) ListTerminals(ctx context.Context, workspaceRoot string) ([]TerminalInfo, error) {
	return s.terminals.List(ctx, workspaceRoot)
}

func (s *Service) CreateTerminal(ctx context.Context, input TerminalCreateInput) (TerminalInfo, error) {
	return s.terminals.Create(ctx, input)
}

func (s *Service) GetTerminal(ctx context.Context, workspaceRoot string, terminalID string) (TerminalInfo, error) {
	return s.terminals.Get(ctx, workspaceRoot, terminalID)
}

func (s *Service) UpdateTerminal(ctx context.Context, input TerminalUpdateInput) (TerminalInfo, error) {
	return s.terminals.Update(ctx, input)
}

func (s *Service) RemoveTerminal(ctx context.Context, workspaceRoot string, terminalID string) error {
	return s.terminals.Remove(ctx, workspaceRoot, terminalID)
}

func (s *Service) AttachTerminal(ctx context.Context, input TerminalAttachInput) (TerminalAttachment, error) {
	return s.terminals.Attach(ctx, input)
}

func (s *Service) PollShellProcess(id string) (ShellProcessInfo, error) {
	return defaultShellProcessRegistry.Poll(id)
}

func (s *Service) WaitShellProcess(ctx context.Context, id string) (ShellProcessInfo, error) {
	return defaultShellProcessRegistry.Wait(ctx, id)
}

func (s *Service) KillShellProcess(id string) (ShellProcessInfo, error) {
	return defaultShellProcessRegistry.Kill(id)
}

func (s *Service) ReadShellProcessOutput(id string) (ShellProcessInfo, error) {
	return defaultShellProcessRegistry.ReadOutput(id)
}

func (s *Service) Shutdown() {
	if s.schedulerCancel != nil {
		s.schedulerCancel()
		s.schedulerCancel = nil
	}
	if s.terminals != nil {
		s.terminals.Shutdown()
	}
	defaultShellProcessRegistry.Shutdown()
	defaultAgentShellRegistry.Shutdown()
}

func (s *Service) AppConfig(ctx context.Context) (domain.AppConfig, error) {
	cfg, err := s.store.LoadConfig(ctx)
	if err != nil {
		return domain.AppConfig{}, err
	}
	normalizeLegacyConfigModels(&cfg)
	cfg.Persistence.Configured = true
	cfg.Persistence.JournalEnabled = true
	cfg.Persistence.DualWriteValidation = true
	cfg.ProviderPolicy = normalizeProviderRuntimePolicy(cfg.ProviderPolicy)
	if cfg.Persistence.ReadPath == "" {
		cfg.Persistence.ReadPath = "sqlite"
	}
	return cfg, nil
}

func normalizeLegacyConfigModels(cfg *domain.AppConfig) {
	if cfg == nil {
		return
	}
	if cfg.Provider != nil && cfg.Provider.ID == "openai" && cfg.Provider.Model == "gpt-5-codex" {
		cfg.Provider.Model = "gpt-5.5"
	}
	if cfg.DefaultModel != nil && cfg.DefaultModel.ProviderID == "openai" && cfg.DefaultModel.ModelID == "gpt-5-codex" {
		cfg.DefaultModel.ModelID = "gpt-5.5"
	}
	if cfg.AuxiliaryModel != nil && cfg.AuxiliaryModel.ProviderID == "openai" && cfg.AuxiliaryModel.ModelID == "gpt-5-codex" {
		cfg.AuxiliaryModel.ModelID = "gpt-5.5"
	}
	cfg.ReasoningEffort = normalizeReasoningEffort(cfg.ReasoningEffort)
	cfg.ServiceTier = normalizeServiceTier(cfg.ServiceTier)
}

func normalizeReasoningEffort(effort string) string {
	switch strings.TrimSpace(strings.ToLower(effort)) {
	case "low", "medium", "high", "ultra":
		return strings.TrimSpace(strings.ToLower(effort))
	case "低":
		return "low"
	case "中":
		return "medium"
	case "高":
		return "high"
	case "超高":
		return "ultra"
	default:
		return "medium"
	}
}

func normalizeServiceTier(serviceTier string) string {
	switch strings.TrimSpace(strings.ToLower(serviceTier)) {
	case "priority", "fast":
		return "priority"
	case "default", "":
		return "default"
	default:
		return "default"
	}
}

func providerSupportsServiceTier(providerID string) bool {
	return strings.TrimSpace(providerID) == "openai"
}

func (s *Service) Catalog(ctx context.Context) (domain.CatalogState, error) {
	cfg, err := s.AppConfig(ctx)
	if err != nil {
		return domain.CatalogState{}, err
	}
	providers := s.applyRefreshedProviderModels(s.defaultProviders())
	if savedProviders, err := s.store.ListProviders(ctx); err == nil {
		providers = s.mergeSavedProviders(providers, savedProviders)
	}
	if caches, err := s.store.ListProviderModelCaches(ctx); err == nil {
		providers = s.mergeProviderModelCaches(providers, caches)
	}
	if health, err := s.store.ListProviderHealth(ctx); err == nil {
		providers = mergeProviderHealth(providers, health)
	}
	var connected []string
	var connectedProviders []domain.ProviderInfo
	for i := range providers {
		def := s.providerDefinitionForConfig(domain.ProviderConfig{ID: providers[i].ID, Type: providers[i].Type, BaseURL: providers[i].BaseURL, APIKeyEnv: providers[i].Environment, Model: providers[i].DefaultModelID})
		authRecord, _ := s.store.LoadProviderAuth(ctx, providers[i].ID)
		authRecords, _ := s.store.ListProviderAuths(ctx, providers[i].ID)
		providers[i].Accounts = providerAccountsFromAuth(authRecords)
		if (cfg.Provider != nil && providers[i].ID == cfg.Provider.ID) || len(authRecords) > 0 || authRecord != nil {
			providers[i].Connected = true
			source := "credential-reference"
			if authRecord != nil {
				source = authRecord.Method
			}
			providers[i].ConnectionSource = source
			providers[i].Auth = &domain.AuthInfo{
				Type:      source,
				Connected: true,
				Source:    source,
			}
			if providers[i].Environment != "" {
				providers[i].Auth.Environment = providers[i].Environment
			}
			if authRecord != nil {
				providers[i].Auth.LastValidatedAt = authRecord.UpdatedAt
				providers[i].Auth.ConnectedAt = authRecord.UpdatedAt
			}
			connected = append(connected, providers[i].ID)
			connectedProviders = append(connectedProviders, providers[i])
		}
		if providers[i].Readiness == nil {
			providers[i].Readiness = providerReadiness(providers[i], def, authRecord)
		}
		if providers[i].ModelRefresh != nil {
			providers[i].ModelRefresh.ModelCount = len(providers[i].Models)
		}
	}
	sortProviderInfo(providers)
	return domain.CatalogState{
		Providers:          providers,
		Models:             flattenModels(providers),
		Connected:          connected,
		DefaultModel:       cfg.DefaultModel,
		ConnectedProviders: connectedProviders,
		PopularProviders:   providers,
	}, nil
}

func mergeProviderHealth(providers []domain.ProviderInfo, health []domain.ProviderHealth) []domain.ProviderInfo {
	if len(health) == 0 {
		return providers
	}
	index := map[string]int{}
	for i := range providers {
		index[providers[i].ID] = i
	}
	for _, item := range health {
		providerID := normalizeProviderID(item.ProviderID)
		if providerID == "" {
			continue
		}
		item.ProviderID = providerID
		if i, ok := index[providerID]; ok {
			providers[i].Health = &item
		}
	}
	return providers
}

func (s *Service) mergeSavedProviders(providers []domain.ProviderInfo, saved []domain.ProviderConfig) []domain.ProviderInfo {
	index := map[string]int{}
	for i := range providers {
		index[providers[i].ID] = i
	}
	for _, cfg := range saved {
		cfg.ID = normalizeProviderID(cfg.ID)
		if cfg.ID == "" {
			continue
		}
		def := s.providerDefinitionForConfig(cfg)
		info := providerInfoFromDefinition(def)
		info.ID = cfg.ID
		info.Type = firstNonEmpty(cfg.Type, info.Type)
		info.BaseURL = firstNonEmpty(cfg.BaseURL, info.BaseURL)
		info.Environment = firstNonEmpty(cfg.APIKeyEnv, info.Environment)
		info.DefaultModelID = firstNonEmpty(cfg.Model, info.DefaultModelID)
		info.Custom = !info.BuiltIn || !s.isBuiltInProvider(cfg.ID)
		if info.DefaultModelID != "" && !modelListContains(info.Models, info.DefaultModelID) {
			info.Models = append([]domain.ModelInfo{{ID: info.DefaultModelID, ProviderID: cfg.ID, Name: info.DefaultModelID, Recommended: true}}, info.Models...)
		}
		if i, ok := index[cfg.ID]; ok {
			providers[i].BaseURL = info.BaseURL
			providers[i].Environment = info.Environment
			providers[i].DefaultModelID = info.DefaultModelID
			providers[i].Type = info.Type
			if info.DefaultModelID != "" {
				markRecommended(providers[i].Models, info.DefaultModelID)
			}
			continue
		}
		index[cfg.ID] = len(providers)
		providers = append(providers, info)
	}
	return providers
}

func (s *Service) mergeProviderModelCaches(providers []domain.ProviderInfo, caches []domain.ProviderModelCache) []domain.ProviderInfo {
	index := map[string]int{}
	for i := range providers {
		index[providers[i].ID] = i
	}
	for _, cache := range caches {
		providerID := normalizeProviderID(cache.ProviderID)
		if providerID == "" || len(cache.Models) == 0 {
			continue
		}
		models := append([]domain.ModelInfo(nil), cache.Models...)
		defaultModel := cache.DefaultModel
		if defaultModel == "" {
			defaultModel = models[0].ID
		}
		markRecommended(models, defaultModel)
		if i, ok := index[providerID]; ok {
			providers[i].Models = models
			providers[i].DefaultModelID = defaultModel
			if providers[i].ModelRefresh == nil {
				providers[i].ModelRefresh = &domain.ProviderModelRefresh{}
			}
			providers[i].ModelRefresh.Status = firstNonEmpty(cache.Status, "ready")
			providers[i].ModelRefresh.LastRefresh = cache.RefreshedAt
			providers[i].ModelRefresh.Error = cache.Error
			providers[i].ModelRefresh.ModelCount = len(models)
			providers[i].ModelRefresh.CacheSource = firstNonEmpty(cache.CacheSource, "sqlite")
			providers[i].ModelRefresh.ParserType = cache.ParserType
			providers[i].ModelRefresh.Endpoint = cache.Endpoint
			providers[i].ModelRefresh.Stale = cache.Status == "stale"
			continue
		}
		def := s.providerDefinitionForConfig(domain.ProviderConfig{ID: providerID, Type: "", Model: defaultModel})
		info := providerInfoFromDefinition(def)
		info.ID = providerID
		info.Models = models
		info.DefaultModelID = defaultModel
		info.Custom = !s.isBuiltInProvider(providerID)
		if info.ModelRefresh == nil {
			info.ModelRefresh = &domain.ProviderModelRefresh{}
		}
		info.ModelRefresh.Status = firstNonEmpty(cache.Status, "ready")
		info.ModelRefresh.LastRefresh = cache.RefreshedAt
		info.ModelRefresh.ModelCount = len(models)
		info.ModelRefresh.CacheSource = firstNonEmpty(cache.CacheSource, "sqlite")
		info.ModelRefresh.ParserType = cache.ParserType
		info.ModelRefresh.Endpoint = cache.Endpoint
		index[providerID] = len(providers)
		providers = append(providers, info)
	}
	return providers
}

func (s *Service) isBuiltInProvider(providerID string) bool {
	def, ok := s.providerDefinition(providerID)
	return ok && def.BuiltIn
}

func modelListContains(models []domain.ModelInfo, modelID string) bool {
	for _, item := range models {
		if item.ID == modelID {
			return true
		}
	}
	return false
}

func (s *Service) ConnectProvider(ctx context.Context, input domain.ProviderConnectInput) (domain.CatalogState, error) {
	cfg, _, err := s.providerConfigFromInput(input)
	if err != nil {
		return domain.CatalogState{}, err
	}
	providerID := cfg.ID
	appCfg, err := s.AppConfig(ctx)
	if err != nil {
		return domain.CatalogState{}, err
	}
	appCfg.Initialized = true
	appCfg.Provider = &cfg
	appCfg.DefaultModel = &domain.ModelRef{ProviderID: cfg.ID, ModelID: cfg.Model}
	if err := s.store.SaveProvider(ctx, cfg); err != nil {
		return domain.CatalogState{}, err
	}
	method := strings.TrimSpace(input.Method)
	if method != "" && method != "env" {
		existingAuth, _ := s.store.LoadProviderAuth(ctx, providerID)
		shouldSaveAuth := !isOAuthMethod(method) || existingAuth == nil
		if shouldSaveAuth {
			if err := s.saveProviderAuth(ctx, domain.ProviderAuthRecord{
				ProviderID: providerID,
				Method:     method,
				AccountID:  connectAccountLabel(providerID, method, strings.TrimSpace(input.APIKey), strings.TrimSpace(input.APIKeyEnv)),
				APIKey:     strings.TrimSpace(input.APIKey),
				UpdatedAt:  domain.NowString(s.now()),
			}); err != nil {
				return domain.CatalogState{}, err
			}
		}
	}
	if err := s.store.SaveConfig(ctx, appCfg); err != nil {
		return domain.CatalogState{}, err
	}
	return s.Catalog(ctx)
}

func (s *Service) SaveProvider(ctx context.Context, input domain.ProviderConnectInput) (domain.CatalogState, error) {
	cfg, _, err := s.providerConfigFromInput(input)
	if err != nil {
		return domain.CatalogState{}, err
	}
	if err := s.store.SaveProvider(ctx, cfg); err != nil {
		return domain.CatalogState{}, err
	}
	method := strings.TrimSpace(input.Method)
	if method != "" && method != "env" && strings.TrimSpace(input.APIKey) != "" {
		if err := s.saveProviderAuth(ctx, domain.ProviderAuthRecord{
			ProviderID: cfg.ID,
			Method:     method,
			AccountID:  connectAccountLabel(cfg.ID, method, strings.TrimSpace(input.APIKey), strings.TrimSpace(input.APIKeyEnv)),
			APIKey:     strings.TrimSpace(input.APIKey),
			UpdatedAt:  domain.NowString(s.now()),
		}); err != nil {
			return domain.CatalogState{}, err
		}
	}
	return s.Catalog(ctx)
}

func (s *Service) DeleteProvider(ctx context.Context, providerID string) (domain.CatalogState, error) {
	providerID = s.normalizeProviderID(providerID)
	if providerID == "" {
		return domain.CatalogState{}, errors.New("provider is required")
	}
	auths, _ := s.store.ListProviderAuths(ctx, providerID)
	for _, auth := range auths {
		next := auth
		_ = s.deleteProviderAuthSecrets(ctx, &next)
	}
	if err := s.store.DeleteProvider(ctx, providerID); err != nil {
		return domain.CatalogState{}, err
	}
	cfg, err := s.AppConfig(ctx)
	if err == nil {
		changed := false
		if cfg.Provider != nil && s.normalizeProviderID(cfg.Provider.ID) == providerID {
			cfg.Provider = nil
			changed = true
		}
		if cfg.DefaultModel != nil && s.normalizeProviderID(cfg.DefaultModel.ProviderID) == providerID {
			cfg.DefaultModel = nil
			changed = true
		}
		if cfg.AuxiliaryModel != nil && s.normalizeProviderID(cfg.AuxiliaryModel.ProviderID) == providerID {
			cfg.AuxiliaryModel = nil
			changed = true
		}
		nextFallbacks := cfg.FallbackModels[:0]
		for _, model := range cfg.FallbackModels {
			if s.normalizeProviderID(model.ProviderID) == providerID {
				changed = true
				continue
			}
			nextFallbacks = append(nextFallbacks, model)
		}
		cfg.FallbackModels = nextFallbacks
		if changed {
			_ = s.store.SaveConfig(ctx, cfg)
		}
	}
	return s.Catalog(ctx)
}

func providerReadiness(provider domain.ProviderInfo, def ProviderDefinition, authRecord *domain.ProviderAuthRecord) *domain.ProviderReadiness {
	ready := provider.Connected
	source := provider.ConnectionSource
	authMode := ""
	if authRecord != nil {
		authMode = authRecord.Method
		source = authRecord.Method
	}
	if source == "" && provider.Environment != "" {
		source = "env"
	}
	reason := ""
	if !ready {
		switch def.DefaultAuthType {
		case AuthAWSSDK:
			if lookupEnv("AWS_ACCESS_KEY_ID") != "" && lookupEnv("AWS_SECRET_ACCESS_KEY") != "" {
				ready = true
				authMode = "aws-sdk"
				source = "aws-sdk"
			} else {
				reason = "AWS credentials are not configured"
			}
		case AuthNone:
			ready = true
			authMode = "none"
			source = "none"
		default:
			reason = "credentials are not configured"
		}
	}
	return &domain.ProviderReadiness{Ready: ready, AuthMode: authMode, Source: source, Environment: provider.Environment, Reason: reason}
}

func (s *Service) UpdateModelPreferences(ctx context.Context, input domain.ModelPreferencesInput) (domain.AppConfig, error) {
	cfg, err := s.AppConfig(ctx)
	if err != nil {
		return domain.AppConfig{}, err
	}
	if input.Model != nil {
		providerID := strings.TrimSpace(input.Model.ProviderID)
		modelID := strings.TrimSpace(input.Model.ModelID)
		if providerID != "" && modelID != "" {
			modelID = normalizeModelIDForProvider(providerID, modelID)
			cfg.DefaultModel = &domain.ModelRef{ProviderID: providerID, ModelID: modelID}
			provider := s.providerConfigForModelRequest(cfg, providerID, modelID)
			cfg.Provider = &provider
		}
	}
	if input.AuxiliaryModel != nil {
		auxiliaryModel := normalizeOptionalModelRef(*input.AuxiliaryModel)
		cfg.AuxiliaryModel = auxiliaryModel
	}
	if input.FallbackModels != nil {
		cfg.FallbackModels = normalizeFallbackModels(input.FallbackModels, cfg.DefaultModel)
	}
	if input.ProviderPolicy != nil {
		cfg.ProviderPolicy = normalizeProviderRuntimePolicy(*input.ProviderPolicy)
	}
	if strings.TrimSpace(input.ReasoningEffort) != "" {
		cfg.ReasoningEffort = normalizeReasoningEffort(input.ReasoningEffort)
	}
	if strings.TrimSpace(input.ServiceTier) != "" {
		if cfg.DefaultModel != nil && providerSupportsServiceTier(cfg.DefaultModel.ProviderID) {
			cfg.ServiceTier = normalizeServiceTier(input.ServiceTier)
		} else {
			cfg.ServiceTier = "default"
		}
	}
	if input.NativeTools != nil {
		cfg.NativeTools = normalizeNativeToolsRuntimeConfig(*input.NativeTools)
	}
	if cfg.ReasoningEffort == "" {
		cfg.ReasoningEffort = "medium"
	}
	if cfg.ServiceTier == "" {
		cfg.ServiceTier = "default"
	}
	if err := s.store.SaveConfig(ctx, cfg); err != nil {
		return domain.AppConfig{}, err
	}
	return s.AppConfig(ctx)
}

func normalizeOptionalModelRef(model domain.ModelRef) *domain.ModelRef {
	providerID := normalizeProviderID(model.ProviderID)
	modelID := normalizeModelIDForProvider(providerID, strings.TrimSpace(model.ModelID))
	if providerID == "" || modelID == "" {
		return nil
	}
	return &domain.ModelRef{ProviderID: providerID, ModelID: modelID}
}

func normalizeFallbackModels(models []domain.ModelRef, primary *domain.ModelRef) []domain.ModelRef {
	out := make([]domain.ModelRef, 0, len(models))
	seen := map[string]bool{}
	if primary != nil {
		seen[normalizeProviderID(primary.ProviderID)+"\x00"+normalizeModelIDForProvider(primary.ProviderID, primary.ModelID)] = true
	}
	for _, model := range models {
		providerID := normalizeProviderID(model.ProviderID)
		modelID := normalizeModelIDForProvider(providerID, strings.TrimSpace(model.ModelID))
		if providerID == "" || modelID == "" {
			continue
		}
		key := providerID + "\x00" + modelID
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, domain.ModelRef{ProviderID: providerID, ModelID: modelID})
	}
	return out
}

func isOAuthMethod(method string) bool {
	switch method {
	case "oauth", "oauth-browser", "browser", "oauth-headless", "headless":
		return true
	default:
		return false
	}
}

func (s *Service) RefreshProviderModels(ctx context.Context, input domain.ProviderConnectInput) (domain.CatalogState, error) {
	provider, _, err := s.providerConfigFromInput(input)
	if err != nil {
		return domain.CatalogState{}, err
	}
	models, defaultModel, err := s.fetchProviderModels(ctx, provider)
	if err != nil {
		return domain.CatalogState{}, err
	}
	s.rememberRefreshedModels(ctx, provider, input.Name, models, defaultModel)
	return s.Catalog(ctx)
}

func (s *Service) ListProviderCallEvents(ctx context.Context, input domain.ProviderCallEventsInput) ([]domain.ProviderCallEvent, error) {
	providerID := s.normalizeProviderID(input.ProviderID)
	return s.store.ListProviderCallEvents(ctx, providerID, input.Limit)
}

func (s *Service) GetProviderUsage(ctx context.Context, input domain.ProviderUsageInput) (domain.ProviderUsageStats, error) {
	providerID := s.normalizeProviderID(input.ProviderID)
	limit := input.Limit
	if limit <= 0 {
		limit = 1000
	}
	events, err := s.store.ListProviderCallEvents(ctx, providerID, limit)
	if err != nil {
		return domain.ProviderUsageStats{}, err
	}
	stats := domain.ProviderUsageStats{ProviderID: providerID}
	for _, event := range events {
		stats.CallCount++
		stats.InputTokens += event.InputTokens
		stats.OutputTokens += event.OutputTokens
		stats.TotalTokens += event.TotalTokens
		stats.CostMicros += event.CostMicros
		stats.Estimated = stats.Estimated || event.Estimated
		if stats.LastCallAt == "" || event.CreatedAt > stats.LastCallAt {
			stats.LastCallAt = event.CreatedAt
		}
		if event.Status == "success" {
			stats.SuccessCount++
			continue
		}
		stats.FailureCount++
		if event.ErrorClass != "" && (stats.LastErrorClass == "" || event.CreatedAt >= stats.LastCallAt) {
			stats.LastErrorClass = event.ErrorClass
			stats.LastErrorMessage = event.ErrorMessage
			stats.LastErrorProvider = event.ProviderID
		}
	}
	return stats, nil
}

func (s *Service) ValidateProvider(ctx context.Context, input domain.ProviderConnectInput) (domain.ProviderValidationResult, error) {
	provider, def, err := s.providerConfigFromInput(input)
	if err != nil {
		return domain.ProviderValidationResult{}, err
	}
	transport := inferTransport(provider.ID, provider.Type, provider.BaseURL)
	now := domain.NowString(s.now())
	result := domain.ProviderValidationResult{
		ProviderID: provider.ID,
		Status:     "checking",
		Transport:  string(transport),
		BaseURL:    provider.BaseURL,
		CheckedAt:  now,
	}
	credential, err := s.resolveCredentialWithDefinition(ctx, provider, def)
	if err != nil {
		result.Status = "failed"
		result.Error = safeProviderError(err)
		_ = s.store.SaveProviderValidation(ctx, result)
		return result, nil
	}
	result.AuthMode = credential.Method
	result.Source = credential.Method
	result.Environment = provider.APIKeyEnv
	if def.ModelFetch == ModelFetchStatic {
		models := append([]domain.ModelInfo(nil), def.Models...)
		result.Ready = true
		result.Status = "ready"
		result.DefaultModel = firstNonEmpty(provider.Model, def.DefaultModelID)
		result.ModelCount = len(models)
		result.Models = models
		_ = s.store.SaveProviderValidation(ctx, result)
		return result, nil
	}
	models, defaultModel, err := s.fetchProviderModels(ctx, provider)
	if err != nil {
		if cache, cacheErr := s.store.LoadProviderModelCache(ctx, provider.ID); cacheErr == nil && cache != nil && len(cache.Models) > 0 {
			result.Ready = true
			result.Status = "stale-cache"
			result.DefaultModel = cache.DefaultModel
			result.ModelCount = len(cache.Models)
			result.Models = cache.Models
			result.Error = safeProviderError(err)
			_ = s.store.SaveProviderValidation(ctx, result)
			return result, nil
		}
		result.Status = "failed"
		result.Error = safeProviderError(err)
		_ = s.store.SaveProviderValidation(ctx, result)
		return result, nil
	}
	result.Ready = true
	result.Status = "ready"
	result.DefaultModel = defaultModel
	result.ModelCount = len(models)
	result.Models = models
	s.rememberRefreshedModels(ctx, provider, input.Name, models, defaultModel)
	_ = s.store.SaveProviderValidation(ctx, result)
	return result, nil
}

func (s *Service) DeleteProviderAccount(ctx context.Context, accountID string) (domain.CatalogState, error) {
	if strings.TrimSpace(accountID) == "" {
		return domain.CatalogState{}, errors.New("account id is required")
	}
	if auth, err := s.store.GetProviderAuth(ctx, accountID); err == nil {
		_ = s.deleteProviderAuthSecrets(ctx, auth)
	}
	if err := s.store.DeleteProviderAuth(ctx, accountID); err != nil {
		return domain.CatalogState{}, err
	}
	return s.Catalog(ctx)
}

func providerAccountsFromAuth(records []domain.ProviderAuthRecord) []domain.ProviderAccountInfo {
	accounts := make([]domain.ProviderAccountInfo, 0, len(records))
	seen := map[string]bool{}
	for _, record := range records {
		if record.Method == "env" {
			continue
		}
		accountID := strings.TrimSpace(record.AccountID)
		if accountID == "" {
			accountID = accountLabelFromAuth(record)
		}
		key := record.ProviderID + "\x00" + record.Method + "\x00" + accountID
		if seen[key] {
			continue
		}
		seen[key] = true
		id := record.ID
		if id == "" {
			id = record.ProviderID + ":" + record.Method + ":" + accountID
		}
		displayName := strings.TrimSpace(record.DisplayName)
		if displayName == "" && record.ProviderID == "openai" && (record.Method == "oauth-browser" || record.Method == "oauth-headless") {
			displayName = extractOpenAIAccountDisplayName(openAITokenResponse{AccessToken: record.AccessToken})
		}
		if displayName == "" {
			displayName = accountID
		}
		accounts = append(accounts, domain.ProviderAccountInfo{
			ID:          id,
			ProviderID:  record.ProviderID,
			Method:      record.Method,
			AccountID:   accountID,
			DisplayName: displayName,
			ConnectedAt: record.UpdatedAt,
		})
	}
	return accounts
}

func connectAccountLabel(providerID string, method string, apiKey string, env string) string {
	if apiKey != "" {
		if len(apiKey) > 8 {
			return "..." + apiKey[len(apiKey)-6:]
		}
		return "API Key"
	}
	if env != "" {
		return env
	}
	if method == "env" {
		return defaultEnvFor(providerID)
	}
	if method == "oauth-browser" || method == "oauth-headless" {
		return "OpenAI"
	}
	return "默认账号"
}

func accountLabelFromAuth(record domain.ProviderAuthRecord) string {
	if record.APIKey != "" {
		return connectAccountLabel(record.ProviderID, record.Method, record.APIKey, "")
	}
	if record.APIKeyRef != "" {
		return "API Key"
	}
	if record.AccessTokenRef != "" || record.RefreshTokenRef != "" {
		if record.ProviderID == "openai" {
			return "OpenAI"
		}
		return "OAuth account"
	}
	return connectAccountLabel(record.ProviderID, record.Method, "", "")
}

func normalizeHeaders(headers map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range headers {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (s *Service) CompleteInitialization(ctx context.Context, provider *domain.ProviderConfig) (domain.AppConfig, error) {
	cfg, err := s.AppConfig(ctx)
	if err != nil {
		return domain.AppConfig{}, err
	}
	cfg.Initialized = true
	if provider != nil {
		cfg.Provider = provider
		if provider.Model != "" {
			cfg.DefaultModel = &domain.ModelRef{ProviderID: provider.ID, ModelID: provider.Model}
		}
	}
	return cfg, s.store.SaveConfig(ctx, cfg)
}

func (s *Service) SelectProjectDirectory(path string) (string, error) {
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

func (s *Service) SubmitSessionMessage(ctx context.Context, input domain.SubmitSessionMessageRequest) (domain.PreparedSessionTurn, error) {
	return s.submitSessionMessage(ctx, input, nil)
}

func (s *Service) SubmitSessionMessageStreaming(ctx context.Context, input domain.SubmitSessionMessageRequest) (domain.PreparedSessionTurn, error) {
	return s.submitSessionMessage(ctx, input, func(ctx context.Context, prepared domain.PreparedSessionTurn, work func(context.Context) (domain.PreparedSessionTurn, error)) (domain.PreparedSessionTurn, error) {
		go func() {
			turnCtx, cancel := context.WithCancel(context.Background())
			s.registerActiveTurn(prepared.Turn.ID, cancel)
			defer s.unregisterActiveTurn(prepared.Turn.ID)
			defer cancel()
			_, _ = work(turnCtx)
			if s.onSessionUpdated != nil {
				s.onSessionUpdated(prepared.Turn.SessionID, nil)
			}
		}()
		return prepared, nil
	})
}

func (s *Service) submitSessionMessage(
	ctx context.Context,
	input domain.SubmitSessionMessageRequest,
	async func(context.Context, domain.PreparedSessionTurn, func(context.Context) (domain.PreparedSessionTurn, error)) (domain.PreparedSessionTurn, error),
) (domain.PreparedSessionTurn, error) {
	text := strings.TrimSpace(input.Text)
	attachments := sanitizeSessionMessageAttachments(input.Attachments)
	input.Attachments = attachments
	if input.SessionID == "" {
		return domain.PreparedSessionTurn{}, errors.New("sessionId is required")
	}
	if text == "" && len(attachments) == 0 {
		return domain.PreparedSessionTurn{}, errors.New("message text or attachment is required")
	}
	eventText := sessionMessageEventText(text, attachments)
	reasoningEffort := normalizeReasoningEffort(input.ReasoningEffort)
	serviceTier := normalizeServiceTier(input.ServiceTier)
	delivery, err := domain.NormalizeInputDelivery(input.Delivery)
	if err != nil {
		return domain.PreparedSessionTurn{}, err
	}
	if delivery != domain.InputDeliveryImmediate {
		state, _ := s.store.GetSessionExecutionState(ctx, input.SessionID)
		if state.Status == domain.ExecutionStatusRunning || state.Status == domain.ExecutionStatusCompacting {
			pending, err := s.store.CreatePendingSessionInput(ctx, domain.PendingSessionInput{
				SessionID: input.SessionID, TurnID: state.TurnID, Text: eventText, Delivery: delivery, Status: domain.PendingInputStatusPending,
			})
			if err != nil {
				return domain.PreparedSessionTurn{}, err
			}
			state.PendingInputIDs = append(state.PendingInputIDs, pending.ID)
			state.Reason = "input queued for " + delivery + " boundary"
			_, _ = s.store.UpsertSessionExecutionState(ctx, state)
			_, _ = s.AppendEvent(ctx, domain.AppendEventRequest{
				SessionID: input.SessionID, TurnID: state.TurnID, Type: domain.EventTypeSystemNote, Role: domain.EventRoleSystem,
				Visibility: domain.EventVisibilityInternal, Content: "Queued session input",
				Payload: map[string]any{"kind": "pending_input", "pendingInputId": pending.ID, "delivery": delivery},
			})
			return domain.PreparedSessionTurn{
				Turn:      domain.Turn{SessionID: input.SessionID, Status: domain.TurnStatusRunning, TimeCreated: pending.TimeCreated, TimeUpdated: pending.TimeUpdated},
				UserEvent: domain.SessionEvent{SessionID: input.SessionID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser, Visibility: domain.EventVisibilityInternal, Content: eventText, TimeCreated: pending.TimeCreated},
			}, nil
		}
	}
	if input.Model != nil || strings.TrimSpace(input.ReasoningEffort) != "" || strings.TrimSpace(input.ServiceTier) != "" {
		_, _ = s.UpdateModelPreferences(ctx, domain.ModelPreferencesInput{Model: input.Model, ReasoningEffort: reasoningEffort, ServiceTier: serviceTier})
	}
	userEvent, err := s.AppendEvent(ctx, domain.AppendEventRequest{
		SessionID:  input.SessionID,
		Type:       domain.EventTypeUserMessage,
		Role:       domain.EventRoleUser,
		Visibility: domain.EventVisibilityNormal,
		Content:    eventText,
		Payload:    sessionMessageEventPayload(attachments),
	})
	if err != nil {
		return domain.PreparedSessionTurn{}, err
	}
	modeDef, err := s.resolveAgentModeForRequest(ctx, input.SessionID, input.AgentMode)
	if err != nil {
		return domain.PreparedSessionTurn{}, err
	}
	turn, err := s.StartTurn(ctx, domain.StartTurnRequest{SessionID: input.SessionID, UserEventID: userEvent.ID, AgentMode: modeDef.ID})
	if err != nil {
		return domain.PreparedSessionTurn{}, err
	}
	_, _ = s.store.UpsertSessionExecutionState(ctx, domain.SessionExecutionState{SessionID: input.SessionID, TurnID: turn.ID, Status: domain.ExecutionStatusRunning, Reason: "turn running"})
	if s.pluginManager != nil {
		_ = s.pluginManager.InvokeHook(ctx, "on_session_start", map[string]any{"sessionId": input.SessionID, "turnId": turn.ID, "agentMode": modeDef.ID})
	}
	history, err := s.modelVisibleSessionHistory(ctx, input.SessionID)
	if err != nil {
		_, _ = s.FailTurn(ctx, domain.FailTurnRequest{TurnID: turn.ID, Error: err.Error()})
		return domain.PreparedSessionTurn{}, err
	}
	attachCurrentTurnFiles(history, userEvent.ID, eventText, attachments)
	prepared := domain.PreparedSessionTurn{Turn: turn, UserEvent: userEvent}
	work := func(workCtx context.Context) (domain.PreparedSessionTurn, error) {
		return s.completeSessionTurn(workCtx, input, text, history, turn, userEvent, reasoningEffort, serviceTier)
	}
	if async != nil {
		return async(ctx, prepared, work)
	}
	return work(ctx)
}

func (s *Service) completeSessionTurn(
	ctx context.Context,
	input domain.SubmitSessionMessageRequest,
	text string,
	history []domain.ChatMessage,
	turn domain.Turn,
	userEvent domain.SessionEvent,
	reasoningEffort string,
	serviceTier string,
) (domain.PreparedSessionTurn, error) {
	emittedText := ""
	reply, model, err := s.runAssistantAgentLoop(ctx, input, history, turn, reasoningEffort, serviceTier, func(delta string) {
		if s.onAssistantDelta != nil && delta != "" {
			emittedText += delta
			s.onAssistantDelta(input.SessionID, turn.ID, delta)
		}
	})
	if err != nil {
		if isModelExecutionUnavailable(err) {
			reply = deterministicAssistantFallback(text)
			model = input.Model
		} else if isContextCancelled(ctx, err) {
			if cancelled, cancelErr := s.store.UpdateTurnStatus(context.Background(), turn.ID, domain.TurnStatusCancelled, "Turn cancelled"); cancelErr == nil && s.onTurnUpdated != nil {
				s.onTurnUpdated(cancelled.SessionID, cancelled)
			}
			_, _ = s.store.UpsertSessionExecutionState(context.Background(), domain.SessionExecutionState{SessionID: input.SessionID, TurnID: turn.ID, Status: domain.ExecutionStatusInterrupted, Reason: "turn cancelled"})
			return domain.PreparedSessionTurn{}, err
		} else {
			_, _ = s.FailTurn(ctx, domain.FailTurnRequest{TurnID: turn.ID, Error: err.Error()})
			_, _ = s.store.UpsertSessionExecutionState(ctx, domain.SessionExecutionState{SessionID: input.SessionID, TurnID: turn.ID, Status: domain.ExecutionStatusFailed, Reason: err.Error()})
			return domain.PreparedSessionTurn{}, err
		}
	}
	if reply == "" {
		replyErr := errors.New("assistant reply is empty")
		_, _ = s.FailTurn(ctx, domain.FailTurnRequest{TurnID: turn.ID, Error: replyErr.Error()})
		_, _ = s.store.UpsertSessionExecutionState(ctx, domain.SessionExecutionState{SessionID: input.SessionID, TurnID: turn.ID, Status: domain.ExecutionStatusFailed, Reason: replyErr.Error()})
		return domain.PreparedSessionTurn{}, replyErr
	}
	if !strings.Contains(emittedText, reply) && s.onAssistantDelta != nil {
		s.onAssistantDelta(input.SessionID, turn.ID, reply)
	}
	assistantEvent, err := s.AppendEvent(ctx, domain.AppendEventRequest{SessionID: input.SessionID, TurnID: turn.ID, Type: domain.EventTypeAssistantMessage, Role: domain.EventRoleAssistant, Visibility: domain.EventVisibilityNormal, Content: reply})
	if err != nil {
		_, _ = s.FailTurn(ctx, domain.FailTurnRequest{TurnID: turn.ID, Error: err.Error()})
		_, _ = s.store.UpsertSessionExecutionState(ctx, domain.SessionExecutionState{SessionID: input.SessionID, TurnID: turn.ID, Status: domain.ExecutionStatusFailed, Reason: err.Error()})
		return domain.PreparedSessionTurn{}, err
	}
	turn, err = s.CompleteTurn(ctx, domain.CompleteTurnRequest{TurnID: turn.ID})
	if err != nil {
		return domain.PreparedSessionTurn{}, err
	}
	_, _ = s.promotePendingInputsAtBoundary(ctx, input.SessionID, turn.ID, domain.InputDeliverySteer)
	_, _ = s.store.UpsertSessionExecutionState(ctx, domain.SessionExecutionState{SessionID: input.SessionID, TurnID: turn.ID, Status: domain.ExecutionStatusIdle, Reason: "turn complete"})
	if s.onSessionUpdated != nil {
		s.onSessionUpdated(input.SessionID, nil)
	}
	go s.ensureGeneratedSessionTitle(context.Background(), input.SessionID, model)
	return domain.PreparedSessionTurn{Turn: turn, Model: model, UserEvent: userEvent, AssistantEvent: &assistantEvent}, nil
}

func (s *Service) promotePendingInputsAtBoundary(ctx context.Context, sessionID string, turnID string, delivery string) ([]domain.PendingSessionInput, error) {
	items, err := s.store.ListPendingSessionInputs(ctx, sessionID, domain.PendingInputStatusPending)
	if err != nil {
		return nil, err
	}
	promoted := []domain.PendingSessionInput{}
	for _, item := range items {
		if item.Delivery != delivery {
			continue
		}
		updated, err := s.store.UpdatePendingSessionInputStatus(ctx, item.ID, domain.PendingInputStatusPromoted, turnID)
		if err != nil {
			return promoted, err
		}
		promoted = append(promoted, updated)
		_, _ = s.AppendEvent(ctx, domain.AppendEventRequest{
			SessionID: sessionID, TurnID: turnID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser,
			Visibility: domain.EventVisibilityNormal, Content: item.Text,
			Payload: map[string]any{"kind": "promoted_input", "delivery": item.Delivery, "pendingInputId": item.ID},
		})
	}
	return promoted, nil
}

func sanitizeSessionMessageAttachments(attachments []domain.MessageAttachment) []domain.MessageAttachment {
	out := make([]domain.MessageAttachment, 0, len(attachments))
	for _, attachment := range attachments {
		data := strings.TrimSpace(attachment.Data)
		text := strings.TrimSpace(attachment.Text)
		if data == "" && text == "" {
			continue
		}
		name := strings.TrimSpace(attachment.Name)
		if name == "" {
			name = "attachment"
		}
		mimeType := strings.TrimSpace(attachment.MIMEType)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		kind := strings.TrimSpace(attachment.Kind)
		if kind == "" {
			if strings.HasPrefix(mimeType, "image/") {
				kind = "image"
			} else {
				kind = "file"
			}
		}
		out = append(out, domain.MessageAttachment{
			ID: attachment.ID, Name: name, MIMEType: mimeType, Kind: kind,
			Data: data, Text: text, Size: attachment.Size,
		})
	}
	return out
}

func sessionMessageEventText(text string, attachments []domain.MessageAttachment) string {
	text = strings.TrimSpace(text)
	if len(attachments) == 0 {
		return text
	}
	lines := make([]string, 0, len(attachments))
	for _, attachment := range attachments {
		detail := strings.TrimSpace(attachment.MIMEType)
		if detail == "" {
			detail = attachment.Kind
		}
		if attachment.Size > 0 {
			detail = fmt.Sprintf("%s, %d bytes", detail, attachment.Size)
		}
		lines = append(lines, fmt.Sprintf("- %s (%s)", attachment.Name, detail))
	}
	summary := "附件:\n" + strings.Join(lines, "\n")
	if text == "" {
		return summary
	}
	return text + "\n\n" + summary
}

func sessionMessageEventPayload(attachments []domain.MessageAttachment) map[string]any {
	if len(attachments) == 0 {
		return nil
	}
	items := make([]map[string]any, 0, len(attachments))
	for _, attachment := range attachments {
		item := map[string]any{
			"id":       strings.TrimSpace(attachment.ID),
			"name":     strings.TrimSpace(attachment.Name),
			"mimeType": strings.TrimSpace(attachment.MIMEType),
			"kind":     strings.TrimSpace(attachment.Kind),
			"size":     attachment.Size,
		}
		if strings.EqualFold(strings.TrimSpace(attachment.Kind), "image") {
			item["data"] = attachment.Data
		}
		items = append(items, item)
	}
	return map[string]any{
		"kind":        "user_message",
		"attachments": items,
	}
}

func attachCurrentTurnFiles(messages []domain.ChatMessage, userEventID string, eventText string, attachments []domain.MessageAttachment) {
	if len(attachments) == 0 || len(messages) == 0 {
		return
	}
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != domain.EventRoleUser {
			continue
		}
		if strings.TrimSpace(messages[i].Text) == strings.TrimSpace(eventText) {
			messages[i].Attachments = attachments
			return
		}
	}
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == domain.EventRoleUser {
			messages[i].Attachments = attachments
			return
		}
	}
}

func isContextCancelled(ctx context.Context, err error) bool {
	if ctx != nil && errors.Is(ctx.Err(), context.Canceled) {
		return true
	}
	return errors.Is(err, context.Canceled)
}

func (s *Service) runAssistantAgentLoop(
	ctx context.Context,
	input domain.SubmitSessionMessageRequest,
	history []domain.ChatMessage,
	turn domain.Turn,
	reasoningEffort string,
	serviceTier string,
	onDelta func(string),
) (string, *domain.ModelRef, error) {
	cc, _ := s.store.GetCodingContext(ctx, input.SessionID)
	modeDef, err := s.resolveAgentModeForRequest(ctx, input.SessionID, firstNonEmpty(input.AgentMode, turn.AgentMode))
	if err != nil {
		return "", nil, err
	}
	if session, err := s.store.GetRuntimeSession(ctx, input.SessionID); err == nil && session.Type == domain.SessionTypeCoding {
		if strings.TrimSpace(cc.ProjectPath) == "" && strings.TrimSpace(session.ProjectPath) != "" {
			cc, _ = s.CreateOrUpdateCodingContext(ctx, input.SessionID, session.ProjectPath)
		} else if projectPath, changed, err := ensureManagedWorkspace(cc.ProjectPath); err == nil && changed {
			cc, _ = s.CreateOrUpdateCodingContext(ctx, input.SessionID, projectPath)
		}
	}
	registry, runtime := s.toolsForWorkspace(strings.TrimSpace(cc.ProjectPath))
	allowedToolsets := allowedToolsetsForRun(modeDef, input)
	requestedModel := s.modelForAgentMode(ctx, modeDef, input.Model)
	messages := append([]domain.ChatMessage(nil), history...)
	modePrompt := "Agent mode: " + modeDef.DisplayName + "\n\n" + modeDef.Prompt
	modePrompt += "\n\nIf current tools cannot perform a required action, call tool_resolve with a concise, specific missing capability. Do not use it for convenience, exploration, planning, or guessing tool names. If no allowed tool matches, stop with a local no_available_tool error."
	if len(messages) > 0 && messages[0].Role == domain.EventRoleSystem {
		messages[0].Text = messages[0].Text + "\n\n" + modePrompt
	} else {
		messages = append([]domain.ChatMessage{{Role: domain.EventRoleSystem, Text: modePrompt}}, messages...)
	}
	var model *domain.ModelRef
	for step := 0; step < defaultAgentMaxSteps; step++ {
		var specs []domain.ToolSpec
		expectedRegistrations := map[string]domain.ToolRegistrationIdentity{}
		if registry != nil {
			specs = visibleToolSpecsForMode(modeDef.ID, registry.SpecsForToolsets(allowedToolsets))
			assembly := AssembleToolSpecsWithActivated(registry, specs, s.rememberedDeferredTools(ctx, input.SessionID))
			specs = assembly.Specs
			expectedRegistrations = assembly.ExpectedRegistrations
		}
		if s.pluginManager != nil {
			_ = s.pluginManager.InvokeHook(ctx, "pre_llm_call", map[string]any{"sessionId": input.SessionID, "turnId": turn.ID, "toolCount": len(specs), "messageCount": len(messages), "agentMode": modeDef.ID})
		}
		resp, activeModel, err := s.GenerateChatResponseStreamWithToolDelta(ctx, domain.ChatRequest{Messages: messages, Tools: specs}, requestedModel, reasoningEffort, serviceTier, onDelta, func(call domain.ChatToolCall) {
			s.emitApplyPatchDraft(input.SessionID, turn.ID, strings.TrimSpace(cc.ProjectPath), call)
		})
		if s.pluginManager != nil {
			_ = s.pluginManager.InvokeHook(ctx, "post_llm_call", map[string]any{"sessionId": input.SessionID, "turnId": turn.ID, "toolCallCount": len(resp.ToolCalls), "textLength": len(resp.Text), "agentMode": modeDef.ID})
		}
		if err != nil {
			return "", activeModel, err
		}
		model = activeModel
		if len(resp.ToolCalls) == 0 {
			return resp.Text, model, nil
		}
		logToolCalls(resp.ToolCalls)
		if strings.TrimSpace(resp.Text) != "" {
			_, _ = s.AppendEvent(ctx, domain.AppendEventRequest{
				SessionID:  input.SessionID,
				TurnID:     turn.ID,
				Type:       domain.EventTypeAssistantMessage,
				Role:       domain.EventRoleAssistant,
				Visibility: domain.EventVisibilityNormal,
				Content:    resp.Text,
				Payload:    map[string]any{"phase": "before_tool"},
			})
		}
		messages = append(messages, domain.ChatMessage{Role: "assistant", Text: resp.Text, ToolCalls: resp.ToolCalls})
		_, _ = s.AppendEvent(ctx, domain.AppendEventRequest{
			SessionID:  input.SessionID,
			TurnID:     turn.ID,
			Type:       domain.EventTypeToolCall,
			Role:       domain.EventRoleAssistant,
			Visibility: domain.EventVisibilityInternal,
			Content:    bounded(resp.Text, 1000),
			Payload:    map[string]any{"toolCalls": toolCallsPayload(resp.ToolCalls)},
		})
		for _, call := range resp.ToolCalls {
			_ = s.recordToolCallStarted(ctx, input.SessionID, turn.ID, call)
			var result domain.ToolResult
			if runtime == nil {
				result = domain.ToolResult{CallID: call.ID, Name: call.Name, OK: false, Error: "tool runtime unavailable: this session has no workspace root"}
			} else {
				result = runtime.ExecuteWithContext(ctx, call, domain.ToolExecutionContext{
					WorkspaceRoot:         strings.TrimSpace(cc.ProjectPath),
					SessionID:             input.SessionID,
					TurnID:                turn.ID,
					AgentMode:             modeDef.ID,
					AllowedToolsets:       allowedToolsets,
					PermissionScope:       firstNonEmpty(input.PermissionScope, defaultPermissionScopeForMode(modeDef.ID)),
					ExpectedRegistrations: expectedRegistrations,
				})
			}
			_ = s.recordToolResult(ctx, input.SessionID, turn.ID, call, result)
			if result.PermissionRequested {
				return "等待你批准工具权限后，我可以继续执行这次修改。", model, nil
			}
			if call.Name == ToolResolveName && result.ToolError != nil && result.ToolError.Code == "no_available_tool" {
				return "", model, errors.New(result.ToolError.Message)
			}
			messages = append(messages, domain.ChatMessage{Role: "tool", Text: encodeToolResultForModel(result), ToolCallID: call.ID, Name: call.Name})
		}
	}
	return "", model, ErrMaxStepsExceeded
}

func (s *Service) resolveToolsWithAuxiliaryModel(ctx context.Context, request ToolResolveRequest) (ToolResolveDecision, error) {
	if len(request.Candidates) == 0 {
		return ToolResolveDecision{}, nil
	}
	maxTools := request.MaxTools
	if maxTools <= 0 {
		maxTools = 8
	}
	catalog := make([]map[string]any, 0, len(request.Candidates))
	for _, entry := range request.Candidates {
		catalog = append(catalog, map[string]any{
			"name":        entry.Name,
			"description": bounded(entry.Description, 400),
			"source":      entry.Source,
			"sourceId":    entry.SourceID,
			"category":    entry.Category,
			"namespace":   entry.Namespace,
			"capability":  entry.Capability,
			"riskLevel":   entry.RiskLevel,
		})
	}
	payload := map[string]any{
		"intent":    request.Intent,
		"maxTools":  maxTools,
		"agentMode": request.AgentMode,
		"catalog":   catalog,
	}
	rawPayload, _ := json.MarshalIndent(payload, "", "  ")
	messages := []domain.ChatMessage{
		{Role: "system", Text: "Select tools only from the provided catalog for the requested missing capability. Return strict JSON: {\"tools\":[\"exact_tool_name\"],\"reason\":\"short reason\"}. Select only clear matches. Do not invent names, infer hidden tools, or choose adjacent tools. If uncertain or no clear match exists, return {\"tools\":[],\"reason\":\"no matching allowed tool\"}."},
		{Role: "user", Text: string(rawPayload)},
	}
	models := s.resolveAuxiliaryModels(ctx, nil)
	var lastErr error
	for _, model := range models {
		reply, _, err := s.GenerateChatReply(ctx, messages, &model, "low", "default")
		if err != nil {
			lastErr = err
			continue
		}
		decision, err := parseToolResolveDecision(reply)
		if err != nil {
			lastErr = err
			continue
		}
		if len(decision.Names) > maxTools {
			decision.Names = decision.Names[:maxTools]
		}
		return decision, nil
	}
	if lastErr != nil {
		return ToolResolveDecision{}, lastErr
	}
	return ToolResolveDecision{}, errors.New("auxiliary model is not configured")
}

func parseToolResolveDecision(raw string) (ToolResolveDecision, error) {
	text := strings.TrimSpace(stripThinkBlocks(raw))
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end >= start {
		text = text[start : end+1]
	}
	var decoded struct {
		Tools  []string `json:"tools"`
		Names  []string `json:"names"`
		Reason string   `json:"reason"`
	}
	if err := json.Unmarshal([]byte(text), &decoded); err != nil {
		return ToolResolveDecision{}, err
	}
	names := decoded.Tools
	if len(names) == 0 {
		names = decoded.Names
	}
	out := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" {
			out = append(out, name)
		}
	}
	return ToolResolveDecision{Names: out, Reason: strings.TrimSpace(decoded.Reason)}, nil
}

func (s *Service) rememberedDeferredTools(ctx context.Context, sessionID string) map[string]bool {
	remembered := map[string]bool{}
	if s == nil || s.store == nil || strings.TrimSpace(sessionID) == "" {
		return remembered
	}
	state, err := s.store.GetSessionExecutionState(ctx, sessionID)
	if err != nil || state.Metadata == nil {
		return remembered
	}
	for _, name := range stringSliceFromAny(state.Metadata[sessionMetadataRememberedDeferredTools]) {
		name = strings.TrimSpace(name)
		if name != "" && !isBridgeToolName(name) {
			remembered[name] = true
		}
	}
	return remembered
}

func (s *Service) GetSessionActiveTools(ctx context.Context, sessionID string) (domain.SessionActiveToolsResult, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return domain.SessionActiveToolsResult{}, errors.New("sessionId is required")
	}
	remembered := s.rememberedDeferredTools(ctx, sessionID)
	names := make([]string, 0, len(remembered))
	for name := range remembered {
		names = append(names, name)
	}
	sort.Strings(names)
	return domain.SessionActiveToolsResult{SessionID: sessionID, ToolNames: names}, nil
}

func (s *Service) SetSessionActiveTools(ctx context.Context, input domain.SessionActiveToolsInput) (domain.SessionActiveToolsResult, error) {
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return domain.SessionActiveToolsResult{}, errors.New("sessionId is required")
	}
	state, err := s.store.GetSessionExecutionState(ctx, sessionID)
	if err != nil {
		return domain.SessionActiveToolsResult{}, err
	}
	names := normalizeDeferredToolNames(input.ToolNames)
	if state.Metadata == nil {
		state.Metadata = map[string]any{}
	}
	state.Metadata[sessionMetadataRememberedDeferredTools] = names
	if _, err := s.store.UpsertSessionExecutionState(ctx, state); err != nil {
		return domain.SessionActiveToolsResult{}, err
	}
	if s.onSessionUpdated != nil {
		s.onSessionUpdated(sessionID, nil)
	}
	return domain.SessionActiveToolsResult{SessionID: sessionID, ToolNames: names}, nil
}

func (s *Service) rememberDeferredToolUsed(ctx context.Context, sessionID string, toolName string) error {
	toolName = strings.TrimSpace(toolName)
	if s == nil || s.store == nil || strings.TrimSpace(sessionID) == "" || toolName == "" || isBridgeToolName(toolName) {
		return nil
	}
	state, err := s.store.GetSessionExecutionState(ctx, sessionID)
	if err != nil {
		return err
	}
	remembered := map[string]bool{}
	for _, name := range stringSliceFromAny(state.Metadata[sessionMetadataRememberedDeferredTools]) {
		name = strings.TrimSpace(name)
		if name != "" && !isBridgeToolName(name) {
			remembered[name] = true
		}
	}
	if remembered[toolName] {
		return nil
	}
	remembered[toolName] = true
	names := make([]string, 0, len(remembered))
	for name := range remembered {
		names = append(names, name)
	}
	sort.Strings(names)
	if state.Metadata == nil {
		state.Metadata = map[string]any{}
	}
	state.Metadata[sessionMetadataRememberedDeferredTools] = names
	_, err = s.store.UpsertSessionExecutionState(ctx, state)
	return err
}

func normalizeDeferredToolNames(toolNames []string) []string {
	seen := map[string]bool{}
	names := make([]string, 0, len(toolNames))
	for _, name := range toolNames {
		name = strings.TrimSpace(name)
		if name == "" || isBridgeToolName(name) || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (s *Service) emitApplyPatchDraft(sessionID string, turnID string, workspaceRoot string, call domain.ChatToolCall) {
	if s.onToolCallUpdated == nil || call.Name != "apply_patch" || strings.TrimSpace(call.ID) == "" {
		return
	}
	patchText, files := applyPatchDraftFiles(call.Arguments, workspaceRoot)
	if strings.TrimSpace(patchText) == "" && len(files) == 0 {
		return
	}
	now := domain.NowString(s.now())
	result := map[string]any{"draft": true}
	if len(files) > 0 {
		result["files"] = files
	}
	if strings.TrimSpace(patchText) != "" {
		result["patchTextPreview"] = bounded(patchText, 4000)
	}
	s.onToolCallUpdated(sessionID, turnID, domain.ToolCall{
		ID:          call.ID,
		SessionID:   sessionID,
		TurnID:      turnID,
		Name:        call.Name,
		Status:      domain.ToolCallStatusRunning,
		Result:      result,
		TimeCreated: now,
		TimeUpdated: now,
	}, false)
}

func (s *Service) recordToolCallStarted(ctx context.Context, sessionID string, turnID string, call domain.ChatToolCall) error {
	args := toolCallArgumentsMap(call)
	_, err := s.SaveToolCall(ctx, domain.CreateToolCallRequest{
		ID:        call.ID,
		SessionID: sessionID,
		TurnID:    turnID,
		Name:      call.Name,
		Arguments: args,
		Status:    domain.ToolCallStatusRunning,
	})
	if err == nil && s.onSessionUpdated != nil {
		s.onSessionUpdated(sessionID, nil)
	}
	return err
}

func (s *Service) sessionChatHistory(ctx context.Context, sessionID string, limit int) ([]domain.ChatMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	events, err := s.store.ListSessionEvents(ctx, sessionID, false, 500)
	if err != nil {
		return nil, err
	}
	messages := make([]domain.ChatMessage, 0, len(events))
	for _, event := range events {
		if event.Type != domain.EventTypeUserMessage && event.Type != domain.EventTypeAssistantMessage {
			continue
		}
		role := event.Role
		if role == "" {
			if event.Type == domain.EventTypeUserMessage {
				role = domain.EventRoleUser
			} else {
				role = domain.EventRoleAssistant
			}
		}
		messages = append(messages, domain.ChatMessage{Role: role, Text: event.Content})
	}
	if len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}
	return messages, nil
}

func (s *Service) toolsForWorkspace(workspaceRoot string) (*Registry, *ToolRuntime) {
	if workspaceRoot == "" {
		return nil, nil
	}
	registry, err := NewCodingToolRegistryWithShellOutputSink(workspaceRoot, func(event ShellOutputEvent) {
		if s.onShellOutput != nil {
			s.onShellOutput(event)
		}
	})
	if err != nil {
		return nil, nil
	}
	if bash, ok := registry.Get("bash"); ok {
		if bashTool, ok := bash.(*BashTool); ok {
			bashTool.SetPersistentCWDHooks(s.loadAgentShellCWD, s.saveAgentShellCWD)
		}
	}
	for _, tool := range newAgentRuntimeTools(s) {
		_ = registry.Register(tool)
	}
	if s.pluginManager == nil {
		s.pluginManager = NewPluginManager(s.store)
	}
	if s.mcpManager == nil {
		s.mcpManager = NewMCPManager(s.store, s.secrets)
	}
	s.pluginManager.RegisterEnabledTools(context.Background(), registry)
	s.mcpManager.RegisterEnabledTools(context.Background(), registry)
	_ = registry.RegisterScoped(NewToolResolveTool(registry, s.resolveToolsWithAuxiliaryModel, s.rememberDeferredToolUsed), domain.ToolSourceBridge, "tool_discovery", "")
	runtime := NewToolRuntime(registry, workspaceRoot)
	runtime.PluginHooks = s.pluginManager
	runtime.Permissions = NewPermissionEngine(s.store)
	runtime.Permissions.notifier = s.permissionNotifier
	runtime.Permissions.onRequest = func(request domain.PermissionRequest) {
		if request.SessionID != "" && request.ToolCallID != "" {
			_, _ = s.SaveToolCall(context.Background(), domain.CreateToolCallRequest{
				ID:            request.ToolCallID,
				SessionID:     request.SessionID,
				TurnID:        request.TurnID,
				Name:          request.ToolName,
				Arguments:     request.Arguments,
				Status:        domain.ToolCallStatusPending,
				ResultSummary: "Waiting for permission approval",
				Result: map[string]any{
					"ok":                 false,
					"call_id":            request.ToolCallID,
					"name":               request.ToolName,
					"pendingApprovalId":  request.ID,
					"permissionDecision": domain.PermissionDecisionAsk,
				},
				Error: "permission approval is required",
			})
		}
		if s.onPermissionRequested != nil {
			s.onPermissionRequested(request)
		}
		if s.onSessionUpdated != nil && request.SessionID != "" {
			s.onSessionUpdated(request.SessionID, nil)
		}
	}
	return registry, runtime
}

func (s *Service) loadAgentShellCWD(sessionID string, workspaceRoot string) string {
	if s == nil || s.store == nil {
		return ""
	}
	cc, err := s.store.GetCodingContext(context.Background(), strings.TrimSpace(sessionID))
	if err != nil {
		return ""
	}
	cwd := workspaceInternalCWD(workspaceRoot, cc.CWD)
	if cwd != "" {
		return cwd
	}
	if strings.TrimSpace(cc.CWD) != "" {
		s.saveAgentShellCWD(sessionID, workspaceRoot, workspaceRoot)
	}
	return ""
}

func (s *Service) saveAgentShellCWD(sessionID string, workspaceRoot string, cwd string) {
	if s == nil || s.store == nil || strings.TrimSpace(sessionID) == "" {
		return
	}
	normalized := workspaceInternalCWD(workspaceRoot, cwd)
	if normalized == "" {
		normalized = workspaceInternalCWD(workspaceRoot, workspaceRoot)
	}
	if normalized == "" {
		return
	}
	ctx := context.Background()
	cc, err := s.store.GetCodingContext(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		cc = domain.CodingContext{
			SessionID:   strings.TrimSpace(sessionID),
			ProjectPath: workspaceInternalCWD(workspaceRoot, workspaceRoot),
			Permissions: []string{"local-filesystem"},
		}
	}
	if strings.TrimSpace(cc.ProjectPath) == "" {
		cc.ProjectPath = workspaceInternalCWD(workspaceRoot, workspaceRoot)
	}
	cc.CWD = normalized
	_, _ = s.store.UpsertCodingContext(ctx, cc)
}

func workspaceInternalCWD(workspaceRoot string, cwd string) string {
	if strings.TrimSpace(workspaceRoot) == "" {
		return ""
	}
	_, normalized, err := normalizeSandboxCWD(workspaceRoot, cwd, false)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(normalized)
}

func logToolCalls(calls []domain.ChatToolCall) {
	names := make([]string, 0, len(calls))
	for _, call := range calls {
		names = append(names, call.Name)
	}
	log.Printf("model returned tool_calls count=%d names=%s", len(calls), strings.Join(names, ","))
}

func (s *Service) recordToolResult(ctx context.Context, sessionID string, turnID string, call domain.ChatToolCall, result domain.ToolResult) error {
	return s.recordToolResultWithMetadata(ctx, sessionID, turnID, call, result, nil)
}

func (s *Service) recordToolResultWithMetadata(ctx context.Context, sessionID string, turnID string, call domain.ChatToolCall, result domain.ToolResult, metadata map[string]any) error {
	args := toolCallArgumentsMap(call)
	status := domain.ToolCallStatusSuccess
	if !result.OK {
		status = domain.ToolCallStatusFailed
	}
	if result.PermissionRequested {
		status = domain.ToolCallStatusPending
	}
	resultMap := map[string]any{"ok": result.OK, "call_id": result.CallID, "name": result.Name}
	if result.Content != "" {
		resultMap["content"] = bounded(result.Content, 2000)
	}
	if result.ModelContent != "" {
		resultMap["modelContent"] = bounded(result.ModelContent, 2000)
	}
	if result.Structured != nil {
		resultMap["structured"] = result.Structured
	}
	if len(result.RetainedOutputRefs) > 0 {
		resultMap["retainedOutputRefs"] = result.RetainedOutputRefs
	}
	if len(result.Files) > 0 {
		resultMap["files"] = result.Files
	}
	if result.Error != "" {
		resultMap["error"] = result.Error
	}
	if result.PendingApprovalID != "" {
		resultMap["pendingApprovalId"] = result.PendingApprovalID
	}
	if result.PermissionDecision != "" {
		resultMap["permissionDecision"] = result.PermissionDecision
	}
	for key, value := range metadata {
		if strings.TrimSpace(key) != "" && value != nil {
			resultMap[key] = value
		}
	}
	_, err := s.SaveToolCall(ctx, domain.CreateToolCallRequest{
		ID:            call.ID,
		SessionID:     sessionID,
		TurnID:        turnID,
		Name:          call.Name,
		Arguments:     args,
		Status:        status,
		ResultSummary: toolResultSummary(result),
		Result:        resultMap,
		Error:         result.Error,
	})
	if appendErr := s.appendToolResultEvent(ctx, sessionID, turnID, result); appendErr != nil && err == nil {
		err = appendErr
	}
	return err
}

func (s *Service) appendToolResultEvent(ctx context.Context, sessionID string, turnID string, result domain.ToolResult) error {
	_, err := s.AppendEvent(ctx, domain.AppendEventRequest{
		SessionID:  sessionID,
		TurnID:     turnID,
		Type:       domain.EventTypeToolResult,
		Role:       domain.EventRoleTool,
		Visibility: domain.EventVisibilityInternal,
		Content:    toolResultSummary(result),
		Payload:    map[string]any{"callId": result.CallID, "name": result.Name, "ok": result.OK},
	})
	return err
}

func encodeToolResultForModel(result domain.ToolResult) string {
	if strings.TrimSpace(result.ModelContent) != "" {
		modelResult := result
		modelResult.Content = result.ModelContent
		raw, err := json.Marshal(modelResult)
		if err == nil {
			return string(raw)
		}
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return result.Error
	}
	return string(raw)
}

func toolCallArgumentsMap(call domain.ChatToolCall) map[string]any {
	args := map[string]any{}
	if len(call.Arguments) == 0 {
		return args
	}
	if err := json.Unmarshal(call.Arguments, &args); err == nil {
		return args
	}
	if call.Name == "apply_patch" {
		args["patchText"] = string(call.Arguments)
		args["freeform"] = true
	}
	return args
}

func toolResultSummary(result domain.ToolResult) string {
	if result.OK {
		return bounded(result.Content, 500)
	}
	return bounded(result.Error, 500)
}

func toolCallsPayload(calls []domain.ChatToolCall) []map[string]any {
	out := make([]map[string]any, 0, len(calls))
	for _, call := range calls {
		out = append(out, map[string]any{"id": call.ID, "name": call.Name, "arguments": string(call.Arguments)})
	}
	return out
}

func isModelExecutionUnavailable(err error) bool {
	if err == nil {
		return false
	}
	text := err.Error()
	return strings.Contains(text, "provider is not configured") ||
		strings.Contains(text, "model is not configured") ||
		strings.Contains(text, "credentials are not configured") ||
		strings.Contains(text, "credentials are missing") ||
		strings.Contains(text, "base URL is not configured")
}

func deterministicAssistantFallback(userText string) string {
	text := strings.TrimSpace(userText)
	if text == "" {
		return "I recorded your request. Configure a model provider to generate a full assistant response."
	}
	if len(text) > 120 {
		text = text[:120]
	}
	return "I recorded your request and saved it to this session. Configure a model provider to continue with an AI-generated response. Request: " + text
}

func providerAuthMethods(id string, env string) []domain.ProviderAuthMethod {
	methods := []domain.ProviderAuthMethod{{ID: "env", Label: "Credential reference", Stable: true, Available: true, Description: env}}
	if id == "openai" {
		methods = append([]domain.ProviderAuthMethod{
			{
				ID:          "oauth-browser",
				Label:       "ChatGPT Pro/Plus (browser)",
				Stable:      false,
				Available:   true,
				Description: "OpenAI browser OAuth with PKCE and localhost callback",
			},
			{
				ID:          "oauth-headless",
				Label:       "ChatGPT Pro/Plus (headless)",
				Stable:      false,
				Available:   true,
				Description: "OpenAI device authorization flow",
			},
			{
				ID:        "api-key",
				Label:     "API Key",
				Stable:    true,
				Available: true,
			},
		}, methods...)
	}
	if id != "openai" && env != "" {
		methods = append(methods, domain.ProviderAuthMethod{ID: "api-key", Label: "API Key", Stable: true, Available: true})
	}
	if id == "custom-api" {
		methods = append(methods, domain.ProviderAuthMethod{ID: "api-key", Label: "API Key", Stable: true, Available: true})
	}
	return methods
}

func flattenModels(providers []domain.ProviderInfo) []domain.ModelInfo {
	var out []domain.ModelInfo
	for _, provider := range providers {
		out = append(out, provider.Models...)
	}
	return out
}
