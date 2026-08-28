package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"aivo/core/domain"
)

func TestFirstPrimaryRequestReceivesHostResolvedSkillInventoryAndDefaultTools(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()

	writeSkill(t, filepath.Join(home, ".claude", "skills", "ui-components"), "ui-components", "Build reusable UI components", "Use the full UI workflow.")
	scan, err := service.ScanGlobalSkills(ctx)
	if err != nil || len(scan.Candidates) != 1 {
		t.Fatalf("scan = %#v, err = %v", scan, err)
	}
	if _, err := service.ImportSkill(ctx, domain.SkillImportInput{CandidateID: scan.Candidates[0].ID, TargetScope: domain.SkillScopeGlobal}); err != nil {
		t.Fatal(err)
	}

	var capturedMessages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	var capturedTools []struct {
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
			Tools []struct {
				Function struct {
					Name string `json:"name"`
				} `json:"function"`
			} `json:"tools"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		capturedMessages = body.Messages
		capturedTools = body.Tools
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"found"}}]}`))
	}))
	defer server.Close()
	if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{ProviderID: "custom-api", Type: "openai-compatible", BaseURL: server.URL, ModelID: "test-model", APIKey: "test-key", Method: "api-key"}); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "Skill inventory", ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "当前有哪些 UI 组件技能"}); err != nil {
		t.Fatal(err)
	}
	joined := ""
	for _, message := range capturedMessages {
		joined += "\n" + message.Content
	}
	if !strings.Contains(joined, "<host_preactivated_resources>") || !strings.Contains(joined, "<name>ui-components</name>") {
		t.Fatalf("primary request did not receive Host-resolved Skill inventory: %s", joined)
	}
	if strings.Contains(joined, "call the skill tool") || !strings.Contains(joined, "call tool_resolve") {
		t.Fatalf("primary request missing the replaceable tool-selection protocol: %s", joined)
	}
	wantCoreTools := []string{"read", "bash", "edit", "write", "update_plan", "ask_user", ToolResolveName}
	if len(capturedTools) != len(wantCoreTools) {
		t.Fatalf("provider tools = %#v, want six core tools plus the Host selection control", capturedTools)
	}
	for index, want := range wantCoreTools {
		if capturedTools[index].Function.Name != want {
			t.Fatalf("provider tool[%d] = %q, want %q", index, capturedTools[index].Function.Name, want)
		}
	}
	automatic, initialized := service.autoSelectedTools(ctx, session.ID)
	if !initialized || len(automatic) != 0 {
		t.Fatalf("automatic selection = %#v initialized=%t, want initialized empty inventory-task set", automatic, initialized)
	}
	activeIDs, _ := service.activeSkills(ctx, session.ID)
	if len(activeIDs) != 0 {
		t.Fatalf("automatic inventory persisted Skill activation: %v", activeIDs)
	}
}

