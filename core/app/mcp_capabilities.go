package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"aivo/core/domain"
)

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
