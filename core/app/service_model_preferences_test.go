package app

import (
	"context"
	"testing"

	"aivo/core/domain"
)

func TestAppConfigHydratesActiveProviderFromSavedConfig(t *testing.T) {
	store := &memoryProviderStore{
		config: &domain.AppConfig{
			Provider:     &domain.ProviderConfig{ID: "deepseek", Type: string(TransportOpenAICompatible), Model: "deepseek-v4-flash"},
			DefaultModel: &domain.ModelRef{ProviderID: "deepseek", ModelID: "deepseek-v4-flash"},
		},
		providers: []domain.ProviderConfig{{
			ID: "deepseek", Type: string(TransportOpenAICompatible), BaseURL: "https://api.deepseek.com", APIKeyEnv: "DEEPSEEK_API_KEY", Model: "deepseek-chat",
		}},
	}
	service := NewService(store)
	defer service.Shutdown()

	cfg, err := service.AppConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider == nil || cfg.Provider.BaseURL != "https://api.deepseek.com" {
		t.Fatalf("provider = %#v, want saved DeepSeek base URL", cfg.Provider)
	}
	if cfg.Provider.APIKeyEnv != "DEEPSEEK_API_KEY" {
		t.Fatalf("provider APIKeyEnv = %q, want DEEPSEEK_API_KEY", cfg.Provider.APIKeyEnv)
	}
	if cfg.Provider.Model != "deepseek-v4-flash" {
		t.Fatalf("provider model = %q, want active model preserved", cfg.Provider.Model)
	}
}

func TestUpdateModelPreferencesUsesSavedProviderConfigWhenSwitchingProvider(t *testing.T) {
	store := &memoryProviderStore{
		config: &domain.AppConfig{
			Provider:     &domain.ProviderConfig{ID: "openai", Type: string(TransportOpenAIResponses), BaseURL: "https://api.openai.com/v1", Model: "gpt-5.5"},
			DefaultModel: &domain.ModelRef{ProviderID: "openai", ModelID: "gpt-5.5"},
		},
		providers: []domain.ProviderConfig{{
			ID: "deepseek", Type: string(TransportOpenAICompatible), BaseURL: "https://api.deepseek.com", APIKeyEnv: "DEEPSEEK_API_KEY", Model: "deepseek-chat",
		}},
	}
	service := NewService(store)
	defer service.Shutdown()

	cfg, err := service.UpdateModelPreferences(context.Background(), domain.ModelPreferencesInput{
		Model: &domain.ModelRef{ProviderID: "deepseek", ModelID: "deepseek-v4-flash"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider == nil || cfg.Provider.BaseURL != "https://api.deepseek.com" {
		t.Fatalf("provider = %#v, want saved DeepSeek configuration", cfg.Provider)
	}
	if cfg.Provider.Model != "deepseek-v4-flash" {
		t.Fatalf("provider model = %q, want selected model", cfg.Provider.Model)
	}
}
