package app

import (
	"context"
	"sync"
	"time"

	"aivo/core/domain"
)

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
	ptyManager            *AgentPTYRegistry
	titleGenerator        func(context.Context, string, *domain.ModelRef) (string, error)
	secrets               SecretStore
	providers             *ProviderRegistry
	providersMu           sync.RWMutex
	providerContributions map[string]ProviderDefinition
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
	skillManager          *SkillManager
}

func NewService(store Store) *Service {
	service := &Service{
		store:                 store,
		now:                   time.Now,
		secrets:               NewDefaultSecretStore(),
		providers:             NewDefaultProviderRegistry(),
		providerContributions: map[string]ProviderDefinition{},
		agentCatalog:          NewAgentCatalog(),
		rateLimiter:           newProviderRateLimiter(),
		permissionNotifier:    newPermissionNotifier(),
		questionNotifier:      newPermissionNotifier(),
		terminals:             NewTerminalService(),
		ptyManager:            defaultAgentPTYRegistry,
		refreshedModels:       map[string][]domain.ModelInfo{},
		refreshedDefault:      map[string]string{},
		refreshedInfo:         map[string]domain.ProviderInfo{},
		activeAgentRunCancel:  map[string]context.CancelFunc{},
		activeTurnCancel:      map[string]context.CancelFunc{},
	}
	service.pluginManager = NewPluginManager(store)
	service.mcpManager = NewMCPManager(store, service.secrets)
	service.skillManager = NewSkillManager(store)
	service.refreshProviderExtensions("")
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
	s.providersMu.Lock()
	defer s.providersMu.Unlock()
	if s.providers == nil {
		s.providers = NewDefaultProviderRegistry()
	}
	if err := s.providers.RegisterDefinition(def); err != nil {
		return err
	}
	if s.providerContributions == nil {
		s.providerContributions = map[string]ProviderDefinition{}
	}
	s.providerContributions[normalizeProviderKey(def.ID)] = def
	return nil
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
	defaultAgentPTYRegistry.Shutdown()
}
