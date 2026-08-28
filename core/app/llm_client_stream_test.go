package app

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestDoLLMRequestSurfacesResponsesFailedEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.failed\",\"response\":{\"status\":\"failed\",\"error\":{\"code\":\"server_error\",\"message\":\"The model failed to generate a response.\"}}}\n\n"))
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := doLLMRequest(req, nil, nil)
	if err == nil {
		t.Fatalf("response = %#v, want provider failure", resp)
	}
	var providerErr *ProviderRequestError
	if !errors.As(err, &providerErr) {
		t.Fatalf("error = %T %v, want ProviderRequestError", err, err)
	}
	if providerErr.Class != providerErrorUnavailable || providerErr.Message != "response failed (server_error)" {
		t.Fatalf("provider error = %#v", providerErr)
	}
	if strings.Contains(err.Error(), "did not include text") || strings.Contains(err.Error(), "The model failed") {
		t.Fatalf("error = %q, want safe structured failure summary", err)
	}
}

func TestDoLLMRequestSurfacesResponsesIncompleteEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.incomplete\",\"response\":{\"status\":\"incomplete\",\"incomplete_details\":{\"reason\":\"max_output_tokens\"}}}\n\n"))
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = doLLMRequest(req, nil, nil)
	var providerErr *ProviderRequestError
	if !errors.As(err, &providerErr) {
		t.Fatalf("error = %T %v, want ProviderRequestError", err, err)
	}
	if providerErr.Class != providerErrorContext || providerErr.Message != "response incomplete (max_output_tokens)" {
		t.Fatalf("provider error = %#v", providerErr)
	}
}

func TestDoLLMRequestSurfacesStreamErrorEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"error\",\"code\":\"rate_limit_exceeded\",\"message\":\"unsafe provider detail\"}\n\n"))
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = doLLMRequest(req, nil, nil)
	var providerErr *ProviderRequestError
	if !errors.As(err, &providerErr) {
		t.Fatalf("error = %T %v, want ProviderRequestError", err, err)
	}
	if providerErr.Class != providerErrorRateLimit || providerErr.Message != "stream error (rate_limit_exceeded)" {
		t.Fatalf("provider error = %#v", providerErr)
	}
	if strings.Contains(err.Error(), "unsafe provider detail") {
		t.Fatalf("error = %q, provider detail must not enter diagnostics", err)
	}
}

