package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"aivo/core/domain"
)

type mcpStore interface {
	SaveMCPServer(context.Context, domain.MCPServerConfig) (domain.MCPServerConfig, error)
	GetMCPServer(context.Context, string) (domain.MCPServerConfig, error)
	ListMCPServers(context.Context, bool) ([]domain.MCPServerConfig, error)
	SetMCPServerEnabled(context.Context, string, bool, string, string) (domain.MCPServerConfig, error)
	ReplaceMCPTools(context.Context, string, []domain.MCPToolRecord) error
	ListMCPTools(context.Context, string) ([]domain.MCPToolRecord, error)
	ReplaceMCPPrompts(context.Context, string, []domain.MCPPromptRecord) error
	ListMCPPrompts(context.Context, string) ([]domain.MCPPromptRecord, error)
	ReplaceMCPResources(context.Context, string, []domain.MCPResourceRecord) error
	ListMCPResources(context.Context, string, bool) ([]domain.MCPResourceRecord, error)
	SavePluginDiagnostic(context.Context, domain.PluginDiagnostic) (domain.PluginDiagnostic, error)
	ListPluginDiagnostics(context.Context, string, string, int) ([]domain.PluginDiagnostic, error)
}

type MCPManager struct {
	store       mcpStore
	secrets     SecretStore
	mu          sync.Mutex
	tools       map[string][]domain.MCPToolRecord
	connections *MCPConnectionManager
	oauthServer *http.Server
	oauthFlows  map[string]*mcpOAuthFlow
}

func NewMCPManager(store any, secrets ...SecretStore) *MCPManager {
	ms, _ := store.(mcpStore)
	var secretStore SecretStore
	if len(secrets) > 0 {
		secretStore = secrets[0]
	}
	if secretStore == nil {
		secretStore = NewDefaultSecretStore()
	}
	return &MCPManager{store: ms, secrets: secretStore, tools: map[string][]domain.MCPToolRecord{}, connections: NewMCPConnectionManager(ms), oauthFlows: map[string]*mcpOAuthFlow{}}
}

func (m *MCPManager) SetSecretStore(store SecretStore) {
	if store == nil {
		store = NewMemorySecretStore()
	}
	m.secrets = store
}

func (m *MCPManager) List(ctx context.Context, input domain.MCPServerListInput) ([]domain.MCPServerListItem, error) {
	if m == nil || m.store == nil {
		return nil, errors.New("mcp store is not configured")
	}
	servers, err := m.store.ListMCPServers(ctx, input.IncludeDisabled)
	if err != nil {
		return nil, err
	}
	out := make([]domain.MCPServerListItem, 0, len(servers))
	for _, server := range servers {
		item := domain.MCPServerListItem{Server: server}
		if input.IncludeTools {
			item.Tools, _ = m.store.ListMCPTools(ctx, server.ID)
			item.Prompts, _ = m.store.ListMCPPrompts(ctx, server.ID)
			item.Resources, _ = m.store.ListMCPResources(ctx, server.ID, false)
			item.ResourceTemplates, _ = m.store.ListMCPResources(ctx, server.ID, true)
		}
		item.Diagnostics, _ = m.store.ListPluginDiagnostics(ctx, "", server.ID, 20)
		out = append(out, item)
	}
	return out, nil
}

func (m *MCPManager) Save(ctx context.Context, input domain.SaveMCPServerInput) (domain.MCPServerConfig, error) {
	if m == nil || m.store == nil {
		return domain.MCPServerConfig{}, errors.New("mcp store is not configured")
	}
	server := input.Server
	if server.ID == "" {
		server.ID = firstNonEmptyApp(server.Name, uuid.NewString())
	}
	if server.Name == "" {
		server.Name = server.ID
	}
	saved, err := m.store.SaveMCPServer(ctx, server)
	if err != nil {
		return domain.MCPServerConfig{}, err
	}
	if saved.Enabled {
		probe, probeErr := m.Probe(ctx, domain.MCPProbeInput{ServerID: saved.ID})
		if probeErr != nil {
			saved, _ = m.store.SetMCPServerEnabled(ctx, saved.ID, true, domain.MCPServerStatusError, sanitizeMCPError(probeErr.Error()))
			return saved, probeErr
		}
		if !probe.OK {
			saved, _ = m.store.SetMCPServerEnabled(ctx, saved.ID, true, domain.MCPServerStatusError, sanitizeMCPError(probe.Error))
			return saved, errors.New(probe.Error)
		}
		saved, _ = m.store.SetMCPServerEnabled(ctx, saved.ID, true, domain.MCPServerStatusReady, "")
	}
	return saved, nil
}

func (m *MCPManager) SetEnabled(ctx context.Context, input domain.SetMCPServerEnabledInput) (domain.MCPServerConfig, error) {
	if m == nil || m.store == nil {
		return domain.MCPServerConfig{}, errors.New("mcp store is not configured")
	}
	status := domain.MCPServerStatusDisabled
	if input.Enabled {
		status = domain.MCPServerStatusEnabled
	}
	server, err := m.store.SetMCPServerEnabled(ctx, input.ServerID, input.Enabled, status, "")
	if err != nil {
		return domain.MCPServerConfig{}, err
	}
	if input.Enabled {
		probe, err := m.Probe(ctx, domain.MCPProbeInput{ServerID: server.ID})
		if err != nil {
			server, _ = m.store.SetMCPServerEnabled(ctx, server.ID, true, domain.MCPServerStatusError, sanitizeMCPError(err.Error()))
			return server, err
		}
		if !probe.OK {
			server, _ = m.store.SetMCPServerEnabled(ctx, server.ID, true, domain.MCPServerStatusError, sanitizeMCPError(probe.Error))
			return server, errors.New(probe.Error)
		}
		server, _ = m.store.SetMCPServerEnabled(ctx, server.ID, true, domain.MCPServerStatusReady, "")
	} else if m.connections != nil {
		m.connections.drop(server.ID)
	}
	return server, nil
}

func (m *MCPManager) Probe(ctx context.Context, input domain.MCPProbeInput) (domain.MCPProbeResult, error) {
	if m == nil || m.store == nil {
		return domain.MCPProbeResult{}, errors.New("mcp store is not configured")
	}
	server := input.Server
	var err error
	if strings.TrimSpace(input.ServerID) != "" {
		server, err = m.store.GetMCPServer(ctx, input.ServerID)
		if err != nil {
			return domain.MCPProbeResult{}, err
		}
	}
	server, err = m.authorizedServer(ctx, server)
	if err != nil {
		m.diagnostic(ctx, server.ID, domain.PluginDiagnosticError, err.Error(), nil)
		return domain.MCPProbeResult{OK: false, ServerID: server.ID, Status: domain.MCPServerStatusError, Error: err.Error()}, nil
	}
	if m.connections == nil {
		m.connections = NewMCPConnectionManager(m.store)
	}
	capabilities, err := m.connections.Probe(ctx, server)
	if err != nil {
		err = m.handleMCPOAuthChallenge(ctx, server, err)
		message := sanitizeMCPError(err.Error())
		m.diagnostic(ctx, server.ID, domain.PluginDiagnosticError, message, nil)
		return domain.MCPProbeResult{OK: false, ServerID: server.ID, Status: domain.MCPServerStatusError, Error: message}, nil
	}
	if server.ID != "" {
		_ = m.store.ReplaceMCPTools(ctx, server.ID, capabilities.Tools)
		_ = m.store.ReplaceMCPPrompts(ctx, server.ID, capabilities.Prompts)
		resources := append([]domain.MCPResourceRecord{}, capabilities.Resources...)
		resources = append(resources, capabilities.ResourceTemplates...)
		_ = m.store.ReplaceMCPResources(ctx, server.ID, resources)
		m.mu.Lock()
		m.tools[server.ID] = capabilities.Tools
		m.mu.Unlock()
	}
	return domain.MCPProbeResult{OK: true, ServerID: server.ID, Status: domain.MCPServerStatusReady, Tools: capabilities.Tools, Prompts: capabilities.Prompts, Resources: capabilities.Resources, ResourceTemplates: capabilities.ResourceTemplates}, nil
}

