package app

import (
	"strings"

	"aivo/core/domain"
)

func chatCompletionsRequestBody(model string, messages []llmChatMessage, tools []domain.ToolSpec) map[string]any {
	input := make([]map[string]any, 0, len(messages))
	for _, message := range messages {
		item := map[string]any{
			"role":    message.Role,
			"content": message.Text,
		}
		if message.Role == "user" && len(message.Attachments) > 0 {
			item["content"] = chatCompletionContentParts(message.Text, message.Attachments)
		}
		if message.Role == "tool" {
			item["tool_call_id"] = message.ToolCallID
			item["name"] = message.Name
		}
		if len(message.ToolCalls) > 0 {
			item["tool_calls"] = chatCompletionToolCalls(message.ToolCalls)
		}
		input = append(input, item)
	}
	body := map[string]any{
		"model":    model,
		"messages": input,
		"stream":   true,
	}
	if len(tools) > 0 {
		if serializedTools := chatCompletionTools(tools); len(serializedTools) > 0 {
			body["tools"] = serializedTools
			body["tool_choice"] = "auto"
		}
	}
	return body
}

func chatCompletionContentParts(text string, attachments []domain.MessageAttachment) []map[string]any {
	parts := []map[string]any{}
	if strings.TrimSpace(text) != "" {
		parts = append(parts, map[string]any{"type": "text", "text": text})
	}
	for _, attachment := range attachments {
		mimeType := normalizeAttachmentMIME(attachment.MIMEType)
		data := strings.TrimSpace(attachment.Data)
		if data == "" {
			if text := attachment.Text; strings.TrimSpace(text) != "" {
				parts = append(parts, map[string]any{"type": "text", "text": attachment.Name + "\n" + text})
			}
			continue
		}
		if !isSupportedBinaryAttachmentMIME(mimeType) {
			continue
		}
		dataURL := dataURLForAttachment(mimeType, data)
		if dataURL == "" {
			continue
		}
		if isImageAttachmentMIME(mimeType) {
			parts = append(parts, map[string]any{
				"type": "image_url",
				"image_url": map[string]string{
					"url": dataURL,
				},
			})
			continue
		}
		parts = append(parts, map[string]any{
			"type": "file",
			"file": map[string]string{
				"filename":  attachment.Name,
				"file_data": dataURL,
			},
		})
	}
	return parts
}

func chatCompletionsReasoningEffort(effort string) string {
	switch normalizeReasoningEffort(effort) {
	case "low":
		return "low"
	case "medium":
		return "medium"
	case "high", "ultra":
		return "high"
	default:
		return ""
	}
}
