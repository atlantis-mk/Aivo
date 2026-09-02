package app

import (
	"encoding/json"
	"fmt"
	"strings"

	"aivo/core/domain"
)

func responsesTools(specs []domain.ToolSpec) []map[string]any {
	tools := make([]map[string]any, 0, len(specs))
	namespaceIndices := map[string]int{}
	for _, spec := range specs {
		if spec.Hosted != nil {
			if hosted := responsesHostedTool(spec.Hosted); hosted != nil {
				tools = append(tools, hosted)
			}
			continue
		}
		if spec.Kind == domain.ToolKindFreeform {
			tool := map[string]any{
				"type":        "custom",
				"name":        spec.Name,
				"description": spec.Description,
			}
			if spec.Format != nil {
				tool["format"] = spec.Format
			}
			tools = append(tools, tool)
			continue
		}
		tool := map[string]any{
			"type":        "function",
			"name":        spec.Name,
			"description": spec.Description,
			"parameters":  spec.InputSchema,
		}
		if spec.Strict != nil {
			tool["strict"] = *spec.Strict
		}
		namespace := responsesToolNamespace(spec)
		if namespace == "" {
			tools = append(tools, tool)
			continue
		}
		if index, ok := namespaceIndices[namespace]; ok {
			namespaceTools, _ := tools[index]["tools"].([]map[string]any)
			tools[index]["tools"] = append(namespaceTools, tool)
			continue
		}
		description := strings.TrimSpace(spec.NamespaceDescription)
		if description == "" {
			description = fmt.Sprintf("Tools in the %s namespace.", namespace)
		}
		namespaceIndices[namespace] = len(tools)
		tools = append(tools, map[string]any{
			"type":        "namespace",
			"name":        namespace,
			"description": description,
			"tools":       []map[string]any{tool},
		})
	}
	return tools
}

func responsesToolNamespace(spec domain.ToolSpec) string {
	namespace := strings.TrimSpace(spec.Namespace)
	if !providerSafeToolName(namespace) {
		return ""
	}
	return namespace
}

func responsesHostedTool(spec *domain.HostedToolSpec) map[string]any {
	if spec == nil {
		return nil
	}
	switch strings.TrimSpace(spec.Type) {
	case "web_search", "web_search_preview", "x_search":
		tool := map[string]any{"type": strings.TrimSpace(spec.Type)}
		if spec.ExternalWebAccess != nil {
			tool["external_web_access"] = *spec.ExternalWebAccess
		}
		if spec.IndexedWebAccess != nil {
			tool["indexed_web_access"] = *spec.IndexedWebAccess
		}
		if size := strings.TrimSpace(spec.SearchContextSize); size != "" {
			tool["search_context_size"] = size
		}
		if len(spec.AllowedDomains) > 0 {
			tool["filters"] = map[string]any{"allowed_domains": append([]string(nil), spec.AllowedDomains...)}
		}
		if spec.UserLocation != nil {
			location := hostedUserLocationMap(spec.UserLocation)
			if len(location) > 0 {
				tool["user_location"] = location
			}
		}
		if len(spec.SearchContentTypes) > 0 {
			tool["search_content_types"] = append([]string(nil), spec.SearchContentTypes...)
		}
		return tool
	case "code_interpreter":
		tool := map[string]any{"type": "code_interpreter"}
		container := map[string]any{"type": "auto"}
		if id := strings.TrimSpace(spec.ContainerID); id != "" {
			container = map[string]any{"type": "container_id", "container_id": id}
		}
		if len(spec.FileIDs) > 0 {
			container["files"] = append([]string(nil), spec.FileIDs...)
		}
		tool["container"] = container
		return tool
	case "file_search":
		if len(spec.VectorStoreIDs) == 0 {
			return nil
		}
		return map[string]any{
			"type":             "file_search",
			"vector_store_ids": append([]string(nil), spec.VectorStoreIDs...),
		}
	case "mcp":
		serverURL := strings.TrimSpace(spec.ServerURL)
		serverLabel := strings.TrimSpace(spec.ServerLabel)
		if serverURL == "" || serverLabel == "" {
			return nil
		}
		tool := map[string]any{"type": "mcp", "server_url": serverURL, "server_label": serverLabel}
		if len(spec.AllowedTools) > 0 {
			tool["allowed_tools"] = append([]string(nil), spec.AllowedTools...)
		}
		return tool
	default:
		return nil
	}
}

func hostedUserLocationMap(location *domain.WebSearchUserLocation) map[string]any {
	if location == nil {
		return nil
	}
	out := map[string]any{}
	if value := strings.TrimSpace(location.Type); value != "" {
		out["type"] = value
	}
	if value := strings.TrimSpace(location.Country); value != "" {
		out["country"] = value
	}
	if value := strings.TrimSpace(location.Region); value != "" {
		out["region"] = value
	}
	if value := strings.TrimSpace(location.City); value != "" {
		out["city"] = value
	}
	if value := strings.TrimSpace(location.Timezone); value != "" {
		out["timezone"] = value
	}
	return out
}

