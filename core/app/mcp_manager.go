package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"

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
	SaveMCPDiagnostic(context.Context, domain.MCPDiagnostic) (domain.MCPDiagnostic, error)
	ListMCPDiagnostics(context.Context, string, int) ([]domain.MCPDiagnostic, error)
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
		item.Diagnostics, _ = m.store.ListMCPDiagnostics(ctx, server.ID, 20)
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
	server.Description = strings.TrimSpace(server.Description)
	existing, existingErr := m.store.GetMCPServer(ctx, server.ID)
	hadExisting := existingErr == nil
	if len(server.Description) > 500 {
		return domain.MCPServerConfig{}, errors.New("mcp functional description must be at most 500 bytes")
	}
	previousBearerRef := ""
	if hadExisting && existing.AuthType == domain.MCPAuthBearer {
		previousBearerRef = strings.TrimSpace(existing.BearerTokenRef)
	}

	rawBearerToken := strings.TrimSpace(input.BearerToken)
	writtenBearerRef := ""
	previousBearerValue := ""
	if server.AuthType == domain.MCPAuthBearer {
		switch {
		case rawBearerToken != "":
			writtenBearerRef = mcpSecretRef(server, "access-token")
			if m.secrets == nil {
				return domain.MCPServerConfig{}, errors.New("mcp bearer credential store is unavailable")
			}
			previousBearerValue, _ = m.secrets.Get(ctx, writtenBearerRef)
			if err := m.secrets.Put(ctx, writtenBearerRef, rawBearerToken); err != nil {
				return domain.MCPServerConfig{}, err
			}
			server.BearerTokenRef = writtenBearerRef
			server.BearerTokenEnv = ""
		case strings.TrimSpace(server.BearerTokenEnv) != "":
			server.BearerTokenRef = ""
		case previousBearerRef != "":
			server.BearerTokenRef = previousBearerRef
		default:
			return domain.MCPServerConfig{}, errors.New("bearer authentication requires a token value or environment variable")
		}
	} else {
		server.BearerTokenRef = ""
	}
	saved, err := m.store.SaveMCPServer(ctx, server)
	if err != nil {
		m.restoreMCPBearerSecret(ctx, writtenBearerRef, previousBearerValue)
		return domain.MCPServerConfig{}, err
	}
	if previousBearerRef != "" && previousBearerRef != saved.BearerTokenRef {
		if err := m.secrets.Delete(ctx, previousBearerRef); err != nil {
			if hadExisting {
				_, _ = m.store.SaveMCPServer(ctx, existing)
			}
			m.restoreMCPBearerSecret(ctx, writtenBearerRef, previousBearerValue)
			return domain.MCPServerConfig{}, err
		}
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

func (m *MCPManager) restoreMCPBearerSecret(ctx context.Context, ref string, previousValue string) {
	if m == nil || m.secrets == nil || strings.TrimSpace(ref) == "" {
		return
	}
	if previousValue == "" {
		_ = m.secrets.Delete(ctx, ref)
		return
	}
	_ = m.secrets.Put(ctx, ref, previousValue)
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
		m.diagnostic(ctx, server.ID, domain.MCPDiagnosticError, err.Error(), nil)
		return domain.MCPProbeResult{OK: false, ServerID: server.ID, Status: domain.MCPServerStatusError, Error: err.Error()}, nil
	}
	if m.connections == nil {
		m.connections = NewMCPConnectionManager(m.store)
	}
	capabilities, err := m.connections.Probe(ctx, server)
	if err != nil {
		err = m.handleMCPOAuthChallenge(ctx, server, err)
		message := sanitizeMCPError(err.Error())
		m.diagnostic(ctx, server.ID, domain.MCPDiagnosticError, message, nil)
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
		m.diagnostic(ctx, server.ID, domain.MCPDiagnosticError, err.Error(), map[string]any{"method": "prompts/get", "name": name})
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
		m.diagnostic(ctx, server.ID, domain.MCPDiagnosticError, err.Error(), map[string]any{"method": "resources/read", "uri": uri})
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
		m.diagnostic(ctx, server.ID, domain.MCPDiagnosticError, err.Error(), map[string]any{"method": "oauth_discovery"})
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
	server, err := resolveMCPAuthSecrets(ctx, m.secrets, server)
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
			m.diagnostic(ctx, server.ID, domain.MCPDiagnosticError, saveErr.Error(), map[string]any{"method": "oauth_challenge"})
		}
		metadata["requestedScopes"] = requestedScopes
	}
	m.diagnostic(ctx, server.ID, domain.MCPDiagnosticError, challenge.Error(), metadata)
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

func (m *MCPManager) PrepareEnabledCatalogs(ctx context.Context) map[string]bool {
	failed := map[string]bool{}
	if m == nil || m.store == nil {
		return failed
	}
	servers, err := m.store.ListMCPServers(ctx, false)
	if err != nil {
		return failed
	}
	for _, server := range servers {
		if !server.Enabled {
			continue
		}
		tools, listErr := m.store.ListMCPTools(ctx, server.ID)
		if listErr == nil && len(tools) > 0 {
			continue
		}
		resources, resourcesErr := m.store.ListMCPResources(ctx, server.ID, false)
		templates, templatesErr := m.store.ListMCPResources(ctx, server.ID, true)
		if resourcesErr == nil && templatesErr == nil && (len(resources) > 0 || len(templates) > 0) {
			continue
		}
		probe, probeErr := m.Probe(ctx, domain.MCPProbeInput{ServerID: server.ID})
		if probeErr != nil || !probe.OK {
			failed[toolSourceEligibilityKey(domain.ToolSourceMCP, server.ID)] = true
		}
	}
	return failed
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
		if (err != nil || len(tools) == 0) && allowProbe {
			probe, probeErr := m.Probe(ctx, domain.MCPProbeInput{ServerID: server.ID})
			if probeErr == nil {
				tools, err = probe.Tools, nil
			}
		}
		if err != nil {
			continue
		}
		for _, tool := range tools {
			selectionGroup := mcpToolSelectionGroup(server)
			spec := domain.ToolSpec{
				Name: mcpToolName(server, tool), Description: tool.Description, InputSchema: tool.InputSchema,
				Namespace: mcpServerToolPrefix(server), NamespaceDescription: server.Description,
				Capability: firstNonEmptyApp(tool.Capability, "mcp.read"), RiskLevel: firstNonEmptyApp(tool.RiskLevel, "medium"),
				Category: "mcp", Toolsets: []string{"mcp", "coding"}, RequiresNetwork: server.Transport != domain.MCPTransportStdio,
				ActivationPolicy: "auto", SelectionGroup: selectionGroup, ImplementationHash: mcpAdapterImplementationHash(server, tool),
			}
			if registerErr := registry.RegisterScoped(&MCPRuntimeTool{manager: m, server: server, tool: tool, spec: spec}, domain.ToolSourceMCP, server.ID, firstNonEmptyApp(tool.TimeUpdated, server.TimeUpdated, "v1")); registerErr != nil {
				m.diagnostic(ctx, server.ID, "error", "MCP tool registration failed", map[string]any{"tool": tool.Name, "error": registerErr.Error()})
			}
		}
		m.registerMCPResourceUtilityTools(ctx, registry, server)
	}
}

func (m *MCPManager) diagnostic(ctx context.Context, serverID string, level string, message string, metadata map[string]any) {
	if m == nil || m.store == nil || message == "" {
		return
	}
	_, _ = m.store.SaveMCPDiagnostic(ctx, domain.MCPDiagnostic{ServerID: serverID, Level: level, Message: sanitizeMCPError(message), Metadata: metadata})
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
