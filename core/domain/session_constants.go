package domain

const (
	SessionTypeCoding  = "coding"
	SessionTypeGeneric = "generic"

	SessionStatusActive   = "active"
	SessionStatusArchived = "archived"
	SessionStatusDeleted  = "deleted"

	SessionSourceWeb     = "web"
	SessionSourceDesktop = "desktop"

	TurnStatusRunning   = "running"
	TurnStatusCompleted = "completed"
	TurnStatusFailed    = "failed"
	TurnStatusCancelled = "cancelled"

	EventTypeUserMessage      = "user_message"
	EventTypeAssistantMessage = "assistant_message"
	EventTypeToolCall         = "tool_call"
	EventTypeToolResult       = "tool_result"
	EventTypeFileRead         = "file_read"
	EventTypeFileWrite        = "file_write"
	EventTypeFilePatch        = "file_patch"
	EventTypeShellCommand     = "shell_command"
	EventTypeShellOutput      = "shell_output"
	EventTypeGitDiff          = "git_diff"
	EventTypePlanUpdate       = "plan_update"
	EventTypeSummary          = "summary"
	EventTypeCheckpoint       = "checkpoint"
	EventTypeError            = "error"
	EventTypeSystemNote       = "system_note"

	EventRoleUser      = "user"
	EventRoleAssistant = "assistant"
	EventRoleTool      = "tool"
	EventRoleSystem    = "system"

	EventVisibilityNormal   = "normal"
	EventVisibilityHidden   = "hidden"
	EventVisibilityInternal = "internal"
	EventVisibilityRedacted = "redacted"

	ToolCallStatusRunning     = "running"
	ToolCallStatusSuccess     = "success"
	ToolCallStatusFailed      = "failed"
	ToolCallStatusPending     = "pending_approval"
	ToolCallStatusInterrupted = "interrupted"

	ExecutionStatusIdle        = "idle"
	ExecutionStatusRunning     = "running"
	ExecutionStatusInterrupted = "interrupted"
	ExecutionStatusFailed      = "failed"
	ExecutionStatusCompacting  = "compacting"

	InputDeliveryImmediate = "immediate"
	InputDeliverySteer     = "steer"
	InputDeliveryQueue     = "queue"

	PendingInputStatusPending  = "pending"
	PendingInputStatusPromoted = "promoted"
	PendingInputStatusCanceled = "canceled"

	PermissionDecisionAllow = "allow"
	PermissionDecisionAsk   = "ask"
	PermissionDecisionDeny  = "deny"

	PermissionRequestStatusPending  = "pending"
	PermissionRequestStatusApproved = "approved"
	PermissionRequestStatusDenied   = "denied"

	QuestionRequestStatusPending  = "pending"
	QuestionRequestStatusAnswered = "answered"
	QuestionRequestStatusRejected = "rejected"

	PermissionModeRequestApproval = "request_approval"
	PermissionModeFullAccess      = "full_access"
)

const legacyDesktopSessionSource = "w" + "ails"
