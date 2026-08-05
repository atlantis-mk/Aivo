package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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
	if strings.Contains(joined, "call the skill tool") || strings.Contains(joined, "call tool_resolve") {
		t.Fatalf("primary request retained removed discovery protocol: %s", joined)
	}
	wantCoreTools := []string{"read", "bash", "edit", "write"}
	if len(capturedTools) != len(wantCoreTools)+3 {
		t.Fatalf("provider tools = %#v, want four core tools plus three builtin project extension tools", capturedTools)
	}
	for index, want := range wantCoreTools {
		if capturedTools[index].Function.Name != want {
			t.Fatalf("provider tool[%d] = %q, want %q", index, capturedTools[index].Function.Name, want)
		}
	}
	projectTools := 0
	for _, tool := range capturedTools[len(wantCoreTools):] {
		if strings.HasPrefix(tool.Function.Name, "aivo_projects_") {
			projectTools++
		}
	}
	if projectTools != 3 {
		t.Fatalf("provider tools = %#v, want three unchanged canonical builtin project extension tools", capturedTools)
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
		"schemaVersion": 1, "id": "com.example.ui", "name": "UI Inspector", "description": "Inspect UI accessibility", "version": "1", "apiVersion": "1",
		"runtime": map[string]any{"type": "builtin"},
		"contributes": map[string]any{
			"tools":    []any{map[string]any{"name": "example_inspect_ui", "description": "Inspect UI accessibility", "schema": map[string]any{"type": "object"}, "activation": "auto"}},
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
	if err := service.mcpManager.store.ReplaceMCPTools(ctx, mcpServer.ID, []domain.MCPToolRecord{{
		ID: "docs:search_docs", ServerID: mcpServer.ID, Name: "search_docs", Description: "Search UI documentation", InputSchema: map[string]any{"type": "object"},
	}}); err != nil {
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
		if index == 1 {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"tools\":[\"example_inspect_ui\",\"mcp_docs_search_docs\"],\"resources\":[\"skill:ui-review\",\"context:com.example.ui:ui-checklist\"],\"skillInstructions\":[\"skill:ui-review\"],\"reason\":\"direct matches\"}"}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"resolved"}}]}`))
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
	if _, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "Review this UI accessibility"}); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	captured := append([]capturedRequest(nil), requests...)
	mu.Unlock()
	if len(captured) != 2 {
		t.Fatalf("provider request count = %d, want one auxiliary resolution plus one primary request", len(captured))
	}
	auxiliaryText := ""
	for _, message := range captured[0].Messages {
		auxiliaryText += message.Content
	}
	if !strings.Contains(auxiliaryText, "Host pre-call resource resolver") || len(captured[0].Tools) != 0 {
		t.Fatalf("first request was not the tool/resource resolver: %#v", captured[0])
	}
	primaryText := ""
	for _, message := range captured[1].Messages {
		primaryText += message.Content
	}
	if !strings.Contains(primaryText, `<skill_summary name="ui-review"`) || !strings.Contains(primaryText, `<skill_content name="ui-review"`) || !strings.Contains(primaryText, `<extension_context extension="com.example.ui" id="ui-checklist"`) {
		t.Fatalf("primary request missing selected Skill/context: %s", primaryText)
	}
	toolNames := map[string]bool{}
	for _, tool := range captured[1].Tools {
		toolNames[tool.Function.Name] = true
	}
	projectTools := 0
	for name := range toolNames {
		if strings.HasPrefix(name, "aivo_projects_") {
			projectTools++
		}
	}
	if len(toolNames) != 9 || projectTools != 3 || !toolNames["read"] || !toolNames["bash"] || !toolNames["edit"] || !toolNames["write"] {
		t.Fatalf("primary tools = %#v, want four core, three builtin project, selected Manifest, and MCP tools", captured[1].Tools)
	}
	activeIDs, _ := service.activeSkills(ctx, session.ID)
	if len(activeIDs) != 0 {
		t.Fatalf("automatic Skill selection persisted across requests: %v", activeIDs)
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
	if len(captured) != 2 {
		t.Fatalf("provider request count = %d, want auxiliary plus primary", len(captured))
	}
	auxiliaryText := ""
	for _, message := range captured[0].Messages {
		auxiliaryText += message.Content
	}
	if !strings.Contains(auxiliaryText, "Explain the available UI component workflow") || strings.Contains(auxiliaryText, "PRIVATE_UI_WORKFLOW") {
		t.Fatalf("auxiliary catalog was not summary-only: %s", auxiliaryText)
	}
	primaryText := ""
	for _, message := range captured[1].Messages {
		primaryText += message.Content
	}
	if !strings.Contains(primaryText, `<skill_summary name="ui-summary"`) || !strings.Contains(primaryText, "Explain the available UI component workflow") {
		t.Fatalf("primary request missing canonical Skill summary: %s", primaryText)
	}
	if strings.Contains(primaryText, "PRIVATE_UI_WORKFLOW") || strings.Contains(primaryText, `<skill_content name="ui-summary"`) {
		t.Fatalf("summary-only selection exposed Skill instructions: %s", primaryText)
	}
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
		"schemaVersion": 1, "id": "com.example.design", "name": "Design Context", "description": "UI design token guidance", "version": "1", "apiVersion": "1",
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

func TestFailedEnabledPluginIsExcludedFromPreCallCandidates(t *testing.T) {
	store := &memoryProviderStore{plugins: []domain.PluginInstall{{
		ID: "broken-plugin", Enabled: true, Status: domain.PluginStatusEnabled,
		Manifest: domain.PluginManifest{
			ID: "broken-plugin", Version: "1", Entrypoint: domain.PluginEntrypoint{Command: "definitely-not-a-real-plugin-command"},
			Tools: []domain.PluginDeclaredTool{{Name: "broken_tool", Description: "Broken tool", InputSchema: map[string]any{"type": "object"}, Toolsets: []string{"coding"}}},
		},
	}}}
	service := NewService(store)
	failed := service.prepareEnabledToolCatalogs(context.Background())
	registry, _ := service.toolsForWorkspace(t.TempDir())
	specs := filterEligibleToolSpecs(registry, registry.Specs(), failed)
	for _, spec := range specs {
		if spec.Name == "broken_tool" {
			t.Fatalf("failed plugin remained eligible: failed = %#v, specs = %#v", failed, specs)
		}
	}
	if _, ok := registry.Get("read"); !ok {
		t.Fatalf("failed plugin removed the core registry: %#v", registry.Specs())
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
		"schemaVersion": 1, "id": "com.example.stopped", "name": "Stopped", "version": "1", "apiVersion": "1",
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
	store := &memoryProviderStore{plugins: []domain.PluginInstall{{
		ID: "cancelled-plugin", Enabled: true, Status: domain.PluginStatusEnabled,
		Manifest: domain.PluginManifest{
			ID: "cancelled-plugin", Version: "1", Entrypoint: domain.PluginEntrypoint{Command: "definitely-not-a-real-plugin-command"},
			Tools: []domain.PluginDeclaredTool{{Name: "cancelled_tool", InputSchema: map[string]any{"type": "object"}, Toolsets: []string{"coding"}}},
		},
	}}}
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
	for _, name := range []string{"read", "bash", "edit", "write"} {
		if !names[name] {
			t.Fatalf("cancelled preparation removed core tool %q: %#v", name, specs)
		}
	}
	if names["cancelled_tool"] {
		t.Fatalf("cancelled extension preparation left an unready candidate: %#v", specs)
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
