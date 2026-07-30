package app

import (
	"encoding/json"
	"os"
)

func hasArg(value string) bool {
	for _, arg := range os.Args {
		if arg == value {
			return true
		}
	}
	return false
}

func textFromMCPToolContent(result map[string]any) string {
	blocks, _ := result["content"].([]any)
	for _, block := range blocks {
		item, _ := block.(map[string]any)
		if item["type"] == "text" {
			text, _ := item["text"].(string)
			return text
		}
	}
	return ""
}

func writeMCPHelperResponse(id any, result map[string]any) {
	raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
	_, _ = os.Stdout.Write(append(raw, '\n'))
}

func writeMCPHelperError(id any, message string) {
	raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": -32000, "message": message}})
	_, _ = os.Stdout.Write(append(raw, '\n'))
}

func mcpToolsChangedHelperTools(name string) map[string]any {
	return map[string]any{"tools": []any{map[string]any{
		"name":        name,
		"description": "Dynamic tool",
		"inputSchema": map[string]any{"type": "object"},
	}}}
}

func mcpHelperResult(method string) map[string]any {
	switch method {
	case "initialize":
		return map[string]any{"protocolVersion": "2025-03-26", "capabilities": map[string]any{}}
	case "tools/list":
		return map[string]any{"tools": []any{map[string]any{
			"name":        "echo",
			"description": "Echo text",
			"inputSchema": map[string]any{"type": "object"},
		}}}
	case "prompts/list":
		return map[string]any{"prompts": []any{map[string]any{
			"name":        "review",
			"description": "Review code",
			"arguments": []any{map[string]any{
				"name":        "path",
				"description": "Path to review",
				"required":    true,
			}},
		}}}
	case "resources/list":
		return map[string]any{"resources": []any{map[string]any{
			"uri":         "file:///README.md",
			"name":        "README.md",
			"description": "Project readme",
			"mimeType":    "text/markdown",
		}}}
	case "resources/templates/list":
		return map[string]any{"resourceTemplates": []any{map[string]any{
			"uriTemplate": "file:///{path}",
			"name":        "Project file",
			"description": "Read a project file",
			"mimeType":    "text/plain",
		}}}
	case "prompts/get":
		return map[string]any{
			"description": "Review code",
			"messages": []any{map[string]any{
				"role": "user",
				"content": map[string]any{
					"type": "text",
					"text": "Review README.md",
				},
			}},
		}
	case "resources/read":
		return map[string]any{"contents": []any{map[string]any{
			"uri":      "file:///README.md",
			"mimeType": "text/markdown",
			"text":     "# Aivo\n",
		}}}
	default:
		return map[string]any{}
	}
}
