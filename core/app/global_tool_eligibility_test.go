package app

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"aivo/core/domain"
	"aivo/core/infra/persistence"
)

func TestGlobalToolVisibilityPersistsAcrossRestartWithoutRevokingSelectedTools(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aivo.db")
	store, err := persistence.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store)
	ctx := context.Background()
	workspace := t.TempDir()

	updated, err := service.SetGlobalToolEnabled(ctx, domain.GlobalToolEnabledInput{Name: "bash", Enabled: false, WorkspaceRoot: workspace})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Enabled {
		t.Fatal("globally disabled catalog entry remained enabled")
	}
	assertGlobalToolVisibility(t, service, ctx, workspace, false)
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := persistence.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	restarted := NewService(reopened)
	assertGlobalToolVisibility(t, restarted, ctx, workspace, false)
	if _, err := restarted.SetGlobalToolEnabled(ctx, domain.GlobalToolEnabledInput{Name: "bash", Enabled: true, WorkspaceRoot: workspace}); err != nil {
		t.Fatal(err)
	}
	assertGlobalToolVisibility(t, restarted, ctx, workspace, true)
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

func assertGlobalToolVisibility(t *testing.T, service *Service, ctx context.Context, workspace string, wantEnabled bool) {
	t.Helper()
	entries, err := service.ListToolCatalog(ctx, domain.ToolCatalogInput{WorkspaceRoot: workspace})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range entries {
		if entry.Name == "bash" {
			found = true
			if entry.Enabled != wantEnabled {
				t.Fatalf("bash catalog enabled = %t, want %t", entry.Enabled, wantEnabled)
			}
		}
	}
	if !found {
		t.Fatal("bash is missing from the management catalog")
	}

	registry, _ := service.toolsForWorkspace(workspace)
	assembly := AssembleToolSpecsWithSources(registry, registry.Specs(), map[string]string{"bash": "manual"})
	if !containsToolNames(toolSpecNames(assembly.Specs), "bash") {
		t.Fatal("global visibility change revoked an already selected tool")
	}
}
