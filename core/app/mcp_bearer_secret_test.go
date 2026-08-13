package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestMCPDirectBearerTokenPersistsOnlyReferenceAndSurvivesBlankEdit(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	secrets := NewMemorySecretStore()
	service.SetSecretStore(secrets)

	server := domain.MCPServerConfig{
		ID:          "direct_bearer",
		Name:        "direct_bearer",
		Description: "Access the remote bearer-authenticated MCP",
		Transport:   domain.MCPTransportStreamableHTTP,
		URL:         "https://mcp.example.test",
		AuthType:    domain.MCPAuthBearer,
	}
	saved, err := service.SaveMCPServer(ctx, domain.SaveMCPServerInput{
		Server:      server,
		BearerToken: "desktop-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.BearerTokenRef == "" || saved.BearerTokenEnv != "" || saved.BearerToken != "" {
		t.Fatalf("saved auth = %#v, want reference-only direct bearer", saved)
	}
	value, err := secrets.Get(ctx, saved.BearerTokenRef)
	if err != nil || value != "desktop-token" {
		t.Fatalf("secret = %q, %v; want stored token", value, err)
	}
	raw, err := json.Marshal(saved)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "desktop-token") || strings.Contains(string(raw), "bearerToken\"") {
		t.Fatalf("saved JSON leaked token: %s", raw)
	}

	saved.DisplayName = "Edited"
	edited, err := service.SaveMCPServer(ctx, domain.SaveMCPServerInput{Server: saved})
	if err != nil {
		t.Fatal(err)
	}
	if edited.BearerTokenRef != saved.BearerTokenRef {
		t.Fatalf("edited ref = %q, want %q", edited.BearerTokenRef, saved.BearerTokenRef)
	}
	value, err = secrets.Get(ctx, edited.BearerTokenRef)
	if err != nil || value != "desktop-token" {
		t.Fatalf("preserved secret = %q, %v; want stored token", value, err)
	}
}

func TestMCPDirectBearerTokenAuthorizesRequestAndEnvironmentModeCleansReference(t *testing.T) {
	ctx := context.Background()
	secrets := NewMemorySecretStore()
	ref := "mcp-auth/direct_bearer/access-token"
	if err := secrets.Put(ctx, ref, "direct-token"); err != nil {
		t.Fatal(err)
	}
	server, err := resolveMCPAuthSecrets(ctx, secrets, domain.MCPServerConfig{
		ID:             "direct_bearer",
		AuthType:       domain.MCPAuthBearer,
		BearerTokenRef: ref,
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "https://mcp.example.test", nil)
	if err := applyMCPRequestAuth(request, server); err != nil {
		t.Fatal(err)
	}
	if got := request.Header.Get("Authorization"); got != "Bearer direct-token" {
		t.Fatalf("Authorization = %q, want direct bearer", got)
	}

	service, cleanup := newSessionTestService(t)
	defer cleanup()
	service.SetSecretStore(secrets)
	configured, err := service.SaveMCPServer(ctx, domain.SaveMCPServerInput{
		Server: domain.MCPServerConfig{
			ID: "direct_bearer", Name: "direct_bearer",
			Description: "Access the remote bearer-authenticated MCP",
			Transport:   domain.MCPTransportStreamableHTTP,
			URL:         "https://mcp.example.test",
			AuthType:    domain.MCPAuthBearer,
		},
		BearerToken: "direct-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	configured.BearerTokenEnv = "AIVO_MCP_TOKEN"
	configured.BearerTokenRef = ""
	environment, err := service.SaveMCPServer(ctx, domain.SaveMCPServerInput{Server: configured})
	if err != nil {
		t.Fatal(err)
	}
	if environment.BearerTokenEnv != "AIVO_MCP_TOKEN" || environment.BearerTokenRef != "" {
		t.Fatalf("environment auth = %#v, want environment reference", environment)
	}
	value, err := secrets.Get(ctx, ref)
	if err != nil || value != "" {
		t.Fatalf("old direct secret = %q, %v; want deleted", value, err)
	}
}

func TestMCPDirectBearerSaveFailureRemovesNewSecret(t *testing.T) {
	ctx := context.Background()
	secrets := NewMemorySecretStore()
	store := &failingMCPBearerStore{memoryProviderStore: &memoryProviderStore{}, failSave: true}
	manager := NewMCPManager(store, secrets)
	_, err := manager.Save(ctx, domain.SaveMCPServerInput{
		Server: domain.MCPServerConfig{
			ID: "failing", Name: "failing",
			Description: "Exercise MCP save failure cleanup",
			Transport:   domain.MCPTransportStreamableHTTP,
			URL:         "https://mcp.example.test",
			AuthType:    domain.MCPAuthBearer,
		},
		BearerToken: "must-not-remain",
	})
	if err == nil {
		t.Fatal("Save() error = nil, want persistence failure")
	}
	value, getErr := secrets.Get(ctx, "mcp-auth/failing/access-token")
	if getErr != nil || value != "" {
		t.Fatalf("secret after failure = %q, %v; want removed", value, getErr)
	}
}

type failingMCPBearerStore struct {
	*memoryProviderStore
	failSave bool
}

func (s *failingMCPBearerStore) SaveMCPServer(ctx context.Context, server domain.MCPServerConfig) (domain.MCPServerConfig, error) {
	if s.failSave {
		return domain.MCPServerConfig{}, errors.New("save failed")
	}
	return s.memoryProviderStore.SaveMCPServer(ctx, server)
}
