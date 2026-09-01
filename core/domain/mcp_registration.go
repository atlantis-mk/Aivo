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

type ResourceRegistrationFile struct {
	Path     string `json:"path"`
	Contents string `json:"contents"`
}

type ResourceRegistrationProposalInput struct {
	Kind         string                     `json:"kind"`
	ID           string                     `json:"id,omitempty"`
	Source       string                     `json:"source,omitempty"`
	Skill        string                     `json:"skill,omitempty"`
	URL          string                     `json:"url,omitempty"`
	Scope        string                     `json:"scope,omitempty"`
	ExpectedHash string                     `json:"expectedHash,omitempty"`
	Files        []ResourceRegistrationFile `json:"files,omitempty"`
}

type ResourceRegistrationResult struct {
	Kind        string   `json:"kind"`
	ID          string   `json:"id,omitempty"`
	Name        string   `json:"name"`
	Scope       string   `json:"scope"`
	Status      string   `json:"status"`
	FileCount   int      `json:"fileCount"`
	ContentHash string   `json:"contentHash"`
	SourceHash  string   `json:"sourceHash,omitempty"`
	Files       []string `json:"files,omitempty"`
}