func (m *MCPManager) GetPrompt(ctx context.Context, input domain.MCPPromptGetInput) (domain.MCPPromptGetResult, error) {
	if m == nil || m.store == nil {
		return domain.MCPPromptGetResult{}, errors.New("mcp store is not configured")
	}
	serverID := strings.TrimSpace(input.ServerID)
	name := strings.TrimSpace(input.Name)
	if serverID == "" || name == "" {
		return domain.MCPPromptGetResult{}, errors.New("serverId and name are required")
	}
	server, err := m.store.GetMCPServer(ctx, serverID)
	if err != nil {
		return domain.MCPPromptGetResult{}, err
	}
	server, err = m.authorizedServer(ctx, server)
	if err != nil {
		return domain.MCPPromptGetResult{}, err
	}
	arguments := input.Arguments
	if arguments == nil {
		arguments = map[string]string{}
	}
	result, err := m.callMCPMethod(ctx, server, "prompts/get", map[string]any{"name": name, "arguments": arguments})
	if err != nil {
		m.diagnostic(ctx, server.ID, domain.PluginDiagnosticError, err.Error(), map[string]any{"method": "prompts/get", "name": name})
		return domain.MCPPromptGetResult{}, err
	}
	return normalizeMCPPromptGetResult(server.ID, name, result), nil
}

func (m *MCPManager) ReadResource(ctx context.Context, input domain.MCPResourceReadInput) (domain.MCPResourceReadResult, error) {
	if m == nil || m.store == nil {
		return domain.MCPResourceReadResult{}, errors.New("mcp store is not configured")
	}
	serverID := strings.TrimSpace(input.ServerID)
	uri := strings.TrimSpace(input.URI)
	if serverID == "" || uri == "" {
		return domain.MCPResourceReadResult{}, errors.New("serverId and uri are required")
	}
	server, err := m.store.GetMCPServer(ctx, serverID)
	if err != nil {
		return domain.MCPResourceReadResult{}, err
	}
	server, err = m.authorizedServer(ctx, server)
	if err != nil {
		return domain.MCPResourceReadResult{}, err
	}
	result, err := m.callMCPMethod(ctx, server, "resources/read", map[string]any{"uri": uri})
	if err != nil {
		m.diagnostic(ctx, server.ID, domain.PluginDiagnosticError, err.Error(), map[string]any{"method": "resources/read", "uri": uri})
		return domain.MCPResourceReadResult{}, err
	}
	return normalizeMCPResourceReadResult(server.ID, uri, result), nil
}

func (m *MCPManager) ReadServerLog(ctx context.Context, input domain.MCPServerLogInput) (domain.MCPServerLogResult, error) {
	if m == nil || m.store == nil {
		return domain.MCPServerLogResult{}, errors.New("mcp store is not configured")
	}
	serverID := strings.TrimSpace(input.ServerID)
	if serverID == "" {
		return domain.MCPServerLogResult{}, errors.New("serverId is required")
	}
	if _, err := m.store.GetMCPServer(ctx, serverID); err != nil {
		return domain.MCPServerLogResult{}, err
	}
	return readMCPServerLog(ctx, input)
}

func (m *MCPManager) DiscoverOAuth(ctx context.Context, input domain.MCPOAuthDiscoveryInput) (domain.MCPOAuthDiscoveryResult, error) {
	if m == nil || m.store == nil {
		return domain.MCPOAuthDiscoveryResult{}, errors.New("mcp store is not configured")
	}
	server := input.Server
	var err error
	if strings.TrimSpace(input.ServerID) != "" {
		server, err = m.store.GetMCPServer(ctx, input.ServerID)
		if err != nil {
			return domain.MCPOAuthDiscoveryResult{}, err
		}
	}
	result, err := discoverMCPOAuth(ctx, server, "")
	if err != nil {
		m.diagnostic(ctx, server.ID, domain.PluginDiagnosticError, err.Error(), map[string]any{"method": "oauth_discovery"})
		return result, err
	}
	return result, nil
}

func (m *MCPManager) StartOAuth(ctx context.Context, input domain.MCPOAuthStartInput) (domain.MCPOAuthStartResult, error) {
	if m == nil || m.store == nil {
		return domain.MCPOAuthStartResult{}, errors.New("mcp store is not configured")
	}
	return m.startOAuth(ctx, input)
}

func (m *MCPManager) OAuthStatus(ctx context.Context, input domain.MCPOAuthStatusInput) (domain.MCPOAuthStatus, error) {
	if m == nil || m.store == nil {
		return domain.MCPOAuthStatus{}, errors.New("mcp store is not configured")
	}
	return m.oauthStatus(ctx, input)
}

func (m *MCPManager) authorizedServer(ctx context.Context, server domain.MCPServerConfig) (domain.MCPServerConfig, error) {
	server, err := resolveMCPOAuthSecrets(ctx, m.secrets, server)
	if err != nil {
		return domain.MCPServerConfig{}, err
	}
	return m.refreshOAuthIfNeeded(ctx, server)
}

func (m *MCPManager) callMCPMethod(ctx context.Context, server domain.MCPServerConfig, method string, params map[string]any) (map[string]any, error) {
	if m.connections == nil {
		m.connections = NewMCPConnectionManager(m.store)
	}
	result, err := m.connections.CallMethod(ctx, server, method, params)
	if err != nil {
		return nil, m.handleMCPOAuthChallenge(ctx, server, err)
	}
	return result, nil
}

func (m *MCPManager) callMCPTool(ctx context.Context, server domain.MCPServerConfig, name string, args json.RawMessage) (map[string]any, error) {
	if m.connections == nil {
		m.connections = NewMCPConnectionManager(m.store)
	}
	result, err := m.connections.CallTool(ctx, server, name, args)
	if err != nil {
		return nil, m.handleMCPOAuthChallenge(ctx, server, err)
	}
	return result, nil
}

