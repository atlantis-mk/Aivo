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
		if body["model"] != "mistral-medium-latest" {
			t.Fatalf("model = %#v", body["model"])
		}
		if stream, _ := body["stream"].(bool); !stream {
			t.Fatalf("stream = %#v, want true", body["stream"])
		}
		streamOptions, _ := body["stream_options"].(map[string]any)
		if streamOptions["include_usage"] != true {
			t.Fatalf("stream_options = %#v, want include_usage true", body["stream_options"])
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n"))
	}))
	defer server.Close()

	got, err := callOpenAICompatible(
		context.Background(),
		domain.ProviderConfig{ID: "mistral", Type: "mistral", BaseURL: server.URL},
		domain.ModelRef{ProviderID: "mistral", ModelID: "mistral-medium-latest"},
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

func TestCallOpenAICompatibleKeepsExplicitChatCompletionsEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/llm/v1/chat/completions" {
			t.Fatalf("path = %q, want explicit /chat/completions endpoint unchanged", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n"))
	}))
	defer server.Close()

	got, err := callOpenAICompatible(
		context.Background(),
		domain.ProviderConfig{ID: "bailing", Type: "bailing", BaseURL: server.URL + "/api/llm/v1/chat/completions"},
		domain.ModelRef{ProviderID: "bailing", ModelID: "Ring-1T"},
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

func TestCallOpenAICompatibleResponsesUsesCodexRequestControls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Fatalf("path = %q, want /responses", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		reasoning, _ := body["reasoning"].(map[string]any)
		if reasoning["effort"] != "max" || reasoning["summary"] != "auto" {
			t.Fatalf("reasoning = %#v", reasoning)
		}
		text, _ := body["text"].(map[string]any)
		if text["verbosity"] != "low" {
			t.Fatalf("text = %#v", text)
		}
		includes, _ := body["include"].([]any)
		if len(includes) != 2 || includes[0] != "web_search_call.action.sources" || includes[1] != "reasoning.encrypted_content" {
			t.Fatalf("include = %#v", body["include"])
		}
		if _, ok := body["reasoningEffort"]; ok {
			t.Fatalf("body retained reasoningEffort alias: %#v", body)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n"))
	}))
	defer server.Close()

	got, err := callOpenAICompatible(
		context.Background(),
		domain.ProviderConfig{
			ID: "openai", Type: "openai", BaseURL: server.URL,
			RequestParams: map[string]any{
				"reasoningEffort": "ultra",
				"textVerbosity":   "low",
				"include":         []any{"web_search_call.action.sources"},
			},
		},
		domain.ModelRef{ProviderID: "openai", ModelID: "gpt-5.5"},
		llmCredential{APIKey: "test-key"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		nil,
		"low",
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

func TestCallOpenAICompatibleUsesResponsesForPoeHostedWebSearchPreview(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Fatalf("path = %q, want /responses", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		tools, _ := body["tools"].([]any)
		if len(tools) != 1 {
			t.Fatalf("tools = %#v, want one hosted tool", body["tools"])
		}
		tool, _ := tools[0].(map[string]any)
		if tool["type"] != "web_search_preview" {
			t.Fatalf("tool = %#v, want Poe web_search_preview", tool)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n"))
	}))
	defer server.Close()

	got, err := callOpenAICompatible(
		context.Background(),
		domain.ProviderConfig{ID: "poe", Type: "poe", BaseURL: server.URL},
		domain.ModelRef{ProviderID: "poe", ModelID: "GPT-5.4"},
		llmCredential{APIKey: "poe-key"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		[]domain.ToolSpec{{Name: "web_search", Hosted: &domain.HostedToolSpec{Type: "web_search_preview"}}},
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

func TestProviderUsesDeclaredResponsesAPIByDefault(t *testing.T) {
	tools := []domain.ToolSpec{{Name: "web_search", Hosted: &domain.HostedToolSpec{Type: "web_search"}}}
	for _, providerID := range []string{"xai", "xiaomi", "deepseek", "openrouter", "groq", "ovhcloud", "perplexity-agent"} {
		t.Run(providerID, func(t *testing.T) {
			if !providerUsesResponsesAPI(domain.ProviderConfig{ID: providerID, Type: providerID}, nil) {
				t.Fatalf("%s should use the provider-declared Responses API by default", providerID)
			}
		})
	}
	if !providerUsesResponsesAPI(domain.ProviderConfig{ID: "deepseek", Type: "deepseek"}, tools) {
		t.Fatal("DeepSeek hosted web_search should use the provider-declared Responses API")
	}
	if providerUsesResponsesAPI(domain.ProviderConfig{ID: "alibaba", Type: "alibaba"}, nil) {
		t.Fatal("Alibaba should not use Responses API without hosted tools")
	}
	if !providerUsesResponsesAPI(domain.ProviderConfig{ID: "alibaba", Type: "alibaba"}, tools) {
		t.Fatal("Alibaba hosted web_search should use the provider-declared Responses API")
	}
	if !providerUsesResponsesAPI(domain.ProviderConfig{ID: "poe", Type: "poe"}, []domain.ToolSpec{{Name: "web_search", Hosted: &domain.HostedToolSpec{Type: "web_search_preview"}}}) {
		t.Fatal("Poe hosted web_search_preview should use the provider-declared Responses API")
	}
	if !providerUsesResponsesAPI(domain.ProviderConfig{ID: "custom-perplexity-agent", Type: "perplexity-agent"}, nil) {
		t.Fatal("custom provider config with Perplexity Agent type should use the provider-declared Responses API")
	}
	if got := providerResponsesBaseURL(domain.ProviderConfig{ID: "deepseek", Type: "deepseek"}, "https://api.deepseek.com/v1"); got != "https://api.deepseek.com" {
		t.Fatalf("responses base URL = %q, want DeepSeek declared Responses base URL", got)
	}
	if got := providerResponsesBaseURL(domain.ProviderConfig{ID: "deepseek", Type: "deepseek", BaseURL: "http://127.0.0.1:11434/v1"}, "http://127.0.0.1:11434/v1"); got != "http://127.0.0.1:11434/v1" {
		t.Fatalf("custom responses base URL = %q, want custom base preserved", got)
	}
}

func TestCallOpenAICompatibleUsesResponsesByDefault(t *testing.T) {
	for _, tt := range []struct {
		providerID string
		modelID    string
	}{
		{providerID: "deepseek", modelID: "deepseek-chat"},
		{providerID: "openrouter", modelID: "openai/gpt-5-codex"},
		{providerID: "groq", modelID: "openai/gpt-oss-120b"},
		{providerID: "xiaomi", modelID: "mimo-v2.5-pro"},
		{providerID: "ovhcloud", modelID: "gpt-oss-20b"},
		{providerID: "perplexity-agent", modelID: "openai/gpt-5.6-terra"},
	} {
		t.Run(tt.providerID, func(t *testing.T) {
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
				if store, _ := body["store"].(bool); store {
					t.Fatalf("store = %#v, want false", body["store"])
				}
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n"))
			}))
			defer server.Close()

			got, err := callOpenAICompatible(
				context.Background(),
				domain.ProviderConfig{ID: tt.providerID, Type: tt.providerID, BaseURL: server.URL},
				domain.ModelRef{ProviderID: tt.providerID, ModelID: tt.modelID},
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
		})
	}
}

func TestCallOpenAICompatibleSkipsOpenAISpecificResponsesDefaults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Fatalf("path = %q, want /responses", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if _, ok := body["include"]; ok {
			t.Fatalf("include = %#v, want no OpenAI-specific encrypted reasoning include", body["include"])
		}
		reasoning, _ := body["reasoning"].(map[string]any)
		if reasoning["effort"] != "high" {
			t.Fatalf("reasoning = %#v, want requested effort preserved", reasoning)
		}
		if _, ok := reasoning["summary"]; ok {
			t.Fatalf("reasoning = %#v, want no OpenAI-specific summary default", reasoning)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n"))
	}))
	defer server.Close()

	got, err := callOpenAICompatible(
		context.Background(),
		domain.ProviderConfig{ID: "xiaomi", Type: "xiaomi", BaseURL: server.URL},
		domain.ModelRef{ProviderID: "xiaomi", ModelID: "mimo-v2.5-pro"},
		llmCredential{APIKey: "test-key"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		nil,
		"high",
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

func TestCallOpenAICompatibleAppliesRequestyWebSearchTool(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %q, want /chat/completions", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		tools, _ := body["tools"].([]any)
		if len(tools) != 1 {
			t.Fatalf("tools = %#v, want Requesty hosted web_search tool", body["tools"])
		}
		tool, _ := tools[0].(map[string]any)
		if tool["type"] != "web_search" {
			t.Fatalf("tool = %#v, want Requesty web_search", tool)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n"))
	}))
	defer server.Close()

	got, err := callOpenAICompatible(
		context.Background(),
		domain.ProviderConfig{ID: "requesty", Type: "requesty", BaseURL: server.URL},
		domain.ModelRef{ProviderID: "requesty", ModelID: "anthropic/claude-sonnet-4-20250514"},
		llmCredential{APIKey: "requesty-key"},
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

func TestCallOpenAICompatibleAppliesVeniceSearchParameters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %q, want /chat/completions", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if _, ok := body["tools"]; ok {
			t.Fatalf("body tools = %#v, did not expect standard tools for Venice search", body["tools"])
		}
		params, _ := body["venice_parameters"].(map[string]any)
		if params["enable_web_search"] != "auto" {
			t.Fatalf("venice_parameters = %#v, want enable_web_search auto", body["venice_parameters"])
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n"))
	}))
	defer server.Close()

	got, err := callOpenAICompatible(
		context.Background(),
		domain.ProviderConfig{ID: "venice", Type: "venice", BaseURL: server.URL},
		domain.ModelRef{ProviderID: "venice", ModelID: "zai-org-glm-5"},
		llmCredential{APIKey: "venice-key"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		[]domain.ToolSpec{{Name: "web_search", Hosted: &domain.HostedToolSpec{Type: "venice_web_search"}}},
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

func TestCallOpenAICompatibleAppliesPerplexityAgentSearchFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Fatalf("path = %q, want /responses", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		tools, _ := body["tools"].([]any)
		if len(tools) != 1 {
			t.Fatalf("tools = %#v, want one Perplexity Agent hosted web_search tool", body["tools"])
		}
		tool, _ := tools[0].(map[string]any)
		if tool["type"] != "web_search" {
			t.Fatalf("tool = %#v, want Perplexity Agent web_search", tool)
		}
		filters, _ := tool["filters"].(map[string]any)
		domains, _ := filters["search_domain_filter"].([]any)
		if len(domains) != 1 || domains[0] != "example.com" {
			t.Fatalf("filters = %#v, want search_domain_filter example.com", filters)
		}
		if _, ok := filters["allowed_domains"]; ok {
			t.Fatalf("filters = %#v, want provider-specific filters without allowed_domains", filters)
		}
		if _, ok := body["include"]; ok {
			t.Fatalf("include = %#v, want no OpenAI-specific encrypted reasoning include", body["include"])
		}
		reasoning, _ := body["reasoning"].(map[string]any)
		if _, ok := reasoning["summary"]; ok {
			t.Fatalf("reasoning = %#v, want no OpenAI-specific summary default", reasoning)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n"))
	}))
	defer server.Close()

	got, err := callOpenAICompatible(
		context.Background(),
		domain.ProviderConfig{ID: "perplexity-agent", Type: "perplexity-agent", BaseURL: server.URL},
		domain.ModelRef{ProviderID: "perplexity-agent", ModelID: "openai/gpt-5.6-terra"},
		llmCredential{APIKey: "pplx-key"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		[]domain.ToolSpec{{Name: "web_search", Hosted: &domain.HostedToolSpec{Type: "web_search", AllowedDomains: []string{"example.com"}}}},
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

func TestCallOpenAICompatibleUsesProviderTypeForNativeSearchAssembly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Fatalf("path = %q, want /responses", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		tools, _ := body["tools"].([]any)
		if len(tools) != 1 {
			t.Fatalf("tools = %#v, want one Perplexity Agent hosted web_search tool", body["tools"])
		}
		tool, _ := tools[0].(map[string]any)
		filters, _ := tool["filters"].(map[string]any)
		if _, ok := filters["search_domain_filter"]; !ok {
			t.Fatalf("filters = %#v, want provider-specific search_domain_filter from provider type", filters)
		}
		if _, ok := filters["allowed_domains"]; ok {
			t.Fatalf("filters = %#v, want no generic allowed_domains after provider-specific assembly", filters)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n"))
	}))
	defer server.Close()

	got, err := callOpenAICompatible(
		context.Background(),
		domain.ProviderConfig{ID: "custom-perplexity-agent", Type: "perplexity-agent", BaseURL: server.URL},
		domain.ModelRef{ProviderID: "custom-perplexity-agent", ModelID: "openai/gpt-5.6-terra"},
		llmCredential{APIKey: "pplx-key"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		[]domain.ToolSpec{{Name: "web_search", Hosted: &domain.HostedToolSpec{Type: "web_search", AllowedDomains: []string{"example.com"}}}},
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
