package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aivo/core/domain"
)

type providerModelRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn providerModelRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestParseCodexModelsOrdersLatestOpenAIModelFirst(t *testing.T) {
	raw := []byte(`{
		"models": [
			{"slug": "gpt-5.4", "display_name": "GPT-5.4", "visibility": "list", "priority": 1},
			{"slug": "gpt-5.5", "display_name": "GPT-5.5", "visibility": "list", "priority": 2},
			{"slug": "gpt-5.3-codex-spark", "display_name": "GPT-5.3-Codex-Spark", "visibility": "list", "priority": 3}
		]
	}`)

	models, defaultModel, err := parseCodexModels(raw, "openai")
	if err != nil {
		t.Fatalf("parseCodexModels returned error: %v", err)
	}
	if defaultModel != "gpt-5.5" {
		t.Fatalf("defaultModel = %q, want %q", defaultModel, "gpt-5.5")
	}
	if got := models[0].ID; got != "gpt-5.5" {
		t.Fatalf("models[0].ID = %q, want %q", got, "gpt-5.5")
	}
}

func TestParseCodexModelsRetainsRecognizedShellDeclarations(t *testing.T) {
	raw := []byte(`{
		"models": [
			{"slug":"supported","visibility":"list","shell_type":"unified_exec"},
			{"slug":"disabled","visibility":"list","shell_type":"disabled"},
			{"slug":"unknown","visibility":"list","shell_type":"future_shell"}
		]
	}`)
	models, _, err := parseCodexModels(raw, "openai")
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]domain.ModelInfo{}
	for _, model := range models {
		byID[model.ID] = model
	}
	supported := byID["supported"]
	if !declaredModelCapabilitySupported(supported, codexShellCapability) {
		t.Fatalf("supported model = %+v, want recognized shell declaration", supported)
	}
	disabled := byID["disabled"]
	if !containsString(disabled.DeclaredCapabilities, codexShellCapability) || modelSupportsCapability(disabled, codexShellCapability) {
		t.Fatalf("disabled model = %+v, want explicit shell denial", disabled)
	}
	unknown := byID["unknown"]
	if containsString(unknown.DeclaredCapabilities, codexShellCapability) {
		t.Fatalf("unknown model = %+v, unrecognized shapes must remain unknown", unknown)
	}
}

func TestParseCodexModelsRetainsRecognizedRuntimeCapabilities(t *testing.T) {
	raw := []byte(`{"models":[{
		"slug":"gpt-runtime","display_name":"GPT Runtime","visibility":"list",
		"context_window":272000,"max_context_window":872000,"auto_compact_token_limit":240000,
		"default_reasoning_level":"xhigh","supported_reasoning_levels":[{"effort":"low"},{"effort":"xhigh"},{"effort":"future"}],
		"support_verbosity":true,"default_verbosity":"low",
		"service_tiers":[{"id":"fast"},{"id":"flex"}],"default_service_tier":"fast",
		"input_modalities":["text","image","future"],"supports_parallel_tool_calls":false,
		"supports_image_detail_original":true,"use_responses_lite":true,"supports_search_tool":true,
		"shell_type":"unified_exec","web_search_tool_type":"text_and_image"
	}]}`)
	models, _, err := parseCodexModels(raw, "openai")
	if err != nil {
		t.Fatal(err)
	}
	model := models[0]
	if model.ContextLength != 272000 || model.MaxContextLength != 872000 || model.AutoCompactTokenLimit != 240000 {
		t.Fatalf("context metadata = %+v", model)
	}
	if model.DefaultReasoningEffort != "xhigh" || len(model.SupportedReasoningEfforts) != 2 || !containsString(model.SupportedReasoningEfforts, "low") || !containsString(model.SupportedReasoningEfforts, "xhigh") {
		t.Fatalf("reasoning metadata = %+v", model)
	}
	if model.SupportsVerbosity == nil || !*model.SupportsVerbosity || model.DefaultVerbosity != "low" || len(model.ServiceTiers) != 2 || model.DefaultServiceTier != "fast" {
		t.Fatalf("request metadata = %+v", model)
	}
	if model.SupportsParallelToolCalls == nil || *model.SupportsParallelToolCalls || model.UseResponsesLite == nil || !*model.UseResponsesLite || model.SupportsImageDetailOriginal == nil || !*model.SupportsImageDetailOriginal {
		t.Fatalf("wire metadata = %+v", model)
	}
	if model.WebSearchToolType != "text_and_image" || !declaredModelCapabilitySupported(model, codexWebSearchCapability) || len(model.Modalities) != 2 {
		t.Fatalf("search/modalities = %+v", model)
	}
}