func (m *MCPManager) handleMCPOAuthChallenge(ctx context.Context, server domain.MCPServerConfig, err error) error {
	if err == nil || server.AuthType != domain.MCPAuthOAuth {
		return err
	}
	var challenge *mcpHTTPAuthChallengeError
	if !errors.As(err, &challenge) {
		return err
	}
	metadata := map[string]any{
		"statusCode":       challenge.StatusCode,
		"error":            challenge.ErrorCode,
		"errorDescription": challenge.ErrorDescription,
		"resourceMetadata": challenge.ResourceMetadataURL,
		"resource":         challenge.Resource,
		"scope":            challenge.Scope,
	}
	requestedScopes := uniqueNonEmptyStrings(strings.Fields(challenge.Scope))
	if len(requestedScopes) > 0 && m != nil && m.store != nil && strings.TrimSpace(server.ID) != "" {
		next := server
		if current, loadErr := m.store.GetMCPServer(ctx, server.ID); loadErr == nil {
			next = current
		}
		next.OAuthScopes = uniqueNonEmptyStrings(append(next.OAuthScopes, requestedScopes...))
		next.Status = domain.MCPServerStatusError
		next.Error = "MCP OAuth authorization requires additional scopes: " + strings.Join(requestedScopes, " ")
		if _, saveErr := m.store.SaveMCPServer(ctx, next); saveErr != nil {
			m.diagnostic(ctx, server.ID, domain.PluginDiagnosticError, saveErr.Error(), map[string]any{"method": "oauth_challenge"})
		}
		metadata["requestedScopes"] = requestedScopes
	}
	m.diagnostic(ctx, server.ID, domain.PluginDiagnosticError, challenge.Error(), metadata)
	if len(requestedScopes) > 0 {
		return fmt.Errorf("%w; reconnect OAuth to grant scopes: %s", err, strings.Join(requestedScopes, " "))
	}
	return err
}

func (m *MCPManager) RegisterEnabledTools(ctx context.Context, registry *Registry) {
	m.registerEnabledTools(ctx, registry, true)
}

func (m *MCPManager) RegisterCachedEnabledTools(ctx context.Context, registry *Registry) {
	m.registerEnabledTools(ctx, registry, false)
}

func (m *MCPManager) registerEnabledTools(ctx context.Context, registry *Registry, allowProbe bool) {
	if m == nil || m.store == nil || registry == nil {
		return
	}
	servers, err := m.store.ListMCPServers(ctx, false)
	if err != nil {
		return
	}
	for _, server := range servers {
		if !server.Enabled {
			continue
		}
		tools, err := m.store.ListMCPTools(ctx, server.ID)
		if err != nil || len(tools) == 0 {
			probe, _ := m.Probe(ctx, domain.MCPProbeInput{ServerID: server.ID})
			tools = probe.Tools
		}
		for _, tool := range tools {
			spec := domain.ToolSpec{
				Name: mcpToolName(server, tool), Description: tool.Description, InputSchema: tool.InputSchema,
				Namespace: server.Name, NamespaceDescription: server.Description,
				Capability: firstNonEmptyApp(tool.Capability, "mcp.read"), RiskLevel: firstNonEmptyApp(tool.RiskLevel, "medium"),
				Category: "mcp", Toolsets: []string{"mcp", "coding"}, RequiresNetwork: server.Transport != domain.MCPTransportStdio,
			}
			_ = registry.RegisterScoped(&MCPRuntimeTool{manager: m, server: server, tool: tool, spec: spec}, domain.ToolSourceMCP, server.ID, server.TimeUpdated)
		}
		m.registerMCPResourceUtilityTools(ctx, registry, server)
	}
}

func (m *MCPManager) diagnostic(ctx context.Context, serverID string, level string, message string, metadata map[string]any) {
	if m == nil || m.store == nil || message == "" {
		return
	}
	_, _ = m.store.SavePluginDiagnostic(ctx, domain.PluginDiagnostic{ServerID: serverID, Level: level, Message: sanitizeMCPError(message), Metadata: metadata})
}

func (m *MCPManager) registerMCPResourceUtilityTools(_ context.Context, registry *Registry, server domain.MCPServerConfig) {
	if registry == nil {
		return
	}
	base := mcpServerToolPrefix(server)
	utilities := []MCPResourceUtilityTool{
		{manager: m, server: server, kind: "list_resources", spec: domain.ToolSpec{
			Name: base + "_list_resources", Description: "List resources exposed by MCP server " + firstNonEmptyApp(server.DisplayName, server.Name, server.ID) + ".",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{
				"cursor": map[string]any{"type": "string", "description": "Optional pagination cursor returned by a previous resources/list call."},
			}, "additionalProperties": false},
			Namespace: server.Name, NamespaceDescription: server.Description, Capability: "mcp.read", RiskLevel: "low",
			Category: "mcp", Toolsets: []string{"mcp", "coding"}, RequiresNetwork: server.Transport != domain.MCPTransportStdio,
		}},
		{manager: m, server: server, kind: "list_resource_templates", spec: domain.ToolSpec{
			Name: base + "_list_resource_templates", Description: "List resource templates exposed by MCP server " + firstNonEmptyApp(server.DisplayName, server.Name, server.ID) + ".",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{
				"cursor": map[string]any{"type": "string", "description": "Optional pagination cursor returned by a previous resources/templates/list call."},
			}, "additionalProperties": false},
			Namespace: server.Name, NamespaceDescription: server.Description, Capability: "mcp.read", RiskLevel: "low",
			Category: "mcp", Toolsets: []string{"mcp", "coding"}, RequiresNetwork: server.Transport != domain.MCPTransportStdio,
		}},
		{manager: m, server: server, kind: "read_resource", spec: domain.ToolSpec{
			Name: base + "_read_resource", Description: "Read a resource URI from MCP server " + firstNonEmptyApp(server.DisplayName, server.Name, server.ID) + ".",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{
				"uri": map[string]any{"type": "string", "description": "Exact MCP resource URI to read."},
			}, "required": []string{"uri"}, "additionalProperties": false},
			Namespace: server.Name, NamespaceDescription: server.Description, Capability: "mcp.read", RiskLevel: "low",
			Category: "mcp", Toolsets: []string{"mcp", "coding"}, RequiresNetwork: server.Transport != domain.MCPTransportStdio,
		}},
	}
	for i := range utilities {
		tool := utilities[i]
		_ = registry.RegisterScoped(&tool, domain.ToolSourceMCP, server.ID, server.TimeUpdated)
	}
}

type MCPRuntimeTool struct {
	server  domain.MCPServerConfig
	tool    domain.MCPToolRecord
	spec    domain.ToolSpec
	secrets SecretStore
	manager *MCPManager
}

func (t *MCPRuntimeTool) Spec() domain.ToolSpec { return t.spec }

func (t *MCPRuntimeTool) Execute(ctx context.Context, args json.RawMessage, _ domain.ToolExecutionContext) domain.ToolResult {
	server := t.server
	var err error
	if t.manager != nil {
		server, err = t.manager.authorizedServer(ctx, server)
	} else {
		server, err = resolveMCPOAuthSecrets(ctx, t.secrets, server)
	}
	if err != nil {
		return toolFailure("", t.spec.Name, "mcp_auth_failed", sanitizeMCPError(err.Error()))
	}
	if t.manager != nil {
		result, err := t.manager.callMCPTool(ctx, server, t.tool.Name, args)
		if err != nil {
			return toolFailure("", t.spec.Name, "mcp_tool_failed", sanitizeMCPError(err.Error()))
		}
		return normalizeMCPToolResult(t.spec.Name, result)
	}
	result, err := callMCPTool(ctx, server, t.tool.Name, args)
	if err != nil {
		return toolFailure("", t.spec.Name, "mcp_tool_failed", sanitizeMCPError(err.Error()))
	}
	return normalizeMCPToolResult(t.spec.Name, result)
}