func TestHostUsesOneAuxiliaryResolutionForToolSkillMCPAndExtensionContextCandidates(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()

	writeSkill(t, filepath.Join(home, ".claude", "skills", "ui-review"), "ui-review", "Review UI accessibility", "Check focus order and accessible names.")
	scan, err := service.ScanGlobalSkills(ctx)
	if err != nil || len(scan.Candidates) != 1 {
		t.Fatalf("scan = %#v, err = %v", scan, err)
	}
	if _, err := service.ImportSkill(ctx, domain.SkillImportInput{CandidateID: scan.Candidates[0].ID, TargetScope: domain.SkillScopeGlobal}); err != nil {
		t.Fatal(err)
	}

	extensionRoot := t.TempDir()
	writeTestFile(t, filepath.Join(extensionRoot, "context.md"), "Use the extension UI review checklist.", 0o600)
	writeTestExtensionManifest(t, extensionRoot, map[string]any{
		"schemaVersion": 2, "id": "com.example.ui", "name": "UI Inspector", "description": "Inspect UI accessibility", "version": "1", "apiVersion": "2",
		"runtime": map[string]any{"type": "builtin"},
		"contributes": map[string]any{
			"toolGroups": []any{map[string]any{
				"id": "ui-review", "name": "UI Inspector", "description": "Inspect UI accessibility",
				"tools": []string{"example_inspect_ui", "example_capture_ui"},
			}},
			"tools": []any{
				map[string]any{"name": "example_inspect_ui", "description": "Inspect UI accessibility", "schema": map[string]any{"type": "object"}, "activation": "auto"},
				map[string]any{"name": "example_capture_ui", "description": "Capture the current UI", "schema": map[string]any{"type": "object"}, "activation": "auto"},
			},
			"contexts": []any{map[string]any{"id": "ui-checklist", "kind": "instructions", "path": "context.md"}},
		},
	})
	events := []string{}
	service.extensionSupervisor.RegisterBuiltin("com.example.ui", func() extensionRuntimeClient { return &builtinExtensionTestClient{events: &events} })
	status, err := service.extensionSupervisor.Discover(extensionRoot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.extensionSupervisor.Enable(ctx, status.ID); err != nil {
		t.Fatal(err)
	}
	mcpServer, err := service.mcpManager.store.SaveMCPServer(ctx, domain.MCPServerConfig{
		ID: "docs", Name: "Docs", Description: "Search UI documentation", Transport: domain.MCPTransportStdio,
		Command: "not-started-with-cached-catalog", Enabled: true, Status: domain.MCPServerStatusReady,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.mcpManager.store.ReplaceMCPTools(ctx, mcpServer.ID, []domain.MCPToolRecord{
		{ID: "docs:search_docs", ServerID: mcpServer.ID, Name: "search_docs", Description: "Search UI documentation", InputSchema: map[string]any{"type": "object"}},
		{ID: "docs:read_docs", ServerID: mcpServer.ID, Name: "read_docs", Description: "Read UI documentation", InputSchema: map[string]any{"type": "object"}},
	}); err != nil {
		t.Fatal(err)
	}

	type capturedRequest struct {
		Messages []struct {
			Content string `json:"content"`
		} `json:"messages"`
		Tools []struct {
			Function struct {
				Name string `json:"name"`
			} `json:"function"`
		} `json:"tools"`
	}
	var mu sync.Mutex
	requests := []capturedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body capturedRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		mu.Lock()
		requests = append(requests, body)
		index := len(requests)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch index {
		case 1:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"intent\":\"use\",\"resources\":[{\"kind\":\"extension\",\"id\":\"extension_com_example_ui_ui_review\"},{\"kind\":\"mcp\",\"id\":\"mcp_group_docs\"}]}"}}]}`))
		case 2:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"resources\":[\"skill:ui-review\",\"context:com.example.ui:ui-checklist\"],\"skillInstructions\":[\"skill:ui-review\"],\"reason\":\"direct matches\"}"}}]}`))
		default:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"resolved"}}]}`))
		}
	}))
	defer server.Close()
	if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{ProviderID: "custom-api", Type: "openai-compatible", BaseURL: server.URL, ModelID: "test-model", APIKey: "test-key", Method: "api-key"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateModelPreferences(ctx, domain.ModelPreferencesInput{
		Model: &domain.ModelRef{ProviderID: "custom-api", ModelID: "test-model"}, AuxiliaryModel: &domain.ModelRef{ProviderID: "custom-api", ModelID: "test-model"},
	}); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "UI review", ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	type visibleSelectionUpdate struct {
		Created bool
		Event   domain.SessionEvent
	}
	visibleSelectionUpdates := []visibleSelectionUpdate{}
	service.SetSessionEventUpdatedHook(func(event domain.SessionEvent, created bool) {
		if event.Payload["kind"] == "host_tool_selection" {
			visibleSelectionUpdates = append(visibleSelectionUpdates, visibleSelectionUpdate{Created: created, Event: event})
		}
	})
	if _, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "Review this UI accessibility"}); err != nil {
		t.Fatal(err)
	}
	if len(visibleSelectionUpdates) != 2 {
		t.Fatalf("live visible selection updates = %#v, want running creation and completed update", visibleSelectionUpdates)
	}
	if !visibleSelectionUpdates[0].Created || visibleSelectionUpdates[0].Event.Payload["status"] != "running" {
		t.Fatalf("first visible selection update = %#v, want created running event", visibleSelectionUpdates[0])
	}
	if visibleSelectionUpdates[1].Created || visibleSelectionUpdates[1].Event.Payload["status"] != "completed" {
		t.Fatalf("second visible selection update = %#v, want completed update", visibleSelectionUpdates[1])
	}
	if visibleSelectionUpdates[0].Event.ID != visibleSelectionUpdates[1].Event.ID {
		t.Fatalf("visible selection changed event identity: %#v", visibleSelectionUpdates)
	}

	mu.Lock()
	captured := append([]capturedRequest(nil), requests...)
	mu.Unlock()
	if len(captured) != 3 {
		t.Fatalf("provider request count = %d, want tool-group resolution, resource resolution, and primary request", len(captured))
	}
	toolGroupText := ""
	for _, message := range captured[0].Messages {
		toolGroupText += message.Content
	}
	if !strings.Contains(toolGroupText, "Host tool-resource selector") || !strings.Contains(toolGroupText, "extension:extension_com_example_ui_ui_review：UI Inspector｜Inspect UI accessibility") || !strings.Contains(toolGroupText, "mcp:mcp_group_docs：Docs｜Search UI documentation") || strings.Contains(toolGroupText, "example_capture_ui") || strings.Contains(toolGroupText, "mcp_docs_read_docs") || len(captured[0].Tools) != 0 {
		t.Fatalf("first request was not the minimal grouped-or-individual resolver: %#v", captured[0])
	}
	primaryText := ""
	for _, message := range captured[2].Messages {
		primaryText += message.Content
	}
	if !strings.Contains(primaryText, `<skill_summary name="ui-review"`) || !strings.Contains(primaryText, `<skill_content name="ui-review"`) || !strings.Contains(primaryText, `<extension_context extension="com.example.ui" id="ui-checklist"`) {
		t.Fatalf("primary request missing selected Skill/context: %s", primaryText)
	}
	toolNames := map[string]bool{}
	for _, tool := range captured[2].Tools {
		toolNames[tool.Function.Name] = true
	}
	if len(toolNames) != 14 || !toolNames["read"] || !toolNames["bash"] || !toolNames["edit"] || !toolNames["write"] || !toolNames["update_plan"] || !toolNames["ask_user"] || !toolNames[ToolResolveName] || !toolNames["example_inspect_ui"] || !toolNames["example_capture_ui"] || !toolNames["mcp_docs_search_docs"] || !toolNames["mcp_docs_read_docs"] || !toolNames["mcp_host_docs_list_resource_templates"] || !toolNames["mcp_host_docs_list_resources"] || !toolNames["mcp_host_docs_read_resource"] {
		t.Fatalf("primary tools = %#v, want core, selection control, and every member of the selected groups", captured[2].Tools)
	}
	automatic, initialized := service.autoSelectedTools(ctx, session.ID)
	if !initialized || !automatic["example_inspect_ui"] || !automatic["example_capture_ui"] || !automatic["mcp_docs_search_docs"] || !automatic["mcp_docs_read_docs"] || !automatic["mcp_host_docs_list_resource_templates"] || !automatic["mcp_host_docs_list_resources"] || !automatic["mcp_host_docs_read_resource"] || len(automatic) != 7 {
		t.Fatalf("persisted automatic selection = %#v initialized=%t", automatic, initialized)
	}
	assertVisibleInitialToolSelection(t, service, ctx, session.ID, "conversation", []hostToolSelectionResource{
		{Kind: "extension", ID: "extension_com_example_ui_ui_review", Name: "UI Inspector", ToolCount: 2},
		{Kind: "mcp", ID: "mcp_group_docs", Name: "Docs", ToolCount: 5},
	})
	activeIDs, _ := service.activeSkills(ctx, session.ID)
	if len(activeIDs) != 0 {
		t.Fatalf("automatic Skill selection persisted across requests: %v", activeIDs)
	}
	if _, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "Review the same UI again"}); err != nil {
		t.Fatal(err)
	}
	if len(visibleSelectionUpdates) != 2 {
		t.Fatalf("later turn emitted duplicate visible selection updates: %#v", visibleSelectionUpdates)
	}
	assertVisibleInitialToolSelection(t, service, ctx, session.ID, "conversation", []hostToolSelectionResource{
		{Kind: "extension", ID: "extension_com_example_ui_ui_review", Name: "UI Inspector", ToolCount: 2},
		{Kind: "mcp", ID: "mcp_group_docs", Name: "Docs", ToolCount: 5},
	})
}

func TestToolInventoryInspectionInjectsAllEligibleToolsForOneProviderRequest(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()

	extensionRoot := t.TempDir()
	const inventoryToolCount = 66
	contributedTools := make([]any, 0, inventoryToolCount)
	groupToolNames := make([]string, 0, inventoryToolCount)
	for index := 0; index < inventoryToolCount; index++ {
		toolName := fmt.Sprintf("inventory_tool_%02d", index)
		contributedTools = append(contributedTools, map[string]any{
			"name":        toolName,
			"description": "Inspect inventory data",
			"schema":      map[string]any{"type": "object"},
			"activation":  "auto",
		})
		groupToolNames = append(groupToolNames, toolName)
	}
	writeTestExtensionManifest(t, extensionRoot, map[string]any{
		"schemaVersion": 2, "id": "com.example.inventory", "name": "Inventory tools", "description": "Inspect inventory data", "version": "1", "apiVersion": "2",
		"runtime": map[string]any{"type": "builtin"},
		"contributes": map[string]any{
			"toolGroups": []any{map[string]any{
				"id": "inventory", "name": "Inventory tools", "description": "Inspect inventory data", "tools": groupToolNames,
			}},
			"tools": contributedTools,
		},
	})
	events := []string{}
	service.extensionSupervisor.RegisterBuiltin("com.example.inventory", func() extensionRuntimeClient {
		return &builtinExtensionTestClient{events: &events}
	})
	status, err := service.extensionSupervisor.Discover(extensionRoot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.extensionSupervisor.Enable(ctx, status.ID); err != nil {
		t.Fatal(err)
	}

	type capturedRequest struct {
		Tools []struct {
			Function struct {
				Name string `json:"name"`
			} `json:"function"`
		} `json:"tools"`
	}
	var mu sync.Mutex
	requests := []capturedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body capturedRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		mu.Lock()
		requests = append(requests, body)
		index := len(requests)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if index == 1 {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"intent\":\"inspect\",\"resources\":[]}"}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"inventory response"}}]}`))
	}))
	defer server.Close()
	if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{ProviderID: "custom-api", Type: "openai-compatible", BaseURL: server.URL, ModelID: "test-model", APIKey: "test-key", Method: "api-key"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateModelPreferences(ctx, domain.ModelPreferencesInput{
		Model: &domain.ModelRef{ProviderID: "custom-api", ModelID: "test-model"}, AuxiliaryModel: &domain.ModelRef{ProviderID: "custom-api", ModelID: "test-model"},
	}); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "Tool inventory", ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "当前有哪些工具可调用"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "谢谢"}); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	captured := append([]capturedRequest(nil), requests...)
	mu.Unlock()
	if len(captured) != 3 {
		t.Fatalf("provider request count = %d, want auxiliary inspection and two primary requests", len(captured))
	}
	firstPrimary := map[string]bool{}
	for _, tool := range captured[1].Tools {
		firstPrimary[tool.Function.Name] = true
	}
	secondPrimary := map[string]bool{}
	for _, tool := range captured[2].Tools {
		secondPrimary[tool.Function.Name] = true
	}
	for index := 0; index < inventoryToolCount; index++ {
		name := fmt.Sprintf("inventory_tool_%02d", index)
		if !firstPrimary[name] {
			t.Fatalf("inspection Provider tools are missing %q from the complete eligible extension group", name)
		}
	}
	if !firstPrimary[ToolResolveName] {
		t.Fatalf("inspection Provider tools are missing %q", ToolResolveName)
	}
	if secondPrimary["inventory_tool_00"] || secondPrimary["inventory_tool_65"] || !secondPrimary[ToolResolveName] {
		t.Fatalf("later Provider tools = %#v, request-only tools leaked into the conversation", secondPrimary)
	}
	automatic, initialized := service.autoSelectedTools(ctx, session.ID)
	if !initialized || len(automatic) != 0 {
		t.Fatalf("inspection automatic set = %#v initialized=%t, want initialized empty", automatic, initialized)
	}
	selectionEvents, err := service.ListEvents(ctx, session.ID, false, 500)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, event := range selectionEvents {
		if event.Type != domain.EventTypeSystemNote || event.Payload["kind"] != "host_tool_selection" {
			continue
		}
		found = true
		if event.Payload["lifetime"] != "request" {
			t.Fatalf("inspection event lifetime = %#v", event.Payload["lifetime"])
		}
		selected := hostToolSelectionResourcesFromAny(event.Payload["resources"])
		var inventory *hostToolSelectionResource
		for index := range selected {
			if selected[index].Kind == "extension" && selected[index].ID == "extension_com_example_inventory_inventory" {
				inventory = &selected[index]
				break
			}
		}
		if inventory == nil || inventory.ToolCount != inventoryToolCount {
			t.Fatalf("inspection event resources = %#v, want one unsplit inventory extension source beyond the legacy member limit", selected)
		}
	}
	if !found {
		t.Fatal("missing visible request-only tool-selection event")
	}
}

