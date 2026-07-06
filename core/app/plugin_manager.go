package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

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
			_ = registry.RegisterScoped(&PluginRuntimeTool{pluginID: plugin.ID, spec: spec, manager: m}, domain.ToolSourcePlugin, plugin.ID, plugin.Manifest.Version)
		}
	}
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
			_ = registry.RegisterScoped(&PluginRuntimeTool{pluginID: plugin.ID, spec: spec, manager: m}, domain.ToolSourcePlugin, plugin.ID, plugin.Manifest.Version)
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

func (t *PluginRuntimeTool) Spec() domain.ToolSpec { return t.spec }

func (t *PluginRuntimeTool) Execute(ctx context.Context, args json.RawMessage, _ domain.ToolExecutionContext) domain.ToolResult {
	if t == nil || t.manager == nil {
		return toolFailure("", t.spec.Name, "plugin_unavailable", "plugin runtime is unavailable")
	}
	t.manager.mu.Lock()
	client := t.manager.clients[t.pluginID]
	t.manager.mu.Unlock()
	if client == nil {
		return toolFailure("", t.spec.Name, "plugin_unavailable", "plugin is not running")
	}
	var arguments any
	if len(args) > 0 {
		_ = json.Unmarshal(args, &arguments)
	}
	result, err := client.call(ctx, "tool.call", map[string]any{"name": t.spec.Name, "arguments": arguments})
	if err != nil {
		return toolFailure("", t.spec.Name, "plugin_tool_failed", err.Error())
	}
	return normalizeExternalToolResult(t.spec.Name, result)
}

type pluginProcessClient struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
	lines *bufio.Reader
	mu    sync.Mutex
}

func startPluginProcess(ctx context.Context, plugin domain.PluginInstall) (*pluginProcessClient, error) {
	entry := plugin.Manifest.Entrypoint
	if strings.TrimSpace(entry.Command) == "" {
		return nil, errors.New("plugin entrypoint.command is required")
	}
	root := plugin.RootPath
	command := entry.Command
	if !filepath.IsAbs(command) && strings.Contains(command, string(os.PathSeparator)) {
		command = filepath.Join(root, command)
	}
	cmd := exec.CommandContext(ctx, command, entry.Args...)
	cmd.Dir = firstNonEmptyApp(entry.CWD, root)
	if !filepath.IsAbs(cmd.Dir) {
		cmd.Dir = filepath.Join(root, cmd.Dir)
	}
	if !pathWithin(root, cmd.Dir) {
		return nil, errors.New("plugin entrypoint cwd escapes plugin root")
	}
	cmd.Env = SanitizedEnvironment(root, defaultEnvAllowlist(), entry.Env, nil)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = pluginStderrWriter(plugin.ID)
	setProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &pluginProcessClient{cmd: cmd, stdin: stdin, lines: bufio.NewReader(stdout)}, nil
}

func (c *pluginProcessClient) call(ctx context.Context, method string, params map[string]any) (map[string]any, error) {
	if c == nil {
		return nil, errors.New("plugin process is not running")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	id := uuid.NewString()
	request := map[string]any{"id": id, "method": method, "params": params}
	raw, _ := json.Marshal(request)
	if _, err := c.stdin.Write(append(raw, '\n')); err != nil {
		return nil, err
	}
	type response struct {
		ID     string         `json:"id"`
		Result map[string]any `json:"result"`
		Error  any            `json:"error"`
	}
	deadline := time.After(20 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline:
			return nil, errors.New("plugin request timed out")
		default:
			line, err := c.lines.ReadString('\n')
			if err != nil {
				return nil, err
			}
			var resp response
			if err := json.Unmarshal([]byte(line), &resp); err != nil {
				continue
			}
			if resp.ID != id {
				continue
			}
			if resp.Error != nil {
				return nil, fmt.Errorf("%v", resp.Error)
			}
			return resp.Result, nil
		}
	}
}

func (c *pluginProcessClient) close() error {
	if c == nil {
		return nil
	}
	_ = c.stdin.Close()
	if c.cmd != nil && c.cmd.Process != nil {
		_ = killProcessGroup(c.cmd.Process)
	}
	return nil
}

func LoadPluginManifest(path string) (string, string, domain.PluginManifest, error) {
	root := strings.TrimSpace(path)
	if root == "" {
		return "", "", domain.PluginManifest{}, errors.New("plugin path is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", domain.PluginManifest{}, err
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return "", "", domain.PluginManifest{}, err
	}
	manifestPath := absRoot
	if info.IsDir() {
		candidates := []string{filepath.Join(absRoot, ".aivo-plugin", "plugin.json"), filepath.Join(absRoot, "aivo.plugin.json")}
		manifestPath = ""
		for _, candidate := range candidates {
			if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
				manifestPath = candidate
				break
			}
		}
		if manifestPath == "" {
			return "", "", domain.PluginManifest{}, errors.New("plugin manifest not found")
		}
	} else {
		absRoot = filepath.Dir(absRoot)
	}
	if !pathWithin(absRoot, manifestPath) {
		return "", "", domain.PluginManifest{}, errors.New("plugin manifest escapes plugin root")
	}
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", "", domain.PluginManifest{}, err
	}
	var manifest domain.PluginManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return "", "", domain.PluginManifest{}, err
	}
	if manifest.ID == "" {
		manifest.ID = firstNonEmptyApp(manifest.Name, filepath.Base(absRoot))
	}
	if manifest.Name == "" {
		manifest.Name = manifest.ID
	}
	if err := validatePluginManifestPaths(absRoot, manifest); err != nil {
		return "", "", domain.PluginManifest{}, err
	}
	return absRoot, manifestPath, manifest, nil
}