func TestParseCodexModelsAcceptsCurrentShellAliasAndResolvedContext(t *testing.T) {
	models, _, err := parseCodexModels([]byte(`{"models":[{"slug":"gpt-alias","visibility":"list","max_context_window":372000,"shell_type":"shell_command","web_search_tool_type":"text","use_responses_lite":false}]}`), "openai")
	if err != nil || len(models) != 1 {
		t.Fatalf("models = %#v err = %v", models, err)
	}
	model := models[0]
	if model.ContextLength != 372000 || !declaredModelCapabilitySupported(model, codexShellCapability) {
		t.Fatalf("model = %#v", model)
	}
}

func TestParseCodexModelsUnknownRuntimeEnumsFailClosed(t *testing.T) {
	models, _, err := parseCodexModels([]byte(`{"models":[{"slug":"future","visibility":"list","shell_type":"unified_exec","web_search_tool_type":"future_search","supports_search_tool":true,"default_reasoning_level":"future","default_verbosity":"future"}]}`), "openai")
	if err != nil {
		t.Fatal(err)
	}
	model := models[0]
	if !model.WebSearchToolTypeKnown || declaredModelCapabilitySupported(model, codexWebSearchCapability) || model.DefaultReasoningEffort != "" || model.DefaultVerbosity != "" {
		t.Fatalf("future enum metadata = %+v, want known search denial and inert request values", model)
	}
}

func TestParseCodexModelsSearchBooleanCanExplicitlyDisableKnownWireType(t *testing.T) {
	models, _, err := parseCodexModels([]byte(`{"models":[{"slug":"no-search","visibility":"list","shell_type":"unified_exec","web_search_tool_type":"text_and_image","supports_search_tool":false}]}`), "openai")
	if err != nil || len(models) != 1 {
		t.Fatalf("models = %#v err = %v", models, err)
	}
	if !models[0].WebSearchToolTypeKnown || declaredModelCapabilitySupported(models[0], codexWebSearchCapability) {
		t.Fatalf("model = %#v", models[0])
	}
}

func TestEnsureDynamicProviderCapabilitiesRefreshesCodexOAuthWithoutUserAction(t *testing.T) {
	originalClient := providerModelHTTPClient
	defer func() { providerModelHTTPClient = originalClient }()
	requests := 0
	providerModelHTTPClient = &http.Client{Transport: providerModelRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.String() != chatGPTCodexModelsURL {
			t.Fatalf("URL = %q, want %q", req.URL.String(), chatGPTCodexModelsURL)
		}
		if req.Header.Get("Authorization") != "Bearer oauth-token" || req.Header.Get("ChatGPT-Account-Id") != "acct-1" {
			t.Fatalf("OAuth headers = %#v", req.Header)
		}
		body := `{"models":[{"slug":"gpt-codex-dynamic","visibility":"list","shell_type":"unified_exec","web_search_tool_type":"text"}]}`
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	})}

	store := &memoryProviderStore{auth: map[string]domain.ProviderAuthRecord{
		"openai": {ProviderID: "openai", Method: "oauth-browser", AccessToken: "oauth-token", ExpiresAt: "2099-01-01T00:00:00Z", AccountID: "acct-1"},
	}}
	service := NewService(store)
	defer service.Shutdown()
	route := ResolvedModelRoute{
		Provider:   domain.ProviderConfig{ID: "openai", Type: string(TransportOpenAIResponses), BaseURL: "https://api.openai.com/v1"},
		Model:      domain.ModelRef{ProviderID: "openai", ModelID: "gpt-codex-dynamic"},
		Definition: ProviderDefinition{ID: "openai", DisplayName: "OpenAI", BuiltIn: true, Transport: TransportOpenAIResponses, ModelFetch: ModelFetchOpenAICompatible},
		Transport:  TransportOpenAIResponses,
		Credential: llmCredential{Method: "oauth-browser"},
	}

	service.ensureDynamicProviderCapabilities(context.Background(), route)
	service.ensureDynamicProviderCapabilities(context.Background(), route)
	if requests != 1 {
		t.Fatalf("Codex catalog requests = %d, want one automatic sync", requests)
	}
	if store.savedCache == nil || store.savedCache.ParserType != "openai-codex" || store.savedCache.Endpoint != chatGPTCodexModelsURL {
		t.Fatalf("saved Codex cache = %+v, want route-specific parser and endpoint provenance", store.savedCache)
	}
	model, ok := service.modelInfoForRoute(context.Background(), route)
	if !ok || !declaredModelCapabilitySupported(model, codexShellCapability) || !declaredModelCapabilitySupported(model, codexWebSearchCapability) {
		t.Fatalf("model = %+v ok=%t, want persisted Codex declarations", model, ok)
	}
}

