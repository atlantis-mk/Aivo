package app

import (
	"context"
	"errors"
	"testing"

	"aivo/core/domain"
)

func TestProviderHTTPErrorClassifiesStatus(t *testing.T) {
	tests := []struct {
		status int
		want   string
	}{
		{401, providerErrorAuth},
		{403, providerErrorAuth},
		{429, providerErrorRateLimit},
		{500, providerErrorUnavailable},
		{400, providerErrorBadRequest},
	}
	for _, tt := range tests {
		err := providerHTTPError(tt.status, "status", "body")
		if got := classifyProviderError(err).Class; got != tt.want {
			t.Fatalf("status %d class = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestProviderErrorRetryable(t *testing.T) {
	if !providerErrorRetryable(providerHTTPError(429, "too many", "")) {
		t.Fatal("429 should be retryable")
	}
	if providerErrorRetryable(providerHTTPError(401, "unauthorized", "")) {
		t.Fatal("401 should not be retryable")
	}
}

func TestCallProviderWithRuntimeRetriesRetryableErrorAndRecordsSuccess(t *testing.T) {
	store := &memoryProviderStore{}
	service := NewService(store)
	route := ResolvedModelRoute{Provider: domain.ProviderConfig{ID: "test-provider"}}
	calls := 0

	resp, err := service.callProviderWithRuntime(context.Background(), route, modelRequirement{}, defaultProviderRuntimePolicy(), 0, func() (domain.ChatResponse, error) {
		calls++
		if calls == 1 {
			return domain.ChatResponse{}, providerHTTPError(503, "unavailable", "")
		}
		return domain.ChatResponse{Text: "ok"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "ok" {
		t.Fatalf("response = %+v, want ok", resp)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
	if store.savedHealth == nil || store.savedHealth.Status != providerHealthReady || store.savedHealth.FailureCount != 0 {
		t.Fatalf("health = %+v, want ready", store.savedHealth)
	}
}

func TestCallProviderWithRuntimeDoesNotRetryWhenDisabled(t *testing.T) {
	store := &memoryProviderStore{}
	service := NewService(store)
	route := ResolvedModelRoute{Provider: domain.ProviderConfig{ID: "test-provider"}}
	calls := 0

	_, err := service.callProviderWithRuntime(context.Background(), route, modelRequirement{Streaming: true}, defaultProviderRuntimePolicy(), 0, func() (domain.ChatResponse, error) {
		calls++
		return domain.ChatResponse{}, providerHTTPError(503, "unavailable", "")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
	if store.savedHealth == nil || store.savedHealth.Status != providerHealthDegraded || store.savedHealth.LastErrorClass != providerErrorUnavailable {
		t.Fatalf("health = %+v, want degraded unavailable", store.savedHealth)
	}
}

func TestCallProviderWithRuntimeHonorsMaxRetriesPolicy(t *testing.T) {
	store := &memoryProviderStore{}
	service := NewService(store)
	route := ResolvedModelRoute{Provider: domain.ProviderConfig{ID: "test-provider"}}
	policy := defaultProviderRuntimePolicy()
	policy.MaxRetries = 0
	calls := 0

	_, err := service.callProviderWithRuntime(context.Background(), route, modelRequirement{}, policy, 0, func() (domain.ChatResponse, error) {
		calls++
		return domain.ChatResponse{}, providerHTTPError(503, "unavailable", "")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want no retry", calls)
	}
}

func TestCallProviderWithRuntimeRecordsAuthFailureUnavailable(t *testing.T) {
	store := &memoryProviderStore{}
	service := NewService(store)
	route := ResolvedModelRoute{Provider: domain.ProviderConfig{ID: "test-provider"}}

	_, err := service.callProviderWithRuntime(context.Background(), route, modelRequirement{}, defaultProviderRuntimePolicy(), 0, func() (domain.ChatResponse, error) {
		return domain.ChatResponse{}, errors.New("credentials are not configured")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if store.savedHealth == nil || store.savedHealth.Status != providerHealthUnavailable || store.savedHealth.LastErrorClass != providerErrorAuth {
		t.Fatalf("health = %+v, want auth unavailable", store.savedHealth)
	}
}

func TestCallProviderWithRuntimeAppliesRateLimitCooldown(t *testing.T) {
	store := &memoryProviderStore{}
	service := NewService(store)
	route := ResolvedModelRoute{Provider: domain.ProviderConfig{ID: "test-provider"}}
	calls := 0

	_, err := service.callProviderWithRuntime(context.Background(), route, modelRequirement{ToolCallCount: 2}, defaultProviderRuntimePolicy(), 0, func() (domain.ChatResponse, error) {
		calls++
		return domain.ChatResponse{}, providerHTTPError(429, "too many requests", "")
	})
	if err == nil {
		t.Fatal("expected rate limit error")
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want only first attempt to reach provider", calls)
	}
	if len(store.callEvents) != 2 {
		t.Fatalf("callEvents = %+v, want provider 429 plus local cooldown event", store.callEvents)
	}
	if store.callEvents[0].ToolCallCount != 2 {
		t.Fatalf("tool count = %d, want 2", store.callEvents[0].ToolCallCount)
	}

	_, err = service.callProviderWithRuntime(context.Background(), route, modelRequirement{}, defaultProviderRuntimePolicy(), 0, func() (domain.ChatResponse, error) {
		calls++
		return domain.ChatResponse{Text: "should not call"}, nil
	})
	if err == nil {
		t.Fatal("expected cooldown error")
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want cooldown to block provider call", calls)
	}
}
