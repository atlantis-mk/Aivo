package app

import (
	"encoding/json"
	"strings"

	"aivo/core/domain"
)

func anthropicRequestBody(model string, messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string) map[string]any {
	input := make([]map[string]any, 0, len(messages))
	var system []string
	for _, message := range messages {
		if message.Role == "system" {
			system = append(system, message.Text)
			continue
		}
		if message.Role == "tool" {
			input = append(input, map[string]any{
				"role": "user",
				"content": []map[string]any{{
					"type":        "tool_result",
					"tool_use_id": message.ToolCallID,
					"content":     message.Text,
				}},
			})
			continue
		}
		role := message.Role
		if role == "assistant" {
			role = "assistant"
		} else {
			role = "user"
		}
		item := map[string]any{"role": role, "content": message.Text}
		if role == "user" && len(message.Attachments) > 0 {
			item["content"] = anthropicContentParts(message.Text, message.Attachments)
		}
		if len(message.ToolCalls) > 0 {
			content := make([]map[string]any, 0, len(message.ToolCalls)+1)
			if strings.TrimSpace(message.Text) != "" {
				content = append(content, map[string]any{"type": "text", "text": message.Text})
			}
			content = append(content, anthropicAttachmentParts(message.Attachments)...)
			for _, call := range message.ToolCalls {
				var input any = map[string]any{}
				_ = json.Unmarshal(call.Arguments, &input)
				content = append(content, map[string]any{
					"type":  "tool_use",
					"id":    call.ID,
					"name":  call.Name,
					"input": input,
				})
			}
			item["content"] = content
		}
		input = append(input, item)
	}
	body := map[string]any{
		"model":      model,
		"max_tokens": anthropicDefaultMaxTokens(model),
		"messages":   input,
		"stream":     true,
	}
	if len(tools) > 0 {
		if serializedTools := anthropicTools(tools); len(serializedTools) > 0 {
			body["tools"] = serializedTools
		}
	}
	if len(system) > 0 {
		body["system"] = strings.Join(system, "\n")
	}
	applyAnthropicReasoning(body, model, reasoningEffort)
	return body
}

func anthropicContentParts(text string, attachments []domain.MessageAttachment) []map[string]any {
	content := []map[string]any{}
	if strings.TrimSpace(text) != "" {
		content = append(content, map[string]any{"type": "text", "text": text})
	}
	content = append(content, anthropicAttachmentParts(attachments)...)
	if len(content) == 0 {
		content = append(content, map[string]any{"type": "text", "text": ""})
	}
	return content
}

func anthropicAttachmentParts(attachments []domain.MessageAttachment) []map[string]any {
	parts := []map[string]any{}
	for _, attachment := range attachments {
		mimeType := normalizeAttachmentMIME(attachment.MIMEType)
		data := strings.TrimSpace(attachment.Data)
		if data == "" {
			if text := attachment.Text; strings.TrimSpace(text) != "" {
				parts = append(parts, map[string]any{"type": "text", "text": attachment.Name + "\n" + text})
			}
			continue
		}
		switch {
		case isImageAttachmentMIME(mimeType):
			parts = append(parts, map[string]any{
				"type": "image",
				"source": map[string]string{
					"type":       "base64",
					"media_type": mimeType,
					"data":       data,
				},
			})
		case mimeType == "application/pdf":
			parts = append(parts, map[string]any{
				"type":  "document",
				"title": attachment.Name,
				"source": map[string]string{
					"type":       "base64",
					"media_type": mimeType,
					"data":       data,
				},
			})
		}
	}
	return parts
}

func applyAnthropicReasoning(body map[string]any, model string, reasoningEffort string) {
	effort := normalizeReasoningEffort(reasoningEffort)
	model = strings.ToLower(strings.TrimSpace(model))
	if effort == "" || effort == "medium" || model == "" {
		return
	}
	if usesAnthropicAdaptiveThinking(model) {
		body["thinking"] = map[string]any{"type": "adaptive"}
		body["output_config"] = map[string]any{"effort": anthropicOutputEffort(effort)}
		return
	}
	if !supportsAnthropicBudgetThinking(model) {
		return
	}
	budget := anthropicThinkingBudget(effort)
	if budget <= 0 {
		return
	}
	maxTokens := budget + 4096
	if existing, ok := body["max_tokens"].(int); ok && existing > maxTokens {
		maxTokens = existing
	}
	body["max_tokens"] = maxTokens
	body["thinking"] = map[string]any{"type": "enabled", "budget_tokens": budget}
}

func anthropicDefaultMaxTokens(model string) int {
	model = strings.ToLower(strings.TrimSpace(model))
	limits := []struct {
		match string
		limit int
	}{
		{"claude-3-7-sonnet", 128000},
		{"claude-3-5-sonnet", 8192},
		{"claude-3-5-haiku", 8192},
		{"claude-3-opus", 4096},
		{"claude-3-sonnet", 4096},
		{"claude-3-haiku", 4096},
		{"claude-opus-4-8", 128000},
		{"claude-opus-4-7", 128000},
		{"claude-opus-4-6", 128000},
		{"claude-sonnet-4-6", 64000},
		{"claude-opus-4-5", 64000},
		{"claude-sonnet-4-5", 64000},
		{"claude-haiku-4-5", 64000},
		{"claude-sonnet-4", 64000},
		{"claude-opus-4", 32000},
		{"claude-fable", 128000},
		{"minimax", 131072},
		{"qwen3", 65536},
	}
	for _, item := range limits {
		if strings.Contains(model, item.match) {
			return item.limit
		}
	}
	return 4096
}

func anthropicOutputEffort(effort string) string {
	switch effort {
	case "low":
		return "low"
	case "high":
		return "high"
	case "ultra":
		return "xhigh"
	default:
		return "medium"
	}
}

func usesAnthropicAdaptiveThinking(model string) bool {
	return strings.Contains(model, "4-6") ||
		strings.Contains(model, "4.6") ||
		strings.Contains(model, "4-7") ||
		strings.Contains(model, "4.7") ||
		strings.Contains(model, "4-8") ||
		strings.Contains(model, "4.8") ||
		strings.Contains(model, "claude-fable-5") ||
		strings.Contains(model, "claude-mythos-5")
}

func supportsAnthropicBudgetThinking(model string) bool {
	return strings.Contains(model, "claude-3-7") ||
		strings.Contains(model, "claude-4") ||
		strings.Contains(model, "sonnet-4") ||
		strings.Contains(model, "opus-4")
}

func anthropicThinkingBudget(effort string) int {
	switch effort {
	case "low":
		return 1024
	case "high":
		return 4096
	case "ultra":
		return 8192
	default:
		return 0
	}
}
