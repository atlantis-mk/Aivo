package app

import (
	"context"
	"testing"

	"aivo/core/domain"
)

func TestSaveProviderPersistsCustomProviderWithoutChangingDefault(t *testing.T) {
	cfg := domain.AppConfig{
		Provider:     &domain.ProviderConfig{ID: "openai", Type: "openai", Model: "gpt-5.5"},
		DefaultModel: &domain.ModelRef{ProviderID: "openai", ModelID: "gpt-5.5"},
	}
	store := &memoryProviderStore{config: &cfg}
	service := NewService(store)

	catalog, err := service.SaveProvider(context.Background(), domain.ProviderConnectInput{
		ProviderID: "custom-api",
		Type:       string(TransportOpenAICompatible),
		BaseURL:    "http://127.0.0.1:1234/v1",
		ModelID:    "local-model",
		Method:     "api-key",
		APIKey:     "local-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if store.config.Provider == nil || store.config.Provider.ID != "openai" {
		t.Fatalf("active provider changed: %+v", store.config.Provider)
	}
	if store.savedAuth == nil || store.savedAuth.ProviderID != "custom-api" || store.savedAuth.APIKeyRef == "" || store.savedAuth.APIKey != "" {
		t.Fatalf("saved auth = %+v, want secret reference", store.savedAuth)
	}
	var found bool
	for _, provider := range catalog.Providers {
		if provider.ID == "custom-api" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("custom-api missing from catalog")
	}
}

func TestDeleteProviderClearsConfigReferencesAndSecrets(t *testing.T) {
	auth := domain.ProviderAuthRecord{
		ID:         "auth-1",
		ProviderID: "custom-api",
		Method:     "api-key",
		APIKeyRef:  "provider-auth/custom-api/api-key/default/api-key",
	}
	secrets := NewMemorySecretStore()
	if err := secrets.Put(context.Background(), auth.APIKeyRef, "secret"); err != nil {
		t.Fatal(err)
	}
	cfg := domain.AppConfig{
		Provider:       &domain.ProviderConfig{ID: "custom-api", Type: string(TransportOpenAICompatible), Model: "local-model"},
		DefaultModel:   &domain.ModelRef{ProviderID: "custom-api", ModelID: "local-model"},
		FallbackModels: []domain.ModelRef{{ProviderID: "custom-api", ModelID: "other"}, {ProviderID: "openai", ModelID: "gpt-5.5"}},
	}
	store := &memoryProviderStore{
		config:    &cfg,
		providers: []domain.ProviderConfig{{ID: "custom-api", Type: string(TransportOpenAICompatible), Model: "local-model"}},
		auth:      map[string]domain.ProviderAuthRecord{"custom-api": auth},
		authByID:  map[string]domain.ProviderAuthRecord{"auth-1": auth},
	}
	service := NewService(store)
	service.SetSecretStore(secrets)

	if _, err := service.DeleteProvider(context.Background(), "custom-api"); err != nil {
		t.Fatal(err)
	}
	if store.config.Provider != nil || store.config.DefaultModel != nil {
		t.Fatalf("config = %+v, want active provider/default cleared", store.config)
	}
	if len(store.config.FallbackModels) != 1 || store.config.FallbackModels[0].ProviderID != "openai" {
		t.Fatalf("fallback models = %+v, want custom-api removed", store.config.FallbackModels)
	}
	value, err := secrets.Get(context.Background(), auth.APIKeyRef)
	if err != nil {
		t.Fatal(err)
	}
	if value != "" {
		t.Fatalf("secret value = %q, want deleted", value)
	}
}
