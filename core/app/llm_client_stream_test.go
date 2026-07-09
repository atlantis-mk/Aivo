package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestReadLLMEventStreamEmitsResponsesCustomToolDeltas(t *testing.T) {
	raw := strings.NewReader("event: response.output_item.added\n" +
		"data: {\"type\":\"response.output_item.added\",\"item\":{\"id\":\"ct_1\",\"type\":\"custom_tool_call\",\"status\":\"in_progress\",\"input\":\"\",\"call_id\":\"call_patch\",\"name\":\"apply_patch\"}}\n\n" +
		"event: response.custom_tool_call_input.delta\n" +
		"data: {\"type\":\"response.custom_tool_call_input.delta\",\"item_id\":\"ct_1\",\"delta\":\"*** Begin Patch\\n*** Add File: docs/spec.md\\n+hello\"}\n\n" +
		"event: response.custom_tool_call_input.done\n" +
		"data: {\"type\":\"response.custom_tool_call_input.done\",\"item_id\":\"ct_1\",\"input\":\"*** Begin Patch\\n*** Add File: docs/spec.md\\n+hello\\n*** End Patch\\n\"}\n\n" +
		"event: response.output_item.done\n" +
		"data: {\"type\":\"response.output_item.done\",\"item\":{\"id\":\"ct_1\",\"type\":\"custom_tool_call\",\"status\":\"completed\",\"input\":\"*** Begin Patch\\n*** Add File: docs/spec.md\\n+hello\\n*** End Patch\\n\",\"call_id\":\"call_patch\",\"name\":\"apply_patch\"}}\n\n")
	var toolDeltas []domain.ChatToolCall
	resp, err := readLLMEventStream(raw, nil, func(call domain.ChatToolCall) {
		toolDeltas = append(toolDeltas, call)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].ID != "call_patch" || !strings.Contains(string(resp.ToolCalls[0].Arguments), "docs/spec.md") {
		t.Fatalf("ToolCalls = %#v, want completed custom apply_patch", resp.ToolCalls)
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
		"data: {\"type\":\"response.output_item.added\",\"item\":{\"id\":\"fc_1\",\"type\":\"function_call\",\"status\":\"in_progress\",\"arguments\":\"\",\"call_id\":\"call_patch\",\"name\":\"apply_patch\"}}\n\n" +
		"event: response.function_call_arguments.delta\n" +
		"data: {\"type\":\"response.function_call_arguments.delta\",\"item_id\":\"fc_1\",\"delta\":\"{\\\"patchText\\\":\\\"*** Begin Patch\\\\n*** Add File: docs/spec.md\\\\n+hello\"}\n\n")
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
	if last.ID != "call_patch" || last.Name != "apply_patch" {
		t.Fatalf("last tool delta = %#v", last)
	}
	if !strings.Contains(string(last.Arguments), "docs/spec.md") {
		t.Fatalf("last arguments = %q, want streamed patch text", string(last.Arguments))
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
