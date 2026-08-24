package app

import (
	"testing"

	"aivo/core/domain"
)

func TestNormalizeChatGPTCodexModelMapsLegacyCodexModel(t *testing.T) {
	model := normalizeChatGPTCodexModel(domain.ModelRef{ProviderID: "openai", ModelID: "gpt-5-codex"})

	if model.ModelID != "gpt-5.5" {
		t.Fatalf("ModelID = %q, want %q", model.ModelID, "gpt-5.5")
	}
}

func TestDefaultOpenAIProviderUsesSupportedCodexAccountModel(t *testing.T) {
	model := defaultModelFor("openai")

	if model != "gpt-5.5" {
		t.Fatalf("defaultModelFor(openai) = %q, want %q", model, "gpt-5.5")
	}
}

func assertResponsesContent(t *testing.T, item struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}, role string, contentType string, text string) {
	t.Helper()
	if item.Role != role {
		t.Fatalf("Role = %q, want %q", item.Role, role)
	}
	content, ok := item.Content.([]any)
	if !ok || len(content) != 1 {
		t.Fatalf("Content = %#v, want one content item", item.Content)
	}
	part, ok := content[0].(map[string]any)
	if !ok {
		t.Fatalf("Content[0] = %#v, want object", content[0])
	}
	if part["type"] != contentType || part["text"] != text {
		t.Fatalf("Content[0] = %#v, want type %q text %q", part, contentType, text)
	}
}
