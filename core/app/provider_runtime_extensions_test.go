package app

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"aivo/core/domain"
)

func TestProviderDefinitionFromRuntimeExtensionSupportsCompatibleAndCommandAdapters(t *testing.T) {
	compatible, ok := providerDefinitionFromRuntimeExtension("acme", domain.ProviderExtensionDefinition{
		Protocol: "openai-compatible", DisplayName: "Acme", BaseURL: "https://example.test/v1/",
		CredentialRef: "ACME_API_KEY", Models: []string{"acme-code", "acme-fast"},
	})
	if !ok || compatible.Transport != TransportOpenAICompatible || compatible.DefaultBaseURL != "https://example.test/v1" || compatible.DefaultModelID != "acme-code" {
		t.Fatalf("compatible definition = %#v", compatible)
	}
	command, ok := providerDefinitionFromRuntimeExtension("local", domain.ProviderExtensionDefinition{
		Protocol: "command", Command: os.Args[0], Args: []string{"-test.run=TestExternalProviderHelperProcess", "--", "external-provider"}, Models: []string{"local-code"},
	})
	if !ok || command.Transport != TransportExternalProcess || command.Command == "" {
		t.Fatalf("command definition = %#v", command)
	}
}

func TestExternalProcessProviderUsesJSONContract(t *testing.T) {
	definition, _ := providerDefinitionFromRuntimeExtension("local", domain.ProviderExtensionDefinition{
		Protocol: "command", Command: os.Args[0], Args: []string{"-test.run=TestExternalProviderHelperProcess", "--", "external-provider"},
	})
	response, err := callExternalProcessProvider(context.Background(), definition, domain.ModelRef{ProviderID: "local", ModelID: "local-code"}, []llmChatMessage{{Role: "user", Text: "hello"}}, nil, "low", "default", nil)
	if err != nil {
		t.Fatal(err)
	}
	if response.Text != "local:hello" {
		t.Fatalf("response = %#v", response)
	}
}

func TestRefreshProviderExtensionsPreservesContributionsAndIsolatesInvalidEntries(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	service := NewService(&memoryProviderStore{})
	t.Cleanup(service.Shutdown)
	if err := service.RegisterProviderDefinition(ProviderDefinition{
		ID: "embedded", DisplayName: "Embedded", Transport: TransportOpenAICompatible,
		Models:         []domain.ModelInfo{{ID: "embedded-code", ProviderID: "embedded", Name: "Embedded Code"}},
		DefaultModelID: "embedded-code", ModelFetch: ModelFetchStatic,
	}); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "aivo.json"), []byte(`{
		"providerExtensions": {
			"project-local": {"protocol":"openai-compatible","baseUrl":"http://127.0.0.1:1/v1","models":["project-code"]},
			"broken": {"protocol":"unsupported","models":["ignored"]}
		}
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	service.refreshProviderExtensions(root)
	for _, providerID := range []string{"embedded", "project-local"} {
		definition, ok := service.providerDefinition(providerID)
		if !ok || len(definition.Models) != 1 {
			t.Fatalf("provider %q = %#v, %v", providerID, definition, ok)
		}
	}
	if _, ok := service.providerDefinition("broken"); ok {
		t.Fatal("invalid provider extension should not enter the registry")
	}
}

func TestExternalProviderHelperProcess(t *testing.T) {
	if !hasArg("external-provider") {
		return
	}
	raw, _ := io.ReadAll(os.Stdin)
	var request struct {
		Messages []llmChatMessage `json:"messages"`
	}
	_ = json.Unmarshal(raw, &request)
	text := ""
	if len(request.Messages) > 0 {
		text = request.Messages[len(request.Messages)-1].Text
	}
	_ = json.NewEncoder(os.Stdout).Encode(domain.ChatResponse{Text: "local:" + text})
	os.Exit(0)
}
