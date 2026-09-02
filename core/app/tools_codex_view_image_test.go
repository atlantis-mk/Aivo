package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestCodexViewImageReadsBoundedWorkspaceImageOnlyForOAuth(t *testing.T) {
	store := codexImageTestStore()
	service := NewService(store)
	defer service.Shutdown()
	workspace := t.TempDir()
	raw, err := base64.StdEncoding.DecodeString(onePixelPNGBase64)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "pixel.png"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	model := domain.ModelRef{ProviderID: "openai", ModelID: "gpt-codex"}
	tool := NewCodexViewImageTool(service)
	result := tool.Execute(context.Background(), json.RawMessage(`{"path":"pixel.png"}`), domain.ToolExecutionContext{
		WorkspaceRoot: workspace, ToolCallID: "call-view", ActiveModel: &model,
	})
	if !result.OK || result.Name != CodexViewImageToolName || result.CallID != "call-view" || len(result.ModelAttachments) != 1 {
		t.Fatalf("result = %+v", result)
	}
	if result.ModelAttachments[0].MIMEType != "image/png" || result.Structured["kind"] != "image" {
		t.Fatalf("attachment = %#v, structured = %#v", result.ModelAttachments, result.Structured)
	}
	if serialized, err := json.Marshal(result); err != nil || string(serialized) == "" || strings.Contains(string(serialized), onePixelPNGBase64) {
		t.Fatalf("serialized result leaked or failed: %q, %v", serialized, err)
	}

	store.auth["openai"] = domain.ProviderAuthRecord{ProviderID: "openai", Method: "api-key", APIKey: "key"}
	blocked := tool.Execute(context.Background(), json.RawMessage(`{"path":"pixel.png"}`), domain.ToolExecutionContext{
		WorkspaceRoot: workspace, ToolCallID: "call-api-key", ActiveModel: &model,
	})
	if blocked.OK || blocked.ToolError == nil || blocked.ToolError.Code != "codex_account_required" {
		t.Fatalf("API-key route result = %+v", blocked)
	}

	store.auth["openai"] = domain.ProviderAuthRecord{ProviderID: "openai", Method: "oauth-browser", AccessToken: "oauth-token", ExpiresAt: "2099-01-01T00:00:00Z", AccountID: "acct-1"}
	store.modelCaches["openai"] = domain.ProviderModelCache{ProviderID: "openai", Models: []domain.ModelInfo{{
		ID: "gpt-codex", ProviderID: "openai", Modalities: []string{"text"},
	}}}
	unsupportedModel := tool.Execute(context.Background(), json.RawMessage(`{"path":"pixel.png"}`), domain.ToolExecutionContext{
		WorkspaceRoot: workspace, ToolCallID: "call-text-only", ActiveModel: &model,
	})
	if unsupportedModel.OK || unsupportedModel.ToolError == nil || unsupportedModel.ToolError.Code != "model_image_unsupported" {
		t.Fatalf("text-only model result = %+v", unsupportedModel)
	}
}

func TestCodexViewImageRejectsEscapingAndUnsupportedPaths(t *testing.T) {
	service := NewService(codexImageTestStore())
	defer service.Shutdown()
	workspace := t.TempDir()
	model := domain.ModelRef{ProviderID: "openai", ModelID: "gpt-codex"}
	tool := NewCodexViewImageTool(service)
	for name, arguments := range map[string]string{
		"workspace escape":      `{"path":"../outside.png"}`,
		"unsupported extension": `{"path":"notes.txt"}`,
		"unknown argument":      `{"path":"pixel.png","detail":"original"}`,
	} {
		t.Run(name, func(t *testing.T) {
			result := tool.Execute(context.Background(), json.RawMessage(arguments), domain.ToolExecutionContext{
				WorkspaceRoot: workspace, ToolCallID: "call-invalid", ActiveModel: &model,
			})
			if result.OK || result.ToolError == nil {
				t.Fatalf("result = %+v", result)
			}
		})
	}
}

func TestCodexViewImageToolInjectionIsOAuthScoped(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	defer service.Shutdown()
	spec := NewCodexViewImageTool(service).Spec()
	oauthRoute := ResolvedModelRoute{
		Provider: domain.ProviderConfig{ID: "openai"}, Model: domain.ModelRef{ProviderID: "openai", ModelID: "gpt-codex"},
		Definition: ProviderDefinition{ID: "openai", BuiltIn: true, Transport: TransportOpenAIResponses},
		Transport:  TransportOpenAIResponses, Credential: llmCredential{Method: "oauth-browser"},
	}
	if tools := service.toolsForModelRoute(context.Background(), domain.AppConfig{}, oauthRoute, []domain.ToolSpec{spec}); len(tools) != 1 || tools[0].Name != CodexViewImageToolName {
		t.Fatalf("OAuth tools = %#v", tools)
	}
	apiKeyRoute := oauthRoute
	apiKeyRoute.Credential = llmCredential{Method: "api-key", APIKey: "key"}
	if tools := service.toolsForModelRoute(context.Background(), domain.AppConfig{}, apiKeyRoute, []domain.ToolSpec{spec}); len(tools) != 0 {
		t.Fatalf("API-key tools = %#v, want none", tools)
	}
	otherProviderRoute := oauthRoute
	otherProviderRoute.Provider.ID = "anthropic"
	otherProviderRoute.Model.ProviderID = "anthropic"
	otherProviderRoute.Definition = ProviderDefinition{ID: "anthropic", BuiltIn: true, Transport: TransportAnthropicMessages}
	otherProviderRoute.Transport = TransportAnthropicMessages
	if tools := service.toolsForModelRoute(context.Background(), domain.AppConfig{}, otherProviderRoute, []domain.ToolSpec{spec}); len(tools) != 0 {
		t.Fatalf("other-provider tools = %#v, want none", tools)
	}
}

func TestWorkspaceRegistryRegistersCodexViewImageAsDormantProviderTool(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	defer service.Shutdown()
	registry, _ := service.toolsForWorkspace(t.TempDir())
	if registry == nil {
		t.Fatal("workspace registry is unavailable")
	}
	entry, found := catalogEntryNamed(registry.CatalogEntries(), CodexViewImageToolName)
	if !found || entry.ActivationPolicy != providerAccountActivationPolicy {
		t.Fatalf("catalog entry = %#v, found = %v", entry, found)
	}
}