func TestProviderDeclaredCapabilityParsers(t *testing.T) {
	tests := []struct {
		name       string
		parse      func([]byte, string) ([]domain.ModelInfo, string, error)
		raw        string
		providerID string
		modelID    string
		wantTools  bool
		wantReason bool
		wantStream bool
	}{
		{
			name: "mistral", parse: parseMistralModels, providerID: "mistral", modelID: "mistral-declared",
			raw:       `{"data":[{"id":"mistral-declared","name":"Mistral Declared","max_context_length":131072,"capabilities":{"function_calling":true,"vision":false}}]}`,
			wantTools: true,
		},
		{
			name: "openrouter", parse: parseOpenRouterModels, providerID: "openrouter", modelID: "vendor/model",
			raw:       `{"data":[{"id":"vendor/model","name":"Vendor Model","context_length":200000,"supported_parameters":["tools","reasoning"],"top_provider":{"max_completion_tokens":16000}}]}`,
			wantTools: true, wantReason: true,
		},
		{
			name: "cerebras", parse: parseCerebrasModels, providerID: "cerebras", modelID: "cerebras-declared",
			raw:       `{"data":[{"id":"cerebras-declared","name":"Cerebras Declared","capabilities":{"function_calling":true,"streaming":true,"reasoning":false},"limits":{"max_context_length":128000,"max_completion_tokens":8192}}]}`,
			wantTools: true, wantStream: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			models, _, err := test.parse([]byte(test.raw), test.providerID)
			if err != nil {
				t.Fatal(err)
			}
			if len(models) != 1 || models[0].ID != test.modelID {
				t.Fatalf("models = %+v", models)
			}
			model := models[0]
			if modelSupportsCapability(model, "tools") != test.wantTools || modelSupportsCapability(model, "reasoning") != test.wantReason || modelSupportsCapability(model, "streaming") != test.wantStream {
				t.Fatalf("capabilities = %+v declared=%+v", model.Capabilities, model.DeclaredCapabilities)
			}
			if len(model.DeclaredCapabilities) == 0 {
				t.Fatal("provider declarations were not retained")
			}
		})
	}
}

func TestDeclaredBooleanTreatsNullAndMalformedValuesAsUnknown(t *testing.T) {
	for _, raw := range []string{"null", `{}`, `{"supported":null}`, `"true"`} {
		value, declared := declaredBoolean(map[string]json.RawMessage{"function_calling": json.RawMessage(raw)}, "function_calling")
		if declared || value {
			t.Fatalf("raw %s = value=%t declared=%t, want unknown", raw, value, declared)
		}
	}
}

