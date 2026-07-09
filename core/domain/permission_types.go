package domain

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
