package app

import (
	"context"
	"testing"

	"aivo/core/domain"
)

func TestToolsForModelRouteUsesHostedWebSearchForOpenAIResponses(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	route := ResolvedModelRoute{
		Provider: domain.ProviderConfig{ID: "openai"},
		Model:    domain.ModelRef{ProviderID: "openai", ModelID: "gpt-test"},
		Definition: ProviderDefinition{
			ID:        "openai",
			Transport: TransportOpenAIResponses,
			Models:    []domain.ModelInfo{model("openai", "gpt-test", "GPT Test", true, 1000, []string{"tools", "streaming", "web_search"})},
		},
		Transport: TransportOpenAIResponses,
	}
	specs := []domain.ToolSpec{{Name: "web_search", Description: "Search", InputSchema: map[string]any{"type": "object"}}}

	tools := service.toolsForModelRoute(context.Background(), domain.AppConfig{WebSearch: domain.WebSearchConfig{Mode: domain.WebSearchModeLive, Route: domain.WebSearchRouteAuto, SearchContextSize: "low"}}, route, specs)
	if len(tools) != 1 || tools[0].Hosted == nil || tools[0].Hosted.Type != "web_search" {
		t.Fatalf("tools = %#v, want hosted web_search", tools)
	}
	if tools[0].Hosted.SearchContextSize != "low" {
		t.Fatalf("hosted size = %q, want low", tools[0].Hosted.SearchContextSize)
	}
}

func TestToolsForModelRouteUsesDeclaredHostedWebSearchForCompatibleProvider(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	route := ResolvedModelRoute{
		Provider: domain.ProviderConfig{ID: "search-compatible"},
		Model:    domain.ModelRef{ProviderID: "search-compatible", ModelID: "search-model"},
		Definition: ProviderDefinition{
			ID:                "search-compatible",
			Transport:         TransportOpenAICompatible,
			NativeHostedTools: nativeWebSearch("web_search"),
			Models:            []domain.ModelInfo{model("search-compatible", "search-model", "Search Model", true, 1000, []string{"tools", "streaming", "web_search"})},
		},
		Transport: TransportOpenAICompatible,
	}
	specs := []domain.ToolSpec{{Name: "web_search", Description: "Search", InputSchema: map[string]any{"type": "object"}}}

	tools := service.toolsForModelRoute(context.Background(), domain.AppConfig{WebSearch: domain.WebSearchConfig{Mode: domain.WebSearchModeLive, Route: domain.WebSearchRouteAuto}}, route, specs)
	if len(tools) != 1 || tools[0].Hosted == nil || tools[0].Hosted.Type != "web_search" {
		t.Fatalf("tools = %#v, want data-driven hosted web_search", tools)
	}
}

func TestToolsForModelRouteUsesPoeHostedWebSearchPreview(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	def, ok := providerDefinition("poe")
	if !ok {
		t.Fatal("poe provider definition missing")
	}
	route := ResolvedModelRoute{
		Provider:   domain.ProviderConfig{ID: "poe"},
		Model:      domain.ModelRef{ProviderID: "poe", ModelID: "GPT-5.4"},
		Definition: def,
		Transport:  def.Transport,
	}
	specs := []domain.ToolSpec{{Name: "web_search", Description: "Search", InputSchema: map[string]any{"type": "object"}}}

	tools := service.toolsForModelRoute(context.Background(), domain.AppConfig{WebSearch: domain.WebSearchConfig{Mode: domain.WebSearchModeLive, Route: domain.WebSearchRouteAuto}}, route, specs)
	if len(tools) != 1 || tools[0].Hosted == nil || tools[0].Hosted.Type != "web_search_preview" {
		t.Fatalf("tools = %#v, want Poe hosted web_search_preview", tools)
	}
	if !providerUsesResponsesAPI(domain.ProviderConfig{ID: "poe"}, tools) {
		t.Fatalf("Poe hosted web_search_preview should route through Responses API")
	}
}

