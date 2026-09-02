package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestCodexSearchRequestBodyPreservesIndexedSettings(t *testing.T) {
	body := codexSearchRequestBody("session", "gpt-test", domain.WebSearchConfig{
		Mode: domain.WebSearchModeIndexed, SearchContextSize: "high", AllowedDomains: []string{"example.com"},
	}, "recent news", 9, 7, []string{"openai.com"})
	settings, _ := body["settings"].(map[string]any)
	if settings["external_web_access"] != "indexed" || settings["search_context_size"] != "high" {
		t.Fatalf("settings = %#v", settings)
	}
	commands, _ := body["commands"].(map[string]any)
	if commands["response_length"] != "long" {
		t.Fatalf("commands = %#v", commands)
	}
}

func TestCodexWebSearchToolUsesAuthenticatedProviderEndpoint(t *testing.T) {
	originalClient := codexSearchHTTPClient
	defer func() { codexSearchHTTPClient = originalClient }()
	codexSearchHTTPClient = &http.Client{Transport: providerModelRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != chatGPTCodexSearchURL || req.Header.Get("Authorization") != "Bearer oauth-token" || req.Header.Get("ChatGPT-Account-Id") != "acct-1" {
			t.Fatalf("request = %s headers=%#v", req.URL, req.Header)
		}
		var body map[string]any
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "gpt-codex" {
			t.Fatalf("body = %#v", body)
		}
		response := `{"output":"Provider answer with turn0search0","results":[{"type":"text_result","ref_id":"turn0search0","url":"https://example.com/news","title":"News"}]}`
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response)), Request: req}, nil
	})}
	lite := true
	store := &memoryProviderStore{
		config: &domain.AppConfig{
			Provider:     &domain.ProviderConfig{ID: "openai", Type: string(TransportOpenAIResponses), Model: "gpt-codex"},
			DefaultModel: &domain.ModelRef{ProviderID: "openai", ModelID: "gpt-codex"},
			WebSearch:    domain.WebSearchConfig{Mode: domain.WebSearchModeLive, Route: domain.WebSearchRouteAuto},
		},
		auth: map[string]domain.ProviderAuthRecord{"openai": {
			ProviderID: "openai", Method: "oauth-browser", AccessToken: "oauth-token", ExpiresAt: "2099-01-01T00:00:00Z", AccountID: "acct-1",
		}},
		modelCaches: map[string]domain.ProviderModelCache{"openai": {ProviderID: "openai", Models: []domain.ModelInfo{{
			ID: "gpt-codex", ProviderID: "openai", UseResponsesLite: &lite,
			DeclaredCapabilities: []string{codexWebSearchCapability}, Capabilities: []string{codexWebSearchCapability},
		}}}},
	}
	service := NewService(store)
	defer service.Shutdown()
	model := domain.ModelRef{ProviderID: "openai", ModelID: "gpt-codex"}
	result := NewCodexWebSearchTool(service).Execute(context.Background(), json.RawMessage(`{"query":"recent news","limit":5}`), domain.ToolExecutionContext{
		SessionID: "session", ToolCallID: "call-1", ActiveModel: &model,
	})
	if !result.OK || result.CallID != "call-1" || !strings.Contains(result.ModelContent, "Provider answer") {
		t.Fatalf("result = %+v", result)
	}
	results, _ := result.Structured["results"].([]map[string]any)
	if len(results) != 1 || results[0]["url"] != "https://example.com/news" {
		t.Fatalf("structured results = %#v", result.Structured["results"])
	}
	store.config.WebSearch.Mode = domain.WebSearchModeDisabled
	disabled := NewCodexWebSearchTool(service).Execute(context.Background(), json.RawMessage(`{"query":"recent news"}`), domain.ToolExecutionContext{
		SessionID: "session", ToolCallID: "call-2", ActiveModel: &model,
	})
	if disabled.OK || !strings.Contains(disabled.Error, "disabled") {
		t.Fatalf("disabled result = %+v", disabled)
	}
}

func TestCodexWebSearchToolFallsBackToParallelForUnsupportedProvider(t *testing.T) {
	originalClient := parallelSearchHTTPClient
	defer func() { parallelSearchHTTPClient = originalClient }()
	parallelSearchHTTPClient = &http.Client{Transport: providerModelRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != defaultParallelSearchMCPURL {
			t.Fatalf("request URL = %s, want Parallel MCP", req.URL.String())
		}
		var body map[string]any
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["method"] != "tools/call" {
			t.Fatalf("body = %#v, want MCP tools/call", body)
		}
		response, err := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      "aivo-web-search",
			"result": map[string]any{"content": []map[string]any{{
				"type": "text",
				"text": `{"results":[{"title":"Parallel Result","url":"https://example.com/parallel","snippet":"Fresh result"}]}`,
			}}},
		})
		if err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(response))), Request: req}, nil
	})}
	store := &memoryProviderStore{
		config: &domain.AppConfig{
			Provider:     &domain.ProviderConfig{ID: "anthropic", Type: string(TransportAnthropicMessages), Model: "claude-test"},
			DefaultModel: &domain.ModelRef{ProviderID: "anthropic", ModelID: "claude-test"},
			WebSearch:    domain.WebSearchConfig{Mode: domain.WebSearchModeLive, Route: domain.WebSearchRouteAuto},
		},
	}
	service := NewService(store)
	defer service.Shutdown()
	model := domain.ModelRef{ProviderID: "anthropic", ModelID: "claude-test"}
	result := NewCodexWebSearchTool(service).Execute(context.Background(), json.RawMessage(`{"query":"recent news","limit":3}`), domain.ToolExecutionContext{
		SessionID: "session", ToolCallID: "call-parallel", ActiveModel: &model,
	})
	if !result.OK || result.Structured["provider"] != "parallel" || !strings.Contains(result.ModelContent, "Parallel Result") {
		t.Fatalf("result = %+v", result)
	}
}
