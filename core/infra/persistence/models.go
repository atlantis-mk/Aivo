package persistence

type schemaVersionRow struct {
	Version   int    `gorm:"primaryKey;column:version"`
	AppliedAt string `gorm:"column:applied_at;not null"`
}

func (schemaVersionRow) TableName() string { return "schema_version" }

type appConfigRow struct {
	ID              int    `gorm:"primaryKey;column:id"`
	Initialized     int    `gorm:"column:initialized;not null;default:0"`
	ProviderID      string `gorm:"column:provider_id"`
	ProviderType    string `gorm:"column:provider_type"`
	BaseURL         string `gorm:"column:base_url"`
	APIKeyEnv       string `gorm:"column:api_key_env"`
	Model           string `gorm:"column:model"`
	AuxiliaryModel  string `gorm:"column:auxiliary_model"`
	ReasoningEffort string `gorm:"column:reasoning_effort"`
	ServiceTier     string `gorm:"column:service_tier"`
	FallbackModels  string `gorm:"column:fallback_models"`
	ProviderPolicy  string `gorm:"column:provider_policy"`
	WebSearch       string `gorm:"column:web_search"`
	NativeTools     string `gorm:"column:native_tools"`
	Headers         string `gorm:"column:headers"`
	RequestParams   string `gorm:"column:request_params"`
	UpdatedAt       string `gorm:"column:updated_at;not null"`
}

func (appConfigRow) TableName() string { return "app_config" }

type providerRow struct {
	ID               string `gorm:"primaryKey;column:id"`
	Type             string `gorm:"column:type;not null"`
	BaseURL          string `gorm:"column:base_url"`
	APIKeyEnv        string `gorm:"column:api_key_env"`
	Model            string `gorm:"column:model"`
	Headers          string `gorm:"column:headers"`
	RequestParams    string `gorm:"column:request_params"`
	Status           string `gorm:"column:status;not null"`
	LastValidationAt string `gorm:"column:last_validation_at"`
	UpdatedAt        string `gorm:"column:updated_at;not null"`
}

func (providerRow) TableName() string { return "providers" }

type providerModelCacheRow struct {
	ProviderID   string `gorm:"primaryKey;column:provider_id"`
	Models       string `gorm:"column:models;not null"`
	DefaultModel string `gorm:"column:default_model"`
	Strategy     string `gorm:"column:strategy"`
	ParserType   string `gorm:"column:parser_type"`
	Endpoint     string `gorm:"column:endpoint"`
	CacheSource  string `gorm:"column:cache_source"`
	Status       string `gorm:"column:status;not null"`
	Error        string `gorm:"column:error"`
	RefreshedAt  string `gorm:"column:refreshed_at"`
	UpdatedAt    string `gorm:"column:updated_at;not null"`
}

func (providerModelCacheRow) TableName() string { return "provider_model_caches" }

type providerValidationRow struct {
	ProviderID   string `gorm:"primaryKey;column:provider_id"`
	Ready        int    `gorm:"column:ready;not null;default:0"`
	Status       string `gorm:"column:status;not null"`
	Transport    string `gorm:"column:transport"`
	AuthMode     string `gorm:"column:auth_mode"`
	Source       string `gorm:"column:source"`
	Environment  string `gorm:"column:environment"`
	BaseURL      string `gorm:"column:base_url"`
	DefaultModel string `gorm:"column:default_model"`
	ModelCount   int    `gorm:"column:model_count;not null;default:0"`
	Models       string `gorm:"column:models"`
	Error        string `gorm:"column:error"`
	CheckedAt    string `gorm:"column:checked_at;not null"`
}

func (providerValidationRow) TableName() string { return "provider_validations" }

type providerHealthRow struct {
	ProviderID       string `gorm:"primaryKey;column:provider_id"`
	Status           string `gorm:"column:status;not null"`
	LastSuccessAt    string `gorm:"column:last_success_at"`
	LastFailureAt    string `gorm:"column:last_failure_at"`
	LastLatencyMs    int64  `gorm:"column:last_latency_ms;not null;default:0"`
	LastErrorClass   string `gorm:"column:last_error_class"`
	LastErrorMessage string `gorm:"column:last_error_message"`
	LastHTTPStatus   int    `gorm:"column:last_http_status;not null;default:0"`
	FailureCount     int    `gorm:"column:failure_count;not null;default:0"`
	UpdatedAt        string `gorm:"column:updated_at;not null"`
}

