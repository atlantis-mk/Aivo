package app

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"

	"aivo/core/domain"
)

func doLLMRequest(req *http.Request, onDelta func(string), onToolDelta func(domain.ChatToolCall)) (domain.ChatResponse, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return domain.ChatResponse{}, providerHTTPError(resp.StatusCode, resp.Status, string(raw))
	}
	if shouldReadEventStream(req, resp) {
		response, err := readLLMEventStream(resp.Body, onDelta, onToolDelta)
		if err != nil {
			return domain.ChatResponse{}, err
		}
		if strings.TrimSpace(response.Text) == "" && len(response.ToolCalls) == 0 {
			return domain.ChatResponse{}, providerResponseError("provider response did not include text")
		}
		return response, nil
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	var payload map[string]any
	if json.Unmarshal(raw, &payload) == nil {
		if err := extractProviderPayloadError(payload); err != nil {
			return domain.ChatResponse{}, err
		}
	}
	response := extractChatResponse(raw)
	if strings.TrimSpace(response.Text) == "" && len(response.ToolCalls) == 0 {
		return domain.ChatResponse{}, providerResponseError("provider response did not include text")
	}
	if onToolDelta != nil {
		for _, call := range response.ToolCalls {
			onToolDelta(call)
		}
	}
	if onDelta != nil && strings.TrimSpace(response.Text) != "" && len(response.ToolCalls) == 0 {
		onDelta(response.Text)
	}
	return response, nil
}

func shouldReadEventStream(req *http.Request, resp *http.Response) bool {
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(contentType, "text/event-stream") {
		return true
	}
	return strings.Contains(strings.ToLower(req.Header.Get("Accept")), "text/event-stream")
}

type streamedToolCall struct {
	ID        string
	Namespace string
	Name      string
	Arguments string
}

