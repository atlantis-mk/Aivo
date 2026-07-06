package domain

import (
	"errors"
	"strings"
)

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
	PermissionModeAutoApprove     = "auto_approve"
	PermissionModeFullAccess      = "full_access"
)

const legacyDesktopSessionSource = "w" + "ails"

type Session struct {
	ID                   string            `json:"id"`
	Type                 string            `json:"type"`
	Status               string            `json:"status"`
	Source               string            `json:"source"`
	Title                string            `json:"title"`
	Goal                 string            `json:"goal,omitempty"`
	Summary              string            `json:"summary,omitempty"`
	ParentSessionID      string            `json:"parentSessionId,omitempty"`
	ForkedFromSessionID  string            `json:"forkedFromSessionId,omitempty"`
	ProjectPath          string            `json:"projectPath,omitempty"`
	Model                *ModelRef         `json:"model,omitempty"`
	ModelSnapshot        string            `json:"modelSnapshot,omitempty"`
	SystemPromptSnapshot string            `json:"systemPromptSnapshot,omitempty"`
	AgentMode            string            `json:"agentMode,omitempty"`
	TokenCount           int               `json:"tokenCount,omitempty"`
	CostMicros           int64             `json:"costMicros,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
	ArchivedAt           string            `json:"archivedAt,omitempty"`
	DeletedAt            string            `json:"deletedAt,omitempty"`
	TimeCreated          string            `json:"timeCreated"`
	TimeUpdated          string            `json:"timeUpdated"`
}

type Turn struct {
	ID            string `json:"id"`
	SessionID     string `json:"sessionId"`
	AgentMode     string `json:"agentMode,omitempty"`
	Status        string `json:"status"`
	UserEventID   string `json:"userEventId,omitempty"`
	Error         string `json:"error,omitempty"`
	TimeCreated   string `json:"timeCreated"`
	TimeCompleted string `json:"timeCompleted,omitempty"`
	TimeUpdated   string `json:"timeUpdated"`
}

type SessionEvent struct {
	ID          string         `json:"id"`
	SessionID   string         `json:"sessionId"`
	TurnID      string         `json:"turnId,omitempty"`
	Type        string         `json:"type"`
	Role        string         `json:"role,omitempty"`
	Visibility  string         `json:"visibility"`
	Content     string         `json:"content,omitempty"`
	Payload     map[string]any `json:"payload,omitempty"`
	TokenCount  int            `json:"tokenCount,omitempty"`
	TimeCreated string         `json:"timeCreated"`
}

type ToolCall struct {
	ID            string         `json:"id"`
	SessionID     string         `json:"sessionId"`
	TurnID        string         `json:"turnId,omitempty"`
	EventID       string         `json:"eventId,omitempty"`
	Name          string         `json:"name"`
	Arguments     map[string]any `json:"arguments,omitempty"`
	Status        string         `json:"status"`
	ResultSummary string         `json:"resultSummary,omitempty"`
	Result        map[string]any `json:"result,omitempty"`
	Error         string         `json:"error,omitempty"`
	TimeCreated   string         `json:"timeCreated"`
	TimeUpdated   string         `json:"timeUpdated"`
}

type SessionExecutionState struct {
	ID              string         `json:"id"`
	SessionID       string         `json:"sessionId"`
	TurnID          string         `json:"turnId,omitempty"`
	Status          string         `json:"status"`
	Reason          string         `json:"reason,omitempty"`
	LastEventID     string         `json:"lastEventId,omitempty"`
	PendingInputIDs []string       `json:"pendingInputIds,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	TimeCreated     string         `json:"timeCreated"`
	TimeUpdated     string         `json:"timeUpdated"`
}

