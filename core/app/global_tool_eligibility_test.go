package app

import (
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"aivo/core/domain"
	"aivo/core/infra/persistence"
)

func TestOptionalGlobalToolVisibilityPersistsAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aivo.db")
	store, err := persistence.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store)
	ctx := context.Background()
	workspace := t.TempDir()

	updated, err := service.SetGlobalToolEnabled(ctx, domain.GlobalToolEnabledInput{Name: "grep", Enabled: false, WorkspaceRoot: workspace})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Enabled {
		t.Fatal("globally disabled catalog entry remained enabled")
	}
	assertGlobalToolVisibility(t, service, ctx, workspace, "grep", false)
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := persistence.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	restarted := NewService(reopened)
	assertGlobalToolVisibility(t, restarted, ctx, workspace, "grep", false)
	if _, err := restarted.SetGlobalToolEnabled(ctx, domain.GlobalToolEnabledInput{Name: "grep", Enabled: true, WorkspaceRoot: workspace}); err != nil {
		t.Fatal(err)
	}
	assertGlobalToolVisibility(t, restarted, ctx, workspace, "grep", true)
}

func TestRequiredCoreToolsRejectGlobalDisableAndIgnoreLegacyPreferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aivo.db")
	store, err := persistence.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	for _, name := range []string{"bash", "update_plan", "ask_user"} {
		if err := store.SetGlobalToolEnabled(ctx, name, false); err != nil {
			t.Fatal(err)
		}
	}
	service := NewService(store)
	workspace := t.TempDir()

	for _, name := range []string{"bash", "update_plan", "ask_user"} {
		assertGlobalToolVisibility(t, service, ctx, workspace, name, true)
		if _, err := service.SetGlobalToolEnabled(ctx, domain.GlobalToolEnabledInput{Name: name, Enabled: false, WorkspaceRoot: workspace}); err == nil {
			t.Fatalf("required core tool %s accepted global disable", name)
		}
	}
}

func TestGloballyHiddenToolRemainsExecutableWhenAlreadySelectedInSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aivo.db")
	store, err := persistence.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := NewService(store)
	ctx := context.Background()
	workspace := t.TempDir()
	if _, err := service.SetGlobalToolEnabled(ctx, domain.GlobalToolEnabledInput{Name: projectQueryToolName, Enabled: false, WorkspaceRoot: workspace}); err != nil {
		t.Fatal(err)
	}

	registry, runtime := service.toolsForWorkspace(workspace)
	assembly := AssembleToolSpecsWithSources(registry, registry.Specs(), map[string]string{projectQueryToolName: "automatic"})
	result := runtime.ExecuteWithContext(ctx, domain.ChatToolCall{
		ID:        "selected_project_query",
		Name:      projectQueryToolName,
		Arguments: json.RawMessage(`{}`),
	}, domain.ToolExecutionContext{
		WorkspaceRoot:         workspace,
		ExpectedRegistrations: assembly.ExpectedRegistrations,
		ToolSnapshot:          &assembly.Snapshot,
	})
	if !result.OK {
		t.Fatalf("selected tool result = %#v, want success after global visibility change", result)
	}
}

func TestGlobalToolVisibilityFiltersFutureCandidatesButNotCurrentSelection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aivo.db")
	store, err := persistence.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := NewService(store)
	ctx := context.Background()
	workspace := t.TempDir()
	if _, err := service.SetGlobalToolEnabled(ctx, domain.GlobalToolEnabledInput{Name: projectQueryToolName, Enabled: false, WorkspaceRoot: workspace}); err != nil {
		t.Fatal(err)
	}

	registry, _ := service.toolsForWorkspace(workspace)
	activations, candidates := service.preCallToolCandidates(ctx, "", "turn", registry, registry.Specs())
	if activations[projectQueryToolName] != "" {
		t.Fatalf("uninitialized automatic activation = %q", activations[projectQueryToolName])
	}
	candidates, err = service.filterGloballyVisibleToolCatalogEntries(ctx, candidates)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range candidates {
		if candidate.Name == projectQueryToolName {
			t.Fatal("globally hidden tool entered future automatic candidates")
		}
	}
	assembly := AssembleToolSpecsWithSources(registry, registry.Specs(), map[string]string{projectQueryToolName: "automatic"})
	found := false
	for _, spec := range assembly.Specs {
		if spec.Name == projectQueryToolName {
			found = true
		}
	}
	if !found {
		t.Fatal("globally hidden current selection was revoked from Provider declarations")
	}
}

