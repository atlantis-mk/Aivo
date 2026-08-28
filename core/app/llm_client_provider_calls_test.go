package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestCallOpenAICompatibleUsesChatCompletionsStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %q, want /chat/completions", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want bearer key", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "openai/gpt-5" {
			t.Fatalf("model = %#v", body["model"])
		}
		if stream, _ := body["stream"].(bool); !stream {
			t.Fatalf("stream = %#v, want true", body["stream"])
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n"))
	}))
	defer server.Close()

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

func TestCallChatGPTCodexUsesDeclaredResponsesLiteHeaderAndBody(t *testing.T) {
	originalClient := http.DefaultClient
	defer func() { http.DefaultClient = originalClient }()
	http.DefaultClient = &http.Client{Transport: providerModelRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != chatGPTCodexResponsesURL || req.Header.Get("x-openai-internal-codex-responses-lite") != "true" {
			t.Fatalf("request = %s headers=%#v", req.URL, req.Header)
		}
		var body map[string]any
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if _, ok := body["tools"]; ok || body["parallel_tool_calls"] != false {
			t.Fatalf("body = %#v", body)
		}
		input, _ := body["input"].([]any)
		if len(input) == 0 || input[0].(map[string]any)["type"] != "additional_tools" {
			t.Fatalf("input = %#v", body["input"])
		}
		response := "data: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\ndata: [DONE]\n\n"
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(response)), Request: req}, nil
	})}
	lite := true
	parallel := true
	service := NewService(&memoryProviderStore{})
	defer service.Shutdown()
	response, err := service.callChatGPTCodex(
		context.Background(), domain.ProviderConfig{ID: "openai"}, domain.ModelRef{ProviderID: "openai", ModelID: "gpt-lite"},
		domain.ModelInfo{UseResponsesLite: &lite, SupportsParallelToolCalls: &parallel},
		llmCredential{Method: "oauth-browser", AccessToken: "oauth-token", ExpiresAt: "2099-01-01T00:00:00Z"}, domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "user", Text: "hello"}}, []domain.ToolSpec{{Name: "read", InputSchema: map[string]any{"type": "object"}}},
		"medium", "default", nil, nil,
	)
	if err != nil || response.Text != "ok" {
		t.Fatalf("response = %+v err=%v", response, err)
	}
}

