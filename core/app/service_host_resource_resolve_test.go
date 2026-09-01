package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"aivo/core/domain"
)

func TestFirstPrimaryRequestDoesNotPreSnapshotFilterSkillInventory(t *testing.T) {
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
	if strings.Contains(joined, "<host_selected_resources>") || strings.Contains(joined, "<available_skills>") || strings.Contains(joined, "<name>ui-components</name>") {
		t.Fatalf("primary request pre-injected Host-filtered Skill catalog: %s", joined)
	}
	if strings.Contains(joined, "Use the full UI workflow.") || strings.Contains(joined, `skill_content name="ui-components"`) {
		t.Fatalf("primary request preloaded full Skill content: %s", joined)
	}
	if !strings.Contains(joined, "call resource_resolve with mode \"inspect\"") {
		t.Fatalf("primary request missing the replaceable resource-resolution protocol: %s", joined)
	}
	wantTools := []string{"read", ExecCommandToolName, WriteStdinToolName, "edit", "write", "update_plan", "ask_user", ResourceResolveName}
	if len(capturedTools) != len(wantTools) {
		t.Fatalf("provider tools = %#v, want core tools plus Host resource control only", capturedTools)
	}
	for index, want := range wantTools {
		if capturedTools[index].Function.Name != want {
			t.Fatalf("provider tool[%d] = %q, want %q", index, capturedTools[index].Function.Name, want)
		}
	}
	automatic, initialized := service.autoSelectedTools(ctx, session.ID)
	if initialized || len(automatic) != 0 {
		t.Fatalf("automatic tool selection = %#v initialized=%t, want uninitialized empty set", automatic, initialized)
	}
	activeIDs, _ := service.activeSkills(ctx, session.ID)
	if len(activeIDs) != 0 {
		t.Fatalf("automatic inventory must not persist Skill activation: %v", activeIDs)
	}
	visibleIDs, visibleSkills := service.visibleSkills(ctx, session.ID)
	if len(visibleIDs) != 0 || len(visibleSkills) != 0 {
		t.Fatalf("visible Skill catalog = ids:%#v skills:%#v, want no automatic resource selection", visibleIDs, visibleSkills)
	}
}

