package app

import (
	"encoding/json"
	"strings"

	"aivo/core/domain"
)

func googleRequestBody(model string, messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string) map[string]any {
	contents := make([]map[string]any, 0, len(messages))
	var system []string
	for _, message := range messages {
		if message.Role == "system" {
			system = append(system, message.Text)
			continue
		}
		if message.Role == "tool" {
			var response any = map[string]any{"content": message.Text}
			contents = append(contents, map[string]any{
				"role": "function",
				"parts": []map[string]any{{
					"functionResponse": map[string]any{
						"name":     message.Name,
						"response": response,
					},
				}},
			})
			continue
		}
		role := "user"
		if message.Role == "assistant" {
			role = "model"
		}
		parts := []map[string]any{}
		if strings.TrimSpace(message.Text) != "" {
			parts = append(parts, map[string]any{"text": message.Text})
		}
		if role == "user" {
			parts = append(parts, googleAttachmentParts(message.Attachments)...)
		}
		for _, call := range message.ToolCalls {
			var args any = map[string]any{}
			_ = json.Unmarshal(call.Arguments, &args)
			parts = append(parts, map[string]any{
				"functionCall": map[string]any{
					"name": call.Name,
					"args": args,
				},
			})
		}
		contents = append(contents, map[string]any{"role": role, "parts": parts})
	}
	body := map[string]any{"contents": contents}
	if len(tools) > 0 {
		if serializedTools := googleTools(tools); len(serializedTools) > 0 {
			body["tools"] = serializedTools
		}
	}
	if len(system) > 0 {
		body["systemInstruction"] = map[string]any{
			"parts": []map[string]string{{"text": strings.Join(system, "\n")}},
		}
	}
	if thinkingConfig := googleThinkingConfig(model, reasoningEffort); len(thinkingConfig) > 0 {
		body["generationConfig"] = map[string]any{"thinkingConfig": thinkingConfig}
	}
	return body
}

func googleAttachmentParts(attachments []domain.MessageAttachment) []map[string]any {
	parts := []map[string]any{}
	for _, attachment := range attachments {
		mimeType := normalizeAttachmentMIME(attachment.MIMEType)
		data := strings.TrimSpace(attachment.Data)
		if data == "" {
			if text := attachment.Text; strings.TrimSpace(text) != "" {
				parts = append(parts, map[string]any{"text": attachment.Name + "\n" + text})
			}
			continue
		}
		if !isSupportedBinaryAttachmentMIME(mimeType) {
			continue
		}
		payload, embeddedMIME, err := attachmentBase64Payload(data)
		if err != nil || (embeddedMIME != "" && embeddedMIME != mimeType) {
			continue
		}
		parts = append(parts, map[string]any{
			"inlineData": map[string]string{
				"mimeType": mimeType,
				"data":     payload,
			},
		})
	}
	return parts
}

func googleThinkingConfig(model string, reasoningEffort string) map[string]any {
	effort := normalizeReasoningEffort(reasoningEffort)
	model = strings.ToLower(strings.TrimSpace(model))
	if effort == "" || effort == "medium" || model == "" {
		return nil
	}
	if strings.HasPrefix(model, "gemini-3") {
		return map[string]any{"thinkingLevel": googleThinkingLevel(effort)}
	}
	if strings.HasPrefix(model, "gemini-2.5") {
		return map[string]any{"thinkingBudget": googleThinkingBudget(effort)}
	}
	return nil
}

func googleThinkingLevel(effort string) string {
	switch effort {
	case "low":
		return "low"
	case "high", "ultra":
		return "high"
	default:
		return "medium"
	}
}

func googleThinkingBudget(effort string) int {
	switch effort {
	case "low":
		return 1024
	case "high":
		return 8192
	case "ultra":
		return 24576
	default:
		return -1
	}
}