func TestAnthropicEffortCapabilityUsesDeclaredLevels(t *testing.T) {
	raw := map[string]json.RawMessage{
		"effort": json.RawMessage(`{"low":{"supported":true},"high":{"supported":false}}`),
	}
	capabilities, _, controls, _ := parseAnthropicModelCapabilities(raw)
	if !containsString(capabilities, "reasoning") || len(controls) == 0 {
		t.Fatalf("capabilities = %#v controls=%#v, want declared reasoning effort", capabilities, controls)
	}
	if declared := anthropicDeclaredCapabilityDimensions(map[string]json.RawMessage{"thinking": json.RawMessage("null")}); len(declared) != 0 {
		t.Fatalf("null Anthropic declaration = %#v, want unknown", declared)
	}
}

func TestEnsureDynamicProviderCapabilitiesRefreshesMistralWithoutUserAction(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/models" {
			t.Fatalf("path = %q, want /models", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"mistral-dynamic","capabilities":{"function_calling":false}}]}`))
	}))
	defer server.Close()

	store := &memoryProviderStore{auth: map[string]domain.ProviderAuthRecord{
		"mistral": {ProviderID: "mistral", Method: "api-key", APIKey: "mistral-key"},
	}}
	service := NewService(store)
	defer service.Shutdown()
	route := ResolvedModelRoute{
		Provider: domain.ProviderConfig{ID: "mistral", Type: string(TransportOpenAICompatible), BaseURL: server.URL},
		Model:    domain.ModelRef{ProviderID: "mistral", ModelID: "mistral-dynamic"},
		Definition: ProviderDefinition{
			ID: "mistral", DisplayName: "Mistral", Transport: TransportOpenAICompatible, ModelFetch: ModelFetchMistral,
			Models: []domain.ModelInfo{{ID: "mistral-dynamic", ProviderID: "mistral", ToolSupport: true, Capabilities: []string{"tools", "streaming"}}},
		},
		Transport: TransportOpenAICompatible,
	}

	service.ensureDynamicProviderCapabilities(context.Background(), route)
	service.ensureDynamicProviderCapabilities(context.Background(), route)
	if requests != 1 {
		t.Fatalf("model catalog requests = %d, want one automatic sync", requests)
	}
	model, ok := service.modelInfoForRoute(context.Background(), route)
	if !ok || !containsString(model.DeclaredCapabilities, "tools") || modelSupportsCapability(model, "tools") {
		t.Fatalf("model = %+v ok=%t, want returned tools=false to override static tools=true", model, ok)
	}
	if !modelSupportsCapability(model, "streaming") {
		t.Fatalf("model = %+v, want undeclared static streaming metadata preserved", model)
	}
}

func TestCerebrasCatalogUsesCodeOwnedPublicCapabilityEndpoint(t *testing.T) {
	definition, ok := providerDefinition("cerebras")
	if !ok {
		t.Fatal("cerebras definition missing")
	}
	definition.DefaultBaseURL = "https://renderer-selected.invalid/v1"
	if got := modelEndpointForDefinition(definition); got != cerebrasPublicModelsURL {
		t.Fatalf("model endpoint = %q, want fixed %q", got, cerebrasPublicModelsURL)
	}
	definition.BuiltIn = false
	if got := modelEndpointForDefinition(definition); got == cerebrasPublicModelsURL {
		t.Fatalf("custom definition selected privileged public endpoint %q", got)
	}
}

func TestFetchAnthropicModelsUsesAnthropicAuthHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Fatalf("path = %q, want /models", r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "1000" {
			t.Fatalf("limit = %q, want 1000", got)
		}
		if got := r.Header.Get("x-api-key"); got != "anthropic-key" {
			t.Fatalf("x-api-key = %q, want anthropic-key", got)
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %q, want empty", got)
		}
		if got := r.Header.Get("anthropic-version"); got == "" {
			t.Fatal("anthropic-version header is empty")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"claude-sonnet-4-5","display_name":"Claude Sonnet 4.5","max_input_tokens":200000,"max_tokens":64000,"capabilities":{"code_execution":{"supported":true},"delete_files":{"supported":true},"thinking":{"supported":true},"image_input":{"supported":true},"web_search":{"supported":false}}}]}`))
	}))
	defer server.Close()

	models, defaultModel, err := fetchAnthropicModels(
		context.Background(),
		domain.ProviderConfig{ID: "anthropic", Type: "anthropic", BaseURL: server.URL},
		llmCredential{APIKey: "anthropic-key"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if defaultModel != "claude-sonnet-4-5" {
		t.Fatalf("defaultModel = %q, want claude-sonnet-4-5", defaultModel)
	}
	if len(models) != 1 || models[0].ID != "claude-sonnet-4-5" || !models[0].Recommended {
		t.Fatalf("models = %+v", models)
	}
	if !models[0].NativeToolsKnown || len(models[0].NativeTools) != 1 || models[0].NativeTools[0] != "code_execution" {
		t.Fatalf("native tools = %#v known=%t, want discovered code_execution", models[0].NativeTools, models[0].NativeToolsKnown)
	}
	if !containsString(models[0].DeclaredCapabilities, "reasoning") || !containsString(models[0].DeclaredCapabilities, "vision") || !containsString(models[0].DeclaredCapabilities, "web_search") {
		t.Fatalf("declared capabilities = %#v, want true and false returned dimensions retained", models[0].DeclaredCapabilities)
	}
	if models[0].ContextLength != 200000 || models[0].OutputLimit != 64000 || !modelSupportsCapability(models[0], "reasoning") || !modelSupportsCapability(models[0], "vision") {
		t.Fatalf("model metadata = %+v, want returned limits/reasoning/vision", models[0])
	}
}

func TestEnsureDynamicProviderCapabilitiesRefreshesExpiredMetadata(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		_, _ = w.Write([]byte(`{"data":[{"id":"claude-dynamic","display_name":"Claude Dynamic","capabilities":{"code_execution":{"supported":true}}}]}`))
	}))
	defer server.Close()

	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	store := &memoryProviderStore{
		auth: map[string]domain.ProviderAuthRecord{
			"anthropic": {ProviderID: "anthropic", Method: "api-key", APIKey: "anthropic-key"},
		},
		modelCaches: map[string]domain.ProviderModelCache{
			"anthropic": {
				ProviderID:  "anthropic",
				RefreshedAt: domain.NowString(now.Add(-dynamicProviderCapabilityCacheTTL)),
				Models:      []domain.ModelInfo{{ID: "claude-dynamic", ProviderID: "anthropic", NativeToolsKnown: true}},
			},
		},
	}
	service := NewService(store)
	service.now = func() time.Time { return now }
	defer service.Shutdown()
	route := ResolvedModelRoute{
		Provider:   domain.ProviderConfig{ID: "anthropic", Type: string(TransportAnthropicMessages), BaseURL: server.URL},
		Model:      domain.ModelRef{ProviderID: "anthropic", ModelID: "claude-dynamic"},
		Definition: ProviderDefinition{ID: "anthropic", DisplayName: "Anthropic", Transport: TransportAnthropicMessages, ModelFetch: ModelFetchAnthropic},
		Transport:  TransportAnthropicMessages,
	}

	service.ensureDynamicProviderCapabilities(context.Background(), route)
	if requests != 1 {
		t.Fatalf("model catalog requests = %d, want expired metadata refreshed", requests)
	}
	model, ok := service.modelInfoForRoute(context.Background(), route)
	if !ok || len(model.NativeTools) != 1 || model.NativeTools[0] != "code_execution" {
		t.Fatalf("model = %+v ok=%t, want refreshed code_execution capability", model, ok)
	}
}

