package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestConfiguredAgentOverridesAreResolvedPerProject(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	t.Setenv("HOME", t.TempDir())
	ctx := context.Background()
	root := t.TempDir()
	writeRuntimeConfigTestFile(t, filepath.Join(root, ".aivo", "config.json"), `{
  "agents": {
    "researcher": {
      "description": "Project researcher",
      "prompt": "Inspect this project carefully.",
      "toolsets": ["safe"],
      "permissionScope": "read_only",
      "subagents": ["reviewer"],
      "maxSteps": 3,
      "model": {"providerId": "openai", "modelId": "gpt-5.5"}
    },
    "reviewer": {
      "description": "Project reviewer",
      "prompt": "Review one bounded question.",
      "toolsets": ["safe"],
      "permissionScope": "read_only",
      "mode": "subagent"
    }
  }
}`)
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: root})
	if err != nil {
		t.Fatal(err)
	}
	definition, err := service.resolveAgentModeForRequest(ctx, session.ID, "researcher")
	if err != nil {
		t.Fatal(err)
	}
	if definition.Prompt != "Inspect this project carefully." || definition.MaxSteps != 3 || definition.PermissionScope != "read_only" || len(definition.Subagents) != 1 || definition.Subagents[0] != "reviewer" || definition.Model == nil || definition.Revision == "" {
		t.Fatalf("definition = %#v", definition)
	}
	updated, err := service.SetSessionAgentMode(ctx, domain.SetSessionAgentModeInput{SessionID: session.ID, Mode: "researcher"})
	if err != nil || updated.AgentMode != "researcher" {
		t.Fatalf("updated session = %#v err = %v", updated, err)
	}
}

func TestConfiguredAgentRejectsUnavailableToolsetBeforeExecution(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	t.Setenv("HOME", t.TempDir())
	ctx := context.Background()
	root := t.TempDir()
	writeRuntimeConfigTestFile(t, filepath.Join(root, "aivo.json"), `{
  "agents": {"broken": {"prompt": "Broken", "toolsets": ["does-not-exist"]}}
}`)
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: root})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.resolveAgentModeForRequest(ctx, session.ID, "broken"); err == nil {
		t.Fatal("configured agent with unavailable toolset resolved")
	}
}

type agentSpecTool struct {
	spec domain.ToolSpec
}

type agentDelegateBuiltinClient struct{ service *Service }

func (c *agentDelegateBuiltinClient) Initialize(context.Context, domain.ExtensionManifest) error {
	return nil
}
func (c *agentDelegateBuiltinClient) Execute(ctx context.Context, name string, args json.RawMessage, execCtx domain.ToolExecutionContext) (domain.ToolResult, error) {
	return c.service.delegateTaskToolNamed(ctx, args, execCtx, name), nil
}
func (c *agentDelegateBuiltinClient) UIEvent(context.Context, string, string, any) (any, error) {
	return nil, errors.New("agent delegate extension has no Web view")
}
func (c *agentDelegateBuiltinClient) Shutdown(context.Context) error { return nil }

func (t agentSpecTool) Spec() domain.ToolSpec {
	return t.spec
}

func (t agentSpecTool) Execute(context.Context, json.RawMessage, domain.ToolExecutionContext) domain.ToolResult {
	return domain.ToolResult{Name: t.spec.Name, OK: true}
}

func TestAgentCatalogDefaults(t *testing.T) {
	catalog := NewAgentCatalog()
	modes := catalog.List(false)
	if len(modes) != 1 || modes[0].ID != domain.AgentModeAssistant {
		t.Fatalf("visible modes = %#v, want Assistant only", modes)
	}
	assistant, err := catalog.Get(domain.AgentModeAssistant)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"assistant mode",
		"Use read, bash, edit, and write",
		"runtime permissions",
		"Host activates an extension",
	} {
		if !strings.Contains(assistant.Prompt, required) {
			t.Fatalf("assistant prompt missing rule %q: %q", required, assistant.Prompt)
		}
	}
	for _, retired := range []string{domain.AgentModeCode, domain.AgentModeBuild, domain.AgentModeExplore, domain.AgentModePlan, domain.AgentModePlanner, domain.AgentModeReview, domain.AgentModeDebug} {
		if _, err := catalog.Get(retired); err == nil {
			t.Fatalf("retired built-in %q remained configured", retired)
		}
	}
	if _, err := catalog.Get("unknown"); err == nil {
		t.Fatal("unknown mode should not be configured")
	}
	for _, mode := range modes {
		if mode.Hidden {
			t.Fatalf("hidden mode leaked from visible list: %#v", mode)
		}
	}
	hidden := catalog.List(true)
	if len(hidden) <= len(modes) {
		t.Fatalf("expected hidden modes in full list, visible=%d full=%d", len(modes), len(hidden))
	}
}

