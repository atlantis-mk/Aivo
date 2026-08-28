package app

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aivo/core/domain"
)

const onePixelPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Wl2H0gAAAAASUVORK5CYII="

func TestCodexImageGenerationUsesOAuthAccountEndpointAndSavesArtifact(t *testing.T) {
	originalClient := codexImageHTTPClient
	defer func() { codexImageHTTPClient = originalClient }()
	codexImageHTTPClient = &http.Client{Transport: providerModelRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != chatGPTCodexImageGenerationURL {
			t.Fatalf("URL = %s", req.URL)
		}
		if req.Header.Get("Authorization") != "Bearer oauth-token" || req.Header.Get("ChatGPT-Account-Id") != "acct-1" || req.Header.Get("x-codex-image-turn-id") != "turn-1" {
			t.Fatalf("headers = %#v", req.Header)
		}
		var body map[string]any
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != codexImageModel || body["prompt"] != "draw a tiny dot" || body["quality"] != "auto" || body["size"] != "auto" {
			t.Fatalf("body = %#v", body)
		}
		if _, ok := body["images"]; ok {
			t.Fatalf("generation body unexpectedly contains images: %#v", body)
		}
		response := `{"created":1,"background":"opaque","quality":"auto","size":"1024x1024","data":[{"b64_json":"` + onePixelPNGBase64 + `"}]}`
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response)), Request: req}, nil
	})}
	store := codexImageTestStore()
	service := NewService(store)
	defer service.Shutdown()
	workspace := t.TempDir()
	model := domain.ModelRef{ProviderID: "openai", ModelID: "gpt-codex"}
	result := NewCodexImageGenerationTool(service).Execute(context.Background(), json.RawMessage(`{"prompt":"draw a tiny dot"}`), domain.ToolExecutionContext{
		WorkspaceRoot: workspace, SessionID: "session-1", TurnID: "turn-1", ToolCallID: "call-1", ActiveModel: &model,
	})
	if !result.OK || len(result.ModelAttachments) != 1 || result.ModelAttachments[0].MIMEType != "image/png" || len(result.Files) != 1 {
		t.Fatalf("result = %+v", result)
	}
	path, _ := result.Structured["path"].(string)
	resolvedWorkspace, _ := filepath.EvalSymlinks(workspace)
	if !strings.HasPrefix(path, filepath.Join(resolvedWorkspace, "generated_images")+string(os.PathSeparator)) {
		t.Fatalf("path = %q", path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := base64.StdEncoding.DecodeString(onePixelPNGBase64)
	if string(raw) != string(want) {
		t.Fatal("saved image differs from provider response")
	}
	if encoded, _ := json.Marshal(result); strings.Contains(string(encoded), onePixelPNGBase64) {
		t.Fatal("serialized tool result leaked generated base64")
	}
}

func TestCodexImageGenerationEditsRecentImagesAndRejectsNonOAuthRoute(t *testing.T) {
	originalClient := codexImageHTTPClient
	defer func() { codexImageHTTPClient = originalClient }()
	codexImageHTTPClient = &http.Client{Transport: providerModelRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != chatGPTCodexImageEditURL {
			t.Fatalf("URL = %s", req.URL)
		}
		var body struct {
			Images []map[string]string `json:"images"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.Images) != 1 || !strings.HasPrefix(body.Images[0]["image_url"], "data:image/png;base64,") {
			t.Fatalf("images = %#v", body.Images)
		}
		response := `{"created":1,"data":[{"b64_json":"` + onePixelPNGBase64 + `"}]}`
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response)), Request: req}, nil
	})}
	store := codexImageTestStore()
	service := NewService(store)
	defer service.Shutdown()
	model := domain.ModelRef{ProviderID: "openai", ModelID: "gpt-codex"}
	result := NewCodexImageGenerationTool(service).Execute(context.Background(), json.RawMessage(`{"prompt":"make it blue","num_last_images_to_include":1}`), domain.ToolExecutionContext{
		WorkspaceRoot: t.TempDir(), TurnID: "turn-1", ToolCallID: "call-1", ActiveModel: &model,
		RecentImages: []domain.MessageAttachment{{Name: "input.png", MIMEType: "image/png", Kind: "image", Data: onePixelPNGBase64}},
	})
	if !result.OK || result.Structured["operation"] != "edit" {
		t.Fatalf("edit result = %+v", result)
	}

	store.auth["openai"] = domain.ProviderAuthRecord{ProviderID: "openai", Method: "api-key", APIKey: "key"}
	blocked := NewCodexImageGenerationTool(service).Execute(context.Background(), json.RawMessage(`{"prompt":"draw"}`), domain.ToolExecutionContext{
		WorkspaceRoot: t.TempDir(), ToolCallID: "call-2", ActiveModel: &model,
	})
	if blocked.OK || blocked.ToolError == nil || blocked.ToolError.Code != "codex_account_required" {
		t.Fatalf("API-key route result = %+v", blocked)
	}
}

func TestCodexImageToolActivationAndNamespaceAreAccountScoped(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	defer service.Shutdown()
	spec := NewCodexImageGenerationTool(service).Spec()
	definition := ProviderDefinition{ID: "openai", BuiltIn: true, Transport: TransportOpenAIResponses}
	oauthRoute := ResolvedModelRoute{
		Provider: domain.ProviderConfig{ID: "openai"}, Model: domain.ModelRef{ProviderID: "openai", ModelID: "gpt-codex"},
		Definition: definition, Transport: TransportOpenAIResponses, Credential: llmCredential{Method: "oauth-browser"},
	}
	tools := service.toolsForModelRoute(context.Background(), domain.AppConfig{}, oauthRoute, []domain.ToolSpec{spec})
	if len(tools) != 1 || tools[0].Name != CodexImagegenToolName || tools[0].Namespace != codexImagegenNamespace {
		t.Fatalf("OAuth tools = %#v", tools)
	}
	apiKeyRoute := oauthRoute
	apiKeyRoute.Credential = llmCredential{Method: "api-key", APIKey: "key"}
	if tools := service.toolsForModelRoute(context.Background(), domain.AppConfig{}, apiKeyRoute, []domain.ToolSpec{spec}); len(tools) != 0 {
		t.Fatalf("API-key tools = %#v, want none", tools)
	}
}

func TestCodexImageGenerationUsesReservedResponsesDeclaration(t *testing.T) {
	spec := NewCodexImageGenerationTool(nil).Spec()
	if got := sha256.Sum256([]byte(spec.Description)); got != [32]byte{0x77, 0xa9, 0x92, 0xa7, 0xc9, 0x0e, 0x45, 0xfc, 0xd1, 0x16, 0x23, 0xa1, 0xef, 0xa3, 0x4b, 0xfd, 0x4c, 0x78, 0x70, 0x69, 0x7e, 0x0a, 0xa5, 0x4c, 0xe9, 0xb2, 0x8f, 0x69, 0x08, 0x77, 0x17, 0x0e} {
		t.Fatalf("imagegen description drifted from the Codex reserved declaration: %x", got)
	}
	got, err := json.Marshal(responsesTools([]domain.ToolSpec{spec}))
	if err != nil {
		t.Fatal(err)
	}
	want, err := json.Marshal([]map[string]any{{
		"type":        "namespace",
		"name":        "image_gen",
		"description": "Tools in the image_gen namespace.",
		"tools": []map[string]any{{
			"type":        "function",
			"name":        "imagegen",
			"description": codexImagegenDescription,
			"strict":      false,
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"prompt": map[string]any{"type": "string"},
					"referenced_image_paths": map[string]any{
						"type":  []string{"array", "null"},
						"items": map[string]any{"type": "string", "description": codexAbsolutePathSchemaDescription},
					},
					"num_last_images_to_include": map[string]any{"type": []string{"integer", "null"}},
				},
				"required":             []string{"prompt"},
				"additionalProperties": false,
			},
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("reserved imagegen declaration = %s, want %s", got, want)
	}
}

func TestCodexImageGenerationRejectsInvalidEditSelectionBeforeNetwork(t *testing.T) {
	service := NewService(codexImageTestStore())
	defer service.Shutdown()
	tool := NewCodexImageGenerationTool(service)
	model := domain.ModelRef{ProviderID: "openai", ModelID: "gpt-codex"}
	for name, raw := range map[string]string{
		"too many paths":  `{"prompt":"edit","referenced_image_paths":["1.png","2.png","3.png","4.png","5.png","6.png"]}`,
		"both selectors":  `{"prompt":"edit","referenced_image_paths":["1.png"],"num_last_images_to_include":1}`,
		"too many recent": `{"prompt":"edit","num_last_images_to_include":6}`,
	} {
		t.Run(name, func(t *testing.T) {
			result := tool.Execute(context.Background(), json.RawMessage(raw), domain.ToolExecutionContext{WorkspaceRoot: t.TempDir(), ToolCallID: "call-1", ActiveModel: &model})
			if result.OK || result.ToolError == nil || result.ToolError.Code != "invalid_arguments" {
				t.Fatalf("result = %+v", result)
			}
		})
	}
}

func TestResponsesParserPreservesNamespacedToolCall(t *testing.T) {
	calls := extractResponseToolCalls(map[string]any{"output": []any{map[string]any{
		"type": "function_call", "call_id": "call-1", "namespace": "image_gen", "name": "imagegen", "arguments": `{"prompt":"draw"}`,
	}}})
	if len(calls) != 1 || calls[0].Namespace != "image_gen" || calls[0].Name != "imagegen" {
		t.Fatalf("calls = %#v", calls)
	}
	serialized := responsesToolCalls(calls)
	if serialized[0]["namespace"] != "image_gen" {
		t.Fatalf("serialized = %#v", serialized)
	}
}

func codexImageTestStore() *memoryProviderStore {
	return &memoryProviderStore{
		config: &domain.AppConfig{
			Provider:     &domain.ProviderConfig{ID: "openai", Type: string(TransportOpenAIResponses), Model: "gpt-codex"},
			DefaultModel: &domain.ModelRef{ProviderID: "openai", ModelID: "gpt-codex"},
		},
		auth: map[string]domain.ProviderAuthRecord{"openai": {
			ProviderID: "openai", Method: "oauth-browser", AccessToken: "oauth-token", ExpiresAt: "2099-01-01T00:00:00Z", AccountID: "acct-1",
		}},
		modelCaches: map[string]domain.ProviderModelCache{"openai": {ProviderID: "openai", Models: []domain.ModelInfo{{
			ID: "gpt-codex", ProviderID: "openai", Capabilities: []string{"tools"}, ToolSupport: true,
		}}}},
	}
}
