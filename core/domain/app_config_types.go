package domain

type AppConfig struct {
	Initialized                 bool                     `json:"initialized"`
	AppName                     string                   `json:"appName"`
	InitialWorkspacePath        string                   `json:"initialWorkspacePath,omitempty"`
	DefaultInitialWorkspacePath string                   `json:"defaultInitialWorkspacePath,omitempty"`
	Provider                    *ProviderConfig          `json:"provider,omitempty"`
	Providers                   ProviderSettings         `json:"providers,omitempty"`
	DefaultModel                *ModelRef                `json:"defaultModel,omitempty"`
	AuxiliaryModel              *ModelRef                `json:"auxiliaryModel,omitempty"`
	FallbackModels              []ModelRef               `json:"fallbackModels,omitempty"`
	ProviderPolicy              ProviderRuntimePolicy    `json:"providerPolicy,omitempty"`
	ReasoningEffort             string                   `json:"reasoningEffort,omitempty"`
	ServiceTier                 string                   `json:"serviceTier,omitempty"`
	DefaultPermissionMode       string                   `json:"defaultPermissionMode,omitempty"`
	Persistence                 PersistenceRolloutConfig `json:"persistence,omitempty"`
	WebSearch                   WebSearchConfig          `json:"webSearch,omitempty"`
	NativeTools                 NativeToolsConfig        `json:"nativeTools,omitempty"`
	Runtime                     RuntimeConfig            `json:"runtime,omitempty"`
	ConfigPath                  string                   `json:"configPath,omitempty"`
}

type CompleteInitializationInput struct {
	AppName              *string         `json:"appName,omitempty"`
	InitialWorkspacePath string          `json:"initialWorkspacePath"`
	Provider             *ProviderConfig `json:"provider,omitempty"`
}

type ProviderRuntimePolicy struct {
	EnableFallback           *bool `json:"enableFallback,omitempty"`
	BufferStreamingFallback  *bool `json:"bufferStreamingFallback,omitempty"`
	MaxRetries               int   `json:"maxRetries,omitempty"`
	RetryBaseDelayMs         int   `json:"retryBaseDelayMs,omitempty"`
	RateLimitCooldownSeconds int   `json:"rateLimitCooldownSeconds,omitempty"`
}

type ModelPreferencesInput struct {
	Model                 *ModelRef              `json:"model,omitempty"`
	AuxiliaryModel        *ModelRef              `json:"auxiliaryModel,omitempty"`
	FallbackModels        []ModelRef             `json:"fallbackModels,omitempty"`
	ProviderPolicy        *ProviderRuntimePolicy `json:"providerPolicy,omitempty"`
	ReasoningEffort       string                 `json:"reasoningEffort,omitempty"`
	ServiceTier           string                 `json:"serviceTier,omitempty"`
	DefaultPermissionMode string                 `json:"defaultPermissionMode,omitempty"`
	WebSearch             *WebSearchConfig       `json:"webSearch,omitempty"`
	NativeTools           *NativeToolsConfig     `json:"nativeTools,omitempty"`
}
