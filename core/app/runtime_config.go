package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"aivo/core/domain"
	"github.com/tailscale/hujson"
	"gopkg.in/yaml.v3"
)

const runtimeConfigMaxBytes = 1 << 20

func (s *Service) EffectiveRuntimeConfig(_ context.Context, projectPath string) (domain.EffectiveRuntimeConfig, error) {
	return loadEffectiveRuntimeConfig(projectPath), nil
}

func loadEffectiveRuntimeConfig(projectPath string) domain.EffectiveRuntimeConfig {
	result := domain.EffectiveRuntimeConfig{ProjectPath: strings.TrimSpace(projectPath), Config: defaultRuntimeConfig()}
	for _, layer := range runtimeConfigLayers(projectPath) {
		for _, candidate := range layer.files {
			cfg, exists, err := readRuntimeConfig(candidate)
			if !exists {
				continue
			}
			if err != nil {
				result.Diagnostics = append(result.Diagnostics, domain.RuntimeConfigDiagnostic{Path: candidate, Level: "error", Message: err.Error()})
				continue
			}
			if diagnostics := validateRuntimeConfig(candidate, cfg); len(diagnostics) > 0 {
				result.Diagnostics = append(result.Diagnostics, diagnostics...)
				continue
			}
			mergeRuntimeConfig(&result.Config, cfg)
			result.Sources = append(result.Sources, domain.RuntimeConfigSource{Path: candidate, Scope: layer.scope})
		}
		for _, dir := range layer.markdownDirs {
			entries, diagnostics := loadRuntimeMarkdownEntries(dir)
			result.Diagnostics = append(result.Diagnostics, diagnostics...)
			for _, entry := range entries {
				mergeRuntimeConfig(&result.Config, entry.Config)
				result.Sources = append(result.Sources, domain.RuntimeConfigSource{Path: entry.Path, Scope: layer.scope})
			}
		}
	}
	if defaultAgent := strings.TrimSpace(result.Config.DefaultAgent); defaultAgent != "" {
		definition, err := NewAgentCatalogWithRuntime(result.Config).Get(defaultAgent)
		if err != nil || definition.Mode == "subagent" || definition.Hidden {
			result.Diagnostics = append(result.Diagnostics, domain.RuntimeConfigDiagnostic{
				Path: "<effective>", Level: "error", Message: "defaultAgent must reference a visible enabled primary/all agent",
			})
		}
	}
	return result
}

func defaultRuntimeConfig() domain.RuntimeConfig {
	auto := true
	return domain.RuntimeConfig{
		Commands: map[string]domain.CommandTemplateDefinition{}, Agents: map[string]domain.AgentRuntimeDefinition{},
		LanguageServers: map[string]domain.LanguageServerDefinition{}, ProviderExtensions: map[string]domain.ProviderExtensionDefinition{},
		Toolsets: map[string][]string{}, Permissions: map[string]string{},
		Compaction:          domain.CompactionRuntimeConfig{Auto: &auto, ThresholdPercent: 80, ReserveTokens: 4096},
		MaxParallelChildren: 4,
	}
}

type runtimeConfigLayer struct {
	scope        string
	files        []string
	markdownDirs []string
}

func runtimeConfigLayers(projectPath string) []runtimeConfigLayer {
	var layers []runtimeConfigLayer
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		dirs := []string{filepath.Join(home, ".config", "aivo"), filepath.Join(home, ".aivo")}
		layer := runtimeConfigLayer{scope: "global", markdownDirs: dirs}
		for _, dir := range dirs {
			layer.files = append(layer.files, filepath.Join(dir, "config.json"), filepath.Join(dir, "config.jsonc"))
		}
		layers = append(layers, layer)
	}
	if root, err := filepath.Abs(strings.TrimSpace(projectPath)); err == nil && strings.TrimSpace(projectPath) != "" {
		dir := filepath.Join(root, ".aivo")
		layers = append(layers, runtimeConfigLayer{scope: "project", files: []string{
			filepath.Join(root, "aivo.json"), filepath.Join(root, "aivo.jsonc"), filepath.Join(dir, "config.json"), filepath.Join(dir, "config.jsonc"),
		}, markdownDirs: []string{dir}})
	}
	return layers
}

