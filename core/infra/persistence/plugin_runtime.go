package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aivo/core/domain"
)

func (s *Store) SavePluginInstall(ctx context.Context, plugin domain.PluginInstall) (domain.PluginInstall, error) {
	now := domain.NowString(time.Now())
	if strings.TrimSpace(plugin.ID) == "" {
		plugin.ID = firstNonEmpty(strings.TrimSpace(plugin.Manifest.ID), strings.TrimSpace(plugin.Manifest.Name), uuid.NewString())
	}
	if plugin.TimeCreated == "" {
		plugin.TimeCreated = now
	}
	plugin.TimeUpdated = now
	if plugin.Status == "" {
		if plugin.Enabled {
			plugin.Status = domain.PluginStatusEnabled
		} else {
			plugin.Status = domain.PluginStatusDisabled
		}
	}
	raw, _ := json.Marshal(plugin.Manifest)
	row := pluginInstallRow{
		ID: plugin.ID, ManifestName: plugin.Manifest.Name, Version: plugin.Manifest.Version,
		DisplayName: plugin.Manifest.DisplayName, Description: plugin.Manifest.Description,
		RootPath: normalizeStoredPath(plugin.RootPath), ManifestPath: normalizeStoredPath(plugin.ManifestPath),
		Manifest: string(raw), Enabled: boolInt(plugin.Enabled), Status: plugin.Status, Error: plugin.Error,
		TimeCreated: plugin.TimeCreated, TimeUpdated: plugin.TimeUpdated,
	}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"manifest_name", "version", "display_name", "description", "root_path", "manifest_path", "manifest", "enabled", "status", "error", "time_updated"}),
	}).Create(&row).Error
	if err != nil {
		return domain.PluginInstall{}, err
	}
	return pluginInstallFromRow(row), nil
}

func (s *Store) GetPluginInstall(ctx context.Context, id string) (domain.PluginInstall, error) {
	var row pluginInstallRow
	if err := s.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).First(&row).Error; err != nil {
		return domain.PluginInstall{}, err
	}
	return pluginInstallFromRow(row), nil
}