func TestDoLLMRequestSurfacesJSONProviderFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"failed","error":{"code":"invalid_request_error","message":"unsafe provider detail"}}`))
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = doLLMRequest(req, nil, nil)
	var providerErr *ProviderRequestError
	if !errors.As(err, &providerErr) {
		t.Fatalf("error = %T %v, want ProviderRequestError", err, err)
	}
	if providerErr.Class != providerErrorBadRequest || providerErr.Message != "response failed (invalid_request_error)" {
		t.Fatalf("provider error = %#v", providerErr)
	}
	if strings.Contains(err.Error(), "unsafe provider detail") {
		t.Fatalf("error = %q, provider detail must not enter diagnostics", err)
	}
}

func TestReadLLMEventStreamEmitsResponsesCustomToolDeltas(t *testing.T) {
	raw := strings.NewReader("event: response.output_item.added\n" +
		"data: {\"type\":\"response.output_item.added\",\"item\":{\"id\":\"ct_1\",\"type\":\"custom_tool_call\",\"status\":\"in_progress\",\"input\":\"\",\"call_id\":\"call_custom\",\"name\":\"custom_tool\"}}\n\n" +
		"event: response.custom_tool_call_input.delta\n" +
		"data: {\"type\":\"response.custom_tool_call_input.delta\",\"item_id\":\"ct_1\",\"delta\":\"custom input\"}\n\n" +
		"event: response.custom_tool_call_input.done\n" +
		"data: {\"type\":\"response.custom_tool_call_input.done\",\"item_id\":\"ct_1\",\"input\":\"custom input\"}\n\n" +
		"event: response.output_item.done\n" +
		"data: {\"type\":\"response.output_item.done\",\"item\":{\"id\":\"ct_1\",\"type\":\"custom_tool_call\",\"status\":\"completed\",\"input\":\"custom input\",\"call_id\":\"call_custom\",\"name\":\"custom_tool\"}}\n\n")
	var toolDeltas []domain.ChatToolCall
	resp, err := readLLMEventStream(raw, nil, func(call domain.ChatToolCall) {
		toolDeltas = append(toolDeltas, call)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].ID != "call_custom" || string(resp.ToolCalls[0].Arguments) != "custom input" {
		t.Fatalf("ToolCalls = %#v, want completed custom tool", resp.ToolCalls)
	}
	if len(toolDeltas) < 2 {
		t.Fatalf("toolDeltas = %#v, want streamed custom deltas", toolDeltas)
	}
}

func TestExtractResponseTextReadsResponsesStreamDeltas(t *testing.T) {
	raw := []byte("event: response.output_text.delta\n" +
		"data: {\"type\":\"response.output_text.delta\",\"item_id\":\"msg_1\",\"delta\":\"Hello\"}\n\n" +
		"event: response.output_text.delta\n" +
		"data: {\"type\":\"response.output_text.delta\",\"item_id\":\"msg_1\",\"delta\":\"!\"}\n\n" +
		"event: response.completed\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\"}}\n\n")

	if got := extractResponseText(raw); got != "Hello!" {
		t.Fatalf("extractResponseText(stream) = %q, want %q", got, "Hello!")
	}
}

func TestDoLLMRequestReadsEventStreamWhenContentTypeIsMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); !strings.Contains(got, "text/event-stream") {
			t.Fatalf("Accept = %q, want text/event-stream", got)
		}
		_, _ = w.Write([]byte("event: response.output_text.delta\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"item_id\":\"msg_1\",\"delta\":\"Hello\"}\n\n"))
		_, _ = w.Write([]byte("event: response.output_text.delta\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"item_id\":\"msg_1\",\"delta\":\" streamed\"}\n\n"))
		_, _ = w.Write([]byte("event: response.completed\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\"}}\n\n"))
	}))
	defer server.Close()
	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept", "text/event-stream")
	var deltas []string

	resp, err := doLLMRequest(req, func(delta string) {
		deltas = append(deltas, delta)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "Hello streamed" {
		t.Fatalf("Text = %q, want streamed response", resp.Text)
	}
	if strings.Join(deltas, "") != "Hello streamed" || len(deltas) != 2 {
		t.Fatalf("deltas = %#v, want two streamed chunks", deltas)
	}
}

func TestReadLLMEventStreamSeparatesResponsesToolArgumentsFromText(t *testing.T) {
	raw := strings.NewReader("event: response.output_item.added\n" +
		"data: {\"type\":\"response.output_item.added\",\"item\":{\"id\":\"fc_1\",\"type\":\"function_call\",\"status\":\"in_progress\",\"arguments\":\"\",\"call_id\":\"call_read\",\"name\":\"read_file\"}}\n\n" +
		"event: response.function_call_arguments.delta\n" +
		"data: {\"type\":\"response.function_call_arguments.delta\",\"item_id\":\"fc_1\",\"delta\":\"{\\\"path\\\"\"}\n\n" +
		"event: response.function_call_arguments.delta\n" +
		"data: {\"type\":\"response.function_call_arguments.delta\",\"item_id\":\"fc_1\",\"delta\":\":\\\"README.md\\\"}\"}\n\n" +
		"event: response.function_call_arguments.done\n" +
		"data: {\"type\":\"response.function_call_arguments.done\",\"item_id\":\"fc_1\",\"arguments\":\"{\\\"path\\\":\\\"README.md\\\"}\"}\n\n" +
		"event: response.output_item.done\n" +
		"data: {\"type\":\"response.output_item.done\",\"item\":{\"id\":\"fc_1\",\"type\":\"function_call\",\"status\":\"completed\",\"arguments\":\"{\\\"path\\\":\\\"README.md\\\"}\",\"call_id\":\"call_read\",\"name\":\"read_file\"}}\n\n")
	var deltas []string

	resp, err := readLLMEventStream(raw, func(delta string) {
		deltas = append(deltas, delta)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(deltas) != 0 || resp.Text != "" {
		t.Fatalf("text deltas = %#v text=%q, want no text", deltas, resp.Text)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls = %#v, want one call", resp.ToolCalls)
	}
	call := resp.ToolCalls[0]
	if call.ID != "call_read" || call.Name != "read_file" || string(call.Arguments) != `{"path":"README.md"}` {
		t.Fatalf("ToolCall = %#v", call)
	}
}

func TestReadLLMEventStreamEmitsResponsesToolArgumentDeltas(t *testing.T) {
	raw := strings.NewReader("event: response.output_item.added\n" +
		"data: {\"type\":\"response.output_item.added\",\"item\":{\"id\":\"fc_1\",\"type\":\"function_call\",\"status\":\"in_progress\",\"arguments\":\"\",\"call_id\":\"call_edit\",\"name\":\"edit\"}}\n\n" +
		"event: response.function_call_arguments.delta\n" +
		"data: {\"type\":\"response.function_call_arguments.delta\",\"item_id\":\"fc_1\",\"delta\":\"{\\\"path\\\":\\\"docs/spec.md\\\"}\"}\n\n")
	var toolDeltas []domain.ChatToolCall

	resp, err := readLLMEventStream(raw, nil, func(call domain.ChatToolCall) {
		toolDeltas = append(toolDeltas, call)
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "" {
		t.Fatalf("Text = %q, want empty text", resp.Text)
	}
	if len(toolDeltas) < 2 {
		t.Fatalf("toolDeltas = %#v, want start and argument delta", toolDeltas)
	}
	last := toolDeltas[len(toolDeltas)-1]
	if last.ID != "call_edit" || last.Name != "edit" {
		t.Fatalf("last tool delta = %#v", last)
	}
	if !strings.Contains(string(last.Arguments), "docs/spec.md") {
		t.Fatalf("last arguments = %q, want streamed tool arguments", string(last.Arguments))
	}
}

func TestReadLLMEventStreamDoesNotEmitFinalResponsesItemAsDelta(t *testing.T) {
	raw := strings.NewReader("event: response.output_item.done\n" +
		"data: {\"type\":\"response.output_item.done\",\"item\":{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"final answer\"}]}}\n\n" +
		"event: response.completed\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\"}}\n\n")
	var deltas []string

	resp, err := readLLMEventStream(raw, func(delta string) {
		deltas = append(deltas, delta)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "final answer" {
		t.Fatalf("Text = %q, want final answer", resp.Text)
	}
	if len(deltas) != 0 {
		t.Fatalf("deltas = %#v, want none for final item", deltas)
	}
}

func TestReadLLMEventStreamDoesNotEmitFinalResponsesCompletedTextAsDelta(t *testing.T) {
	raw := strings.NewReader("event: response.completed\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"output\":[{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"final answer\"}]}]}}\n\n")
	var deltas []string

	resp, err := readLLMEventStream(raw, func(delta string) {
		deltas = append(deltas, delta)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "final answer" {
		t.Fatalf("Text = %q, want final answer", resp.Text)
	}
	if len(deltas) != 0 {
		t.Fatalf("deltas = %#v, want none for completed response", deltas)
	}
}

func TestReadLLMEventStreamCollectsChatCompletionToolDeltas(t *testing.T) {
	raw := strings.NewReader("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_read\",\"type\":\"function\",\"function\":{\"name\":\"read_file\",\"arguments\":\"{\\\"path\\\"\"}}]}}]}\n\n" +
		"data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\":\\\"README.md\\\"}\"}}]},\"finish_reason\":\"tool_calls\"}]}\n\n" +
		"data: [DONE]\n\n")
	var deltas []string

	resp, err := readLLMEventStream(raw, func(delta string) {
		deltas = append(deltas, delta)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(deltas) != 0 || resp.Text != "" {
		t.Fatalf("text deltas = %#v text=%q, want no text", deltas, resp.Text)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls = %#v, want one call", resp.ToolCalls)
	}
	call := resp.ToolCalls[0]
	if call.ID != "call_read" || call.Name != "read_file" || string(call.Arguments) != `{"path":"README.md"}` {
		t.Fatalf("ToolCall = %#v", call)
	}
}

func TestReadLLMEventStreamCollectsAnthropicToolDeltas(t *testing.T) {
	raw := strings.NewReader("event: content_block_start\n" +
		"data: {\"type\":\"content_block_start\",\"index\":1,\"content_block\":{\"type\":\"tool_use\",\"id\":\"toolu_read\",\"name\":\"read_file\",\"input\":{}}}\n\n" +
		"event: content_block_delta\n" +
		"data: {\"type\":\"content_block_delta\",\"index\":1,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"path\\\"\"}}\n\n" +
		"event: content_block_delta\n" +
		"data: {\"type\":\"content_block_delta\",\"index\":1,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\":\\\"README.md\\\"}\"}}\n\n" +
		"event: content_block_stop\n" +
		"data: {\"type\":\"content_block_stop\",\"index\":1}\n\n")
	var deltas []string

	resp, err := readLLMEventStream(raw, func(delta string) {
		deltas = append(deltas, delta)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(deltas) != 0 || resp.Text != "" {
		t.Fatalf("text deltas = %#v text=%q, want no text", deltas, resp.Text)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls = %#v, want one call", resp.ToolCalls)
	}
	call := resp.ToolCalls[0]
	if call.ID != "toolu_read" || call.Name != "read_file" || string(call.Arguments) != `{"path":"README.md"}` {
		t.Fatalf("ToolCall = %#v", call)
	}
}

func TestReadLLMEventStreamEmitsGeminiToolCallAsModelActivity(t *testing.T) {
	raw := strings.NewReader("data: {\"candidates\":[{\"content\":{\"parts\":[{\"functionCall\":{\"name\":\"read_file\",\"args\":{\"path\":\"README.md\"}}}]}}]}\n\n")
	var activities []domain.ChatToolCall

	resp, err := readLLMEventStream(raw, nil, func(call domain.ChatToolCall) {
		activities = append(activities, call)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.ToolCalls) != 1 || len(activities) != 1 {
		t.Fatalf("response calls = %#v, activities = %#v", resp.ToolCalls, activities)
	}
	if activities[0].Name != "read_file" || !strings.Contains(string(activities[0].Arguments), "README.md") {
		t.Fatalf("activity = %#v", activities[0])
	}
}

func TestExtractResponseTextReadsChatCompletionStreamDeltas(t *testing.T) {
	raw := []byte("data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n" +
		"data: {\"choices\":[{\"delta\":{\"content\":\"!\"},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n")

	if got := extractResponseText(raw); got != "Hello!" {
		t.Fatalf("extractResponseText(chat stream) = %q, want %q", got, "Hello!")
	}
}

func TestExtractResponseTextReadsAnthropicStreamDeltas(t *testing.T) {
	raw := []byte("event: content_block_delta\n" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\n" +
		"event: content_block_delta\n" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"!\"}}\n\n" +
		"event: message_stop\n" +
		"data: {\"type\":\"message_stop\"}\n\n")

	if got := extractResponseText(raw); got != "Hello!" {
		t.Fatalf("extractResponseText(anthropic stream) = %q, want %q", got, "Hello!")
	}
}

func TestExtractChatResponseReadsOpenAIUsage(t *testing.T) {
	resp := extractChatResponse([]byte(`{
		"choices":[{"message":{"content":"ok"}}],
		"usage":{"prompt_tokens":11,"completion_tokens":7,"total_tokens":18}
	}`))
	if resp.Usage == nil {
		t.Fatal("usage is nil")
	}
	if resp.Usage.InputTokens != 11 || resp.Usage.OutputTokens != 7 || resp.Usage.TotalTokens != 18 {
		t.Fatalf("usage = %+v, want 11/7/18", resp.Usage)
	}
}

func TestExtractChatResponseNormalizesOpenAICachedInputUsage(t *testing.T) {
	resp := extractChatResponse([]byte(`{
		"choices":[{"message":{"content":"ok"}}],
		"usage":{
			"prompt_tokens":100,
			"completion_tokens":7,
			"total_tokens":107,
			"prompt_tokens_details":{"cached_tokens":90}
		}
	}`))
	if resp.Usage == nil {
		t.Fatal("usage is nil")
	}
	if resp.Usage.InputTokens != 100 || resp.Usage.CacheReadTokens != 90 || resp.Usage.OutputTokens != 7 {
		t.Fatalf("usage = %+v, want input/cache/output 100/90/7", resp.Usage)
	}
	if !resp.Usage.CacheReadTokensAvailable {
		t.Fatalf("usage = %+v, want OpenAI cache availability", resp.Usage)
	}
}

func TestExtractChatResponseNormalizesAnthropicCacheBuckets(t *testing.T) {
	resp := extractChatResponse([]byte(`{
		"content":[{"type":"text","text":"ok"}],
		"usage":{
			"input_tokens":10,
			"output_tokens":5,
			"cache_read_input_tokens":80,
			"cache_creation_input_tokens":10
		}
	}`))
	if resp.Usage == nil {
		t.Fatal("usage is nil")
	}
	if resp.Usage.InputTokens != 100 || resp.Usage.CacheReadTokens != 80 || resp.Usage.CacheWriteTokens != 10 || resp.Usage.TotalTokens != 105 {
		t.Fatalf("usage = %+v, want normalized input/cache-read/cache-write/total 100/80/10/105", resp.Usage)
	}
	if !resp.Usage.CacheReadTokensAvailable || !resp.Usage.CacheWriteTokensAvailable {
		t.Fatalf("usage = %+v, want Anthropic cache bucket availability", resp.Usage)
	}
}

func TestMergeTokenUsageKeepsNormalizedInputWithLaterOutput(t *testing.T) {
	merged := mergeTokenUsage(
		&domain.TokenUsage{InputTokens: 100, TotalTokens: 100, CacheReadTokens: 80, CacheWriteTokens: 10},
		&domain.TokenUsage{OutputTokens: 5, TotalTokens: 5},
	)
	if merged.InputTokens != 100 || merged.OutputTokens != 5 || merged.TotalTokens != 105 || merged.CacheReadTokens != 80 || merged.CacheWriteTokens != 10 {
		t.Fatalf("usage = %+v, want merged normalized input/output/total/cache buckets", merged)
	}
}

func TestExtractChatResponseReadsOpenAICompatibleContentPartArrays(t *testing.T) {
	resp := extractChatResponse([]byte(`{
		"choices":[{"message":{"content":[{"type":"text","text":"ok"}]}}]
	}`))

	if resp.Text != "ok" {
		t.Fatalf("Text = %q, want ok", resp.Text)
	}
}

func TestExtractChatResponseReadsCommonProviderTextEnvelopes(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "ollama chat message", raw: `{"message":{"role":"assistant","content":"ok"}}`},
		{name: "ollama generate response", raw: `{"response":"ok"}`},
		{name: "wrapped responses payload", raw: `{"response":{"output_text":"ok"}}`},
		{name: "bedrock converse output", raw: `{"output":{"message":{"content":[{"text":"ok"}]}}}`},
		{name: "legacy completion text", raw: `{"choices":[{"text":"ok"}]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := extractChatResponse([]byte(tt.raw))
			if resp.Text != "ok" {
				t.Fatalf("Text = %q, want ok", resp.Text)
			}
		})
	}
}

