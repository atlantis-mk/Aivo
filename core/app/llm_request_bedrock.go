package app

import (
	"encoding/json"
	"strings"

	"aivo/core/domain"
)

func bedrockConverseRequestBody(messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string) map[string]any {
	conversation := make([]map[string]any, 0, len(messages))
	var system []map[string]string
	for _, message := range messages {
		if message.Role == "system" {
			if text := strings.TrimSpace(message.Text); text != "" {
				system = append(system, map[string]string{"text": text})
			}
			continue
		}
		if message.Role == "tool" {
			content := []map[string]any{{
				"toolResult": map[string]any{
					"toolUseId": message.ToolCallID,
					"content":   bedrockToolResultContent(message.Text),
				},
			}}
			conversation = append(conversation, map[string]any{"role": "user", "content": content})
			continue
		}
		role := "user"
		if message.Role == "assistant" {
			role = "assistant"
		}
		content := make([]map[string]any, 0, len(message.ToolCalls)+1)
		if text := strings.TrimSpace(message.Text); text != "" {
			content = append(content, map[string]any{"text": text})
		}
		for _, call := range message.ToolCalls {
			var input any = map[string]any{}
			_ = json.Unmarshal(call.Arguments, &input)
			content = append(content, map[string]any{
				"toolUse": map[string]any{
					"toolUseId": call.ID,
					"name":      call.Name,
					"input":     input,
				},
			})
		}
		if len(content) == 0 {
			continue
		}
		conversation = append(conversation, map[string]any{"role": role, "content": content})
	}
	body := map[string]any{
		"messages":        conversation,
		"inferenceConfig": map[string]any{"maxTokens": 4096},
	}
	if len(system) > 0 {
		body["system"] = system
	}
	if len(tools) > 0 {
		body["toolConfig"] = map[string]any{
			"tools":      bedrockTools(tools),
			"toolChoice": map[string]any{"auto": map[string]any{}},
		}
	}
	if budget := bedrockReasoningBudget(reasoningEffort); budget > 0 {
		body["additionalModelRequestFields"] = map[string]any{
			"thinking": map[string]any{"type": "enabled", "budget_tokens": budget},
		}
	}
	return body
}

func bedrockToolResultContent(text string) []map[string]any {
	text = strings.TrimSpace(text)
	if text == "" {
		return []map[string]any{{"text": ""}}
	}
	var parsed any
	if json.Unmarshal([]byte(text), &parsed) == nil {
		return []map[string]any{{"json": parsed}}
	}
	return []map[string]any{{"text": text}}
}

func bedrockReasoningBudget(reasoningEffort string) int {
	switch normalizeReasoningEffort(reasoningEffort) {
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
