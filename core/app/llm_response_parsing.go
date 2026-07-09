package app

import (
	"encoding/json"
	"fmt"
	"strings"

	"aivo/core/domain"
)

func extractChatResponse(raw []byte) domain.ChatResponse {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return domain.ChatResponse{Text: extractResponseStreamText(raw)}
	}
	return domain.ChatResponse{Text: extractResponsePayloadText(payload), ToolCalls: extractResponseToolCalls(payload), Usage: extractTokenUsage(payload)}
}

func extractResponseText(raw []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return extractResponseStreamText(raw)
	}
	return extractResponsePayloadText(payload)
}

func extractResponsePayloadText(payload map[string]any) string {
	if text, _ := payload["output_text"].(string); strings.TrimSpace(text) != "" {
		return text
	}
	if text := textFromContentValue(payload["message"]); strings.TrimSpace(text) != "" {
		return text
	}
	if text := textFromContentValue(payload["content"]); strings.TrimSpace(text) != "" {
		return text
	}
	if text, _ := payload["text"].(string); strings.TrimSpace(text) != "" {
		return text
	}
	if text, _ := payload["response"].(string); strings.TrimSpace(text) != "" {
		return text
	}
	if response, _ := payload["response"].(map[string]any); response != nil {
		if text := extractResponsePayloadText(response); strings.TrimSpace(text) != "" {
			return text
		}
	}
	if choices, ok := payload["choices"].([]any); ok {
		var parts []string
		for _, choice := range choices {
			item, _ := choice.(map[string]any)
			if text, _ := item["text"].(string); strings.TrimSpace(text) != "" {
				parts = append(parts, text)
			}
			message, _ := item["message"].(map[string]any)
			if text := textFromContentValue(message); strings.TrimSpace(text) != "" {
				parts = append(parts, text)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
	}
	if output, ok := payload["output"].([]any); ok {
		var parts []string
		for _, item := range output {
			outputItem, _ := item.(map[string]any)
			content, _ := outputItem["content"].([]any)
			for _, contentItem := range content {
				part, _ := contentItem.(map[string]any)
				if text, _ := part["text"].(string); strings.TrimSpace(text) != "" {
					parts = append(parts, text)
				}
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
	}
	if output, _ := payload["output"].(map[string]any); output != nil {
		if message, _ := output["message"].(map[string]any); message != nil {
			if text := textFromContentValue(message); strings.TrimSpace(text) != "" {
				return text
			}
		}
	}
	if candidates, ok := payload["candidates"].([]any); ok {
		var parts []string
		for _, candidate := range candidates {
			item, _ := candidate.(map[string]any)
			content, _ := item["content"].(map[string]any)
			partsRaw, _ := content["parts"].([]any)
			for _, partRaw := range partsRaw {
				part, _ := partRaw.(map[string]any)
				if text, _ := part["text"].(string); strings.TrimSpace(text) != "" {
					parts = append(parts, text)
				}
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
	}
	return ""
}

func textFromContentValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case map[string]any:
		if text, _ := typed["content"].(string); strings.TrimSpace(text) != "" {
			return text
		}
		if text, _ := typed["text"].(string); strings.TrimSpace(text) != "" {
			return text
		}
		if content, ok := typed["content"].([]any); ok {
			return textFromContentParts(content)
		}
	case []any:
		return textFromContentParts(typed)
	}
	return ""
}

func textFromContentParts(content []any) string {
	var parts []string
	for _, itemRaw := range content {
		item, _ := itemRaw.(map[string]any)
		if text, _ := item["text"].(string); strings.TrimSpace(text) != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

func extractTokenUsage(payload map[string]any) *domain.TokenUsage {
	if payload == nil {
		return nil
	}
	if usage := tokenUsageFromMap(mapValue(payload, "usage")); usage != nil {
		return usage
	}
	if usage := tokenUsageFromMap(mapValue(payload, "usageMetadata", "usage_metadata")); usage != nil {
		return usage
	}
	if response := mapValue(payload, "response"); response != nil {
		if usage := extractTokenUsage(response); usage != nil {
			return usage
		}
	}
	return nil
}

func tokenUsageFromMap(usage map[string]any) *domain.TokenUsage {
	if usage == nil {
		return nil
	}
	input := firstUsageInt(usage, "input_tokens", "prompt_tokens", "promptTokenCount", "inputTokenCount", "inputTokens", "cache_read_input_tokens")
	output := firstUsageInt(usage, "output_tokens", "completion_tokens", "candidatesTokenCount", "outputTokenCount", "outputTokens")
	total := firstUsageInt(usage, "total_tokens", "totalTokenCount", "totalTokens")
	if total == 0 && (input > 0 || output > 0) {
		total = input + output
	}
	if input == 0 && output == 0 && total == 0 {
		return nil
	}
	return &domain.TokenUsage{InputTokens: input, OutputTokens: output, TotalTokens: total}
}

func mapValue(payload map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		if value, _ := payload[key].(map[string]any); value != nil {
			return value
		}
	}
	return nil
}

func firstUsageInt(payload map[string]any, keys ...string) int {
	for _, key := range keys {
		if value, ok := numberAsInt(payload[key]); ok && value > 0 {
			return value
		}
	}
	return 0
}

func mergeTokenUsage(primary *domain.TokenUsage, next *domain.TokenUsage) *domain.TokenUsage {
	if primary == nil {
		return next
	}
	if next == nil {
		return primary
	}
	if next.InputTokens > 0 {
		primary.InputTokens = next.InputTokens
	}
	if next.OutputTokens > 0 {
		primary.OutputTokens = next.OutputTokens
	}
	if next.TotalTokens > 0 {
		primary.TotalTokens = next.TotalTokens
	}
	primary.Estimated = primary.Estimated && next.Estimated
	return primary
}

func extractResponseToolCalls(payload map[string]any) []domain.ChatToolCall {
	var calls []domain.ChatToolCall
	if choices, ok := payload["choices"].([]any); ok {
		for _, choiceRaw := range choices {
			choice, _ := choiceRaw.(map[string]any)
			message, _ := choice["message"].(map[string]any)
			calls = append(calls, extractOpenAIChatToolCalls(message)...)
		}
	}
	if output, ok := payload["output"].([]any); ok {
		for _, itemRaw := range output {
			item, _ := itemRaw.(map[string]any)
			itemType, _ := item["type"].(string)
			if itemType != "function_call" {
				continue
			}
			id := firstString(item, "call_id", "id")
			name, _ := item["name"].(string)
			args := rawJSONFromAny(firstNonNil(item["arguments"], item["input"]))
			calls = append(calls, domain.ChatToolCall{ID: id, Name: name, Arguments: args})
		}
	}
	if content, ok := payload["content"].([]any); ok {
		for _, itemRaw := range content {
			item, _ := itemRaw.(map[string]any)
			itemType, _ := item["type"].(string)
			if itemType != "tool_use" {
				continue
			}
			id, _ := item["id"].(string)
			name, _ := item["name"].(string)
			calls = append(calls, domain.ChatToolCall{ID: id, Name: name, Arguments: rawJSONFromAny(item["input"])})
		}
	}
	if output, _ := payload["output"].(map[string]any); output != nil {
		if message, _ := output["message"].(map[string]any); message != nil {
			if content, _ := message["content"].([]any); content != nil {
				for _, itemRaw := range content {
					item, _ := itemRaw.(map[string]any)
					toolUse, _ := item["toolUse"].(map[string]any)
					if toolUse == nil {
						continue
					}
					id, _ := toolUse["toolUseId"].(string)
					name, _ := toolUse["name"].(string)
					calls = append(calls, domain.ChatToolCall{ID: id, Name: name, Arguments: rawJSONFromAny(toolUse["input"])})
				}
			}
		}
	}
	if candidates, ok := payload["candidates"].([]any); ok {
		for _, candidateRaw := range candidates {
			candidate, _ := candidateRaw.(map[string]any)
			content, _ := candidate["content"].(map[string]any)
			parts, _ := content["parts"].([]any)
			for _, partRaw := range parts {
				part, _ := partRaw.(map[string]any)
				fc, _ := part["functionCall"].(map[string]any)
				if fc == nil {
					continue
				}
				name, _ := fc["name"].(string)
				calls = append(calls, domain.ChatToolCall{ID: name, Name: name, Arguments: rawJSONFromAny(fc["args"])})
			}
		}
	}
	for i := range calls {
		if calls[i].ID == "" {
			calls[i].ID = fmt.Sprintf("call_%d", i+1)
		}
		if len(calls[i].Arguments) == 0 {
			calls[i].Arguments = json.RawMessage(`{}`)
		}
	}
	return calls
}

func extractOpenAIChatToolCalls(message map[string]any) []domain.ChatToolCall {
	if message == nil {
		return nil
	}
	rawCalls, _ := message["tool_calls"].([]any)
	calls := make([]domain.ChatToolCall, 0, len(rawCalls))
	for _, rawCall := range rawCalls {
		call, _ := rawCall.(map[string]any)
		id, _ := call["id"].(string)
		fn, _ := call["function"].(map[string]any)
		if fn == nil {
			continue
		}
		name, _ := fn["name"].(string)
		args, _ := fn["arguments"].(string)
		calls = append(calls, domain.ChatToolCall{ID: id, Name: name, Arguments: json.RawMessage(firstNonEmpty(args, "{}"))})
	}
	return calls
}

func extractResponseStreamText(raw []byte) string {
	var deltas []string
	var completed []string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		if delta := extractResponseDeltaText(event); delta != "" {
			deltas = append(deltas, delta)
			continue
		}
		if text := extractResponsePayloadText(event); strings.TrimSpace(text) != "" {
			completed = append(completed, text)
			continue
		}
		if item, _ := event["item"].(map[string]any); item != nil {
			if text := extractResponsePayloadText(item); strings.TrimSpace(text) != "" {
				completed = append(completed, text)
				continue
			}
		}
		if response, _ := event["response"].(map[string]any); response != nil {
			if text := extractResponsePayloadText(response); strings.TrimSpace(text) != "" {
				completed = append(completed, text)
			}
		}
	}
	if len(deltas) > 0 {
		return strings.Join(deltas, "")
	}
	if len(completed) > 0 {
		return strings.Join(completed, "\n")
	}
	return ""
}

func extractResponseDeltaText(event map[string]any) string {
	if eventType, _ := event["type"].(string); eventType == "response.output_text.delta" || eventType == "response.refusal.delta" {
		if delta, _ := event["delta"].(string); delta != "" {
			return delta
		}
		if text, _ := event["text"].(string); text != "" {
			return text
		}
	}
	if text := extractChatCompletionDeltaText(event); text != "" {
		return text
	}
	if text := extractAnthropicDeltaText(event); text != "" {
		return text
	}
	if _, ok := event["candidates"]; ok {
		return extractResponsePayloadText(event)
	}
	return ""
}

func extractChatCompletionDeltaText(event map[string]any) string {
	choices, _ := event["choices"].([]any)
	var parts []string
	for _, choiceRaw := range choices {
		choice, _ := choiceRaw.(map[string]any)
		delta, _ := choice["delta"].(map[string]any)
		if text, _ := delta["content"].(string); text != "" {
			parts = append(parts, text)
		}
		if content, ok := delta["content"].([]any); ok {
			for _, itemRaw := range content {
				item, _ := itemRaw.(map[string]any)
				if text, _ := item["text"].(string); text != "" {
					parts = append(parts, text)
				}
			}
		}
		if text, _ := delta["text"].(string); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "")
}

func extractAnthropicDeltaText(event map[string]any) string {
	eventType, _ := event["type"].(string)
	switch eventType {
	case "content_block_delta":
		delta, _ := event["delta"].(map[string]any)
		if deltaType, _ := delta["type"].(string); deltaType != "" && deltaType != "text_delta" {
			return ""
		}
		text, _ := delta["text"].(string)
		return text
	case "content_block_start":
		block, _ := event["content_block"].(map[string]any)
		if blockType, _ := block["type"].(string); blockType != "" && blockType != "text" {
			return ""
		}
		text, _ := block["text"].(string)
		return text
	default:
		return ""
	}
}
