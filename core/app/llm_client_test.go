package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestResponsesRequestBodyUsesInputItemList(t *testing.T) {
	body := responsesRequestBody("gpt-5.5", []llmChatMessage{
		{Role: "system", Text: "be concise"},
		{Role: "user", Text: "hello"},
		{Role: "assistant", Text: "hi"},
	}, nil, "high", "priority")
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	var rawPayload map[string]any
	if err := json.Unmarshal(raw, &rawPayload); err != nil {
		t.Fatal(err)
	}
	if store, ok := rawPayload["store"].(bool); !ok || store {
		t.Fatalf("store = %#v, want false", rawPayload["store"])
	}
	if stream, ok := rawPayload["stream"].(bool); !ok || !stream {
		t.Fatalf("stream = %#v, want true", rawPayload["stream"])
	}

	var payload struct {
		Model  string `json:"model"`
		Store  bool   `json:"store"`
		Stream bool   `json:"stream"`
		Input  []struct {
			Role    string `json:"role"`
			Content any    `json:"content"`
		} `json:"input"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}

	if payload.Model != "gpt-5.5" {
		t.Fatalf("Model = %q, want %q", payload.Model, "gpt-5.5")
	}
	if payload.Store {
		t.Fatal("Store = true, want false")
	}
	if !payload.Stream {
		t.Fatal("Stream = false, want true")
	}
	reasoning, _ := rawPayload["reasoning"].(map[string]any)
	if reasoning["effort"] != "high" {
		t.Fatalf("reasoning.effort = %#v, want high", reasoning["effort"])
	}
	if rawPayload["service_tier"] != "priority" {
		t.Fatalf("service_tier = %#v, want priority", rawPayload["service_tier"])
	}
	if len(payload.Input) != 3 {
		t.Fatalf("len(Input) = %d, want 3", len(payload.Input))
	}
	if payload.Input[0].Role != "system" || payload.Input[0].Content != "be concise" {
		t.Fatalf("system input item = %+v", payload.Input[0])
	}
	assertResponsesContent(t, payload.Input[1], "user", "input_text", "hello")
	assertResponsesContent(t, payload.Input[2], "assistant", "output_text", "hi")
}

func TestResponsesRequestBodyRendersImageAndFileAttachments(t *testing.T) {
	body := responsesRequestBody("gpt-5.5", []llmChatMessage{{
		Role: "user",
		Text: "inspect these",
		Attachments: []domain.MessageAttachment{
			{Name: "screen.png", MIMEType: "image/png", Kind: "image", Data: "aW1hZ2U="},
			{Name: "brief.pdf", MIMEType: "application/pdf", Kind: "file", Data: "cGRm"},
		},
	}}, nil, "", "")

	input, _ := body["input"].([]map[string]any)
	if len(input) != 1 {
		t.Fatalf("input = %#v, want one item", body["input"])
	}
	content, _ := input[0]["content"].([]map[string]string)
	if len(content) != 3 {
		t.Fatalf("content = %#v, want text plus two attachments", input[0]["content"])
	}
	if content[1]["type"] != "input_image" || content[1]["image_url"] != "data:image/png;base64,aW1hZ2U=" {
		t.Fatalf("image part = %#v", content[1])
	}
	if content[2]["type"] != "input_file" || content[2]["filename"] != "brief.pdf" || content[2]["file_data"] != "data:application/pdf;base64,cGRm" {
		t.Fatalf("file part = %#v", content[2])
	}
}

func TestResponsesRequestBodyUsesCodexToolStreamingShape(t *testing.T) {
	body := responsesRequestBody("gpt-5.5", []llmChatMessage{{Role: "user", Text: "hello"}}, []domain.ToolSpec{{
		Name:        "read_file",
		Description: "Read a file",
		InputSchema: map[string]any{"type": "object"},
		Namespace:   "functions",
	}}, "", "")

	if stream, _ := body["stream"].(bool); !stream {
		t.Fatalf("stream = %#v, want true", body["stream"])
	}
	if toolChoice := body["tool_choice"]; toolChoice != "auto" {
		t.Fatalf("tool_choice = %#v, want auto", toolChoice)
	}
	if parallel, _ := body["parallel_tool_calls"].(bool); !parallel {
		t.Fatalf("parallel_tool_calls = %#v, want true", body["parallel_tool_calls"])
	}
	if tools, _ := body["tools"].([]map[string]any); len(tools) != 1 {
		t.Fatalf("tools = %#v, want one tool", body["tools"])
	}
	tools, _ := body["tools"].([]map[string]any)
	if tools[0]["type"] != "namespace" || tools[0]["name"] != "functions" {
		t.Fatalf("tools[0] = %#v, want functions namespace", tools[0])
	}
}

func TestResponsesToolsGroupsNamespaceTools(t *testing.T) {
	tools := responsesTools([]domain.ToolSpec{
		{
			Name:                 "read_file",
			Description:          "Read a file",
			InputSchema:          map[string]any{"type": "object"},
			Namespace:            "functions",
			NamespaceDescription: "Workspace file tools.",
		},
		{
			Name:        "list_files",
			Description: "List files",
			InputSchema: map[string]any{"type": "object"},
			Namespace:   "functions",
		},
		{
			Name:        "standalone",
			Description: "Standalone tool",
			InputSchema: map[string]any{"type": "object"},
		},
	})

	if len(tools) != 2 {
		t.Fatalf("len(tools) = %d, want 2: %#v", len(tools), tools)
	}
	namespace := tools[0]
	if namespace["type"] != "namespace" || namespace["name"] != "functions" || namespace["description"] != "Workspace file tools." {
		t.Fatalf("namespace = %#v", namespace)
	}
	namespaceTools, _ := namespace["tools"].([]map[string]any)
	if len(namespaceTools) != 2 || namespaceTools[0]["name"] != "read_file" || namespaceTools[1]["name"] != "list_files" {
		t.Fatalf("namespace tools = %#v", namespace["tools"])
	}
	if tools[1]["type"] != "function" || tools[1]["name"] != "standalone" {
		t.Fatalf("standalone tool = %#v", tools[1])
	}
}

func TestResponsesToolsDoesNotUseReservedBrowserNamespace(t *testing.T) {
	tools := responsesTools([]domain.ToolSpec{{
		Name:        "browser_click",
		Description: "Click browser",
		InputSchema: map[string]any{"type": "object"},
		Namespace:   "browser",
		Category:    "browser",
	}})

	if len(tools) != 1 {
		t.Fatalf("len(tools) = %d, want 1: %#v", len(tools), tools)
	}
	if tools[0]["type"] != "function" || tools[0]["name"] != "browser_click" {
		t.Fatalf("browser tool = %#v, want standalone function", tools[0])
	}
	if _, ok := tools[0]["tools"]; ok {
		t.Fatalf("browser tool = %#v, should not be namespace-wrapped", tools[0])
	}
}

func TestResponsesToolsRendersFreeformCustomTool(t *testing.T) {
	tools := responsesTools([]domain.ToolSpec{{
		Name:        "apply_patch",
		Description: "Apply patch",
		Kind:        domain.ToolKindFreeform,
		Format:      &domain.ToolFormat{Type: "grammar", Syntax: "lark", Definition: "start: /.+/"},
		InputSchema: map[string]any{"type": "object"},
		Namespace:   "functions",
	}})
	if len(tools) != 1 {
		t.Fatalf("tools = %#v, want one custom tool", tools)
	}
	if tools[0]["type"] != "custom" || tools[0]["name"] != "apply_patch" {
		t.Fatalf("tool = %#v, want custom apply_patch", tools[0])
	}
	format, _ := tools[0]["format"].(*domain.ToolFormat)
	if format == nil || format.Type != "grammar" || format.Syntax != "lark" {
		t.Fatalf("format = %#v, want grammar/lark", tools[0]["format"])
	}
}

func TestResponsesToolsRendersHostedWebSearch(t *testing.T) {
	external := true
	tools := responsesTools([]domain.ToolSpec{{
		Name: "web_search",
		Hosted: &domain.HostedToolSpec{
			Type:              "web_search",
			ExternalWebAccess: &external,
			SearchContextSize: "high",
			AllowedDomains:    []string{"example.com"},
			UserLocation:      &domain.WebSearchUserLocation{Type: "approximate", Country: "US", City: "San Francisco"},
		},
	}})

	if len(tools) != 1 {
		t.Fatalf("tools = %#v, want one hosted tool", tools)
	}
	if tools[0]["type"] != "web_search" || tools[0]["external_web_access"] != true || tools[0]["search_context_size"] != "high" {
		t.Fatalf("hosted tool = %#v", tools[0])
	}
	filters, _ := tools[0]["filters"].(map[string]any)
	domains, _ := filters["allowed_domains"].([]string)
	if len(domains) != 1 || domains[0] != "example.com" {
		t.Fatalf("filters = %#v", tools[0]["filters"])
	}
	location, _ := tools[0]["user_location"].(map[string]any)
	if location["type"] != "approximate" || location["country"] != "US" || location["city"] != "San Francisco" {
		t.Fatalf("user_location = %#v", tools[0]["user_location"])
	}
}

func TestResponsesToolsRendersHostedProviderTools(t *testing.T) {
	tools := responsesTools([]domain.ToolSpec{
		{Name: "x_search", Hosted: &domain.HostedToolSpec{Type: "x_search"}},
		{Name: "code_interpreter", Hosted: &domain.HostedToolSpec{Type: "code_interpreter", FileIDs: []string{"file_1"}}},
		{Name: "file_search", Hosted: &domain.HostedToolSpec{Type: "file_search", VectorStoreIDs: []string{"vs_1"}}},
		{Name: "remote_mcp", Hosted: &domain.HostedToolSpec{Type: "mcp", ServerURL: "https://mcp.example.com", ServerLabel: "docs", AllowedTools: []string{"search"}}},
	})

	if len(tools) != 4 {
		t.Fatalf("tools = %#v, want four hosted tools", tools)
	}
	if tools[0]["type"] != "x_search" {
		t.Fatalf("x_search tool = %#v", tools[0])
	}
	container, _ := tools[1]["container"].(map[string]any)
	files, _ := container["files"].([]string)
	if tools[1]["type"] != "code_interpreter" || container["type"] != "auto" || len(files) != 1 || files[0] != "file_1" {
		t.Fatalf("code_interpreter tool = %#v", tools[1])
	}
	vectorStores, _ := tools[2]["vector_store_ids"].([]string)
	if tools[2]["type"] != "file_search" || len(vectorStores) != 1 || vectorStores[0] != "vs_1" {
		t.Fatalf("file_search tool = %#v", tools[2])
	}
	allowedTools, _ := tools[3]["allowed_tools"].([]string)
	if tools[3]["type"] != "mcp" || tools[3]["server_url"] != "https://mcp.example.com" || len(allowedTools) != 1 || allowedTools[0] != "search" {
		t.Fatalf("mcp tool = %#v", tools[3])
	}
}

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

func TestToolsForModelRouteInjectsConfiguredXAINativeTools(t *testing.T) {
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
	if !toolSpecNamed(tools, "x_search") || !toolSpecNamed(tools, "code_interpreter") || !toolSpecNamed(tools, "file_search") || !toolSpecNamed(tools, "remote_mcp_docs") {
		t.Fatalf("tools = %#v, want configured xAI native tools", tools)
	}
}

func TestToolsForModelRouteInjectsConfiguredGeminiNativeTools(t *testing.T) {
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
	if !toolSpecNamed(tools, "code_execution") || !toolSpecNamed(tools, "file_search") {
		t.Fatalf("tools = %#v, want configured Gemini native tools", tools)
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

func TestReadLLMEventStreamEmitsResponsesCustomToolDeltas(t *testing.T) {
	raw := strings.NewReader("event: response.output_item.added\n" +
		"data: {\"type\":\"response.output_item.added\",\"item\":{\"id\":\"ct_1\",\"type\":\"custom_tool_call\",\"status\":\"in_progress\",\"input\":\"\",\"call_id\":\"call_patch\",\"name\":\"apply_patch\"}}\n\n" +
		"event: response.custom_tool_call_input.delta\n" +
		"data: {\"type\":\"response.custom_tool_call_input.delta\",\"item_id\":\"ct_1\",\"delta\":\"*** Begin Patch\\n*** Add File: docs/spec.md\\n+hello\"}\n\n" +
		"event: response.custom_tool_call_input.done\n" +
		"data: {\"type\":\"response.custom_tool_call_input.done\",\"item_id\":\"ct_1\",\"input\":\"*** Begin Patch\\n*** Add File: docs/spec.md\\n+hello\\n*** End Patch\\n\"}\n\n" +
		"event: response.output_item.done\n" +
		"data: {\"type\":\"response.output_item.done\",\"item\":{\"id\":\"ct_1\",\"type\":\"custom_tool_call\",\"status\":\"completed\",\"input\":\"*** Begin Patch\\n*** Add File: docs/spec.md\\n+hello\\n*** End Patch\\n\",\"call_id\":\"call_patch\",\"name\":\"apply_patch\"}}\n\n")
	var toolDeltas []domain.ChatToolCall
	resp, err := readLLMEventStream(raw, nil, func(call domain.ChatToolCall) {
		toolDeltas = append(toolDeltas, call)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].ID != "call_patch" || !strings.Contains(string(resp.ToolCalls[0].Arguments), "docs/spec.md") {
		t.Fatalf("ToolCalls = %#v, want completed custom apply_patch", resp.ToolCalls)
	}
	if len(toolDeltas) < 2 {
		t.Fatalf("toolDeltas = %#v, want streamed custom deltas", toolDeltas)
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

func TestExtractResponseTextReadsResponsesStreamDeltas(t *testing.T) {
	raw := []byte("event: response.output_text.delta\n" +
		"data: {\"type\":\"response.output_text.delta\",\"item_id\":\"msg_1\",\"delta\":\"Hello\"}\n\n" +
		"event: response.output_text.delta\n" +
		"data: {\"type\":\"response.output_text.delta\",\"item_id\":\"msg_1\",\"delta\":\"!\"}\n\n" +
		"event: response.completed\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\"}}\n\n")

	if got := extractResponseText(raw); got != "Hello!" {
		t.Fatalf("extractResponseText(stream) = %q, want %q", got, "Hello!")
	}
}

func TestDoLLMRequestReadsEventStreamWhenContentTypeIsMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); !strings.Contains(got, "text/event-stream") {
			t.Fatalf("Accept = %q, want text/event-stream", got)
		}
		_, _ = w.Write([]byte("event: response.output_text.delta\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"item_id\":\"msg_1\",\"delta\":\"Hello\"}\n\n"))
		_, _ = w.Write([]byte("event: response.output_text.delta\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"item_id\":\"msg_1\",\"delta\":\" streamed\"}\n\n"))
		_, _ = w.Write([]byte("event: response.completed\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\"}}\n\n"))
	}))
	defer server.Close()
	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept", "text/event-stream")
	var deltas []string

	resp, err := doLLMRequest(req, func(delta string) {
		deltas = append(deltas, delta)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "Hello streamed" {
		t.Fatalf("Text = %q, want streamed response", resp.Text)
	}
	if strings.Join(deltas, "") != "Hello streamed" || len(deltas) != 2 {
		t.Fatalf("deltas = %#v, want two streamed chunks", deltas)
	}
}

func TestReadLLMEventStreamSeparatesResponsesToolArgumentsFromText(t *testing.T) {
	raw := strings.NewReader("event: response.output_item.added\n" +
		"data: {\"type\":\"response.output_item.added\",\"item\":{\"id\":\"fc_1\",\"type\":\"function_call\",\"status\":\"in_progress\",\"arguments\":\"\",\"call_id\":\"call_read\",\"name\":\"read_file\"}}\n\n" +
		"event: response.function_call_arguments.delta\n" +
		"data: {\"type\":\"response.function_call_arguments.delta\",\"item_id\":\"fc_1\",\"delta\":\"{\\\"path\\\"\"}\n\n" +
		"event: response.function_call_arguments.delta\n" +
		"data: {\"type\":\"response.function_call_arguments.delta\",\"item_id\":\"fc_1\",\"delta\":\":\\\"README.md\\\"}\"}\n\n" +
		"event: response.function_call_arguments.done\n" +
		"data: {\"type\":\"response.function_call_arguments.done\",\"item_id\":\"fc_1\",\"arguments\":\"{\\\"path\\\":\\\"README.md\\\"}\"}\n\n" +
		"event: response.output_item.done\n" +
		"data: {\"type\":\"response.output_item.done\",\"item\":{\"id\":\"fc_1\",\"type\":\"function_call\",\"status\":\"completed\",\"arguments\":\"{\\\"path\\\":\\\"README.md\\\"}\",\"call_id\":\"call_read\",\"name\":\"read_file\"}}\n\n")
	var deltas []string

	resp, err := readLLMEventStream(raw, func(delta string) {
		deltas = append(deltas, delta)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(deltas) != 0 || resp.Text != "" {
		t.Fatalf("text deltas = %#v text=%q, want no text", deltas, resp.Text)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls = %#v, want one call", resp.ToolCalls)
	}
	call := resp.ToolCalls[0]
	if call.ID != "call_read" || call.Name != "read_file" || string(call.Arguments) != `{"path":"README.md"}` {
		t.Fatalf("ToolCall = %#v", call)
	}
}

func TestReadLLMEventStreamEmitsResponsesToolArgumentDeltas(t *testing.T) {
	raw := strings.NewReader("event: response.output_item.added\n" +
		"data: {\"type\":\"response.output_item.added\",\"item\":{\"id\":\"fc_1\",\"type\":\"function_call\",\"status\":\"in_progress\",\"arguments\":\"\",\"call_id\":\"call_patch\",\"name\":\"apply_patch\"}}\n\n" +
		"event: response.function_call_arguments.delta\n" +
		"data: {\"type\":\"response.function_call_arguments.delta\",\"item_id\":\"fc_1\",\"delta\":\"{\\\"patchText\\\":\\\"*** Begin Patch\\\\n*** Add File: docs/spec.md\\\\n+hello\"}\n\n")
	var toolDeltas []domain.ChatToolCall

	resp, err := readLLMEventStream(raw, nil, func(call domain.ChatToolCall) {
		toolDeltas = append(toolDeltas, call)
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "" {
		t.Fatalf("Text = %q, want empty text", resp.Text)
	}
	if len(toolDeltas) < 2 {
		t.Fatalf("toolDeltas = %#v, want start and argument delta", toolDeltas)
	}
	last := toolDeltas[len(toolDeltas)-1]
	if last.ID != "call_patch" || last.Name != "apply_patch" {
		t.Fatalf("last tool delta = %#v", last)
	}
	if !strings.Contains(string(last.Arguments), "docs/spec.md") {
		t.Fatalf("last arguments = %q, want streamed patch text", string(last.Arguments))
	}
}

func TestReadLLMEventStreamDoesNotEmitFinalResponsesItemAsDelta(t *testing.T) {
	raw := strings.NewReader("event: response.output_item.done\n" +
		"data: {\"type\":\"response.output_item.done\",\"item\":{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"final answer\"}]}}\n\n" +
		"event: response.completed\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\"}}\n\n")
	var deltas []string

	resp, err := readLLMEventStream(raw, func(delta string) {
		deltas = append(deltas, delta)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "final answer" {
		t.Fatalf("Text = %q, want final answer", resp.Text)
	}
	if len(deltas) != 0 {
		t.Fatalf("deltas = %#v, want none for final item", deltas)
	}
}

func TestReadLLMEventStreamDoesNotEmitFinalResponsesCompletedTextAsDelta(t *testing.T) {
	raw := strings.NewReader("event: response.completed\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"output\":[{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"final answer\"}]}]}}\n\n")
	var deltas []string

	resp, err := readLLMEventStream(raw, func(delta string) {
		deltas = append(deltas, delta)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "final answer" {
		t.Fatalf("Text = %q, want final answer", resp.Text)
	}
	if len(deltas) != 0 {
		t.Fatalf("deltas = %#v, want none for completed response", deltas)
	}
}

func TestReadLLMEventStreamCollectsChatCompletionToolDeltas(t *testing.T) {
	raw := strings.NewReader("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_read\",\"type\":\"function\",\"function\":{\"name\":\"read_file\",\"arguments\":\"{\\\"path\\\"\"}}]}}]}\n\n" +
		"data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\":\\\"README.md\\\"}\"}}]},\"finish_reason\":\"tool_calls\"}]}\n\n" +
		"data: [DONE]\n\n")
	var deltas []string

	resp, err := readLLMEventStream(raw, func(delta string) {
		deltas = append(deltas, delta)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(deltas) != 0 || resp.Text != "" {
		t.Fatalf("text deltas = %#v text=%q, want no text", deltas, resp.Text)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls = %#v, want one call", resp.ToolCalls)
	}
	call := resp.ToolCalls[0]
	if call.ID != "call_read" || call.Name != "read_file" || string(call.Arguments) != `{"path":"README.md"}` {
		t.Fatalf("ToolCall = %#v", call)
	}
}

func TestReadLLMEventStreamCollectsAnthropicToolDeltas(t *testing.T) {
	raw := strings.NewReader("event: content_block_start\n" +
		"data: {\"type\":\"content_block_start\",\"index\":1,\"content_block\":{\"type\":\"tool_use\",\"id\":\"toolu_read\",\"name\":\"read_file\",\"input\":{}}}\n\n" +
		"event: content_block_delta\n" +
		"data: {\"type\":\"content_block_delta\",\"index\":1,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"path\\\"\"}}\n\n" +
		"event: content_block_delta\n" +
		"data: {\"type\":\"content_block_delta\",\"index\":1,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\":\\\"README.md\\\"}\"}}\n\n" +
		"event: content_block_stop\n" +
		"data: {\"type\":\"content_block_stop\",\"index\":1}\n\n")
	var deltas []string

	resp, err := readLLMEventStream(raw, func(delta string) {
		deltas = append(deltas, delta)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(deltas) != 0 || resp.Text != "" {
		t.Fatalf("text deltas = %#v text=%q, want no text", deltas, resp.Text)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls = %#v, want one call", resp.ToolCalls)
	}
	call := resp.ToolCalls[0]
	if call.ID != "toolu_read" || call.Name != "read_file" || string(call.Arguments) != `{"path":"README.md"}` {
		t.Fatalf("ToolCall = %#v", call)
	}
}

func TestChatCompletionsRequestBodyStreams(t *testing.T) {
	body := chatCompletionsRequestBody("openai/gpt-5", []llmChatMessage{{Role: "user", Text: "hello"}}, nil)

	if stream, _ := body["stream"].(bool); !stream {
		t.Fatalf("stream = %#v, want true", body["stream"])
	}
}

func TestAnthropicRequestBodyStreams(t *testing.T) {
	body := anthropicRequestBody("claude-sonnet-4-5", []llmChatMessage{{Role: "user", Text: "hello"}}, nil, "")

	if stream, _ := body["stream"].(bool); !stream {
		t.Fatalf("stream = %#v, want true", body["stream"])
	}
}

func TestAnthropicRequestBodyRendersImageAndPDFAttachments(t *testing.T) {
	body := anthropicRequestBody("claude-sonnet-4-5", []llmChatMessage{{
		Role: "user",
		Text: "inspect these",
		Attachments: []domain.MessageAttachment{
			{Name: "screen.png", MIMEType: "image/png", Kind: "image", Data: "aW1hZ2U="},
			{Name: "brief.pdf", MIMEType: "application/pdf", Kind: "file", Data: "cGRm"},
		},
	}}, nil, "")

	messages, _ := body["messages"].([]map[string]any)
	if len(messages) != 1 {
		t.Fatalf("messages = %#v, want one message", body["messages"])
	}
	content, _ := messages[0]["content"].([]map[string]any)
	if len(content) != 3 {
		t.Fatalf("content = %#v, want text plus two attachments", messages[0]["content"])
	}
	if content[1]["type"] != "image" {
		t.Fatalf("image part = %#v", content[1])
	}
	imageSource, _ := content[1]["source"].(map[string]string)
	if imageSource["media_type"] != "image/png" || imageSource["data"] != "aW1hZ2U=" {
		t.Fatalf("image source = %#v", imageSource)
	}
	if content[2]["type"] != "document" || content[2]["title"] != "brief.pdf" {
		t.Fatalf("document part = %#v", content[2])
	}
}

func TestAnthropicRequestBodyStreamsWithTools(t *testing.T) {
	body := anthropicRequestBody("claude-sonnet-4-5", []llmChatMessage{{Role: "user", Text: "hello"}}, []domain.ToolSpec{{
		Name:        "read_file",
		Description: "Read a file",
		InputSchema: map[string]any{"type": "object"},
	}}, "")

	if stream, _ := body["stream"].(bool); !stream {
		t.Fatalf("stream = %#v, want true", body["stream"])
	}
	if tools, _ := body["tools"].([]map[string]any); len(tools) != 1 {
		t.Fatalf("tools = %#v, want one tool", body["tools"])
	}
}

func TestAnthropicRequestBodyRendersHostedWebSearch(t *testing.T) {
	body := anthropicRequestBody("claude-sonnet-4-5", []llmChatMessage{{Role: "user", Text: "hello"}}, []domain.ToolSpec{{
		Name: "web_search",
		Hosted: &domain.HostedToolSpec{
			Type:           "web_search_20250305",
			AllowedDomains: []string{"example.com"},
			UserLocation:   &domain.WebSearchUserLocation{Type: "approximate", Country: "US"},
		},
	}}, "")

	tools, _ := body["tools"].([]map[string]any)
	if len(tools) != 1 {
		t.Fatalf("tools = %#v, want one hosted tool", body["tools"])
	}
	if tools[0]["type"] != "web_search_20250305" || tools[0]["name"] != "web_search" {
		t.Fatalf("tool = %#v, want anthropic web_search server tool", tools[0])
	}
	domains, _ := tools[0]["allowed_domains"].([]string)
	if len(domains) != 1 || domains[0] != "example.com" {
		t.Fatalf("allowed_domains = %#v", tools[0]["allowed_domains"])
	}
}

func TestAnthropicRequestBodyRendersHostedWebFetch(t *testing.T) {
	body := anthropicRequestBody("claude-sonnet-4-5", []llmChatMessage{{Role: "user", Text: "hello"}}, []domain.ToolSpec{{
		Name:   "web_fetch",
		Hosted: &domain.HostedToolSpec{Type: "web_fetch_20250910", MaxUses: 3},
	}}, "")

	tools, _ := body["tools"].([]map[string]any)
	if len(tools) != 1 {
		t.Fatalf("tools = %#v, want one hosted tool", body["tools"])
	}
	if tools[0]["type"] != "web_fetch_20250910" || tools[0]["name"] != "web_fetch" || tools[0]["max_uses"] != 3 {
		t.Fatalf("tool = %#v, want anthropic web_fetch server tool", tools[0])
	}
}

func TestAnthropicRequestBodyUsesBudgetThinkingForClaudeFourFive(t *testing.T) {
	body := anthropicRequestBody("claude-sonnet-4-5", []llmChatMessage{{Role: "user", Text: "hello"}}, nil, "high")

	thinking, _ := body["thinking"].(map[string]any)
	if thinking["type"] != "enabled" || thinking["budget_tokens"] != 4096 {
		t.Fatalf("thinking = %#v, want enabled budget 4096", thinking)
	}
	if body["max_tokens"] != 64000 {
		t.Fatalf("max_tokens = %#v, want 64000", body["max_tokens"])
	}
}

func TestAnthropicRequestBodyUsesEffortForClaudeFourSix(t *testing.T) {
	body := anthropicRequestBody("claude-sonnet-4-6", []llmChatMessage{{Role: "user", Text: "hello"}}, nil, "ultra")

	thinking, _ := body["thinking"].(map[string]any)
	if thinking["type"] != "adaptive" {
		t.Fatalf("thinking = %#v, want adaptive", thinking)
	}
	outputConfig, _ := body["output_config"].(map[string]any)
	if outputConfig["effort"] != "xhigh" {
		t.Fatalf("output_config.effort = %#v, want xhigh", outputConfig["effort"])
	}
	if _, ok := thinking["budget_tokens"]; ok {
		t.Fatalf("thinking = %#v, did not expect budget_tokens for Claude 4.6", thinking)
	}
}

func TestGoogleRequestBodyMapsThinkingByModelGeneration(t *testing.T) {
	gemini25 := googleRequestBody("gemini-2.5-pro", []llmChatMessage{{Role: "user", Text: "hello"}}, nil, "high")
	generationConfig, _ := gemini25["generationConfig"].(map[string]any)
	thinkingConfig, _ := generationConfig["thinkingConfig"].(map[string]any)
	if thinkingConfig["thinkingBudget"] != 8192 {
		t.Fatalf("thinkingBudget = %#v, want 8192", thinkingConfig["thinkingBudget"])
	}

	gemini3 := googleRequestBody("gemini-3-pro-preview", []llmChatMessage{{Role: "user", Text: "hello"}}, nil, "low")
	generationConfig, _ = gemini3["generationConfig"].(map[string]any)
	thinkingConfig, _ = generationConfig["thinkingConfig"].(map[string]any)
	if thinkingConfig["thinkingLevel"] != "low" {
		t.Fatalf("thinkingLevel = %#v, want low", thinkingConfig["thinkingLevel"])
	}
}

func TestGoogleRequestBodyRendersInlineAttachmentParts(t *testing.T) {
	body := googleRequestBody("gemini-2.5-pro", []llmChatMessage{{
		Role: "user",
		Text: "inspect this",
		Attachments: []domain.MessageAttachment{
			{Name: "screen.png", MIMEType: "image/png", Kind: "image", Data: "aW1hZ2U="},
		},
	}}, nil, "")

	contents, _ := body["contents"].([]map[string]any)
	if len(contents) != 1 {
		t.Fatalf("contents = %#v, want one content", body["contents"])
	}
	parts, _ := contents[0]["parts"].([]map[string]any)
	if len(parts) != 2 {
		t.Fatalf("parts = %#v, want text plus inline data", contents[0]["parts"])
	}
	inlineData, _ := parts[1]["inlineData"].(map[string]string)
	if inlineData["mimeType"] != "image/png" || inlineData["data"] != "aW1hZ2U=" {
		t.Fatalf("inlineData = %#v", inlineData)
	}
}

func TestGoogleRequestBodyRendersHostedWebSearch(t *testing.T) {
	body := googleRequestBody("gemini-3-pro-preview", []llmChatMessage{{Role: "user", Text: "hello"}}, []domain.ToolSpec{
		{Name: "web_search", Hosted: &domain.HostedToolSpec{Type: "google_search"}},
		{Name: "read_file", Description: "Read a file", InputSchema: map[string]any{"type": "object"}},
	}, "")

	tools, _ := body["tools"].([]map[string]any)
	if len(tools) != 2 {
		t.Fatalf("tools = %#v, want google_search plus function declarations", body["tools"])
	}
	if _, ok := tools[0]["google_search"].(map[string]any); !ok {
		t.Fatalf("tools[0] = %#v, want google_search", tools[0])
	}
	declarations, _ := tools[1]["functionDeclarations"].([]map[string]any)
	if len(declarations) != 1 || declarations[0]["name"] != "read_file" {
		t.Fatalf("functionDeclarations = %#v", tools[1]["functionDeclarations"])
	}
}

func TestGoogleRequestBodyRendersURLContext(t *testing.T) {
	body := googleRequestBody("gemini-3-pro-preview", []llmChatMessage{{Role: "user", Text: "hello"}}, []domain.ToolSpec{
		{Name: "web_fetch", Hosted: &domain.HostedToolSpec{Type: "url_context"}},
	}, "")

	tools, _ := body["tools"].([]map[string]any)
	if len(tools) != 1 {
		t.Fatalf("tools = %#v, want one url_context tool", body["tools"])
	}
	if _, ok := tools[0]["url_context"].(map[string]any); !ok {
		t.Fatalf("tools[0] = %#v, want url_context", tools[0])
	}
}

func TestGoogleRequestBodyRendersHostedCodeExecutionAndFileSearch(t *testing.T) {
	body := googleRequestBody("gemini-3-pro-preview", []llmChatMessage{{Role: "user", Text: "hello"}}, []domain.ToolSpec{
		{Name: "code_execution", Hosted: &domain.HostedToolSpec{Type: "code_execution"}},
		{Name: "file_search", Hosted: &domain.HostedToolSpec{Type: "file_search", VectorStoreIDs: []string{"fileSearchStores/store_1"}}},
	}, "")

	tools, _ := body["tools"].([]map[string]any)
	if len(tools) != 2 {
		t.Fatalf("tools = %#v, want code_execution plus file_search", body["tools"])
	}
	if _, ok := tools[0]["code_execution"].(map[string]any); !ok {
		t.Fatalf("tools[0] = %#v, want code_execution", tools[0])
	}
	fileSearch, _ := tools[1]["file_search"].(map[string]any)
	storeNames, _ := fileSearch["file_search_store_names"].([]string)
	if len(storeNames) != 1 || storeNames[0] != "fileSearchStores/store_1" {
		t.Fatalf("file_search = %#v, want configured store names", tools[1]["file_search"])
	}
}

func TestExtractResponseTextReadsChatCompletionStreamDeltas(t *testing.T) {
	raw := []byte("data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n" +
		"data: {\"choices\":[{\"delta\":{\"content\":\"!\"},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n")

	if got := extractResponseText(raw); got != "Hello!" {
		t.Fatalf("extractResponseText(chat stream) = %q, want %q", got, "Hello!")
	}
}

func TestExtractResponseTextReadsAnthropicStreamDeltas(t *testing.T) {
	raw := []byte("event: content_block_delta\n" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\n" +
		"event: content_block_delta\n" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"!\"}}\n\n" +
		"event: message_stop\n" +
		"data: {\"type\":\"message_stop\"}\n\n")

	if got := extractResponseText(raw); got != "Hello!" {
		t.Fatalf("extractResponseText(anthropic stream) = %q, want %q", got, "Hello!")
	}
}

func TestCallOpenAICompatibleUsesChatCompletionsStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %q, want /chat/completions", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want bearer key", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "openai/gpt-5" {
			t.Fatalf("model = %#v", body["model"])
		}
		if stream, _ := body["stream"].(bool); !stream {
			t.Fatalf("stream = %#v, want true", body["stream"])
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n"))
	}))
	defer server.Close()

	got, err := callOpenAICompatible(
		context.Background(),
		domain.ProviderConfig{ID: "openrouter", Type: "openrouter", BaseURL: server.URL},
		domain.ModelRef{ProviderID: "openrouter", ModelID: "openai/gpt-5"},
		llmCredential{APIKey: "test-key"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		nil,
		"",
		"",
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "ok" {
		t.Fatalf("reply = %q, want ok", got.Text)
	}
}

func TestCallOpenAICompatibleAppliesRequestProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Profile"); got != "default" {
			t.Fatalf("X-Profile = %q, want default", got)
		}
		if got := r.Header.Get("X-Model"); got != "matched" {
			t.Fatalf("X-Model = %q, want matched", got)
		}
		if got := r.Header.Get("X-Team"); got != "custom" {
			t.Fatalf("X-Team = %q, want custom", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want bearer key", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "openai/gpt-5" {
			t.Fatalf("model = %#v, want generated model", body["model"])
		}
		if body["reasoning"].(map[string]any)["effort"] != "high" {
			t.Fatalf("reasoning = %#v, want high override", body["reasoning"])
		}
		if body["temperature"] != float64(0.2) {
			t.Fatalf("temperature = %#v, want 0.2", body["temperature"])
		}
		if body["service_tier"] != "priority" {
			t.Fatalf("service_tier = %#v, want priority", body["service_tier"])
		}
		provider, _ := body["provider"].(map[string]any)
		if provider["sort"] != "throughput" || provider["allow_fallbacks"] != false {
			t.Fatalf("provider = %#v, want merged provider params", provider)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n"))
	}))
	defer server.Close()

	profile := domain.ProviderRequestProfile{
		Headers: map[string]string{"X-Profile": "default", "Authorization": "Bearer ignored"},
		Params: map[string]any{
			"temperature": 0.1,
			"provider":    map[string]any{"sort": "latency", "allow_fallbacks": true},
			"model":       "ignored",
		},
		ModelOverrides: map[string]domain.ProviderRequestOverride{
			"openai/gpt": {
				Headers: map[string]string{"X-Model": "matched"},
				Params:  map[string]any{"reasoning": map[string]any{"effort": "medium"}},
			},
		},
	}
	provider := domain.ProviderConfig{
		ID:      "openrouter",
		Type:    "openrouter",
		BaseURL: server.URL,
		Headers: map[string]string{"X-Team": "custom"},
		RequestParams: map[string]any{
			"temperature":  0.2,
			"service_tier": "priority",
			"reasoning":    map[string]any{"effort": "high"},
			"provider":     map[string]any{"sort": "throughput", "allow_fallbacks": false},
		},
	}
	got, err := callOpenAICompatible(
		context.Background(),
		provider,
		domain.ModelRef{ProviderID: "openrouter", ModelID: "openai/gpt-5"},
		llmCredential{APIKey: "test-key"},
		profile,
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		nil,
		"",
		"",
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "ok" {
		t.Fatalf("reply = %q, want ok", got.Text)
	}
}

func TestCallAzureOpenAIUsesResponsesEndpointAndAPIKeyHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/openai/v1/responses" {
			t.Fatalf("path = %q, want /openai/v1/responses", r.URL.Path)
		}
		if got := r.Header.Get("api-key"); got != "azure-key" {
			t.Fatalf("api-key = %q, want azure-key", got)
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %q, want empty", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "gpt-5.5" {
			t.Fatalf("model = %#v, want deployment name", body["model"])
		}
		if _, ok := body["input"]; !ok {
			t.Fatalf("body = %#v, want responses input", body)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n"))
	}))
	defer server.Close()

	got, err := callAzureOpenAI(
		context.Background(),
		domain.ProviderConfig{ID: "azure-openai", Type: string(TransportAzureOpenAI), BaseURL: server.URL + "/openai/v1"},
		domain.ModelRef{ProviderID: "azure-openai", ModelID: "gpt-5.5"},
		llmCredential{APIKey: "azure-key"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		nil,
		"",
		"",
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "ok" {
		t.Fatalf("reply = %q, want ok", got.Text)
	}
}

func TestCallOpenAICompatibleUsesResponsesForXAIHostedWebSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Fatalf("path = %q, want /responses", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if _, ok := body["input"]; !ok {
			t.Fatalf("body = %#v, want responses input", body)
		}
		tools, _ := body["tools"].([]any)
		if len(tools) != 1 {
			t.Fatalf("tools = %#v, want one hosted tool", body["tools"])
		}
		tool, _ := tools[0].(map[string]any)
		if tool["type"] != "web_search" {
			t.Fatalf("tool = %#v, want xAI web_search", tool)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n"))
	}))
	defer server.Close()

	got, err := callOpenAICompatible(
		context.Background(),
		domain.ProviderConfig{ID: "xai", Type: "xai", BaseURL: server.URL},
		domain.ModelRef{ProviderID: "xai", ModelID: "grok-4.3"},
		llmCredential{APIKey: "xai-key"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		[]domain.ToolSpec{{Name: "web_search", Hosted: &domain.HostedToolSpec{Type: "web_search"}}},
		"",
		"",
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "ok" {
		t.Fatalf("reply = %q, want ok", got.Text)
	}
}

func TestCallOpenAICompatibleAppliesPerplexitySearchOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %q, want /chat/completions", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if _, ok := body["tools"]; ok {
			t.Fatalf("body tools = %#v, did not expect local function tools", body["tools"])
		}
		domains, _ := body["search_domain_filter"].([]any)
		if len(domains) != 1 || domains[0] != "example.com" {
			t.Fatalf("search_domain_filter = %#v", body["search_domain_filter"])
		}
		options, _ := body["web_search_options"].(map[string]any)
		if options["search_context_size"] != "high" {
			t.Fatalf("web_search_options = %#v", body["web_search_options"])
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n"))
	}))
	defer server.Close()

	got, err := callOpenAICompatible(
		context.Background(),
		domain.ProviderConfig{ID: "perplexity", Type: "perplexity", BaseURL: server.URL},
		domain.ModelRef{ProviderID: "perplexity", ModelID: "sonar-pro"},
		llmCredential{APIKey: "pplx-key"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		[]domain.ToolSpec{{Name: "web_search", Hosted: &domain.HostedToolSpec{Type: "perplexity_search", SearchContextSize: "high", AllowedDomains: []string{"example.com"}}}},
		"",
		"",
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "ok" {
		t.Fatalf("reply = %q, want ok", got.Text)
	}
}

func TestCallGitHubCopilotUsesChatCompletionsEndpointAndHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %q, want /chat/completions", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer copilot-token" {
			t.Fatalf("Authorization = %q, want bearer token", got)
		}
		if got := r.Header.Get("Copilot-Integration-Id"); got == "" {
			t.Fatalf("Copilot-Integration-Id = empty")
		}
		if got := r.Header.Get("Editor-Version"); got == "" {
			t.Fatalf("Editor-Version = empty")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "gpt-5.1-codex-max" {
			t.Fatalf("model = %#v, want gpt-5.1-codex-max", body["model"])
		}
		if body["stream"] != true {
			t.Fatalf("stream = %#v, want true", body["stream"])
		}
		if body["reasoning_effort"] != "low" {
			t.Fatalf("reasoning_effort = %#v, want low", body["reasoning_effort"])
		}
		messages, ok := body["messages"].([]any)
		if !ok || len(messages) != 1 {
			t.Fatalf("messages = %#v, want one message", body["messages"])
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n"))
	}))
	defer server.Close()

	got, err := callGitHubCopilot(
		context.Background(),
		domain.ProviderConfig{ID: "github-copilot", Type: string(TransportGitHubCopilot), BaseURL: server.URL},
		domain.ModelRef{ProviderID: "github-copilot", ModelID: "gpt-5.1-codex-max"},
		llmCredential{APIKey: "copilot-token"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		nil,
		"low",
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "ok" {
		t.Fatalf("reply = %q, want ok", got.Text)
	}
}

func TestExtractChatResponseReadsOpenAIUsage(t *testing.T) {
	resp := extractChatResponse([]byte(`{
		"choices":[{"message":{"content":"ok"}}],
		"usage":{"prompt_tokens":11,"completion_tokens":7,"total_tokens":18}
	}`))
	if resp.Usage == nil {
		t.Fatal("usage is nil")
	}
	if resp.Usage.InputTokens != 11 || resp.Usage.OutputTokens != 7 || resp.Usage.TotalTokens != 18 {
		t.Fatalf("usage = %+v, want 11/7/18", resp.Usage)
	}
}

func TestExtractChatResponseReadsOpenAICompatibleContentPartArrays(t *testing.T) {
	resp := extractChatResponse([]byte(`{
		"choices":[{"message":{"content":[{"type":"text","text":"ok"}]}}]
	}`))

	if resp.Text != "ok" {
		t.Fatalf("Text = %q, want ok", resp.Text)
	}
}

func TestExtractChatResponseReadsCommonProviderTextEnvelopes(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "ollama chat message", raw: `{"message":{"role":"assistant","content":"ok"}}`},
		{name: "ollama generate response", raw: `{"response":"ok"}`},
		{name: "wrapped responses payload", raw: `{"response":{"output_text":"ok"}}`},
		{name: "bedrock converse output", raw: `{"output":{"message":{"content":[{"text":"ok"}]}}}`},
		{name: "legacy completion text", raw: `{"choices":[{"text":"ok"}]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := extractChatResponse([]byte(tt.raw))
			if resp.Text != "ok" {
				t.Fatalf("Text = %q, want ok", resp.Text)
			}
		})
	}
}

