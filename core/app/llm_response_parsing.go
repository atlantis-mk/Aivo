package app

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"aivo/core/domain"
)

func extractChatResponse(raw []byte) domain.ChatResponse {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return domain.ChatResponse{Text: extractResponseStreamText(raw)}
	}
	sources := extractResponseSources(payload)
	return domain.ChatResponse{Text: appendResponseSources(extractResponsePayloadText(payload), sources), ToolCalls: extractResponseToolCalls(payload), Usage: extractTokenUsage(payload), Sources: sources}
}

const maxResponseSources = 20

func extractResponseSources(payload map[string]any) []domain.ChatSource {
	sources := []domain.ChatSource{}
	seen := map[string]bool{}
	var visit func(any, bool, int)
	visit = func(value any, sourceContext bool, depth int) {
		if depth > 12 || len(sources) >= maxResponseSources {
			return
		}
		switch typed := value.(type) {
		case []any:
			for _, item := range typed {
				visit(item, sourceContext, depth+1)
			}
		case map[string]any:
			itemType := strings.ToLower(strings.TrimSpace(firstString(typed, "type")))
			isCitation := itemType == "url_citation" || strings.Contains(itemType, "search_result") || strings.Contains(itemType, "text_result")
			if sourceContext || isCitation {
				url := strings.TrimSpace(firstString(typed, "url"))
				if normalized, err := normalizeWebURL(url); err == nil && !seen[normalized] {
					seen[normalized] = true
					sources = append(sources, domain.ChatSource{
						URL: normalized, Title: bounded(strings.TrimSpace(firstString(typed, "title", "name")), 500),
						RefID: bounded(strings.TrimSpace(firstString(typed, "ref_id", "refId")), 200),
					})
				}
			}
			for key, child := range typed {
				key = strings.ToLower(strings.TrimSpace(key))
				visit(child, sourceContext || key == "annotations" || key == "citations" || key == "sources" || key == "results", depth+1)
			}
		}
	}
	visit(payload, false, 0)
	return sources
}

func appendResponseSources(text string, sources []domain.ChatSource) string {
	text = strings.TrimSpace(text)
	if text == "" || len(sources) == 0 {
		return text
	}
	var builder strings.Builder
	builder.WriteString(text)
	builder.WriteString("\n\nSources:")
	for index, source := range sources {
		label := strings.TrimSpace(source.Title)
		if label == "" {
			label = source.URL
		}
		label = strings.NewReplacer("[", "", "]", "", "\n", " ", "\r", " ").Replace(label)
		builder.WriteString(fmt.Sprintf("\n%d. [%s](%s)", index+1, label, source.URL))
	}
	return builder.String()
}

func extractProviderPayloadError(payload map[string]any) error {
	if payload == nil {
		return nil
	}
	eventType := strings.TrimSpace(firstString(payload, "type"))
	response := mapValue(payload, "response")
	if response == nil {
		response = payload
	}
	status := strings.TrimSpace(firstString(response, "status"))
	switch {
	case eventType == "error":
		code := providerErrorCode(payload)
		return providerPayloadStatusError("stream error", code)
	case eventType == "response.failed" || status == "failed":
		code := providerErrorCode(response)
		return providerPayloadStatusError("response failed", code)
	case eventType == "response.incomplete" || status == "incomplete":
		reason := firstString(mapValue(response, "incomplete_details"), "reason")
		return providerPayloadStatusError("response incomplete", reason)
	default:
		return nil
	}
}

func providerErrorCode(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	if code := firstString(payload, "code"); code != "" {
		return code
	}
	if details := mapValue(payload, "error"); details != nil {
		return firstString(details, "code", "type")
	}
	return ""
}

func providerPayloadStatusError(summary string, code string) error {
	code = strings.TrimSpace(code)
	message := summary
	if code != "" {
		message += " (" + code + ")"
	}
	return &ProviderRequestError{Class: providerPayloadErrorClass(code), Message: message}
}

