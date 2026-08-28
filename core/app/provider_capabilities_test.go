package app

import (
	"context"
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestValidateModelCapabilitiesRejectsUnsupportedTools(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	route := capabilityTestRoute(domain.ModelInfo{
		ID: "plain-model", ProviderID: "test-provider", Name: "Plain Model", Streaming: true,
		Capabilities: []string{"streaming"},
	})

	err := service.validateModelCapabilities(context.Background(), route, modelRequirement{Tools: true})
	if err == nil || !strings.Contains(err.Error(), "does not support tools") {
		t.Fatalf("err = %v, want tools capability error", err)
	}
}

func TestValidateModelCapabilitiesRejectsUnsupportedReasoning(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	route := capabilityTestRoute(domain.ModelInfo{
		ID: "plain-model", ProviderID: "test-provider", Name: "Plain Model", ToolSupport: true, Streaming: true,
		Capabilities: []string{"tools", "streaming"},
	})

	err := service.validateModelCapabilities(context.Background(), route, modelRequirement{Reasoning: true})
	if err == nil || !strings.Contains(err.Error(), "does not support reasoning") {
		t.Fatalf("err = %v, want reasoning capability error", err)
	}
}

func TestValidateModelCapabilitiesAcceptsDeclaredCapabilities(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	route := capabilityTestRoute(domain.ModelInfo{
		ID: "agent-model", ProviderID: "test-provider", Name: "Agent Model", Streaming: true, ToolSupport: true,
		Capabilities: []string{"tools", "streaming", "reasoning"},
	})

	err := service.validateModelCapabilities(context.Background(), route, modelRequirement{Tools: true, Streaming: true, Reasoning: true})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateModelCapabilitiesRejectsDeprecatedModel(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	route := capabilityTestRoute(domain.ModelInfo{
		ID: "old-model", ProviderID: "test-provider", Name: "Old Model", Deprecated: true,
	})

	err := service.validateModelCapabilities(context.Background(), route, modelRequirement{})
	if err == nil || !strings.Contains(err.Error(), "deprecated") {
		t.Fatalf("err = %v, want deprecated error", err)
	}
}

func TestValidateModelCapabilitiesUsesPersistedModelCache(t *testing.T) {
	service := NewService(&memoryProviderStore{modelCaches: map[string]domain.ProviderModelCache{
		"test-provider": {
			ProviderID: "test-provider",
			Models: []domain.ModelInfo{{
				ID: "cached-model", ProviderID: "test-provider", Name: "Cached Model",
				Capabilities: []string{"streaming"}, Streaming: true,
			}},
		},
	}})
	route := ResolvedModelRoute{
		Provider: domain.ProviderConfig{ID: "test-provider"},
		Model:    domain.ModelRef{ProviderID: "test-provider", ModelID: "cached-model"},
		Definition: ProviderDefinition{
			ID: "test-provider", Models: nil,
		},
	}

	err := service.validateModelCapabilities(context.Background(), route, modelRequirement{Tools: true})
	if err == nil || !strings.Contains(err.Error(), "does not support tools") {
		t.Fatalf("err = %v, want cache-backed tools error", err)
	}
}

func TestValidateModelCapabilitiesAllowsCacheWithoutCapabilityMetadata(t *testing.T) {
	service := NewService(&memoryProviderStore{modelCaches: map[string]domain.ProviderModelCache{
		"deepseek": {
			ProviderID: "deepseek",
			Models: []domain.ModelInfo{{
				ID: "deepseek-v4-flash", ProviderID: "deepseek", Name: "DeepSeek V4 Flash",
			}},
		},
	}})
	route := ResolvedModelRoute{
		Provider: domain.ProviderConfig{ID: "deepseek"},
		Model:    domain.ModelRef{ProviderID: "deepseek", ModelID: "deepseek-v4-flash"},
		Definition: ProviderDefinition{
			ID: "deepseek", Models: nil,
		},
	}

	err := service.validateModelCapabilities(context.Background(), route, modelRequirement{Tools: true, Streaming: true})
	if err != nil {
		t.Fatalf("validateModelCapabilities returned error for unknown metadata: %v", err)
	}
}

func TestValidateModelCapabilitiesAllowsUnknownModelForCompatibility(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	route := ResolvedModelRoute{
		Provider:   domain.ProviderConfig{ID: "test-provider"},
		Model:      domain.ModelRef{ProviderID: "test-provider", ModelID: "unknown-model"},
		Definition: ProviderDefinition{ID: "test-provider"},
	}

	err := service.validateModelCapabilities(context.Background(), route, modelRequirement{Tools: true, Streaming: true, Reasoning: true})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateModelCapabilitiesUsesOnlyExplicitDynamicDimensions(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	route := capabilityTestRoute(domain.ModelInfo{
		ID: "declared-model", ProviderID: "test-provider", Name: "Declared Model",
		DeclaredCapabilities: []string{"tools"}, Capabilities: []string{"streaming"}, Streaming: true,
	})

	if err := service.validateModelCapabilities(context.Background(), route, modelRequirement{Tools: true}); err == nil {
		t.Fatal("tools=false declaration was not enforced")
	}
	if err := service.validateModelCapabilities(context.Background(), route, modelRequirement{Reasoning: true}); err != nil {
		t.Fatalf("undeclared reasoning dimension should remain unknown: %v", err)
	}
}

func TestRequestRequirementsDerivesCapabilities(t *testing.T) {
	req := requestRequirements(domain.ChatRequest{Tools: []domain.ToolSpec{{Name: "read_file"}}}, "high", func(string) {})
	if !req.Tools || !req.Streaming || !req.Reasoning {
		t.Fatalf("requirements = %+v, want all true", req)
	}
}

func TestModelInfoForCodexRouteDoesNotUnionStaticCapabilities(t *testing.T) {
	service := NewService(&memoryProviderStore{modelCaches: map[string]domain.ProviderModelCache{
		"openai": {ProviderID: "openai", Models: []domain.ModelInfo{{
			ID: "gpt-codex", ProviderID: "openai", Modalities: []string{"text"},
			DeclaredCapabilities: []string{codexRuntimeCapability},
		}}},
	}})
	defer service.Shutdown()
	route := ResolvedModelRoute{
		Provider: domain.ProviderConfig{ID: "openai"}, Model: domain.ModelRef{ProviderID: "openai", ModelID: "gpt-codex"},
		Definition: ProviderDefinition{ID: "openai", BuiltIn: true, Transport: TransportOpenAIResponses, Models: []domain.ModelInfo{{
			ID: "gpt-codex", ProviderID: "openai", Modalities: []string{"text", "image"}, Capabilities: []string{"vision"},
		}}},
		Transport: TransportOpenAIResponses, Credential: llmCredential{Method: "oauth-browser"},
	}
	model, ok := service.modelInfoForRoute(context.Background(), route)
	if !ok || len(model.Modalities) != 1 || model.Modalities[0] != "text" || containsString(model.Capabilities, "vision") {
		t.Fatalf("model = %#v", model)
	}
}

func capabilityTestRoute(model domain.ModelInfo) ResolvedModelRoute {
	return ResolvedModelRoute{
		Provider: domain.ProviderConfig{ID: model.ProviderID},
		Model:    domain.ModelRef{ProviderID: model.ProviderID, ModelID: model.ID},
		Definition: ProviderDefinition{
			ID:     model.ProviderID,
			Models: []domain.ModelInfo{model},
		},
	}
}