func TestExtractChatResponseReadsGeminiUsageMetadata(t *testing.T) {
	resp := extractChatResponse([]byte(`{
		"candidates":[{"content":{"parts":[{"text":"ok"}]}}],
		"usageMetadata":{"promptTokenCount":13,"candidatesTokenCount":5,"totalTokenCount":18}
	}`))
	if resp.Usage == nil {
		t.Fatal("usage is nil")
	}
	if resp.Usage.InputTokens != 13 || resp.Usage.OutputTokens != 5 || resp.Usage.TotalTokens != 18 {
		t.Fatalf("usage = %+v, want 13/5/18", resp.Usage)
	}
}

func TestCallBedrockConverseSignsRequestAndParsesResponse(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIATEST")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	t.Setenv("AWS_REGION", "us-west-2")
	var gotPath string
	var gotAuth string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		gotAuth = r.Header.Get("Authorization")
		if r.Header.Get("X-Amz-Date") == "" || r.Header.Get("X-Amz-Content-Sha256") == "" {
			t.Fatalf("missing SigV4 headers: %+v", r.Header)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"output":{"message":{"content":[
				{"text":"ok"},
				{"toolUse":{"toolUseId":"tool-1","name":"read_file","input":{"path":"README.md"}}}
			]}},
			"usage":{"inputTokens":12,"outputTokens":5,"totalTokens":17}
		}`))
	}))
	defer server.Close()

	resp, err := callBedrockConverse(
		context.Background(),
		domain.ProviderConfig{ID: "amazon-bedrock", Type: string(TransportBedrockConverse), BaseURL: server.URL},
		domain.ModelRef{ProviderID: "amazon-bedrock", ModelID: "anthropic.claude-sonnet-4-20250514-v1:0"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "system", Text: "be concise"}, {Role: "user", Text: "hello"}},
		[]domain.ToolSpec{{Name: "read_file", Description: "Read file", InputSchema: map[string]any{"type": "object"}}},
		"high",
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/model/anthropic.claude-sonnet-4-20250514-v1:0/converse" {
		t.Fatalf("path = %q", gotPath)
	}
	if !strings.HasPrefix(gotAuth, "AWS4-HMAC-SHA256 Credential=AKIATEST/") || !strings.Contains(gotAuth, "/us-west-2/bedrock/aws4_request") {
		t.Fatalf("Authorization = %q, want SigV4 credential scope", gotAuth)
	}
	if _, ok := gotBody["messages"].([]any); !ok {
		t.Fatalf("messages missing from body: %#v", gotBody)
	}
	if _, ok := gotBody["toolConfig"].(map[string]any); !ok {
		t.Fatalf("toolConfig missing from body: %#v", gotBody)
	}
	if resp.Text != "ok" {
		t.Fatalf("Text = %q, want ok", resp.Text)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].ID != "tool-1" || resp.ToolCalls[0].Name != "read_file" || !strings.Contains(string(resp.ToolCalls[0].Arguments), "README.md") {
		t.Fatalf("ToolCalls = %+v", resp.ToolCalls)
	}
	if resp.Usage == nil || resp.Usage.InputTokens != 12 || resp.Usage.OutputTokens != 5 || resp.Usage.TotalTokens != 17 {
		t.Fatalf("Usage = %+v, want 12/5/17", resp.Usage)
	}
}

func TestCallOpenAICompatibleStreamsWithNonSSEContentTypeWhenDeltaHandlerExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n"))
	}))
	defer server.Close()

	var deltas []string
	got, err := callOpenAICompatible(
		context.Background(),
		domain.ProviderConfig{ID: "openrouter", Type: "openrouter", BaseURL: server.URL},
		domain.ModelRef{ProviderID: "openrouter", ModelID: "openai/gpt-5"},
		llmCredential{APIKey: "test-key"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		nil,
		"",
		"",
		func(delta string) {
			deltas = append(deltas, delta)
		},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "ok" {
		t.Fatalf("reply = %q, want ok", got.Text)
	}
	if len(deltas) != 1 || deltas[0] != "ok" {
		t.Fatalf("deltas = %#v, want [ok]", deltas)
	}
}

func TestCallAnthropicUsesMessagesStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages" {
			t.Fatalf("path = %q, want /messages", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "anthropic-key" {
			t.Fatalf("x-api-key = %q, want anthropic-key", got)
		}
		if got := r.Header.Get("anthropic-version"); got == "" {
			t.Fatal("anthropic-version header is empty")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "claude-sonnet-4-5" {
			t.Fatalf("model = %#v", body["model"])
		}
		if stream, _ := body["stream"].(bool); !stream {
			t.Fatalf("stream = %#v, want true", body["stream"])
		}
		if body["max_tokens"] != float64(64000) {
			t.Fatalf("max_tokens = %#v, want 64000", body["max_tokens"])
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"ok\"}}\n\n"))
	}))
	defer server.Close()

	got, err := callAnthropic(
		context.Background(),
		domain.ProviderConfig{ID: "anthropic", Type: "anthropic", BaseURL: server.URL},
		domain.ModelRef{ProviderID: "anthropic", ModelID: "claude-sonnet-4-5"},
		llmCredential{APIKey: "anthropic-key"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		nil,
		"high",
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "ok" {
		t.Fatalf("reply = %q, want ok", got.Text)
	}
}

func TestCallGoogleUsesStreamGenerateContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models/gemini-2.5-pro:streamGenerateContent" {
			t.Fatalf("path = %q, want /models/gemini-2.5-pro:streamGenerateContent", r.URL.Path)
		}
		if got := r.URL.Query().Get("alt"); got != "sse" {
			t.Fatalf("alt = %q, want sse", got)
		}
		if got := r.URL.Query().Get("key"); got != "gemini-key" {
			t.Fatalf("key = %q, want gemini-key", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if _, ok := body["contents"].([]any); !ok {
			t.Fatalf("contents = %#v, want array", body["contents"])
		}
		generationConfig, _ := body["generationConfig"].(map[string]any)
		thinkingConfig, _ := generationConfig["thinkingConfig"].(map[string]any)
		if thinkingConfig["thinkingBudget"] != float64(1024) {
			t.Fatalf("thinkingBudget = %#v, want 1024", thinkingConfig["thinkingBudget"])
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"ok\"}]}}]}\n\n"))
	}))
	defer server.Close()

	got, err := callGoogle(
		context.Background(),
		domain.ProviderConfig{ID: "gemini", Type: "google", BaseURL: server.URL},
		domain.ModelRef{ProviderID: "gemini", ModelID: "gemini-2.5-pro"},
		llmCredential{APIKey: "gemini-key"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		nil,
		"low",
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "ok" {
		t.Fatalf("reply = %q, want ok", got.Text)
	}
}

func TestCallGoogleVertexUsesPublisherEndpointAndBearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models/gemini-2.5-pro:streamGenerateContent" {
			t.Fatalf("path = %q, want /models/gemini-2.5-pro:streamGenerateContent", r.URL.Path)
		}
		if got := r.URL.Query().Get("alt"); got != "sse" {
			t.Fatalf("alt = %q, want sse", got)
		}
		if got := r.URL.Query().Get("key"); got != "" {
			t.Fatalf("key = %q, want empty", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer vertex-token" {
			t.Fatalf("Authorization = %q, want bearer token", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if _, ok := body["contents"].([]any); !ok {
			t.Fatalf("contents = %#v, want array", body["contents"])
		}
		generationConfig, _ := body["generationConfig"].(map[string]any)
		thinkingConfig, _ := generationConfig["thinkingConfig"].(map[string]any)
		if thinkingConfig["thinkingBudget"] != float64(1024) {
			t.Fatalf("thinkingBudget = %#v, want 1024", thinkingConfig["thinkingBudget"])
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"ok\"}]}}]}\n\n"))
	}))
	defer server.Close()

	got, err := callGoogleVertex(
		context.Background(),
		domain.ProviderConfig{ID: "google-vertex", Type: string(TransportGoogleVertex), BaseURL: server.URL},
		domain.ModelRef{ProviderID: "google-vertex", ModelID: "gemini-2.5-pro"},
		llmCredential{APIKey: "vertex-token"},
		domain.ProviderRequestProfile{},
		[]llmChatMessage{{Role: "user", Text: "hello"}},
		nil,
		"low",
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "ok" {
		t.Fatalf("reply = %q, want ok", got.Text)
	}
}