type MCPResourceUtilityTool struct {
	manager *MCPManager
	server  domain.MCPServerConfig
	kind    string
	spec    domain.ToolSpec
}

func (t *MCPResourceUtilityTool) Spec() domain.ToolSpec { return t.spec }

func (t *MCPResourceUtilityTool) Execute(ctx context.Context, args json.RawMessage, _ domain.ToolExecutionContext) domain.ToolResult {
	if t == nil || t.manager == nil {
		return toolFailure("", t.spec.Name, "mcp_tool_failed", "mcp manager is not configured")
	}
	server, err := t.manager.authorizedServer(ctx, t.server)
	if err != nil {
		return toolFailure("", t.spec.Name, "mcp_auth_failed", sanitizeMCPError(err.Error()))
	}
	var input struct {
		URI    string `json:"uri"`
		Cursor string `json:"cursor"`
	}
	if len(args) > 0 {
		_ = json.Unmarshal(args, &input)
	}
	switch t.kind {
	case "list_resources":
		return t.executeList(ctx, server, "resources/list", strings.TrimSpace(input.Cursor), false)
	case "list_resource_templates":
		return t.executeList(ctx, server, "resources/templates/list", strings.TrimSpace(input.Cursor), true)
	case "read_resource":
		uri := strings.TrimSpace(input.URI)
		if uri == "" {
			return toolFailure("", t.spec.Name, "invalid_arguments", "uri is required")
		}
		result, err := t.manager.callMCPMethod(ctx, server, "resources/read", map[string]any{"uri": uri})
		if err != nil {
			return toolFailure("", t.spec.Name, "mcp_tool_failed", sanitizeMCPError(err.Error()))
		}
		read := normalizeMCPResourceReadResult(server.ID, uri, result)
		content := strings.TrimSpace(read.Content)
		if content == "" {
			raw, _ := json.MarshalIndent(result, "", "  ")
			content = string(raw)
		}
		return domain.ToolResult{Name: t.spec.Name, OK: true, Content: content, ModelContent: content, Structured: result}
	default:
		return toolFailure("", t.spec.Name, "mcp_tool_failed", "unknown MCP resource utility")
	}
}

func (t *MCPResourceUtilityTool) executeList(ctx context.Context, server domain.MCPServerConfig, method string, cursor string, templates bool) domain.ToolResult {
	params := map[string]any{}
	if cursor != "" {
		params["cursor"] = cursor
	}
	result, err := t.manager.callMCPMethod(ctx, server, method, params)
	if err != nil {
		return toolFailure("", t.spec.Name, "mcp_tool_failed", sanitizeMCPError(err.Error()))
	}
	records := parseMCPResources(server, result, templates)
	items := make([]map[string]any, 0, len(records))
	for _, record := range records {
		items = append(items, map[string]any{
			"name": record.Name, "uri": record.URI, "uriTemplate": record.URITemplate,
			"description": record.Description, "mimeType": record.MimeType,
		})
	}
	structured := map[string]any{"resources": items, "count": len(items)}
	if next, _ := result["nextCursor"].(string); strings.TrimSpace(next) != "" {
		structured["nextCursor"] = next
	}
	raw, _ := json.MarshalIndent(structured, "", "  ")
	return domain.ToolResult{Name: t.spec.Name, OK: true, Content: string(raw), ModelContent: string(raw), Structured: structured}
}

type mcpServerCapabilities struct {
	Tools             []domain.MCPToolRecord
	Prompts           []domain.MCPPromptRecord
	Resources         []domain.MCPResourceRecord
	ResourceTemplates []domain.MCPResourceRecord
}

func probeMCPServer(ctx context.Context, server domain.MCPServerConfig) (mcpServerCapabilities, error) {
	if server.Transport == "" {
		server.Transport = domain.MCPTransportStdio
	}
	if server.Transport == domain.MCPTransportStreamableHTTP || server.Transport == domain.MCPTransportSSE {
		client, err := newMCPHTTPClient(server)
		if err != nil {
			return mcpServerCapabilities{}, err
		}
		if _, err := client.call(ctx, "initialize", mcpInitializeParams(server)); err != nil {
			return mcpServerCapabilities{}, err
		}
		result, _ := client.call(ctx, "tools/list", map[string]any{})
		tools := parseMCPTools(server, result)
		tools = refreshMCPHTTPToolsIfChanged(ctx, client, server, tools)
		promptsResult, _ := client.call(ctx, "prompts/list", map[string]any{})
		tools = refreshMCPHTTPToolsIfChanged(ctx, client, server, tools)
		resourcesResult, _ := client.call(ctx, "resources/list", map[string]any{})
		tools = refreshMCPHTTPToolsIfChanged(ctx, client, server, tools)
		templatesResult, _ := client.call(ctx, "resources/templates/list", map[string]any{})
		tools = refreshMCPHTTPToolsIfChanged(ctx, client, server, tools)
		return mcpServerCapabilities{
			Tools:             tools,
			Prompts:           parseMCPPrompts(server, promptsResult),
			Resources:         parseMCPResources(server, resourcesResult, false),
			ResourceTemplates: parseMCPResources(server, templatesResult, true),
		}, nil
	}
	if server.Transport != domain.MCPTransportStdio {
		return mcpServerCapabilities{}, fmt.Errorf("unsupported mcp transport %s", server.Transport)
	}
	client, err := startMCPStdio(ctx, server)
	if err != nil {
		return mcpServerCapabilities{}, err
	}
	defer client.close()
	if _, err := client.call(ctx, "initialize", mcpInitializeParams(server)); err != nil {
		return mcpServerCapabilities{}, err
	}
	result, _ := client.call(ctx, "tools/list", map[string]any{})
	tools := parseMCPTools(server, result)
	tools = refreshMCPToolsIfChanged(ctx, client, server, tools)
	promptsResult, _ := client.call(ctx, "prompts/list", map[string]any{})
	tools = refreshMCPToolsIfChanged(ctx, client, server, tools)
	resourcesResult, _ := client.call(ctx, "resources/list", map[string]any{})
	tools = refreshMCPToolsIfChanged(ctx, client, server, tools)
	templatesResult, _ := client.call(ctx, "resources/templates/list", map[string]any{})
	tools = refreshMCPToolsIfChanged(ctx, client, server, tools)
	return mcpServerCapabilities{
		Tools:             tools,
		Prompts:           parseMCPPrompts(server, promptsResult),
		Resources:         parseMCPResources(server, resourcesResult, false),
		ResourceTemplates: parseMCPResources(server, templatesResult, true),
	}, nil
}

func refreshMCPToolsIfChanged(ctx context.Context, client *mcpStdioClient, server domain.MCPServerConfig, current []domain.MCPToolRecord) []domain.MCPToolRecord {
	if client == nil || !client.consumeToolsListChanged() {
		return current
	}
	result, err := client.call(ctx, "tools/list", map[string]any{})
	if err != nil {
		return current
	}
	next := parseMCPTools(server, result)
	if len(next) == 0 {
		return current
	}
	return next
}

func refreshMCPHTTPToolsIfChanged(ctx context.Context, client *mcpHTTPClient, server domain.MCPServerConfig, current []domain.MCPToolRecord) []domain.MCPToolRecord {
	if client == nil || !client.consumeToolsListChanged() {
		return current
	}
	result, err := client.call(ctx, "tools/list", map[string]any{})
	if err != nil {
		return current
	}
	next := parseMCPTools(server, result)
	if len(next) == 0 {
		return current
	}
	return next
}

