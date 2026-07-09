package app

import (
	"strings"

	"aivo/core/domain"
)

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
