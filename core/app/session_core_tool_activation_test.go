package app

import (
	"context"
	"testing"

	"aivo/core/domain"
)

func TestSessionCoreToolsStayActiveWhenOmittedFromManualSelection(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	active, err := service.GetSessionActiveTools(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !containsToolNames(active.CoreToolNames, "read", ExecCommandToolName, WriteStdinToolName, "edit", "write", "update_plan", "ask_user") {
		t.Fatalf("default core tools = %#v, want all core tools", active.CoreToolNames)
	}

	updated, err := service.SetSessionActiveTools(ctx, domain.SessionActiveToolsInput{
		SessionID: session.ID,
		ToolNames: []string{"extension_notes"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsToolNames(updated.CoreToolNames, "read", ExecCommandToolName, WriteStdinToolName, "edit", "write", "update_plan", "ask_user") || !containsToolNames(updated.ToolNames, "extension_notes") {
		t.Fatalf("updated active tools = %#v, core = %#v", updated.ToolNames, updated.CoreToolNames)
	}

	disabled := service.disabledCoreTools(ctx, session.ID)
	if len(disabled) != 0 {
		t.Fatalf("disabled core tools = %#v, want none", disabled)
	}
}

func TestSessionCoreToolsIgnoreLegacyDisabledMetadata(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	state, err := service.store.GetSessionExecutionState(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if state.Metadata == nil {
		state.Metadata = map[string]any{}
	}
	state.Metadata[sessionMetadataDisabledCoreTools] = []string{ExecCommandToolName, "write", "update_plan", "ask_user"}
	if _, err := service.store.UpsertSessionExecutionState(ctx, state); err != nil {
		t.Fatal(err)
	}

	active, err := service.GetSessionActiveTools(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !containsToolNames(active.CoreToolNames, "read", ExecCommandToolName, WriteStdinToolName, "edit", "write", "update_plan", "ask_user") {
		t.Fatalf("legacy metadata suppressed required core tools: %#v", active.CoreToolNames)
	}
}

func TestCoreToolAssemblyOmitsExplicitlyDisabledTool(t *testing.T) {
	registry, err := NewCodingToolRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	assembly := AssembleToolSpecsWithSources(registry, registry.Specs(), map[string]string{ExecCommandToolName: "disabled"})
	for _, spec := range assembly.Specs {
		if spec.Name == ExecCommandToolName {
			t.Fatalf("assembly exposed disabled exec_command: %#v", assembly.Specs)
		}
	}
	if !containsToolNames(toolSpecNames(assembly.Specs), "read", WriteStdinToolName, "edit", "write") {
		t.Fatalf("assembly = %#v, want remaining core tools", assembly.Specs)
	}
}

func containsToolNames(names []string, want ...string) bool {
	seen := map[string]bool{}
	for _, name := range names {
		seen[name] = true
	}
	for _, name := range want {
		if !seen[name] {
			return false
		}
	}
	return true
}

func toolSpecNames(specs []domain.ToolSpec) []string {
	names := make([]string, 0, len(specs))
	for _, spec := range specs {
		names = append(names, spec.Name)
	}
	return names
}