type PendingSessionInput struct {
	ID             string `json:"id"`
	SessionID      string `json:"sessionId"`
	TurnID         string `json:"turnId,omitempty"`
	Text           string `json:"text"`
	Delivery       string `json:"delivery"`
	Status         string `json:"status"`
	PromotedTurnID string `json:"promotedTurnId,omitempty"`
	TimeCreated    string `json:"timeCreated"`
	TimeUpdated    string `json:"timeUpdated"`
}

type InterruptSessionExecutionInput struct {
	SessionID string `json:"sessionId"`
	Reason    string `json:"reason,omitempty"`
}

type ResumeSessionExecutionInput struct {
	SessionID string `json:"sessionId"`
}

type CompactSessionContextInput struct {
	SessionID       string `json:"sessionId"`
	CharacterBudget int    `json:"characterBudget,omitempty"`
}

type CompactSessionContextResult struct {
	State            SessionExecutionState     `json:"state"`
	Summary          SessionSummary            `json:"summary"`
	Context          BuildSessionContextResult `json:"context"`
	CompactedEventID string                    `json:"compactedEventId,omitempty"`
}

type ListSessionEventsAfterCursorInput struct {
	SessionID        string `json:"sessionId"`
	Cursor           string `json:"cursor,omitempty"`
	IncludeNonNormal bool   `json:"includeNonNormal,omitempty"`
	Limit            int    `json:"limit,omitempty"`
}

type ListSessionEventsAfterCursorResult struct {
	Events     []SessionEvent `json:"events"`
	NextCursor string         `json:"nextCursor"`
}

type PermissionRequest struct {
	ID          string         `json:"id"`
	SessionID   string         `json:"sessionId,omitempty"`
	TurnID      string         `json:"turnId,omitempty"`
	ToolCallID  string         `json:"toolCallId,omitempty"`
	ToolName    string         `json:"toolName"`
	Action      string         `json:"action"`
	Paths       []string       `json:"paths,omitempty"`
	Arguments   map[string]any `json:"arguments,omitempty"`
	Status      string         `json:"status"`
	Remember    bool           `json:"remember,omitempty"`
	Reason      string         `json:"reason,omitempty"`
	TimeCreated string         `json:"timeCreated"`
	TimeUpdated string         `json:"timeUpdated"`
}

type QuestionOption struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

type QuestionPrompt struct {
	ID       string           `json:"id,omitempty"`
	Header   string           `json:"header,omitempty"`
	Question string           `json:"question"`
	Options  []QuestionOption `json:"options,omitempty"`
	Multiple bool             `json:"multiple,omitempty"`
}

type QuestionRequest struct {
	ID          string           `json:"id"`
	SessionID   string           `json:"sessionId,omitempty"`
	TurnID      string           `json:"turnId,omitempty"`
	ToolCallID  string           `json:"toolCallId,omitempty"`
	ToolName    string           `json:"toolName"`
	Questions   []QuestionPrompt `json:"questions"`
	Answers     [][]string       `json:"answers,omitempty"`
	Status      string           `json:"status"`
	Reason      string           `json:"reason,omitempty"`
	Arguments   map[string]any   `json:"arguments,omitempty"`
	TimeCreated string           `json:"timeCreated"`
	TimeUpdated string           `json:"timeUpdated"`
}

type ReplyQuestionRequestInput struct {
	RequestID string     `json:"requestId"`
	Answers   [][]string `json:"answers"`
}

type RejectQuestionRequestInput struct {
	RequestID string `json:"requestId"`
	Reason    string `json:"reason,omitempty"`
}

type PermissionRule struct {
	ID            string   `json:"id"`
	Scope         string   `json:"scope"`
	SessionID     string   `json:"sessionId,omitempty"`
	WorkspaceRoot string   `json:"workspaceRoot,omitempty"`
	ToolName      string   `json:"toolName"`
	Action        string   `json:"action"`
	Decision      string   `json:"decision"`
	Paths         []string `json:"paths,omitempty"`
	TimeCreated   string   `json:"timeCreated"`
	TimeUpdated   string   `json:"timeUpdated"`
}

