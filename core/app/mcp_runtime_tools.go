package app

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"aivo/core/domain"
)

func (m *MCPManager) registerMCPResourceUtilityTools(ctx context.Context, registry *Registry, server domain.MCPServerConfig) {
	if registry == nil {
		return
	}
	base := generatedToolName("mcp", "host", firstNonEmptyApp(server.ID, server.Name))
	selectionGroup := mcpToolSelectionGroup(server)
	utilities := []MCPResourceUtilityTool{
		{manager: m, server: server, kind: "list_resources", spec: domain.ToolSpec{
			Name: generatedToolName(base, "list_resources"), Description: "List resources exposed by MCP server " + firstNonEmptyApp(server.DisplayName, server.Name, server.ID) + ".",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{
				"cursor": map[string]any{"type": "string", "description": "Optional pagination cursor returned by a previous resources/list call."},
			}, "additionalProperties": false},
			Namespace: mcpServerToolPrefix(server), NamespaceDescription: server.Description, Capability: "mcp.read", RiskLevel: "low",
			Category: "mcp", Toolsets: []string{"mcp", "coding"}, RequiresNetwork: server.Transport != domain.MCPTransportStdio, ActivationPolicy: "auto", SelectionGroup: cloneToolSelectionGroup(selectionGroup), ImplementationHash: mcpResourceAdapterImplementationHash(server, "list_resources"),
		}},
		{manager: m, server: server, kind: "list_resource_templates", spec: domain.ToolSpec{
			Name: generatedToolName(base, "list_resource_templates"), Description: "List resource templates exposed by MCP server " + firstNonEmptyApp(server.DisplayName, server.Name, server.ID) + ".",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{
				"cursor": map[string]any{"type": "string", "description": "Optional pagination cursor returned by a previous resources/templates/list call."},
			}, "additionalProperties": false},
			Namespace: mcpServerToolPrefix(server), NamespaceDescription: server.Description, Capability: "mcp.read", RiskLevel: "low",
			Category: "mcp", Toolsets: []string{"mcp", "coding"}, RequiresNetwork: server.Transport != domain.MCPTransportStdio, ActivationPolicy: "auto", SelectionGroup: cloneToolSelectionGroup(selectionGroup), ImplementationHash: mcpResourceAdapterImplementationHash(server, "list_resource_templates"),
		}},
		{manager: m, server: server, kind: "read_resource", spec: domain.ToolSpec{
			Name: generatedToolName(base, "read_resource"), Description: "Read a resource URI from MCP server " + firstNonEmptyApp(server.DisplayName, server.Name, server.ID) + ".",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{
				"uri": map[string]any{"type": "string", "description": "Exact MCP resource URI to read."},
			}, "required": []string{"uri"}, "additionalProperties": false},
			Namespace: mcpServerToolPrefix(server), NamespaceDescription: server.Description, Capability: "mcp.read", RiskLevel: "low",
			Category: "mcp", Toolsets: []string{"mcp", "coding"}, RequiresNetwork: server.Transport != domain.MCPTransportStdio, ActivationPolicy: "auto", SelectionGroup: cloneToolSelectionGroup(selectionGroup), ImplementationHash: mcpResourceAdapterImplementationHash(server, "read_resource"),
		}},
	}
	for i := range utilities {
		tool := utilities[i]
		if err := registry.RegisterScoped(&tool, domain.ToolSourceMCP, server.ID, firstNonEmptyApp(server.TimeUpdated, "v1")); err != nil {
			m.diagnostic(ctx, server.ID, "error", "MCP resource utility registration failed", map[string]any{"tool": tool.spec.Name, "error": err.Error()})
		}
	}
}

func mcpAdapterImplementationHash(server domain.MCPServerConfig, tool domain.MCPToolRecord) string {
	raw, _ := json.Marshal(map[string]any{"adapter": "mcp-v1", "serverId": server.ID, "transport": server.Transport, "tool": tool.Name, "schema": tool.InputSchema})
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}

func mcpResourceAdapterImplementationHash(server domain.MCPServerConfig, kind string) string {
	raw, _ := json.Marshal(map[string]any{"adapter": "mcp-v1", "serverId": server.ID, "transport": server.Transport, "resource": kind})
	return fmt.Sprintf("%x", sha256.Sum256(raw))
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
		server, err = resolveMCPAuthSecrets(ctx, t.secrets, server)
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