func TestAuxiliarySummaryOnlySkillSelectionInjectsCanonicalSummary(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()

	writeSkill(t, filepath.Join(home, ".claude", "skills", "ui-summary"), "ui-summary", "Explain the available UI component workflow", "PRIVATE_UI_WORKFLOW")
	scan, err := service.ScanGlobalSkills(ctx)
	if err != nil || len(scan.Candidates) != 1 {
		t.Fatalf("scan = %#v, err = %v", scan, err)
	}
	if _, err := service.ImportSkill(ctx, domain.SkillImportInput{CandidateID: scan.Candidates[0].ID, TargetScope: domain.SkillScopeGlobal}); err != nil {
		t.Fatal(err)
	}

	type capturedRequest struct {
		Messages []struct {
			Content string `json:"content"`
		} `json:"messages"`
	}
	var mu sync.Mutex
	requests := []capturedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body capturedRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		mu.Lock()
		requests = append(requests, body)
		index := len(requests)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if index == 1 {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"intent\":\"use\",\"resources\":[]}"}}]}`))
			return
		}
		if index == 2 {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"tools\":[],\"resources\":[\"skill:ui-summary\"],\"reason\":\"summary is sufficient\"}"}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"summary received"}}]}`))
	}))
	defer server.Close()
	if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{ProviderID: "custom-api", Type: "openai-compatible", BaseURL: server.URL, ModelID: "test-model", APIKey: "test-key", Method: "api-key"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateModelPreferences(ctx, domain.ModelPreferencesInput{
		Model: &domain.ModelRef{ProviderID: "custom-api", ModelID: "test-model"}, AuxiliaryModel: &domain.ModelRef{ProviderID: "custom-api", ModelID: "test-model"},
	}); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "Skill summary", ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "Explain what the ui-summary Skill offers"}); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	captured := append([]capturedRequest(nil), requests...)
	mu.Unlock()
	if len(captured) != 3 {
		t.Fatalf("provider request count = %d, want tool-resource auxiliary, instruction auxiliary, and primary", len(captured))
	}
	auxiliaryText := ""
	for _, message := range captured[1].Messages {
		auxiliaryText += message.Content
	}
	if !strings.Contains(auxiliaryText, "Explain the available UI component workflow") || strings.Contains(auxiliaryText, "PRIVATE_UI_WORKFLOW") {
		t.Fatalf("auxiliary catalog was not summary-only: %s", auxiliaryText)
	}
	primaryText := ""
	for _, message := range captured[2].Messages {
		primaryText += message.Content
	}
	if !strings.Contains(primaryText, `<skill_summary name="ui-summary"`) || !strings.Contains(primaryText, "Explain the available UI component workflow") {
		t.Fatalf("primary request missing canonical Skill summary: %s", primaryText)
	}
	if strings.Contains(primaryText, "PRIVATE_UI_WORKFLOW") || strings.Contains(primaryText, `<skill_content name="ui-summary"`) {
		t.Fatalf("summary-only selection exposed Skill instructions: %s", primaryText)
	}
	assertVisibleInitialToolSelection(t, service, ctx, session.ID, "conversation", nil)
}