func TestEnsureDynamicProviderCapabilitiesPreservesCacheOnRefreshFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	store := &memoryProviderStore{
		auth: map[string]domain.ProviderAuthRecord{
			"anthropic": {ProviderID: "anthropic", Method: "api-key", APIKey: "anthropic-key"},
		},
		modelCaches: map[string]domain.ProviderModelCache{
			"anthropic": {
				ProviderID:  "anthropic",
				RefreshedAt: domain.NowString(now.Add(-dynamicProviderCapabilityCacheTTL)),
				Models: []domain.ModelInfo{{
					ID: "claude-dynamic", ProviderID: "anthropic", NativeToolsKnown: true, NativeTools: []string{"code_execution"},
				}},
			},
		},
	}
	service := NewService(store)
	service.now = func() time.Time { return now }
	defer service.Shutdown()
	route := ResolvedModelRoute{
		Provider:   domain.ProviderConfig{ID: "anthropic", Type: string(TransportAnthropicMessages), BaseURL: server.URL},
		Model:      domain.ModelRef{ProviderID: "anthropic", ModelID: "claude-dynamic"},
		Definition: ProviderDefinition{ID: "anthropic", DisplayName: "Anthropic", Transport: TransportAnthropicMessages, ModelFetch: ModelFetchAnthropic},
		Transport:  TransportAnthropicMessages,
	}

	service.ensureDynamicProviderCapabilities(context.Background(), route)
	if store.savedCache != nil {
		t.Fatalf("failed automatic refresh saved cache = %+v, want prior cache preserved", store.savedCache)
	}
	model, ok := service.modelInfoForRoute(context.Background(), route)
	if !ok || !model.NativeToolsKnown || len(model.NativeTools) != 1 || model.NativeTools[0] != "code_execution" {
		t.Fatalf("model = %+v ok=%t, want last authoritative cache preserved", model, ok)
	}
}