func providerPayloadErrorClass(code string) string {
	normalized := strings.ToLower(strings.TrimSpace(code))
	switch {
	case strings.Contains(normalized, "rate_limit") || strings.Contains(normalized, "quota"):
		return providerErrorRateLimit
	case strings.Contains(normalized, "auth") || strings.Contains(normalized, "permission") || strings.Contains(normalized, "forbidden"):
		return providerErrorAuth
	case strings.Contains(normalized, "timeout"):
		return providerErrorTimeout
	case strings.Contains(normalized, "server") || strings.Contains(normalized, "overload") || strings.Contains(normalized, "unavailable"):
		return providerErrorUnavailable
	case strings.Contains(normalized, "max_output_tokens") || strings.Contains(normalized, "context"):
		return providerErrorContext
	case strings.Contains(normalized, "invalid") || strings.Contains(normalized, "bad_request"):
		return providerErrorBadRequest
	default:
		return providerErrorUnknown
	}
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
	if usage := tokenUsageFromMap(payload); usage != nil {
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
	input, inputAvailable := firstUsageInt(usage,
		"input_tokens", "prompt_tokens", "promptTokenCount", "inputTokenCount", "inputTokens", "prompt_eval_count", "tokens_evaluated",
	)
	output, outputAvailable := firstUsageInt(usage,
		"output_tokens", "completion_tokens", "outputTokenCount", "outputTokens", "eval_count", "tokens_predicted",
	)
	reasoning, reasoningAvailable := firstUsageInt(usage, "reasoning_tokens", "reasoningTokens", "thoughtsTokenCount")
	if candidates, available := firstUsageInt(usage, "candidatesTokenCount"); available && !outputAvailable {
		output = candidates + reasoning
		outputAvailable = true
	}

	cacheRead, cacheReadAvailable := firstUsageInt(usage,
		"cache_read_input_tokens", "cacheReadInputTokens", "cachedContentTokenCount", "cacheReadTokens", "cachedInputTokens",
		"prompt_cache_hit_tokens", "promptCacheHitTokens",
	)
	cacheWrite, cacheWriteAvailable := firstUsageInt(usage,
		"cache_creation_input_tokens", "cacheCreationInputTokens", "cache_write_input_tokens", "cacheWriteInputTokens", "cacheWriteTokens",
	)
	noCacheInput, noCacheInputAvailable := firstUsageInt(usage,
		"uncached_input_tokens", "uncachedInputTokens", "prompt_cache_miss_tokens", "promptCacheMissTokens", "noCacheTokens",
	)
	for _, detailsKey := range []string{"input_tokens_details", "prompt_tokens_details", "inputTokensDetails", "promptTokensDetails", "inputTokenDetails"} {
		if details := mapValue(usage, detailsKey); details != nil {
			if value, available := firstUsageInt(details, "cached_tokens", "cachedTokens", "cache_read_tokens", "cacheReadTokens"); available {
				cacheRead = max(cacheRead, value)
				cacheReadAvailable = true
			}
			if value, available := firstUsageInt(details, "cache_write_tokens", "cacheWriteTokens"); available {
				cacheWrite = max(cacheWrite, value)
				cacheWriteAvailable = true
			}
			if value, available := firstUsageInt(details, "no_cache_tokens", "noCacheTokens"); available {
				noCacheInput = value
				noCacheInputAvailable = true
			}
		}
	}
	for _, detailsKey := range []string{"output_tokens_details", "completion_tokens_details", "outputTokensDetails", "completionTokensDetails", "outputTokenDetails"} {
		if details := mapValue(usage, detailsKey); details != nil {
			if value, available := firstUsageInt(details, "reasoning_tokens", "reasoningTokens"); available {
				reasoning = max(reasoning, value)
				reasoningAvailable = true
			}
		}
	}
	if !cacheWriteAvailable {
		if creation := mapValue(usage, "cache_creation", "cacheCreation"); creation != nil {
			fiveMinutes, fiveMinutesAvailable := firstUsageInt(creation, "ephemeral_5m_input_tokens", "ephemeral5mInputTokens")
			oneHour, oneHourAvailable := firstUsageInt(creation, "ephemeral_1h_input_tokens", "ephemeral1hInputTokens")
			if fiveMinutesAvailable || oneHourAvailable {
				cacheWrite = fiveMinutes + oneHour
				cacheWriteAvailable = true
			}
		}
	}

	_, anthropicRead := usage["cache_read_input_tokens"]
	_, anthropicWrite := usage["cache_creation_input_tokens"]
	_, anthropicCreation := usage["cache_creation"]
	if _, anthropicInput := usage["input_tokens"]; anthropicInput && (anthropicRead || anthropicWrite || anthropicCreation) {
		input += cacheRead + cacheWrite
		inputAvailable = true
	} else if !inputAvailable && noCacheInputAvailable {
		input = noCacheInput + cacheRead + cacheWrite
		inputAvailable = true
	} else if !inputAvailable && (cacheReadAvailable || cacheWriteAvailable) {
		input = cacheRead + cacheWrite
		inputAvailable = true
	}
	if cacheRead+cacheWrite > input {
		input = cacheRead + cacheWrite
		inputAvailable = true
	}
	cacheRead = min(cacheRead, input)
	cacheWrite = min(cacheWrite, input-cacheRead)

	total, totalAvailable := firstUsageInt(usage, "total_tokens", "totalTokenCount", "totalTokens")
	if combined := input + output; (inputAvailable || outputAvailable) && combined > total {
		total = combined
		totalAvailable = true
	}
	if !inputAvailable && !outputAvailable && !totalAvailable && !cacheReadAvailable && !cacheWriteAvailable && !reasoningAvailable {
		for _, nestedKey := range []string{"tokens", "billed_units", "billedUnits"} {
			if nested := tokenUsageFromMap(mapValue(usage, nestedKey)); nested != nil {
				return nested
			}
		}
		return nil
	}
	return &domain.TokenUsage{
		InputTokens:               input,
		OutputTokens:              output,
		TotalTokens:               total,
		CacheReadTokens:           cacheRead,
		CacheWriteTokens:          cacheWrite,
		ReasoningTokens:           reasoning,
		InputTokensAvailable:      inputAvailable,
		OutputTokensAvailable:     outputAvailable,
		TotalTokensAvailable:      totalAvailable,
		CacheReadTokensAvailable:  cacheReadAvailable,
		CacheWriteTokensAvailable: cacheWriteAvailable,
		ReasoningTokensAvailable:  reasoningAvailable,
	}
}

func mapValue(payload map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		if value, _ := payload[key].(map[string]any); value != nil {
			return value
		}
	}
	return nil
}