func parseMCPTools(server domain.MCPServerConfig, result map[string]any) []domain.MCPToolRecord {
	if result == nil {
		return nil
	}
	rawTools, _ := result["tools"].([]any)
	tools := make([]domain.MCPToolRecord, 0, len(rawTools))
	now := domain.NowString(time.Now())
	for _, rawTool := range rawTools {
		item, _ := rawTool.(map[string]any)
		name, _ := item["name"].(string)
		if strings.TrimSpace(name) == "" {
			continue
		}
		desc, _ := item["description"].(string)
		schema, _ := item["inputSchema"].(map[string]any)
		if schema == nil {
			schema = map[string]any{"type": "object", "additionalProperties": true}
		}
		tools = append(tools, domain.MCPToolRecord{ID: server.ID + ":" + name, ServerID: server.ID, Name: name, Description: desc, InputSchema: schema, Capability: "mcp.read", RiskLevel: "medium", TimeUpdated: now})
	}
	return tools
}

func parseMCPPrompts(server domain.MCPServerConfig, result map[string]any) []domain.MCPPromptRecord {
	if result == nil {
		return nil
	}
	rawPrompts, _ := result["prompts"].([]any)
	prompts := make([]domain.MCPPromptRecord, 0, len(rawPrompts))
	now := domain.NowString(time.Now())
	for _, rawPrompt := range rawPrompts {
		item, _ := rawPrompt.(map[string]any)
		name, _ := item["name"].(string)
		if strings.TrimSpace(name) == "" {
			continue
		}
		desc, _ := item["description"].(string)
		prompts = append(prompts, domain.MCPPromptRecord{ID: server.ID + ":" + name, ServerID: server.ID, Name: name, Description: desc, Arguments: parseMCPPromptArguments(item["arguments"]), TimeUpdated: now})
	}
	return prompts
}

func parseMCPPromptArguments(value any) []domain.MCPPromptArgument {
	rawArgs, _ := value.([]any)
	args := make([]domain.MCPPromptArgument, 0, len(rawArgs))
	for _, rawArg := range rawArgs {
		item, _ := rawArg.(map[string]any)
		name, _ := item["name"].(string)
		if strings.TrimSpace(name) == "" {
			continue
		}
		desc, _ := item["description"].(string)
		required, _ := item["required"].(bool)
		args = append(args, domain.MCPPromptArgument{Name: name, Description: desc, Required: required})
	}
	return args
}

func parseMCPResources(server domain.MCPServerConfig, result map[string]any, templates bool) []domain.MCPResourceRecord {
	if result == nil {
		return nil
	}
	key := "resources"
	if templates {
		key = "resourceTemplates"
	}
	rawResources, _ := result[key].([]any)
	resources := make([]domain.MCPResourceRecord, 0, len(rawResources))
	now := domain.NowString(time.Now())
	for _, rawResource := range rawResources {
		item, _ := rawResource.(map[string]any)
		uri, _ := item["uri"].(string)
		uriTemplate, _ := item["uriTemplate"].(string)
		name, _ := item["name"].(string)
		if strings.TrimSpace(name) == "" {
			name = firstNonEmptyApp(uri, uriTemplate)
		}
		if strings.TrimSpace(name) == "" {
			continue
		}
		desc, _ := item["description"].(string)
		mimeType, _ := item["mimeType"].(string)
		idPart := firstNonEmptyApp(uri, uriTemplate, name)
		resources = append(resources, domain.MCPResourceRecord{ID: server.ID + ":" + idPart, ServerID: server.ID, URI: uri, URITemplate: uriTemplate, Name: name, Description: desc, MimeType: mimeType, Template: templates, TimeUpdated: now})
	}
	return resources
}

func callMCPTool(ctx context.Context, server domain.MCPServerConfig, name string, args json.RawMessage) (map[string]any, error) {
	var arguments any = map[string]any{}
	if len(args) > 0 {
		_ = json.Unmarshal(args, &arguments)
	}
	return callMCPMethod(ctx, server, "tools/call", map[string]any{"name": name, "arguments": arguments})
}

func callMCPMethod(ctx context.Context, server domain.MCPServerConfig, method string, params map[string]any) (map[string]any, error) {
	if server.Transport == "" {
		server.Transport = domain.MCPTransportStdio
	}
	if server.Transport == domain.MCPTransportStreamableHTTP || server.Transport == domain.MCPTransportSSE {
		client, err := newMCPHTTPClient(server)
		if err != nil {
			return nil, err
		}
		if _, err := client.call(ctx, "initialize", mcpInitializeParams(server)); err != nil {
			return nil, err
		}
		return client.call(ctx, method, params)
	}
	if server.Transport != domain.MCPTransportStdio {
		return nil, fmt.Errorf("unsupported mcp transport %s", server.Transport)
	}
	client, err := startMCPStdio(ctx, server)
	if err != nil {
		return nil, err
	}
	defer client.close()
	if _, err := client.call(ctx, "initialize", mcpInitializeParams(server)); err != nil {
		return nil, err
	}
	return client.call(ctx, method, params)
}

func mcpInitializeParams(server domain.MCPServerConfig) map[string]any {
	capabilities := map[string]any{}
	if len(mcpRootEntries(server)) > 0 {
		capabilities["roots"] = map[string]any{"listChanged": true}
	}
	return map[string]any{
		"protocolVersion": "2025-03-26",
		"capabilities":    capabilities,
		"clientInfo":      map[string]any{"name": "aivo", "version": "phase6"},
	}
}

func mcpRootEntries(server domain.MCPServerConfig) []map[string]any {
	roots := make([]map[string]any, 0, len(server.Roots))
	seen := map[string]bool{}
	for _, root := range server.Roots {
		uri, name := mcpRootURIAndName(root)
		if uri == "" || seen[uri] {
			continue
		}
		seen[uri] = true
		roots = append(roots, map[string]any{"uri": uri, "name": name})
	}
	return roots
}

func mcpRootURIAndName(root string) (string, string) {
	clean := strings.TrimSpace(root)
	if clean == "" {
		return "", ""
	}
	if parsed, err := url.Parse(clean); err == nil && parsed.IsAbs() {
		return parsed.String(), firstNonEmptyApp(filepath.Base(parsed.Path), parsed.Host, parsed.String())
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return "", ""
	}
	uri := (&url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}).String()
	return uri, filepath.Base(abs)
}

type mcpHTTPClient struct {
	server           domain.MCPServerConfig
	client           *http.Client
	toolsListChanged bool
}

func newMCPHTTPClient(server domain.MCPServerConfig) (*mcpHTTPClient, error) {
	rawURL := strings.TrimSpace(server.URL)
	if rawURL == "" {
		return nil, errors.New("http MCP server URL is required")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || !parsed.IsAbs() {
		return nil, errors.New("http MCP server URL must be absolute")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("http MCP server URL must use http or https")
	}
	timeout := time.Duration(server.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &mcpHTTPClient{
		server: server,
		client: &http.Client{Timeout: timeout},
	}, nil
}

func (c *mcpHTTPClient) call(ctx context.Context, method string, params map[string]any) (map[string]any, error) {
	id := uuid.NewString()
	raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.server.URL, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", "2025-03-26")
	for key, value := range c.server.Headers {
		if strings.TrimSpace(key) == "" {
			continue
		}
		req.Header.Set(key, value)
	}
	if err := c.applyAuth(req); err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, mcpHTTPAuthError(resp.StatusCode, resp.Header.Get("WWW-Authenticate"), body)
		}
		return nil, fmt.Errorf("mcp http request failed with status %d: %s", resp.StatusCode, bounded(strings.TrimSpace(string(body)), 1000))
	}
	return c.parseResponse(ctx, resp.Header.Get("Content-Type"), body)
}

