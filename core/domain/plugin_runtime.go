package domain

import "encoding/json"

const (
	PluginStatusEnabled  = "enabled"
	PluginStatusDisabled = "disabled"
	PluginStatusError    = "error"

	PluginDiagnosticInfo  = "info"
	PluginDiagnosticWarn  = "warn"
	PluginDiagnosticError = "error"

	ToolSourceBuiltin = "builtin"
	ToolSourcePlugin  = "plugin"
	ToolSourceMCP     = "mcp"
	ToolSourceBridge  = "bridge"
)

type PluginManifest struct {
	ID          string                                 `json:"id"`
	Name        string                                 `json:"name"`
	Version     string                                 `json:"version,omitempty"`
	DisplayName string                                 `json:"displayName,omitempty"`
	Description string                                 `json:"description,omitempty"`
	Author      string                                 `json:"author,omitempty"`
	Keywords    []string                               `json:"keywords,omitempty"`
	Entrypoint  PluginEntrypoint                       `json:"entrypoint,omitempty"`
	MCPServers  []MCPServerConfig                      `json:"mcpServers,omitempty"`
	Hooks       []string                               `json:"hooks,omitempty"`
	Tools       []PluginDeclaredTool                   `json:"tools,omitempty"`
	Providers   map[string]ProviderExtensionDefinition `json:"providers,omitempty"`
	Permissions PluginPermissionSummary                `json:"permissions,omitempty"`
	Interface   map[string]any                         `json:"interface,omitempty"`
}

type PluginEntrypoint struct {
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	CWD     string            `json:"cwd,omitempty"`
}

type PluginPermissionSummary struct {
	Capabilities   []string `json:"capabilities,omitempty"`
	RiskLevel      string   `json:"riskLevel,omitempty"`
	Network        bool     `json:"network,omitempty"`
	Workspace      bool     `json:"workspace,omitempty"`
	TouchesSecrets bool     `json:"touchesSecrets,omitempty"`
}

type PluginDeclaredTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
	Capability  string         `json:"capability,omitempty"`
	RiskLevel   string         `json:"riskLevel,omitempty"`
	Toolsets    []string       `json:"toolsets,omitempty"`
}

type PluginInstall struct {
	ID           string         `json:"id"`
	Manifest     PluginManifest `json:"manifest"`
	RootPath     string         `json:"rootPath"`
	ManifestPath string         `json:"manifestPath"`
	Enabled      bool           `json:"enabled"`
	Status       string         `json:"status"`
	Error        string         `json:"error,omitempty"`
	TimeCreated  string         `json:"timeCreated"`
	TimeUpdated  string         `json:"timeUpdated"`
}

type PluginDiagnostic struct {
	ID          string         `json:"id"`
	PluginID    string         `json:"pluginId,omitempty"`
	ServerID    string         `json:"serverId,omitempty"`
	Level       string         `json:"level"`
	Message     string         `json:"message"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	TimeCreated string         `json:"timeCreated"`
}

type PluginListInput struct {
	IncludeDisabled    bool `json:"includeDisabled,omitempty"`
	IncludeDiagnostics bool `json:"includeDiagnostics,omitempty"`
	Limit              int  `json:"limit,omitempty"`
}

type PluginListItem struct {
	Plugin      PluginInstall      `json:"plugin"`
	Diagnostics []PluginDiagnostic `json:"diagnostics,omitempty"`
	Tools       []ToolCatalogEntry `json:"tools,omitempty"`
}

type InstallPluginInput struct {
	Path   string `json:"path"`
	Enable bool   `json:"enable,omitempty"`
}

type SetPluginEnabledInput struct {
	PluginID string `json:"pluginId"`
	Enabled  bool   `json:"enabled"`
}

type ToolCatalogEntry struct {
	Name           string         `json:"name"`
	Description    string         `json:"description,omitempty"`
	InputSchema    map[string]any `json:"inputSchema,omitempty"`
	Namespace      string         `json:"namespace,omitempty"`
	Capability     string         `json:"capability,omitempty"`
	RiskLevel      string         `json:"riskLevel,omitempty"`
	Category       string         `json:"category,omitempty"`
	Toolsets       []string       `json:"toolsets,omitempty"`
	Source         string         `json:"source"`
	SourceID       string         `json:"sourceId,omitempty"`
	RegistrationID string         `json:"registrationId,omitempty"`
	Enabled        bool           `json:"enabled"`
}

type ToolCatalogInput struct {
	WorkspaceRoot   string `json:"workspaceRoot,omitempty"`
	IncludeDeferred bool   `json:"includeDeferred,omitempty"`
	Source          string `json:"source,omitempty"`
}

type ToolDescribeInput struct {
	Name          string `json:"name"`
	WorkspaceRoot string `json:"workspaceRoot,omitempty"`
}

type SessionActiveToolsInput struct {
	SessionID string   `json:"sessionId"`
	ToolNames []string `json:"toolNames,omitempty"`
}

type SessionActiveToolsResult struct {
	SessionID string   `json:"sessionId"`
	ToolNames []string `json:"toolNames"`
}

type ToolRegistrationIdentity struct {
	Name           string `json:"name"`
	RegistrationID string `json:"registrationId,omitempty"`
	Source         string `json:"source,omitempty"`
	SourceID       string `json:"sourceId,omitempty"`
	Version        string `json:"version,omitempty"`
}

type ToolCatalogSnapshot struct {
	Entries    []ToolCatalogEntry                  `json:"entries"`
	Identities map[string]ToolRegistrationIdentity `json:"identities,omitempty"`
}

func CloneRawMap(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	raw, _ := json.Marshal(value)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out
}
