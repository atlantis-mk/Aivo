package domain

const (
	MCPTransportStdio          = "stdio"
	MCPTransportStreamableHTTP = "streamable_http"
	MCPTransportSSE            = "sse"

	MCPServerStatusEnabled  = "enabled"
	MCPServerStatusDisabled = "disabled"
	MCPServerStatusError    = "error"
	MCPServerStatusReady    = "ready"

	MCPAuthNone   = "none"
	MCPAuthBearer = "bearer"
	MCPAuthOAuth  = "oauth"

	MCPDiagnosticInfo  = "info"
	MCPDiagnosticWarn  = "warn"
	MCPDiagnosticError = "error"
)

type MCPDiagnostic struct {
	ID          string         `json:"id"`
	ServerID    string         `json:"serverId,omitempty"`
	Level       string         `json:"level"`
	Message     string         `json:"message"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	TimeCreated string         `json:"timeCreated"`
}

type MCPServerConfig struct {
	ID                    string            `json:"id"`
	Name                  string            `json:"name"`
	DisplayName           string            `json:"displayName,omitempty"`
	Description           string            `json:"description,omitempty"`
	Transport             string            `json:"transport"`
	Command               string            `json:"command,omitempty"`
	Args                  []string          `json:"args,omitempty"`
	CWD                   string            `json:"cwd,omitempty"`
	Env                   map[string]string `json:"env,omitempty"`
	URL                   string            `json:"url,omitempty"`
	Headers               map[string]string `json:"headers,omitempty"`
	AuthType              string            `json:"authType,omitempty"`
	BearerTokenEnv        string            `json:"bearerTokenEnv,omitempty"`
	OAuthIssuerURL        string            `json:"oauthIssuerUrl,omitempty"`
	OAuthClientID         string            `json:"oauthClientId,omitempty"`
	OAuthScopes           []string          `json:"oauthScopes,omitempty"`
	OAuthAccessTokenRef   string            `json:"oauthAccessTokenRef,omitempty"`
	OAuthRefreshTokenRef  string            `json:"oauthRefreshTokenRef,omitempty"`
	OAuthExpiresAt        string            `json:"oauthExpiresAt,omitempty"`
	OAuthAccessToken      string            `json:"-"`
	OAuthRefreshToken     string            `json:"-"`
	Roots                 []string          `json:"roots,omitempty"`
	TimeoutSeconds        int               `json:"timeoutSeconds,omitempty"`
	ConnectTimeoutSeconds int               `json:"connectTimeoutSeconds,omitempty"`
	Enabled               bool              `json:"enabled"`
	Status                string            `json:"status,omitempty"`
	Error                 string            `json:"error,omitempty"`
	TimeCreated           string            `json:"timeCreated,omitempty"`
	TimeUpdated           string            `json:"timeUpdated,omitempty"`
}

type MCPToolRecord struct {
	ID          string         `json:"id"`
	ServerID    string         `json:"serverId"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
	Capability  string         `json:"capability,omitempty"`
	RiskLevel   string         `json:"riskLevel,omitempty"`
	TimeUpdated string         `json:"timeUpdated"`
}

type MCPPromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type MCPPromptRecord struct {
	ID          string              `json:"id"`
	ServerID    string              `json:"serverId"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Arguments   []MCPPromptArgument `json:"arguments,omitempty"`
	TimeUpdated string              `json:"timeUpdated"`
}

type MCPResourceRecord struct {
	ID          string `json:"id"`
	ServerID    string `json:"serverId"`
	URI         string `json:"uri,omitempty"`
	URITemplate string `json:"uriTemplate,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
	Template    bool   `json:"template,omitempty"`
	TimeUpdated string `json:"timeUpdated"`
}

type MCPServerListInput struct {
	IncludeDisabled bool `json:"includeDisabled,omitempty"`
	IncludeTools    bool `json:"includeTools,omitempty"`
}

type MCPServerListItem struct {
	Server            MCPServerConfig     `json:"server"`
	Tools             []MCPToolRecord     `json:"tools,omitempty"`
	Prompts           []MCPPromptRecord   `json:"prompts,omitempty"`
	Resources         []MCPResourceRecord `json:"resources,omitempty"`
	ResourceTemplates []MCPResourceRecord `json:"resourceTemplates,omitempty"`
	Diagnostics       []MCPDiagnostic     `json:"diagnostics,omitempty"`
}

type SaveMCPServerInput struct {
	Server MCPServerConfig `json:"server"`
}

type SetMCPServerEnabledInput struct {
	ServerID string `json:"serverId"`
	Enabled  bool   `json:"enabled"`
}

type MCPProbeInput struct {
	ServerID string          `json:"serverId,omitempty"`
	Server   MCPServerConfig `json:"server,omitempty"`
}