func TestResourceResolveAppliesSkillAndExtensionContextAsSessionResources(t *testing.T) {
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
		"runtime":     map[string]any{"type": "static"},
		"contributes": map[string]any{"contexts": []any{map[string]any{"id": "ui-checklist", "kind": "instructions", "path": "context.md"}}},
	})
	status, err := service.extensionSupervisor.Discover(extensionRoot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.extensionSupervisor.Enable(ctx, status.ID); err != nil {
		t.Fatal(err)
	}

	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "UI review", ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	registry, _ := service.toolsForWorkspace(session.ProjectPath)
	rawTool, ok := registry.Get(ResourceResolveName)
	if !ok {
		t.Fatal("resource_resolve was not registered")
	}
	result := rawTool.Execute(ctx, json.RawMessage(`{"mode":"use","intent":"Review UI accessibility with UI checklist"}`), domain.ToolExecutionContext{
		SessionID: session.ID, WorkspaceRoot: session.ProjectPath, AgentMode: domain.AgentModeAssistant, ToolCallID: "resolve",
	})

	if !result.OK {
		t.Fatalf("resource_resolve result = %#v", result)
	}
	if result.Structured["status"] != "applied" || result.Structured["resourceCount"] != 2 {
		t.Fatalf("resource_resolve structured result = %#v", result.Structured)
	}
	if !strings.Contains(result.ModelContent, "<available_skills>") || !strings.Contains(result.ModelContent, "<name>ui-review</name>") {
		t.Fatalf("resource_resolve did not expose filtered Skill catalog: %s", result.ModelContent)
	}
	if strings.Contains(result.ModelContent, `<skill_content name="ui-review"`) || strings.Contains(result.ModelContent, "Check focus order and accessible names.") {
		t.Fatalf("resource_resolve injected full Skill content instead of filtered metadata: %s", result.ModelContent)
	}
	if !strings.Contains(result.ModelContent, `<extension_context extension="com.example.ui" id="ui-checklist"`) || !strings.Contains(result.ModelContent, "Use the extension UI review checklist.") {
		t.Fatalf("resource_resolve did not inject extension context: %s", result.ModelContent)
	}
	activeIDs, _ := service.activeSkills(ctx, session.ID)
	if len(activeIDs) != 0 {
		t.Fatalf("active Skills = %#v, want none from filtered catalog selection", activeIDs)
	}
	visibleIDs, visibleSkills := service.visibleSkills(ctx, session.ID)
	if len(visibleIDs) != 1 || len(visibleSkills) != 1 || visibleSkills[0].Name != "ui-review" {
		t.Fatalf("visible Skills = ids:%#v skills:%#v, want ui-review", visibleIDs, visibleSkills)
	}
	if contextText := service.activeExtensionContextsContext(ctx, session.ID); !strings.Contains(contextText, "Use the extension UI review checklist.") {
		t.Fatalf("active extension context = %q", contextText)
	}
	history, err := service.modelVisibleSessionHistory(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	joined := ""
	for _, message := range history {
		joined += "\n" + message.Text
	}
	if !strings.Contains(joined, "<available_skills>") || !strings.Contains(joined, "<name>ui-review</name>") || strings.Contains(joined, "Check focus order and accessible names.") || !strings.Contains(joined, "Use the extension UI review checklist.") {
		t.Fatalf("session resource context was not persisted into later model history: %s", joined)
	}
}

func TestResourceResolveExpandsSkillSelectionGroup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()

	writeSkill(t, filepath.Join(home, ".codex", "skills", "hyperframes"), "hyperframes", "Mandatory entry point for HyperFrames video work", "Read routing instructions first.")
	writeSkill(t, filepath.Join(home, ".codex", "skills", "hyperframes-core"), "hyperframes-core", "HyperFrames composition contract", "Follow the composition contract.")
	writeSkill(t, filepath.Join(home, ".codex", "skills", "hyperframes-cli"), "hyperframes-cli", "HyperFrames CLI loop", "Use the CLI loop.")
	scan, err := service.ScanGlobalSkills(ctx)
	if err != nil || len(scan.Candidates) != 3 {
		t.Fatalf("scan = %#v, err = %v", scan, err)
	}
	for _, candidate := range scan.Candidates {
		if _, err := service.ImportSkill(ctx, domain.SkillImportInput{CandidateID: candidate.ID, TargetScope: domain.SkillScopeGlobal}); err != nil {
			t.Fatal(err)
		}
	}

	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "HyperFrames", ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	registry, _ := service.toolsForWorkspace(session.ProjectPath)
	rawTool, ok := registry.Get(ResourceResolveName)
	if !ok {
		t.Fatal("resource_resolve was not registered")
	}
	result := rawTool.Execute(ctx, json.RawMessage(`{"mode":"use","intent":"HyperFrames animation"}`), domain.ToolExecutionContext{
		SessionID: session.ID, WorkspaceRoot: session.ProjectPath, AgentMode: domain.AgentModeAssistant, ToolCallID: "resolve",
	})

	if !result.OK {
		t.Fatalf("resource_resolve result = %#v", result)
	}
	if result.Structured["status"] != "applied" || result.Structured["resourceCount"] != 1 {
		t.Fatalf("resource_resolve structured result = %#v", result.Structured)
	}
	for _, name := range []string{"hyperframes", "hyperframes-core", "hyperframes-cli"} {
		if !strings.Contains(result.ModelContent, "<name>"+name+"</name>") {
			t.Fatalf("resource_resolve did not expose grouped Skill %q: %s", name, result.ModelContent)
		}
	}
	if strings.Contains(result.ModelContent, "Read routing instructions first.") || strings.Contains(result.ModelContent, `<skill_content name="hyperframes"`) {
		t.Fatalf("resource_resolve preloaded grouped Skill content: %s", result.ModelContent)
	}
	visibleIDs, visibleSkills := service.visibleSkills(ctx, session.ID)
	if len(visibleIDs) != 3 || len(visibleSkills) != 3 {
		t.Fatalf("visible grouped Skills = ids:%#v skills:%#v, want three HyperFrames skills", visibleIDs, visibleSkills)
	}
}

