package app

import (
	"strings"
	"unicode"

	"aivo/core/domain"
)

func mcpToolName(server domain.MCPServerConfig, tool domain.MCPToolRecord) string {
	return generatedToolName(mcpServerToolPrefix(server), tool.Name)
}

func mcpServerToolPrefix(server domain.MCPServerConfig) string {
	prefix := strings.TrimSpace(server.ID)
	if prefix == "" {
		prefix = strings.TrimSpace(server.Name)
	}
	return generatedToolName("mcp", prefix)
}

func mcpToolSelectionGroup(server domain.MCPServerConfig) *domain.ToolSelectionGroup {
	name := firstNonEmptyApp(server.DisplayName, server.Name, server.ID)
	return &domain.ToolSelectionGroup{
		ID:          generatedToolName("mcp_group", firstNonEmptyApp(server.ID, server.Name)),
		Name:        name,
		Description: strings.TrimSpace(server.Description),
	}
}

// sanitizeMCPExtensionNameComponent produces one stable Manifest v2 namespace
// component. The original MCP tool name is retained separately for execution.
func sanitizeMCPExtensionNameComponent(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var out strings.Builder
	separator := false
	for _, current := range value {
		if current <= unicode.MaxASCII && (unicode.IsLetter(current) || unicode.IsDigit(current)) {
			if separator && out.Len() > 0 {
				out.WriteByte('_')
			}
			out.WriteRune(current)
			separator = false
			continue
		}
		separator = true
	}
	if out.Len() == 0 {
		return "server"
	}
	return out.String()
}

func sanitizeMCPToolNameComponent(value string) string {
	value = strings.TrimSpace(value)
	value = strings.NewReplacer(" ", "_", "-", "_", ".", "_", ":", "_", "/", "_", "\\", "_").Replace(value)
	if value == "" {
		return "server"
	}
	return value
}
