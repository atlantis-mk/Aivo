package app

import (
	"strings"

	"aivo/core/domain"
)

func responsesRequestBody(model string, messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string, serviceTier string) map[string]any {
	input := make([]map[string]any, 0, len(messages))
	for _, message := range messages {
		if message.Role == "tool" {
			input = append(input, map[string]any{
				"type":    "function_call_output",
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
	mimeType := normalizeAttachmentMIME(attachment.MIMEType)
	if !isSupportedBinaryAttachmentMIME(mimeType) {
		return nil
	}
	dataURL := dataURLForAttachment(mimeType, data)
	if dataURL == "" {
		return nil
	}
	if isImageAttachmentMIME(mimeType) {
		return map[string]string{"type": "input_image", "image_url": dataURL}
	}
	return map[string]string{"type": "input_file", "filename": attachment.Name, "file_data": dataURL}
}

func dataURLForAttachment(mimeType string, data string) string {
	mimeType = normalizeAttachmentMIME(mimeType)
	if !isSupportedBinaryAttachmentMIME(mimeType) {
		return ""
	}
	data = strings.TrimSpace(data)
	if strings.HasPrefix(strings.ToLower(data), "data:") {
		_, embeddedMIME, err := attachmentBase64Payload(data)
		if err != nil || embeddedMIME != mimeType {
			return ""
		}
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
	case "none":
		return "none"
	case "minimal":
		return "minimal"
	case "low":
		return "low"
	case "xhigh":
		return "xhigh"
	case "max":
		return "max"
	case "high":
		return "high"
	case "ultra":
		return "ultra"
	case "medium":
		return "medium"
	default:
		return ""
	}
}

func applyCodexRequestCapabilities(body map[string]any, model domain.ModelInfo, tools []domain.ToolSpec, reasoningEffort, serviceTier string) {
	if body == nil {
		return
	}
	effort := codexEffectiveReasoningEffort(model, reasoningEffort)
	if effort == "" || effort == "none" {
		delete(body, "reasoning")
	} else {
		body["reasoning"] = map[string]any{"effort": effort}
	}
	if tier := codexEffectiveServiceTier(model, serviceTier); tier != "" {
		body["service_tier"] = tier
	} else {
		delete(body, "service_tier")
	}
	if model.SupportsParallelToolCalls != nil && !*model.SupportsParallelToolCalls {
		body["parallel_tool_calls"] = false
	}
	if model.SupportsVerbosity != nil && *model.SupportsVerbosity && codexVerbositySupported(model.DefaultVerbosity) {
		body["text"] = map[string]any{"verbosity": model.DefaultVerbosity}
	}
	if !codexModelUsesResponsesLite(model) {
		return
	}
	input, _ := body["input"].([]map[string]any)
	if len(tools) > 0 {
		input = append([]map[string]any{{
			"type": "additional_tools", "role": "developer", "tools": responsesTools(tools),
		}}, input...)
	}
	body["input"] = input
	delete(body, "tools")
	body["parallel_tool_calls"] = false
	reasoning, _ := body["reasoning"].(map[string]any)
	if reasoning == nil {
		reasoning = map[string]any{}
	}
	reasoning["context"] = "all_turns"
	body["reasoning"] = reasoning
}

func codexEffectiveReasoningEffort(model domain.ModelInfo, requested string) string {
	requested = normalizeReasoningEffort(requested)
	if len(model.SupportedReasoningEfforts) == 0 {
		return requested
	}
	if containsString(model.SupportedReasoningEfforts, requested) {
		return requested
	}
	if containsString(model.SupportedReasoningEfforts, model.DefaultReasoningEffort) {
		return model.DefaultReasoningEffort
	}
	return ""
}

func codexEffectiveServiceTier(model domain.ModelInfo, requested string) string {
	requested = normalizeServiceTier(requested)
	if requested == "default" {
		return ""
	}
	declared := requested
	if requested == "priority" {
		declared = "fast"
	}
	if len(model.ServiceTiers) > 0 && !containsString(model.ServiceTiers, declared) && !containsString(model.ServiceTiers, requested) {
		return ""
	}
	return requested
}