func TestExplicitCapabilityActivationSkipsInitialAuxiliarySelection(t *testing.T) {
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
	imported, err := service.ImportSkill(ctx, domain.SkillImportInput{CandidateID: scan.Candidates[0].ID, TargetScope: domain.SkillScopeGlobal})
	if err != nil {
		t.Fatal(err)
	}

	registry := NewRegistry()
	for _, tool := range []struct {
		name, description string
	}{
		{name: "manual_notes", description: "User-selected notes tool"},
		{name: "auto_notes", description: "Automatically selected notes tool"},
	} {
		if err := registry.RegisterScoped(phase6EchoTool{spec: domain.ToolSpec{
			Name: tool.name, Description: tool.description, InputSchema: map[string]any{"type": "object"},
			Category: "extension", Toolsets: []string{"coding"}, ActivationPolicy: "auto",
		}}, domain.ToolSourceExtension, "notes", "v1"); err != nil {
			t.Fatal(err)
		}
	}

	skillSession, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetSessionActiveSkills(ctx, domain.SessionActiveSkillsInput{SessionID: skillSession.ID, SkillIDs: []string{imported.ID}}); err != nil {
		t.Fatal(err)
	}
	skillResolution, err := service.resolveHostResources(ctx, hostResourceResolveRequest{
		SessionID: skillSession.ID, TurnID: "turn-skill", Intent: "use notes", Registry: registry, Specs: registry.Specs(), AgentMode: domain.AgentModeCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	if skillResolution.ToolActivations["auto_notes"] != "" || skillResolution.ToolActivations[SkillsListToolName] != "skillCatalog" || skillResolution.ToolActivations[SkillsReadToolName] != "skillCatalog" {
		t.Fatalf("skill-session activations = %#v, want explicit Skill catalog only and no automatic tool", skillResolution.ToolActivations)
	}
	automatic, initialized := service.autoSelectedTools(ctx, skillSession.ID)
	if !initialized || len(automatic) != 0 {
		t.Fatalf("skill-session automatic set = %#v initialized=%t, want initialized empty", automatic, initialized)
	}
	visibleIDs, visibleSkills := service.visibleSkills(ctx, skillSession.ID)
	if len(visibleIDs) != 1 || len(visibleSkills) != 1 || visibleSkills[0].Name != "ui-review" {
		t.Fatalf("visible Skill catalog = ids:%#v skills:%#v, want explicit ui-review", visibleIDs, visibleSkills)
	}

	toolSession, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetSessionActiveTools(ctx, domain.SessionActiveToolsInput{SessionID: toolSession.ID, ToolNames: []string{"manual_notes"}}); err != nil {
		t.Fatal(err)
	}
	toolResolution, err := service.resolveHostResources(ctx, hostResourceResolveRequest{
		SessionID: toolSession.ID, TurnID: "turn-tool", Intent: "use notes", Registry: registry, Specs: registry.Specs(), AgentMode: domain.AgentModeCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	if toolResolution.ToolActivations["manual_notes"] != "manual" || toolResolution.ToolActivations["auto_notes"] != "" {
		t.Fatalf("tool-session activations = %#v, want explicit manual tool only and no automatic tool", toolResolution.ToolActivations)
	}
	automatic, initialized = service.autoSelectedTools(ctx, toolSession.ID)
	if !initialized || len(automatic) != 0 {
		t.Fatalf("tool-session automatic set = %#v initialized=%t, want initialized empty", automatic, initialized)
	}
}

func TestResourceResolveUseResolvesToolGroupsAndInstructionResources(t *testing.T) {
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
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"resources\":[{\"kind\":\"extension\",\"id\":\"extension_com_example_ui_ui_review\"},{\"kind\":\"mcp\",\"id\":\"mcp_group_docs\"}]}"}}]}`))
		case 2:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"resources\":[\"skill:ui-review\",\"context:com.example.ui:ui-checklist\"]}"}}]}`))
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
	registry, _ := service.toolsForWorkspace(session.ProjectPath)
	rawTool, ok := registry.Get(ResourceResolveName)
	if !ok {
		t.Fatal("resource_resolve was not registered")
	}
	result := rawTool.Execute(ctx, json.RawMessage(`{"mode":"use","intent":"Review this UI accessibility"}`), domain.ToolExecutionContext{
		SessionID: session.ID, WorkspaceRoot: session.ProjectPath, AgentMode: domain.AgentModeAssistant, ToolCallID: "resolve",
	})
	if !result.OK {
		t.Fatalf("resource_resolve result = %#v", result)
	}
	if result.Structured["status"] != "replaced" || result.Structured["count"] != 7 || result.Structured["resourceCount"] != 4 {
		t.Fatalf("resource_resolve structured result = %#v", result.Structured)
	}

	mu.Lock()
	captured := append([]capturedRequest(nil), requests...)
	mu.Unlock()
	if len(captured) != 2 {
		t.Fatalf("provider request count = %d, want resource_resolve-owned tool and instruction-resource selections", len(captured))
	}
	toolGroupText := ""
	for _, message := range captured[0].Messages {
		toolGroupText += message.Content
	}
	if !strings.Contains(toolGroupText, "Host resource-group selector") || !strings.Contains(toolGroupText, "extension:extension_com_example_ui_ui_review：UI Inspector｜Inspect UI accessibility") || !strings.Contains(toolGroupText, "mcp:mcp_group_docs：Docs｜Search UI documentation") || strings.Contains(toolGroupText, "example_capture_ui") || strings.Contains(toolGroupText, "mcp_docs_read_docs") || len(captured[0].Tools) != 0 {
		t.Fatalf("first request was not the minimal grouped-or-individual resolver: %#v", captured[0])
	}
	instructionText := ""
	for _, message := range captured[1].Messages {
		instructionText += message.Content
	}
	if !strings.Contains(instructionText, `"resources"`) || !strings.Contains(instructionText, "skill:ui-review") || !strings.Contains(instructionText, "context:com.example.ui:ui-checklist") || len(captured[1].Tools) != 0 {
		t.Fatalf("second request was not the instruction-resource resolver: %#v", captured[1])
	}
	if !strings.Contains(result.ModelContent, "<available_skills>") || !strings.Contains(result.ModelContent, "<name>ui-review</name>") || !strings.Contains(result.ModelContent, `<extension_context extension="com.example.ui" id="ui-checklist"`) {
		t.Fatalf("resource_resolve missing Host-selected filtered catalog or extension context: %s", result.ModelContent)
	}
	if strings.Contains(result.ModelContent, "Check focus order and accessible names.") || strings.Contains(result.ModelContent, `<skill_content name="ui-review"`) {
		t.Fatalf("resource_resolve preloaded full Skill content instead of filtered catalog: %s", result.ModelContent)
	}
	activations, _ := service.snapshotToolCandidates(ctx, session.ID, "next-turn", registry, registry.Specs())
	for name, source := range service.visibleSkillToolActivations(ctx, session.ID) {
		if activations[name] == "" {
			activations[name] = source
		}
	}
	assembly := AssembleToolSpecsWithSources(registry, registry.Specs(), activations)
	toolNames := map[string]bool{}
	for _, tool := range assembly.Specs {
		toolNames[tool.Name] = true
	}
	if len(toolNames) != 17 || !toolNames["read"] || !toolNames[ExecCommandToolName] || !toolNames[WriteStdinToolName] || !toolNames["edit"] || !toolNames["write"] || !toolNames["update_plan"] || !toolNames["ask_user"] || !toolNames[ResourceResolveName] || !toolNames[SkillsListToolName] || !toolNames[SkillsReadToolName] || !toolNames["example_inspect_ui"] || !toolNames["example_capture_ui"] || !toolNames["mcp_docs_search_docs"] || !toolNames["mcp_docs_read_docs"] || !toolNames["mcp_host_docs_list_resource_templates"] || !toolNames["mcp_host_docs_list_resources"] || !toolNames["mcp_host_docs_read_resource"] {
		t.Fatalf("next-step tools = %#v, want core, selection control, filtered Skill controls, and every member of the selected groups", assembly.Specs)
	}
	automatic, initialized := service.autoSelectedTools(ctx, session.ID)
	if !initialized || !automatic["example_inspect_ui"] || !automatic["example_capture_ui"] || !automatic["mcp_docs_search_docs"] || !automatic["mcp_docs_read_docs"] || !automatic["mcp_host_docs_list_resource_templates"] || !automatic["mcp_host_docs_list_resources"] || !automatic["mcp_host_docs_read_resource"] || len(automatic) != 7 {
		t.Fatalf("persisted automatic selection = %#v initialized=%t", automatic, initialized)
	}
	activeIDs, _ := service.activeSkills(ctx, session.ID)
	if len(activeIDs) != 0 {
		t.Fatalf("automatic Skill selection must not persist active Skill content: %v", activeIDs)
	}
	visibleIDs, visibleSkills := service.visibleSkills(ctx, session.ID)
	if len(visibleIDs) != 1 || len(visibleSkills) != 1 || visibleSkills[0].Name != "ui-review" {
		t.Fatalf("automatic Skill selection did not persist filtered visible catalog: ids:%#v skills:%#v", visibleIDs, visibleSkills)
	}
}