func readRuntimeConfig(path string) (domain.RuntimeConfig, bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return domain.RuntimeConfig{}, false, nil
	}
	if err != nil {
		return domain.RuntimeConfig{}, true, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return domain.RuntimeConfig{}, true, fmt.Errorf("runtime config must be a regular non-symlink file")
	}
	if info.Size() > runtimeConfigMaxBytes {
		return domain.RuntimeConfig{}, true, fmt.Errorf("runtime config exceeds %d bytes", runtimeConfigMaxBytes)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return domain.RuntimeConfig{}, true, err
	}
	standardized, err := hujson.Standardize(raw)
	if err != nil {
		return domain.RuntimeConfig{}, true, fmt.Errorf("decode runtime config: %w", err)
	}
	var cfg domain.RuntimeConfig
	decoder := json.NewDecoder(bytes.NewReader(standardized))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return domain.RuntimeConfig{}, true, fmt.Errorf("decode runtime config: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return domain.RuntimeConfig{}, true, fmt.Errorf("runtime config contains trailing JSON values")
	}
	return cfg, true, nil
}

type runtimeMarkdownEntry struct {
	Path   string
	Config domain.RuntimeConfig
}

func loadRuntimeMarkdownEntries(root string) ([]runtimeMarkdownEntry, []domain.RuntimeConfigDiagnostic) {
	var entries []runtimeMarkdownEntry
	var diagnostics []domain.RuntimeConfigDiagnostic
	for _, kind := range []string{"agent", "agents", "mode", "modes", "command", "commands"} {
		base := filepath.Join(root, kind)
		_ = filepath.WalkDir(base, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 || !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
				return nil
			}
			info, err := entry.Info()
			if err != nil || !info.Mode().IsRegular() || info.Size() > runtimeConfigMaxBytes {
				diagnostics = append(diagnostics, domain.RuntimeConfigDiagnostic{Path: path, Level: "error", Message: "runtime markdown entry must be a bounded regular non-symlink file"})
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				diagnostics = append(diagnostics, domain.RuntimeConfigDiagnostic{Path: path, Level: "error", Message: err.Error()})
				return nil
			}
			values, content, err := parseRuntimeMarkdown(string(raw))
			if err != nil {
				diagnostics = append(diagnostics, domain.RuntimeConfigDiagnostic{Path: path, Level: "error", Message: err.Error()})
				return nil
			}
			rel, _ := filepath.Rel(base, path)
			name := filepath.ToSlash(strings.TrimSuffix(rel, filepath.Ext(rel)))
			var cfg domain.RuntimeConfig
			if kind == "command" || kind == "commands" {
				command, err := commandDefinitionFromMarkdown(values, content)
				if err != nil {
					diagnostics = append(diagnostics, domain.RuntimeConfigDiagnostic{Path: path, Level: "error", Message: err.Error()})
					return nil
				}
				cfg.Commands = map[string]domain.CommandTemplateDefinition{name: command}
			} else {
				agent, err := agentDefinitionFromMarkdown(values, content, kind == "mode" || kind == "modes")
				if err != nil {
					diagnostics = append(diagnostics, domain.RuntimeConfigDiagnostic{Path: path, Level: "error", Message: err.Error()})
					return nil
				}
				cfg.Agents = map[string]domain.AgentRuntimeDefinition{name: agent}
			}
			if validation := validateRuntimeConfig(path, cfg); len(validation) > 0 {
				diagnostics = append(diagnostics, validation...)
				return nil
			}
			entries = append(entries, runtimeMarkdownEntry{Path: path, Config: cfg})
			return nil
		})
	}
	return entries, diagnostics
}

func parseRuntimeMarkdown(raw string) (map[string]string, string, error) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(raw, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, "", errors.New("runtime markdown must start with YAML frontmatter")
	}
	end := -1
	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) == "---" {
			end = index
			break
		}
	}
	if end < 0 {
		return nil, "", errors.New("runtime markdown frontmatter is not closed")
	}
	var decoded map[string]any
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &decoded); err != nil {
		return nil, "", fmt.Errorf("decode runtime markdown frontmatter: %w", err)
	}
	values := map[string]string{}
	for key, value := range decoded {
		if value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			values[strings.TrimSpace(key)] = typed
		case bool, int, int64, uint64, float32, float64:
			values[strings.TrimSpace(key)] = fmt.Sprint(typed)
		default:
			rawValue, err := json.Marshal(typed)
			if err != nil {
				return nil, "", fmt.Errorf("encode runtime markdown field %q: %w", key, err)
			}
			values[strings.TrimSpace(key)] = string(rawValue)
		}
	}
	return values, strings.TrimSpace(strings.Join(lines[end+1:], "\n")), nil
}

