package domain

import "time"

type ModelRef struct {
	ProviderID string `json:"providerId"`
	ModelID    string `json:"modelId"`
}

type ProviderConfig struct {
	ID            string            `json:"id"`
	Type          string            `json:"type"`
	BaseURL       string            `json:"baseUrl,omitempty"`
	APIKey        string            `json:"apiKey,omitempty"`
	APIKeyEnv     string            `json:"apiKeyEnv,omitempty"`
	Model         string            `json:"model,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	RequestParams map[string]any    `json:"requestParams,omitempty"`
}

type ProviderSettings struct {
	Custom   map[string]ProviderConfig `json:"custom,omitempty"`
	Disabled []string                  `json:"disabled,omitempty"`
}

type PersistenceRolloutConfig struct {
	Configured          bool   `json:"configured,omitempty"`
	JournalEnabled      bool   `json:"journalEnabled"`
	DualWriteValidation bool   `json:"dualWriteValidation"`
	ReadPath            string `json:"readPath,omitempty"`
}

type WebSearchConfig struct {
	Mode              string                 `json:"mode,omitempty"`
	Route             string                 `json:"route,omitempty"`
	LocalProvider     string                 `json:"localProvider,omitempty"`
	SearchContextSize string                 `json:"searchContextSize,omitempty"`
	AllowedDomains    []string               `json:"allowedDomains,omitempty"`
	UserLocation      *WebSearchUserLocation `json:"userLocation,omitempty"`
}

type NativeToolsConfig struct {
	XSearch       NativeToolToggle          `json:"xSearch,omitempty"`
	CodeExecution NativeCodeExecutionConfig `json:"codeExecution,omitempty"`
	FileSearch    NativeFileSearchConfig    `json:"fileSearch,omitempty"`
	RemoteMCP     []NativeMCPToolConfig     `json:"remoteMcp,omitempty"`
}

type NativeToolToggle struct {
	Enabled bool `json:"enabled,omitempty"`
}

type NativeCodeExecutionConfig struct {
	Enabled     bool     `json:"enabled,omitempty"`
	ContainerID string   `json:"containerId,omitempty"`
	FileIDs     []string `json:"fileIds,omitempty"`
}

type NativeFileSearchConfig struct {
	Enabled        bool     `json:"enabled,omitempty"`
	VectorStoreIDs []string `json:"vectorStoreIds,omitempty"`
}

type NativeMCPToolConfig struct {
	Enabled      bool     `json:"enabled,omitempty"`
	ServerURL    string   `json:"serverUrl,omitempty"`
	ServerLabel  string   `json:"serverLabel,omitempty"`
	AllowedTools []string `json:"allowedTools,omitempty"`
}

const (
	WebSearchModeDisabled  = "disabled"
	WebSearchModeLive      = "live"
	WebSearchRouteAuto     = "auto"
	WebSearchRouteLocal    = "local"
	WebSearchRouteProvider = "provider"
)

type AppConfig struct {
	Initialized     bool                     `json:"initialized"`
	Provider        *ProviderConfig          `json:"provider,omitempty"`
	Providers       ProviderSettings         `json:"providers,omitempty"`
	DefaultModel    *ModelRef                `json:"defaultModel,omitempty"`
	AuxiliaryModel  *ModelRef                `json:"auxiliaryModel,omitempty"`
	FallbackModels  []ModelRef               `json:"fallbackModels,omitempty"`
	ProviderPolicy  ProviderRuntimePolicy    `json:"providerPolicy,omitempty"`
	ReasoningEffort string                   `json:"reasoningEffort,omitempty"`
	ServiceTier     string                   `json:"serviceTier,omitempty"`
	Persistence     PersistenceRolloutConfig `json:"persistence,omitempty"`
	WebSearch       WebSearchConfig          `json:"webSearch,omitempty"`
	NativeTools     NativeToolsConfig        `json:"nativeTools,omitempty"`
	ConfigPath      string                   `json:"configPath,omitempty"`
}

type ProviderRuntimePolicy struct {
	EnableFallback           *bool `json:"enableFallback,omitempty"`
	BufferStreamingFallback  *bool `json:"bufferStreamingFallback,omitempty"`
	MaxRetries               int   `json:"maxRetries,omitempty"`
	RetryBaseDelayMs         int   `json:"retryBaseDelayMs,omitempty"`
	RateLimitCooldownSeconds int   `json:"rateLimitCooldownSeconds,omitempty"`
}

type ModelPreferencesInput struct {
	Model           *ModelRef              `json:"model,omitempty"`
	AuxiliaryModel  *ModelRef              `json:"auxiliaryModel,omitempty"`
	FallbackModels  []ModelRef             `json:"fallbackModels,omitempty"`
	ProviderPolicy  *ProviderRuntimePolicy `json:"providerPolicy,omitempty"`
	ReasoningEffort string                 `json:"reasoningEffort,omitempty"`
	ServiceTier     string                 `json:"serviceTier,omitempty"`
	NativeTools     *NativeToolsConfig     `json:"nativeTools,omitempty"`
}

type ProviderConnectInput struct {
	ProviderID    string            `json:"providerId"`
	Name          string            `json:"name,omitempty"`
	Type          string            `json:"type,omitempty"`
	BaseURL       string            `json:"baseUrl,omitempty"`
	APIKey        string            `json:"apiKey,omitempty"`
	APIKeyEnv     string            `json:"apiKeyEnv,omitempty"`
	ModelID       string            `json:"modelId,omitempty"`
	Method        string            `json:"method,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	RequestParams map[string]any    `json:"requestParams,omitempty"`
}

