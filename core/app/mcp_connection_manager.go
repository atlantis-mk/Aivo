package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"aivo/core/domain"
)

const (
	defaultMCPToolTimeout      = 30 * time.Second
	defaultMCPConnectTimeout   = 30 * time.Second
	defaultMCPKeepalive        = 2 * time.Minute
	minimumMCPKeepalive        = 5 * time.Second
	mcpKeepaliveFailureBackoff = 250 * time.Millisecond
)

type mcpRPCClient interface {
	call(context.Context, string, map[string]any) (map[string]any, error)
	consumeToolsListChanged() bool
	close()
}

type MCPConnectionManager struct {
	store mcpStore
	mu    sync.Mutex
	conns map[string]*mcpManagedConnection
}

func NewMCPConnectionManager(store mcpStore) *MCPConnectionManager {
	return &MCPConnectionManager{store: store, conns: map[string]*mcpManagedConnection{}}
}

func (m *MCPConnectionManager) Probe(ctx context.Context, server domain.MCPServerConfig) (mcpServerCapabilities, error) {
	conn, err := m.ensureConnection(ctx, server)
	if err != nil {
		return mcpServerCapabilities{}, err
	}
	capabilities, err := conn.capabilities(ctx)
	if err != nil {
		m.drop(server.ID)
		return mcpServerCapabilities{}, err
	}
	m.persistCapabilities(ctx, server.ID, capabilities)
	return capabilities, nil
}

func (m *MCPConnectionManager) CallMethod(ctx context.Context, server domain.MCPServerConfig, method string, params map[string]any) (map[string]any, error) {
	result, err := m.callMethodOnce(ctx, server, method, params)
	if err == nil {
		return result, nil
	}
	m.drop(server.ID)
	result, retryErr := m.callMethodOnce(ctx, server, method, params)
	if retryErr != nil {
		return nil, retryErr
	}
	return result, nil
}

func (m *MCPConnectionManager) CallTool(ctx context.Context, server domain.MCPServerConfig, name string, args json.RawMessage) (map[string]any, error) {
	var arguments any = map[string]any{}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &arguments); err != nil {
			return nil, errors.New("invalid MCP tool arguments")
		}
	}
	return m.CallMethod(ctx, server, "tools/call", map[string]any{"name": name, "arguments": arguments})
}

func (m *MCPConnectionManager) callMethodOnce(ctx context.Context, server domain.MCPServerConfig, method string, params map[string]any) (map[string]any, error) {
	conn, err := m.ensureConnection(ctx, server)
	if err != nil {
		return nil, err
	}
	result, err := conn.call(ctx, method, params)
	if err != nil {
		return nil, err
	}
	if capabilities, changed := conn.refreshToolsIfChanged(ctx); changed {
		m.persistCapabilities(ctx, server.ID, capabilities)
	}
	return result, nil
}

func (m *MCPConnectionManager) ensureConnection(ctx context.Context, server domain.MCPServerConfig) (*mcpManagedConnection, error) {
	if server.Transport == "" {
		server.Transport = domain.MCPTransportStdio
	}
	key := strings.TrimSpace(server.ID)
	if key == "" {
		key = strings.TrimSpace(server.Name)
	}
	if key == "" {
		return nil, errors.New("MCP server id or name is required")
	}
	fingerprint := mcpServerConnectionFingerprint(server)
	m.mu.Lock()
	conn := m.conns[key]
	if conn == nil || conn.fingerprint != fingerprint {
		if conn != nil {
			conn.close()
		}
		conn = &mcpManagedConnection{
			server:      server,
			fingerprint: fingerprint,
			store:       m.store,
		}
		m.conns[key] = conn
	}
	m.mu.Unlock()
	if err := conn.ensureInitialized(ctx); err != nil {
		m.drop(key)
		return nil, err
	}
	return conn, nil
}

func (m *MCPConnectionManager) drop(serverID string) {
	key := strings.TrimSpace(serverID)
	if key == "" {
		return
	}
	m.mu.Lock()
	conn := m.conns[key]
	delete(m.conns, key)
	m.mu.Unlock()
	if conn != nil {
		conn.close()
	}
}

