package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"aivo/core/domain"
)

func TestGenerateChatResponseFallsBackToConfiguredModel(t *testing.T) {
	primaryHits := 0
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryHits++
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
	}))
	defer primary.Close()
	fallbackHits := 0
	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackHits++
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"fallback ok\"}}]}\n\n"))
	}))
	defer fallback.Close()

	cfg := domain.AppConfig{
		Provider:       &domain.ProviderConfig{ID: "primary-test", Type: string(TransportOpenAICompatible), BaseURL: primary.URL, Model: "primary-model"},
		DefaultModel:   &domain.ModelRef{ProviderID: "primary-test", ModelID: "primary-model"},
		FallbackModels: []domain.ModelRef{{ProviderID: "fallback-test", ModelID: "fallback-model"}},
	}
	store := &memoryProviderStore{config: &cfg}
	service := NewService(store)
	registerNoAuthProvider(t, service, "primary-test", primary.URL, "primary-model")
	registerNoAuthProvider(t, service, "fallback-test", fallback.URL, "fallback-model")

	resp, activeModel, err := service.GenerateChatResponseStream(context.Background(), domain.ChatRequest{
		Messages: []domain.ChatMessage{{Role: "user", Text: "hello"}},
	}, nil, "none", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "fallback ok" {
		t.Fatalf("response = %q, want fallback ok", resp.Text)
	}
	if activeModel == nil || activeModel.ProviderID != "fallback-test" || activeModel.ModelID != "fallback-model" {
		t.Fatalf("activeModel = %+v, want fallback-test/fallback-model", activeModel)
	}
	if primaryHits != 2 {
		t.Fatalf("primaryHits = %d, want retry once before fallback", primaryHits)
	}
	if fallbackHits != 1 {
		t.Fatalf("fallbackHits = %d, want 1", fallbackHits)
	}
	if len(store.callEvents) != 3 {
		t.Fatalf("callEvents = %+v, want 3 attempts", store.callEvents)
	}
	last := store.callEvents[len(store.callEvents)-1]
	if last.ProviderID != "fallback-test" || last.Status != "success" || last.FallbackIndex != 1 {
		t.Fatalf("last event = %+v, want fallback success", last)
	}
}

func TestGenerateChatResponseFallsBackForStreamingBeforeOutput(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
	}))
	defer primary.Close()
	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"stream fallback\"}}]}\n\n"))
	}))
	defer fallback.Close()

	cfg := domain.AppConfig{
		Provider:       &domain.ProviderConfig{ID: "primary-stream", Type: string(TransportOpenAICompatible), BaseURL: primary.URL, Model: "primary-model"},
		DefaultModel:   &domain.ModelRef{ProviderID: "primary-stream", ModelID: "primary-model"},
		FallbackModels: []domain.ModelRef{{ProviderID: "fallback-stream", ModelID: "fallback-model"}},
	}
	store := &memoryProviderStore{config: &cfg}
	service := NewService(store)
	registerNoAuthProvider(t, service, "primary-stream", primary.URL, "primary-model")
	registerNoAuthProvider(t, service, "fallback-stream", fallback.URL, "fallback-model")

	var deltas []string
	resp, activeModel, err := service.GenerateChatResponseStream(context.Background(), domain.ChatRequest{
		Messages: []domain.ChatMessage{{Role: "user", Text: "hello"}},
	}, nil, "none", "", func(delta string) {
		deltas = append(deltas, delta)
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "stream fallback" {
		t.Fatalf("response = %q, want stream fallback", resp.Text)
	}
	if activeModel == nil || activeModel.ProviderID != "fallback-stream" {
		t.Fatalf("activeModel = %+v, want fallback-stream", activeModel)
	}
	if len(deltas) != 1 || deltas[0] != "stream fallback" {
		t.Fatalf("deltas = %+v, want only fallback delta", deltas)
	}
}

func TestGenerateChatResponseHonorsFallbackDisabledPolicy(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
	}))
	defer primary.Close()
	fallbackHits := 0
	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackHits++
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"fallback\"}}]}\n\n"))
	}))
	defer fallback.Close()
	enableFallback := false
	cfg := domain.AppConfig{
		Provider:       &domain.ProviderConfig{ID: "primary-disabled", Type: string(TransportOpenAICompatible), BaseURL: primary.URL, Model: "primary-model"},
		DefaultModel:   &domain.ModelRef{ProviderID: "primary-disabled", ModelID: "primary-model"},
		FallbackModels: []domain.ModelRef{{ProviderID: "fallback-disabled", ModelID: "fallback-model"}},
		ProviderPolicy: domain.ProviderRuntimePolicy{EnableFallback: &enableFallback, MaxRetries: 0},
	}
	store := &memoryProviderStore{config: &cfg}
	service := NewService(store)
	registerNoAuthProvider(t, service, "primary-disabled", primary.URL, "primary-model")
	registerNoAuthProvider(t, service, "fallback-disabled", fallback.URL, "fallback-model")

	_, _, err := service.GenerateChatResponseStream(context.Background(), domain.ChatRequest{
		Messages: []domain.ChatMessage{{Role: "user", Text: "hello"}},
	}, nil, "", "", nil)
	if err == nil {
		t.Fatal("expected primary error")
	}
	if fallbackHits != 0 {
		t.Fatalf("fallbackHits = %d, want fallback disabled", fallbackHits)
	}
}