func TestToolsForModelRouteUsesPerplexityAgentHostedWebSearch(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	def, ok := providerDefinition("perplexity-agent")
	if !ok {
		t.Fatal("perplexity-agent provider definition missing")
	}
	route := ResolvedModelRoute{
		Provider:   domain.ProviderConfig{ID: "perplexity-agent"},
		Model:      domain.ModelRef{ProviderID: "perplexity-agent", ModelID: "openai/gpt-5.6-terra"},
		Definition: def,
		Transport:  def.Transport,
	}
	specs := []domain.ToolSpec{{Name: "web_search", Description: "Search", InputSchema: map[string]any{"type": "object"}}}

	tools := service.toolsForModelRoute(context.Background(), domain.AppConfig{WebSearch: domain.WebSearchConfig{Mode: domain.WebSearchModeLive, Route: domain.WebSearchRouteAuto}}, route, specs)
	if len(tools) != 1 || tools[0].Hosted == nil || tools[0].Hosted.Type != "web_search" {
		t.Fatalf("tools = %#v, want Perplexity Agent hosted web_search", tools)
	}
	if !providerUsesResponsesAPI(domain.ProviderConfig{ID: "perplexity-agent"}, tools) {
		t.Fatalf("Perplexity Agent hosted web_search should route through Responses API")
	}
}

func TestToolsForModelRouteUsesRequestyHostedWebSearchWithoutResponsesSwitch(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	def, ok := providerDefinition("requesty")
	if !ok {
		t.Fatal("requesty provider definition missing")
	}
	route := ResolvedModelRoute{
		Provider:   domain.ProviderConfig{ID: "requesty"},
		Model:      domain.ModelRef{ProviderID: "requesty", ModelID: "anthropic/claude-sonnet-4-20250514"},
		Definition: def,
		Transport:  def.Transport,
	}
	specs := []domain.ToolSpec{{Name: "web_search", Description: "Search", InputSchema: map[string]any{"type": "object"}}}

	tools := service.toolsForModelRoute(context.Background(), domain.AppConfig{WebSearch: domain.WebSearchConfig{Mode: domain.WebSearchModeLive, Route: domain.WebSearchRouteAuto}}, route, specs)
	if len(tools) != 1 || tools[0].Hosted == nil || tools[0].Hosted.Type != "web_search" {
		t.Fatalf("tools = %#v, want Requesty hosted web_search", tools)
	}
	if providerUsesResponsesAPI(domain.ProviderConfig{ID: "requesty"}, tools) {
		t.Fatalf("Requesty chat-completions web_search should not force Responses API")
	}
}

func TestToolsForModelRouteUsesVeniceHostedWebSearchParameters(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	def, ok := providerDefinition("venice")
	if !ok {
		t.Fatal("venice provider definition missing")
	}
	route := ResolvedModelRoute{
		Provider:   domain.ProviderConfig{ID: "venice"},
		Model:      domain.ModelRef{ProviderID: "venice", ModelID: "zai-org-glm-5"},
		Definition: def,
		Transport:  def.Transport,
	}
	specs := []domain.ToolSpec{{Name: "web_search", Description: "Search", InputSchema: map[string]any{"type": "object"}}}

	tools := service.toolsForModelRoute(context.Background(), domain.AppConfig{WebSearch: domain.WebSearchConfig{Mode: domain.WebSearchModeLive, Route: domain.WebSearchRouteAuto}}, route, specs)
	if len(tools) != 1 || tools[0].Hosted == nil || tools[0].Hosted.Type != "venice_web_search" {
		t.Fatalf("tools = %#v, want Venice provider-specific hosted web search", tools)
	}
}