func (s *Store) ListPluginInstalls(ctx context.Context, includeDisabled bool) ([]domain.PluginInstall, error) {
	q := s.db.WithContext(ctx).Model(&pluginInstallRow{})
	if !includeDisabled {
		q = q.Where("enabled = 1")
	}
	var rows []pluginInstallRow
	if err := q.Order("time_updated DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.PluginInstall, 0, len(rows))
	for _, row := range rows {
		out = append(out, pluginInstallFromRow(row))
	}
	return out, nil
}

func (s *Store) SetPluginEnabled(ctx context.Context, id string, enabled bool, status string, errText string) (domain.PluginInstall, error) {
	now := domain.NowString(time.Now())
	if status == "" {
		if enabled {
			status = domain.PluginStatusEnabled
		} else {
			status = domain.PluginStatusDisabled
		}
	}
	if err := s.db.WithContext(ctx).Model(&pluginInstallRow{}).Where("id = ?", strings.TrimSpace(id)).Updates(map[string]any{
		"enabled": boolInt(enabled), "status": status, "error": strings.TrimSpace(errText), "time_updated": now,
	}).Error; err != nil {
		return domain.PluginInstall{}, err
	}
	return s.GetPluginInstall(ctx, id)
}

func (s *Store) SavePluginDiagnostic(ctx context.Context, diagnostic domain.PluginDiagnostic) (domain.PluginDiagnostic, error) {
	if diagnostic.ID == "" {
		diagnostic.ID = uuid.NewString()
	}
	if diagnostic.Level == "" {
		diagnostic.Level = domain.PluginDiagnosticInfo
	}
	if diagnostic.TimeCreated == "" {
		diagnostic.TimeCreated = domain.NowString(time.Now())
	}
	row := pluginDiagnosticRow{
		ID: diagnostic.ID, PluginID: diagnostic.PluginID, ServerID: diagnostic.ServerID,
		Level: diagnostic.Level, Message: diagnostic.Message, Metadata: encodeAnyMap(diagnostic.Metadata),
		TimeCreated: diagnostic.TimeCreated,
	}
	return diagnostic, s.db.WithContext(ctx).Create(&row).Error
}

func (s *Store) ListPluginDiagnostics(ctx context.Context, pluginID string, serverID string, limit int) ([]domain.PluginDiagnostic, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := s.db.WithContext(ctx).Model(&pluginDiagnosticRow{})
	if strings.TrimSpace(pluginID) != "" {
		q = q.Where("plugin_id = ?", strings.TrimSpace(pluginID))
	}
	if strings.TrimSpace(serverID) != "" {
		q = q.Where("server_id = ?", strings.TrimSpace(serverID))
	}
	var rows []pluginDiagnosticRow
	if err := q.Order("time_created DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.PluginDiagnostic, 0, len(rows))
	for _, row := range rows {
		out = append(out, pluginDiagnosticFromRow(row))
	}
	return out, nil
}

func (s *Store) SaveMCPServer(ctx context.Context, server domain.MCPServerConfig) (domain.MCPServerConfig, error) {
	now := domain.NowString(time.Now())
	if strings.TrimSpace(server.ID) == "" {
		server.ID = firstNonEmpty(strings.TrimSpace(server.Name), uuid.NewString())
	}
	if server.Name == "" {
		server.Name = server.ID
	}
	if server.Transport == "" {
		server.Transport = domain.MCPTransportStdio
	}
	server.AuthType = normalizeMCPAuthType(server.AuthType)
	if server.TimeCreated == "" {
		server.TimeCreated = now
	}
	server.TimeUpdated = now
	if server.Status == "" {
		if server.Enabled {
			server.Status = domain.MCPServerStatusEnabled
		} else {
			server.Status = domain.MCPServerStatusDisabled
		}
	}
	row := mcpServerRow{
		ID: server.ID, Name: server.Name, DisplayName: server.DisplayName, Description: server.Description,
		Transport: server.Transport, Command: server.Command, Args: encodeStrings(server.Args), CWD: server.CWD,
		Env: encodeStringMap(redactSecretStringMap(server.Env)), URL: server.URL,
		Headers: encodeStringMap(redactSecretStringMap(server.Headers)), AuthType: server.AuthType,
		BearerTokenEnv: server.BearerTokenEnv, OAuthIssuerURL: server.OAuthIssuerURL, OAuthClientID: server.OAuthClientID,
		OAuthScopes: encodeStrings(server.OAuthScopes), OAuthAccessTokenRef: server.OAuthAccessTokenRef,
		OAuthRefreshTokenRef: server.OAuthRefreshTokenRef, OAuthExpiresAt: server.OAuthExpiresAt,
		Roots: encodeStrings(server.Roots), TimeoutSeconds: server.TimeoutSeconds,
		ConnectTimeoutSeconds: server.ConnectTimeoutSeconds, Enabled: boolInt(server.Enabled), PluginID: server.PluginID,
		Status: server.Status, Error: server.Error, TimeCreated: server.TimeCreated, TimeUpdated: server.TimeUpdated,
	}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "display_name", "description", "transport", "command", "args", "cwd", "env", "url", "headers", "auth_type", "bearer_token_env", "oauth_issuer_url", "oauth_client_id", "oauth_scopes", "oauth_access_token_ref", "oauth_refresh_token_ref", "oauth_expires_at", "roots", "timeout_seconds", "connect_timeout_seconds", "enabled", "plugin_id", "status", "error", "time_updated"}),
	}).Create(&row).Error
	if err != nil {
		return domain.MCPServerConfig{}, err
	}
	return mcpServerFromRow(row), nil
}

func (s *Store) GetMCPServer(ctx context.Context, id string) (domain.MCPServerConfig, error) {
	var row mcpServerRow
	if err := s.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).First(&row).Error; err != nil {
		return domain.MCPServerConfig{}, err
	}
	return mcpServerFromRow(row), nil
}

