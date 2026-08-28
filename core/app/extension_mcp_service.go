package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"aivo/core/domain"
)

func (s *Service) ListMCPServers(ctx context.Context, input domain.MCPServerListInput) ([]domain.MCPServerListItem, error) {
	if s.mcpManager == nil {
		s.mcpManager = NewMCPManager(s.store, s.secrets)
	}
	return s.mcpManager.List(ctx, input)
}

func (s *Service) SaveMCPServer(ctx context.Context, input domain.SaveMCPServerInput) (domain.MCPServerConfig, error) {
	if s.mcpManager == nil {
		s.mcpManager = NewMCPManager(s.store, s.secrets)
	}
	return s.mcpManager.Save(ctx, input)
}

func (s *Service) SetMCPServerEnabled(ctx context.Context, input domain.SetMCPServerEnabledInput) (domain.MCPServerConfig, error) {
	if s.mcpManager == nil {
		s.mcpManager = NewMCPManager(s.store, s.secrets)
	}
	return s.mcpManager.SetEnabled(ctx, input)
}

func (s *Service) ProbeMCPServer(ctx context.Context, input domain.MCPProbeInput) (domain.MCPProbeResult, error) {
	if s.mcpManager == nil {
		s.mcpManager = NewMCPManager(s.store, s.secrets)
	}
	return s.mcpManager.Probe(ctx, input)
}

func (s *Service) GetMCPPrompt(ctx context.Context, input domain.MCPPromptGetInput) (domain.MCPPromptGetResult, error) {
	if s.mcpManager == nil {
		s.mcpManager = NewMCPManager(s.store, s.secrets)
	}
	return s.mcpManager.GetPrompt(ctx, input)
}

func (s *Service) ReadMCPResource(ctx context.Context, input domain.MCPResourceReadInput) (domain.MCPResourceReadResult, error) {
	if s.mcpManager == nil {
		s.mcpManager = NewMCPManager(s.store, s.secrets)
	}
	return s.mcpManager.ReadResource(ctx, input)
}

func (s *Service) InsertMCPPromptIntoSession(ctx context.Context, input domain.InsertMCPPromptIntoSessionInput) (domain.SessionEvent, error) {
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return domain.SessionEvent{}, errors.New("sessionId is required")
	}
	result, err := s.GetMCPPrompt(ctx, domain.MCPPromptGetInput{ServerID: input.ServerID, Name: input.Name, Arguments: input.Arguments})
	if err != nil {
		return domain.SessionEvent{}, err
	}
	content := strings.TrimSpace(result.Content)
	if content == "" {
		content = stringifyMCPStructured(result.Structured)
	}
	if content == "" {
		content = fmt.Sprintf("MCP prompt %s returned no text content.", result.Name)
	}
	event, err := s.AppendEvent(ctx, domain.AppendEventRequest{
		SessionID: sessionID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser, Visibility: domain.EventVisibilityNormal,
		Content: content,
		Payload: map[string]any{"kind": "mcp_prompt", "serverId": result.ServerID, "name": result.Name, "arguments": input.Arguments},
	})
	if err == nil && s.onSessionUpdated != nil {
		s.onSessionUpdated(sessionID, nil)
	}
	return event, err
}

func (s *Service) InsertMCPResourceIntoSession(ctx context.Context, input domain.InsertMCPResourceIntoSessionInput) (domain.SessionEvent, error) {
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return domain.SessionEvent{}, errors.New("sessionId is required")
	}
	result, err := s.ReadMCPResource(ctx, domain.MCPResourceReadInput{ServerID: input.ServerID, URI: input.URI})
	if err != nil {
		return domain.SessionEvent{}, err
	}
	content := strings.TrimSpace(result.Content)
	if content == "" {
		content = stringifyMCPStructured(result.Structured)
	}
	if content == "" {
		content = fmt.Sprintf("MCP resource %s returned no text content.", result.URI)
	}
	event, err := s.AppendEvent(ctx, domain.AppendEventRequest{
		SessionID: sessionID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser, Visibility: domain.EventVisibilityNormal,
		Content: content,
		Payload: map[string]any{"kind": "mcp_resource", "serverId": result.ServerID, "uri": result.URI},
	})
	if err == nil && s.onSessionUpdated != nil {
		s.onSessionUpdated(sessionID, nil)
	}
	return event, err
}

func stringifyMCPStructured(value map[string]any) string {
	if len(value) == 0 {
		return ""
	}
	parts := make([]string, 0, len(value))
	for key, item := range value {
		parts = append(parts, fmt.Sprintf("%s: %v", key, item))
	}
	return strings.Join(parts, "\n")
}

