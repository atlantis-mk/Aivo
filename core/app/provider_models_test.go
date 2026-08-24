package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestRefreshProviderModelsUsesPersistedProviderConfigForIDOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("path = %q, want /v1/models", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer stored-key" {
			t.Fatalf("Authorization = %q, want stored credential", got)
		}
		if got := r.Header.Get("X-Team"); got != "aivo" {
			t.Fatalf("X-Team = %q, want persisted header", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"new-default","name":"New Default"},{"id":"new-secondary","name":"New Secondary"}]}`))
	}))
	defer server.Close()

	store := &memoryProviderStore{
		config: &domain.AppConfig{DefaultModel: &domain.ModelRef{
			ProviderID: "team-provider",
			ModelID:    "active-model",
		}},
		providers: []domain.ProviderConfig{{
			ID: "team-provider", Type: string(TransportOpenAICompatible), BaseURL: server.URL + "/v1",
			Model: "old-default", Headers: map[string]string{"X-Team": "aivo"},
		}},
		auth: map[string]domain.ProviderAuthRecord{
			"team-provider": {ProviderID: "team-provider", Method: "api-key", APIKey: "stored-key"},
		},
	}
	service := NewService(store)
	defer service.Shutdown()

	catalog, err := service.RefreshProviderModels(context.Background(), domain.ProviderConnectInput{
		ProviderID: "team-provider",
		Name:       "Team Provider",
	})
	if err != nil {
		t.Fatal(err)
	}
	var refreshed *domain.ProviderInfo
	for i := range catalog.Providers {
		if catalog.Providers[i].ID == "team-provider" {
			refreshed = &catalog.Providers[i]
			break
		}
	}
	if refreshed == nil {
		t.Fatal("refreshed provider missing from catalog")
	}
	if refreshed.DefaultModelID != "new-default" || len(refreshed.Models) != 2 || refreshed.Models[0].ID != "new-default" {
		t.Fatalf("refreshed provider = %+v, want replacement model list/default", refreshed)
	}
	if store.savedCache == nil || store.savedCache.DefaultModel != "new-default" || len(store.savedCache.Models) != 2 {
		t.Fatalf("saved cache = %+v, want refreshed models", store.savedCache)
	}
	if store.config == nil || store.config.DefaultModel == nil || store.config.DefaultModel.ModelID != "active-model" {
		t.Fatalf("app config = %+v, want active model preference preserved", store.config)
	}
}

func TestRefreshProviderModelsPreservesExistingCacheWhenEndpointIsUnsupported(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	existing := domain.ProviderModelCache{
		ProviderID: "catalog-provider",
		Models: []domain.ModelInfo{{
			ID: "known-model", ProviderID: "catalog-provider", Name: "Known Model",
		}},
		DefaultModel: "known-model",
		CacheSource:  "ecosystem",
	}
	store := &memoryProviderStore{
		providers: []domain.ProviderConfig{{
			ID: "catalog-provider", Type: string(TransportOpenAICompatible), BaseURL: server.URL,
			Model: "known-model",
		}},
		auth: map[string]domain.ProviderAuthRecord{
			"catalog-provider": {ProviderID: "catalog-provider", Method: "api-key", APIKey: "stored-key"},
		},
		modelCaches: map[string]domain.ProviderModelCache{"catalog-provider": existing},
	}
	service := NewService(store)
	defer service.Shutdown()

	_, err := service.RefreshProviderModels(context.Background(), domain.ProviderConnectInput{
		ProviderID: "catalog-provider",
		Name:       "Catalog Provider",
	})
	if err == nil || !strings.Contains(err.Error(), "model endpoint is not supported") {
		t.Fatalf("RefreshProviderModels error = %v, want unsupported endpoint", err)
	}
	if store.savedCache != nil {
		t.Fatalf("failed refresh saved cache = %+v, want no replacement", store.savedCache)
	}
	preserved := store.modelCaches["catalog-provider"]
	if preserved.DefaultModel != "known-model" || len(preserved.Models) != 1 || preserved.Models[0].ID != "known-model" {
		t.Fatalf("preserved cache = %+v, want previous catalog", preserved)
	}
}
