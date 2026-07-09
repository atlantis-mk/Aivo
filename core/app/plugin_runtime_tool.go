package app

import (
	"context"
	"encoding/json"

	"aivo/core/domain"
)

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