type MCPProbeResult struct {
	OK                bool                `json:"ok"`
	ServerID          string              `json:"serverId,omitempty"`
	Status            string              `json:"status,omitempty"`
	Error             string              `json:"error,omitempty"`
	Tools             []MCPToolRecord     `json:"tools,omitempty"`
	Prompts           []MCPPromptRecord   `json:"prompts,omitempty"`
	Resources         []MCPResourceRecord `json:"resources,omitempty"`
	ResourceTemplates []MCPResourceRecord `json:"resourceTemplates,omitempty"`
	Diagnostics       []MCPDiagnostic     `json:"diagnostics,omitempty"`
}

type MCPPromptGetInput struct {
	ServerID  string            `json:"serverId"`
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

type MCPContentBlock struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	URI      string `json:"uri,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Blob     string `json:"blob,omitempty"`
}

type MCPPromptMessage struct {
	Role    string            `json:"role"`
	Content []MCPContentBlock `json:"content,omitempty"`
}

type MCPPromptGetResult struct {
	ServerID    string             `json:"serverId"`
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Messages    []MCPPromptMessage `json:"messages,omitempty"`
	Content     string             `json:"content,omitempty"`
	Structured  map[string]any     `json:"structured,omitempty"`
}

type MCPResourceReadInput struct {
	ServerID string `json:"serverId"`
	URI      string `json:"uri"`
}

type MCPResourceContent struct {
	URI      string `json:"uri,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}

type MCPResourceReadResult struct {
	ServerID   string               `json:"serverId"`
	URI        string               `json:"uri"`
	Contents   []MCPResourceContent `json:"contents,omitempty"`
	Content    string               `json:"content,omitempty"`
	Structured map[string]any       `json:"structured,omitempty"`
}

type InsertMCPPromptIntoSessionInput struct {
	SessionID string            `json:"sessionId"`
	ServerID  string            `json:"serverId"`
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

type InsertMCPResourceIntoSessionInput struct {
	SessionID string `json:"sessionId"`
	ServerID  string `json:"serverId"`
	URI       string `json:"uri"`
}

type MCPServerLogInput struct {
	ServerID string `json:"serverId"`
	Offset   int    `json:"offset,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Tail     bool   `json:"tail,omitempty"`
}

type MCPServerLogResult struct {
	ServerID   string `json:"serverId"`
	Content    string `json:"content"`
	Offset     int    `json:"offset"`
	NextOffset int    `json:"nextOffset"`
	Size       int    `json:"size"`
	Truncated  bool   `json:"truncated,omitempty"`
}

type MCPOAuthDiscoveryInput struct {
	ServerID string          `json:"serverId,omitempty"`
	Server   MCPServerConfig `json:"server,omitempty"`
}

type MCPOAuthDiscoveryResult struct {
	ServerID                 string         `json:"serverId,omitempty"`
	Resource                 string         `json:"resource,omitempty"`
	ResourceMetadataURL      string         `json:"resourceMetadataUrl,omitempty"`
	AuthorizationServers     []string       `json:"authorizationServers,omitempty"`
	ScopesSupported          []string       `json:"scopesSupported,omitempty"`
	SelectedIssuer           string         `json:"selectedIssuer,omitempty"`
	AuthorizationEndpoint    string         `json:"authorizationEndpoint,omitempty"`
	TokenEndpoint            string         `json:"tokenEndpoint,omitempty"`
	RegistrationEndpoint     string         `json:"registrationEndpoint,omitempty"`
	IntrospectionEndpoint    string         `json:"introspectionEndpoint,omitempty"`
	RevocationEndpoint       string         `json:"revocationEndpoint,omitempty"`
	CodeChallengeMethods     []string       `json:"codeChallengeMethods,omitempty"`
	ResponseTypesSupported   []string       `json:"responseTypesSupported,omitempty"`
	GrantTypesSupported      []string       `json:"grantTypesSupported,omitempty"`
	AuthorizationURL         string         `json:"authorizationUrl,omitempty"`
	DiscoveryErrors          []string       `json:"discoveryErrors,omitempty"`
	ResourceMetadata         map[string]any `json:"resourceMetadata,omitempty"`
	AuthorizationMetadata    map[string]any `json:"authorizationMetadata,omitempty"`
	RequiresDynamicClientReg bool           `json:"requiresDynamicClientRegistration,omitempty"`
}

type MCPOAuthStartInput struct {
	ServerID string `json:"serverId"`
}

type MCPOAuthStartResult struct {
	ServerID     string `json:"serverId"`
	Status       string `json:"status"`
	URL          string `json:"url,omitempty"`
	Instructions string `json:"instructions,omitempty"`
	ExpiresAt    string `json:"expiresAt,omitempty"`
}

type MCPOAuthStatusInput struct {
	ServerID string `json:"serverId"`
}

type MCPOAuthStatus struct {
	ServerID    string `json:"serverId"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
	Connected   bool   `json:"connected,omitempty"`
	ExpiresAt   string `json:"expiresAt,omitempty"`
	ClientID    string `json:"clientId,omitempty"`
	TokenSource string `json:"tokenSource,omitempty"`
}
