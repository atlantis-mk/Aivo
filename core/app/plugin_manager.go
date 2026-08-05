package app

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/google/uuid"

	"aivo/core/domain"
)

type pluginStore interface {
	SavePluginInstall(context.Context, domain.PluginInstall) (domain.PluginInstall, error)
	GetPluginInstall(context.Context, string) (domain.PluginInstall, error)
	ListPluginInstalls(context.Context, bool) ([]domain.PluginInstall, error)
	SetPluginEnabled(context.Context, string, bool, string, string) (domain.PluginInstall, error)
	SavePluginDiagnostic(context.Context, domain.PluginDiagnostic) (domain.PluginDiagnostic, error)
	ListPluginDiagnostics(context.Context, string, string, int) ([]domain.PluginDiagnostic, error)
}

type PluginManager struct {
	store   pluginStore
	mu      sync.Mutex
	clients map[string]*pluginProcessClient
	tools   map[string][]domain.ToolSpec
	hooks   map[string][]string
}

func NewPluginManager(store any) *PluginManager {
	ps, _ := store.(pluginStore)
	return &PluginManager{store: ps, clients: map[string]*pluginProcessClient{}, tools: map[string][]domain.ToolSpec{}, hooks: map[string][]string{}}
}

func (m *PluginManager) InstallFromPath(ctx context.Context, path string, enable bool) (domain.PluginInstall, error) {
	if m == nil || m.store == nil {
		return domain.PluginInstall{}, errors.New("plugin store is not configured")
	}
	root, manifestPath, manifest, err := LoadPluginManifest(path)
	if err != nil {
		return domain.PluginInstall{}, err
	}
	plugin := domain.PluginInstall{
		ID: firstNonEmptyApp(manifest.ID, manifest.Name), Manifest: manifest, RootPath: root, ManifestPath: manifestPath,
		Enabled: enable, Status: domain.PluginStatusDisabled,
	}
	if plugin.ID == "" {
		plugin.ID = uuid.NewString()
	}
	if enable {
		plugin.Status = domain.PluginStatusEnabled
	}
	saved, err := m.store.SavePluginInstall(ctx, plugin)
	if err != nil {
		return domain.PluginInstall{}, err
	}
	if enable {
		_ = m.ensurePluginStarted(ctx, saved)
	}
	return saved, nil
}

func (m *PluginManager) List(ctx context.Context, input domain.PluginListInput) ([]domain.PluginListItem, error) {
	if m == nil || m.store == nil {
		return nil, errors.New("plugin store is not configured")
	}
	plugins, err := m.store.ListPluginInstalls(ctx, input.IncludeDisabled)
	if err != nil {
		return nil, err
	}
	out := make([]domain.PluginListItem, 0, len(plugins))
	for _, plugin := range plugins {
		item := domain.PluginListItem{Plugin: plugin}
		if input.IncludeDiagnostics {
			item.Diagnostics, _ = m.store.ListPluginDiagnostics(ctx, plugin.ID, "", 20)
		}
		for _, spec := range m.tools[plugin.ID] {
			item.Tools = append(item.Tools, domain.ToolCatalogEntry{Name: spec.Name, Description: spec.Description, InputSchema: spec.InputSchema, Capability: spec.Capability, RiskLevel: spec.RiskLevel, Category: spec.Category, Toolsets: spec.Toolsets, Source: domain.ToolSourcePlugin, SourceID: plugin.ID, Enabled: plugin.Enabled})
		}
		out = append(out, item)
	}
	return out, nil
}

func (m *PluginManager) SetEnabled(ctx context.Context, input domain.SetPluginEnabledInput) (domain.PluginInstall, error) {
	if m == nil || m.store == nil {
		return domain.PluginInstall{}, errors.New("plugin store is not configured")
	}
	status := domain.PluginStatusDisabled
	if input.Enabled {
		status = domain.PluginStatusEnabled
	}
	plugin, err := m.store.SetPluginEnabled(ctx, input.PluginID, input.Enabled, status, "")
	if err != nil {
		return domain.PluginInstall{}, err
	}
	if input.Enabled {
		err = m.ensurePluginStarted(ctx, plugin)
		if err != nil {
			plugin, _ = m.store.SetPluginEnabled(ctx, plugin.ID, true, domain.PluginStatusError, err.Error())
			return plugin, err
		}
	} else {
		m.stopPlugin(plugin.ID)
	}
	return plugin, nil
}

func (m *PluginManager) RegisterEnabledTools(ctx context.Context, registry *Registry) {
	if m == nil || m.store == nil || registry == nil {
		return
	}
	plugins, err := m.store.ListPluginInstalls(ctx, false)
	if err != nil {
		return
	}
	for _, plugin := range plugins {
		if !plugin.Enabled {
			continue
		}
		if err := m.ensurePluginStarted(ctx, plugin); err != nil {
			m.diagnostic(ctx, plugin.ID, "", domain.PluginDiagnosticError, err.Error(), nil)
			continue
		}
		for _, spec := range m.tools[plugin.ID] {
			if registerErr := registry.RegisterScoped(&PluginRuntimeTool{pluginID: plugin.ID, spec: spec, manager: m}, domain.ToolSourcePlugin, plugin.ID, plugin.Manifest.Version); registerErr != nil {
				m.diagnostic(ctx, plugin.ID, "", domain.PluginDiagnosticError, registerErr.Error(), map[string]any{"tool": spec.Name})
			}
		}
	}
}

