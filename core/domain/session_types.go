package domain

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
