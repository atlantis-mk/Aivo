package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestCheckProviderIntegrationReadyForSavedProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Fatalf("path = %q, want /models", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"local-model","name":"Local Model"}]}`))
	}))
	defer server.Close()
	store := &memoryProviderStore{providers: []domain.ProviderConfig{{
		ID:      "check-provider",
		Type:    string(TransportOpenAICompatible),
		BaseURL: server.URL,
		Model:   "local-model",
	}}}
	service := NewService(store)
	registerNoAuthProvider(t, service, "check-provider", server.URL, "local-model")

	result, err := service.CheckProviderIntegration(context.Background(), domain.ProviderIntegrationCheckInput{ProviderID: "check-provider"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready || result.Status != "ready" {
		t.Fatalf("result = %+v, want ready", result)
	}
	if result.ProviderID != "check-provider" || result.ModelID != "local-model" || result.ModelCount != 1 {
		t.Fatalf("result = %+v, want check-provider local-model count=1", result)
	}
	if result.Validation == nil || result.Validation.Models != nil {
		t.Fatalf("validation = %+v, want compact validation without models", result.Validation)
	}
	if !hasIntegrationStep(result.Steps, "config", "ok") || !hasIntegrationStep(result.Steps, "auth", "ok") || !hasIntegrationStep(result.Steps, "runtime-route", "ok") {
		t.Fatalf("steps = %+v, want config/auth/runtime ok", result.Steps)
	}
}

func TestCheckProviderIntegrationReportsAuthFailure(t *testing.T) {
	service := NewService(&memoryProviderStore{})

	result, err := service.CheckProviderIntegration(context.Background(), domain.ProviderIntegrationCheckInput{ProviderID: "openai", ModelID: "gpt-5.5"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready || result.Status != "failed" {
		t.Fatalf("result = %+v, want failed", result)
	}
	if !hasIntegrationStep(result.Steps, "auth", "failed") {
		t.Fatalf("steps = %+v, want auth failure", result.Steps)
	}
	if len(result.Recommended) == 0 {
		t.Fatalf("recommended = %+v, want remediation", result.Recommended)
	}
}

func TestCheckProviderIntegrationReportsBedrockAWSPreflightFailure(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_SESSION_TOKEN", "")
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")
	service := NewService(&memoryProviderStore{})

	result, err := service.CheckProviderIntegration(context.Background(), domain.ProviderIntegrationCheckInput{ProviderID: "bedrock"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready || result.Status != "failed" {
		t.Fatalf("result = %+v, want failed", result)
	}
	if !hasIntegrationStep(result.Steps, "runtime-preflight", "failed") {
		t.Fatalf("steps = %+v, want runtime-preflight failure", result.Steps)
	}
	if len(result.Recommended) == 0 || !strings.Contains(result.Recommended[0], "AWS_ACCESS_KEY_ID") {
		t.Fatalf("recommended = %+v, want AWS credential remediation", result.Recommended)
	}
}

func TestCheckProviderIntegrationReadyForBedrockWithAWSRuntimeEnv(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIATEST")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	t.Setenv("AWS_SESSION_TOKEN", "")
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")
	service := NewService(&memoryProviderStore{})

	result, err := service.CheckProviderIntegration(context.Background(), domain.ProviderIntegrationCheckInput{ProviderID: "amazon-bedrock"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready || result.Status != "ready" {
		t.Fatalf("result = %+v, want ready", result)
	}
	if result.Validation == nil || !result.Validation.Ready || result.Validation.ModelCount == 0 {
		t.Fatalf("validation = %+v, want static Bedrock model metadata", result.Validation)
	}
	if !hasIntegrationStep(result.Steps, "runtime-preflight", "ok") {
		t.Fatalf("steps = %+v, want runtime-preflight ok", result.Steps)
	}
}

func hasIntegrationStep(steps []domain.ProviderIntegrationCheckStep, id string, status string) bool {
	for _, step := range steps {
		if step.ID == id && step.Status == status {
			return true
		}
	}
	return false
}