func (m *MCPConnectionManager) Close() {
	if m == nil {
		return
	}
	m.mu.Lock()
	conns := make([]*mcpManagedConnection, 0, len(m.conns))
	for key, conn := range m.conns {
		conns = append(conns, conn)
		delete(m.conns, key)
	}
	m.mu.Unlock()
	for _, conn := range conns {
		conn.close()
	}
}

func (m *MCPConnectionManager) persistCapabilities(ctx context.Context, serverID string, capabilities mcpServerCapabilities) {
	if m == nil || m.store == nil || strings.TrimSpace(serverID) == "" {
		return
	}
	_ = m.store.ReplaceMCPTools(ctx, serverID, capabilities.Tools)
	_ = m.store.ReplaceMCPPrompts(ctx, serverID, capabilities.Prompts)
	resources := append([]domain.MCPResourceRecord{}, capabilities.Resources...)
	resources = append(resources, capabilities.ResourceTemplates...)
	_ = m.store.ReplaceMCPResources(ctx, serverID, resources)
}

type mcpManagedConnection struct {
	mu                sync.Mutex
	server            domain.MCPServerConfig
	fingerprint       string
	client            mcpRPCClient
	initialized       bool
	capabilitiesCache mcpServerCapabilities
	store             mcpStore
	keepaliveCancel   context.CancelFunc
	closed            bool
}

func (c *mcpManagedConnection) ensureInitialized(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return errors.New("MCP connection is closed")
	}
	if c.initialized && c.client != nil {
		return nil
	}
	client, err := startMCPClient(ctx, c.server)
	if err != nil {
		return err
	}
	c.client = client
	callCtx, cancel := withMCPTimeout(ctx, c.server, true)
	defer cancel()
	if _, err := c.client.call(callCtx, "initialize", mcpInitializeParams(c.server)); err != nil {
		c.client.close()
		c.client = nil
		return err
	}
	c.capabilitiesCache = discoverMCPCapabilitiesWithClient(ctx, c.server, c.client)
	c.initialized = true
	c.startKeepaliveLocked()
	return nil
}

func (c *mcpManagedConnection) capabilities(ctx context.Context) (mcpServerCapabilities, error) {
	if err := c.ensureInitialized(ctx); err != nil {
		return mcpServerCapabilities{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.capabilitiesCache = discoverMCPCapabilitiesWithClient(ctx, c.server, c.client)
	return c.capabilitiesCache, nil
}

func (c *mcpManagedConnection) call(ctx context.Context, method string, params map[string]any) (map[string]any, error) {
	if err := c.ensureInitialized(ctx); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.client == nil {
		return nil, errors.New("MCP connection is closed")
	}
	callCtx, cancel := withMCPTimeout(ctx, c.server, false)
	defer cancel()
	return c.client.call(callCtx, method, params)
}

func (c *mcpManagedConnection) refreshToolsIfChanged(ctx context.Context) (mcpServerCapabilities, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.client == nil || !c.client.consumeToolsListChanged() {
		return c.capabilitiesCache, false
	}
	c.capabilitiesCache = discoverMCPCapabilitiesWithClient(ctx, c.server, c.client)
	return c.capabilitiesCache, true
}

func (c *mcpManagedConnection) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	c.initialized = false
	if c.keepaliveCancel != nil {
		c.keepaliveCancel()
		c.keepaliveCancel = nil
	}
	if c.client != nil {
		c.client.close()
		c.client = nil
	}
}

func (c *mcpManagedConnection) startKeepaliveLocked() {
	if c.keepaliveCancel != nil {
		return
	}
	interval := defaultMCPKeepalive
	if c.server.ConnectTimeoutSeconds > 0 {
		interval = time.Duration(c.server.ConnectTimeoutSeconds) * time.Second
	}
	if interval < minimumMCPKeepalive {
		interval = minimumMCPKeepalive
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.keepaliveCancel = cancel
	go c.keepaliveLoop(ctx, interval)
}

func (c *mcpManagedConnection) keepaliveLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.keepalive(ctx); err != nil {
				time.Sleep(mcpKeepaliveFailureBackoff)
				c.close()
				return
			}
		}
	}
}