func (providerHealthRow) TableName() string { return "provider_health" }

type providerCallEventRow struct {
	ID            string `gorm:"primaryKey;column:id"`
	ProviderID    string `gorm:"column:provider_id;not null;index:provider_call_events_provider_created_idx"`
	ModelID       string `gorm:"column:model_id;not null"`
	Transport     string `gorm:"column:transport"`
	Status        string `gorm:"column:status;not null"`
	ErrorClass    string `gorm:"column:error_class"`
	ErrorMessage  string `gorm:"column:error_message"`
	HTTPStatus    int    `gorm:"column:http_status;not null;default:0"`
	LatencyMs     int64  `gorm:"column:latency_ms;not null;default:0"`
	InputTokens   int    `gorm:"column:input_tokens;not null;default:0"`
	OutputTokens  int    `gorm:"column:output_tokens;not null;default:0"`
	TotalTokens   int    `gorm:"column:total_tokens;not null;default:0"`
	CostMicros    int64  `gorm:"column:cost_micros;not null;default:0"`
	Estimated     int    `gorm:"column:estimated;not null;default:0"`
	Attempt       int    `gorm:"column:attempt;not null;default:0"`
	FallbackIndex int    `gorm:"column:fallback_index;not null;default:0"`
	Streaming     int    `gorm:"column:streaming;not null;default:0"`
	ToolCallCount int    `gorm:"column:tool_call_count;not null;default:0"`
	CreatedAt     string `gorm:"column:created_at;not null;index:provider_call_events_provider_created_idx"`
}

func (providerCallEventRow) TableName() string { return "provider_call_events" }

type providerAuthRow struct {
	ID              string `gorm:"primaryKey;column:id"`
	ProviderID      string `gorm:"column:provider_id;not null;index:provider_auth_provider_id_idx"`
	Method          string `gorm:"column:method;not null"`
	AccessToken     string `gorm:"column:access_token"`
	AccessTokenRef  string `gorm:"column:access_token_ref"`
	RefreshToken    string `gorm:"column:refresh_token"`
	RefreshTokenRef string `gorm:"column:refresh_token_ref"`
	ExpiresAt       string `gorm:"column:expires_at"`
	AccountID       string `gorm:"column:account_id"`
	DisplayName     string `gorm:"column:display_name"`
	APIKey          string `gorm:"column:api_key"`
	APIKeyRef       string `gorm:"column:api_key_ref"`
	UpdatedAt       string `gorm:"column:updated_at;not null"`
}

func (providerAuthRow) TableName() string { return "provider_auth" }

type legacyProviderAuthRow struct {
	ProviderID   string `gorm:"column:provider_id"`
	Method       string `gorm:"column:method"`
	AccessToken  string `gorm:"column:access_token"`
	RefreshToken string `gorm:"column:refresh_token"`
	ExpiresAt    string `gorm:"column:expires_at"`
	AccountID    string `gorm:"column:account_id"`
	APIKey       string `gorm:"column:api_key"`
	UpdatedAt    string `gorm:"column:updated_at"`
}

func (legacyProviderAuthRow) TableName() string { return "provider_auth_old" }

type projectRow struct {
	ID            string `gorm:"primaryKey;column:id"`
	Name          string `gorm:"column:name;not null"`
	RootPath      string `gorm:"column:root_path;not null;uniqueIndex"`
	GitBranch     string `gorm:"column:git_branch"`
	GitDirty      int    `gorm:"column:git_dirty;not null;default:0"`
	GitAvailable  int    `gorm:"column:git_available;not null;default:0"`
	SidebarHidden int    `gorm:"column:sidebar_hidden;not null;default:0;index:projects_sidebar_hidden_updated_idx"`
	TimeOpened    string `gorm:"column:time_opened;not null"`
	TimeUpdated   string `gorm:"column:time_updated;not null;index:projects_sidebar_hidden_updated_idx"`
}

func (projectRow) TableName() string { return "projects" }

