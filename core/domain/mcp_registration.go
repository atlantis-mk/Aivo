package domain

type MCPRegistrationProposalInput struct {
	ID                    string   `json:"id"`
	DisplayName           string   `json:"displayName"`
	Description           string   `json:"description,omitempty"`
	Transport             string   `json:"transport"`
	Command               string   `json:"command,omitempty"`
	Args                  []string `json:"args,omitempty"`
	CWD                   string   `json:"cwd,omitempty"`
	URL                   string   `json:"url,omitempty"`
	Roots                 []string `json:"roots,omitempty"`
	AuthType              string   `json:"authType,omitempty"`
	BearerTokenEnv        string   `json:"bearerTokenEnv,omitempty"`
	TimeoutSeconds        int      `json:"timeoutSeconds,omitempty"`
	ConnectTimeoutSeconds int      `json:"connectTimeoutSeconds,omitempty"`
}

type MCPRegistrationResult struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"displayName"`
	Transport   string   `json:"transport"`
	Status      string   `json:"status"`
	ToolNames   []string `json:"toolNames,omitempty"`
}