func TestResourceResolveInspectSummarizesEligibleToolsWithoutActivation(t *testing.T) {
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
		_ = index
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
	registry, _ := service.toolsForWorkspace(session.ProjectPath)
	rawTool, ok := registry.Get(ResourceResolveName)
	if !ok {
		t.Fatal("resource_resolve was not registered")
	}
	result := rawTool.Execute(ctx, json.RawMessage(`{"mode":"inspect","intent":"当前有哪些工具可调用"}`), domain.ToolExecutionContext{
		SessionID: session.ID, WorkspaceRoot: session.ProjectPath, AgentMode: domain.AgentModeAssistant, ToolCallID: "resolve",
	})
	if !result.OK {
		t.Fatalf("resource_resolve inspect result = %#v", result)
	}
	if result.Structured["status"] != "inspected" || result.Structured["appliesNextStep"] != false {
		t.Fatalf("resource_resolve inspect structured result = %#v", result.Structured)
	}
	resources := hostResourceSelectionResourcesFromAny(result.Structured["resources"])
	var inventory *hostResourceSelectionResource
	for index := range resources {
		if resources[index].Kind == "extension" && resources[index].ID == "extension_com_example_inventory_inventory" {
			inventory = &resources[index]
			break
		}
	}
	if inventory == nil || inventory.ToolCount != inventoryToolCount {
		t.Fatalf("inspect resources = %#v, want one unsplit inventory extension source beyond the legacy member limit", resources)
	}
	automatic, initialized := service.autoSelectedTools(ctx, session.ID)
	if initialized || len(automatic) != 0 {
		t.Fatalf("inspection automatic set = %#v initialized=%t, want unchanged empty", automatic, initialized)
	}
	if _, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "谢谢"}); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	captured := append([]capturedRequest(nil), requests...)
	mu.Unlock()
	if len(captured) != 1 {
		t.Fatalf("provider request count = %d, want one primary request after inspection", len(captured))
	}
	firstPrimary := map[string]bool{}
	for _, tool := range captured[0].Tools {
		firstPrimary[tool.Function.Name] = true
	}
	for index := 0; index < inventoryToolCount; index++ {
		name := fmt.Sprintf("inventory_tool_%02d", index)
		if firstPrimary[name] {
			t.Fatalf("inspection leaked %q into a later Provider request", name)
		}
	}
	if !firstPrimary[ResourceResolveName] {
		t.Fatalf("later Provider tools are missing %q", ResourceResolveName)
	}
}

