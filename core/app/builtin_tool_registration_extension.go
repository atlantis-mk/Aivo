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
	toolRegistrationExtensionID  = "aivo.tools"
	toolRegistrationMCPName      = "aivo_tools_register_mcp"
	toolRegistrationResourceName = "aivo_tools_register_resource"
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
	case toolRegistrationResourceName:
		var input domain.ResourceRegistrationProposalInput
		if err := decodeStrictToolArgs(args, &input); err != nil {
			return primitiveError(name, "invalid_arguments", err), nil
		}
		result, err := c.service.commitResourceRegistrationProposal(ctx, input, execCtx)
		if err != nil {
			return primitiveError(name, resourceRegistrationErrorCode(err), errors.New(err.Error())), nil
		}
		content := fmt.Sprintf("Installed Skill %s into Aivo-managed %s Skill storage with %d files.", result.Name, result.Scope, result.FileCount)
		return domain.ToolResult{Name: name, OK: true, Content: content, Structured: projectStructuredResult(result)}, nil
	default:
		return primitiveError(name, "invalid_arguments", errors.New("unknown tool registration operation")), nil
	}
}

func (*toolRegistrationBuiltinExtensionClient) UIEvent(context.Context, string, string, any) (any, error) {
	return nil, errors.New("tool registration extension does not expose a view")
}

func (*toolRegistrationBuiltinExtensionClient) Shutdown(context.Context) error { return nil }
