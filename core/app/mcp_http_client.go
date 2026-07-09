package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"aivo/core/domain"
)

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