func TestToolsForModelRouteUsesHostedWebSearchForAnthropic(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	route := ResolvedModelRoute{
		Provider: domain.ProviderConfig{ID: "anthropic"},
		Model:    domain.ModelRef{ProviderID: "anthropic", ModelID: "claude-test"},
		Definition: ProviderDefinition{
			ID:        "anthropic",
			Transport: TransportAnthropicMessages,
			Models:    []domain.ModelInfo{model("anthropic", "claude-test", "Claude Test", true, 1000, []string{"tools", "streaming", "web_search"})},
		},
		Transport: TransportAnthropicMessages,
	}
	specs := []domain.ToolSpec{{Name: "web_search", Description: "Search", InputSchema: map[string]any{"type": "object"}}}

	tools := service.toolsForModelRoute(context.Background(), domain.AppConfig{WebSearch: domain.WebSearchConfig{Mode: domain.WebSearchModeLive, Route: domain.WebSearchRouteAuto}}, route, specs)
	if len(tools) != 1 || tools[0].Hosted == nil || tools[0].Hosted.Type != "web_search_20250305" {
		t.Fatalf("tools = %#v, want anthropic hosted web_search", tools)
	}
}

func TestToolsForModelRouteUsesHostedWebFetchForAnthropic(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	route := ResolvedModelRoute{
		Provider: domain.ProviderConfig{ID: "anthropic"},
		Model:    domain.ModelRef{ProviderID: "anthropic", ModelID: "claude-test"},
		Definition: ProviderDefinition{
			ID:        "anthropic",
			Transport: TransportAnthropicMessages,
			Models:    []domain.ModelInfo{model("anthropic", "claude-test", "Claude Test", true, 1000, []string{"tools", "streaming", "web_fetch"})},
		},
		Transport: TransportAnthropicMessages,
	}
	specs := []domain.ToolSpec{{Name: "web_fetch", Description: "Fetch", InputSchema: map[string]any{"type": "object"}}}

	tools := service.toolsForModelRoute(context.Background(), domain.AppConfig{}, route, specs)
	if len(tools) != 1 || tools[0].Hosted == nil || tools[0].Hosted.Type != "web_fetch_20250910" {
		t.Fatalf("tools = %#v, want anthropic hosted web_fetch", tools)
	}
}

func TestToolsForModelRouteAutomaticallyBridgesDiscoveredAnthropicCodeExecution(t *testing.T) {
	store := &memoryProviderStore{modelCaches: map[string]domain.ProviderModelCache{
		"anthropic": {ProviderID: "anthropic", Models: []domain.ModelInfo{{
			ID: "claude-dynamic", ProviderID: "anthropic", Name: "Claude Dynamic",
			Capabilities: []string{"tools", "streaming", "code_execution"},
			NativeTools:  []string{"code_execution"}, NativeToolsKnown: true, ToolSupport: true, Streaming: true,
		}}},
	}}
	service := NewService(store)
	defer service.Shutdown()
	route := ResolvedModelRoute{
		Provider:   domain.ProviderConfig{ID: "anthropic"},
		Model:      domain.ModelRef{ProviderID: "anthropic", ModelID: "claude-dynamic"},
		Definition: ProviderDefinition{ID: "anthropic", Transport: TransportAnthropicMessages},
		Transport:  TransportAnthropicMessages,
	}

	tools := service.toolsForModelRoute(context.Background(), domain.AppConfig{}, route, nil)
	if len(tools) != 1 || tools[0].Name != "code_execution" || tools[0].Hosted == nil || tools[0].Hosted.Type != "code_execution_20250825" {
		t.Fatalf("tools = %#v, want automatically bridged Anthropic code execution", tools)
	}
}