func chatCompletionTools(specs []domain.ToolSpec) []map[string]any {
	tools := make([]map[string]any, 0, len(specs))
	for _, spec := range specs {
		if spec.Hosted != nil {
			continue
		}
		tools = append(tools, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        spec.Name,
				"description": spec.Description,
				"parameters":  spec.InputSchema,
			},
		})
	}
	return tools
}

func anthropicTools(specs []domain.ToolSpec) []map[string]any {
	tools := make([]map[string]any, 0, len(specs))
	for _, spec := range specs {
		if spec.Hosted != nil {
			if tool := anthropicHostedTool(spec.Hosted); tool != nil {
				tools = append(tools, tool)
			}
			continue
		}
		tools = append(tools, map[string]any{
			"name":         spec.Name,
			"description":  spec.Description,
			"input_schema": spec.InputSchema,
		})
	}
	return tools
}

func anthropicHostedTool(spec *domain.HostedToolSpec) map[string]any {
	if spec == nil {
		return nil
	}
	if !strings.HasPrefix(spec.Type, "web_search_") && !strings.HasPrefix(spec.Type, "web_fetch_") && !strings.HasPrefix(spec.Type, "code_execution_") {
		return nil
	}
	name := "web_search"
	if strings.HasPrefix(spec.Type, "web_fetch_") {
		name = "web_fetch"
	} else if strings.HasPrefix(spec.Type, "code_execution_") {
		name = "code_execution"
	}
	tool := map[string]any{
		"type": spec.Type,
		"name": name,
	}
	if spec.MaxUses > 0 {
		tool["max_uses"] = spec.MaxUses
	}
	if len(spec.AllowedDomains) > 0 {
		tool["allowed_domains"] = append([]string(nil), spec.AllowedDomains...)
	}
	if spec.UserLocation != nil {
		location := hostedUserLocationMap(spec.UserLocation)
		if len(location) > 0 {
			tool["user_location"] = location
		}
	}
	return tool
}

func googleTools(specs []domain.ToolSpec) []map[string]any {
	declarations := make([]map[string]any, 0, len(specs))
	tools := make([]map[string]any, 0, len(specs))
	for _, spec := range specs {
		if spec.Hosted != nil {
			switch spec.Hosted.Type {
			case "google_search":
				tools = append(tools, map[string]any{"google_search": map[string]any{}})
			case "url_context":
				tools = append(tools, map[string]any{"url_context": map[string]any{}})
			case "code_execution":
				tools = append(tools, map[string]any{"code_execution": map[string]any{}})
			case "file_search":
				if len(spec.Hosted.VectorStoreIDs) > 0 {
					tools = append(tools, map[string]any{"file_search": map[string]any{"file_search_store_names": append([]string(nil), spec.Hosted.VectorStoreIDs...)}})
				}
			}
			continue
		}
		declarations = append(declarations, map[string]any{
			"name":        spec.Name,
			"description": spec.Description,
			"parameters":  spec.InputSchema,
		})
	}
	if len(declarations) > 0 {
		tools = append(tools, map[string]any{"functionDeclarations": declarations})
	}
	return tools
}

func bedrockTools(specs []domain.ToolSpec) []map[string]any {
	tools := make([]map[string]any, 0, len(specs))
	for _, spec := range specs {
		if spec.Hosted != nil {
			continue
		}
		schema := spec.InputSchema
		if schema == nil {
			schema = map[string]any{"type": "object"}
		}
		tools = append(tools, map[string]any{
			"toolSpec": map[string]any{
				"name":        spec.Name,
				"description": spec.Description,
				"inputSchema": map[string]any{"json": schema},
			},
		})
	}
	return tools
}

func responsesToolCalls(calls []domain.ChatToolCall) []map[string]any {
	out := make([]map[string]any, 0, len(calls))
	for _, call := range calls {
		item := map[string]any{
			"type":      "function_call",
			"call_id":   call.ID,
			"name":      call.Name,
			"arguments": string(call.Arguments),
		}
		if namespace := strings.TrimSpace(call.Namespace); namespace != "" {
			item["namespace"] = namespace
		}
		out = append(out, item)
	}
	return out
}

func chatCompletionToolCalls(calls []domain.ChatToolCall) []map[string]any {
	out := make([]map[string]any, 0, len(calls))
	for _, call := range calls {
		out = append(out, map[string]any{
			"id":   call.ID,
			"type": "function",
			"function": map[string]any{
				"name":      call.Name,
				"arguments": string(call.Arguments),
			},
		})
	}
	return out
}

func rawJSONFromAny(value any) json.RawMessage {
	switch typed := value.(type) {
	case nil:
		return json.RawMessage(`{}`)
	case string:
		if strings.TrimSpace(typed) == "" {
			return json.RawMessage(`{}`)
		}
		return json.RawMessage(typed)
	default:
		raw, err := json.Marshal(typed)
		if err != nil {
			return json.RawMessage(`{}`)
		}
		return raw
	}
}

func argumentStringFromAny(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	default:
		raw, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(raw)
	}
}

func firstString(item map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, _ := item[key].(string); value != "" {
			return value
		}
	}
	return ""
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
