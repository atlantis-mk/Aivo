package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"aivo/core/domain"
)

func TestParseCodexModelsOrdersLatestOpenAIModelFirst(t *testing.T) {
	raw := []byte(`{
		"models": [
			{"slug": "gpt-5.4", "display_name": "GPT-5.4", "visibility": "list", "priority": 1},
			{"slug": "gpt-5.5", "display_name": "GPT-5.5", "visibility": "list", "priority": 2},
			{"slug": "gpt-5.3-codex-spark", "display_name": "GPT-5.3-Codex-Spark", "visibility": "list", "priority": 3}
		]
	}`)

	models, defaultModel, err := parseCodexModels(raw, "openai")
	if err != nil {
		t.Fatalf("parseCodexModels returned error: %v", err)
	}
	if defaultModel != "gpt-5.5" {
		t.Fatalf("defaultModel = %q, want %q", defaultModel, "gpt-5.5")
	}
	if got := models[0].ID; got != "gpt-5.5" {
		t.Fatalf("models[0].ID = %q, want %q", got, "gpt-5.5")
	}
}

func TestFetchAnthropicModelsUsesAnthropicAuthHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Fatalf("path = %q, want /models", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "anthropic-key" {
			t.Fatalf("x-api-key = %q, want anthropic-key", got)
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %q, want empty", got)
		}
		if got := r.Header.Get("anthropic-version"); got == "" {
			t.Fatal("anthropic-version header is empty")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"claude-sonnet-4-5","display_name":"Claude Sonnet 4.5"}]}`))
	}))
	defer server.Close()

	models, defaultModel, err := fetchAnthropicModels(
		context.Background(),
		domain.ProviderConfig{ID: "anthropic", Type: "anthropic", BaseURL: server.URL},
		llmCredential{APIKey: "anthropic-key"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if defaultModel != "claude-sonnet-4-5" {
		t.Fatalf("defaultModel = %q, want claude-sonnet-4-5", defaultModel)
	}
	if len(models) != 1 || models[0].ID != "claude-sonnet-4-5" || !models[0].Recommended {
		t.Fatalf("models = %+v", models)
	}
}

func TestFetchOpenAICompatibleModelsUsesAzureAPIKeyHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/openai/v1/models" {
			t.Fatalf("path = %q, want /openai/v1/models", r.URL.Path)
		}
		if got := r.Header.Get("api-key"); got != "azure-key" {
			t.Fatalf("api-key = %q, want azure-key", got)
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %q, want empty", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-5.5","name":"GPT-5.5"}]}`))
	}))
	defer server.Close()

	models, defaultModel, err := fetchOpenAICompatibleModels(
		context.Background(),
		domain.ProviderConfig{ID: "azure-openai", Type: string(TransportAzureOpenAI), BaseURL: server.URL + "/openai/v1"},
		llmCredential{APIKey: "azure-key"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if defaultModel != "gpt-5.5" {
		t.Fatalf("defaultModel = %q, want gpt-5.5", defaultModel)
	}
	if len(models) != 1 || models[0].ID != "gpt-5.5" || !models[0].Recommended {
		t.Fatalf("models = %+v", models)
	}
}
