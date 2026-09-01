package app

import (
	"context"
	"log"
	"sync"
	"time"

	"aivo/core/domain"
)

type Service struct {
	store                         Store
	now                           func() time.Time
	authFlows                     *ProviderAuthManager
	onAuthSuccess                 func()
	onProviderAuthUpdated         func(domain.ProviderAuthStatus)
	onSessionUpdated              func(string, *domain.Session)
	onSessionEventUpdated         func(domain.SessionEvent, bool)
	onTurnUpdated                 func(string, domain.Turn)
	onAssistantDelta              func(sessionID string, turnID string, delta string)
	onToolCallUpdated             func(string, string, domain.ToolCall, bool)
	onShellOutput                 func(ShellOutputEvent)
	onTodoItemsUpdated            func(sessionID string, projectPath string, items []domain.TodoItem)
	onPermissionRequested         func(domain.PermissionRequest)
	onPermissionResolved          func(domain.PermissionRequest)
	onQuestionRequested           func(domain.QuestionRequest)
	onQuestionResolved            func(domain.QuestionRequest)
	onTerminalEvent               func(string, TerminalInfo)
	permissionNotifier            *permissionNotifier
	questionNotifier              *permissionNotifier
	terminals                     *DefaultTerminalService
	ptyManager                    *AgentPTYRegistry
	titleGenerator                func(context.Context, string, *domain.ModelRef) (string, error)
	secrets                       SecretStore
	providers                     *ProviderRegistry
	providersMu                   sync.RWMutex
	providerContributions         map[string]ProviderDefinition
	agentCatalog                  *AgentCatalog
	rateLimiter                   *providerRateLimiter
	modelRefreshMu                sync.Mutex
	refreshedModels               map[string][]domain.ModelInfo
	refreshedDefault              map[string]string
	refreshedInfo                 map[string]domain.ProviderInfo
	providerCapabilitySyncMu      sync.Mutex
	providerCapabilitySynced      map[string]bool
	schedulerCancel               context.CancelFunc
	activeAgentRunMu              sync.Mutex
	activeAgentRunCancel          map[string]context.CancelFunc
	activeTurnMu                  sync.Mutex
	activeTurnCancel              map[string]context.CancelFunc
	mcpManager                    *MCPManager
	mcpRegistrationProposals      *mcpRegistrationProposalStore
	resourceRegistrationProposals *resourceRegistrationProposalStore
	skillManager                  *SkillManager
	extensionSupervisor           *ExtensionSupervisor
	extensionCredentials          *HostCredentialBroker
	prompts                       *PromptRegistry
}

func NewService(store Store) *Service {
	prompts, promptErr := NewBuiltinPromptRegistry()
	if rootStore, ok := store.(promptRootStore); ok {
		if root, err := rootStore.ManagedPromptRoot(); err == nil {
			prompts, promptErr = NewPromptRegistry(root)
		} else {
			promptErr = err
		}
	}
	if promptErr != nil {
		log.Printf("prompt_catalog init_failed error_class=prompt_initialization")
		prompts, _ = NewBuiltinPromptRegistry()
	}
	ptyManager := NewAgentPTYRegistry()
	service := &Service{
		store:                         store,
		now:                           time.Now,
		secrets:                       NewDefaultSecretStore(),
		providers:                     NewDefaultProviderRegistry(),
		providerContributions:         map[string]ProviderDefinition{},
		agentCatalog:                  NewAgentCatalog(),
		rateLimiter:                   newProviderRateLimiter(),
		permissionNotifier:            newPermissionNotifier(),
		questionNotifier:              newPermissionNotifier(),
		terminals:                     NewTerminalServiceWithRegistry(ptyManager),
		ptyManager:                    ptyManager,
		refreshedModels:               map[string][]domain.ModelInfo{},
		refreshedDefault:              map[string]string{},
		refreshedInfo:                 map[string]domain.ProviderInfo{},
		providerCapabilitySynced:      map[string]bool{},
		activeAgentRunCancel:          map[string]context.CancelFunc{},
		activeTurnCancel:              map[string]context.CancelFunc{},
		mcpRegistrationProposals:      newMCPRegistrationProposalStore(),
		resourceRegistrationProposals: newResourceRegistrationProposalStore(),
		prompts:                       prompts,
	}
	service.mcpManager = NewMCPManager(store, service.secrets)
	service.skillManager = NewSkillManager(store)
	if err := service.skillManager.SyncAivoSystemSkills(context.Background()); err != nil {
		log.Printf("aivo_system_skills init_failed error_class=system_skill_initialization")
	}
	service.extensionSupervisor = NewExtensionSupervisor()
	service.extensionCredentials = NewHostCredentialBroker(service.secrets)
	service.extensionSupervisor.SetCredentialBroker(service.extensionCredentials)
	if loaded, err := LoadBuiltinExtensionManifest(projectExtensionManifest); err != nil {
		log.Printf("builtin_extension init_failed id=%s error_class=manifest_validation", projectExtensionID)
	} else if _, err := service.extensionSupervisor.InstallBuiltin(context.Background(), loaded, func() extensionRuntimeClient {
		return &projectBuiltinExtensionClient{service: service}
	}); err != nil {
		log.Printf("builtin_extension init_failed id=%s error_class=runtime_initialization", projectExtensionID)
	}
	if loaded, err := LoadBuiltinExtensionManifest(toolRegistrationExtensionManifest); err != nil {
		log.Printf("builtin_extension init_failed id=%s error_class=manifest_validation", toolRegistrationExtensionID)
	} else if _, err := service.extensionSupervisor.InstallBuiltin(context.Background(), loaded, func() extensionRuntimeClient {
		return &toolRegistrationBuiltinExtensionClient{service: service}
	}); err != nil {
		log.Printf("builtin_extension init_failed id=%s error_class=runtime_initialization", toolRegistrationExtensionID)
	}
	service.restoreInstalledExtensions(context.Background())
	service.refreshProviderExtensions("")
	service.authFlows = NewProviderAuthManager(service)
	service.startSchedulerLoop()
	service.terminals.SetEventHook(func(name string, info TerminalInfo) {
		if service.onTerminalEvent != nil {
			service.onTerminalEvent(name, info)
		}
	})
	service.titleGenerator = service.generateSessionTitle
	reclaimStaleRetainedOutput(service.now())
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
	if s.extensionCredentials != nil {
		s.extensionCredentials.SetStore(store)
	}
}

func (s *Service) Shutdown() {
	if s.mcpRegistrationProposals != nil {
		s.mcpRegistrationProposals.clear()
	}
	if s.schedulerCancel != nil {
		s.schedulerCancel()
		s.schedulerCancel = nil
	}
	if s.terminals != nil {
		s.terminals.Shutdown()
	}
	defaultShellProcessRegistry.Shutdown()
	defaultAgentShellRegistry.Shutdown()
	if s.ptyManager != nil {
		s.ptyManager.Shutdown()
	}
	if s.extensionSupervisor != nil {
		s.extensionSupervisor.mu.Lock()
		ids := make([]string, 0, len(s.extensionSupervisor.items))
		for id := range s.extensionSupervisor.items {
			ids = append(ids, id)
		}
		s.extensionSupervisor.mu.Unlock()
		for _, id := range ids {
			_, _ = s.extensionSupervisor.Stop(context.Background(), id)
		}
	}
}
