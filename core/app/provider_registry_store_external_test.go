package app

import (
	"context"
	"errors"
	"strings"

	"aivo/core/domain"
)

func (m *memoryProviderStore) SaveMCPDiagnostic(_ context.Context, diagnostic domain.MCPDiagnostic) (domain.MCPDiagnostic, error) {
	m.mcpDiagnostics = append(m.mcpDiagnostics, diagnostic)
	return diagnostic, nil
}

func (m *memoryProviderStore) ListMCPDiagnostics(_ context.Context, serverID string, limit int) ([]domain.MCPDiagnostic, error) {
	out := []domain.MCPDiagnostic{}
	for _, diagnostic := range m.mcpDiagnostics {
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