func TestExtractChatResponseReadsGeminiUsageMetadata(t *testing.T) {
	resp := extractChatResponse([]byte(`{
		"candidates":[{"content":{"parts":[{"text":"ok"}]}}],
		"usageMetadata":{"promptTokenCount":13,"candidatesTokenCount":5,"totalTokenCount":18}
	}`))
	if resp.Usage == nil {
		t.Fatal("usage is nil")
	}
	if resp.Usage.InputTokens != 13 || resp.Usage.OutputTokens != 5 || resp.Usage.TotalTokens != 18 {
		t.Fatalf("usage = %+v, want 13/5/18", resp.Usage)
	}
}

func TestExtractChatResponseNormalizesCommonProviderUsageShapes(t *testing.T) {
	tests := []struct {
		name           string
		raw            string
		input          int
		output         int
		reasoning      int
		cacheRead      int
		cacheWrite     int
		cacheAvailable bool
	}{
		{
			name:  "OpenAI Responses",
			raw:   `{"output_text":"ok","usage":{"input_tokens":100,"output_tokens":20,"input_tokens_details":{"cached_tokens":75},"output_tokens_details":{"reasoning_tokens":8}}}`,
			input: 100, output: 20, reasoning: 8, cacheRead: 75, cacheAvailable: true,
		},
		{
			name:  "Anthropic cache TTL details",
			raw:   `{"content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":10,"output_tokens":5,"cache_read_input_tokens":80,"cache_creation":{"ephemeral_5m_input_tokens":6,"ephemeral_1h_input_tokens":4}}}`,
			input: 100, output: 5, cacheRead: 80, cacheWrite: 10, cacheAvailable: true,
		},
		{
			name:  "Gemini thoughts and cache",
			raw:   `{"candidates":[{"content":{"parts":[{"text":"ok"}]}}],"usageMetadata":{"promptTokenCount":100,"candidatesTokenCount":12,"thoughtsTokenCount":8,"cachedContentTokenCount":60,"totalTokenCount":120}}`,
			input: 100, output: 20, reasoning: 8, cacheRead: 60, cacheAvailable: true,
		},
		{
			name:  "Bedrock Converse",
			raw:   `{"output":{"message":{"content":[{"text":"ok"}]}},"usage":{"inputTokens":100,"outputTokens":20,"totalTokens":120,"cacheReadInputTokens":70,"cacheWriteInputTokens":10}}`,
			input: 100, output: 20, cacheRead: 70, cacheWrite: 10, cacheAvailable: true,
		},
		{
			name:  "AI SDK normalized usage",
			raw:   `{"choices":[{"message":{"content":"ok"}}],"usage":{"inputTokens":100,"outputTokens":20,"totalTokens":120,"inputTokenDetails":{"noCacheTokens":20,"cacheReadTokens":70,"cacheWriteTokens":10},"outputTokenDetails":{"reasoningTokens":8}}}`,
			input: 100, output: 20, reasoning: 8, cacheRead: 70, cacheWrite: 10, cacheAvailable: true,
		},
		{
			name:  "OpenAI-compatible usage without cache fields",
			raw:   `{"choices":[{"message":{"content":"ok"}}],"usage":{"prompt_tokens":100,"completion_tokens":20,"total_tokens":120}}`,
			input: 100, output: 20, cacheAvailable: false,
		},
		{
			name:  "explicit zero cache hit",
			raw:   `{"choices":[{"message":{"content":"ok"}}],"usage":{"prompt_tokens":100,"completion_tokens":20,"prompt_tokens_details":{"cached_tokens":0}}}`,
			input: 100, output: 20, cacheAvailable: true,
		},
		{
			name:  "DeepSeek cache buckets",
			raw:   `{"choices":[{"message":{"content":"ok"}}],"usage":{"prompt_tokens":100,"completion_tokens":20,"prompt_cache_hit_tokens":75,"prompt_cache_miss_tokens":25}}`,
			input: 100, output: 20, cacheRead: 75, cacheAvailable: true,
		},
		{
			name:  "Cohere nested token usage",
			raw:   `{"message":{"content":[{"type":"text","text":"ok"}]},"usage":{"tokens":{"input_tokens":100,"output_tokens":20}}}`,
			input: 100, output: 20, cacheAvailable: false,
		},
		{
			name:  "Ollama evaluation counts",
			raw:   `{"message":{"role":"assistant","content":"ok"},"prompt_eval_count":100,"eval_count":20}`,
			input: 100, output: 20, cacheAvailable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := extractChatResponse([]byte(tt.raw))
			if response.Usage == nil {
				t.Fatal("usage is nil")
			}
			usage := response.Usage
			if usage.InputTokens != tt.input || usage.OutputTokens != tt.output || usage.ReasoningTokens != tt.reasoning || usage.CacheReadTokens != tt.cacheRead || usage.CacheWriteTokens != tt.cacheWrite {
				t.Fatalf("usage = %+v, want input/output/reasoning/cache-read/cache-write %d/%d/%d/%d/%d", usage, tt.input, tt.output, tt.reasoning, tt.cacheRead, tt.cacheWrite)
			}
			if usage.CacheReadTokensAvailable != tt.cacheAvailable {
				t.Fatalf("cache availability = %v, want %v (usage %+v)", usage.CacheReadTokensAvailable, tt.cacheAvailable, usage)
			}
		})
	}
}

