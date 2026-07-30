package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"aivo/core/domain"
)

// mcpSDKClient adapts the official SDK session to Aivo's narrow app-layer MCP
// boundary. The SDK owns protocol negotiation, initialization, cancellation,
// sessions, and transport conformance; Aivo owns policy and normalization.
type mcpSDKClient struct {
	session        *mcp.ClientSession
	catalogChanged atomic.Bool
}

func startMCPSDKClient(ctx context.Context, server domain.MCPServerConfig) (*mcpSDKClient, error) {
	client := &mcpSDKClient{}
	capabilities := &mcp.ClientCapabilities{}
	if len(mcpRootEntries(server)) > 0 {
		capabilities.RootsV2 = &mcp.RootCapabilities{ListChanged: true}
	}
	sdkClient := mcp.NewClient(&mcp.Implementation{Name: "aivo", Version: "1.0.0"}, &mcp.ClientOptions{
		Capabilities: capabilities,
		ToolListChangedHandler: func(context.Context, *mcp.ToolListChangedRequest) {
			client.catalogChanged.Store(true)
		},
		PromptListChangedHandler: func(context.Context, *mcp.PromptListChangedRequest) {
			client.catalogChanged.Store(true)
		},
		ResourceListChangedHandler: func(context.Context, *mcp.ResourceListChangedRequest) {
			client.catalogChanged.Store(true)
		},
	})
	for _, root := range mcpRootEntries(server) {
		uri, _ := root["uri"].(string)
		name, _ := root["name"].(string)
		sdkClient.AddRoots(&mcp.Root{URI: uri, Name: name})
	}

	transport, err := mcpSDKTransport(server)
	if err != nil {
		return nil, err
	}
	session, err := sdkClient.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("initialize MCP SDK session: %w", err)
	}
	client.session = session
	return client, nil
}

func mcpSDKTransport(server domain.MCPServerConfig) (mcp.Transport, error) {
	transport := server.Transport
	if transport == "" {
		transport = domain.MCPTransportStdio
	}
	switch transport {
	case domain.MCPTransportStdio:
		if strings.TrimSpace(server.Command) == "" {
			return nil, errors.New("stdio MCP server command is required")
		}
		cmd := exec.Command(server.Command, server.Args...)
		if server.CWD != "" {
			cmd.Dir = server.CWD
		} else if wd, err := os.Getwd(); err == nil {
			cmd.Dir = wd
		}
		cmd.Env = SanitizedEnvironment(firstNonEmptyApp(cmd.Dir, "."), defaultEnvAllowlist(), server.Env, nil)
		cmd.Stderr = mcpStderrWriter(server.ID)
		setProcessGroup(cmd)
		return &mcp.CommandTransport{Command: cmd}, nil
	case domain.MCPTransportStreamableHTTP:
		if strings.TrimSpace(server.URL) == "" {
			return nil, errors.New("http MCP server URL is required")
		}
		return &mcp.StreamableClientTransport{Endpoint: server.URL, HTTPClient: mcpSDKHTTPClient(server)}, nil
	case domain.MCPTransportSSE:
		if strings.TrimSpace(server.URL) == "" {
			return nil, errors.New("SSE MCP server URL is required")
		}
		return &mcp.SSEClientTransport{Endpoint: server.URL, HTTPClient: mcpSDKHTTPClient(server)}, nil
	default:
		return nil, fmt.Errorf("unsupported mcp transport %s", transport)
	}
}

func mcpSDKHTTPClient(server domain.MCPServerConfig) *http.Client {
	return &http.Client{Transport: &mcpSDKRoundTripper{server: server, base: http.DefaultTransport}}
}

type mcpSDKRoundTripper struct {
	server domain.MCPServerConfig
	base   http.RoundTripper
}

func (t *mcpSDKRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.Header = req.Header.Clone()
	for key, value := range t.server.Headers {
		if strings.TrimSpace(key) != "" {
			cloned.Header.Set(key, value)
		}
	}
	if err := applyMCPRequestAuth(cloned, t.server); err != nil {
		return nil, err
	}
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	resp, err := base.RoundTrip(cloned)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		return nil, mcpHTTPAuthError(resp.StatusCode, resp.Header.Get("WWW-Authenticate"), body)
	}
	return resp, nil
}

func applyMCPRequestAuth(req *http.Request, server domain.MCPServerConfig) error {
	authType := strings.TrimSpace(server.AuthType)
	if authType == "" || authType == domain.MCPAuthNone {
		return nil
	}
	switch authType {
	case domain.MCPAuthBearer:
		envName := strings.TrimSpace(server.BearerTokenEnv)
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
		if token := strings.TrimSpace(server.OAuthAccessToken); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
			return nil
		}
		envName := strings.TrimSpace(server.BearerTokenEnv)
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

func (c *mcpSDKClient) call(ctx context.Context, method string, params map[string]any) (map[string]any, error) {
	if c == nil || c.session == nil {
		return nil, errors.New("MCP SDK session is closed")
	}
	var result any
	switch method {
	case "ping":
		if err := c.session.Ping(ctx, &mcp.PingParams{}); err != nil {
			return nil, err
		}
		return map[string]any{}, nil
	case "tools/list":
		p := &mcp.ListToolsParams{}
		decodeMCPParams(params, p)
		var err error
		result, err = c.session.ListTools(ctx, p)
		if err != nil {
			return nil, err
		}
	case "tools/call":
		name, _ := params["name"].(string)
		var err error
		result, err = c.session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: params["arguments"]})
		if err != nil {
			return nil, err
		}
	case "prompts/list":
		p := &mcp.ListPromptsParams{}
		decodeMCPParams(params, p)
		var err error
		result, err = c.session.ListPrompts(ctx, p)
		if err != nil {
			return nil, err
		}
	case "prompts/get":
		p := &mcp.GetPromptParams{}
		decodeMCPParams(params, p)
		var err error
		result, err = c.session.GetPrompt(ctx, p)
		if err != nil {
			return nil, err
		}
	case "resources/list":
		p := &mcp.ListResourcesParams{}
		decodeMCPParams(params, p)
		var err error
		result, err = c.session.ListResources(ctx, p)
		if err != nil {
			return nil, err
		}
	case "resources/templates/list":
		p := &mcp.ListResourceTemplatesParams{}
		decodeMCPParams(params, p)
		var err error
		result, err = c.session.ListResourceTemplates(ctx, p)
		if err != nil {
			return nil, err
		}
	case "resources/read":
		p := &mcp.ReadResourceParams{}
		decodeMCPParams(params, p)
		var err error
		result, err = c.session.ReadResource(ctx, p)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported MCP SDK method %s", method)
	}
	if result == nil {
		return nil, fmt.Errorf("MCP %s returned no result", method)
	}
	return mcpResultMap(result)
}

func decodeMCPParams(params map[string]any, target any) {
	raw, _ := json.Marshal(params)
	_ = json.Unmarshal(raw, target)
}

func mcpResultMap(result any) (map[string]any, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	var mapped map[string]any
	if err := json.Unmarshal(raw, &mapped); err != nil {
		return nil, err
	}
	return mapped, nil
}

func (c *mcpSDKClient) consumeToolsListChanged() bool {
	return c != nil && c.catalogChanged.Swap(false)
}

func (c *mcpSDKClient) close() {
	if c != nil && c.session != nil {
		_ = c.session.Close()
		c.session = nil
	}
}