func TestEnsureDynamicProviderCapabilitiesRefreshesAnthropicOnceWithoutUserAction(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/models" {
			t.Fatalf("path = %q, want /models", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"claude-dynamic","display_name":"Claude Dynamic","capabilities":{"code_execution":{"supported":true}}}]}`))
	}))
	defer server.Close()

	store := &memoryProviderStore{auth: map[string]domain.ProviderAuthRecord{
		"anthropic": {ProviderID: "anthropic", Method: "api-key", APIKey: "anthropic-key"},
	}}
	service := NewService(store)
	defer service.Shutdown()
	route := ResolvedModelRoute{
		Provider:   domain.ProviderConfig{ID: "anthropic", Type: string(TransportAnthropicMessages), BaseURL: server.URL},
		Model:      domain.ModelRef{ProviderID: "anthropic", ModelID: "claude-dynamic"},
		Definition: ProviderDefinition{ID: "anthropic", DisplayName: "Anthropic", Transport: TransportAnthropicMessages, ModelFetch: ModelFetchAnthropic},
		Transport:  TransportAnthropicMessages,
	}

	service.ensureDynamicProviderCapabilities(context.Background(), route)
	service.ensureDynamicProviderCapabilities(context.Background(), route)
	if requests != 1 {
		t.Fatalf("model catalog requests = %d, want one automatic sync", requests)
	}
	model, ok := service.modelInfoForRoute(context.Background(), route)
	if !ok || !model.NativeToolsKnown || len(model.NativeTools) != 1 || model.NativeTools[0] != "code_execution" {
		t.Fatalf("model = %+v ok=%t, want persisted dynamic native tool metadata", model, ok)
	}
}

func TestFetchOpenAICompatibleModelsUsesAzureAPIKeyHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/openai/v1/models" {
			t.Fatalf("path = %q, want /openai/v1/models", r.URL.Path)
		}
		if got := r.Header.Get("api-key"); got != "azure-key" {
			t.Fatalf("api-key = %q, want azure-key", got)
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %q, want empty", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-5.5","name":"GPT-5.5"}]}`))
	}))
	defer server.Close()

	models, defaultModel, err := fetchOpenAICompatibleModels(
		context.Background(),
		domain.ProviderConfig{ID: "azure-openai", Type: string(TransportAzureOpenAI), BaseURL: server.URL + "/openai/v1"},
		llmCredential{APIKey: "azure-key"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if defaultModel != "gpt-5.5" {
		t.Fatalf("defaultModel = %q, want gpt-5.5", defaultModel)
	}
	if len(models) != 1 || models[0].ID != "gpt-5.5" || !models[0].Recommended {
		t.Fatalf("models = %+v", models)
	}
}

func TestRefreshProviderModelsUsesPersistedProviderConfigForIDOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("path = %q, want /v1/models", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer stored-key" {
			t.Fatalf("Authorization = %q, want stored credential", got)
		}
		if got := r.Header.Get("X-Team"); got != "aivo" {
			t.Fatalf("X-Team = %q, want persisted header", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"new-default","name":"New Default"},{"id":"new-secondary","name":"New Secondary"}]}`))
	}))
	defer server.Close()

	store := &memoryProviderStore{
		config: &domain.AppConfig{DefaultModel: &domain.ModelRef{
			ProviderID: "team-provider",
			ModelID:    "active-model",
		}},
		providers: []domain.ProviderConfig{{
			ID: "team-provider", Type: string(TransportOpenAICompatible), BaseURL: server.URL + "/v1",
			Model: "old-default", Headers: map[string]string{"X-Team": "aivo"},
		}},
		auth: map[string]domain.ProviderAuthRecord{
			"team-provider": {ProviderID: "team-provider", Method: "api-key", APIKey: "stored-key"},
		},
	}
	service := NewService(store)
	defer service.Shutdown()

	catalog, err := service.RefreshProviderModels(context.Background(), domain.ProviderConnectInput{
		ProviderID: "team-provider",
		Name:       "Team Provider",
	})
	if err != nil {
		t.Fatal(err)
	}
	var refreshed *domain.ProviderInfo
	for i := range catalog.Providers {
		if catalog.Providers[i].ID == "team-provider" {
			refreshed = &catalog.Providers[i]
			break
		}
	}
	if refreshed == nil {
		t.Fatal("refreshed provider missing from catalog")
	}
	if refreshed.DefaultModelID != "new-default" || len(refreshed.Models) != 2 || refreshed.Models[0].ID != "new-default" {
		t.Fatalf("refreshed provider = %+v, want replacement model list/default", refreshed)
	}
	if store.savedCache == nil || store.savedCache.DefaultModel != "new-default" || len(store.savedCache.Models) != 2 {
		t.Fatalf("saved cache = %+v, want refreshed models", store.savedCache)
	}
	if store.config == nil || store.config.DefaultModel == nil || store.config.DefaultModel.ModelID != "active-model" {
		t.Fatalf("app config = %+v, want active model preference preserved", store.config)
	}
}

func TestRefreshProviderModelsPreservesExistingCacheWhenEndpointIsUnsupported(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	existing := domain.ProviderModelCache{
		ProviderID: "catalog-provider",
		Models: []domain.ModelInfo{{
			ID: "known-model", ProviderID: "catalog-provider", Name: "Known Model",
		}},
		DefaultModel: "known-model",
		CacheSource:  "ecosystem",
	}
	store := &memoryProviderStore{
		providers: []domain.ProviderConfig{{
			ID: "catalog-provider", Type: string(TransportOpenAICompatible), BaseURL: server.URL,
			Model: "known-model",
		}},
		auth: map[string]domain.ProviderAuthRecord{
			"catalog-provider": {ProviderID: "catalog-provider", Method: "api-key", APIKey: "stored-key"},
		},
		modelCaches: map[string]domain.ProviderModelCache{"catalog-provider": existing},
	}
	service := NewService(store)
	defer service.Shutdown()

	_, err := service.RefreshProviderModels(context.Background(), domain.ProviderConnectInput{
		ProviderID: "catalog-provider",
		Name:       "Catalog Provider",
	})
	if err == nil || !strings.Contains(err.Error(), "model endpoint is not supported") {
		t.Fatalf("RefreshProviderModels error = %v, want unsupported endpoint", err)
	}
	if store.savedCache != nil {
		t.Fatalf("failed refresh saved cache = %+v, want no replacement", store.savedCache)
	}
	preserved := store.modelCaches["catalog-provider"]
	if preserved.DefaultModel != "known-model" || len(preserved.Models) != 1 || preserved.Models[0].ID != "known-model" {
		t.Fatalf("preserved cache = %+v, want previous catalog", preserved)
	}
}