type ApprovePermissionRequestInput struct {
	RequestID string `json:"requestId"`
	Remember  bool   `json:"remember,omitempty"`
}

type DenyPermissionRequestInput struct {
	RequestID string `json:"requestId"`
	Remember  bool   `json:"remember,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type PermissionModeInput struct {
	SessionID string `json:"sessionId"`
	Mode      string `json:"mode"`
}

type PermissionModeState struct {
	SessionID     string `json:"sessionId,omitempty"`
	WorkspaceRoot string `json:"workspaceRoot,omitempty"`
	Mode          string `json:"mode"`
	TimeUpdated   string `json:"timeUpdated,omitempty"`
}

type SessionSummary struct {
	ID                  string   `json:"id"`
	SessionID           string   `json:"sessionId"`
	FromEventID         string   `json:"fromEventId,omitempty"`
	ToEventID           string   `json:"toEventId,omitempty"`
	Summary             string   `json:"summary"`
	Facts               []string `json:"facts,omitempty"`
	Decisions           []string `json:"decisions,omitempty"`
	OpenTasks           []string `json:"openTasks,omitempty"`
	ChangedFiles        []string `json:"changedFiles,omitempty"`
	NextSuggestedAction string   `json:"nextSuggestedAction,omitempty"`
	TimeCreated         string   `json:"timeCreated"`
}

type SessionCheckpoint struct {
	ID                  string   `json:"id"`
	SessionID           string   `json:"sessionId"`
	Branch              string   `json:"branch,omitempty"`
	CommitSHA           string   `json:"commitSha,omitempty"`
	ChangedFiles        []string `json:"changedFiles,omitempty"`
	DiffSummary         string   `json:"diffSummary,omitempty"`
	ConversationSummary string   `json:"conversationSummary,omitempty"`
	OpenTodos           []string `json:"openTodos,omitempty"`
	KnownIssues         []string `json:"knownIssues,omitempty"`
	NextSuggestedAction string   `json:"nextSuggestedAction,omitempty"`
	TimeCreated         string   `json:"timeCreated"`
}

type CodingContext struct {
	ID             string   `json:"id"`
	SessionID      string   `json:"sessionId"`
	ProjectPath    string   `json:"projectPath"`
	GitBranch      string   `json:"gitBranch,omitempty"`
	CommitSHA      string   `json:"commitSha,omitempty"`
	RepoURL        string   `json:"repoUrl,omitempty"`
	ChangedFiles   []string `json:"changedFiles,omitempty"`
	LanguageStack  []string `json:"languageStack,omitempty"`
	PackageManager string   `json:"packageManager,omitempty"`
	CWD            string   `json:"cwd,omitempty"`
	Permissions    []string `json:"permissions,omitempty"`
	LastCommand    string   `json:"lastCommand,omitempty"`
	TimeCreated    string   `json:"timeCreated"`
	TimeUpdated    string   `json:"timeUpdated"`
}

type ResumeRecap struct {
	SessionID           string             `json:"sessionId"`
	Title               string             `json:"title"`
	Goal                string             `json:"goal,omitempty"`
	LatestSummary       *SessionSummary    `json:"latestSummary,omitempty"`
	ProjectPath         string             `json:"projectPath,omitempty"`
	Branch              string             `json:"branch,omitempty"`
	ChangedFiles        []string           `json:"changedFiles,omitempty"`
	OpenTodos           []string           `json:"openTodos,omitempty"`
	LastCommand         string             `json:"lastCommand,omitempty"`
	NextSuggestedAction string             `json:"nextSuggestedAction,omitempty"`
	UpdatedTime         string             `json:"updatedTime"`
	LatestCheckpoint    *SessionCheckpoint `json:"latestCheckpoint,omitempty"`
	RecentEvents        []SessionEvent     `json:"recentEvents,omitempty"`
}

type ContextSection struct {
	Name      string `json:"name"`
	Content   string `json:"content"`
	Truncated bool   `json:"truncated,omitempty"`
}

type BuildSessionContextResult struct {
	SessionID         string           `json:"sessionId"`
	Sections          []ContextSection `json:"sections"`
	EstimatedTokens   int              `json:"estimatedTokens"`
	CharacterBudget   int              `json:"characterBudget,omitempty"`
	TruncatedSections []string         `json:"truncatedSections,omitempty"`
}

type SubmitSessionMessageRequest struct {
	SessionID       string              `json:"sessionId"`
	Text            string              `json:"text"`
	Attachments     []MessageAttachment `json:"attachments,omitempty"`
	Delivery        string              `json:"delivery,omitempty"`
	Model           *ModelRef           `json:"model,omitempty"`
	AgentMode       string              `json:"agentMode,omitempty"`
	Toolsets        []string            `json:"toolsets,omitempty"`
	PermissionScope string              `json:"permissionScope,omitempty"`
	ReasoningEffort string              `json:"reasoningEffort,omitempty"`
	ServiceTier     string              `json:"serviceTier,omitempty"`
}

type PreparedSessionTurn struct {
	Turn           Turn          `json:"turn"`
	Model          *ModelRef     `json:"model,omitempty"`
	UserEvent      SessionEvent  `json:"userEvent"`
	AssistantEvent *SessionEvent `json:"assistantEvent,omitempty"`
}

type CreateSessionRequest struct {
	Type                 string            `json:"type,omitempty"`
	Source               string            `json:"source,omitempty"`
	Title                string            `json:"title,omitempty"`
	Goal                 string            `json:"goal,omitempty"`
	ProjectPath          string            `json:"projectPath,omitempty"`
	Model                *ModelRef         `json:"model,omitempty"`
	ModelSnapshot        string            `json:"modelSnapshot,omitempty"`
	SystemPromptSnapshot string            `json:"systemPromptSnapshot,omitempty"`
	AgentMode            string            `json:"agentMode,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
}