func assertVisibleInitialToolSelection(t *testing.T, service *Service, ctx context.Context, sessionID, lifetime string, want []hostToolSelectionResource) {
	t.Helper()
	events, err := service.ListEvents(ctx, sessionID, false, 500)
	if err != nil {
		t.Fatal(err)
	}
	matching := make([]domain.SessionEvent, 0, 1)
	for _, event := range events {
		if event.Type == domain.EventTypeSystemNote && event.Payload["kind"] == "host_tool_selection" {
			matching = append(matching, event)
		}
	}
	if len(matching) != 1 {
		t.Fatalf("visible initial tool-selection events = %#v, want exactly one", matching)
	}
	event := matching[0]
	got := hostToolSelectionResourcesFromAny(event.Payload["resources"])
	if !reflect.DeepEqual(got, normalizeHostToolSelectionResources(want)) {
		t.Fatalf("visible initial tool selection = %#v, want %#v", got, want)
	}
	if event.Payload["lifetime"] != lifetime {
		t.Fatalf("visible initial tool selection lifetime = %#v, want %q", event.Payload["lifetime"], lifetime)
	}
	if event.Payload["status"] != "completed" {
		t.Fatalf("visible initial tool selection status = %#v, want completed", event.Payload["status"])
	}
	for _, forbidden := range []string{"reason", "description", "schema", "prompt", "candidates"} {
		if _, ok := event.Payload[forbidden]; ok {
			t.Fatalf("visible initial selection exposed %q: %#v", forbidden, event.Payload)
		}
	}
}

