package domain

import "encoding/json"

const (
	ToolSourceBuiltin   = "builtin"
	ToolSourceMCP       = "mcp"
	ToolSourceExtension = "extension"
	ToolSourceBridge    = "bridge"
)

type ToolCatalogEntry struct {
	Name                 string              `json:"name"`
	Description          string              `json:"description,omitempty"`
	InputSchema          map[string]any      `json:"inputSchema,omitempty"`
	Namespace            string              `json:"namespace,omitempty"`
	NamespaceDescription string              `json:"namespaceDescription,omitempty"`
	Capability           string              `json:"capability,omitempty"`
	RiskLevel            string              `json:"riskLevel,omitempty"`
	Category             string              `json:"category,omitempty"`
	Toolsets             []string            `json:"toolsets,omitempty"`
	Source               string              `json:"source"`
	SourceID             string              `json:"sourceId,omitempty"`
	RegistrationID       string              `json:"registrationId,omitempty"`
	SchemaHash           string              `json:"schemaHash,omitempty"`
	Version              string              `json:"version,omitempty"`
	Enabled              bool                `json:"enabled"`
	ActivationPolicy     string              `json:"activationPolicy,omitempty"`
	SelectionGroup       *ToolSelectionGroup `json:"selectionGroup,omitempty"`
	ImplementationHash   string              `json:"implementationHash,omitempty"`
}

type ToolCatalogInput struct {
	WorkspaceRoot   string `json:"workspaceRoot,omitempty"`
	IncludeDeferred bool   `json:"includeDeferred,omitempty"`
	Source          string `json:"source,omitempty"`
}

type GlobalToolEnabledInput struct {
	Name          string `json:"name"`
	Enabled       bool   `json:"enabled"`
	WorkspaceRoot string `json:"workspaceRoot,omitempty"`
}

type ToolDescribeInput struct {
	Name          string `json:"name"`
	WorkspaceRoot string `json:"workspaceRoot,omitempty"`
}

type SessionActiveToolsInput struct {
	SessionID string   `json:"sessionId"`
	ToolNames []string `json:"toolNames,omitempty"`
}

type SessionActiveToolsResult struct {
	SessionID     string   `json:"sessionId"`
	ToolNames     []string `json:"toolNames"`
	CoreToolNames []string `json:"coreToolNames"`
}

type ToolRegistrationIdentity struct {
	Name               string `json:"name"`
	RegistrationID     string `json:"registrationId,omitempty"`
	SchemaHash         string `json:"schemaHash,omitempty"`
	Source             string `json:"source,omitempty"`
	SourceID           string `json:"sourceId,omitempty"`
	Version            string `json:"version,omitempty"`
	ImplementationHash string `json:"implementationHash,omitempty"`
}

type ToolSnapshotEntry struct {
	Name             string `json:"name"`
	RegistrationID   string `json:"registrationId"`
	SchemaHash       string `json:"schemaHash"`
	SourceID         string `json:"sourceId,omitempty"`
	SourceVersion    string `json:"sourceVersion,omitempty"`
	ActivationSource string `json:"activationSource"`
}

type ToolSnapshot struct {
	Revision string              `json:"revision"`
	Tools    []ToolSnapshotEntry `json:"tools"`
}

type ToolCatalogSnapshot struct {
	Entries    []ToolCatalogEntry                  `json:"entries"`
	Identities map[string]ToolRegistrationIdentity `json:"identities,omitempty"`
}

func CloneRawMap(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	raw, _ := json.Marshal(value)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out
}