type ProviderCallEventsInput struct {
	ProviderID string `json:"providerId,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

type ProviderUsageInput struct {
	ProviderID string `json:"providerId,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

type ProviderUsageStats struct {
	ProviderID        string `json:"providerId,omitempty"`
	CallCount         int    `json:"callCount"`
	SuccessCount      int    `json:"successCount"`
	FailureCount      int    `json:"failureCount"`
	InputTokens       int    `json:"inputTokens"`
	OutputTokens      int    `json:"outputTokens"`
	TotalTokens       int    `json:"totalTokens"`
	CostMicros        int64  `json:"costMicros"`
	Estimated         bool   `json:"estimated"`
	LastCallAt        string `json:"lastCallAt,omitempty"`
	LastErrorClass    string `json:"lastErrorClass,omitempty"`
	LastErrorMessage  string `json:"lastErrorMessage,omitempty"`
	LastErrorProvider string `json:"lastErrorProvider,omitempty"`
}

type ProviderIntegrationCheckInput struct {
	ProviderID       string `json:"providerId"`
	ModelID          string `json:"modelId,omitempty"`
	IncludeModelList bool   `json:"includeModelList,omitempty"`
}

type ProviderIntegrationCheckResult struct {
	ProviderID   string                         `json:"providerId"`
	ModelID      string                         `json:"modelId,omitempty"`
	Ready        bool                           `json:"ready"`
	Status       string                         `json:"status"`
	CheckedAt    string                         `json:"checkedAt"`
	Steps        []ProviderIntegrationCheckStep `json:"steps"`
	Transport    string                         `json:"transport,omitempty"`
	BaseURL      string                         `json:"baseUrl,omitempty"`
	AuthMode     string                         `json:"authMode,omitempty"`
	ModelCount   int                            `json:"modelCount,omitempty"`
	Capabilities []string                       `json:"capabilities,omitempty"`
	Policy       ProviderRuntimePolicy          `json:"policy,omitempty"`
	Health       *ProviderHealth                `json:"health,omitempty"`
	Usage        *ProviderUsageStats            `json:"usage,omitempty"`
	Validation   *ProviderValidationResult      `json:"validation,omitempty"`
	RecentEvents []ProviderCallEvent            `json:"recentEvents,omitempty"`
	Recommended  []string                       `json:"recommended,omitempty"`
}

type ProviderIntegrationCheckStep struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

type ProviderAuthStartInput struct {
	ProviderID string `json:"providerId"`
	Method     string `json:"method"`
}

type ProviderAuthStartResult struct {
	ProviderID    string `json:"providerId"`
	Method        string `json:"method"`
	Status        string `json:"status"`
	URL           string `json:"url,omitempty"`
	Instructions  string `json:"instructions,omitempty"`
	UserCode      string `json:"userCode,omitempty"`
	ExpiresAt     string `json:"expiresAt,omitempty"`
	Authorization string `json:"authorization,omitempty"`
}

type ProviderAuthStatus struct {
	ProviderID   string `json:"providerId"`
	Method       string `json:"method"`
	Status       string `json:"status"`
	Error        string `json:"error,omitempty"`
	AccountID    string `json:"accountId,omitempty"`
	Instructions string `json:"instructions,omitempty"`
	UserCode     string `json:"userCode,omitempty"`
}

type ProviderAuthRecord struct {
	ID              string `json:"id,omitempty"`
	ProviderID      string `json:"providerId"`
	Method          string `json:"method"`
	AccessToken     string `json:"accessToken,omitempty"`
	AccessTokenRef  string `json:"accessTokenRef,omitempty"`
	RefreshToken    string `json:"refreshToken,omitempty"`
	RefreshTokenRef string `json:"refreshTokenRef,omitempty"`
	ExpiresAt       string `json:"expiresAt,omitempty"`
	AccountID       string `json:"accountId,omitempty"`
	DisplayName     string `json:"displayName,omitempty"`
	APIKey          string `json:"apiKey,omitempty"`
	APIKeyRef       string `json:"apiKeyRef,omitempty"`
	UpdatedAt       string `json:"updatedAt"`
}

type ProviderAccountInfo struct {
	ID          string `json:"id"`
	ProviderID  string `json:"providerId"`
	Method      string `json:"method"`
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
	ConnectedAt string `json:"connectedAt,omitempty"`
}

type ModelInfo struct {
	ID                string             `json:"id"`
	ProviderID        string             `json:"providerId"`
	Name              string             `json:"name"`
	Recommended       bool               `json:"recommended,omitempty"`
	Deprecated        bool               `json:"deprecated,omitempty"`
	ContextLength     int                `json:"contextLength,omitempty"`
	OutputLimit       int                `json:"outputLimit,omitempty"`
	Capabilities      []string           `json:"capabilities,omitempty"`
	Modalities        []string           `json:"modalities,omitempty"`
	Streaming         bool               `json:"streaming,omitempty"`
	ToolSupport       bool               `json:"toolSupport,omitempty"`
	ReasoningControls []string           `json:"reasoningControls,omitempty"`
	Pricing           map[string]float64 `json:"pricing,omitempty"`
	Status            string             `json:"status,omitempty"`
	LastRefreshed     string             `json:"lastRefreshed,omitempty"`
}

type ProviderAuthMethod struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Stable      bool   `json:"stable"`
	Available   bool   `json:"available"`
	Description string `json:"description,omitempty"`
}