func TestToolsForModelRouteDoesNotBridgeUnknownOrDisabledDynamicTools(t *testing.T) {
	for _, tt := range []struct {
		name     string
		model    domain.ModelInfo
		disabled []string
	}{
		{name: "unknown metadata", model: domain.ModelInfo{ID: "claude-dynamic", ProviderID: "anthropic", Capabilities: []string{"code_execution"}}},
		{name: "explicitly disabled", model: domain.ModelInfo{ID: "claude-dynamic", ProviderID: "anthropic", NativeToolsKnown: true, NativeTools: []string{"code_execution"}}, disabled: []string{"code_execution"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(&memoryProviderStore{modelCaches: map[string]domain.ProviderModelCache{
				"anthropic": {ProviderID: "anthropic", Models: []domain.ModelInfo{tt.model}},
			}})
			defer service.Shutdown()
			route := ResolvedModelRoute{
				Provider: domain.ProviderConfig{ID: "anthropic"}, Model: domain.ModelRef{ProviderID: "anthropic", ModelID: "claude-dynamic"},
				Definition: ProviderDefinition{ID: "anthropic", Transport: TransportAnthropicMessages}, Transport: TransportAnthropicMessages,
			}
			tools := service.toolsForModelRoute(context.Background(), domain.AppConfig{NativeTools: domain.NativeToolsConfig{Disabled: tt.disabled}}, route, nil)
			if len(tools) != 0 {
				t.Fatalf("tools = %#v, want no automatic native tools", tools)
			}
		})
	}
}

func TestToolsForModelRouteDoesNotTurnGenericDeclarationIntoHostedTool(t *testing.T) {
	service := NewService(&memoryProviderStore{modelCaches: map[string]domain.ProviderModelCache{
		"openrouter": {ProviderID: "openrouter", Models: []domain.ModelInfo{{
			ID: "vendor/model", ProviderID: "openrouter", DeclaredCapabilities: []string{"tools"},
			Capabilities: []string{"tools"}, ToolSupport: true,
		}}},
	}})
	defer service.Shutdown()
	route := ResolvedModelRoute{
		Provider: domain.ProviderConfig{ID: "openrouter"},
		Model:    domain.ModelRef{ProviderID: "openrouter", ModelID: "vendor/model"},
		Definition: ProviderDefinition{
			ID: "openrouter", Transport: TransportOpenAICompatible, ModelFetch: ModelFetchOpenRouter,
		},
		Transport: TransportOpenAICompatible,
	}

	tools := service.toolsForModelRoute(context.Background(), domain.AppConfig{}, route, nil)
	if len(tools) != 0 {
		t.Fatalf("tools = %#v, generic function-calling support must not create hosted tools", tools)
	}
}

func TestToolsForModelRouteBridgesCodexShellOnlyForOAuth(t *testing.T) {
	store := &memoryProviderStore{modelCaches: map[string]domain.ProviderModelCache{
		"openai": {ProviderID: "openai", Models: []domain.ModelInfo{{
			ID: "gpt-codex", ProviderID: "openai",
			DeclaredCapabilities: []string{codexShellCapability},
			Capabilities:         []string{codexShellCapability},
		}}},
	}}
	service := NewService(store)
	defer service.Shutdown()
	definition := ProviderDefinition{ID: "openai", BuiltIn: true, Transport: TransportOpenAIResponses}
	specs := []domain.ToolSpec{
		NewReadTool("").Spec(), NewExecCommandTool("", nil, nil).Spec(),
	}
	oauthRoute := ResolvedModelRoute{
		Provider: domain.ProviderConfig{ID: "openai"}, Model: domain.ModelRef{ProviderID: "openai", ModelID: "gpt-codex"},
		Definition: definition, Transport: TransportOpenAIResponses, Credential: llmCredential{Method: "oauth-browser"},
	}
	oauthTools := service.toolsForModelRoute(context.Background(), domain.AppConfig{}, oauthRoute, specs)
	if !toolSpecNamed(oauthTools, "read") || !toolSpecNamed(oauthTools, ExecCommandToolName) {
		t.Fatalf("OAuth Codex tools = %#v, want read plus declared shell", oauthTools)
	}
	apiKeyRoute := oauthRoute
	apiKeyRoute.Credential = llmCredential{Method: "api-key", APIKey: "key"}
	apiKeyTools := service.toolsForModelRoute(context.Background(), domain.AppConfig{}, apiKeyRoute, specs)
	if !toolSpecNamed(apiKeyTools, ExecCommandToolName) {
		t.Fatalf("API-key tools = %#v, want ordinary exec_command", apiKeyTools)
	}
}

