package app

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"aivo/core/domain"
)

const (
	toolRegistrationExtensionID = "aivo.tools"
	toolRegistrationListName    = "aivo_tools_list_mcp"
	toolRegistrationMCPName     = "aivo_tools_register_mcp"
)

//go:embed builtin_extensions/aivo.tools.json
var toolRegistrationExtensionManifest []byte

type toolRegistrationBuiltinExtensionClient struct{ service *Service }

func (c *toolRegistrationBuiltinExtensionClient) Initialize(_ context.Context, manifest domain.ExtensionManifest) error {
	if c == nil || c.service == nil {
		return errors.New("tool registration service is unavailable")
	}
	if manifest.ID != toolRegistrationExtensionID || manifest.Runtime.Type != domain.ExtensionRuntimeBuiltin {
		return errors.New("invalid tool registration extension manifest")
	}
	return nil
}

func (c *toolRegistrationBuiltinExtensionClient) Execute(ctx context.Context, name string, args json.RawMessage, execCtx domain.ToolExecutionContext) (domain.ToolResult, error) {
	switch name {
	case toolRegistrationListName:
		var input struct {
			IncludeDisabled bool `json:"includeDisabled"`
		}
		if err := decodeStrictToolArgs(args, &input); err != nil {
			return primitiveError(name, "invalid_arguments", err), nil
		}
		items, err := c.service.ListMCPServers(ctx, domain.MCPServerListInput{IncludeDisabled: input.IncludeDisabled, IncludeTools: true})
		if err != nil {
			return primitiveError(name, "mcp_list_failed", errors.New(sanitizeMCPError(err.Error()))), nil
		}
		summaries := make([]map[string]any, 0, len(items))
		lines := make([]string, 0, len(items))
		for _, item := range items {
			displayName := firstNonEmptyApp(item.Server.DisplayName, item.Server.Name, item.Server.ID)
			summaries = append(summaries, map[string]any{
				"id": item.Server.ID, "displayName": displayName, "description": bounded(item.Server.Description, 300),
				"transport": item.Server.Transport, "enabled": item.Server.Enabled, "status": item.Server.Status, "toolCount": len(item.Tools),
			})
			lines = append(lines, fmt.Sprintf("- %s | %s | %s | enabled=%t | tools=%d", item.Server.ID, displayName, item.Server.Transport, item.Server.Enabled, len(item.Tools)))
		}
		if len(lines) == 0 {
			lines = append(lines, "No MCP sources are registered.")
		}
		return domain.ToolResult{Name: name, OK: true, Content: strings.Join(lines, "\n"), Structured: map[string]any{"sources": summaries}}, nil
	case toolRegistrationMCPName:
		var input domain.MCPRegistrationProposalInput
		if err := decodeStrictToolArgs(args, &input); err != nil {
			return primitiveError(name, "invalid_arguments", err), nil
		}
		result, err := c.service.commitMCPRegistrationProposal(ctx, input, execCtx)
		if err != nil {
			return primitiveError(name, mcpRegistrationErrorCode(err), errors.New(sanitizeMCPError(err.Error()))), nil
		}
		toolNames := append([]string(nil), result.ToolNames...)
		sort.Strings(toolNames)
		content := fmt.Sprintf("Registered MCP source %s (%s). It is ready and globally eligible for later conversations.", result.DisplayName, result.ID)
		if len(toolNames) > 0 {
			content += " Available tools: " + strings.Join(toolNames, ", ")
		}
		return domain.ToolResult{Name: name, OK: true, Content: content, Structured: projectStructuredResult(result)}, nil
	default:
		return primitiveError(name, "invalid_arguments", errors.New("unknown tool registration operation")), nil
	}
}

func (*toolRegistrationBuiltinExtensionClient) UIEvent(context.Context, string, string, any) (any, error) {
	return nil, errors.New("tool registration extension does not expose a view")
}

func (*toolRegistrationBuiltinExtensionClient) Shutdown(context.Context) error { return nil }