func hostToolSelectionResourcesFromAny(value any) []hostToolSelectionResource {
	raw, _ := json.Marshal(value)
	var resources []hostToolSelectionResource
	_ = json.Unmarshal(raw, &resources)
	return resources
}

func TestHostPreCallSkillInventoryIsInjectedWithoutSessionActivation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()

	writeSkill(t, filepath.Join(home, ".claude", "skills", "ui-components"), "ui-components", "Build and inspect reusable UI components", "Use the private UI component workflow.")
	scan, err := service.ScanGlobalSkills(ctx)
	if err != nil || len(scan.Candidates) != 1 {
		t.Fatalf("scan = %#v, err = %v", scan, err)
	}
	if _, err := service.ImportSkill(ctx, domain.SkillImportInput{CandidateID: scan.Candidates[0].ID, TargetScope: domain.SkillScopeGlobal}); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	contextText := service.preCallInstructionContext(ctx, session.ID, "当前有哪些 UI 组件技能", domain.AgentModeCode, session.ProjectPath)
	if !strings.Contains(contextText, "<available_skills>") || !strings.Contains(contextText, "<name>ui-components</name>") || !strings.Contains(contextText, "Build and inspect reusable UI components") {
		t.Fatalf("pre-call inventory missing enabled Skill: %q", contextText)
	}
	if strings.Contains(contextText, "private UI component workflow") || strings.Contains(contextText, "skill tool") {
		t.Fatalf("inventory exposed full Skill content or stale discovery instructions: %q", contextText)
	}
	activeIDs, _ := service.activeSkills(ctx, session.ID)
	if len(activeIDs) != 0 {
		t.Fatalf("inventory query persisted automatic activation: %v", activeIDs)
	}
	repeated := service.preCallInstructionContext(ctx, session.ID, "列出当前技能", domain.AgentModeCode, session.ProjectPath)
	if repeated != contextText {
		t.Fatalf("repeated inventory resolution changed: first=%q second=%q", contextText, repeated)
	}
	other, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	otherActive, _ := service.activeSkills(ctx, other.ID)
	if len(otherActive) != 0 {
		t.Fatalf("automatic Skill activation leaked to another conversation: %v", otherActive)
	}
}