func (c *mcpHTTPClient) parseResponse(ctx context.Context, contentType string, body []byte) (map[string]any, error) {
	return parseMCPHTTPResponseWithNotifications(contentType, body, func(id json.RawMessage, method string) error {
		return c.handleServerMessage(ctx, id, method)
	})
}

func (c *mcpHTTPClient) handleNotification(method string) {
	switch method {
	case "notifications/tools/list_changed", "tools/list_changed":
		c.toolsListChanged = true
	}
}

func (c *mcpHTTPClient) handleServerMessage(ctx context.Context, id json.RawMessage, method string) error {
	if len(id) == 0 || string(id) == "null" {
		c.handleNotification(method)
		return nil
	}
	switch method {
	case "roots/list":
		return c.sendServerResponse(ctx, id, map[string]any{"roots": mcpRootEntries(c.server)}, nil)
	default:
		return c.sendServerResponse(ctx, id, nil, map[string]any{"code": -32601, "message": "method not found"})
	}
}

func (c *mcpHTTPClient) sendServerResponse(ctx context.Context, id json.RawMessage, result map[string]any, rpcError map[string]any) error {
	payload := map[string]any{"jsonrpc": "2.0", "id": id}
	if rpcError != nil {
		payload["error"] = rpcError
	} else {
		payload["result"] = result
	}
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.server.URL, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", "2025-03-26")
	for key, value := range c.server.Headers {
		if strings.TrimSpace(key) == "" {
			continue
		}
		req.Header.Set(key, value)
	}
	if err := c.applyAuth(req); err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("mcp http server request response failed with status %d: %s", resp.StatusCode, bounded(strings.TrimSpace(string(body)), 500))
	}
	return nil
}

func (c *mcpHTTPClient) consumeToolsListChanged() bool {
	if c == nil || !c.toolsListChanged {
		return false
	}
	c.toolsListChanged = false
	return true
}

func (c *mcpHTTPClient) close() {}

func (c *mcpHTTPClient) applyAuth(req *http.Request) error {
	authType := strings.TrimSpace(c.server.AuthType)
	if authType == "" || authType == domain.MCPAuthNone {
		return nil
	}
	switch authType {
	case domain.MCPAuthBearer:
		envName := strings.TrimSpace(c.server.BearerTokenEnv)
		if envName == "" {
			return fmt.Errorf("mcp %s auth requires bearerTokenEnv", authType)
		}
		token := strings.TrimSpace(os.Getenv(envName))
		if token == "" {
			return fmt.Errorf("mcp %s auth token environment variable %s is not set", authType, envName)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	case domain.MCPAuthOAuth:
		if token := strings.TrimSpace(c.server.OAuthAccessToken); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
			return nil
		}
		envName := strings.TrimSpace(c.server.BearerTokenEnv)
		if envName == "" {
			return nil
		}
		token := strings.TrimSpace(os.Getenv(envName))
		if token == "" {
			return fmt.Errorf("mcp %s auth token environment variable %s is not set", authType, envName)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	default:
		return fmt.Errorf("unsupported mcp auth type %s", authType)
	}
}

type mcpHTTPAuthChallengeError struct {
	StatusCode          int
	Challenge           string
	Body                string
	ErrorCode           string
	ErrorDescription    string
	ResourceMetadataURL string
	Resource            string
	Scope               string
}

func (e *mcpHTTPAuthChallengeError) Error() string {
	if e == nil {
		return ""
	}
	message := fmt.Sprintf("mcp http request requires authentication (status %d)", e.StatusCode)
	if e.ErrorCode != "" {
		message += "; error: " + e.ErrorCode
	}
	if e.ErrorDescription != "" {
		message += "; description: " + e.ErrorDescription
	}
	if e.ResourceMetadataURL != "" {
		message += "; oauth resource metadata: " + e.ResourceMetadataURL
	}
	if e.Scope != "" {
		message += "; required scope: " + e.Scope
	}
	if e.Body != "" {
		message += "; response: " + bounded(e.Body, 500)
	}
	return message
}

func mcpHTTPAuthError(statusCode int, challenge string, body []byte) error {
	params := mcpWWWAuthenticateParams(challenge)
	metadataURL := firstNonEmptyApp(params["resource_metadata"], params["resource"])
	return &mcpHTTPAuthChallengeError{
		StatusCode:          statusCode,
		Challenge:           challenge,
		Body:                strings.TrimSpace(string(body)),
		ErrorCode:           params["error"],
		ErrorDescription:    params["error_description"],
		ResourceMetadataURL: metadataURL,
		Resource:            params["resource"],
		Scope:               params["scope"],
	}
}

func mcpWWWAuthenticateParam(challenge string, name string) string {
	return mcpWWWAuthenticateParams(challenge)[strings.ToLower(strings.TrimSpace(name))]
}

func mcpWWWAuthenticateParams(challenge string) map[string]string {
	out := map[string]string{}
	if strings.TrimSpace(challenge) == "" {
		return out
	}
	text := strings.TrimSpace(challenge)
	if idx := strings.IndexAny(text, " \t"); idx >= 0 {
		scheme := strings.ToLower(strings.TrimSpace(text[:idx]))
		if scheme == "bearer" || scheme == "basic" || scheme == "digest" {
			text = strings.TrimSpace(text[idx+1:])
		}
	}
	for i := 0; i < len(text); {
		for i < len(text) && (text[i] == ',' || text[i] == ' ' || text[i] == '\t') {
			i++
		}
		start := i
		for i < len(text) && text[i] != '=' && text[i] != ',' {
			i++
		}
		if i >= len(text) || text[i] != '=' {
			for i < len(text) && text[i] != ',' {
				i++
			}
			continue
		}
		key := strings.ToLower(strings.TrimSpace(text[start:i]))
		i++
		value := ""
		if i < len(text) && text[i] == '"' {
			i++
			var builder strings.Builder
			for i < len(text) {
				if text[i] == '\\' && i+1 < len(text) {
					builder.WriteByte(text[i+1])
					i += 2
					continue
				}
				if text[i] == '"' {
					i++
					break
				}
				builder.WriteByte(text[i])
				i++
			}
			value = builder.String()
		} else {
			startValue := i
			for i < len(text) && text[i] != ',' {
				i++
			}
			value = strings.TrimSpace(text[startValue:i])
		}
		if key != "" {
			out[key] = strings.TrimSpace(value)
		}
		for i < len(text) && text[i] != ',' {
			i++
		}
	}
	return out
}

func parseMCPHTTPResponse(contentType string, body []byte) (map[string]any, error) {
	return parseMCPHTTPResponseWithNotifications(contentType, body, nil)
}

func parseMCPHTTPResponseWithNotifications(contentType string, body []byte, handle func(json.RawMessage, string) error) (map[string]any, error) {
	payload := bytes.TrimSpace(body)
	if strings.Contains(strings.ToLower(contentType), "text/event-stream") || bytes.HasPrefix(payload, []byte("event:")) || bytes.HasPrefix(payload, []byte("data:")) {
		return parseMCPSSEResponse(payload, handle)
	}
	result, _, err := parseMCPJSONRPCMessage(payload, handle)
	return result, err
}

func parseMCPSSEResponse(body []byte, handle func(json.RawMessage, string) error) (map[string]any, error) {
	payloads, err := mcpJSONPayloadsFromSSE(body)
	if err != nil {
		return nil, err
	}
	for _, payload := range payloads {
		result, response, err := parseMCPJSONRPCMessage(payload, handle)
		if err != nil {
			return nil, err
		}
		if response {
			return result, nil
		}
	}
	return map[string]any{}, nil
}

func parseMCPJSONRPCResult(payload []byte) (map[string]any, error) {
	result, _, err := parseMCPJSONRPCMessage(payload, nil)
	return result, err
}

func parseMCPJSONRPCMessage(payload []byte, handle func(json.RawMessage, string) error) (map[string]any, bool, error) {
	payload = bytes.TrimSpace(payload)
	if len(payload) == 0 {
		return map[string]any{}, false, nil
	}
	var message struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Result map[string]any  `json:"result"`
		Error  any             `json:"error"`
	}
	if err := json.Unmarshal(payload, &message); err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(message.Method) != "" {
		if handle != nil {
			if err := handle(message.ID, message.Method); err != nil {
				return nil, false, err
			}
		}
		return map[string]any{}, false, nil
	}
	if len(message.ID) == 0 || string(message.ID) == "null" {
		return map[string]any{}, false, nil
	}
	if message.Error != nil {
		return nil, true, mcpRPCError(message.Error)
	}
	if message.Result == nil {
		return map[string]any{}, true, nil
	}
	return message.Result, true, nil
}

