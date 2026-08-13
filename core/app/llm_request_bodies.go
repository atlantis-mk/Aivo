package app

import (
	"strings"

	"aivo/core/domain"
)

func responsesRequestBody(model string, messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string, serviceTier string) map[string]any {
	input := make([]map[string]any, 0, len(messages))
	for _, message := range messages {
		if message.Role == "tool" {
			outputType := "function_call_output"
			if strings.HasPrefix(strings.TrimSpace(message.Text), "{") && strings.Contains(message.Text, `"name":"apply_patch"`) {
				outputType = "custom_tool_call_output"
			}
			input = append(input, map[string]any{
				"type":    outputType,
				"call_id": message.ToolCallID,
				"output":  message.Text,
			})
			continue
		}
		if len(message.ToolCalls) > 0 {
			if strings.TrimSpace(message.Text) != "" {
				input = append(input, responsesMessageItem(message.Role, message.Text, nil))
			}
			for _, call := range message.ToolCalls {
				item := map[string]any{
					"type":      "function_call",
					"call_id":   call.ID,
					"name":      call.Name,
					"arguments": string(call.Arguments),
				}
				if strings.HasPrefix(strings.TrimSpace(string(call.Arguments)), "*** Begin Patch") {
					item = map[string]any{
						"type":    "custom_tool_call",
						"call_id": call.ID,
						"name":    call.Name,
						"input":   string(call.Arguments),
					}
				}
				input = append(input, item)
			}
			continue
		}
		if message.Role == "system" {
			input = append(input, map[string]any{
				"role":    "system",
				"content": message.Text,
			})
			continue
		}
		input = append(input, responsesMessageItem(message.Role, message.Text, message.Attachments))
	}
	body := map[string]any{
		"model":               model,
		"input":               input,
		"tool_choice":         "auto",
		"parallel_tool_calls": len(tools) > 0,
		"stream":              true,
		"store":               false,
	}
	if len(tools) > 0 {
		body["tools"] = responsesTools(tools)
	}
	if effort := responsesReasoningEffort(reasoningEffort); effort != "" {
		body["reasoning"] = map[string]string{"effort": effort}
	}
	if tier := responsesServiceTier(serviceTier); tier != "" {
		body["service_tier"] = tier
	}
	return body
}

func responsesMessageItem(role string, text string, attachments []domain.MessageAttachment) map[string]any {
	contentType := "input_text"
	if role == "assistant" {
		contentType = "output_text"
	}
	content := []map[string]string{}
	if strings.TrimSpace(text) != "" {
		content = append(content, map[string]string{"type": contentType, "text": text})
	}
	if role == "user" {
		for _, attachment := range attachments {
			if part := responsesAttachmentPart(attachment); len(part) > 0 {
				content = append(content, part)
			}
		}
	}
	return map[string]any{
		"role":    role,
		"content": content,
	}
}

func responsesAttachmentPart(attachment domain.MessageAttachment) map[string]string {
	data := strings.TrimSpace(attachment.Data)
	if data == "" {
		text := attachment.Text
		if strings.TrimSpace(text) == "" {
			return nil
		}
		return map[string]string{"type": "input_text", "text": attachment.Name + "\n" + text}
	}
	mimeType := strings.TrimSpace(attachment.MIMEType)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	dataURL := dataURLForAttachment(mimeType, data)
	if strings.HasPrefix(mimeType, "image/") || attachment.Kind == "image" {
		return map[string]string{"type": "input_image", "image_url": dataURL}
	}
	return map[string]string{"type": "input_file", "filename": attachment.Name, "file_data": dataURL}
}

func dataURLForAttachment(mimeType string, data string) string {
	mimeType = strings.TrimSpace(mimeType)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	data = strings.TrimSpace(data)
	if strings.HasPrefix(data, "data:") {
		return data
	}
	return "data:" + mimeType + ";base64," + data
}

func responsesServiceTier(serviceTier string) string {
	serviceTier = normalizeServiceTier(serviceTier)
	if serviceTier == "default" {
		return ""
	}
	return serviceTier
}

func responsesReasoningEffort(effort string) string {
	switch normalizeReasoningEffort(effort) {
	case "low":
		return "low"
	case "high":
		return "high"
	case "ultra":
		return "high"
	case "medium":
		return "medium"
	default:
		return ""
	}
}