func TestUpdateModelPreferencesPersistsFallbackModels(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	cfg, err := service.UpdateModelPreferences(context.Background(), domain.ModelPreferencesInput{
		Model: &domain.ModelRef{ProviderID: "openai", ModelID: "gpt-5.5"},
		FallbackModels: []domain.ModelRef{
			{ProviderID: "openai", ModelID: "gpt-5.5"},
			{ProviderID: "anthropic", ModelID: "claude-sonnet-4"},
			{ProviderID: "claude", ModelID: "claude-sonnet-4"},
			{ProviderID: "", ModelID: "missing-provider"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.FallbackModels) != 1 {
		t.Fatalf("fallback models = %+v, want 1 normalized fallback", cfg.FallbackModels)
	}
	if cfg.FallbackModels[0].ProviderID != "anthropic" || cfg.FallbackModels[0].ModelID != "claude-sonnet-4" {
		t.Fatalf("fallback models = %+v, want anthropic claude-sonnet-4", cfg.FallbackModels)
	}
}

func TestUpdateModelPreferencesPersistsProviderPolicy(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	enableFallback := false
	bufferStreaming := false
	cfg, err := service.UpdateModelPreferences(context.Background(), domain.ModelPreferencesInput{
		ProviderPolicy: &domain.ProviderRuntimePolicy{
			EnableFallback:           &enableFallback,
			BufferStreamingFallback:  &bufferStreaming,
			MaxRetries:               0,
			RetryBaseDelayMs:         250,
			RateLimitCooldownSeconds: 45,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ProviderPolicy.EnableFallback == nil || *cfg.ProviderPolicy.EnableFallback {
		t.Fatalf("ProviderPolicy = %+v, want fallback disabled", cfg.ProviderPolicy)
	}
	if cfg.ProviderPolicy.BufferStreamingFallback == nil || *cfg.ProviderPolicy.BufferStreamingFallback {
		t.Fatalf("ProviderPolicy = %+v, want streaming buffer disabled", cfg.ProviderPolicy)
	}
	if cfg.ProviderPolicy.MaxRetries != 0 || cfg.ProviderPolicy.RetryBaseDelayMs != 250 || cfg.ProviderPolicy.RateLimitCooldownSeconds != 45 {
		t.Fatalf("ProviderPolicy = %+v, want configured retry/cooldown", cfg.ProviderPolicy)
	}
}

func TestUpdateModelPreferencesPersistsNativeTools(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	nativeTools := domain.NativeToolsConfig{
		XSearch: domain.NativeToolToggle{Enabled: true},
		CodeExecution: domain.NativeCodeExecutionConfig{
			Enabled: true,
			FileIDs: []string{" file_1 ", "file_1", "file_2"},
		},
		FileSearch: domain.NativeFileSearchConfig{
			Enabled:        true,
			VectorStoreIDs: []string{" vs_1 ", "vs_1"},
		},
		RemoteMCP: []domain.NativeMCPToolConfig{
			{
				Enabled:      true,
				ServerURL:    " https://mcp.example.com ",
				ServerLabel:  " Docs ",
				AllowedTools: []string{" search ", "search"},
			},
			{Enabled: true, ServerLabel: "missing-url"},
		},
	}
	cfg, err := service.UpdateModelPreferences(context.Background(), domain.ModelPreferencesInput{
		NativeTools: &nativeTools,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.NativeTools.XSearch.Enabled || !cfg.NativeTools.CodeExecution.Enabled {
		t.Fatalf("NativeTools = %+v, want x search and code execution enabled", cfg.NativeTools)
	}
	if len(cfg.NativeTools.CodeExecution.FileIDs) != 2 || cfg.NativeTools.CodeExecution.FileIDs[0] != "file_1" || cfg.NativeTools.CodeExecution.FileIDs[1] != "file_2" {
		t.Fatalf("CodeExecution = %+v, want trimmed unique file ids", cfg.NativeTools.CodeExecution)
	}
	if len(cfg.NativeTools.FileSearch.VectorStoreIDs) != 1 || cfg.NativeTools.FileSearch.VectorStoreIDs[0] != "vs_1" {
		t.Fatalf("FileSearch = %+v, want trimmed unique vector store ids", cfg.NativeTools.FileSearch)
	}
	if len(cfg.NativeTools.RemoteMCP) != 1 {
		t.Fatalf("RemoteMCP = %+v, want one valid server", cfg.NativeTools.RemoteMCP)
	}
	server := cfg.NativeTools.RemoteMCP[0]
	if server.ServerURL != "https://mcp.example.com" || server.ServerLabel != "Docs" || len(server.AllowedTools) != 1 || server.AllowedTools[0] != "search" {
		t.Fatalf("RemoteMCP server = %+v, want normalized server config", server)
	}
}

func registerNoAuthProvider(t *testing.T, service *Service, id string, baseURL string, modelID string) {
	t.Helper()
	if err := service.RegisterProviderDefinition(ProviderDefinition{
		ID:              id,
		DisplayName:     id,
		Transport:       TransportOpenAICompatible,
		DefaultBaseURL:  baseURL,
		DefaultModelID:  modelID,
		AuthTypes:       []AuthType{AuthNone},
		DefaultAuthType: AuthNone,
		Models: []domain.ModelInfo{{
			ID:         modelID,
			ProviderID: id,
			Name:       modelID,
			Streaming:  true,
		}},
	}); err != nil {
		t.Fatal(err)
	}
}