func mcpJSONFromSSE(body []byte) ([]byte, error) {
	payloads, err := mcpJSONPayloadsFromSSE(body)
	if err != nil {
		return nil, err
	}
	for _, payload := range payloads {
		if len(bytes.TrimSpace(payload)) > 0 {
			return payload, nil
		}
	}
	return nil, errors.New("mcp sse response did not contain data")
}

func mcpJSONPayloadsFromSSE(body []byte) ([][]byte, error) {
	events := strings.Split(string(body), "\n\n")
	payloads := make([][]byte, 0, len(events))
	for _, event := range events {
		lines := strings.Split(event, "\n")
		data := strings.Builder{}
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "data:") {
				data.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
		}
		if strings.TrimSpace(data.String()) != "" {
			payloads = append(payloads, []byte(data.String()))
		}
	}
	if len(payloads) == 0 {
		return nil, errors.New("mcp sse response did not contain data")
	}
	return payloads, nil
}

func mcpRPCError(value any) error {
	if item, ok := value.(map[string]any); ok {
		if message, _ := item["message"].(string); message != "" {
			return errors.New(sanitizeMCPError(message))
		}
	}
	return errors.New(sanitizeMCPError(fmt.Sprintf("%v", value)))
}

var mcpCredentialPattern = regexp.MustCompile(`(?i)(Bearer\s+[A-Za-z0-9._~+/=-]+|sk-[A-Za-z0-9._-]{8,}|ghp_[A-Za-z0-9_]{8,}|(?:token|key|api_key|password|secret)=["']?[^&\s,"']+)`)

func sanitizeMCPError(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	return mcpCredentialPattern.ReplaceAllString(message, "[redacted]")
}

type mcpStdioClient struct {
	server           domain.MCPServerConfig
	cmd              *exec.Cmd
	stdin            io.WriteCloser
	lines            *bufio.Reader
	mu               sync.Mutex
	toolsListChanged bool
}

func startMCPStdio(ctx context.Context, server domain.MCPServerConfig) (*mcpStdioClient, error) {
	if strings.TrimSpace(server.Command) == "" {
		return nil, errors.New("stdio MCP server command is required")
	}
	command := server.Command
	cmd := exec.CommandContext(ctx, command, server.Args...)
	if server.CWD != "" {
		cmd.Dir = server.CWD
	} else if wd, err := os.Getwd(); err == nil {
		cmd.Dir = wd
	}
	cmd.Env = SanitizedEnvironment(firstNonEmptyApp(cmd.Dir, "."), defaultEnvAllowlist(), server.Env, nil)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = mcpStderrWriter(server.ID)
	setProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &mcpStdioClient{server: server, cmd: cmd, stdin: stdin, lines: bufio.NewReader(stdout)}, nil
}

func (c *mcpStdioClient) call(ctx context.Context, method string, params map[string]any) (map[string]any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := uuid.NewString()
	raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	if _, err := c.stdin.Write(append(raw, '\n')); err != nil {
		return nil, err
	}
	timeout := time.Duration(c.server.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	deadline := time.After(timeout)
	for {
		line, err := c.readLine(ctx, deadline)
		if err != nil {
			return nil, err
		}
		var resp struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Result map[string]any  `json:"result"`
			Error  any             `json:"error"`
		}
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			continue
		}
		if resp.Method != "" {
			_ = c.handleServerMessage(resp.ID, resp.Method)
			continue
		}
		var responseID string
		_ = json.Unmarshal(resp.ID, &responseID)
		if responseID != id {
			continue
		}
		if resp.Error != nil {
			return nil, mcpRPCError(resp.Error)
		}
		return resp.Result, nil
	}
}

func (c *mcpStdioClient) readLine(ctx context.Context, deadline <-chan time.Time) (string, error) {
	type readResult struct {
		line string
		err  error
	}
	ch := make(chan readResult, 1)
	go func() {
		line, err := c.lines.ReadString('\n')
		ch <- readResult{line: line, err: err}
	}()
	select {
	case <-ctx.Done():
		c.close()
		return "", ctx.Err()
	case <-deadline:
		c.close()
		return "", errors.New("mcp request timed out")
	case result := <-ch:
		return result.line, result.err
	}
}

func (c *mcpStdioClient) handleServerMessage(id json.RawMessage, method string) error {
	if len(id) == 0 || string(id) == "null" {
		c.handleServerNotification(method)
		return nil
	}
	return c.handleServerRequest(id, method)
}

func (c *mcpStdioClient) handleServerNotification(method string) {
	switch method {
	case "notifications/tools/list_changed", "tools/list_changed":
		c.toolsListChanged = true
	}
}

func (c *mcpStdioClient) consumeToolsListChanged() bool {
	if c == nil || !c.toolsListChanged {
		return false
	}
	c.toolsListChanged = false
	return true
}

func (c *mcpStdioClient) handleServerRequest(id json.RawMessage, method string) error {
	var result map[string]any
	switch method {
	case "roots/list":
		result = map[string]any{"roots": mcpRootEntries(c.server)}
	default:
		raw, _ := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      json.RawMessage(id),
			"error":   map[string]any{"code": -32601, "message": "method not found"},
		})
		_, err := c.stdin.Write(append(raw, '\n'))
		return err
	}
	raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(id), "result": result})
	_, err := c.stdin.Write(append(raw, '\n'))
	return err
}