func TestCallOpenAICompatibleAppliesRequestProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Profile"); got != "default" {
			t.Fatalf("X-Profile = %q, want default", got)
		}
		if got := r.Header.Get("X-Model"); got != "matched" {
			t.Fatalf("X-Model = %q, want matched", got)
		}
		if got := r.Header.Get("X-Team"); got != "custom" {
			t.Fatalf("X-Team = %q, want custom", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want bearer key", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "openai/gpt-5" {
			t.Fatalf("model = %#v, want generated model", body["model"])
		}
		if body["reasoning"].(map[string]any)["effort"] != "high" {
			t.Fatalf("reasoning = %#v, want high override", body["reasoning"])
		}
		if body["temperature"] != float64(0.2) {
			t.Fatalf("temperature = %#v, want 0.2", body["temperature"])
		}
		if body["service_tier"] != "priority" {
			t.Fatalf("service_tier = %#v, want priority", body["service_tier"])
		}
		provider, _ := body["provider"].(map[string]any)
		if provider["sort"] != "throughput" || provider["allow_fallbacks"] != false {
			t.Fatalf("provider = %#v, want merged provider params", provider)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n"))
	}))
	defer server.Close()

	profile := domain.ProviderRequestProfile{
		Headers: map[string]string{"X-Profile": "default", "Authorization": "Bearer ignored"},
		Params: map[string]any{
			"temperature": 0.1,
			"provider":    map[string]any{"sort": "latency", "allow_fallbacks": true},
			"model":       "ignored",
		},
		ModelOverrides: map[string]domain.ProviderRequestOverride{
			"openai/gpt": {
				Headers: map[string]string{"X-Model": "matched"},
				Params:  map[string]any{"reasoning": map[string]any{"effort": "medium"}},
			},
		},
	}
	provider := domain.ProviderConfig{
		ID:      "openrouter",
		Type:    "openrouter",
		BaseURL: server.URL,
		Headers: map[string]string{"X-Team": "custom"},
		RequestParams: map[string]any{
			"temperature":  0.2,
			"service_tier": "priority",
			"reasoning":    map[string]any{"effort": "high"},
			"provider":     map[string]any{"sort": "throughput", "allow_fallbacks": false},
		},
	}
	got, err := callOpenAICompatible(
		context.Background(),
		provider,
		domain.ModelRef{ProviderID: "openrouter", ModelID: "openai/gpt-5"},
		llmCredential{APIKey: "test-key"},
		profile,
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		nil,
		"",
		"",
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

func TestCallAzureOpenAIUsesResponsesEndpointAndAPIKeyHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/openai/v1/responses" {
			t.Fatalf("path = %q, want /openai/v1/responses", r.URL.Path)
		}
		if got := r.Header.Get("api-key"); got != "azure-key" {
			t.Fatalf("api-key = %q, want azure-key", got)
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %q, want empty", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "gpt-5.5" {
			t.Fatalf("model = %#v, want deployment name", body["model"])
		}
		if _, ok := body["input"]; !ok {
			t.Fatalf("body = %#v, want responses input", body)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n"))
	}))
	defer server.Close()

	got, err := callAzureOpenAI(
		context.Background(),
		domain.ProviderConfig{ID: "azure-openai", Type: string(TransportAzureOpenAI), BaseURL: server.URL + "/openai/v1"},
		domain.ModelRef{ProviderID: "azure-openai", ModelID: "gpt-5.5"},
		llmCredential{APIKey: "azure-key"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		nil,
		"",
		"",
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

func TestCallOpenAICompatibleUsesResponsesForXAIHostedWebSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Fatalf("path = %q, want /responses", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if _, ok := body["input"]; !ok {
			t.Fatalf("body = %#v, want responses input", body)
		}
		tools, _ := body["tools"].([]any)
		if len(tools) != 1 {
			t.Fatalf("tools = %#v, want one hosted tool", body["tools"])
		}
		tool, _ := tools[0].(map[string]any)
		if tool["type"] != "web_search" {
			t.Fatalf("tool = %#v, want xAI web_search", tool)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n"))
	}))
	defer server.Close()

	got, err := callOpenAICompatible(
		context.Background(),
		domain.ProviderConfig{ID: "xai", Type: "xai", BaseURL: server.URL},
		domain.ModelRef{ProviderID: "xai", ModelID: "grok-4.3"},
		llmCredential{APIKey: "xai-key"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		[]domain.ToolSpec{{Name: "web_search", Hosted: &domain.HostedToolSpec{Type: "web_search"}}},
		"",
		"",
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

func TestCallOpenAICompatibleAppliesPerplexitySearchOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %q, want /chat/completions", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if _, ok := body["tools"]; ok {
			t.Fatalf("body tools = %#v, did not expect local function tools", body["tools"])
		}
		domains, _ := body["search_domain_filter"].([]any)
		if len(domains) != 1 || domains[0] != "example.com" {
			t.Fatalf("search_domain_filter = %#v", body["search_domain_filter"])
		}
		options, _ := body["web_search_options"].(map[string]any)
		if options["search_context_size"] != "high" {
			t.Fatalf("web_search_options = %#v", body["web_search_options"])
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n"))
	}))
	defer server.Close()

	got, err := callOpenAICompatible(
		context.Background(),
		domain.ProviderConfig{ID: "perplexity", Type: "perplexity", BaseURL: server.URL},
		domain.ModelRef{ProviderID: "perplexity", ModelID: "sonar-pro"},
		llmCredential{APIKey: "pplx-key"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		[]domain.ToolSpec{{Name: "web_search", Hosted: &domain.HostedToolSpec{Type: "perplexity_search", SearchContextSize: "high", AllowedDomains: []string{"example.com"}}}},
		"",
		"",
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
