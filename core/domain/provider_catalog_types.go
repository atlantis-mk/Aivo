package domain

type ModelInfo struct {
	ID                          string             `json:"id"`
	ProviderID                  string             `json:"providerId"`
	Name                        string             `json:"name"`
	Recommended                 bool               `json:"recommended,omitempty"`
	Deprecated                  bool               `json:"deprecated,omitempty"`
	ContextLength               int                `json:"contextLength,omitempty"`
	MaxContextLength            int                `json:"maxContextLength,omitempty"`
	AutoCompactTokenLimit       int                `json:"autoCompactTokenLimit,omitempty"`
	OutputLimit                 int                `json:"outputLimit,omitempty"`
	Capabilities                []string           `json:"capabilities,omitempty"`
	DeclaredCapabilities        []string           `json:"declaredCapabilities,omitempty"`
	NativeTools                 []string           `json:"nativeTools,omitempty"`
	NativeToolsKnown            bool               `json:"nativeToolsKnown,omitempty"`
	Modalities                  []string           `json:"modalities,omitempty"`
	Streaming                   bool               `json:"streaming,omitempty"`
	ToolSupport                 bool               `json:"toolSupport,omitempty"`
	ReasoningControls           []string           `json:"reasoningControls,omitempty"`
	SupportedReasoningEfforts   []string           `json:"supportedReasoningEfforts,omitempty"`
	DefaultReasoningEffort      string             `json:"defaultReasoningEffort,omitempty"`
	SupportsVerbosity           *bool              `json:"supportsVerbosity,omitempty"`
	DefaultVerbosity            string             `json:"defaultVerbosity,omitempty"`
	ServiceTiers                []string           `json:"serviceTiers,omitempty"`
	DefaultServiceTier          string             `json:"defaultServiceTier,omitempty"`
	SupportsParallelToolCalls   *bool              `json:"supportsParallelToolCalls,omitempty"`
	WebSearchToolType           string             `json:"webSearchToolType,omitempty"`
	WebSearchToolTypeKnown      bool               `json:"webSearchToolTypeKnown,omitempty"`
	UseResponsesLite            *bool              `json:"useResponsesLite,omitempty"`
	SupportsImageDetailOriginal *bool              `json:"supportsImageDetailOriginal,omitempty"`
	Pricing                     map[string]float64 `json:"pricing,omitempty"`
	Status                      string             `json:"status,omitempty"`
	LastRefreshed               string             `json:"lastRefreshed,omitempty"`
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
	ID                 string                      `json:"id"`
	Name               string                      `json:"name"`
	Description        string                      `json:"description,omitempty"`
	Type               string                      `json:"type"`
	BaseURL            string                      `json:"baseUrl,omitempty"`
	BuiltIn            bool                        `json:"builtIn"`
	Custom             bool                        `json:"custom"`
	Experimental       bool                        `json:"experimental,omitempty"`
	Deprecated         bool                        `json:"deprecated,omitempty"`
	Connected          bool                        `json:"connected"`
	ConnectionSource   string                      `json:"connectionSource,omitempty"`
	Environment        string                      `json:"environment,omitempty"`
	DefaultModelID     string                      `json:"defaultModelId,omitempty"`
	Models             []ModelInfo                 `json:"models"`
	AuthMethods        []ProviderAuthMethod        `json:"authMethods"`
	Auth               *AuthInfo                   `json:"auth,omitempty"`
	Accounts           []ProviderAccountInfo       `json:"accounts,omitempty"`
	Readiness          *ProviderReadiness          `json:"readiness,omitempty"`
	Health             *ProviderHealth             `json:"health,omitempty"`
	ModelRefresh       *ProviderModelRefresh       `json:"modelRefresh,omitempty"`
	Profile            *ProviderProfile            `json:"profile,omitempty"`
	NativeCapabilities *ProviderNativeCapabilities `json:"nativeCapabilities,omitempty"`
}

// ProviderNativeCapabilities are provider-owned runtime feature upper bounds.
// They are distinct from the capabilities declared by an individual model.
type ProviderNativeCapabilities struct {
	NamespaceTools  bool   `json:"namespaceTools"`
	ImageGeneration bool   `json:"imageGeneration"`
	WebSearch       bool   `json:"webSearch"`
	Source          string `json:"source"`
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

type ProviderEcosystemRefreshInput struct {
	URL   string `json:"url,omitempty"`
	Force bool   `json:"force,omitempty"`
}

type ProviderEcosystemRefreshResult struct {
	Source           string `json:"source"`
	CachePath        string `json:"cachePath"`
	RefreshedAt      string `json:"refreshedAt"`
	ProviderCount    int    `json:"providerCount"`
	ModelCount       int    `json:"modelCount"`
	UnsupportedCount int    `json:"unsupportedCount,omitempty"`
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