type AuthInfo struct {
	Type            string `json:"type"`
	Connected       bool   `json:"connected"`
	Source          string `json:"source"`
	Environment     string `json:"environment,omitempty"`
	LastValidatedAt string `json:"lastValidatedAt,omitempty"`
	ConnectedAt     string `json:"connectedAt,omitempty"`
}

type ProviderInfo struct {
	ID               string                `json:"id"`
	Name             string                `json:"name"`
	Description      string                `json:"description,omitempty"`
	Type             string                `json:"type"`
	BaseURL          string                `json:"baseUrl,omitempty"`
	BuiltIn          bool                  `json:"builtIn"`
	Custom           bool                  `json:"custom"`
	Experimental     bool                  `json:"experimental,omitempty"`
	Deprecated       bool                  `json:"deprecated,omitempty"`
	Connected        bool                  `json:"connected"`
	ConnectionSource string                `json:"connectionSource,omitempty"`
	Environment      string                `json:"environment,omitempty"`
	DefaultModelID   string                `json:"defaultModelId,omitempty"`
	Models           []ModelInfo           `json:"models"`
	AuthMethods      []ProviderAuthMethod  `json:"authMethods"`
	Auth             *AuthInfo             `json:"auth,omitempty"`
	Accounts         []ProviderAccountInfo `json:"accounts,omitempty"`
	Readiness        *ProviderReadiness    `json:"readiness,omitempty"`
	Health           *ProviderHealth       `json:"health,omitempty"`
	ModelRefresh     *ProviderModelRefresh `json:"modelRefresh,omitempty"`
	Profile          *ProviderProfile      `json:"profile,omitempty"`
}