func (s *Service) ReadMCPServerLog(ctx context.Context, input domain.MCPServerLogInput) (domain.MCPServerLogResult, error) {
	if s.mcpManager == nil {
		s.mcpManager = NewMCPManager(s.store, s.secrets)
	}
	return s.mcpManager.ReadServerLog(ctx, input)
}

func (s *Service) DiscoverMCPOAuth(ctx context.Context, input domain.MCPOAuthDiscoveryInput) (domain.MCPOAuthDiscoveryResult, error) {
	if s.mcpManager == nil {
		s.mcpManager = NewMCPManager(s.store, s.secrets)
	}
	return s.mcpManager.DiscoverOAuth(ctx, input)
}

func (s *Service) StartMCPOAuth(ctx context.Context, input domain.MCPOAuthStartInput) (domain.MCPOAuthStartResult, error) {
	if s.mcpManager == nil {
		s.mcpManager = NewMCPManager(s.store, s.secrets)
	}
	return s.mcpManager.StartOAuth(ctx, input)
}

func (s *Service) GetMCPOAuthStatus(ctx context.Context, input domain.MCPOAuthStatusInput) (domain.MCPOAuthStatus, error) {
	if s.mcpManager == nil {
		s.mcpManager = NewMCPManager(s.store, s.secrets)
	}
	return s.mcpManager.OAuthStatus(ctx, input)
}

func (s *Service) ListToolCatalog(ctx context.Context, input domain.ToolCatalogInput) ([]domain.ToolCatalogEntry, error) {
	var registry *Registry
	workspaceRoot := strings.TrimSpace(input.WorkspaceRoot)
	if workspaceRoot == "" {
		registry = s.globalToolCatalogRegistry(ctx)
	} else {
		registry = s.workspaceToolCatalogRegistry(ctx, workspaceRoot)
	}
	if registry == nil {
		return nil, errors.New("tool registry is unavailable")
	}
	entries := registry.CatalogEntries()
	if input.Source == "" {
		return s.applyGlobalToolEligibility(ctx, entries)
	}
	filtered := make([]domain.ToolCatalogEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Source == input.Source {
			filtered = append(filtered, entry)
		}
	}
	return s.applyGlobalToolEligibility(ctx, filtered)
}

func (s *Service) globalToolCatalogRegistry(ctx context.Context) *Registry {
	registry, err := NewCodingToolRegistry("")
	if err != nil {
		registry = NewRegistry()
	}
	if err := registerDefaultHostControlTools(registry, s); err != nil {
		return nil
	}
	if s.extensionSupervisor != nil {
		_ = s.extensionSupervisor.RegisterAllReadyTools(registry)
	}
	if s.mcpManager != nil {
		s.mcpManager.RegisterCachedEnabledTools(ctx, registry)
	}
	return registry
}

func (s *Service) workspaceToolCatalogRegistry(ctx context.Context, workspaceRoot string) *Registry {
	registry, err := NewCodingToolRegistry(strings.TrimSpace(workspaceRoot))
	if err != nil {
		return nil
	}
	if err := registerDefaultHostControlTools(registry, s); err != nil {
		return nil
	}
	if s.extensionSupervisor != nil {
		_ = s.extensionSupervisor.RegisterAllReadyTools(registry)
	}
	if s.mcpManager != nil {
		s.mcpManager.RegisterCachedEnabledTools(ctx, registry)
	}
	runtime := NewToolRuntime(registry, strings.TrimSpace(workspaceRoot))
	runtime.ExtensionHooks = s.extensionSupervisor
	runtime.Permissions = NewPermissionEngine(s.store)
	runtime.Permissions.ProjectPreflight = s.prepareProjectPermission
	runtime.Permissions.MCPRegistrationPreflight = s.prepareMCPRegistrationPermission
	return registry
}

func (s *Service) DescribeTool(ctx context.Context, input domain.ToolDescribeInput) (domain.ToolCatalogEntry, error) {
	registry, _ := s.toolsForWorkspace(strings.TrimSpace(input.WorkspaceRoot))
	if registry == nil {
		return domain.ToolCatalogEntry{}, errors.New("tool registry is unavailable without a workspace root")
	}
	for _, entry := range registry.CatalogEntries() {
		if entry.Name == strings.TrimSpace(input.Name) {
			entries, eligibilityErr := s.applyGlobalToolEligibility(ctx, []domain.ToolCatalogEntry{entry})
			if eligibilityErr != nil {
				return domain.ToolCatalogEntry{}, eligibilityErr
			}
			return entries[0], nil
		}
	}
	return domain.ToolCatalogEntry{}, errors.New("tool not found")
}
