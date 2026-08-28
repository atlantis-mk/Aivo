package domain

const (
	ExtensionRuntimeBuiltin  = "builtin"
	ExtensionRuntimeProcess  = "process"
	ExtensionRuntimeService  = "service"
	ExtensionRuntimeExternal = "external"
	ExtensionRuntimeStatic   = "static"

	ExtensionInstallModeLinked  = "linked"
	ExtensionInstallModeManaged = "managed"

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
	Permissions   []string               `json:"permissions,omitempty"`
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
	ToolGroups  []ExtensionToolGroupContribution  `json:"toolGroups,omitempty"`
	Views       []ExtensionViewContribution       `json:"views,omitempty"`
	Contexts    []ExtensionContextContribution    `json:"contexts,omitempty"`
	Policies    []string                          `json:"policies,omitempty"`
	Environment *ExtensionEnvironmentContribution `json:"environment,omitempty"`
}

type ExtensionToolGroupContribution struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tools       []string `json:"tools"`
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

type ExtensionInstallSummary struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Version       string   `json:"version"`
	Description   string   `json:"description,omitempty"`
	APIVersion    string   `json:"apiVersion"`
	RuntimeType   string   `json:"runtimeType"`
	Transport     string   `json:"transport,omitempty"`
	Command       string   `json:"command,omitempty"`
	Permissions   []string `json:"permissions,omitempty"`
	CredentialIDs []string `json:"credentialIds,omitempty"`
	Platforms     []string `json:"platforms,omitempty"`
	Network       bool     `json:"network,omitempty"`
	Tools         []string `json:"tools,omitempty"`
	Views         []string `json:"views,omitempty"`
	Contexts      []string `json:"contexts,omitempty"`
	Policies      []string `json:"policies,omitempty"`
	Executable    bool     `json:"executable"`
}

func ExtensionSummary(manifest ExtensionManifest) ExtensionInstallSummary {
	summary := ExtensionInstallSummary{
		ID: manifest.ID, Name: manifest.Name, Version: manifest.Version, Description: manifest.Description,
		APIVersion: manifest.APIVersion, RuntimeType: manifest.Runtime.Type, Transport: manifest.Runtime.Transport,
		Command: manifest.Runtime.Command, Permissions: append([]string(nil), manifest.Permissions...),
		CredentialIDs: append([]string(nil), manifest.Requirements.Credentials...), Platforms: append([]string(nil), manifest.Requirements.Platforms...),
		Network: manifest.Requirements.Network, Policies: append([]string(nil), manifest.Contributes.Policies...),
		Executable: manifest.Runtime.Type == ExtensionRuntimeProcess || manifest.Runtime.Type == ExtensionRuntimeService,
	}
	for _, item := range manifest.Contributes.Tools {
		summary.Tools = append(summary.Tools, item.Name)
	}
	for _, item := range manifest.Contributes.Views {
		summary.Views = append(summary.Views, item.ID)
	}
	for _, item := range manifest.Contributes.Contexts {
		summary.Contexts = append(summary.Contexts, item.ID)
	}
	return summary
}

type ExtensionInstallPreview struct {
	Path         string                  `json:"path"`
	ManifestPath string                  `json:"manifestPath"`
	Integrity    string                  `json:"integrity"`
	Summary      ExtensionInstallSummary `json:"summary"`
	Update       bool                    `json:"update"`
}

type ExtensionInstall struct {
	ID           string                  `json:"id"`
	Manifest     ExtensionManifest       `json:"-"`
	Summary      ExtensionInstallSummary `json:"summary"`
	InstallMode  string                  `json:"installMode"`
	RootPath     string                  `json:"rootPath"`
	ManifestPath string                  `json:"manifestPath"`
	Integrity    string                  `json:"integrity"`
	Enabled      bool                    `json:"enabled"`
	Status       string                  `json:"status"`
	Error        string                  `json:"error,omitempty"`
	TimeCreated  string                  `json:"timeCreated"`
	TimeUpdated  string                  `json:"timeUpdated"`
}

type PreviewExtensionInstallInput struct {
	Path string `json:"path"`
}

type InstallExtensionInput struct {
	Path      string `json:"path"`
	Integrity string `json:"integrity"`
	Enable    bool   `json:"enable"`
}

type SetExtensionEnabledInput struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
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
	Permissions  []string `json:"permissions,omitempty"`
	CSP          string   `json:"csp"`
}

type ExtensionToolViewRef struct {
	ExtensionID string `json:"extensionId"`
	ViewID      string `json:"viewId"`
	Surface     string `json:"surface"`
	Title       string `json:"title,omitempty"`
}