type sessionRow struct {
	ID                   string `gorm:"primaryKey;column:id"`
	Title                string `gorm:"column:title;not null"`
	ProjectID            string `gorm:"column:project_id"`
	ModelProviderID      string `gorm:"column:model_provider_id"`
	ModelID              string `gorm:"column:model_id"`
	TimeCreated          string `gorm:"column:time_created;not null"`
	TimeUpdated          string `gorm:"column:time_updated;not null;index:sessions_status_updated_idx;index:sessions_type_status_idx"`
	Type                 string `gorm:"column:type;not null;default:coding;index:sessions_type_status_idx"`
	Status               string `gorm:"column:status;not null;default:active;index:sessions_status_updated_idx;index:sessions_type_status_idx"`
	Source               string `gorm:"column:source;not null;default:desktop"`
	Goal                 string `gorm:"column:goal"`
	Summary              string `gorm:"column:summary"`
	ParentSessionID      string `gorm:"column:parent_session_id"`
	ForkedFromSessionID  string `gorm:"column:forked_from_session_id"`
	ModelSnapshot        string `gorm:"column:model_snapshot"`
	SystemPromptSnapshot string `gorm:"column:system_prompt_snapshot"`
	AgentMode            string `gorm:"column:agent_mode;not null;default:assistant;index:sessions_agent_mode_idx"`
	TokenCount           int    `gorm:"column:token_count;not null;default:0"`
	CostMicros           int64  `gorm:"column:cost_micros;not null;default:0"`
	Metadata             string `gorm:"column:metadata"`
	ArchivedAt           string `gorm:"column:archived_at"`
	DeletedAt            string `gorm:"column:deleted_at"`
}

func (sessionRow) TableName() string { return "sessions" }

type turnRow struct {
	ID            string `gorm:"primaryKey;column:id"`
	SessionID     string `gorm:"column:session_id;not null;index:turns_session_idx"`
	AgentMode     string `gorm:"column:agent_mode;not null;default:assistant;index:turns_agent_mode_idx"`
	Status        string `gorm:"column:status;not null"`
	UserEventID   string `gorm:"column:user_event_id"`
	Error         string `gorm:"column:error"`
	TimeCreated   string `gorm:"column:time_created;not null;index:turns_session_idx"`
	TimeCompleted string `gorm:"column:time_completed"`
	TimeUpdated   string `gorm:"column:time_updated;not null"`
}

func (turnRow) TableName() string { return "turns" }

type sessionEventRow struct {
	ID          string `gorm:"primaryKey;column:id"`
	SessionID   string `gorm:"column:session_id;not null;index:session_events_session_created_idx;index:session_events_visibility_idx"`
	TurnID      string `gorm:"column:turn_id"`
	Type        string `gorm:"column:type;not null"`
	Role        string `gorm:"column:role"`
	Visibility  string `gorm:"column:visibility;not null;index:session_events_visibility_idx"`
	Content     string `gorm:"column:content"`
	Payload     string `gorm:"column:payload"`
	TokenCount  int    `gorm:"column:token_count;not null;default:0"`
	TimeCreated string `gorm:"column:time_created;not null;index:session_events_session_created_idx;index:session_events_visibility_idx"`
}

func (sessionEventRow) TableName() string { return "session_events" }

type toolCallRow struct {
	ID            string `gorm:"primaryKey;column:id"`
	SessionID     string `gorm:"column:session_id;not null;index:tool_calls_session_idx"`
	TurnID        string `gorm:"column:turn_id"`
	EventID       string `gorm:"column:event_id"`
	Name          string `gorm:"column:name;not null"`
	Arguments     string `gorm:"column:arguments"`
	Status        string `gorm:"column:status;not null"`
	ResultSummary string `gorm:"column:result_summary"`
	Result        string `gorm:"column:result"`
	Error         string `gorm:"column:error"`
	TimeCreated   string `gorm:"column:time_created;not null;index:tool_calls_session_idx"`
	TimeUpdated   string `gorm:"column:time_updated;not null"`
}

func (toolCallRow) TableName() string { return "tool_calls" }

type sessionExecutionStateRow struct {
	ID              string `gorm:"primaryKey;column:id"`
	SessionID       string `gorm:"column:session_id;not null;uniqueIndex"`
	TurnID          string `gorm:"column:turn_id"`
	Status          string `gorm:"column:status;not null;index:session_execution_states_status_idx"`
	Reason          string `gorm:"column:reason"`
	LastEventID     string `gorm:"column:last_event_id"`
	PendingInputIDs string `gorm:"column:pending_input_ids"`
	Metadata        string `gorm:"column:metadata"`
	TimeCreated     string `gorm:"column:time_created;not null"`
	TimeUpdated     string `gorm:"column:time_updated;not null"`
}

func (sessionExecutionStateRow) TableName() string { return "session_execution_states" }

