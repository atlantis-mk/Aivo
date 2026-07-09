package app

import (
	"context"
	"errors"
	"strings"

	"aivo/core/domain"
)

func (m *memoryProviderStore) SavePluginInstall(_ context.Context, plugin domain.PluginInstall) (domain.PluginInstall, error) {
	m.plugins = append(m.plugins, plugin)
	return plugin, nil
}

func (m *memoryProviderStore) GetPluginInstall(_ context.Context, id string) (domain.PluginInstall, error) {
	for _, plugin := range m.plugins {
		if plugin.ID == id {
			return plugin, nil
		}
	}
	return domain.PluginInstall{}, errors.New("plugin not found")
}

func (m *memoryProviderStore) ListPluginInstalls(_ context.Context, includeDisabled bool) ([]domain.PluginInstall, error) {
	out := make([]domain.PluginInstall, 0, len(m.plugins))
	for _, plugin := range m.plugins {
		if includeDisabled || plugin.Enabled {
			out = append(out, plugin)
		}
	}
	return out, nil
}

func (m *memoryProviderStore) SetPluginEnabled(_ context.Context, id string, enabled bool, status string, statusMessage string) (domain.PluginInstall, error) {
	for i, plugin := range m.plugins {
		if plugin.ID == id {
			plugin.Enabled = enabled
			plugin.Status = status
			plugin.Error = statusMessage
			m.plugins[i] = plugin
			return plugin, nil
		}
	}
	return domain.PluginInstall{}, errors.New("plugin not found")
}

func (m *memoryProviderStore) SavePluginDiagnostic(_ context.Context, diagnostic domain.PluginDiagnostic) (domain.PluginDiagnostic, error) {
	m.pluginDiagnostics = append(m.pluginDiagnostics, diagnostic)
	return diagnostic, nil
}

func (m *memoryProviderStore) ListPluginDiagnostics(_ context.Context, pluginID string, serverID string, limit int) ([]domain.PluginDiagnostic, error) {
	out := []domain.PluginDiagnostic{}
	for _, diagnostic := range m.pluginDiagnostics {
		if pluginID != "" && diagnostic.PluginID != pluginID {
			continue
		}
		if serverID != "" && diagnostic.ServerID != serverID {
			continue
		}
		out = append(out, diagnostic)
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *memoryProviderStore) SaveMCPServer(_ context.Context, server domain.MCPServerConfig) (domain.MCPServerConfig, error) {
	m.mcpServers = append(m.mcpServers, server)
	return server, nil
}

func (m *memoryProviderStore) GetMCPServer(_ context.Context, id string) (domain.MCPServerConfig, error) {
	for _, server := range m.mcpServers {
		if server.ID == id {
			return server, nil
		}
	}
	return domain.MCPServerConfig{}, errors.New("mcp server not found")
}

func (m *memoryProviderStore) ListMCPServers(_ context.Context, includeDisabled bool) ([]domain.MCPServerConfig, error) {
	out := make([]domain.MCPServerConfig, 0, len(m.mcpServers))
	for _, server := range m.mcpServers {
		if includeDisabled || server.Enabled {
			out = append(out, server)
		}
	}
	return out, nil
}

func (m *memoryProviderStore) SetMCPServerEnabled(_ context.Context, id string, enabled bool, status string, statusMessage string) (domain.MCPServerConfig, error) {
	for i, server := range m.mcpServers {
		if server.ID == id {
			server.Enabled = enabled
			server.Status = status
			server.Error = statusMessage
			m.mcpServers[i] = server
			return server, nil
		}
	}
	return domain.MCPServerConfig{}, errors.New("mcp server not found")
}

func (m *memoryProviderStore) DeleteMCPServer(_ context.Context, id string) error {
	next := m.mcpServers[:0]
	for _, server := range m.mcpServers {
		if server.ID != id {
			next = append(next, server)
		}
	}
	m.mcpServers = next
	delete(m.mcpTools, id)
	return nil
}

func (m *memoryProviderStore) ReplaceMCPTools(_ context.Context, serverID string, tools []domain.MCPToolRecord) error {
	if m.mcpTools == nil {
		m.mcpTools = map[string][]domain.MCPToolRecord{}
	}
	m.mcpTools[serverID] = append([]domain.MCPToolRecord(nil), tools...)
	return nil
}

func (m *memoryProviderStore) ListMCPTools(_ context.Context, serverID string) ([]domain.MCPToolRecord, error) {
	if strings.TrimSpace(serverID) != "" {
		return append([]domain.MCPToolRecord(nil), m.mcpTools[serverID]...), nil
	}
	out := []domain.MCPToolRecord{}
	for _, tools := range m.mcpTools {
		out = append(out, tools...)
	}
	return out, nil
}

func (m *memoryProviderStore) ReplaceMCPPrompts(context.Context, string, []domain.MCPPromptRecord) error {
	return nil
}

func (m *memoryProviderStore) ListMCPPrompts(context.Context, string) ([]domain.MCPPromptRecord, error) {
	return nil, nil
}

func (m *memoryProviderStore) ReplaceMCPResources(context.Context, string, []domain.MCPResourceRecord) error {
	return nil
}

func (m *memoryProviderStore) ListMCPResources(context.Context, string, bool) ([]domain.MCPResourceRecord, error) {
	return nil, nil
}