func commandDefinitionFromMarkdown(values map[string]string, content string) (domain.CommandTemplateDefinition, error) {
	if strings.TrimSpace(content) == "" {
		return domain.CommandTemplateDefinition{}, errors.New("command markdown template is empty")
	}
	definition := domain.CommandTemplateDefinition{
		Description: values["description"], Template: strings.TrimSpace(content), Agent: values["agent"],
		Toolsets: parseRuntimeStringList(values["toolsets"]), Subtask: parseRuntimeBool(values["subtask"]),
	}
	if model := parseRuntimeModelRef(values["model"]); model != nil {
		definition.Model = model
	}
	return definition, nil
}

func agentDefinitionFromMarkdown(values map[string]string, content string, primary bool) (domain.AgentRuntimeDefinition, error) {
	definition := domain.AgentRuntimeDefinition{
		DisplayName: values["name"], Description: values["description"], Prompt: strings.TrimSpace(content),
		Toolsets: parseRuntimeStringList(values["toolsets"]), PermissionScope: firstNonEmpty(values["permissionScope"], values["permission_scope"]),
		Mode: values["mode"], Variant: values["variant"], Hidden: parseRuntimeBool(values["hidden"]),
		Disabled: parseRuntimeBool(firstNonEmpty(values["disabled"], values["disable"])),
	}
	if primary {
		definition.Mode = "primary"
	}
	definition.Model = parseRuntimeModelRef(values["model"])
	if value, err := parseOptionalRuntimeFloat(values["temperature"]); err != nil {
		return domain.AgentRuntimeDefinition{}, err
	} else {
		definition.Temperature = value
	}
	if value, err := parseOptionalRuntimeFloat(firstNonEmpty(values["topP"], values["top_p"])); err != nil {
		return domain.AgentRuntimeDefinition{}, err
	} else {
		definition.TopP = value
	}
	if raw := firstNonEmpty(values["maxSteps"], values["steps"]); raw != "" {
		steps, err := strconv.Atoi(raw)
		if err != nil {
			return domain.AgentRuntimeDefinition{}, errors.New("agent steps must be an integer")
		}
		definition.MaxSteps = steps
	}
	if raw := values["options"]; raw != "" {
		if err := json.Unmarshal([]byte(raw), &definition.Options); err != nil {
			return domain.AgentRuntimeDefinition{}, errors.New("agent options must be a JSON object")
		}
	}
	return definition, nil
}

func parseRuntimeStringList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var values []string
	if strings.HasPrefix(raw, "[") && json.Unmarshal([]byte(raw), &values) == nil {
		return nonEmptyTrimmedStrings(values)
	}
	return nonEmptyTrimmedStrings(strings.Split(raw, ","))
}

func nonEmptyTrimmedStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}

func parseRuntimeBool(raw string) bool {
	value, _ := strconv.ParseBool(strings.TrimSpace(raw))
	return value
}

func parseOptionalRuntimeFloat(raw string) (*float64, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return nil, errors.New("agent generation setting must be numeric")
	}
	return &value, nil
}

func parseRuntimeModelRef(raw string) *domain.ModelRef {
	provider, model, ok := strings.Cut(strings.TrimSpace(raw), "/")
	if !ok || strings.TrimSpace(provider) == "" || strings.TrimSpace(model) == "" {
		return nil
	}
	return &domain.ModelRef{ProviderID: strings.TrimSpace(provider), ModelID: strings.TrimSpace(model)}
}