type ProviderReadiness struct {
	Ready       bool   `json:"ready"`
	AuthMode    string `json:"authMode,omitempty"`
	Source      string `json:"source,omitempty"`
	Environment string `json:"environment,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

type ProviderModelRefresh struct {
	Strategy    string `json:"strategy"`
	Status      string `json:"status"`
	LastRefresh string `json:"lastRefresh,omitempty"`
	Error       string `json:"error,omitempty"`
	ModelCount  int    `json:"modelCount"`
	Refreshable bool   `json:"refreshable"`
	CacheSource string `json:"cacheSource,omitempty"`
	ParserType  string `json:"parserType,omitempty"`
	Endpoint    string `json:"endpoint,omitempty"`
	Stale       bool   `json:"stale,omitempty"`
}

type ProviderHealth struct {
	ProviderID       string `json:"providerId"`
	Status           string `json:"status"`
	LastSuccessAt    string `json:"lastSuccessAt,omitempty"`
	LastFailureAt    string `json:"lastFailureAt,omitempty"`
	LastLatencyMs    int64  `json:"lastLatencyMs,omitempty"`
	LastErrorClass   string `json:"lastErrorClass,omitempty"`
	LastErrorMessage string `json:"lastErrorMessage,omitempty"`
	LastHTTPStatus   int    `json:"lastHttpStatus,omitempty"`
	FailureCount     int    `json:"failureCount,omitempty"`
	UpdatedAt        string `json:"updatedAt"`
}

type ProviderCallEvent struct {
	ID            string `json:"id,omitempty"`
	ProviderID    string `json:"providerId"`
	ModelID       string `json:"modelId"`
	Transport     string `json:"transport,omitempty"`
	Status        string `json:"status"`
	ErrorClass    string `json:"errorClass,omitempty"`
	ErrorMessage  string `json:"errorMessage,omitempty"`
	HTTPStatus    int    `json:"httpStatus,omitempty"`
	LatencyMs     int64  `json:"latencyMs,omitempty"`
	InputTokens   int    `json:"inputTokens,omitempty"`
	OutputTokens  int    `json:"outputTokens,omitempty"`
	TotalTokens   int    `json:"totalTokens,omitempty"`
	CostMicros    int64  `json:"costMicros,omitempty"`
	Estimated     bool   `json:"estimated,omitempty"`
	Attempt       int    `json:"attempt,omitempty"`
	FallbackIndex int    `json:"fallbackIndex,omitempty"`
	Streaming     bool   `json:"streaming,omitempty"`
	ToolCallCount int    `json:"toolCallCount,omitempty"`
	CreatedAt     string `json:"createdAt"`
}

type ProviderModelCache struct {
	ProviderID   string      `json:"providerId"`
	Models       []ModelInfo `json:"models"`
	DefaultModel string      `json:"defaultModel,omitempty"`
	Strategy     string      `json:"strategy,omitempty"`
	ParserType   string      `json:"parserType,omitempty"`
	Endpoint     string      `json:"endpoint,omitempty"`
	CacheSource  string      `json:"cacheSource,omitempty"`
	Status       string      `json:"status,omitempty"`
	Error        string      `json:"error,omitempty"`
	RefreshedAt  string      `json:"refreshedAt,omitempty"`
	UpdatedAt    string      `json:"updatedAt,omitempty"`
}

type ProviderValidationResult struct {
	ProviderID   string      `json:"providerId"`
	Ready        bool        `json:"ready"`
	Status       string      `json:"status"`
	Transport    string      `json:"transport,omitempty"`
	AuthMode     string      `json:"authMode,omitempty"`
	Source       string      `json:"source,omitempty"`
	Environment  string      `json:"environment,omitempty"`
	BaseURL      string      `json:"baseUrl,omitempty"`
	DefaultModel string      `json:"defaultModel,omitempty"`
	ModelCount   int         `json:"modelCount,omitempty"`
	Models       []ModelInfo `json:"models,omitempty"`
	Error        string      `json:"error,omitempty"`
	CheckedAt    string      `json:"checkedAt"`
}

type ProviderProfile struct {
	ID              string                  `json:"id"`
	DisplayName     string                  `json:"displayName"`
	ProviderType    string                  `json:"providerType"`
	InteractiveAuth bool                    `json:"interactiveAuth,omitempty"`
	ModelEndpoint   string                  `json:"modelEndpoint,omitempty"`
	ModelFetch      string                  `json:"modelFetch,omitempty"`
	ParserType      string                  `json:"parserType,omitempty"`
	CacheTtlSeconds int                     `json:"cacheTtlSeconds,omitempty"`
	Paginated       bool                    `json:"paginated,omitempty"`
	MessageShape    string                  `json:"messageShape,omitempty"`
	SupportedExtras []string                `json:"supportedExtras,omitempty"`
	RequestProfile  *ProviderRequestProfile `json:"requestProfile,omitempty"`
}

type ProviderRequestProfile struct {
	Headers        map[string]string                  `json:"headers,omitempty"`
	Params         map[string]any                     `json:"params,omitempty"`
	ModelOverrides map[string]ProviderRequestOverride `json:"modelOverrides,omitempty"`
}

type ProviderRequestOverride struct {
	Headers map[string]string `json:"headers,omitempty"`
	Params  map[string]any    `json:"params,omitempty"`
}

type CatalogState struct {
	Providers          []ProviderInfo `json:"providers"`
	Models             []ModelInfo    `json:"models"`
	Connected          []string       `json:"connected"`
	DefaultModel       *ModelRef      `json:"defaultModel,omitempty"`
	ConnectedProviders []ProviderInfo `json:"connectedProviders,omitempty"`
	PopularProviders   []ProviderInfo `json:"popularProviders,omitempty"`
	CustomProviders    []ProviderInfo `json:"customProviders,omitempty"`
}

type AssistantProject struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	RootPath      string `json:"rootPath"`
	GitBranch     string `json:"gitBranch,omitempty"`
	GitDirty      bool   `json:"gitDirty,omitempty"`
	GitAvailable  bool   `json:"gitAvailable"`
	SidebarHidden bool   `json:"sidebarHidden,omitempty"`
	TimeOpened    string `json:"timeOpened"`
	TimeUpdated   string `json:"timeUpdated"`
}

func NowString(now time.Time) string {
	return now.UTC().Format(time.RFC3339Nano)
}