func TestToolsForModelRouteCodexDeclarationsFailClosed(t *testing.T) {
	store := &memoryProviderStore{modelCaches: map[string]domain.ProviderModelCache{
		"openai": {ProviderID: "openai", Models: []domain.ModelInfo{{
			ID: "gpt-codex", ProviderID: "openai",
			DeclaredCapabilities: []string{codexShellCapability},
		}}},
	}}
	service := NewService(store)
	defer service.Shutdown()
	route := ResolvedModelRoute{
		Provider: domain.ProviderConfig{ID: "openai"}, Model: domain.ModelRef{ProviderID: "openai", ModelID: "gpt-codex"},
		Definition: ProviderDefinition{ID: "openai", BuiltIn: true, Transport: TransportOpenAIResponses},
		Transport:  TransportOpenAIResponses, Credential: llmCredential{Method: "oauth-browser"},
	}
	specs := []domain.ToolSpec{NewReadTool("").Spec(), NewExecCommandTool("", nil, nil).Spec()}
	tools := service.toolsForModelRoute(context.Background(), domain.AppConfig{}, route, specs)
	if !toolSpecNamed(tools, "read") || toolSpecNamed(tools, ExecCommandToolName) {
		t.Fatalf("explicitly disabled Codex tools = %#v, want only unaffected read", tools)
	}
}

func TestToolsForModelRouteUsesCodexDeclaredHostedAndLiteSearch(t *testing.T) {
	lite := false
	model := domain.ModelInfo{
		ID: "gpt-codex", ProviderID: "openai", WebSearchToolType: "text_and_image", WebSearchToolTypeKnown: true,
		UseResponsesLite: &lite, DeclaredCapabilities: []string{codexWebSearchCapability}, Capabilities: []string{codexWebSearchCapability},
	}
	store := &memoryProviderStore{modelCaches: map[string]domain.ProviderModelCache{"openai": {ProviderID: "openai", Models: []domain.ModelInfo{model}}}}
	service := NewService(store)
	defer service.Shutdown()
	route := ResolvedModelRoute{
		Provider: domain.ProviderConfig{ID: "openai"}, Model: domain.ModelRef{ProviderID: "openai", ModelID: "gpt-codex"},
		Definition: ProviderDefinition{ID: "openai", BuiltIn: true, Transport: TransportOpenAIResponses},
		Transport:  TransportOpenAIResponses, Credential: llmCredential{Method: "oauth-browser"},
	}
	specs := []domain.ToolSpec{NewCodexWebSearchTool(service).Spec()}
	config := domain.AppConfig{WebSearch: domain.WebSearchConfig{Mode: domain.WebSearchModeIndexed, Route: domain.WebSearchRouteAuto}}
	tools := service.toolsForModelRoute(context.Background(), config, route, nil)
	if len(tools) != 1 || tools[0].Hosted == nil || tools[0].Hosted.Type != "web_search" || tools[0].Hosted.IndexedWebAccess == nil || !*tools[0].Hosted.IndexedWebAccess || len(tools[0].Hosted.SearchContentTypes) != 2 {
		t.Fatalf("hosted Codex tools = %#v", tools)
	}
	lite = true
	model.UseResponsesLite = &lite
	store.modelCaches["openai"] = domain.ProviderModelCache{ProviderID: "openai", Models: []domain.ModelInfo{model}}
	tools = service.toolsForModelRoute(context.Background(), config, route, specs)
	if len(tools) != 1 || tools[0].Hosted != nil || tools[0].Name != "web_search" {
		t.Fatalf("Responses Lite tools = %#v, want local Codex search executor", tools)
	}
	config.WebSearch.Mode = domain.WebSearchModeDisabled
	if tools := service.toolsForModelRoute(context.Background(), config, route, specs); len(tools) != 0 {
		t.Fatalf("disabled search tools = %#v", tools)
	}
}