type ListSessionsRequest struct {
	Type           string `json:"type,omitempty"`
	Status         string `json:"status,omitempty"`
	Source         string `json:"source,omitempty"`
	ProjectPath    string `json:"projectPath,omitempty"`
	Search         string `json:"search,omitempty"`
	IncludeDeleted bool   `json:"includeDeleted,omitempty"`
	Limit          int    `json:"limit,omitempty"`
}

type UpdateSessionRequest struct {
	SessionID string `json:"sessionId"`
	Title     string `json:"title,omitempty"`
	Goal      string `json:"goal,omitempty"`
	Summary   string `json:"summary,omitempty"`
	Status    string `json:"status,omitempty"`
}

type AppendEventRequest struct {
	SessionID  string         `json:"sessionId"`
	TurnID     string         `json:"turnId,omitempty"`
	Type       string         `json:"type"`
	Role       string         `json:"role,omitempty"`
	Visibility string         `json:"visibility,omitempty"`
	Content    string         `json:"content,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
	TokenCount int            `json:"tokenCount,omitempty"`
}

type UpdateSessionEventRequest struct {
	EventID string `json:"eventId"`
	Content string `json:"content"`
}

type DeleteSessionEventRequest struct {
	EventID string `json:"eventId"`
}

type StartTurnRequest struct {
	SessionID   string `json:"sessionId"`
	UserEventID string `json:"userEventId,omitempty"`
	AgentMode   string `json:"agentMode,omitempty"`
}

type CompleteTurnRequest struct {
	TurnID string `json:"turnId"`
}

type FailTurnRequest struct {
	TurnID string `json:"turnId"`
	Error  string `json:"error,omitempty"`
}

type CancelTurnRequest struct {
	TurnID string `json:"turnId"`
	Reason string `json:"reason,omitempty"`
}

type CreateToolCallRequest struct {
	ID            string         `json:"id,omitempty"`
	SessionID     string         `json:"sessionId"`
	TurnID        string         `json:"turnId,omitempty"`
	EventID       string         `json:"eventId,omitempty"`
	Name          string         `json:"name"`
	Arguments     map[string]any `json:"arguments,omitempty"`
	Status        string         `json:"status,omitempty"`
	ResultSummary string         `json:"resultSummary,omitempty"`
	Result        map[string]any `json:"result,omitempty"`
	Error         string         `json:"error,omitempty"`
}

type ReplaySessionToolCallRequest struct {
	SessionID       string `json:"sessionId,omitempty"`
	ToolCallID      string `json:"toolCallId"`
	PermissionScope string `json:"permissionScope,omitempty"`
}

type CreateSummaryRequest struct {
	SessionID           string   `json:"sessionId"`
	FromEventID         string   `json:"fromEventId,omitempty"`
	ToEventID           string   `json:"toEventId,omitempty"`
	Summary             string   `json:"summary,omitempty"`
	Facts               []string `json:"facts,omitempty"`
	Decisions           []string `json:"decisions,omitempty"`
	OpenTasks           []string `json:"openTasks,omitempty"`
	ChangedFiles        []string `json:"changedFiles,omitempty"`
	NextSuggestedAction string   `json:"nextSuggestedAction,omitempty"`
}

type CreateCheckpointRequest struct {
	SessionID           string   `json:"sessionId"`
	DiffSummary         string   `json:"diffSummary,omitempty"`
	ConversationSummary string   `json:"conversationSummary,omitempty"`
	OpenTodos           []string `json:"openTodos,omitempty"`
	KnownIssues         []string `json:"knownIssues,omitempty"`
	NextSuggestedAction string   `json:"nextSuggestedAction,omitempty"`
}

type ForkSessionRequest struct {
	SessionID string `json:"sessionId"`
	Title     string `json:"title,omitempty"`
	Goal      string `json:"goal,omitempty"`
}

type ResumeSessionRequest struct {
	SessionID   string `json:"sessionId,omitempty"`
	ProjectPath string `json:"projectPath,omitempty"`
}

type BuildSessionContextRequest struct {
	SessionID       string `json:"sessionId"`
	CurrentInput    string `json:"currentInput,omitempty"`
	MaxTokens       int    `json:"maxTokens,omitempty"`
	CharacterBudget int    `json:"characterBudget,omitempty"`
}

type RetrySessionTurnRequest struct {
	SessionID       string    `json:"sessionId,omitempty"`
	TurnID          string    `json:"turnId"`
	Model           *ModelRef `json:"model,omitempty"`
	AgentMode       string    `json:"agentMode,omitempty"`
	Toolsets        []string  `json:"toolsets,omitempty"`
	PermissionScope string    `json:"permissionScope,omitempty"`
	ReasoningEffort string    `json:"reasoningEffort,omitempty"`
	ServiceTier     string    `json:"serviceTier,omitempty"`
}

type GetSessionTurnDiffRequest struct {
	SessionID string `json:"sessionId,omitempty"`
	TurnID    string `json:"turnId"`
}

type ApplySessionTurnFileStateRequest struct {
	SessionID   string `json:"sessionId,omitempty"`
	TurnID      string `json:"turnId"`
	ToolCallID  string `json:"toolCallId,omitempty"`
	Path        string `json:"path,omitempty"`
	TargetState string `json:"targetState"`
}

type SessionTurnDiff struct {
	SessionID string                `json:"sessionId"`
	TurnID    string                `json:"turnId"`
	Files     []SessionTurnDiffFile `json:"files"`
	Diff      string                `json:"diff,omitempty"`
}

type SessionTurnDiffFile struct {
	ToolCallID      string `json:"toolCallId"`
	ToolName        string `json:"toolName"`
	Path            string `json:"path"`
	MovePath        string `json:"movePath,omitempty"`
	Type            string `json:"type"`
	Additions       int    `json:"additions"`
	Deletions       int    `json:"deletions"`
	Diff            string `json:"diff,omitempty"`
	BaseHash        string `json:"baseHash,omitempty"`
	CurrentHash     string `json:"currentHash,omitempty"`
	CurrentFileHash string `json:"currentFileHash,omitempty"`
	Revertible      bool   `json:"revertible"`
	Unrevertible    bool   `json:"unrevertible"`
	Reason          string `json:"reason,omitempty"`
	TimeUpdated     string `json:"timeUpdated,omitempty"`
}

func NormalizeSessionType(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", SessionTypeGeneric:
		return SessionTypeGeneric, nil
	case SessionTypeCoding:
		return SessionTypeCoding, nil
	default:
		return "", errors.New("invalid session type")
	}
}

func NormalizeSessionStatus(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", SessionStatusActive:
		return SessionStatusActive, nil
	case SessionStatusArchived:
		return SessionStatusArchived, nil
	case SessionStatusDeleted:
		return SessionStatusDeleted, nil
	default:
		return "", errors.New("invalid session status")
	}
}

func NormalizeSessionSource(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", SessionSourceDesktop:
		return SessionSourceDesktop, nil
	case SessionSourceWeb:
		return strings.TrimSpace(value), nil
	case legacyDesktopSessionSource:
		return SessionSourceDesktop, nil
	default:
		return "", errors.New("invalid session source")
	}
}

func ValidateTurnStatus(value string) error {
	switch value {
	case TurnStatusRunning, TurnStatusCompleted, TurnStatusFailed, TurnStatusCancelled:
		return nil
	default:
		return errors.New("invalid turn status")
	}
}

func ValidateEventType(value string) error {
	switch value {
	case EventTypeUserMessage, EventTypeAssistantMessage, EventTypeToolCall, EventTypeToolResult, EventTypeFileRead, EventTypeFileWrite, EventTypeFilePatch, EventTypeShellCommand, EventTypeShellOutput, EventTypeGitDiff, EventTypePlanUpdate, EventTypeSummary, EventTypeCheckpoint, EventTypeError, EventTypeSystemNote:
		return nil
	default:
		return errors.New("invalid event type")
	}
}

func NormalizeEventRole(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", EventRoleSystem:
		return strings.TrimSpace(value), nil
	case EventRoleUser, EventRoleAssistant, EventRoleTool:
		return strings.TrimSpace(value), nil
	default:
		return "", errors.New("invalid event role")
	}
}

func NormalizeEventVisibility(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", EventVisibilityNormal:
		return EventVisibilityNormal, nil
	case EventVisibilityHidden, EventVisibilityInternal, EventVisibilityRedacted:
		return strings.TrimSpace(value), nil
	default:
		return "", errors.New("invalid event visibility")
	}
}

func NormalizeToolCallStatus(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", ToolCallStatusRunning:
		return ToolCallStatusRunning, nil
	case ToolCallStatusSuccess, ToolCallStatusFailed, ToolCallStatusPending, ToolCallStatusInterrupted:
		return strings.TrimSpace(value), nil
	default:
		return "", errors.New("invalid tool call status")
	}
}

func NormalizeExecutionStatus(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", ExecutionStatusIdle:
		return ExecutionStatusIdle, nil
	case ExecutionStatusRunning, ExecutionStatusInterrupted, ExecutionStatusFailed, ExecutionStatusCompacting:
		return strings.TrimSpace(value), nil
	default:
		return "", errors.New("invalid execution status")
	}
}

func NormalizeInputDelivery(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", InputDeliveryImmediate:
		return InputDeliveryImmediate, nil
	case InputDeliverySteer, InputDeliveryQueue:
		return strings.TrimSpace(value), nil
	default:
		return "", errors.New("invalid input delivery")
	}
}
