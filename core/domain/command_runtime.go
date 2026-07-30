package domain

type CommandCatalogInput struct {
	ProjectPath string `json:"projectPath,omitempty"`
}

type CommandCatalogEntry struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Source      string            `json:"source"`
	SourceID    string            `json:"sourceId,omitempty"`
	Arguments   []CommandArgument `json:"arguments,omitempty"`
	Agent       string            `json:"agent,omitempty"`
	Model       *ModelRef         `json:"model,omitempty"`
	Toolsets    []string          `json:"toolsets,omitempty"`
	Subtask     bool              `json:"subtask,omitempty"`
}

type InvokeCommandInput struct {
	SessionID   string            `json:"sessionId,omitempty"`
	ProjectPath string            `json:"projectPath,omitempty"`
	CommandID   string            `json:"commandId"`
	Arguments   map[string]string `json:"arguments,omitempty"`
}

type InvokeCommandResult struct {
	CommandID      string            `json:"commandId"`
	Source         string            `json:"source"`
	SourceID       string            `json:"sourceId,omitempty"`
	Prompt         string            `json:"prompt"`
	Agent          string            `json:"agent,omitempty"`
	Model          *ModelRef         `json:"model,omitempty"`
	Toolsets       []string          `json:"toolsets,omitempty"`
	Subtask        bool              `json:"subtask,omitempty"`
	Provenance     map[string]string `json:"provenance,omitempty"`
	ChildSessionID string            `json:"childSessionId,omitempty"`
	AgentRunID     string            `json:"agentRunId,omitempty"`
	Response       string            `json:"response,omitempty"`
}