func (c *mcpStdioClient) close() {
	if c == nil {
		return
	}
	_ = c.stdin.Close()
	if c.cmd != nil && c.cmd.Process != nil {
		_ = killProcessGroup(c.cmd.Process)
	}
}

func mcpToolName(server domain.MCPServerConfig, tool domain.MCPToolRecord) string {
	return mcpServerToolPrefix(server) + "_" + sanitizeMCPToolNameComponent(tool.Name)
}

func mcpServerToolPrefix(server domain.MCPServerConfig) string {
	prefix := strings.TrimSpace(server.Name)
	if prefix == "" {
		prefix = strings.TrimSpace(server.ID)
	}
	return "mcp_" + sanitizeMCPToolNameComponent(prefix)
}

func sanitizeMCPToolNameComponent(value string) string {
	value = strings.TrimSpace(value)
	value = strings.NewReplacer(" ", "_", "-", "_", ".", "_", ":", "_", "/", "_", "\\", "_").Replace(value)
	if value == "" {
		return "server"
	}
	return value
}

func normalizeMCPToolResult(name string, result map[string]any) domain.ToolResult {
	content := ""
	if blocks, ok := result["content"].([]any); ok {
		parts := []string{}
		for _, block := range blocks {
			item, _ := block.(map[string]any)
			if item["type"] == "text" {
				if text, _ := item["text"].(string); text != "" {
					parts = append(parts, text)
				}
			}
		}
		content = strings.Join(parts, "\n")
	}
	if content == "" {
		if text, _ := result["content"].(string); text != "" {
			content = text
		}
	}
	if content == "" {
		raw, _ := json.MarshalIndent(result, "", "  ")
		content = string(raw)
	}
	ok := true
	if isError, _ := result["isError"].(bool); isError {
		ok = false
	}
	return domain.ToolResult{Name: name, OK: ok, Content: content, ModelContent: content, Structured: result, Error: errorFromMCPResult(result)}
}

func normalizeMCPPromptGetResult(serverID string, name string, result map[string]any) domain.MCPPromptGetResult {
	description, _ := result["description"].(string)
	rawMessages, _ := result["messages"].([]any)
	messages := make([]domain.MCPPromptMessage, 0, len(rawMessages))
	parts := []string{}
	for _, rawMessage := range rawMessages {
		item, _ := rawMessage.(map[string]any)
		role, _ := item["role"].(string)
		blocks := parseMCPContentBlocks(item["content"])
		messages = append(messages, domain.MCPPromptMessage{Role: role, Content: blocks})
		text := textFromMCPContentBlocks(blocks)
		if text != "" {
			if role != "" {
				parts = append(parts, role+": "+text)
			} else {
				parts = append(parts, text)
			}
		}
	}
	return domain.MCPPromptGetResult{ServerID: serverID, Name: name, Description: description, Messages: messages, Content: strings.Join(parts, "\n"), Structured: result}
}

func normalizeMCPResourceReadResult(serverID string, uri string, result map[string]any) domain.MCPResourceReadResult {
	rawContents, _ := result["contents"].([]any)
	contents := make([]domain.MCPResourceContent, 0, len(rawContents))
	parts := []string{}
	for _, rawContent := range rawContents {
		item, _ := rawContent.(map[string]any)
		contentURI, _ := item["uri"].(string)
		mimeType, _ := item["mimeType"].(string)
		text, _ := item["text"].(string)
		blob, _ := item["blob"].(string)
		contents = append(contents, domain.MCPResourceContent{URI: contentURI, MimeType: mimeType, Text: text, Blob: blob})
		if text != "" {
			parts = append(parts, text)
		}
	}
	return domain.MCPResourceReadResult{ServerID: serverID, URI: uri, Contents: contents, Content: strings.Join(parts, "\n"), Structured: result}
}

func parseMCPContentBlocks(value any) []domain.MCPContentBlock {
	switch typed := value.(type) {
	case []any:
		blocks := make([]domain.MCPContentBlock, 0, len(typed))
		for _, rawBlock := range typed {
			if block := parseMCPContentBlock(rawBlock); block.Type != "" {
				blocks = append(blocks, block)
			}
		}
		return blocks
	case map[string]any:
		if block := parseMCPContentBlock(typed); block.Type != "" {
			return []domain.MCPContentBlock{block}
		}
	}
	return nil
}

func parseMCPContentBlock(value any) domain.MCPContentBlock {
	item, _ := value.(map[string]any)
	blockType, _ := item["type"].(string)
	text, _ := item["text"].(string)
	uri, _ := item["uri"].(string)
	mimeType, _ := item["mimeType"].(string)
	blob, _ := item["blob"].(string)
	return domain.MCPContentBlock{Type: blockType, Text: text, URI: uri, MimeType: mimeType, Blob: blob}
}

func textFromMCPContentBlocks(blocks []domain.MCPContentBlock) string {
	parts := []string{}
	for _, block := range blocks {
		if block.Text != "" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func errorFromMCPResult(result map[string]any) string {
	if isError, _ := result["isError"].(bool); !isError {
		return ""
	}
	if content, _ := result["content"].(string); content != "" {
		return content
	}
	return "MCP tool returned an error"
}

func mcpStderrWriter(serverID string) io.Writer {
	path, err := mcpLogPath(serverID)
	if err != nil {
		return io.Discard
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return io.Discard
	}
	return file
}

const (
	defaultMCPLogReadLimit = 16000
	maxMCPLogReadLimit     = 100000
)

func readMCPServerLog(ctx context.Context, input domain.MCPServerLogInput) (domain.MCPServerLogResult, error) {
	select {
	case <-ctx.Done():
		return domain.MCPServerLogResult{}, ctx.Err()
	default:
	}
	path, err := mcpLogPath(input.ServerID)
	if err != nil {
		return domain.MCPServerLogResult{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.MCPServerLogResult{ServerID: strings.TrimSpace(input.ServerID)}, nil
		}
		return domain.MCPServerLogResult{}, err
	}
	if info.IsDir() {
		return domain.MCPServerLogResult{}, errors.New("mcp log path is a directory")
	}
	size := int(info.Size())
	limit := input.Limit
	if limit <= 0 {
		limit = defaultMCPLogReadLimit
	}
	if limit > maxMCPLogReadLimit {
		limit = maxMCPLogReadLimit
	}
	offset := input.Offset
	if input.Tail {
		offset = size - limit
	}
	if offset < 0 {
		offset = 0
	}
	if offset > size {
		offset = size
	}
	file, err := os.Open(path)
	if err != nil {
		return domain.MCPServerLogResult{}, err
	}
	defer file.Close()
	buffer := make([]byte, limit)
	n, err := file.ReadAt(buffer, int64(offset))
	if err != nil && n == 0 && offset < size {
		return domain.MCPServerLogResult{}, err
	}
	nextOffset := offset + n
	return domain.MCPServerLogResult{
		ServerID:   strings.TrimSpace(input.ServerID),
		Content:    string(buffer[:n]),
		Offset:     offset,
		NextOffset: nextOffset,
		Size:       size,
		Truncated:  nextOffset < size,
	}, nil
}

func mcpLogPath(serverID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cleanServerID := safeArtifactPart(strings.TrimSpace(serverID))
	if cleanServerID == "" {
		cleanServerID = "server"
	}
	return filepath.Join(home, ".aivo", "logs", "mcp-"+cleanServerID+".log"), nil
}
