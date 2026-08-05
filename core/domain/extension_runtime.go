package domain

const (
	ExtensionRuntimeBuiltin  = "builtin"
	ExtensionRuntimeProcess  = "process"
	ExtensionRuntimeService  = "service"
	ExtensionRuntimeExternal = "external"
	ExtensionRuntimeStatic   = "static"

	ExtensionStateDiscovered = "discovered"
	ExtensionStateValidated  = "validated"
	ExtensionStateUntrusted  = "untrusted"
	ExtensionStateEnabled    = "enabled"
	ExtensionStateStarting   = "starting"
	ExtensionStateReady      = "ready"
	ExtensionStateActive     = "active"
	ExtensionStateDraining   = "draining"
	ExtensionStateStopped    = "stopped"
	ExtensionStateError      = "error"
)

type ExtensionManifest struct {
	SchemaVersion int                    `json:"schemaVersion"`
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Version       string                 `json:"version"`
	Description   string                 `json:"description,omitempty"`
	APIVersion    string                 `json:"apiVersion"`
	Runtime       ExtensionRuntime       `json:"runtime"`
	Contributes   ExtensionContributions `json:"contributes,omitempty"`
	Requirements  ExtensionRequirements  `json:"requirements,omitempty"`
}

type ExtensionRuntime struct {
	Type      string   `json:"type"`
	Command   string   `json:"command,omitempty"`
	Args      []string `json:"args,omitempty"`
	Transport string   `json:"transport,omitempty"`
	URL       string   `json:"url,omitempty"`
}

type ExtensionContributions struct {
	Tools       []ExtensionToolContribution       `json:"tools,omitempty"`
	Views       []ExtensionViewContribution       `json:"views,omitempty"`
	Contexts    []ExtensionContextContribution    `json:"contexts,omitempty"`
	Policies    []string                          `json:"policies,omitempty"`
	Environment *ExtensionEnvironmentContribution `json:"environment,omitempty"`
}

type ExtensionToolContribution struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Schema      any    `json:"schema"`
	Activation  string `json:"activation,omitempty"`
	Capability  string `json:"capability,omitempty"`
}

type ExtensionViewContribution struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Type     string   `json:"type"`
	Route    string   `json:"route"`
	Surfaces []string `json:"surfaces"`
	Tools    []string `json:"tools,omitempty"`
	Actions  []string `json:"actions,omitempty"`
}

type ExtensionContextContribution struct {
	ID   string `json:"id"`
	Path string `json:"path"`
	Kind string `json:"kind"`
}

type ExtensionEnvironmentContribution struct {
	ID string `json:"id"`
}

type ExtensionRequirements struct {
	Network     bool     `json:"network,omitempty"`
	Credentials []string `json:"credentials,omitempty"`
	Platforms   []string `json:"platforms,omitempty"`
}

type ExtensionStatus struct {
	ID        string `json:"id"`
	Version   string `json:"version"`
	State     string `json:"state"`
	Trusted   bool   `json:"trusted"`
	Enabled   bool   `json:"enabled"`
	Integrity string `json:"integrity"`
	Error     string `json:"error,omitempty"`
}

type DiscoverExtensionInput struct {
	Path string `json:"path"`
}
type TrustExtensionInput struct {
	ID        string `json:"id"`
	Integrity string `json:"integrity"`
}
type ExtensionControlInput struct {
	ID string `json:"id"`
}
type ResolveExtensionViewInput struct {
	ID     string `json:"id"`
	ViewID string `json:"viewId"`
}
type ExtensionViewActionInput struct {
	ID     string `json:"id"`
	ViewID string `json:"viewId"`
	Action string `json:"action"`
	Data   any    `json:"data,omitempty"`
}
type BindExtensionCredentialInput struct {
	ID        string `json:"id"`
	Slot      string `json:"slot"`
	SecretRef string `json:"secretRef"`
}
type ExtensionCredentialBinding struct {
	ID    string `json:"id"`
	Slot  string `json:"slot"`
	Bound bool   `json:"bound"`
}

type ExtensionContextResource struct {
	ExtensionID string `json:"extensionId"`
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Content     string `json:"content"`
	SHA256      string `json:"sha256"`
}

type ExtensionViewDescriptor struct {
	ExtensionID  string   `json:"extensionId"`
	ViewID       string   `json:"viewId"`
	Title        string   `json:"title,omitempty"`
	LogicalURL   string   `json:"logicalUrl"`
	BackendURL   string   `json:"backendUrl"`
	BackendToken string   `json:"backendToken,omitempty"`
	Surface      []string `json:"surfaces"`
	Actions      []string `json:"actions,omitempty"`
	CSP          string   `json:"csp"`
}