func TestToolsForModelRouteUsesURLContextForGeminiWhenSafeToCombine(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	route := ResolvedModelRoute{
		Provider: domain.ProviderConfig{ID: "gemini"},
		Model:    domain.ModelRef{ProviderID: "gemini", ModelID: "gemini-3-pro-preview"},
		Definition: ProviderDefinition{
			ID:        "gemini",
			Transport: TransportGoogleGemini,
			Models:    []domain.ModelInfo{model("gemini", "gemini-3-pro-preview", "Gemini 3 Pro", true, 1000, []string{"tools", "streaming", "web_fetch"})},
		},
		Transport: TransportGoogleGemini,
	}
	specs := []domain.ToolSpec{
		{Name: "web_fetch", Description: "Fetch", InputSchema: map[string]any{"type": "object"}},
		{Name: "read_file", Description: "Read", InputSchema: map[string]any{"type": "object"}},
	}

	tools := service.toolsForModelRoute(context.Background(), domain.AppConfig{}, route, specs)
	if len(tools) != 2 || tools[0].Hosted == nil || tools[0].Hosted.Type != "url_context" {
		t.Fatalf("tools = %#v, want url_context plus local read_file", tools)
	}
}

func TestToolsForModelRouteDoesNotImplicitlyInjectConfiguredXAINativeTools(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	route := ResolvedModelRoute{
		Provider: domain.ProviderConfig{ID: "xai"},
		Model:    domain.ModelRef{ProviderID: "xai", ModelID: "grok-test"},
		Definition: ProviderDefinition{
			ID:        "xai",
			Transport: TransportOpenAICompatible,
			Models:    []domain.ModelInfo{model("xai", "grok-test", "Grok Test", true, 1000, []string{"tools", "streaming", "x_search", "code_interpreter", "file_search", "remote_mcp"})},
		},
		Transport: TransportOpenAICompatible,
	}
	cfg := domain.AppConfig{NativeTools: domain.NativeToolsConfig{
		XSearch:       domain.NativeToolToggle{Enabled: true},
		CodeExecution: domain.NativeCodeExecutionConfig{Enabled: true, FileIDs: []string{"file_1"}},
		FileSearch:    domain.NativeFileSearchConfig{Enabled: true, VectorStoreIDs: []string{"vs_1"}},
		RemoteMCP:     []domain.NativeMCPToolConfig{{Enabled: true, ServerURL: "https://mcp.example.com", ServerLabel: "Docs", AllowedTools: []string{"search"}}},
	}}

	tools := service.toolsForModelRoute(context.Background(), cfg, route, nil)
	if len(tools) != 0 {
		t.Fatalf("tools = %#v, want hosted capabilities deferred to extensions", tools)
	}
}

func TestToolsForModelRouteDoesNotImplicitlyInjectConfiguredGeminiNativeTools(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	route := ResolvedModelRoute{
		Provider: domain.ProviderConfig{ID: "gemini"},
		Model:    domain.ModelRef{ProviderID: "gemini", ModelID: "gemini-test"},
		Definition: ProviderDefinition{
			ID:        "gemini",
			Transport: TransportGoogleGemini,
			Models:    []domain.ModelInfo{model("gemini", "gemini-test", "Gemini Test", true, 1000, []string{"tools", "streaming", "code_execution", "file_search"})},
		},
		Transport: TransportGoogleGemini,
	}
	cfg := domain.AppConfig{NativeTools: domain.NativeToolsConfig{
		CodeExecution: domain.NativeCodeExecutionConfig{Enabled: true},
		FileSearch:    domain.NativeFileSearchConfig{Enabled: true, VectorStoreIDs: []string{"fileSearchStores/store_1"}},
	}}

	tools := service.toolsForModelRoute(context.Background(), cfg, route, nil)
	if len(tools) != 0 {
		t.Fatalf("tools = %#v, want hosted capabilities deferred to extensions", tools)
	}
}

