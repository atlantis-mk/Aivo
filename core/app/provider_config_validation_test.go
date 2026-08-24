package app

import (
	"context"
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestProviderConfigRejectsInvalidBaseURL(t *testing.T) {
	_, _, err := providerConfigFromInput(domain.ProviderConnectInput{
		ProviderID: "custom-api",
		BaseURL:    "not a url",
		ModelID:    "model",
	})
	if err == nil || !strings.Contains(err.Error(), "absolute URL") {
		t.Fatalf("err = %v, want absolute URL error", err)
	}
}

func TestProviderConfigRejectsRemotePlainHTTP(t *testing.T) {
	_, _, err := providerConfigFromInput(domain.ProviderConnectInput{
		ProviderID: "custom-api",
		BaseURL:    "http://api.example.com/v1",
		ModelID:    "model",
	})
	if err == nil || !strings.Contains(err.Error(), "only allowed for local") {
		t.Fatalf("err = %v, want remote http rejection", err)
	}
}

func TestProviderConfigAllowsLocalPlainHTTP(t *testing.T) {
	cfg, _, err := providerConfigFromInput(domain.ProviderConnectInput{
		ProviderID: "custom-api",
		BaseURL:    "http://127.0.0.1:1234/v1",
		ModelID:    "model",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL != "http://127.0.0.1:1234/v1" {
		t.Fatalf("BaseURL = %q", cfg.BaseURL)
	}
}

func TestProviderConfigRejectsCredentialHeaders(t *testing.T) {
	_, _, err := providerConfigFromInput(domain.ProviderConnectInput{
		ProviderID: "custom-api",
		BaseURL:    "https://proxy.example.com/v1",
		ModelID:    "model",
		Headers:    map[string]string{"Authorization": "Bearer secret"},
	})
	if err == nil || !strings.Contains(err.Error(), "must be configured as a credential") {
		t.Fatalf("err = %v, want credential header rejection", err)
	}
}

func TestProviderConfigRejectsInvalidEnvName(t *testing.T) {
	_, _, err := providerConfigFromInput(domain.ProviderConnectInput{
		ProviderID: "custom-api",
		BaseURL:    "https://proxy.example.com/v1",
		ModelID:    "model",
		APIKeyEnv:  "bad-env-name",
	})
	if err == nil || !strings.Contains(err.Error(), "invalid credential environment variable") {
		t.Fatalf("err = %v, want invalid env rejection", err)
	}
}

func TestConnectProviderRejectsUnsupportedAuthMethodBeforePersisting(t *testing.T) {
	store := &memoryProviderStore{}
	service := NewService(store)

	_, err := service.ConnectProvider(context.Background(), domain.ProviderConnectInput{
		ProviderID: "anthropic",
		BaseURL:    "https://api.anthropic.com/v1",
		ModelID:    "claude-sonnet-4",
		Method:     "oauth-browser",
	})
	if err == nil || !strings.Contains(err.Error(), "does not support auth method") {
		t.Fatalf("err = %v, want unsupported auth method", err)
	}
	if store.savedAuth != nil {
		t.Fatalf("savedAuth = %+v, want nil", store.savedAuth)
	}
}

func TestValidateProviderRejectsURLCredentialBeforeNetwork(t *testing.T) {
	service := NewService(&memoryProviderStore{})

	result, err := service.ValidateProvider(context.Background(), domain.ProviderConnectInput{
		ProviderID: "custom-api",
		BaseURL:    "https://proxy.example.com/v1?key=secret",
		ModelID:    "model",
	})
	if err == nil {
		t.Fatalf("result = %+v, want validation error before result", result)
	}
	if !strings.Contains(err.Error(), "must not contain credentials") {
		t.Fatalf("err = %v, want credential URL rejection", err)
	}
}