func TestGlobalToolVisibilityBlocksNewManualAndAuxiliarySelectionOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aivo.db")
	store, err := persistence.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := NewService(store)
	ctx := context.Background()
	workspace := t.TempDir()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: workspace})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetSessionActiveTools(ctx, domain.SessionActiveToolsInput{SessionID: session.ID, ToolNames: []string{projectAddToolName}}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetGlobalToolEnabled(ctx, domain.GlobalToolEnabledInput{Name: projectAddToolName, Enabled: false, WorkspaceRoot: workspace}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetGlobalToolEnabled(ctx, domain.GlobalToolEnabledInput{Name: projectQueryToolName, Enabled: false, WorkspaceRoot: workspace}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetSessionActiveTools(ctx, domain.SessionActiveToolsInput{SessionID: session.ID, ToolNames: []string{projectAddToolName}}); err != nil {
		t.Fatalf("existing hidden manual selection was revoked: %v", err)
	}
	if _, err := service.SetSessionActiveTools(ctx, domain.SessionActiveToolsInput{SessionID: session.ID, ToolNames: []string{projectAddToolName, projectQueryToolName}}); err == nil {
		t.Fatal("new globally hidden manual selection was accepted")
	}
	decision, err := service.resolveSessionToolReplacement(ctx, ToolResolveRequest{
		Intent: "query or add projects", MaxTools: 8, SessionID: session.ID, AgentMode: domain.AgentModeCode,
		Candidates: []domain.ToolCatalogEntry{
			{Name: projectAddToolName, Description: "add projects"},
			{Name: projectQueryToolName, Description: "query projects"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(decision.Names) != 0 {
		t.Fatalf("hidden/manual tools entered auxiliary replacement: %#v", decision)
	}
}

func assertGlobalToolVisibility(t *testing.T, service *Service, ctx context.Context, workspace, toolName string, wantEnabled bool) {
	t.Helper()
	entries, err := service.ListToolCatalog(ctx, domain.ToolCatalogInput{WorkspaceRoot: workspace})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range entries {
		if entry.Name == toolName {
			found = true
			if entry.Enabled != wantEnabled {
				t.Fatalf("%s catalog enabled = %t, want %t", toolName, entry.Enabled, wantEnabled)
			}
		}
	}
	if !found {
		t.Fatalf("%s is missing from the management catalog", toolName)
	}

	registry, _ := service.toolsForWorkspace(workspace)
	assembly := AssembleToolSpecsWithSources(registry, registry.Specs(), map[string]string{toolName: "manual"})
	if !containsToolNames(toolSpecNames(assembly.Specs), toolName) {
		t.Fatal("global visibility change revoked an already selected tool")
	}
}

func TestPiStyleOptionalFileToolsSupportGlobalManualAndAutomaticSelection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aivo.db")
	store, err := persistence.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := NewService(store)
	ctx := context.Background()
	workspace := t.TempDir()

	entries, err := service.ListToolCatalog(ctx, domain.ToolCatalogInput{})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"grep", "find", "ls"} {
		entry, ok := catalogEntryNamed(entries, name)
		if !ok || entry.Source != domain.ToolSourceBuiltin || !entry.Enabled {
			t.Fatalf("global catalog %s = %#v ok=%t, want enabled builtin", name, entry, ok)
		}
	}
	for _, oldName := range []string{"search_files", "glob", "list_files"} {
		if _, ok := catalogEntryNamed(entries, oldName); ok {
			t.Fatalf("removed tool name %s remained globally registered", oldName)
		}
	}

	if _, err := service.SetGlobalToolEnabled(ctx, domain.GlobalToolEnabledInput{Name: "grep", Enabled: false}); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: workspace})
	if err != nil {
		t.Fatal(err)
	}
	manualNames := append(coreToolNames(), "find", "ls")
	active, err := service.SetSessionActiveTools(ctx, domain.SessionActiveToolsInput{SessionID: session.ID, ToolNames: manualNames})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(active.ToolNames, []string{"find", "ls"}) || !reflect.DeepEqual(active.CoreToolNames, coreToolNames()) {
		t.Fatalf("manual activation = %#v, want optional find/ls plus six core tools", active)
	}
	if _, err := service.SetSessionActiveTools(ctx, domain.SessionActiveToolsInput{SessionID: session.ID, ToolNames: append(manualNames, "grep")}); err == nil {
		t.Fatal("globally hidden grep was accepted as a new manual activation")
	}

	registry, _ := service.toolsForWorkspace(workspace)
	_, candidates := service.preCallToolCandidates(ctx, "", "turn", registry, registry.Specs())
	candidates, err = service.filterGloballyVisibleToolCatalogEntries(ctx, candidates)
	if err != nil {
		t.Fatal(err)
	}
	automaticNames := hostStandaloneToolNames(candidates)
	if !containsToolNames(automaticNames, "find", "ls") || containsToolNames(automaticNames, "grep") {
		t.Fatalf("automatic standalone tools = %#v, want find/ls and globally hidden grep excluded", automaticNames)
	}
}