func TestToolsForModelRouteUsesAlibabaHostedWebSearchOnlyForDeclaredModels(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	def, ok := providerDefinition("alibaba")
	if !ok {
		t.Fatal("alibaba provider definition missing")
	}
	specs := []domain.ToolSpec{{Name: "web_search", Description: "Search", InputSchema: map[string]any{"type": "object"}}}
	cfg := domain.AppConfig{WebSearch: domain.WebSearchConfig{Mode: domain.WebSearchModeLive, Route: domain.WebSearchRouteAuto}}

	hostedRoute := ResolvedModelRoute{
		Provider:   domain.ProviderConfig{ID: "alibaba"},
		Model:      domain.ModelRef{ProviderID: "alibaba", ModelID: "qwen3-max"},
		Definition: def,
		Transport:  def.Transport,
	}
	hosted := service.toolsForModelRoute(context.Background(), cfg, hostedRoute, specs)
	if len(hosted) != 1 || hosted[0].Hosted == nil || hosted[0].Hosted.Type != "web_search" {
		t.Fatalf("hosted tools = %#v, want Alibaba hosted web_search", hosted)
	}

	localRoute := hostedRoute
	localRoute.Model.ModelID = "qwen3-235b-a22b"
	local := service.toolsForModelRoute(context.Background(), cfg, localRoute, specs)
	if len(local) != 1 || local[0].Hosted != nil || local[0].Name != "web_search" {
		t.Fatalf("local tools = %#v, want local web_search fallback for default Alibaba model", local)
	}
}

func TestToolsForModelRouteSkipsFileSearchWithoutVectorStore(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	route := ResolvedModelRoute{
		Provider: domain.ProviderConfig{ID: "openai"},
		Model:    domain.ModelRef{ProviderID: "openai", ModelID: "gpt-test"},
		Definition: ProviderDefinition{
			ID:        "openai",
			Transport: TransportOpenAIResponses,
			Models:    []domain.ModelInfo{model("openai", "gpt-test", "GPT Test", true, 1000, []string{"tools", "streaming", "file_search"})},
		},
		Transport: TransportOpenAIResponses,
	}

	tools := service.toolsForModelRoute(context.Background(), domain.AppConfig{NativeTools: domain.NativeToolsConfig{FileSearch: domain.NativeFileSearchConfig{Enabled: true}}}, route, nil)
	if toolSpecNamed(tools, "file_search") {
		t.Fatalf("tools = %#v, did not expect file_search without vector stores", tools)
	}
}

func TestToolsForModelRouteUsesGoogleSearchWhenSafeToCombine(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	route := ResolvedModelRoute{
		Provider: domain.ProviderConfig{ID: "google-vertex"},
		Model:    domain.ModelRef{ProviderID: "google-vertex", ModelID: "gemini-3-pro-preview"},
		Definition: ProviderDefinition{
			ID:        "google-vertex",
			Transport: TransportGoogleVertex,
			Models:    []domain.ModelInfo{model("google-vertex", "gemini-3-pro-preview", "Gemini 3 Pro", true, 1000, []string{"tools", "streaming", "web_search"})},
		},
		Transport: TransportGoogleVertex,
	}
	specs := []domain.ToolSpec{
		{Name: "web_search", Description: "Search", InputSchema: map[string]any{"type": "object"}},
		{Name: "read_file", Description: "Read", InputSchema: map[string]any{"type": "object"}},
	}

	tools := service.toolsForModelRoute(context.Background(), domain.AppConfig{WebSearch: domain.WebSearchConfig{Mode: domain.WebSearchModeLive, Route: domain.WebSearchRouteAuto}}, route, specs)
	if len(tools) != 2 || tools[0].Hosted == nil || tools[0].Hosted.Type != "google_search" {
		t.Fatalf("tools = %#v, want google_search plus local read_file", tools)
	}
}

