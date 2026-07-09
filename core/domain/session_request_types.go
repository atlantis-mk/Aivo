package domain

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