func (c *mcpManagedConnection) keepalive(parent context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.client == nil {
		return nil
	}
	ctx, cancel := withMCPTimeout(parent, c.server, false)
	defer cancel()
	if _, err := c.client.call(ctx, "ping", map[string]any{}); err == nil {
		return nil
	}
	listCtx, listCancel := withMCPTimeout(parent, c.server, false)
	defer listCancel()
	_, err := c.client.call(listCtx, "tools/list", map[string]any{})
	return err
}

func startMCPClient(ctx context.Context, server domain.MCPServerConfig) (mcpRPCClient, error) {
	if server.Transport == "" {
		server.Transport = domain.MCPTransportStdio
	}
	switch server.Transport {
	case domain.MCPTransportStreamableHTTP, domain.MCPTransportSSE:
		return newMCPHTTPClient(server)
	case domain.MCPTransportStdio:
		return startMCPStdio(ctx, server)
	default:
		return nil, fmt.Errorf("unsupported mcp transport %s", server.Transport)
	}
}

func discoverMCPCapabilitiesWithClient(ctx context.Context, server domain.MCPServerConfig, client mcpRPCClient) mcpServerCapabilities {
	listCtx, listCancel := withMCPTimeout(ctx, server, false)
	result, _ := client.call(listCtx, "tools/list", map[string]any{})
	listCancel()
	tools := parseMCPTools(server, result)
	if client.consumeToolsListChanged() {
		refreshCtx, refreshCancel := withMCPTimeout(ctx, server, false)
		refreshed, err := client.call(refreshCtx, "tools/list", map[string]any{})
		refreshCancel()
		if err == nil {
			if next := parseMCPTools(server, refreshed); len(next) > 0 {
				tools = next
			}
		}
	}
	promptsCtx, promptsCancel := withMCPTimeout(ctx, server, false)
	promptsResult, _ := client.call(promptsCtx, "prompts/list", map[string]any{})
	promptsCancel()
	resourcesCtx, resourcesCancel := withMCPTimeout(ctx, server, false)
	resourcesResult, _ := client.call(resourcesCtx, "resources/list", map[string]any{})
	resourcesCancel()
	templatesCtx, templatesCancel := withMCPTimeout(ctx, server, false)
	templatesResult, _ := client.call(templatesCtx, "resources/templates/list", map[string]any{})
	templatesCancel()
	return mcpServerCapabilities{
		Tools:             tools,
		Prompts:           parseMCPPrompts(server, promptsResult),
		Resources:         parseMCPResources(server, resourcesResult, false),
		ResourceTemplates: parseMCPResources(server, templatesResult, true),
	}
}

func withMCPTimeout(ctx context.Context, server domain.MCPServerConfig, connect bool) (context.Context, context.CancelFunc) {
	timeout := time.Duration(server.TimeoutSeconds) * time.Second
	if connect && server.ConnectTimeoutSeconds > 0 {
		timeout = time.Duration(server.ConnectTimeoutSeconds) * time.Second
	}
	if timeout <= 0 {
		if connect {
			timeout = defaultMCPConnectTimeout
		} else {
			timeout = defaultMCPToolTimeout
		}
	}
	return context.WithTimeout(ctx, timeout)
}

func mcpServerConnectionFingerprint(server domain.MCPServerConfig) string {
	raw, _ := json.Marshal(map[string]any{
		"id": server.ID, "name": server.Name, "transport": server.Transport,
		"command": server.Command, "args": server.Args, "cwd": server.CWD, "env": server.Env,
		"url": server.URL, "headers": server.Headers, "authType": server.AuthType,
		"bearerTokenEnv": server.BearerTokenEnv, "oauthAccessTokenRef": server.OAuthAccessTokenRef,
		"oauthRefreshTokenRef": server.OAuthRefreshTokenRef, "oauthExpiresAt": server.OAuthExpiresAt,
		"roots": server.Roots, "timeoutSeconds": server.TimeoutSeconds,
		"connectTimeoutSeconds": server.ConnectTimeoutSeconds, "updated": server.TimeUpdated,
	})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
