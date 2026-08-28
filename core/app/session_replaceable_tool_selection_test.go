package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"aivo/core/domain"
)

type capturedReplaceableTool struct {
	Function struct {
		Name string `json:"name"`
	} `json:"function"`
}

func TestAgentRequestedToolReplacementChangesNextProviderSnapshot(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	events := []string{}
	for _, extension := range []struct {
		id, name, description, toolName, toolDescription string
	}{
		{id: "com.example.old", name: "Old capability", description: "Old capability", toolName: "old_auto", toolDescription: "Old capability"},
		{id: "com.example.new", name: "New capability", description: "New capability", toolName: "new_auto", toolDescription: "New capability"},
	} {
		extensionRoot := t.TempDir()
		writeTestExtensionManifest(t, extensionRoot, map[string]any{
			"schemaVersion": 2, "id": extension.id, "name": extension.name, "description": extension.description, "version": "1", "apiVersion": "2",
			"runtime": map[string]any{"type": "builtin"},
			"contributes": map[string]any{"tools": []any{
				map[string]any{"name": extension.toolName, "description": extension.toolDescription, "schema": map[string]any{"type": "object"}, "activation": "auto"},
			}},
		})
		id := extension.id
		service.extensionSupervisor.RegisterBuiltin(id, func() extensionRuntimeClient {
			return &builtinExtensionTestClient{events: &events}
		})
		status, err := service.extensionSupervisor.Discover(extensionRoot)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.extensionSupervisor.Enable(ctx, status.ID); err != nil {
			t.Fatal(err)
		}
	}

	type capturedRequest struct {
		Tools []capturedReplaceableTool `json:"tools"`
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
			_, _ = w.Write([]byte(`{"choices":[{"message":{"tool_calls":[{"id":"replace_tools","type":"function","function":{"name":"tool_resolve","arguments":"{\"intent\":\"new capability\"}"}}]}}]}`))
		case 2:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"intent\":\"use\",\"resources\":[{\"kind\":\"tool\",\"id\":\"new_auto\"}]}"}}]}`))
		default:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"replacement complete"}}]}`))
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
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	manualNames := append(coreToolNames(), projectQueryToolName)
	if _, err := service.SetSessionActiveTools(ctx, domain.SessionActiveToolsInput{SessionID: session.ID, ToolNames: manualNames}); err != nil {
		t.Fatal(err)
	}
	if err := service.replaceAutoSelectedTools(ctx, session.ID, []string{"old_auto"}); err != nil {
		t.Fatal(err)
	}
	result, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{SessionID: session.ID, Text: "use the current capability and refresh if needed"})
	if err != nil {
		t.Fatal(err)
	}
	if result.AssistantEvent == nil || result.AssistantEvent.Content != "replacement complete" {
		t.Fatalf("assistant result = %#v", result.AssistantEvent)
	}

	mu.Lock()
	captured := append([]capturedRequest(nil), requests...)
	mu.Unlock()
	if len(captured) != 3 {
		t.Fatalf("provider request count = %d, want primary, auxiliary replacement, primary", len(captured))
	}
	first := capturedToolNameSet(captured[0].Tools)
	auxiliary := capturedToolNameSet(captured[1].Tools)
	last := capturedToolNameSet(captured[2].Tools)
	if !first["old_auto"] || first["new_auto"] || !first[projectQueryToolName] || !first[ToolResolveName] {
		t.Fatalf("first Provider tools = %#v", first)
	}
	if len(auxiliary) != 0 {
		t.Fatalf("auxiliary replacement received executable tools: %#v", auxiliary)
	}
	if last["old_auto"] || !last["new_auto"] || !last[projectQueryToolName] || !last[ToolResolveName] {
		t.Fatalf("next Provider tools = %#v", last)
	}
	automatic, initialized := service.autoSelectedTools(ctx, session.ID)
	if !initialized || len(automatic) != 1 || !automatic["new_auto"] {
		t.Fatalf("final automatic set = %#v initialized=%t", automatic, initialized)
	}
}

func capturedToolNameSet(tools []capturedReplaceableTool) map[string]bool {
	names := map[string]bool{}
	for _, tool := range tools {
		names[tool.Function.Name] = true
	}
	return names
}