type pendingSessionInputRow struct {
	ID             string `gorm:"primaryKey;column:id"`
	SessionID      string `gorm:"column:session_id;not null;index:pending_session_inputs_session_status_idx"`
	TurnID         string `gorm:"column:turn_id"`
	Text           string `gorm:"column:text;not null"`
	Delivery       string `gorm:"column:delivery;not null"`
	Status         string `gorm:"column:status;not null;index:pending_session_inputs_session_status_idx"`
	PromotedTurnID string `gorm:"column:promoted_turn_id"`
	TimeCreated    string `gorm:"column:time_created;not null"`
	TimeUpdated    string `gorm:"column:time_updated;not null"`
}

func (pendingSessionInputRow) TableName() string { return "pending_session_inputs" }

type permissionRequestRow struct {
	ID          string `gorm:"primaryKey;column:id"`
	SessionID   string `gorm:"column:session_id;index:permission_requests_session_idx"`
	TurnID      string `gorm:"column:turn_id"`
	ToolCallID  string `gorm:"column:tool_call_id"`
	ToolName    string `gorm:"column:tool_name;not null"`
	Action      string `gorm:"column:action;not null"`
	Paths       string `gorm:"column:paths"`
	Arguments   string `gorm:"column:arguments"`
	Status      string `gorm:"column:status;not null;index:permission_requests_status_idx"`
	Remember    int    `gorm:"column:remember;not null;default:0"`
	Reason      string `gorm:"column:reason"`
	TimeCreated string `gorm:"column:time_created;not null;index:permission_requests_session_idx"`
	TimeUpdated string `gorm:"column:time_updated;not null"`
}

func (permissionRequestRow) TableName() string { return "permission_requests" }

type questionRequestRow struct {
	ID          string `gorm:"primaryKey;column:id"`
	SessionID   string `gorm:"column:session_id;index:question_requests_session_idx"`
	TurnID      string `gorm:"column:turn_id"`
	ToolCallID  string `gorm:"column:tool_call_id"`
	ToolName    string `gorm:"column:tool_name;not null"`
	Questions   string `gorm:"column:questions;not null"`
	Answers     string `gorm:"column:answers"`
	Status      string `gorm:"column:status;not null;index:question_requests_status_idx"`
	Reason      string `gorm:"column:reason"`
	Arguments   string `gorm:"column:arguments"`
	TimeCreated string `gorm:"column:time_created;not null;index:question_requests_session_idx"`
	TimeUpdated string `gorm:"column:time_updated;not null"`
}

func (questionRequestRow) TableName() string { return "question_requests" }

type permissionRuleRow struct {
	ID            string `gorm:"primaryKey;column:id"`
	Scope         string `gorm:"column:scope;not null"`
	SessionID     string `gorm:"column:session_id;index:permission_rules_session_idx"`
	WorkspaceRoot string `gorm:"column:workspace_root;index:permission_rules_workspace_idx"`
	ToolName      string `gorm:"column:tool_name;not null"`
	Action        string `gorm:"column:action;not null"`
	Decision      string `gorm:"column:decision;not null"`
	Paths         string `gorm:"column:paths"`
	TimeCreated   string `gorm:"column:time_created;not null"`
	TimeUpdated   string `gorm:"column:time_updated;not null"`
}

func (permissionRuleRow) TableName() string { return "permission_rules" }

type sessionSummaryRow struct {
	ID                  string `gorm:"primaryKey;column:id"`
	SessionID           string `gorm:"column:session_id;not null;index:session_summaries_session_created_idx"`
	FromEventID         string `gorm:"column:from_event_id"`
	ToEventID           string `gorm:"column:to_event_id"`
	Summary             string `gorm:"column:summary;not null"`
	Facts               string `gorm:"column:facts"`
	Decisions           string `gorm:"column:decisions"`
	OpenTasks           string `gorm:"column:open_tasks"`
	ChangedFiles        string `gorm:"column:changed_files"`
	NextSuggestedAction string `gorm:"column:next_suggested_action"`
	TimeCreated         string `gorm:"column:time_created;not null;index:session_summaries_session_created_idx"`
}

func (sessionSummaryRow) TableName() string { return "session_summaries" }

