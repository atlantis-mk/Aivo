package persistence

type pluginInstallRow struct {
	ID           string `gorm:"primaryKey;column:id"`
	ManifestName string `gorm:"column:manifest_name;not null"`
	Version      string `gorm:"column:version"`
	DisplayName  string `gorm:"column:display_name"`
	Description  string `gorm:"column:description"`
	RootPath     string `gorm:"column:root_path;not null;uniqueIndex"`
	ManifestPath string `gorm:"column:manifest_path;not null"`
	Manifest     string `gorm:"column:manifest;not null"`
	Enabled      int    `gorm:"column:enabled;not null;default:0;index:plugin_installs_enabled_idx"`
	Status       string `gorm:"column:status;not null"`
	Error        string `gorm:"column:error"`
	TimeCreated  string `gorm:"column:time_created;not null"`
	TimeUpdated  string `gorm:"column:time_updated;not null"`
}

func (pluginInstallRow) TableName() string { return "plugin_installs" }

type pluginDiagnosticRow struct {
	ID          string `gorm:"primaryKey;column:id"`
	PluginID    string `gorm:"column:plugin_id;index:plugin_diagnostics_plugin_idx"`
	ServerID    string `gorm:"column:server_id;index:plugin_diagnostics_server_idx"`
	Level       string `gorm:"column:level;not null"`
	Message     string `gorm:"column:message;not null"`
	Metadata    string `gorm:"column:metadata"`
	TimeCreated string `gorm:"column:time_created;not null;index:plugin_diagnostics_created_idx"`
}

func (pluginDiagnosticRow) TableName() string { return "plugin_diagnostics" }

type mcpServerRow struct {
	ID                    string `gorm:"primaryKey;column:id"`
	Name                  string `gorm:"column:name;not null"`
	DisplayName           string `gorm:"column:display_name"`
	Description           string `gorm:"column:description"`
	Transport             string `gorm:"column:transport;not null"`
	Command               string `gorm:"column:command"`
	Args                  string `gorm:"column:args"`
	CWD                   string `gorm:"column:cwd"`
	Env                   string `gorm:"column:env"`
	URL                   string `gorm:"column:url"`
	Headers               string `gorm:"column:headers"`
	AuthType              string `gorm:"column:auth_type"`
	BearerTokenEnv        string `gorm:"column:bearer_token_env"`
	OAuthIssuerURL        string `gorm:"column:oauth_issuer_url"`
	OAuthClientID         string `gorm:"column:oauth_client_id"`
	OAuthScopes           string `gorm:"column:oauth_scopes"`
	OAuthAccessTokenRef   string `gorm:"column:oauth_access_token_ref"`
	OAuthRefreshTokenRef  string `gorm:"column:oauth_refresh_token_ref"`
	OAuthExpiresAt        string `gorm:"column:oauth_expires_at"`
	Roots                 string `gorm:"column:roots"`
	TimeoutSeconds        int    `gorm:"column:timeout_seconds;not null;default:0"`
	ConnectTimeoutSeconds int    `gorm:"column:connect_timeout_seconds;not null;default:0"`
	Enabled               int    `gorm:"column:enabled;not null;default:0;index:mcp_servers_enabled_idx"`
	PluginID              string `gorm:"column:plugin_id;index:mcp_servers_plugin_idx"`
	Status                string `gorm:"column:status;not null"`
	Error                 string `gorm:"column:error"`
	TimeCreated           string `gorm:"column:time_created;not null"`
	TimeUpdated           string `gorm:"column:time_updated;not null"`
}

func (mcpServerRow) TableName() string { return "mcp_servers" }

type mcpToolRow struct {
	ID          string `gorm:"primaryKey;column:id"`
	ServerID    string `gorm:"column:server_id;not null;index:mcp_tools_server_idx"`
	Name        string `gorm:"column:name;not null;index:mcp_tools_name_idx"`
	Description string `gorm:"column:description"`
	InputSchema string `gorm:"column:input_schema"`
	Capability  string `gorm:"column:capability"`
	RiskLevel   string `gorm:"column:risk_level"`
	TimeUpdated string `gorm:"column:time_updated;not null"`
}

func (mcpToolRow) TableName() string { return "mcp_tools" }

type mcpPromptRow struct {
	ID          string `gorm:"primaryKey;column:id"`
	ServerID    string `gorm:"column:server_id;not null;index:mcp_prompts_server_idx"`
	Name        string `gorm:"column:name;not null;index:mcp_prompts_name_idx"`
	Description string `gorm:"column:description"`
	Arguments   string `gorm:"column:arguments"`
	TimeUpdated string `gorm:"column:time_updated;not null"`
}