func TestHostPreCallRendersOnlyValidatedSkillAndExtensionContextSelections(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()

	writeSkill(t, filepath.Join(home, ".claude", "skills", "ui-review"), "ui-review", "Review UI component accessibility", "Check focus order and accessible names.")
	scan, err := service.ScanGlobalSkills(ctx)
	if err != nil || len(scan.Candidates) != 1 {
		t.Fatalf("scan = %#v, err = %v", scan, err)
	}
	if _, err := service.ImportSkill(ctx, domain.SkillImportInput{CandidateID: scan.Candidates[0].ID, TargetScope: domain.SkillScopeGlobal}); err != nil {
		t.Fatal(err)
	}

	extensionRoot := t.TempDir()
	writeTestFile(t, filepath.Join(extensionRoot, "context", "tokens.md"), "Use semantic color and spacing tokens.", 0o600)
	writeTestExtensionManifest(t, extensionRoot, map[string]any{
		"schemaVersion": 2, "id": "com.example.design", "name": "Design Context", "description": "UI design token guidance", "version": "1", "apiVersion": "2",
		"runtime":     map[string]any{"type": "static"},
		"contributes": map[string]any{"contexts": []any{map[string]any{"id": "design-tokens", "kind": "instructions", "path": "context/tokens.md"}}},
	})
	status, err := service.extensionSupervisor.Discover(extensionRoot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.extensionSupervisor.Enable(ctx, status.ID); err != nil {
		t.Fatal(err)
	}

	skills, err := service.hostSkillCandidates(ctx, "")
	if err != nil || len(skills) != 1 {
		t.Fatalf("skills = %#v, err = %v", skills, err)
	}
	contexts := service.extensionSupervisor.ContextCatalog()
	if len(contexts) != 1 {
		t.Fatalf("contexts = %#v", contexts)
	}
	candidates := []hostInstructionCandidate{
		{Key: "skill:" + skills[0].Name, Kind: "skill", Name: skills[0].Name, Description: skills[0].Description, Source: skills[0].Source, Skill: &skills[0]},
		{Key: contexts[0].Key, Kind: "context", Name: contexts[0].Name, Description: contexts[0].Description, Source: contexts[0].ExtensionID, Context: &contexts[0]},
	}
	selected := validateHostInstructionSelection(candidates, []string{"skill:ui-review", contexts[0].Key, "skill:not-installed", contexts[0].Key}, 4)
	if len(selected) != 2 {
		t.Fatalf("selected = %#v, want exact valid deduplicated candidates", selected)
	}
	instructionKeys := validateHostSkillInstructionSelection(selected, []string{"skill:ui-review", contexts[0].Key, "skill:not-installed"})
	rendered := service.renderSelectedHostInstructions(ctx, "", "", selected, instructionKeys)
	if !strings.Contains(rendered, `<skill_summary name="ui-review"`) || !strings.Contains(rendered, `<skill_content name="ui-review"`) || !strings.Contains(rendered, "Check focus order and accessible names.") {
		t.Fatalf("selected Skill was not rendered: %q", rendered)
	}
	if !strings.Contains(rendered, `<extension_context extension="com.example.design" id="design-tokens"`) || !strings.Contains(rendered, "Use semantic color and spacing tokens.") {
		t.Fatalf("selected extension context was not rendered: %q", rendered)
	}
	if len(rendered) > hostPreCallContextLimit {
		t.Fatalf("rendered context length = %d, limit = %d", len(rendered), hostPreCallContextLimit)
	}
}

func TestHostPreCallSummaryOnlySkillSelectionDoesNotReadInstructions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()

	writeSkill(t, filepath.Join(home, ".claude", "skills", "ui-summary"), "ui-summary", "Canonical UI summary", "PRIVATE_INSTRUCTION_BODY")
	scan, err := service.ScanGlobalSkills(ctx)
	if err != nil || len(scan.Candidates) != 1 {
		t.Fatalf("scan = %#v, err = %v", scan, err)
	}
	if _, err := service.ImportSkill(ctx, domain.SkillImportInput{CandidateID: scan.Candidates[0].ID, TargetScope: domain.SkillScopeGlobal}); err != nil {
		t.Fatal(err)
	}

	_, candidates := service.hostInstructionCandidates(ctx, "")
	selected := validateHostInstructionSelection(candidates, []string{"skill:ui-summary"}, hostInstructionSelectionLimit)
	instructionKeys := validateHostSkillInstructionSelection(selected, []string{"context:not-a-skill", "skill:not-selected"})
	rendered := service.renderSelectedHostInstructions(ctx, "", "", selected, instructionKeys)
	if !strings.Contains(rendered, `<skill_summary name="ui-summary"`) || !strings.Contains(rendered, "Canonical UI summary") {
		t.Fatalf("summary-only selection missing canonical summary: %q", rendered)
	}
	if strings.Contains(rendered, "PRIVATE_INSTRUCTION_BODY") || strings.Contains(rendered, `<skill_content name="ui-summary"`) {
		t.Fatalf("summary-only selection exposed Skill instructions: %q", rendered)
	}
}