func validatePluginManifestPaths(root string, manifest domain.PluginManifest) error {
	if manifest.Entrypoint.Command != "" && strings.Contains(manifest.Entrypoint.Command, string(os.PathSeparator)) && !filepath.IsAbs(manifest.Entrypoint.Command) {
		if !pathWithin(root, filepath.Join(root, manifest.Entrypoint.Command)) {
			return errors.New("entrypoint command escapes plugin root")
		}
	}
	if manifest.Entrypoint.CWD != "" && !pathWithin(root, filepath.Join(root, manifest.Entrypoint.CWD)) {
		return errors.New("entrypoint cwd escapes plugin root")
	}
	return nil
}

func parsePluginToolSpecs(plugin domain.PluginInstall, initResult map[string]any) []domain.ToolSpec {
	specs := []domain.ToolSpec{}
	if rawTools, ok := initResult["tools"].([]any); ok {
		for _, rawTool := range rawTools {
			bytes, _ := json.Marshal(rawTool)
			var spec domain.ToolSpec
			if err := json.Unmarshal(bytes, &spec); err == nil && strings.TrimSpace(spec.Name) != "" {
				specs = append(specs, normalizePluginToolSpec(plugin, spec))
			}
		}
	}
	for _, tool := range plugin.Manifest.Tools {
		specs = append(specs, normalizePluginToolSpec(plugin, domain.ToolSpec{Name: tool.Name, Description: tool.Description, InputSchema: tool.InputSchema, Capability: tool.Capability, RiskLevel: tool.RiskLevel, Toolsets: tool.Toolsets}))
	}
	return dedupeToolSpecs(specs)
}

func parsePluginHooks(plugin domain.PluginInstall, initResult map[string]any) []string {
	hooks := append([]string(nil), plugin.Manifest.Hooks...)
	if rawHooks, ok := initResult["hooks"].([]any); ok {
		for _, rawHook := range rawHooks {
			if hook, ok := rawHook.(string); ok {
				hooks = append(hooks, hook)
			}
		}
	}
	return appendUniqueStrings(nil, hooks...)
}

func normalizePluginToolSpec(plugin domain.PluginInstall, spec domain.ToolSpec) domain.ToolSpec {
	spec.Name = strings.TrimSpace(spec.Name)
	if spec.InputSchema == nil {
		spec.InputSchema = map[string]any{"type": "object", "additionalProperties": true}
	}
	if spec.Category == "" {
		spec.Category = "plugin"
	}
	if spec.Capability == "" {
		spec.Capability = "plugin.read"
	}
	if spec.RiskLevel == "" {
		spec.RiskLevel = firstNonEmptyApp(plugin.Manifest.Permissions.RiskLevel, "medium")
	}
	if len(spec.Toolsets) == 0 {
		spec.Toolsets = []string{"plugin", "coding"}
	}
	return spec
}

func dedupeToolSpecs(specs []domain.ToolSpec) []domain.ToolSpec {
	seen := map[string]bool{}
	out := []domain.ToolSpec{}
	for _, spec := range specs {
		if spec.Name == "" || seen[spec.Name] {
			continue
		}
		seen[spec.Name] = true
		out = append(out, spec)
	}
	return out
}

func normalizeExternalToolResult(name string, result map[string]any) domain.ToolResult {
	if result == nil {
		return domain.ToolResult{Name: name, OK: true}
	}
	ok := true
	if rawOK, exists := result["ok"].(bool); exists {
		ok = rawOK
	}
	content, _ := result["content"].(string)
	if content == "" {
		content, _ = result["output"].(string)
	}
	if content == "" {
		raw, _ := json.MarshalIndent(result, "", "  ")
		content = string(raw)
	}
	errorText, _ := result["error"].(string)
	return domain.ToolResult{Name: name, OK: ok, Content: content, ModelContent: content, Structured: result, Error: errorText}
}

func pathWithin(root string, target string) bool {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absTarget)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}

func pluginStderrWriter(pluginID string) io.Writer {
	home, err := os.UserHomeDir()
	if err != nil {
		return io.Discard
	}
	dir := filepath.Join(home, ".aivo", "logs")
	_ = os.MkdirAll(dir, 0o755)
	file, err := os.OpenFile(filepath.Join(dir, "plugins-"+pluginID+".log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return io.Discard
	}
	return file
}

func firstNonEmptyApp(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func pluginContainsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