type sessionCheckpointRow struct {
	ID                  string `gorm:"primaryKey;column:id"`
	SessionID           string `gorm:"column:session_id;not null;index:session_checkpoints_session_created_idx"`
	Branch              string `gorm:"column:branch"`
	CommitSHA           string `gorm:"column:commit_sha"`
	ChangedFiles        string `gorm:"column:changed_files"`
	DiffSummary         string `gorm:"column:diff_summary"`
	ConversationSummary string `gorm:"column:conversation_summary"`
	OpenTodos           string `gorm:"column:open_todos"`
	KnownIssues         string `gorm:"column:known_issues"`
	NextSuggestedAction string `gorm:"column:next_suggested_action"`
	TimeCreated         string `gorm:"column:time_created;not null;index:session_checkpoints_session_created_idx"`
}

func (sessionCheckpointRow) TableName() string { return "session_checkpoints" }

type codingContextRow struct {
	ID             string `gorm:"primaryKey;column:id"`
	SessionID      string `gorm:"column:session_id;not null;uniqueIndex"`
	ProjectPath    string `gorm:"column:project_path;not null;index:coding_contexts_project_path_idx"`
	GitBranch      string `gorm:"column:git_branch"`
	CommitSHA      string `gorm:"column:commit_sha"`
	RepoURL        string `gorm:"column:repo_url"`
	ChangedFiles   string `gorm:"column:changed_files"`
	LanguageStack  string `gorm:"column:language_stack"`
	PackageManager string `gorm:"column:package_manager"`
	CWD            string `gorm:"column:cwd"`
	Permissions    string `gorm:"column:permissions"`
	LastCommand    string `gorm:"column:last_command"`
	TimeCreated    string `gorm:"column:time_created;not null"`
	TimeUpdated    string `gorm:"column:time_updated;not null"`
}

func (codingContextRow) TableName() string { return "coding_contexts" }

type agentRunRow struct {
	ID              string `gorm:"primaryKey;column:id"`
	ParentSessionID string `gorm:"column:parent_session_id;index:agent_runs_parent_session_idx"`
	SessionID       string `gorm:"column:session_id;index:agent_runs_session_idx"`
	Mode            string `gorm:"column:mode;not null;index:agent_runs_mode_idx"`
	Status          string `gorm:"column:status;not null;index:agent_runs_status_idx"`
	Prompt          string `gorm:"column:prompt"`
	Result          string `gorm:"column:result"`
	Error           string `gorm:"column:error"`
	Metadata        string `gorm:"column:metadata"`
	TimeCreated     string `gorm:"column:time_created;not null;index:agent_runs_status_idx"`
	TimeUpdated     string `gorm:"column:time_updated;not null"`
	TimeCompleted   string `gorm:"column:time_completed"`
}

func (agentRunRow) TableName() string { return "agent_runs" }

type todoItemRow struct {
	ID            string `gorm:"primaryKey;column:id"`
	SessionID     string `gorm:"column:session_id;index:todo_items_session_idx"`
	ProjectPath   string `gorm:"column:project_path;index:todo_items_project_idx"`
	Title         string `gorm:"column:title;not null"`
	Status        string `gorm:"column:status;not null;index:todo_items_status_idx"`
	Position      int    `gorm:"column:position;not null;default:0"`
	OwnerMode     string `gorm:"column:owner_mode"`
	SourceEventID string `gorm:"column:source_event_id"`
	Metadata      string `gorm:"column:metadata"`
	TimeCreated   string `gorm:"column:time_created;not null"`
	TimeUpdated   string `gorm:"column:time_updated;not null"`
}

func (todoItemRow) TableName() string { return "todo_items" }

type scheduledJobRow struct {
	ID              string `gorm:"primaryKey;column:id"`
	SessionID       string `gorm:"column:session_id;index:scheduled_jobs_session_idx"`
	Title           string `gorm:"column:title;not null"`
	Prompt          string `gorm:"column:prompt;not null"`
	Schedule        string `gorm:"column:schedule;not null"`
	WorkerMode      string `gorm:"column:worker_mode;not null"`
	Toolsets        string `gorm:"column:toolsets"`
	PermissionScope string `gorm:"column:permission_scope"`
	Status          string `gorm:"column:status;not null;index:scheduled_jobs_status_next_run_idx"`
	NextRunAt       string `gorm:"column:next_run_at;index:scheduled_jobs_status_next_run_idx"`
	LastRunAt       string `gorm:"column:last_run_at"`
	LastResult      string `gorm:"column:last_result"`
	LastError       string `gorm:"column:last_error"`
	Metadata        string `gorm:"column:metadata"`
	TimeCreated     string `gorm:"column:time_created;not null"`
	TimeUpdated     string `gorm:"column:time_updated;not null"`
}

func (scheduledJobRow) TableName() string { return "scheduled_jobs" }

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
