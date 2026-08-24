package app

import (
	"encoding/json"
	"strings"

	"aivo/core/domain"
)

func normalizeMCPToolResult(name string, result map[string]any) domain.ToolResult {
	content := ""
	if blocks, ok := result["content"].([]any); ok {
		parts := []string{}
		for _, block := range blocks {
			item, _ := block.(map[string]any)
			if item["type"] == "text" {
				if text, _ := item["text"].(string); text != "" {
					parts = append(parts, text)
				}
			}
		}
		content = strings.Join(parts, "\n")
	}
	if content == "" {
		if text, _ := result["content"].(string); text != "" {
			content = text
		}
	}
	if content == "" {
		raw, _ := json.MarshalIndent(result, "", "  ")
		content = string(raw)
	}
	ok := true
	if isError, _ := result["isError"].(bool); isError {
		ok = false
	}
	return domain.ToolResult{Name: name, OK: ok, Content: content, ModelContent: content, Structured: result, Error: errorFromMCPResult(result)}
}

func normalizeMCPPromptGetResult(serverID string, name string, result map[string]any) domain.MCPPromptGetResult {
	description, _ := result["description"].(string)
	rawMessages, _ := result["messages"].([]any)
	messages := make([]domain.MCPPromptMessage, 0, len(rawMessages))
	parts := []string{}
	for _, rawMessage := range rawMessages {
		item, _ := rawMessage.(map[string]any)
		role, _ := item["role"].(string)
		blocks := parseMCPContentBlocks(item["content"])
		messages = append(messages, domain.MCPPromptMessage{Role: role, Content: blocks})
		text := textFromMCPContentBlocks(blocks)
		if text != "" {
			if role != "" {
				parts = append(parts, role+": "+text)
			} else {
				parts = append(parts, text)
			}
		}
	}
	return domain.MCPPromptGetResult{ServerID: serverID, Name: name, Description: description, Messages: messages, Content: strings.Join(parts, "\n"), Structured: result}
}

func normalizeMCPResourceReadResult(serverID string, uri string, result map[string]any) domain.MCPResourceReadResult {
	rawContents, _ := result["contents"].([]any)
	contents := make([]domain.MCPResourceContent, 0, len(rawContents))
	parts := []string{}
	for _, rawContent := range rawContents {
		item, _ := rawContent.(map[string]any)
		contentURI, _ := item["uri"].(string)
		mimeType, _ := item["mimeType"].(string)
		text, _ := item["text"].(string)
		blob, _ := item["blob"].(string)
		contents = append(contents, domain.MCPResourceContent{URI: contentURI, MimeType: mimeType, Text: text, Blob: blob})
		if text != "" {
			parts = append(parts, text)
		}
	}
	return domain.MCPResourceReadResult{ServerID: serverID, URI: uri, Contents: contents, Content: strings.Join(parts, "\n"), Structured: result}
}

func parseMCPContentBlocks(value any) []domain.MCPContentBlock {
	switch typed := value.(type) {
	case []any:
		blocks := make([]domain.MCPContentBlock, 0, len(typed))
		for _, rawBlock := range typed {
			if block := parseMCPContentBlock(rawBlock); block.Type != "" {
				blocks = append(blocks, block)
			}
		}
		return blocks
	case map[string]any:
		if block := parseMCPContentBlock(typed); block.Type != "" {
			return []domain.MCPContentBlock{block}
		}
	}
	return nil
}

func parseMCPContentBlock(value any) domain.MCPContentBlock {
	item, _ := value.(map[string]any)
	blockType, _ := item["type"].(string)
	text, _ := item["text"].(string)
	uri, _ := item["uri"].(string)
	mimeType, _ := item["mimeType"].(string)
	blob, _ := item["blob"].(string)
	return domain.MCPContentBlock{Type: blockType, Text: text, URI: uri, MimeType: mimeType, Blob: blob}
}

func textFromMCPContentBlocks(blocks []domain.MCPContentBlock) string {
	parts := []string{}
	for _, block := range blocks {
		if block.Text != "" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func errorFromMCPResult(result map[string]any) string {
	if isError, _ := result["isError"].(bool); !isError {
		return ""
	}
	if content, _ := result["content"].(string); content != "" {
		return content
	}
	return "MCP tool returned an error"
}