func (mcpPromptRow) TableName() string { return "mcp_prompts" }

type mcpResourceRow struct {
	ID          string `gorm:"primaryKey;column:id"`
	ServerID    string `gorm:"column:server_id;not null;index:mcp_resources_server_idx"`
	URI         string `gorm:"column:uri;index:mcp_resources_uri_idx"`
	URITemplate string `gorm:"column:uri_template"`
	Name        string `gorm:"column:name;not null;index:mcp_resources_name_idx"`
	Description string `gorm:"column:description"`
	MimeType    string `gorm:"column:mime_type"`
	Template    int    `gorm:"column:template;not null;default:0;index:mcp_resources_template_idx"`
	TimeUpdated string `gorm:"column:time_updated;not null"`
}

func (mcpResourceRow) TableName() string { return "mcp_resources" }

type skillRow struct {
	ID          string `gorm:"primaryKey;column:id"`
	Name        string `gorm:"column:name;not null;uniqueIndex:skills_name_scope_idx"`
	Description string `gorm:"column:description"`
	Scope       string `gorm:"column:scope;not null;uniqueIndex:skills_name_scope_idx;index:skills_scope_idx"`
	Source      string `gorm:"column:source;not null;index:skills_source_idx"`
	RootPath    string `gorm:"column:root_path;not null"`
	SkillPath   string `gorm:"column:skill_path;not null"`
	ContentHash string `gorm:"column:content_hash;not null;index:skills_hash_idx"`
	Enabled     int    `gorm:"column:enabled;not null;default:1;index:skills_enabled_idx"`
	Metadata    string `gorm:"column:metadata"`
	TimeCreated string `gorm:"column:time_created;not null"`
	TimeUpdated string `gorm:"column:time_updated;not null"`
}

func (skillRow) TableName() string { return "skills" }

type skillSourceRow struct {
	ID          string `gorm:"primaryKey;column:id"`
	SkillID     string `gorm:"column:skill_id;not null;index:skill_sources_skill_idx"`
	Source      string `gorm:"column:source;not null;index:skill_sources_source_idx"`
	Scope       string `gorm:"column:scope;not null"`
	RootPath    string `gorm:"column:root_path;not null"`
	SkillPath   string `gorm:"column:skill_path;not null"`
	ContentHash string `gorm:"column:content_hash;not null"`
	LastSeenAt  string `gorm:"column:last_seen_at;not null"`
}

func (skillSourceRow) TableName() string { return "skill_sources" }

type skillImportCandidateRow struct {
	ID          string `gorm:"primaryKey;column:id"`
	Name        string `gorm:"column:name;not null;index:skill_candidates_name_idx"`
	Description string `gorm:"column:description"`
	Scope       string `gorm:"column:scope;not null;index:skill_candidates_scope_idx"`
	Source      string `gorm:"column:source;not null;index:skill_candidates_source_idx"`
	RootPath    string `gorm:"column:root_path;not null;uniqueIndex:skill_candidates_path_idx"`
	SkillPath   string `gorm:"column:skill_path;not null;uniqueIndex:skill_candidates_path_idx"`
	ContentHash string `gorm:"column:content_hash;not null;index:skill_candidates_hash_idx"`
	Status      string `gorm:"column:status;not null;index:skill_candidates_status_idx"`
	ConflictID  string `gorm:"column:conflict_id"`
	Error       string `gorm:"column:error"`
	LastSeenAt  string `gorm:"column:last_seen_at;not null"`
}

func (skillImportCandidateRow) TableName() string { return "skill_import_candidates" }

type toolRegistrationRow struct {
	ID          string `gorm:"primaryKey;column:id"`
	Name        string `gorm:"column:name;not null;index:tool_registrations_name_idx"`
	Source      string `gorm:"column:source;not null;index:tool_registrations_source_idx"`
	SourceID    string `gorm:"column:source_id;index:tool_registrations_source_idx"`
	Version     string `gorm:"column:version"`
	Spec        string `gorm:"column:spec;not null"`
	Enabled     int    `gorm:"column:enabled;not null;default:1"`
	TimeCreated string `gorm:"column:time_created;not null"`
	TimeUpdated string `gorm:"column:time_updated;not null"`
}

func (toolRegistrationRow) TableName() string { return "tool_registrations" }