func firstUsageInt(payload map[string]any, keys ...string) (int, bool) {
	for _, key := range keys {
		if value, ok := nonNegativeUsageInt(payload[key]); ok {
			return value, true
		}
	}
	return 0, false
}

func nonNegativeUsageInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, typed >= 0
	case int64:
		if typed < 0 || int64(int(typed)) != typed {
			return 0, false
		}
		return int(typed), true
	case float64:
		if typed < 0 || math.IsNaN(typed) || math.IsInf(typed, 0) || math.Trunc(typed) != typed {
			return 0, false
		}
		converted := int(typed)
		if float64(converted) != typed {
			return 0, false
		}
		return converted, true
	default:
		return 0, false
	}
}

func mergeTokenUsage(primary *domain.TokenUsage, next *domain.TokenUsage) *domain.TokenUsage {
	if primary == nil {
		return next
	}
	if next == nil {
		return primary
	}
	if next.InputTokensAvailable || next.InputTokens > 0 {
		primary.InputTokens = next.InputTokens
		primary.InputTokensAvailable = true
	}
	if next.OutputTokensAvailable || next.OutputTokens > 0 {
		primary.OutputTokens = next.OutputTokens
		primary.OutputTokensAvailable = true
	}
	if next.TotalTokensAvailable || next.TotalTokens > 0 {
		primary.TotalTokens = next.TotalTokens
		primary.TotalTokensAvailable = true
	}
	if next.CacheReadTokensAvailable || next.CacheReadTokens > 0 {
		primary.CacheReadTokens = next.CacheReadTokens
		primary.CacheReadTokensAvailable = true
	}
	if next.CacheWriteTokensAvailable || next.CacheWriteTokens > 0 {
		primary.CacheWriteTokens = next.CacheWriteTokens
		primary.CacheWriteTokensAvailable = true
	}
	if next.ReasoningTokensAvailable || next.ReasoningTokens > 0 {
		primary.ReasoningTokens = next.ReasoningTokens
		primary.ReasoningTokensAvailable = true
	}
	if combined := primary.InputTokens + primary.OutputTokens; combined > primary.TotalTokens {
		primary.TotalTokens = combined
		primary.TotalTokensAvailable = true
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
			namespace, _ := item["namespace"].(string)
			name, _ := item["name"].(string)
			args := rawJSONFromAny(firstNonNil(item["arguments"], item["input"]))
			calls = append(calls, domain.ChatToolCall{ID: id, Namespace: namespace, Name: name, Arguments: args})
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