func TestHostResourceDecisionParsesAndBoundsSkillInstructionSubset(t *testing.T) {
	decision, err := parseHostResourceDecision(`{"tools":[],"resources":["skill:ui-review","context:design:tokens"],"skillInstructions":["skill:ui-review","skill:ui-review","context:design:tokens","skill:invented"],"reason":"execute review"}`)
	if err != nil {
		t.Fatal(err)
	}
	candidates := []hostInstructionCandidate{
		{Key: "skill:ui-review", Kind: "skill", Name: "ui-review", Skill: &SkillResolveCandidate{Name: "ui-review"}},
		{Key: "context:design:tokens", Kind: "context", Name: "tokens", Context: &extensionContextCandidate{}},
	}
	selected := validateHostInstructionSelection(candidates, decision.ResourceKeys, hostInstructionSelectionLimit)
	validated := validateHostSkillInstructionSelection(selected, decision.SkillInstructionKeys)
	if len(validated) != 1 || !validated["skill:ui-review"] {
		t.Fatalf("validated Skill instruction subset = %#v", validated)
	}
}

func TestHostPreCallLocalResolutionIsBoundedAndExact(t *testing.T) {
	candidates := []hostInstructionCandidate{
		{Key: "skill:ui-review", Kind: "skill", Name: "ui-review", Description: "Review UI accessibility"},
		{Key: "context:design:tokens", Kind: "context", Name: "design tokens", Description: "Semantic UI colors and spacing"},
		{Key: "skill:database", Kind: "skill", Name: "database", Description: "Tune SQL queries"},
	}
	decision := localHostInstructionResolve("Review UI accessibility with design tokens", candidates, 2)
	selected := validateHostInstructionSelection(candidates, append(decision.Keys, "skill:invented"), 2)
	selectedKeys := map[string]bool{}
	for _, candidate := range selected {
		selectedKeys[candidate.Key] = true
	}
	if len(selected) != 2 || !selectedKeys["skill:ui-review"] || !selectedKeys["context:design:tokens"] {
		t.Fatalf("decision = %#v, selected = %#v", decision, selected)
	}
}

func TestCachedEnabledMCPWithoutToolRowsStillExposesResourceUtilities(t *testing.T) {
	store := &memoryProviderStore{
		mcpServers: []domain.MCPServerConfig{{
			ID: "resource-only", Name: "Resource Only", Description: "Documentation resources",
			Transport: domain.MCPTransportStdio, Command: "not-started-by-cached-registration", Enabled: true,
		}},
		mcpTools: map[string][]domain.MCPToolRecord{"resource-only": {}},
	}
	service := NewService(store)
	registry := service.globalToolCatalogRegistry(context.Background())
	names := map[string]bool{}
	for _, entry := range registry.CatalogEntries() {
		names[entry.Name] = true
	}
	for _, name := range []string{
		"mcp_host_resource_only_list_resources",
		"mcp_host_resource_only_list_resource_templates",
		"mcp_host_resource_only_read_resource",
	} {
		if !names[name] {
			t.Fatalf("missing resource-only MCP utility %q; entries = %#v", name, registry.CatalogEntries())
		}
	}
}