func (s *Store) ListMCPServers(ctx context.Context, includeDisabled bool) ([]domain.MCPServerConfig, error) {
	q := s.db.WithContext(ctx).Model(&mcpServerRow{})
	if !includeDisabled {
		q = q.Where("enabled = 1")
	}
	var rows []mcpServerRow
	if err := q.Order("time_updated DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.MCPServerConfig, 0, len(rows))
	for _, row := range rows {
		out = append(out, mcpServerFromRow(row))
	}
	return out, nil
}

func (s *Store) SetMCPServerEnabled(ctx context.Context, id string, enabled bool, status string, errText string) (domain.MCPServerConfig, error) {
	now := domain.NowString(time.Now())
	if status == "" {
		if enabled {
			status = domain.MCPServerStatusEnabled
		} else {
			status = domain.MCPServerStatusDisabled
		}
	}
	if err := s.db.WithContext(ctx).Model(&mcpServerRow{}).Where("id = ?", strings.TrimSpace(id)).Updates(map[string]any{
		"enabled": boolInt(enabled), "status": status, "error": strings.TrimSpace(errText), "time_updated": now,
	}).Error; err != nil {
		return domain.MCPServerConfig{}, err
	}
	return s.GetMCPServer(ctx, id)
}

func (s *Store) ReplaceMCPTools(ctx context.Context, serverID string, tools []domain.MCPToolRecord) error {
	serverID = strings.TrimSpace(serverID)
	now := domain.NowString(time.Now())
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("server_id = ?", serverID).Delete(&mcpToolRow{}).Error; err != nil {
			return err
		}
		for _, tool := range tools {
			if tool.ID == "" {
				tool.ID = serverID + ":" + tool.Name
			}
			tool.ServerID = serverID
			tool.TimeUpdated = now
			row := mcpToolRow{ID: tool.ID, ServerID: tool.ServerID, Name: tool.Name, Description: tool.Description, InputSchema: encodeAnyMap(tool.InputSchema), Capability: tool.Capability, RiskLevel: tool.RiskLevel, TimeUpdated: tool.TimeUpdated}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) ListMCPTools(ctx context.Context, serverID string) ([]domain.MCPToolRecord, error) {
	q := s.db.WithContext(ctx).Model(&mcpToolRow{})
	if strings.TrimSpace(serverID) != "" {
		q = q.Where("server_id = ?", strings.TrimSpace(serverID))
	}
	var rows []mcpToolRow
	if err := q.Order("name ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.MCPToolRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, mcpToolFromRow(row))
	}
	return out, nil
}

func (s *Store) ReplaceMCPPrompts(ctx context.Context, serverID string, prompts []domain.MCPPromptRecord) error {
	serverID = strings.TrimSpace(serverID)
	now := domain.NowString(time.Now())
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("server_id = ?", serverID).Delete(&mcpPromptRow{}).Error; err != nil {
			return err
		}
		for _, prompt := range prompts {
			if prompt.ID == "" {
				prompt.ID = serverID + ":" + prompt.Name
			}
			prompt.ServerID = serverID
			prompt.TimeUpdated = now
			rawArgs, _ := json.Marshal(prompt.Arguments)
			row := mcpPromptRow{ID: prompt.ID, ServerID: prompt.ServerID, Name: prompt.Name, Description: prompt.Description, Arguments: string(rawArgs), TimeUpdated: prompt.TimeUpdated}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) ListMCPPrompts(ctx context.Context, serverID string) ([]domain.MCPPromptRecord, error) {
	q := s.db.WithContext(ctx).Model(&mcpPromptRow{})
	if strings.TrimSpace(serverID) != "" {
		q = q.Where("server_id = ?", strings.TrimSpace(serverID))
	}
	var rows []mcpPromptRow
	if err := q.Order("name ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.MCPPromptRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, mcpPromptFromRow(row))
	}
	return out, nil
}

func (s *Store) ReplaceMCPResources(ctx context.Context, serverID string, resources []domain.MCPResourceRecord) error {
	serverID = strings.TrimSpace(serverID)
	now := domain.NowString(time.Now())
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("server_id = ?", serverID).Delete(&mcpResourceRow{}).Error; err != nil {
			return err
		}
		for _, resource := range resources {
			if resource.ID == "" {
				resource.ID = serverID + ":" + firstNonEmpty(resource.URI, resource.URITemplate, resource.Name)
			}
			resource.ServerID = serverID
			resource.TimeUpdated = now
			row := mcpResourceRow{ID: resource.ID, ServerID: resource.ServerID, URI: resource.URI, URITemplate: resource.URITemplate, Name: resource.Name, Description: resource.Description, MimeType: resource.MimeType, Template: boolInt(resource.Template), TimeUpdated: resource.TimeUpdated}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) ListMCPResources(ctx context.Context, serverID string, templates bool) ([]domain.MCPResourceRecord, error) {
	q := s.db.WithContext(ctx).Model(&mcpResourceRow{}).Where("template = ?", boolInt(templates))
	if strings.TrimSpace(serverID) != "" {
		q = q.Where("server_id = ?", strings.TrimSpace(serverID))
	}
	var rows []mcpResourceRow
	if err := q.Order("name ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.MCPResourceRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, mcpResourceFromRow(row))
	}
	return out, nil
}

func pluginInstallFromRow(row pluginInstallRow) domain.PluginInstall {
	var manifest domain.PluginManifest
	_ = json.Unmarshal([]byte(row.Manifest), &manifest)
	if manifest.ID == "" {
		manifest.ID = row.ID
	}
	return domain.PluginInstall{ID: row.ID, Manifest: manifest, RootPath: row.RootPath, ManifestPath: row.ManifestPath, Enabled: row.Enabled != 0, Status: row.Status, Error: row.Error, TimeCreated: row.TimeCreated, TimeUpdated: row.TimeUpdated}
}

func pluginDiagnosticFromRow(row pluginDiagnosticRow) domain.PluginDiagnostic {
	return domain.PluginDiagnostic{ID: row.ID, PluginID: row.PluginID, ServerID: row.ServerID, Level: row.Level, Message: row.Message, Metadata: decodeAnyMap(row.Metadata), TimeCreated: row.TimeCreated}
}

func mcpServerFromRow(row mcpServerRow) domain.MCPServerConfig {
	return domain.MCPServerConfig{ID: row.ID, Name: row.Name, DisplayName: row.DisplayName, Description: row.Description, Transport: row.Transport, Command: row.Command, Args: decodeStrings(row.Args), CWD: row.CWD, Env: decodeStringMap(row.Env), URL: row.URL, Headers: decodeStringMap(row.Headers), AuthType: normalizeMCPAuthType(row.AuthType), BearerTokenEnv: row.BearerTokenEnv, OAuthIssuerURL: row.OAuthIssuerURL, OAuthClientID: row.OAuthClientID, OAuthScopes: decodeStrings(row.OAuthScopes), OAuthAccessTokenRef: row.OAuthAccessTokenRef, OAuthRefreshTokenRef: row.OAuthRefreshTokenRef, OAuthExpiresAt: row.OAuthExpiresAt, Roots: decodeStrings(row.Roots), TimeoutSeconds: row.TimeoutSeconds, ConnectTimeoutSeconds: row.ConnectTimeoutSeconds, Enabled: row.Enabled != 0, PluginID: row.PluginID, Status: row.Status, Error: row.Error, TimeCreated: row.TimeCreated, TimeUpdated: row.TimeUpdated}
}

func normalizeMCPAuthType(value string) string {
	switch strings.TrimSpace(value) {
	case domain.MCPAuthBearer:
		return domain.MCPAuthBearer
	case domain.MCPAuthOAuth:
		return domain.MCPAuthOAuth
	default:
		return domain.MCPAuthNone
	}
}

func mcpToolFromRow(row mcpToolRow) domain.MCPToolRecord {
	return domain.MCPToolRecord{ID: row.ID, ServerID: row.ServerID, Name: row.Name, Description: row.Description, InputSchema: decodeAnyMap(row.InputSchema), Capability: row.Capability, RiskLevel: row.RiskLevel, TimeUpdated: row.TimeUpdated}
}

func mcpPromptFromRow(row mcpPromptRow) domain.MCPPromptRecord {
	var args []domain.MCPPromptArgument
	_ = json.Unmarshal([]byte(row.Arguments), &args)
	return domain.MCPPromptRecord{ID: row.ID, ServerID: row.ServerID, Name: row.Name, Description: row.Description, Arguments: args, TimeUpdated: row.TimeUpdated}
}

func mcpResourceFromRow(row mcpResourceRow) domain.MCPResourceRecord {
	return domain.MCPResourceRecord{ID: row.ID, ServerID: row.ServerID, URI: row.URI, URITemplate: row.URITemplate, Name: row.Name, Description: row.Description, MimeType: row.MimeType, Template: row.Template != 0, TimeUpdated: row.TimeUpdated}
}

func redactSecretStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		if looksSecretKey(key) {
			out[key] = "<redacted>"
		} else {
			out[key] = value
		}
	}
	return out
}

func looksSecretKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	for _, marker := range []string{"key", "token", "secret", "password", "authorization", "cookie"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

var errPluginNotFound = errors.New("plugin not found")

func (s *Store) DeletePluginInstall(ctx context.Context, id string) error {
	err := s.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).Delete(&pluginInstallRow{}).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errPluginNotFound
	}
	return err
}
