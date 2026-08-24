package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestCallGitHubCopilotUsesChatCompletionsEndpointAndHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %q, want /chat/completions", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer copilot-token" {
			t.Fatalf("Authorization = %q, want bearer token", got)
		}
		if got := r.Header.Get("Copilot-Integration-Id"); got == "" {
			t.Fatalf("Copilot-Integration-Id = empty")
		}
		if got := r.Header.Get("Editor-Version"); got == "" {
			t.Fatalf("Editor-Version = empty")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "gpt-5.1-codex-max" {
			t.Fatalf("model = %#v, want gpt-5.1-codex-max", body["model"])
		}
		if body["stream"] != true {
			t.Fatalf("stream = %#v, want true", body["stream"])
		}
		if body["reasoning_effort"] != "low" {
			t.Fatalf("reasoning_effort = %#v, want low", body["reasoning_effort"])
		}
		messages, ok := body["messages"].([]any)
		if !ok || len(messages) != 1 {
			t.Fatalf("messages = %#v, want one message", body["messages"])
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n"))
	}))
	defer server.Close()

	got, err := callGitHubCopilot(
		context.Background(),
		domain.ProviderConfig{ID: "github-copilot", Type: string(TransportGitHubCopilot), BaseURL: server.URL},
		domain.ModelRef{ProviderID: "github-copilot", ModelID: "gpt-5.1-codex-max"},
		llmCredential{APIKey: "copilot-token"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		nil,
		"low",
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "ok" {
		t.Fatalf("reply = %q, want ok", got.Text)
	}
}

func TestCallBedrockConverseSignsRequestAndParsesResponse(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIATEST")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	t.Setenv("AWS_REGION", "us-west-2")
	var gotPath string
	var gotAuth string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		gotAuth = r.Header.Get("Authorization")
		if r.Header.Get("X-Amz-Date") == "" || r.Header.Get("X-Amz-Content-Sha256") == "" {
			t.Fatalf("missing SigV4 headers: %+v", r.Header)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"output":{"message":{"content":[
				{"text":"ok"},
				{"toolUse":{"toolUseId":"tool-1","name":"read_file","input":{"path":"README.md"}}}
			]}},
			"usage":{"inputTokens":12,"outputTokens":5,"totalTokens":17}
		}`))
	}))
	defer server.Close()

	resp, err := callBedrockConverse(
		context.Background(),
		domain.ProviderConfig{ID: "amazon-bedrock", Type: string(TransportBedrockConverse), BaseURL: server.URL},
		domain.ModelRef{ProviderID: "amazon-bedrock", ModelID: "anthropic.claude-sonnet-4-20250514-v1:0"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "system", Text: "be concise"}, {Role: "user", Text: "hello"}},
		[]domain.ToolSpec{{Name: "read_file", Description: "Read file", InputSchema: map[string]any{"type": "object"}}},
		"high",
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/model/anthropic.claude-sonnet-4-20250514-v1:0/converse" {
		t.Fatalf("path = %q", gotPath)
	}
	if !strings.HasPrefix(gotAuth, "AWS4-HMAC-SHA256 Credential=AKIATEST/") || !strings.Contains(gotAuth, "/us-west-2/bedrock/aws4_request") {
		t.Fatalf("Authorization = %q, want SigV4 credential scope", gotAuth)
	}
	if _, ok := gotBody["messages"].([]any); !ok {
		t.Fatalf("messages missing from body: %#v", gotBody)
	}
	if _, ok := gotBody["toolConfig"].(map[string]any); !ok {
		t.Fatalf("toolConfig missing from body: %#v", gotBody)
	}
	if resp.Text != "ok" {
		t.Fatalf("Text = %q, want ok", resp.Text)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].ID != "tool-1" || resp.ToolCalls[0].Name != "read_file" || !strings.Contains(string(resp.ToolCalls[0].Arguments), "README.md") {
		t.Fatalf("ToolCalls = %+v", resp.ToolCalls)
	}
	if resp.Usage == nil || resp.Usage.InputTokens != 12 || resp.Usage.OutputTokens != 5 || resp.Usage.TotalTokens != 17 {
		t.Fatalf("Usage = %+v, want 12/5/17", resp.Usage)
	}
}

func TestCallOpenAICompatibleStreamsWithNonSSEContentTypeWhenDeltaHandlerExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n"))
	}))
	defer server.Close()

	var deltas []string
	got, err := callOpenAICompatible(
		context.Background(),
		domain.ProviderConfig{ID: "openrouter", Type: "openrouter", BaseURL: server.URL},
		domain.ModelRef{ProviderID: "openrouter", ModelID: "openai/gpt-5"},
		llmCredential{APIKey: "test-key"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		nil,
		"",
		"",
		func(delta string) {
			deltas = append(deltas, delta)
		},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "ok" {
		t.Fatalf("reply = %q, want ok", got.Text)
	}
	if len(deltas) != 1 || deltas[0] != "ok" {
		t.Fatalf("deltas = %#v, want [ok]", deltas)
	}
}

func TestCallAnthropicUsesMessagesStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages" {
			t.Fatalf("path = %q, want /messages", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "anthropic-key" {
			t.Fatalf("x-api-key = %q, want anthropic-key", got)
		}
		if got := r.Header.Get("anthropic-version"); got == "" {
			t.Fatal("anthropic-version header is empty")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "claude-sonnet-4-5" {
			t.Fatalf("model = %#v", body["model"])
		}
		if stream, _ := body["stream"].(bool); !stream {
			t.Fatalf("stream = %#v, want true", body["stream"])
		}
		if body["max_tokens"] != float64(64000) {
			t.Fatalf("max_tokens = %#v, want 64000", body["max_tokens"])
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"ok\"}}\n\n"))
	}))
	defer server.Close()

	got, err := callAnthropic(
		context.Background(),
		domain.ProviderConfig{ID: "anthropic", Type: "anthropic", BaseURL: server.URL},
		domain.ModelRef{ProviderID: "anthropic", ModelID: "claude-sonnet-4-5"},
		llmCredential{APIKey: "anthropic-key"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		nil,
		"high",
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "ok" {
		t.Fatalf("reply = %q, want ok", got.Text)
	}
}

func TestCallGoogleUsesStreamGenerateContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models/gemini-2.5-pro:streamGenerateContent" {
			t.Fatalf("path = %q, want /models/gemini-2.5-pro:streamGenerateContent", r.URL.Path)
		}
		if got := r.URL.Query().Get("alt"); got != "sse" {
			t.Fatalf("alt = %q, want sse", got)
		}
		if got := r.URL.Query().Get("key"); got != "gemini-key" {
			t.Fatalf("key = %q, want gemini-key", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if _, ok := body["contents"].([]any); !ok {
			t.Fatalf("contents = %#v, want array", body["contents"])
		}
		generationConfig, _ := body["generationConfig"].(map[string]any)
		thinkingConfig, _ := generationConfig["thinkingConfig"].(map[string]any)
		if thinkingConfig["thinkingBudget"] != float64(1024) {
			t.Fatalf("thinkingBudget = %#v, want 1024", thinkingConfig["thinkingBudget"])
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"ok\"}]}}]}\n\n"))
	}))
	defer server.Close()

	got, err := callGoogle(
		context.Background(),
		domain.ProviderConfig{ID: "gemini", Type: "google", BaseURL: server.URL},
		domain.ModelRef{ProviderID: "gemini", ModelID: "gemini-2.5-pro"},
		llmCredential{APIKey: "gemini-key"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		nil,
		"low",
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "ok" {
		t.Fatalf("reply = %q, want ok", got.Text)
	}
}

func TestCallGoogleVertexUsesPublisherEndpointAndBearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models/gemini-2.5-pro:streamGenerateContent" {
			t.Fatalf("path = %q, want /models/gemini-2.5-pro:streamGenerateContent", r.URL.Path)
		}
		if got := r.URL.Query().Get("alt"); got != "sse" {
			t.Fatalf("alt = %q, want sse", got)
		}
		if got := r.URL.Query().Get("key"); got != "" {
			t.Fatalf("key = %q, want empty", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer vertex-token" {
			t.Fatalf("Authorization = %q, want bearer token", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if _, ok := body["contents"].([]any); !ok {
			t.Fatalf("contents = %#v, want array", body["contents"])
		}
		generationConfig, _ := body["generationConfig"].(map[string]any)
		thinkingConfig, _ := generationConfig["thinkingConfig"].(map[string]any)
		if thinkingConfig["thinkingBudget"] != float64(1024) {
			t.Fatalf("thinkingBudget = %#v, want 1024", thinkingConfig["thinkingBudget"])
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"ok\"}]}}]}\n\n"))
	}))
	defer server.Close()

	got, err := callGoogleVertex(
		context.Background(),
		domain.ProviderConfig{ID: "google-vertex", Type: string(TransportGoogleVertex), BaseURL: server.URL},
		domain.ModelRef{ProviderID: "google-vertex", ModelID: "gemini-2.5-pro"},
		llmCredential{APIKey: "vertex-token"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		nil,
		"low",
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "ok" {
		t.Fatalf("reply = %q, want ok", got.Text)
	}
}