func validateRuntimeConfig(path string, cfg domain.RuntimeConfig) []domain.RuntimeConfigDiagnostic {
	var messages []string
	for name, command := range cfg.Commands {
		if strings.TrimSpace(name) == "" || strings.TrimSpace(command.Template) == "" {
			messages = append(messages, "commands require a non-empty name and template")
		}
	}
	for name, agent := range cfg.Agents {
		if _, err := domain.NormalizeAgentMode(name); err != nil {
			messages = append(messages, "agent names must be valid identifiers")
		}
		if agent.Temperature != nil && (*agent.Temperature < 0 || *agent.Temperature > 2) {
			messages = append(messages, "agent temperature must be between 0 and 2")
		}
		if agent.MaxSteps < 0 || agent.MaxSteps > 100 {
			messages = append(messages, "agent maxSteps must be between 0 and 100")
		}
		if agent.TopP != nil && (*agent.TopP < 0 || *agent.TopP > 1) {
			messages = append(messages, "agent topP must be between 0 and 1")
		}
		if agent.Mode != "" && agent.Mode != "primary" && agent.Mode != "subagent" && agent.Mode != "all" {
			messages = append(messages, "agent mode must be primary, subagent, or all")
		}
	}
	if cfg.DefaultAgent != "" {
		if agent, ok := cfg.Agents[cfg.DefaultAgent]; ok && (agent.Disabled || agent.Mode == "subagent") {
			messages = append(messages, "defaultAgent must reference an enabled primary/all agent")
		}
	}
	for name, lsp := range cfg.LanguageServers {
		if strings.TrimSpace(name) == "" || (!lsp.Disabled && strings.TrimSpace(lsp.Command) == "") {
			messages = append(messages, "languageServers require a name and command unless disabled")
		}
	}
	for name, provider := range cfg.ProviderExtensions {
		if strings.TrimSpace(name) == "" || strings.TrimSpace(provider.Protocol) == "" {
			messages = append(messages, "providerExtensions require a name and protocol")
		}
		for key, value := range provider.Headers {
			if looksSecretBearing(key) && strings.TrimSpace(value) != "" {
				messages = append(messages, "providerExtensions headers must use credential references instead of secret values")
			}
		}
	}
	if cfg.Compaction.ThresholdPercent < 0 || cfg.Compaction.ThresholdPercent > 100 {
		messages = append(messages, "compaction.thresholdPercent must be between 0 and 100")
	}
	if cfg.MaxParallelChildren < 0 || cfg.MaxParallelChildren > 32 {
		messages = append(messages, "maxParallelChildren must be between 0 and 32")
	}
	sort.Strings(messages)
	diagnostics := make([]domain.RuntimeConfigDiagnostic, 0, len(messages))
	for _, message := range messages {
		diagnostics = append(diagnostics, domain.RuntimeConfigDiagnostic{Path: path, Level: "error", Message: message})
	}
	return diagnostics
}

func mergeRuntimeConfig(dst *domain.RuntimeConfig, src domain.RuntimeConfig) {
	dst.Instructions = append(dst.Instructions, src.Instructions...)
	if strings.TrimSpace(src.DefaultAgent) != "" {
		dst.DefaultAgent = strings.TrimSpace(src.DefaultAgent)
	}
	mergeMap(dst.Commands, src.Commands)
	mergeMap(dst.Agents, src.Agents)
	mergeMap(dst.LanguageServers, src.LanguageServers)
	mergeMap(dst.ProviderExtensions, src.ProviderExtensions)
	mergeMap(dst.Toolsets, src.Toolsets)
	mergeMap(dst.Permissions, src.Permissions)
	if src.Compaction.Auto != nil {
		dst.Compaction.Auto = src.Compaction.Auto
	}
	if src.Compaction.ThresholdPercent > 0 {
		dst.Compaction.ThresholdPercent = src.Compaction.ThresholdPercent
	}
	if src.Compaction.ReserveTokens > 0 {
		dst.Compaction.ReserveTokens = src.Compaction.ReserveTokens
	}
	if src.MaxParallelChildren > 0 {
		dst.MaxParallelChildren = src.MaxParallelChildren
	}
}

func mergeMap[K comparable, V any](dst map[K]V, src map[K]V) {
	for key, value := range src {
		dst[key] = value
	}
}

func looksSecretBearing(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(lower, "authorization") || strings.Contains(lower, "api-key") || strings.Contains(lower, "token") || strings.Contains(lower, "secret")
}
