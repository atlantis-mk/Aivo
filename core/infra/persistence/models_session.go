package persistence

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

type gitWorktreeRow struct {
	ID             string `gorm:"primaryKey;column:id"`
	RepositoryRoot string `gorm:"column:repository_root;not null;index:git_worktrees_repo_status_idx"`
	Path           string `gorm:"column:path;not null;uniqueIndex"`
	Branch         string `gorm:"column:branch"`
	BaseRef        string `gorm:"column:base_ref"`
	Head           string `gorm:"column:head"`
	Status         string `gorm:"column:status;not null;index:git_worktrees_repo_status_idx"`
	Managed        int    `gorm:"column:managed;not null;default:1"`
	OwnsBranch     int    `gorm:"column:owns_branch;not null;default:0"`
	Detached       int    `gorm:"column:detached;not null;default:0"`
	Error          string `gorm:"column:error"`
	TimeCreated    string `gorm:"column:time_created;not null"`
	TimeUpdated    string `gorm:"column:time_updated;not null"`
}

func (gitWorktreeRow) TableName() string { return "git_worktrees" }

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