func TestResourceResolveUsesAuxiliarySkillSelectionAndPersistsSessionResource(t *testing.T) {
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
		if index == 2 {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"tools\":[],\"resources\":[\"skill:ui-summary\"],\"reason\":\"summary is sufficient\"}"}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"unexpected"}}]}`))
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
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "Skill catalog", ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	registry, _ := service.toolsForWorkspace(session.ProjectPath)
	rawTool, ok := registry.Get(ResourceResolveName)
	if !ok {
		t.Fatal("resource_resolve was not registered")
	}
	result := rawTool.Execute(ctx, json.RawMessage(`{"mode":"use","intent":"Explain what the ui-summary Skill offers"}`), domain.ToolExecutionContext{
		SessionID: session.ID, WorkspaceRoot: session.ProjectPath, AgentMode: domain.AgentModeAssistant, ToolCallID: "resolve",
	})
	if !result.OK {
		t.Fatalf("resource_resolve result = %#v", result)
	}

	mu.Lock()
	captured := append([]capturedRequest(nil), requests...)
	mu.Unlock()
	if len(captured) != 2 {
		t.Fatalf("provider request count = %d, want explicit resource_resolve-owned tool and instruction-resource selections", len(captured))
	}
	auxiliaryText := ""
	for _, message := range captured[1].Messages {
		auxiliaryText += message.Content
	}
	if !strings.Contains(auxiliaryText, "Explain the available UI component workflow") || strings.Contains(auxiliaryText, "PRIVATE_UI_WORKFLOW") {
		t.Fatalf("auxiliary catalog exposed more than bounded Skill descriptions: %s", auxiliaryText)
	}
	if !strings.Contains(result.ModelContent, "<available_skills>") || !strings.Contains(result.ModelContent, "<name>ui-summary</name>") {
		t.Fatalf("resource_resolve did not expose selected Skill catalog: %s", result.ModelContent)
	}
	if strings.Contains(result.ModelContent, "PRIVATE_UI_WORKFLOW") || strings.Contains(result.ModelContent, `<skill_content name="ui-summary"`) {
		t.Fatalf("resource_resolve leaked selected Skill body before skills_read: %s", result.ModelContent)
	}
	activeIDs, _ := service.activeSkills(ctx, session.ID)
	if len(activeIDs) != 0 {
		t.Fatalf("active Skills = %#v, want none from filtered catalog selection", activeIDs)
	}
	visibleIDs, visibleSkills := service.visibleSkills(ctx, session.ID)
	if len(visibleIDs) != 1 || len(visibleSkills) != 1 || visibleSkills[0].Name != "ui-summary" {
		t.Fatalf("visible Skills = ids:%#v skills:%#v, want selected Skill persisted as filtered catalog", visibleIDs, visibleSkills)
	}
}

func hostResourceSelectionResourcesFromAny(value any) []hostResourceSelectionResource {
	raw, _ := json.Marshal(value)
	var resources []hostResourceSelectionResource
	_ = json.Unmarshal(raw, &resources)
	return resources
}

func TestHostSkillCandidatesIncludeAvailableSystemSkills(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()
	if err := service.store.SaveProviderAuth(ctx, domain.ProviderAuthRecord{ProviderID: "openai", Method: "oauth-browser", AccessToken: "token"}); err != nil {
		t.Fatal(err)
	}

	candidates, err := service.hostSkillCandidates(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]SkillResolveCandidate{}
	for _, candidate := range candidates {
		byName[candidate.Name] = candidate
	}
	for _, name := range []string{"skill-creator", "skill-installer", "openai-docs", "plugin-creator", "review-agent"} {
		if byName[name].Name == "" {
			t.Fatalf("missing available system Skill candidate %q in %#v", name, candidates)
		}
	}
	if byName["imagegen"].Name == "" {
		t.Fatalf("missing available Codex tool-backed Skill candidate imagegen in %#v", candidates)
	}
}

func TestHostResourceDecisionIgnoresLegacySkillInstructionsField(t *testing.T) {
	decision, err := parseHostResourceDecision(`{"tools":[],"resources":["skill:ui-review","context:design:tokens"],"skillInstructions":["skill:ui-review","skill:ui-review","context:design:tokens","skill:invented"],"reason":"execute review"}`)
	if err != nil {
		t.Fatal(err)
	}
	candidates := []hostInstructionCandidate{
		{Key: "skill:ui-review", Kind: "skill", Name: "ui-review", Skill: &SkillResolveCandidate{Name: "ui-review"}},
		{Key: "context:design:tokens", Kind: "context", Name: "tokens", Context: &extensionContextCandidate{}},
	}
	selected := validateHostInstructionSelection(candidates, decision.ResourceKeys)
	selectedKeys := map[string]bool{}
	for _, candidate := range selected {
		selectedKeys[candidate.Key] = true
	}
	if len(selected) != 2 || !selectedKeys["skill:ui-review"] || !selectedKeys["context:design:tokens"] {
		t.Fatalf("selected resources = %#v", selected)
	}
}

func TestHostPreSnapshotLocalResolutionIsBoundedAndExact(t *testing.T) {
	candidates := []hostInstructionCandidate{
		{Key: "skill:ui-review", Kind: "skill", Name: "ui-review", Description: "Review UI accessibility"},
		{Key: "context:design:tokens", Kind: "context", Name: "design tokens", Description: "Semantic UI colors and spacing"},
		{Key: "skill:database", Kind: "skill", Name: "database", Description: "Tune SQL queries"},
	}
	decision := localHostInstructionResolve("Review UI accessibility with design tokens", candidates)
	selected := validateHostInstructionSelection(candidates, append(decision.Keys, "skill:invented"))
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

func TestFailedEnabledMCPRefreshIsExcludedFromPreSnapshotCandidates(t *testing.T) {
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
	for _, name := range []string{"read", ExecCommandToolName, WriteStdinToolName, "edit", "write", "update_plan", "ask_user"} {
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

func TestHostPreSnapshotContextMessageIsEphemeralAndOrderedAfterAgentPrompt(t *testing.T) {
	messages := []domain.ChatMessage{{Role: domain.EventRoleSystem, Text: "agent"}, {Role: domain.EventRoleUser, Text: "request"}}
	resolved := appendHostPreSnapshotContext(messages, "selected")
	if len(resolved) != 3 || resolved[0].Text != "agent" || !strings.Contains(resolved[1].Text, "<host_selected_resources>") || resolved[2].Text != "request" {
		t.Fatalf("resolved messages = %#v", resolved)
	}
	if len(messages) != 2 {
		t.Fatalf("source messages mutated: %#v", messages)
	}
}