func TestMergeTokenUsageAcceptsExplicitZeroFinalFields(t *testing.T) {
	merged := mergeTokenUsage(
		&domain.TokenUsage{InputTokens: 10, OutputTokens: 5, CacheReadTokens: 4},
		&domain.TokenUsage{
			InputTokens: 10, OutputTokens: 5, CacheReadTokens: 0,
			InputTokensAvailable: true, OutputTokensAvailable: true, CacheReadTokensAvailable: true,
		},
	)
	if merged.CacheReadTokens != 0 || !merged.CacheReadTokensAvailable {
		t.Fatalf("usage = %+v, want an explicitly reported zero cache read", merged)
	}
}

func TestExtractChatResponsePreservesBoundedURLCitations(t *testing.T) {
	response := extractChatResponse([]byte(`{
		"output":[{"type":"message","content":[{"type":"output_text","text":"Recent result","annotations":[
			{"type":"url_citation","url":"https://example.com/news","title":"Example News"},
			{"type":"url_citation","url":"https://example.com/news","title":"Duplicate"}
		]}]}]
	}`))
	if len(response.Sources) != 1 || response.Sources[0].URL != "https://example.com/news" || !strings.Contains(response.Text, "[Example News](https://example.com/news)") {
		t.Fatalf("response = %+v", response)
	}
}

func TestReadLLMEventStreamCollectsSearchSourcesWithoutDuplicatingText(t *testing.T) {
	raw := strings.NewReader("data: {\"type\":\"response.output_text.delta\",\"delta\":\"Answer\"}\n\n" +
		"data: {\"type\":\"response.output_item.done\",\"item\":{\"type\":\"web_search_call\",\"results\":[{\"type\":\"text_result\",\"url\":\"https://example.com/source\",\"title\":\"Source\"}]}}\n\n" +
		"data: [DONE]\n\n")
	response, err := readLLMEventStream(raw, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if response.Text != "Answer\n\nSources:\n1. [Source](https://example.com/source)" || len(response.Sources) != 1 {
		t.Fatalf("response = %+v", response)
	}
}
