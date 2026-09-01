package app

import "aivo/core/domain"

func resourceResolveSpec(deferredCount int) domain.ToolSpec {
	_ = deferredCount
	return domain.ToolSpec{
		Name: ResourceResolveName, Description: "Inspect or activate optional resources for this conversation. Use mode inspect when the user asks what tools, Skills, or capabilities are available; it returns bounded summaries and does not activate tools. Use mode use when the currently visible tools, filtered Skill catalog, or active instruction resources cannot perform the required action; selected MCP, extension, and optional tool resources replace the automatic tool set and become ordinary Provider tool schemas in the next Tool Snapshot, selected Skills replace the filtered Skill catalog, and selected extension context is added to the session. Manual tools are unchanged. This does not call tools or bypass permissions.",
		Capability: "resource.resolve", Category: "resource_resolution", RiskLevel: "low", Toolsets: []string{"safe", "coding", "mcp", "extension"},
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{
			"intent":    map[string]any{"type": "string", "description": "Concise inventory request or missing capability. Describe the required action or requested resource inventory, not a guessed tool name or plan."},
			"mode":      map[string]any{"type": "string", "enum": []string{"inspect", "use"}, "description": "Required. inspect returns non-persistent resource summaries; use activates matching resources for subsequent model steps."},
			"required":  map[string]any{"type": "boolean", "description": "Whether the task cannot proceed without a matching tool. Defaults to true."},
			"source":    map[string]any{"type": "string", "description": "Optional source filter, such as mcp, extension, builtin, or bridge."},
			"category":  map[string]any{"type": "string", "description": "Optional category filter, such as mcp, extension, automation, or filesystem."},
			"riskLevel": map[string]any{"type": "string", "description": "Optional risk filter, such as low, medium, or high."},
		}, "required": []string{"intent", "mode"}, "additionalProperties": false},
	}
}

func toolSearchSpec(deferredCount int) domain.ToolSpec {
	_ = deferredCount
	return domain.ToolSpec{
		Name: ToolSearchName, Description: "Search additional long-tail tools by capability keywords. Returns tool names and descriptions only; use tool_detail to inspect one tool's parameters, then use tool_call to invoke the selected deferred tool.",
		Capability: "tool.search", Category: "tool_discovery", RiskLevel: "low", Toolsets: []string{"safe", "coding", "mcp"},
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{
			"query": map[string]any{"type": "string", "description": "Keywords describing the capability you need."},
			"limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 20, "description": "Maximum number of matches. Defaults to 5."},
		}, "required": []string{"query"}, "additionalProperties": false},
	}
}

func toolListSpec() domain.ToolSpec {
	return domain.ToolSpec{
		Name: ToolListName, Description: "List available tools with names and descriptions only. Use this for MCP/tool inventory, then call tool_detail for the selected tool's exact input schema.",
		Capability: "tool.list", Category: "tool_discovery", RiskLevel: "low", Toolsets: []string{"safe", "coding", "mcp"},
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{
			"source":          map[string]any{"type": "string", "description": "Optional source filter, such as mcp, extension, builtin, or bridge."},
			"category":        map[string]any{"type": "string", "description": "Optional category filter, such as mcp, extension, automation, or filesystem."},
			"query":           map[string]any{"type": "string", "description": "Optional substring filter over name, description, namespace, capability, category, and source."},
			"includeCore":     map[string]any{"type": "boolean", "description": "Include core visible tools such as file, shell, web, bridge, and planning tools. Defaults to true."},
			"includeLongTail": map[string]any{"type": "boolean", "description": "Include deferred long-tail tools from MCP, extensions, automation, and admin sources. Defaults to true."},
			"limit":           map[string]any{"type": "integer", "minimum": 1, "maximum": 200, "description": "Maximum number of tools to return. Defaults to 100."},
			"offset":          map[string]any{"type": "integer", "minimum": 0, "description": "Pagination offset. Defaults to 0."},
		}, "additionalProperties": false},
	}
}

func toolDetailSpec(name string) domain.ToolSpec {
	description := "Load the full JSON schema and metadata for one tool returned by tool_list or tool_search."
	return domain.ToolSpec{
		Name: name, Description: description,
		Capability: "tool.describe", Category: "tool_discovery", RiskLevel: "low", Toolsets: []string{"safe", "coding", "mcp"},
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{
			"name": map[string]any{"type": "string", "description": "Exact tool name returned by tool_list or tool_search."},
		}, "required": []string{"name"}, "additionalProperties": false},
	}
}

func toolCallSpec() domain.ToolSpec {
	return domain.ToolSpec{
		Name: ToolCallName, Description: "Invoke a deferred tool by name with arguments matching its schema. Permissions, toolset checks, and hooks run as for direct calls.",
		Capability: "tool.call", Category: "tool_discovery", RiskLevel: "medium", Toolsets: []string{"safe", "coding", "mcp"},
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{
			"name":      map[string]any{"type": "string", "description": "Exact underlying tool name."},
			"arguments": map[string]any{"type": "object", "description": "Arguments for the underlying tool."},
		}, "required": []string{"name", "arguments"}, "additionalProperties": false},
	}
}

func (t *ResourceResolveTool) Spec() domain.ToolSpec { return resourceResolveSpec(0) }
func (t *ToolSearchTool) Spec() domain.ToolSpec      { return toolSearchSpec(0) }
func (t *ToolListTool) Spec() domain.ToolSpec        { return toolListSpec() }
func (t *ToolDetailTool) Spec() domain.ToolSpec {
	return toolDetailSpec(firstNonEmpty(t.name, ToolDetailName))
}
func (t *ToolCallTool) Spec() domain.ToolSpec { return toolCallSpec() }
