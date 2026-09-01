package domain

type RuntimeConfig struct {
	Instructions         []string                               `json:"instructions,omitempty"`
	DefaultAgent         string                                 `json:"defaultAgent,omitempty"`
	Commands             map[string]CommandTemplateDefinition   `json:"commands,omitempty"`
	Agents               map[string]AgentRuntimeDefinition      `json:"agents,omitempty"`
	LanguageServers      map[string]LanguageServerDefinition    `json:"languageServers,omitempty"`
	ProviderExtensions   map[string]ProviderExtensionDefinition `json:"providerExtensions,omitempty"`
	Toolsets             map[string][]string                    `json:"toolsets,omitempty"`
	Permissions          map[string]string                      `json:"permissions,omitempty"`
	Compaction           CompactionRuntimeConfig                `json:"compaction,omitempty"`
	MaxParallelChildren  int                                    `json:"maxParallelChildren,omitempty"`
	ExecutionEnvironment ExecutionEnvironmentConfig             `json:"executionEnvironment,omitempty"`
}

type ExecutionEnvironmentConfig struct {
}

type CommandTemplateDefinition struct {
	Description string            `json:"description,omitempty"`
	Template    string            `json:"template"`
	Arguments   []CommandArgument `json:"arguments,omitempty"`
	Agent       string            `json:"agent,omitempty"`
	Model       *ModelRef         `json:"model,omitempty"`
	Toolsets    []string          `json:"toolsets,omitempty"`
	Subtask     bool              `json:"subtask,omitempty"`
}

type CommandArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Default     string `json:"default,omitempty"`
}

type AgentRuntimeDefinition struct {
	DisplayName     string         `json:"displayName,omitempty"`
	Description     string         `json:"description,omitempty"`
	Prompt          string         `json:"prompt,omitempty"`
	Model           *ModelRef      `json:"model,omitempty"`
	Temperature     *float64       `json:"temperature,omitempty"`
	TopP            *float64       `json:"topP,omitempty"`
	MaxSteps        int            `json:"maxSteps,omitempty"`
	Toolsets        []string       `json:"toolsets,omitempty"`
	PermissionScope string         `json:"permissionScope,omitempty"`
	Mode            string         `json:"mode,omitempty"`
	Subagents       []string       `json:"subagents,omitempty"`
	Variant         string         `json:"variant,omitempty"`
	Options         map[string]any `json:"options,omitempty"`
	Hidden          bool           `json:"hidden,omitempty"`
	Disabled        bool           `json:"disabled,omitempty"`
}

type LanguageServerDefinition struct {
	LanguageIDs           []string          `json:"languageIds,omitempty"`
	Extensions            []string          `json:"extensions,omitempty"`
	Filenames             []string          `json:"filenames,omitempty"`
	RootMarkers           []string          `json:"rootMarkers,omitempty"`
	StrictRoot            bool              `json:"strictRoot,omitempty"`
	Command               string            `json:"command"`
	Args                  []string          `json:"args,omitempty"`
	Env                   map[string]string `json:"env,omitempty"`
	InitializationOptions map[string]any    `json:"initializationOptions,omitempty"`
	TimeoutSeconds        int               `json:"timeoutSeconds,omitempty"`
	Disabled              bool              `json:"disabled,omitempty"`
}

type ProviderExtensionDefinition struct {
	Protocol      string            `json:"protocol"`
	DisplayName   string            `json:"displayName,omitempty"`
	BaseURL       string            `json:"baseUrl,omitempty"`
	Command       string            `json:"command,omitempty"`
	Args          []string          `json:"args,omitempty"`
	CredentialRef string            `json:"credentialRef,omitempty"`
	Models        []string          `json:"models,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
}

type CompactionRuntimeConfig struct {
	Auto             *bool `json:"auto,omitempty"`
	ThresholdPercent int   `json:"thresholdPercent,omitempty"`
	ReserveTokens    int   `json:"reserveTokens,omitempty"`
}

type RuntimeConfigSource struct {
	Path  string `json:"path"`
	Scope string `json:"scope"`
}

type RuntimeConfigDiagnostic struct {
	Path    string `json:"path"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

type EffectiveRuntimeConfig struct {
	ProjectPath string                    `json:"projectPath,omitempty"`
	Config      RuntimeConfig             `json:"config"`
	Sources     []RuntimeConfigSource     `json:"sources,omitempty"`
	Diagnostics []RuntimeConfigDiagnostic `json:"diagnostics,omitempty"`
}