func TestRetiredBuiltInSessionModeResolvesThroughAssistant(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{
		Type: domain.SessionTypeCoding, AgentMode: domain.AgentModeCode, ProjectPath: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := service.resolveAgentModeForRequest(ctx, session.ID, "")
	if err != nil || resolved.ID != domain.AgentModeAssistant {
		t.Fatalf("legacy session resolved = %#v, err = %v", resolved, err)
	}
	resolved, err = service.resolveAgentModeForRequest(ctx, session.ID, domain.AgentModeCode)
	if err != nil || resolved.ID != domain.AgentModeAssistant {
		t.Fatalf("legacy submitted mode resolved = %#v, err = %v", resolved, err)
	}
	if _, err := service.resolveAgentModeForRequest(ctx, session.ID, domain.AgentModeBuild); err == nil {
		t.Fatal("unrelated retired mode was accepted as a new selection")
	}
	if _, err := service.resolveAgentModeForRequest(ctx, session.ID, "unknown"); err == nil {
		t.Fatal("unknown mode was accepted")
	}
}

func TestAssistantModeWildcardToolsetsExposeAllTools(t *testing.T) {
	catalog := NewAgentCatalog()
	assistant, err := catalog.Get(domain.AgentModeAssistant)
	if err != nil {
		t.Fatal(err)
	}
	registry := NewRegistry()
	for _, tool := range []domain.Tool{
		agentSpecTool{spec: domain.ToolSpec{Name: "read_file", Toolsets: []string{"safe", "coding"}}},
		agentSpecTool{spec: domain.ToolSpec{Name: "bash", Toolsets: []string{"shell", "coding"}}},
		agentSpecTool{spec: domain.ToolSpec{Name: "extension_echo", Toolsets: []string{"extension"}}},
		agentSpecTool{spec: domain.ToolSpec{Name: "mcp_fetch", Toolsets: []string{"mcp"}}},
	} {
		if err := registry.Register(tool); err != nil {
			t.Fatal(err)
		}
	}
	specs := visibleToolSpecsForMode(assistant.ID, registry.SpecsForToolsets(assistant.Toolsets))
	if len(specs) != 4 {
		t.Fatalf("assistant visible tools = %#v, want all registered tools", specs)
	}
}

func TestAssociatedSubagentsControlDelegateToolSchemaAndPrompt(t *testing.T) {
	specs := []domain.ToolSpec{
		{Name: "read", InputSchema: map[string]any{"type": "object"}},
		{Name: "agent_delegate_task", InputSchema: map[string]any{"type": "object"}},
	}
	withoutAssociations := configureAgentDelegateToolSpecs(domain.AgentModeDefinition{ID: "orchestrator"}, specs)
	if len(withoutAssociations) != 1 || withoutAssociations[0].Name != "read" {
		t.Fatalf("delegate tool should be omitted without associations: %#v", withoutAssociations)
	}
	mode := domain.AgentModeDefinition{ID: "orchestrator", DisplayName: "Orchestrator", Prompt: "Coordinate work.", Subagents: []string{domain.AgentModeExplore, domain.AgentModeReview}}
	configured := configureAgentDelegateToolSpecs(mode, specs)
	if len(configured) != 2 {
		t.Fatalf("configured specs = %#v", configured)
	}
	properties, _ := configured[1].InputSchema["properties"].(map[string]any)
	modeSchema, _ := properties["mode"].(map[string]any)
	enum, _ := modeSchema["enum"].([]string)
	if len(enum) != 2 || enum[0] != domain.AgentModeExplore || enum[1] != domain.AgentModeReview {
		t.Fatalf("delegate mode enum = %#v", modeSchema["enum"])
	}
	prompt := buildAgentSystemPrompt(mode)
	if !strings.Contains(prompt, "associated_subagents") || !strings.Contains(prompt, "`explore`") || !strings.Contains(prompt, "do not delegate routine work") {
		t.Fatalf("association prompt = %q", prompt)
	}
	registry := NewRegistry()
	for _, tool := range newAgentRuntimeTools(&Service{}) {
		if tool.Spec().Name == "agent_delegate_task" {
			if err := registry.Register(tool); err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	dynamicSpecs := configureAgentDelegateToolSpecs(mode, registry.SpecsForToolsets([]string{"safe"}))
	assembly := AssembleToolSpecsWithSources(registry, dynamicSpecs, map[string]string{"agent_delegate_task": "modeAssociation"})
	if len(assembly.Specs) != 1 || len(assembly.Snapshot.Tools) != 1 || assembly.Snapshot.Tools[0].ActivationSource != "modeAssociation" {
		t.Fatalf("associated delegate assembly = %#v", assembly)
	}
	identity, ok := registry.IdentityFor("agent_delegate_task")
	if !ok || assembly.Snapshot.Tools[0].SchemaHash == identity.SchemaHash || assembly.Snapshot.Tools[0].SchemaHash != toolSchemaHash(assembly.Specs[0]) {
		t.Fatalf("dynamic delegate snapshot = %#v identity = %#v", assembly.Snapshot.Tools[0], identity)
	}
}

func TestDelegateTaskRejectsUnassociatedModeBeforeCreatingChild(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	assistant, err := service.GetAgentMode(ctx, domain.AgentModeAssistant)
	if err != nil {
		t.Fatal(err)
	}
	child, err := service.SaveAgentMode(ctx, domain.AgentModeDefinition{ID: "review_child", DisplayName: "Review child", Prompt: "Review one bounded task.", Mode: "subagent"})
	if err != nil {
		t.Fatal(err)
	}
	assistant.Subagents = []string{child.ID}
	if _, err := service.SaveAgentMode(ctx, assistant); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, AgentMode: domain.AgentModeAssistant, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	result := service.delegateTaskToolNamed(ctx, json.RawMessage(`{"mode":"forged_child","prompt":"inspect"}`), domain.ToolExecutionContext{
		SessionID: session.ID, AgentMode: domain.AgentModeAssistant,
	}, "agent_delegate_task")
	if result.OK || !strings.Contains(result.Error, "not associated") {
		t.Fatalf("forged delegation result = %#v", result)
	}
	runs, err := service.ListAgentRuns(ctx, domain.AgentRunListRequest{SessionID: session.ID, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 0 {
		t.Fatalf("forged delegation created runs: %#v", runs)
	}
}

func TestAgentRuntimeToolsUsePreferredNamesOnly(t *testing.T) {
	service := &Service{}
	names := map[string]bool{}
	for _, tool := range newAgentRuntimeTools(service) {
		names[tool.Spec().Name] = true
	}
	for _, name := range []string{
		"agent_mode_list",
		"agent_mode_set",
		"agent_delegate_task",
		"agent_run_list",
		"agent_run_cancel",
		"ask_user",
		"automation_create",
		"automation_list",
		"automation_update",
		"automation_cancel",
	} {
		if !names[name] {
			t.Fatalf("preferred tool name %s was not registered; names = %#v", name, names)
		}
	}
	for _, oldName := range []string{
		"agent_list_modes",
		"agent_set_mode",
		"delegate_task",
		"agent_status",
		"agent_cancel",
		"question",
		"scheduler_create",
		"scheduler_list",
		"scheduler_update",
		"scheduler_cancel",
	} {
		if names[oldName] {
			t.Fatalf("old tool name %s should not be registered; names = %#v", oldName, names)
		}
	}
}

func TestWorkspaceRegistryRegistersOnlyDefaultHostControls(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	defer service.Shutdown()
	registry, _ := service.toolsForWorkspace(t.TempDir())
	if registry == nil {
		t.Fatal("workspace registry is nil")
	}
	for _, name := range []string{"update_plan", "ask_user"} {
		if _, ok := registry.Get(name); !ok {
			t.Fatalf("default Host control %q was not registered in the workspace Registry", name)
		}
	}
	for _, name := range []string{
		"agent_mode_list",
		"agent_mode_set",
		"agent_delegate_task",
		"agent_run_list",
		"agent_run_cancel",
		"automation_create",
		"automation_list",
		"automation_update",
		"automation_cancel",
		"create_goal",
		"get_goal",
		"update_goal",
	} {
		if _, ok := registry.Get(name); ok {
			t.Fatalf("non-default Host tool %q was directly registered in the workspace Registry", name)
		}
	}
}

func TestRegistrySpecsForToolsets(t *testing.T) {
	registry := NewRegistry()
	for _, tool := range []domain.Tool{
		agentSpecTool{spec: domain.ToolSpec{Name: "read_file", Toolsets: []string{"safe", "coding"}}},
		agentSpecTool{spec: domain.ToolSpec{Name: "write_file", Toolsets: []string{"coding"}}},
		agentSpecTool{spec: domain.ToolSpec{Name: "extension_echo", Toolsets: []string{"extension", "safe"}}},
	} {
		if err := registry.Register(tool); err != nil {
			t.Fatal(err)
		}
	}
	specs := registry.SpecsForToolsets([]string{"extension"})
	if len(specs) != 1 || specs[0].Name != "extension_echo" {
		t.Fatalf("extension specs = %#v", specs)
	}
	specs = registry.SpecsForToolsets([]string{"safe"})
	if len(specs) != 2 {
		t.Fatalf("safe specs = %#v", specs)
	}
}

func TestPlannerVisibleToolSpecsExcludeMutation(t *testing.T) {
	specs := []domain.ToolSpec{
		{Name: "read_file", Capability: "filesystem.read"},
		{Name: "write_file", Capability: "filesystem.write"},
		{Name: "run_tests", Capability: "shell.test"},
		{Name: "bash", Capability: "shell.exec"},
		{Name: "automation_create_job", Capability: "scheduler.write", Category: "automation"},
	}
	visible := visibleToolSpecsForMode(domain.AgentModePlan, specs)
	names := map[string]bool{}
	for _, spec := range visible {
		names[spec.Name] = true
	}
	if !names["read_file"] || len(names) != 1 {
		t.Fatalf("plan visible tools = %#v, want only read_file", visible)
	}
	if names["write_file"] || names["run_tests"] || names["bash"] {
		t.Fatalf("plan exposed mutation tools: %#v", visible)
	}

	visible = visibleToolSpecsForMode(domain.AgentModeDebug, specs)
	names = map[string]bool{}
	for _, spec := range visible {
		names[spec.Name] = true
	}
	if !names["read_file"] || !names["run_tests"] || !names["bash"] {
		t.Fatalf("debug missing diagnostic tools: %#v", visible)
	}
	if names["write_file"] || names["automation_create_job"] {
		t.Fatalf("debug exposed mutation tools: %#v", visible)
	}
}

func TestModePermissionHardDenials(t *testing.T) {
	engine := NewPermissionEngine(nil)
	writeTool := agentSpecTool{spec: domain.ToolSpec{Name: "write_file", Capability: "filesystem.write", Category: "filesystem"}}
	planWrite := engine.Evaluate(context.Background(), writeTool, json.RawMessage(`{}`), domain.ToolExecutionContext{AgentMode: domain.AgentModePlan})
	if planWrite.Decision != domain.PermissionDecisionDeny {
		t.Fatalf("plan write decision = %#v", planWrite)
	}
	testTool := agentSpecTool{spec: domain.ToolSpec{Name: "run_tests", Capability: "shell.test", Category: "shell"}}
	reviewTest := engine.Evaluate(context.Background(), testTool, json.RawMessage(`{}`), domain.ToolExecutionContext{AgentMode: domain.AgentModeReview})
	if reviewTest.Decision != domain.PermissionDecisionDeny {
		t.Fatalf("review test decision = %#v", reviewTest)
	}
	debugTest := engine.Evaluate(context.Background(), testTool, json.RawMessage(`{}`), domain.ToolExecutionContext{AgentMode: domain.AgentModeDebug, PermissionScope: "read_only"})
	if debugTest.Decision != domain.PermissionDecisionDeny {
		t.Fatalf("read-only debug test decision = %#v", debugTest)
	}
	debugWrite := engine.Evaluate(context.Background(), writeTool, json.RawMessage(`{}`), domain.ToolExecutionContext{AgentMode: domain.AgentModeDebug})
	if debugWrite.Decision != domain.PermissionDecisionDeny {
		t.Fatalf("debug write decision = %#v", debugWrite)
	}
}

func TestUpdatePlanDoesNotRequirePermission(t *testing.T) {
	engine := NewPermissionEngine(nil)
	tool := agentSpecTool{spec: updatePlanToolSpec()}
	result := engine.Evaluate(context.Background(), tool, json.RawMessage(`{"plan":[{"step":"Inspect","status":"in_progress"}]}`), domain.ToolExecutionContext{
		AgentMode:       domain.AgentModePlanner,
		PermissionScope: "read_only",
	})
	if result.Decision != domain.PermissionDecisionAllow {
		t.Fatalf("update_plan decision = %#v, want allow", result)
	}
}

func TestToolRuntimeRejectsToolsOutsideAllowedToolsets(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(agentSpecTool{spec: domain.ToolSpec{Name: "extension_echo", Toolsets: []string{"extension"}, InputSchema: map[string]any{"type": "object"}}}); err != nil {
		t.Fatal(err)
	}
	runtime := NewToolRuntime(registry, "")
	result := runtime.ExecuteWithContext(context.Background(), domain.ChatToolCall{Name: "extension_echo", Arguments: json.RawMessage(`{}`)}, domain.ToolExecutionContext{AllowedToolsets: []string{"safe"}})
	if result.OK || result.ToolError == nil || result.ToolError.Code != "toolset_denied" {
		t.Fatalf("result = %#v", result)
	}
}

func TestDelegateTaskCompletedRunIsNotMarkedCancelledByCleanup(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	extensionRoot := t.TempDir()
	writeTestExtensionManifest(t, extensionRoot, map[string]any{
		"schemaVersion": 2, "id": "aivo.agent", "name": "Agent", "version": "1.0.0", "apiVersion": "2",
		"runtime": map[string]any{"type": "builtin"},
		"contributes": map[string]any{"tools": []any{map[string]any{
			"name": "agent_delegate", "description": "Delegate a bounded task", "activation": "manual",
			"schema": map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"mode": map[string]any{"type": "string"}, "prompt": map[string]any{"type": "string"}, "title": map[string]any{"type": "string"}}, "required": []string{"mode", "prompt"}},
		}}},
	})
	service.extensionSupervisor.RegisterBuiltin("aivo.agent", func() extensionRuntimeClient { return &agentDelegateBuiltinClient{service: service} })
	status, err := service.extensionSupervisor.Discover(extensionRoot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.extensionSupervisor.Enable(ctx, status.ID); err != nil {
		t.Fatal(err)
	}
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Tools []any `json:"tools"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		if len(body.Tools) == 0 {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"tools\":[],\"reason\":\"pinned delegate is already active\"}"}}]}`))
			return
		}
		requestCount++
		switch requestCount {
		case 1:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"tool_calls":[{"id":"call_delegate","type":"function","function":{"name":"agent_delegate","arguments":"{\"mode\":\"plan_child\",\"prompt\":\"delegate demo\",\"title\":\"delegate demo\"}"}}]}}]}`))
		case 2:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"child done"}}]}`))
		default:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"parent done"}}]}`))
		}
	}))
	defer server.Close()
	if _, err := service.ConnectProvider(ctx, domain.ProviderConnectInput{ProviderID: "custom-api", Type: "openai-compatible", BaseURL: server.URL, ModelID: "test-model", APIKey: "test-key", Method: "api-key"}); err != nil {
		t.Fatal(err)
	}
	assistant, err := service.GetAgentMode(ctx, domain.AgentModeAssistant)
	if err != nil {
		t.Fatal(err)
	}
	child, err := service.SaveAgentMode(ctx, domain.AgentModeDefinition{ID: "plan_child", DisplayName: "Plan child", Prompt: "Plan one bounded task.", Mode: "subagent"})
	if err != nil {
		t.Fatal(err)
	}
	assistant.Subagents = []string{child.ID}
	if _, err := service.SaveAgentMode(ctx, assistant); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Source: domain.SessionSourceDesktop, AgentMode: domain.AgentModeAssistant, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetSessionActiveTools(ctx, domain.SessionActiveToolsInput{SessionID: session.ID, ToolNames: []string{"agent_delegate"}}); err != nil {
		t.Fatal(err)
	}
	run, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "start delegate"})
	if err != nil {
		t.Fatal(err)
	}
	if run.AssistantEvent == nil || run.AssistantEvent.Content != "parent done" {
		t.Fatalf("run = %#v", run)
	}
	agentRuns, err := service.ListAgentRuns(ctx, domain.AgentRunListRequest{SessionID: session.ID, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(agentRuns) != 1 {
		t.Fatalf("agentRuns = %#v, want one delegate run", agentRuns)
	}
	agentRun := agentRuns[0]
	if agentRun.Status != domain.AgentRunStatusCompleted || agentRun.Error != "" || agentRun.Result != "child done" {
		t.Fatalf("agentRun = %#v, want completed child result", agentRun)
	}
	if agentRun.Metadata["toolCallId"] != "call_delegate" {
		t.Fatalf("metadata = %#v, want toolCallId", agentRun.Metadata)
	}
}

func TestPermissionScopeHardDenials(t *testing.T) {
	engine := NewPermissionEngine(nil)
	writeTool := agentSpecTool{spec: domain.ToolSpec{Name: "write_file", Capability: "filesystem.write", Category: "filesystem"}}
	result := engine.Evaluate(context.Background(), writeTool, json.RawMessage(`{}`), domain.ToolExecutionContext{PermissionScope: "read_only"})
	if result.Decision != domain.PermissionDecisionDeny {
		t.Fatalf("read_only write decision = %#v", result)
	}
}
