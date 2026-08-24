package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"aivo/core/domain"
)

func TestGenerateMCPDescriptionUsesCompleteSafeToolProjectionWithoutMutation(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()

	server, err := service.SaveMCPServer(ctx, domain.SaveMCPServerInput{Server: domain.MCPServerConfig{
		ID: "linear", Name: "linear", DisplayName: "Private Linear", Description: "original description",
		Transport: domain.MCPTransportStreamableHTTP, URL: "https://private.example.test/mcp",
		Command: "secret-command", Args: []string{"--secret-argument"}, CWD: "/private/root",
		Env: map[string]string{"PRIVATE_ENV": "secret-environment-value"}, Headers: map[string]string{"X-Private": "secret-header-value"},
		Roots: []string{"/private/project"}, AuthType: domain.MCPAuthNone,
	}})
	if err != nil {
		t.Fatal(err)
	}
	tools := []domain.MCPToolRecord{
		{Name: "update_issue", Description: "更新指定 issue"},
		{Name: "get_issue", Description: "读取一个 issue"},
		{Name: "list_projects", Description: "列出项目和团队"},
	}
	if err := service.mcpManager.store.ReplaceMCPTools(ctx, server.ID, tools); err != nil {
		t.Fatal(err)
	}

	var requestBody []byte
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		requestBody, _ = json.Marshal(body)
		if declared, ok := body["tools"].([]any); ok && len(declared) != 0 {
			t.Fatalf("description generation received executable tools: %#v", declared)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"choices\":[{\"message\":{\"content\":\"```text\\n查询、创建和更新 Linear 中的 issue、项目与团队信息。\\n```\"}}]}"))
	}))
	defer provider.Close()
	if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{
		ProviderID: "custom-api", Type: "openai-compatible", BaseURL: provider.URL,
		ModelID: "test-model", APIKey: "test-key", Method: "api-key",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateModelPreferences(ctx, domain.ModelPreferencesInput{
		AuxiliaryModel: &domain.ModelRef{ProviderID: "custom-api", ModelID: "test-model"},
	}); err != nil {
		t.Fatal(err)
	}

	result, err := service.GenerateMCPDescription(ctx, domain.MCPDescriptionGenerateInput{ServerID: server.ID})
	if err != nil {
		t.Fatal(err)
	}
	if result.Description != "查询、创建和更新 Linear 中的 issue、项目与团队信息。" {
		t.Fatalf("description = %q", result.Description)
	}
	body := string(requestBody)
	for _, expected := range []string{"get_issue", "读取一个 issue", "list_projects", "列出项目和团队", "update_issue", "更新指定 issue"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("provider prompt omitted %q: %s", expected, body)
		}
	}
	for _, secret := range []string{"Private Linear", "original description", "private.example.test", "secret-command", "secret-argument", "private/root", "PRIVATE_ENV", "secret-environment-value", "X-Private", "secret-header-value", "private/project"} {
		if strings.Contains(body, secret) {
			t.Fatalf("provider prompt exposed MCP configuration %q: %s", secret, body)
		}
	}
	stored, err := service.mcpManager.store.GetMCPServer(ctx, server.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Description != "original description" {
		t.Fatalf("stored description mutated to %q", stored.Description)
	}
}

func TestBuildMCPDescriptionCatalogRefusesIncompleteGeneration(t *testing.T) {
	if _, err := buildMCPDescriptionCatalog(nil); err == nil || !strings.Contains(err.Error(), "no discovered tools") {
		t.Fatalf("empty catalog error = %v", err)
	}
	tools := make([]domain.MCPToolRecord, maxMCPDescriptionCatalogTools+1)
	for index := range tools {
		tools[index] = domain.MCPToolRecord{Name: "tool"}
	}
	if _, err := buildMCPDescriptionCatalog(tools); err == nil || !strings.Contains(err.Error(), "generation limit") {
		t.Fatalf("oversized catalog error = %v", err)
	}
}

func TestGenerateMCPDescriptionRequiresConfiguredAuxiliaryModel(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	server, err := service.SaveMCPServer(ctx, domain.SaveMCPServerInput{Server: domain.MCPServerConfig{
		ID: "docs", Name: "docs", Transport: domain.MCPTransportStdio, Command: "docs-mcp",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.mcpManager.store.ReplaceMCPTools(ctx, server.ID, []domain.MCPToolRecord{{Name: "search", Description: "Search docs"}}); err != nil {
		t.Fatal(err)
	}
	_, err = service.GenerateMCPDescription(ctx, domain.MCPDescriptionGenerateInput{ServerID: server.ID})
	if err == nil || !strings.Contains(err.Error(), "auxiliary model is not configured") {
		t.Fatalf("missing auxiliary model error = %v", err)
	}
}

func TestCleanMCPDescriptionPreservesValidUTF8AtByteLimit(t *testing.T) {
	value := cleanMCPDescription(strings.Repeat("能力", 300))
	if len(value) > maxMCPDescriptionBytes || !utf8.ValidString(value) {
		t.Fatalf("cleaned description bytes=%d valid=%t", len(value), utf8.ValidString(value))
	}
}
