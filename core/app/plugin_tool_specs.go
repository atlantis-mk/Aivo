package app

import (
	"encoding/json"
	"strings"

	"aivo/core/domain"
)

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
