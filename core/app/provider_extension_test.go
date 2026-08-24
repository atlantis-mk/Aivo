package app

import (
	"context"
	"testing"

	"aivo/core/domain"
)

func TestServiceRegisterProviderDefinitionAppearsInCatalog(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	err := service.RegisterProviderDefinition(ProviderDefinition{
		ID:              "team-gateway",
		DisplayName:     "Team Gateway",
		Description:     "Internal team model gateway.",
		Aliases:         []string{"team"},
		Transport:       TransportAnthropicMessages,
		AuthTypes:       []AuthType{AuthAPIKey},
		DefaultAuthType: AuthAPIKey,
		DefaultBaseURL:  "https://gateway.example.com/anthropic/v1",
		APIKeyEnvVars:   []string{"TEAM_GATEWAY_KEY"},
		ModelFetch:      ModelFetchAnthropic,
		DefaultModelID:  "team-sonnet",
		Models: []domain.ModelInfo{{
			ID: "team-sonnet", Name: "Team Sonnet", Capabilities: []string{"tools", "streaming"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	catalog, err := service.Catalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var found *domain.ProviderInfo
	for i := range catalog.Providers {
		if catalog.Providers[i].ID == "team-gateway" {
			found = &catalog.Providers[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("registered provider missing from catalog")
	}
	if found.Type != string(TransportAnthropicMessages) || found.Environment != "TEAM_GATEWAY_KEY" || found.DefaultModelID != "team-sonnet" {
		t.Fatalf("provider = %+v, want registered metadata", found)
	}
	if found.Profile == nil || found.Profile.MessageShape != string(TransportAnthropicMessages) {
		t.Fatalf("profile = %+v, want anthropic message shape", found.Profile)
	}
}

func TestRegisteredProviderAliasConfigAndRouteResolution(t *testing.T) {
	oldLookup := lookupEnv
	defer func() { lookupEnv = oldLookup }()
	lookupEnv = func(name string) string {
		if name == "TEAM_GATEWAY_KEY" {
			return "team-key"
		}
		return ""
	}
	service := NewService(&memoryProviderStore{})
	err := service.RegisterProviderDefinition(ProviderDefinition{
		ID:              "team-gateway",
		DisplayName:     "Team Gateway",
		Aliases:         []string{"team"},
		Transport:       TransportAnthropicMessages,
		AuthTypes:       []AuthType{AuthAPIKey},
		DefaultAuthType: AuthAPIKey,
		DefaultBaseURL:  "https://gateway.example.com/anthropic/v1",
		APIKeyEnvVars:   []string{"TEAM_GATEWAY_KEY"},
		ModelFetch:      ModelFetchAnthropic,
		DefaultModelID:  "team-sonnet",
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg, def, err := service.providerConfigFromInput(domain.ProviderConnectInput{
		ProviderID: "team",
		Method:     "api-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ID != "team-gateway" || cfg.Type != string(TransportAnthropicMessages) || cfg.APIKeyEnv != "TEAM_GATEWAY_KEY" || def.ID != "team-gateway" {
		t.Fatalf("cfg=%+v def=%+v, want registered provider defaults", cfg, def)
	}

	route, err := service.ResolveModelRoute(context.Background(), domain.AppConfig{
		DefaultModel: &domain.ModelRef{ProviderID: "team", ModelID: "team-sonnet"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if route.Provider.ID != "team-gateway" || route.Transport != TransportAnthropicMessages || route.Credential.APIKey != "team-key" {
		t.Fatalf("route = %+v credential=%+v, want registered anthropic route", route, route.Credential)
	}
}

func TestProviderRegistryClonesDefinitionsOnRead(t *testing.T) {
	registry := NewDefaultProviderRegistry()
	if err := registry.RegisterDefinition(ProviderDefinition{
		ID: "clone-test", Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey},
		DefaultBaseURL: "https://clone.example.com/v1", DefaultModelID: "model",
		Models: []domain.ModelInfo{{ID: "model", Name: "Model"}},
	}); err != nil {
		t.Fatal(err)
	}
	def, ok := registry.Definition("clone-test")
	if !ok {
		t.Fatal("definition missing")
	}
	def.Models[0].ID = "mutated"
	def.Aliases = append(def.Aliases, "mutated")

	next, ok := registry.Definition("clone-test")
	if !ok {
		t.Fatal("definition missing after mutation")
	}
	if next.Models[0].ID != "model" || len(next.Aliases) != 0 {
		t.Fatalf("definition leaked mutation: %+v", next)
	}
}