func TestFailedEnabledMCPRefreshIsExcludedFromPreCallCandidates(t *testing.T) {
	store := &memoryProviderStore{
		mcpServers: []domain.MCPServerConfig{{
			ID: "broken-mcp", Name: "Broken MCP", Transport: domain.MCPTransportStdio,
			Command: "definitely-not-a-real-mcp-command", Enabled: true,
		}},
		mcpTools: map[string][]domain.MCPToolRecord{"broken-mcp": {}},
	}
	service := NewService(store)
	failed := service.prepareEnabledToolCatalogs(context.Background())
	registry, _ := service.toolsForWorkspace(t.TempDir())
	specs := filterEligibleToolSpecs(registry, registry.Specs(), failed)
	for _, spec := range specs {
		if strings.HasPrefix(spec.Name, "mcp_host_broken_mcp_") {
			t.Fatalf("failed MCP source remained eligible: failed = %#v, specs = %#v", failed, specs)
		}
	}
	if _, ok := registry.Get("read"); !ok {
		t.Fatalf("failed MCP source removed the core registry: %#v", registry.Specs())
	}
}

func TestStoppedExtensionContextIsNotEligible(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "context.md"), "private context", 0o600)
	writeTestExtensionManifest(t, root, map[string]any{
		"schemaVersion": 2, "id": "com.example.stopped", "name": "Stopped", "version": "1", "apiVersion": "2",
		"runtime":     map[string]any{"type": "static"},
		"contributes": map[string]any{"contexts": []any{map[string]any{"id": "only", "kind": "instructions", "path": "context.md"}}},
	})
	status, err := service.extensionSupervisor.Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.extensionSupervisor.Enable(context.Background(), status.ID); err != nil {
		t.Fatal(err)
	}
	if len(service.extensionSupervisor.ContextCatalog()) != 1 {
		t.Fatal("enabled extension context was not eligible")
	}
	if _, err := service.extensionSupervisor.Stop(context.Background(), status.ID); err != nil {
		t.Fatal(err)
	}
	if catalog := service.extensionSupervisor.ContextCatalog(); len(catalog) != 0 {
		t.Fatalf("stopped extension context remained eligible: %#v", catalog)
	}
}

func TestCancelledHostCatalogPreparationPreservesCoreEligibility(t *testing.T) {
	store := &memoryProviderStore{
		mcpServers: []domain.MCPServerConfig{{
			ID: "cancelled-mcp", Name: "Cancelled MCP", Transport: domain.MCPTransportStdio,
			Command: "definitely-not-a-real-mcp-command", Enabled: true,
		}},
		mcpTools: map[string][]domain.MCPToolRecord{"cancelled-mcp": {}},
	}
	service := NewService(store)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	failed := service.prepareEnabledToolCatalogs(ctx)
	registry, _ := service.toolsForWorkspace(t.TempDir())
	specs := filterEligibleToolSpecs(registry, registry.Specs(), failed)
	names := map[string]bool{}
	for _, spec := range specs {
		names[spec.Name] = true
	}
	for _, name := range []string{"read", "bash", "edit", "write", "update_plan", "ask_user"} {
		if !names[name] {
			t.Fatalf("cancelled preparation removed core tool %q: %#v", name, specs)
		}
	}
	for name := range names {
		if strings.HasPrefix(name, "mcp_host_cancelled_mcp_") {
			t.Fatalf("cancelled MCP preparation left an unready candidate: %#v", specs)
		}
	}
}

func TestHostPreCallContextMessageIsEphemeralAndOrderedAfterAgentPrompt(t *testing.T) {
	messages := []domain.ChatMessage{{Role: domain.EventRoleSystem, Text: "agent"}, {Role: domain.EventRoleUser, Text: "request"}}
	resolved := appendHostPreCallContext(messages, "selected")
	if len(resolved) != 3 || resolved[0].Text != "agent" || !strings.Contains(resolved[1].Text, "<host_preactivated_resources>") || resolved[2].Text != "request" {
		t.Fatalf("resolved messages = %#v", resolved)
	}
	if len(messages) != 2 {
		t.Fatalf("source messages mutated: %#v", messages)
	}
}