func (m *PluginManager) PrepareEnabled(ctx context.Context) map[string]bool {
	failed := map[string]bool{}
	if m == nil || m.store == nil {
		return failed
	}
	plugins, err := m.store.ListPluginInstalls(ctx, false)
	if err != nil {
		return failed
	}
	for _, plugin := range plugins {
		if !plugin.Enabled {
			continue
		}
		if err := m.ensurePluginStarted(ctx, plugin); err != nil {
			failed[toolSourceEligibilityKey(domain.ToolSourcePlugin, plugin.ID)] = true
			m.diagnostic(ctx, plugin.ID, "", domain.PluginDiagnosticError, err.Error(), nil)
		}
	}
	return failed
}

func (m *PluginManager) RegisterCachedEnabledTools(ctx context.Context, registry *Registry) {
	if m == nil || m.store == nil || registry == nil {
		return
	}
	plugins, err := m.store.ListPluginInstalls(ctx, false)
	if err != nil {
		return
	}
	for _, plugin := range plugins {
		if !plugin.Enabled {
			continue
		}
		specs := append([]domain.ToolSpec(nil), m.tools[plugin.ID]...)
		for _, tool := range plugin.Manifest.Tools {
			specs = append(specs, normalizePluginToolSpec(plugin, domain.ToolSpec{
				Name: tool.Name, Description: tool.Description, InputSchema: tool.InputSchema,
				Capability: tool.Capability, RiskLevel: tool.RiskLevel, Toolsets: tool.Toolsets,
			}))
		}
		for _, spec := range dedupeToolSpecs(specs) {
			if registerErr := registry.RegisterScoped(&PluginRuntimeTool{pluginID: plugin.ID, spec: spec, manager: m}, domain.ToolSourcePlugin, plugin.ID, plugin.Manifest.Version); registerErr != nil {
				m.diagnostic(ctx, plugin.ID, "", domain.PluginDiagnosticError, registerErr.Error(), map[string]any{"tool": spec.Name})
			}
		}
	}
}

func (m *PluginManager) InvokeHook(ctx context.Context, hook string, payload map[string]any) []map[string]any {
	if m == nil || hook == "" {
		return nil
	}
	m.mu.Lock()
	clients := make(map[string]*pluginProcessClient, len(m.clients))
	for id, client := range m.clients {
		if pluginContainsString(m.hooks[id], hook) {
			clients[id] = client
		}
	}
	m.mu.Unlock()
	results := []map[string]any{}
	for pluginID, client := range clients {
		result, err := client.call(ctx, "hook.invoke", map[string]any{"hook": hook, "payload": payload})
		if err != nil {
			m.diagnostic(ctx, pluginID, "", domain.PluginDiagnosticWarn, "hook "+hook+" failed: "+err.Error(), nil)
			continue
		}
		if result != nil {
			results = append(results, result)
		}
	}
	return results
}

func (m *PluginManager) ensurePluginStarted(ctx context.Context, plugin domain.PluginInstall) error {
	m.mu.Lock()
	if _, ok := m.clients[plugin.ID]; ok {
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()
	client, err := startPluginProcess(ctx, plugin)
	if err != nil {
		return err
	}
	initResult, err := client.call(ctx, "initialize", map[string]any{"pluginId": plugin.ID, "manifest": plugin.Manifest})
	if err != nil {
		_ = client.close()
		return err
	}
	tools := parsePluginToolSpecs(plugin, initResult)
	hooks := parsePluginHooks(plugin, initResult)
	m.mu.Lock()
	if old := m.clients[plugin.ID]; old != nil {
		_ = old.close()
	}
	m.clients[plugin.ID] = client
	m.tools[plugin.ID] = tools
	m.hooks[plugin.ID] = hooks
	m.mu.Unlock()
	return nil
}

func (m *PluginManager) stopPlugin(pluginID string) {
	m.mu.Lock()
	client := m.clients[pluginID]
	delete(m.clients, pluginID)
	delete(m.tools, pluginID)
	delete(m.hooks, pluginID)
	m.mu.Unlock()
	if client != nil {
		_ = client.close()
	}
}

func (m *PluginManager) diagnostic(ctx context.Context, pluginID string, serverID string, level string, message string, metadata map[string]any) {
	if m == nil || m.store == nil || strings.TrimSpace(message) == "" {
		return
	}
	_, _ = m.store.SavePluginDiagnostic(ctx, domain.PluginDiagnostic{PluginID: pluginID, ServerID: serverID, Level: level, Message: message, Metadata: metadata})
}

type PluginRuntimeTool struct {
	pluginID string
	spec     domain.ToolSpec
	manager  *PluginManager
}

type pluginProcessClient struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
	lines *bufio.Reader
	mu    sync.Mutex
}
