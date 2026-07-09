package app

import (
	"encoding/json"
	"testing"

	"aivo/core/domain"
)

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
