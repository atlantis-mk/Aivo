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
	specs := []domain.ToolSpec{{Name: "web_search", Description: "Search", InputSchema: map[string]any{"type": "object"}}}

	tools := service.toolsForModelRoute(context.Background(), domain.AppConfig{WebSearch: domain.WebSearchConfig{Mode: domain.WebSearchModeLive, Route: domain.WebSearchRouteAuto}}, route, specs)
	if len(tools) != 1 || tools[0].Hosted != nil || tools[0].Name != "web_search" {
		t.Fatalf("tools = %#v, want local web_search", tools)
	}
}
