package domain

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
	Disabled      []string                  `json:"disabled,omitempty"`
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
	WebSearchModeCached    = "cached"
	WebSearchModeIndexed   = "indexed"
	WebSearchModeLive      = "live"
	WebSearchRouteAuto     = "auto"
	WebSearchRouteLocal    = "local"
	WebSearchRouteProvider = "provider"
)
