package persistence

type schemaVersionRow struct {
	Version   int    `gorm:"primaryKey;column:version"`
	AppliedAt string `gorm:"column:applied_at;not null"`
}

func (schemaVersionRow) TableName() string { return "schema_version" }

type appConfigRow struct {
	ID                    int    `gorm:"primaryKey;column:id"`
	Initialized           int    `gorm:"column:initialized;not null;default:0"`
	AppName               string `gorm:"column:app_name"`
	InitialWorkspacePath  string `gorm:"column:initial_workspace_path"`
	ProviderID            string `gorm:"column:provider_id"`
	ProviderType          string `gorm:"column:provider_type"`
	BaseURL               string `gorm:"column:base_url"`
	APIKeyEnv             string `gorm:"column:api_key_env"`
	Model                 string `gorm:"column:model"`
	AuxiliaryModel        string `gorm:"column:auxiliary_model"`
	ReasoningEffort       string `gorm:"column:reasoning_effort"`
	ServiceTier           string `gorm:"column:service_tier"`
	DefaultPermissionMode string `gorm:"column:default_permission_mode"`
	FallbackModels        string `gorm:"column:fallback_models"`
	ProviderPolicy        string `gorm:"column:provider_policy"`
	WebSearch             string `gorm:"column:web_search"`
	NativeTools           string `gorm:"column:native_tools"`
	Headers               string `gorm:"column:headers"`
	RequestParams         string `gorm:"column:request_params"`
	UpdatedAt             string `gorm:"column:updated_at;not null"`
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
	Description   string `gorm:"column:description"`
	RootPath      string `gorm:"column:root_path;not null;uniqueIndex"`
	GitBranch     string `gorm:"column:git_branch"`
	GitDirty      int    `gorm:"column:git_dirty;not null;default:0"`
	GitAvailable  int    `gorm:"column:git_available;not null;default:0"`
	SidebarHidden int    `gorm:"column:sidebar_hidden;not null;default:0;index:projects_sidebar_hidden_updated_idx"`
	TimeOpened    string `gorm:"column:time_opened;not null"`
	TimeUpdated   string `gorm:"column:time_updated;not null;index:projects_sidebar_hidden_updated_idx"`
}

func (projectRow) TableName() string { return "projects" }