func TestToolResolveReplacesAutomaticSetWithoutChangingManualSet(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetSessionActiveTools(ctx, domain.SessionActiveToolsInput{SessionID: session.ID, ToolNames: []string{"manual_tool"}}); err != nil {
		t.Fatal(err)
	}
	if err := service.replaceAutoSelectedTools(ctx, session.ID, []string{"old_auto"}); err != nil {
		t.Fatal(err)
	}

	registry := NewRegistry()
	for _, name := range []string{"manual_tool", "old_auto", "new_auto"} {
		if err := registry.RegisterScoped(phase6EchoTool{spec: domain.ToolSpec{
			Name: name, Description: name, InputSchema: map[string]any{"type": "object"},
			Category: "extension", Toolsets: []string{"coding"}, ActivationPolicy: "auto",
		}}, domain.ToolSourceExtension, "replaceable", "v1"); err != nil {
			t.Fatal(err)
		}
	}
	resolver := func(_ context.Context, request ToolResolveRequest) (ToolResolveDecision, error) {
		return ToolResolveDecision{Names: []string{"new_auto"}, Reason: "new capability matched"}, nil
	}
	if err := registry.RegisterScoped(NewToolResolveTool(registry, resolver, service.replaceAutoSelectedTools), domain.ToolSourceBridge, "tool_selection", "v1"); err != nil {
		t.Fatal(err)
	}

	beforeActivations, beforeCandidates := service.preCallToolCandidates(ctx, session.ID, "turn-1", registry, registry.Specs())
	if len(beforeCandidates) != 0 || beforeActivations["manual_tool"] != "manual" || beforeActivations["old_auto"] != "automatic" {
		t.Fatalf("before activations = %#v candidates = %#v", beforeActivations, beforeCandidates)
	}
	before := AssembleToolSpecsWithSources(registry, registry.Specs(), beforeActivations)
	result := NewToolRuntime(registry, t.TempDir()).ExecuteWithContext(ctx, domain.ChatToolCall{
		ID: "replace", Name: ToolResolveName, Arguments: json.RawMessage(`{"intent":"use the new capability"}`),
	}, domain.ToolExecutionContext{
		SessionID: session.ID, TurnID: "turn-1", AgentMode: domain.AgentModeCode,
		AllowedToolsets: []string{"coding"}, ExpectedRegistrations: before.ExpectedRegistrations, ToolSnapshot: &before.Snapshot,
	})
	if !result.OK || result.Structured["status"] != "replaced" {
		t.Fatalf("tool_resolve result = %#v", result)
	}

	automatic, initialized := service.autoSelectedTools(ctx, session.ID)
	if !initialized || len(automatic) != 1 || !automatic["new_auto"] || automatic["old_auto"] {
		t.Fatalf("automatic set = %#v initialized=%t, want replacement only", automatic, initialized)
	}
	manual := service.rememberedDeferredTools(ctx, session.ID)
	if len(manual) != 1 || !manual["manual_tool"] {
		t.Fatalf("manual set changed during automatic replacement: %#v", manual)
	}
	afterActivations, afterCandidates := service.preCallToolCandidates(ctx, session.ID, "turn-1", registry, registry.Specs())
	if len(afterCandidates) != 0 || afterActivations["manual_tool"] != "manual" || afterActivations["new_auto"] != "automatic" || afterActivations["old_auto"] != "" {
		t.Fatalf("after activations = %#v candidates = %#v", afterActivations, afterCandidates)
	}
	after := AssembleToolSpecsWithSources(registry, registry.Specs(), afterActivations)
	names := toolSpecNames(after.Specs)
	if !containsToolNames(names, "manual_tool", "new_auto", ToolResolveName) || containsToolNames(names, "old_auto") {
		t.Fatalf("next Provider tools = %v", names)
	}
}

func TestToolResolveFailurePreservesAutomaticSet(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.replaceAutoSelectedTools(ctx, session.ID, []string{"old_auto"}); err != nil {
		t.Fatal(err)
	}
	registry := NewRegistry()
	if err := registry.RegisterScoped(phase6EchoTool{spec: domain.ToolSpec{
		Name: "candidate", Description: "candidate", InputSchema: map[string]any{"type": "object"}, Category: "extension", Toolsets: []string{"coding"}, ActivationPolicy: "auto",
	}}, domain.ToolSourceExtension, "replaceable", "v1"); err != nil {
		t.Fatal(err)
	}
	resolver := func(context.Context, ToolResolveRequest) (ToolResolveDecision, error) {
		return ToolResolveDecision{Reason: "no match"}, nil
	}
	tool := NewToolResolveTool(registry, resolver, service.replaceAutoSelectedTools)
	result := tool.Execute(ctx, json.RawMessage(`{"intent":"missing capability"}`), domain.ToolExecutionContext{SessionID: session.ID, ToolCallID: "resolve"})
	if result.OK || result.ToolError == nil || result.ToolError.Code != "no_available_tool" {
		t.Fatalf("resolve result = %#v, want required no-match", result)
	}
	automatic, initialized := service.autoSelectedTools(ctx, session.ID)
	if !initialized || len(automatic) != 1 || !automatic["old_auto"] {
		t.Fatalf("failed replacement changed automatic set: %#v initialized=%t", automatic, initialized)
	}
}