func readLLMEventStream(reader io.Reader, onDelta func(string), onToolDelta func(domain.ChatToolCall)) (domain.ChatResponse, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var deltas []string
	var completed []string
	var rawLines []string
	var toolCalls []domain.ChatToolCall
	var usage *domain.TokenUsage
	var sources []domain.ChatSource
	responseTools := map[string]*streamedToolCall{}
	chatTools := map[int]*streamedToolCall{}
	anthropicTools := map[int]*streamedToolCall{}
	for scanner.Scan() {
		rawLine := scanner.Text()
		rawLines = append(rawLines, rawLine)
		line := strings.TrimSpace(rawLine)
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
		if err := extractProviderPayloadError(event); err != nil {
			return domain.ChatResponse{}, err
		}
		if nextUsage := extractTokenUsage(event); nextUsage != nil {
			usage = mergeTokenUsage(usage, nextUsage)
		}
		sources = appendUniqueChatSources(sources, extractResponseSources(event)...)
		toolCalls = appendUniqueToolCalls(toolCalls, updateResponsesStreamToolCalls(responseTools, event, onToolDelta)...)
		toolCalls = appendUniqueToolCalls(toolCalls, updateChatCompletionsStreamToolCalls(chatTools, event, onToolDelta)...)
		toolCalls = appendUniqueToolCalls(toolCalls, updateAnthropicStreamToolCalls(anthropicTools, event, onToolDelta)...)
		googleCalls := extractGoogleStreamToolCalls(event)
		previousToolCount := len(toolCalls)
		toolCalls = appendUniqueToolCalls(toolCalls, googleCalls...)
		if onToolDelta != nil {
			for _, call := range toolCalls[previousToolCount:] {
				onToolDelta(call)
			}
		}
		if delta := extractResponseDeltaText(event); delta != "" {
			deltas = append(deltas, delta)
			if onDelta != nil {
				onDelta(delta)
			}
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
			toolCalls = appendUniqueToolCalls(toolCalls, extractResponseToolCalls(response)...)
			if text := extractResponsePayloadText(response); strings.TrimSpace(text) != "" {
				completed = append(completed, text)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return domain.ChatResponse{}, err
	}
	toolCalls = appendUniqueToolCalls(toolCalls, finishChatCompletionsStreamToolCalls(chatTools)...)
	toolCalls = appendUniqueToolCalls(toolCalls, finishIndexedStreamToolCalls(anthropicTools)...)
	if len(deltas) > 0 {
		text := appendResponseSources(strings.Join(deltas, ""), sources)
		return domain.ChatResponse{Text: text, ToolCalls: toolCalls, Usage: usage, Sources: sources}, nil
	}
	if len(completed) > 0 {
		text := appendResponseSources(strings.Join(completed, "\n"), sources)
		return domain.ChatResponse{Text: text, ToolCalls: toolCalls, Usage: usage, Sources: sources}, nil
	}
	if len(rawLines) > 0 {
		response := extractChatResponse([]byte(strings.Join(rawLines, "\n")))
		response.ToolCalls = appendUniqueToolCalls(toolCalls, response.ToolCalls...)
		response.Usage = mergeTokenUsage(response.Usage, usage)
		response.Sources = appendUniqueChatSources(sources, response.Sources...)
		response.Text = appendResponseSources(stripResponseSources(response.Text), response.Sources)
		if strings.TrimSpace(response.Text) != "" {
			return response, nil
		}
		if len(response.ToolCalls) > 0 {
			return response, nil
		}
	}
	return domain.ChatResponse{ToolCalls: toolCalls, Usage: usage, Sources: sources}, nil
}

func appendUniqueChatSources(existing []domain.ChatSource, next ...domain.ChatSource) []domain.ChatSource {
	seen := make(map[string]bool, len(existing)+len(next))
	out := make([]domain.ChatSource, 0, len(existing)+len(next))
	for _, source := range append(append([]domain.ChatSource(nil), existing...), next...) {
		if source.URL == "" || seen[source.URL] || len(out) >= maxResponseSources {
			continue
		}
		seen[source.URL] = true
		out = append(out, source)
	}
	return out
}

func stripResponseSources(text string) string {
	if index := strings.LastIndex(text, "\n\nSources:\n"); index >= 0 {
		return strings.TrimSpace(text[:index])
	}
	return text
}

func previewLogText(text string, limit int) string {
	text = strings.ReplaceAll(text, "\n", "\\n")
	text = strings.ReplaceAll(text, "\r", "\\r")
	if limit <= 0 || len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}

func updateResponsesStreamToolCalls(tools map[string]*streamedToolCall, event map[string]any, onToolDelta func(domain.ChatToolCall)) []domain.ChatToolCall {
	eventType, _ := event["type"].(string)
	if eventType == "response.output_item.added" {
		item, _ := event["item"].(map[string]any)
		itemType, _ := item["type"].(string)
		if itemType != "function_call" && itemType != "custom_tool_call" {
			return nil
		}
		key := firstString(item, "id")
		if key == "" {
			return nil
		}
		tools[key] = &streamedToolCall{
			ID:        firstString(item, "call_id", "id"),
			Namespace: firstString(item, "namespace"),
			Name:      firstString(item, "name"),
			Arguments: argumentStringFromAny(firstNonNil(item["arguments"], item["input"])),
		}
		emitStreamedToolDelta(tools[key], onToolDelta)
		return nil
	}
	if eventType == "response.function_call_arguments.delta" {
		key := firstString(event, "item_id")
		if key == "" {
			return nil
		}
		tool := tools[key]
		if tool == nil {
			tool = &streamedToolCall{ID: key}
			tools[key] = tool
		}
		if delta, _ := event["delta"].(string); delta != "" {
			tool.Arguments += delta
		}
		emitStreamedToolDelta(tool, onToolDelta)
		return nil
	}
	if eventType == "response.custom_tool_call_input.delta" {
		key := firstString(event, "item_id")
		if key == "" {
			return nil
		}
		tool := tools[key]
		if tool == nil {
			tool = &streamedToolCall{ID: key}
			tools[key] = tool
		}
		if delta, _ := event["delta"].(string); delta != "" {
			tool.Arguments += delta
		}
		emitStreamedToolDelta(tool, onToolDelta)
		return nil
	}
	if eventType == "response.custom_tool_call_input.done" {
		key := firstString(event, "item_id")
		tool := tools[key]
		if tool != nil {
			if args, _ := event["input"].(string); args != "" {
				tool.Arguments = args
			}
			emitStreamedToolDelta(tool, onToolDelta)
		}
		return nil
	}
	if eventType == "response.function_call_arguments.done" {
		key := firstString(event, "item_id")
		tool := tools[key]
		if tool != nil {
			if args, _ := event["arguments"].(string); args != "" {
				tool.Arguments = args
			}
			emitStreamedToolDelta(tool, onToolDelta)
		}
		return nil
	}
	if eventType != "response.output_item.done" {
		return nil
	}
	item, _ := event["item"].(map[string]any)
	itemType, _ := item["type"].(string)
	if itemType != "function_call" && itemType != "custom_tool_call" {
		return nil
	}
	key := firstString(item, "id")
	tool := tools[key]
	if tool == nil {
		tool = &streamedToolCall{}
	}
	if id := firstString(item, "call_id", "id"); id != "" {
		tool.ID = id
	}
	if name := firstString(item, "name"); name != "" {
		tool.Name = name
	}
	if namespace := firstString(item, "namespace"); namespace != "" {
		tool.Namespace = namespace
	}
	if args, _ := item["arguments"].(string); args != "" {
		tool.Arguments = args
	}
	if args, _ := item["input"].(string); args != "" {
		tool.Arguments = args
	}
	delete(tools, key)
	return []domain.ChatToolCall{tool.toChatToolCall()}
}

func updateChatCompletionsStreamToolCalls(tools map[int]*streamedToolCall, event map[string]any, onToolDelta func(domain.ChatToolCall)) []domain.ChatToolCall {
	choices, _ := event["choices"].([]any)
	var finished []domain.ChatToolCall
	for _, choiceRaw := range choices {
		choice, _ := choiceRaw.(map[string]any)
		delta, _ := choice["delta"].(map[string]any)
		rawCalls, _ := delta["tool_calls"].([]any)
		for _, rawCall := range rawCalls {
			call, _ := rawCall.(map[string]any)
			index, ok := numberAsInt(call["index"])
			if !ok {
				index = len(tools)
			}
			tool := tools[index]
			if tool == nil {
				tool = &streamedToolCall{}
				tools[index] = tool
			}
			if id, _ := call["id"].(string); id != "" {
				tool.ID = id
			}
			fn, _ := call["function"].(map[string]any)
			if name, _ := fn["name"].(string); name != "" {
				tool.Name = name
			}
			if args, _ := fn["arguments"].(string); args != "" {
				tool.Arguments += args
			}
			emitStreamedToolDelta(tool, onToolDelta)
		}
		if reason, _ := choice["finish_reason"].(string); reason == "tool_calls" || reason == "function_call" {
			finished = append(finished, finishChatCompletionsStreamToolCalls(tools)...)
		}
	}
	return finished
}

func updateAnthropicStreamToolCalls(tools map[int]*streamedToolCall, event map[string]any, onToolDelta func(domain.ChatToolCall)) []domain.ChatToolCall {
	eventType, _ := event["type"].(string)
	index, _ := numberAsInt(event["index"])
	switch eventType {
	case "content_block_start":
		block, _ := event["content_block"].(map[string]any)
		if blockType, _ := block["type"].(string); blockType != "tool_use" {
			return nil
		}
		tools[index] = &streamedToolCall{
			ID:        firstString(block, "id"),
			Name:      firstString(block, "name"),
			Arguments: initialAnthropicToolInput(block["input"]),
		}
		emitStreamedToolDelta(tools[index], onToolDelta)
	case "content_block_delta":
		delta, _ := event["delta"].(map[string]any)
		if deltaType, _ := delta["type"].(string); deltaType != "input_json_delta" {
			return nil
		}
		tool := tools[index]
		if tool == nil {
			tool = &streamedToolCall{}
			tools[index] = tool
		}
		if partial, _ := delta["partial_json"].(string); partial != "" {
			tool.Arguments += partial
		}
		emitStreamedToolDelta(tool, onToolDelta)
	case "content_block_stop":
		tool := tools[index]
		if tool == nil {
			return nil
		}
		delete(tools, index)
		return []domain.ChatToolCall{tool.toChatToolCall()}
	}
	return nil
}

func extractGoogleStreamToolCalls(event map[string]any) []domain.ChatToolCall {
	if _, ok := event["candidates"]; !ok {
		return nil
	}
	return extractResponseToolCalls(event)
}

func initialAnthropicToolInput(value any) string {
	if object, ok := value.(map[string]any); ok && len(object) == 0 {
		return ""
	}
	return argumentStringFromAny(value)
}

func finishChatCompletionsStreamToolCalls(tools map[int]*streamedToolCall) []domain.ChatToolCall {
	return finishIndexedStreamToolCalls(tools)
}

func finishIndexedStreamToolCalls(tools map[int]*streamedToolCall) []domain.ChatToolCall {
	if len(tools) == 0 {
		return nil
	}
	keys := make([]int, 0, len(tools))
	for key := range tools {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	calls := make([]domain.ChatToolCall, 0, len(keys))
	for _, key := range keys {
		calls = append(calls, tools[key].toChatToolCall())
		delete(tools, key)
	}
	return calls
}

func emitStreamedToolDelta(call *streamedToolCall, onToolDelta func(domain.ChatToolCall)) {
	if onToolDelta == nil || call == nil {
		return
	}
	if strings.TrimSpace(call.Name) == "" && strings.TrimSpace(call.ID) == "" && strings.TrimSpace(call.Arguments) == "" {
		return
	}
	onToolDelta(call.toChatToolCall())
}

func (call streamedToolCall) toChatToolCall() domain.ChatToolCall {
	id := call.ID
	if id == "" {
		id = call.Name
	}
	args := strings.TrimSpace(call.Arguments)
	if args == "" {
		args = "{}"
	}
	return domain.ChatToolCall{ID: id, Namespace: call.Namespace, Name: call.Name, Arguments: json.RawMessage(args)}
}

func appendUniqueToolCalls(existing []domain.ChatToolCall, next ...domain.ChatToolCall) []domain.ChatToolCall {
	seen := make(map[string]bool, len(existing)+len(next))
	for _, call := range existing {
		seen[toolCallKey(call)] = true
	}
	for _, call := range next {
		if call.Name == "" && call.ID == "" {
			continue
		}
		if len(call.Arguments) == 0 {
			call.Arguments = json.RawMessage(`{}`)
		}
		key := toolCallKey(call)
		if seen[key] {
			continue
		}
		seen[key] = true
		existing = append(existing, call)
	}
	return existing
}

func toolCallKey(call domain.ChatToolCall) string {
	return firstNonEmpty(call.ID, call.Name) + "\x00" + call.Namespace + "\x00" + call.Name
}

func numberAsInt(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	default:
		return 0, false
	}
}
