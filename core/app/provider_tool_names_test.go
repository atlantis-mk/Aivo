package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aivo/core/domain"
)

func TestGenerateChatResponseUsesCanonicalToolNameUnchanged(t *testing.T) {
	const canonical = "mcp_chrome_list_tabs"
	const historical = "mcp_chrome_previous_tabs"
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		tools, _ := body["tools"].([]any)
		if len(tools) != 1 {
			t.Fatalf("tools = %#v, want one", body["tools"])
		}
		tool, _ := tools[0].(map[string]any)
		function, _ := tool["function"].(map[string]any)
		if got, _ := function["name"].(string); got != canonical {
			t.Fatalf("declared tool name = %q, want unchanged canonical %q", got, canonical)
		}
		messages, _ := body["messages"].([]any)
		assistant, _ := messages[0].(map[string]any)
		calls, _ := assistant["tool_calls"].([]any)
		prior, _ := calls[0].(map[string]any)
		priorFunction, _ := prior["function"].(map[string]any)
		if got, _ := priorFunction["name"].(string); got != historical {
			t.Fatalf("historical tool name = %q, want unchanged canonical %q", got, historical)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_tabs\",\"function\":{\"name\":\"" + canonical + "\",\"arguments\":\"{}\"}}]},\"finish_reason\":\"tool_calls\"}]}\n\n"))
	}))
	defer server.Close()

	cfg := domain.AppConfig{
		Provider:     &domain.ProviderConfig{ID: "tool-name-provider", Type: string(TransportOpenAICompatible), BaseURL: server.URL, Model: "tool-name-model"},
		DefaultModel: &domain.ModelRef{ProviderID: "tool-name-provider", ModelID: "tool-name-model"},
	}
	service := NewService(&memoryProviderStore{config: &cfg})
	if err := service.RegisterProviderDefinition(ProviderDefinition{
		ID: "tool-name-provider", DisplayName: "tool-name-provider", Transport: TransportOpenAICompatible,
		DefaultBaseURL: server.URL, DefaultModelID: "tool-name-model", AuthTypes: []AuthType{AuthNone}, DefaultAuthType: AuthNone,
		Models: []domain.ModelInfo{{ID: "tool-name-model", ProviderID: "tool-name-provider", Name: "tool-name-model", Streaming: true, ToolSupport: true}},
	}); err != nil {
		t.Fatal(err)
	}
	request := domain.ChatRequest{
		Messages: []domain.ChatMessage{
			{Role: "assistant", ToolCalls: []domain.ChatToolCall{{ID: "call_previous", Name: historical, Arguments: json.RawMessage(`{}`)}}},
			{Role: "tool", ToolCallID: "call_previous", Name: historical, Text: "done"},
			{Role: "user", Text: "list tabs"},
		},
		Tools: []domain.ToolSpec{{Name: canonical, Description: "List Chrome tabs", InputSchema: map[string]any{"type": "object"}}},
	}
	for attempt := 0; attempt < 2; attempt++ {
		var streamed []domain.ChatToolCall
		response, _, err := service.GenerateChatResponseStreamWithToolDelta(context.Background(), request, nil, "none", "", nil, func(call domain.ChatToolCall) {
			streamed = append(streamed, call)
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(response.ToolCalls) != 1 || response.ToolCalls[0].Name != canonical {
			t.Fatalf("response calls = %#v, want canonical %q", response.ToolCalls, canonical)
		}
		if len(streamed) == 0 || streamed[len(streamed)-1].Name != canonical {
			t.Fatalf("streamed calls = %#v, want canonical %q", streamed, canonical)
		}
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}

func TestValidateProviderToolIdentitiesRejectsUnsafeNames(t *testing.T) {
	if err := validateProviderToolIdentities([]domain.ToolSpec{{Name: "mcp.chrome.list_tabs"}}, nil); err == nil {
		t.Fatal("unsafe declaration was accepted")
	}
	if err := validateProviderToolIdentities(nil, []llmChatMessage{{Role: "assistant", ToolCalls: []domain.ChatToolCall{{Name: "mcp.chrome.list_tabs"}}}}); err == nil {
		t.Fatal("unsafe historical call was accepted")
	}
	if err := validateProviderToolIdentities([]domain.ToolSpec{{Name: "mcp_chrome_list_tabs"}}, []llmChatMessage{{Role: "assistant", ToolCalls: []domain.ChatToolCall{{Name: "mcp_chrome_list_tabs"}}}}); err != nil {
		t.Fatalf("safe canonical names rejected: %v", err)
	}
}
