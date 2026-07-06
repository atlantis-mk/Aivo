package app

import (
	"context"
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestStartProviderAuthRejectsUnsupportedProviderThroughDriverRegistry(t *testing.T) {
	service := NewService(&memoryProviderStore{})

	_, err := service.StartProviderAuth(context.Background(), domain.ProviderAuthStartInput{
		ProviderID: "anthropic",
		Method:     "oauth-browser",
	})
	if err == nil || !strings.Contains(err.Error(), "does not support interactive auth") {
		t.Fatalf("err = %v, want unsupported interactive auth", err)
	}
}

func TestStartProviderAuthUsesOpenAIDriverMethodValidation(t *testing.T) {
	service := NewService(&memoryProviderStore{})

	_, err := service.StartProviderAuth(context.Background(), domain.ProviderAuthStartInput{
		ProviderID: "openai",
		Method:     "unsupported",
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported OpenAI auth method") {
		t.Fatalf("err = %v, want unsupported OpenAI method", err)
	}
}

func TestProviderAuthStatusAndCancelDefaultToIdle(t *testing.T) {
	service := NewService(&memoryProviderStore{})

	status, err := service.GetProviderAuthStatus(context.Background(), "openai")
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "idle" || status.ProviderID != "openai" {
		t.Fatalf("status = %+v, want idle", status)
	}
	status, err = service.CancelProviderAuth(context.Background(), "openai")
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "idle" || status.ProviderID != "openai" {
		t.Fatalf("cancel status = %+v, want idle", status)
	}
}

func TestProviderAuthManagerRegistersOpenAIDriver(t *testing.T) {
	service := NewService(&memoryProviderStore{})

	driver, ok := service.authFlows.driver("openai-api")
	if !ok {
		t.Fatal("openai driver missing for alias")
	}
	if driver.ProviderID() != "openai" {
		t.Fatalf("driver provider = %q, want openai", driver.ProviderID())
	}
	methods := strings.Join(driver.SupportedMethods(), ",")
	if !strings.Contains(methods, "oauth-browser") || !strings.Contains(methods, "oauth-headless") {
		t.Fatalf("methods = %q, want browser and headless", methods)
	}
}

func TestServiceRegisterProviderAuthDriver(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	service.RegisterProviderAuthDriver(fakeProviderAuthDriver{})

	result, err := service.StartProviderAuth(context.Background(), domain.ProviderAuthStartInput{
		ProviderID: "fake-oauth",
		Method:     "oauth-browser",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ProviderID != "fake-oauth" || result.Status != "pending" {
		t.Fatalf("result = %+v, want fake pending auth", result)
	}
}

type fakeProviderAuthDriver struct{}

func (fakeProviderAuthDriver) ProviderID() string         { return "fake-oauth" }
func (fakeProviderAuthDriver) SupportedMethods() []string { return []string{"oauth-browser"} }
func (fakeProviderAuthDriver) Start(_ context.Context, input domain.ProviderAuthStartInput) (domain.ProviderAuthStartResult, error) {
	return domain.ProviderAuthStartResult{ProviderID: input.ProviderID, Method: input.Method, Status: "pending"}, nil
}