func TestToolsForModelRouteKeepsLocalWebSearchForGeminiTwoFiveWithOtherTools(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	route := ResolvedModelRoute{
		Provider: domain.ProviderConfig{ID: "gemini"},
		Model:    domain.ModelRef{ProviderID: "gemini", ModelID: "gemini-2.5-pro"},
		Definition: ProviderDefinition{
			ID:        "gemini",
			Transport: TransportGoogleGemini,
			Models:    []domain.ModelInfo{model("gemini", "gemini-2.5-pro", "Gemini 2.5 Pro", true, 1000, []string{"tools", "streaming", "web_search"})},
		},
		Transport: TransportGoogleGemini,
	}
	specs := []domain.ToolSpec{
		{Name: "web_search", Description: "Search", InputSchema: map[string]any{"type": "object"}},
		{Name: "read_file", Description: "Read", InputSchema: map[string]any{"type": "object"}},
	}

	tools := service.toolsForModelRoute(context.Background(), domain.AppConfig{WebSearch: domain.WebSearchConfig{Mode: domain.WebSearchModeLive, Route: domain.WebSearchRouteAuto}}, route, specs)
	if len(tools) != 2 || tools[0].Hosted != nil || tools[0].Name != "web_search" {
		t.Fatalf("tools = %#v, want local web_search for Gemini 2.5 mixed tools", tools)
	}
}

func TestToolsForModelRouteKeepsLocalWebSearchForUnsupportedProvider(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	route := ResolvedModelRoute{
		Provider:   domain.ProviderConfig{ID: "anthropic"},
		Model:      domain.ModelRef{ProviderID: "anthropic", ModelID: "claude-test"},
		Definition: ProviderDefinition{ID: "anthropic", Transport: TransportAnthropicMessages},
		Transport:  TransportAnthropicMessages,
	}
	specs := []domain.ToolSpec{NewCodexWebSearchTool(service).Spec()}

	tools := service.toolsForModelRoute(context.Background(), domain.AppConfig{WebSearch: domain.WebSearchConfig{Mode: domain.WebSearchModeLive, Route: domain.WebSearchRouteAuto}}, route, specs)
	if len(tools) != 1 || tools[0].Hosted != nil || tools[0].Name != "web_search" {
		t.Fatalf("tools = %#v, want local web_search", tools)
	}
}

func TestToolsForModelRouteKeepsLocalWebSearchForXiaomi(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	def, ok := providerDefinition("xiaomi")
	if !ok {
		t.Fatal("xiaomi provider definition missing")
	}
	route := ResolvedModelRoute{
		Provider:   domain.ProviderConfig{ID: "xiaomi"},
		Model:      domain.ModelRef{ProviderID: "xiaomi", ModelID: "mimo-v2.5-pro"},
		Definition: def,
		Transport:  def.Transport,
	}
	specs := []domain.ToolSpec{NewCodexWebSearchTool(service).Spec()}

	tools := service.toolsForModelRoute(context.Background(), domain.AppConfig{WebSearch: domain.WebSearchConfig{Mode: domain.WebSearchModeLive, Route: domain.WebSearchRouteAuto}}, route, specs)
	if len(tools) != 1 || tools[0].Hosted != nil || tools[0].Name != "web_search" {
		t.Fatalf("tools = %#v, want local web_search fallback", tools)
	}
}

func TestProviderDeclaredLocalToolActivationsAddsParallelFallbackForUnsupportedProvider(t *testing.T) {
	cfg := domain.AppConfig{
		Provider:     &domain.ProviderConfig{ID: "local-compatible", Type: string(TransportOpenAICompatible), BaseURL: "http://127.0.0.1:1/v1", Model: "local-model"},
		DefaultModel: &domain.ModelRef{ProviderID: "local-compatible", ModelID: "local-model"},
		WebSearch:    domain.WebSearchConfig{Mode: domain.WebSearchModeLive, Route: domain.WebSearchRouteAuto},
	}
	store := &memoryProviderStore{config: &cfg}
	service := NewService(store)
	defer service.Shutdown()
	registerNoAuthProvider(t, service, "local-compatible", "http://127.0.0.1:1/v1", "local-model")

	activations := service.providerDeclaredLocalToolActivations(context.Background(), cfg.DefaultModel)
	if activations["web_search"] != "providerCapability" {
		t.Fatalf("activations = %#v, want local web_search fallback", activations)
	}
}
