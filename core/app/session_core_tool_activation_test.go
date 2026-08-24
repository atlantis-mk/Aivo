package app

import (
	"context"
	"testing"

	"aivo/core/domain"
)

func TestSessionCoreToolActivationDefaultsAndCanDisableOneTool(t *testing.T) {
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
	if !containsToolNames(active.CoreToolNames, "read", "bash", "edit", "write") {
		t.Fatalf("default core tools = %#v, want all core tools", active.CoreToolNames)
	}

	updated, err := service.SetSessionActiveTools(ctx, domain.SessionActiveToolsInput{
		SessionID: session.ID,
		ToolNames: []string{"read", "edit", "write", "extension_notes"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if containsToolNames(updated.CoreToolNames, "bash") || !containsToolNames(updated.CoreToolNames, "read", "edit", "write") || !containsToolNames(updated.ToolNames, "extension_notes") {
		t.Fatalf("updated active tools = %#v, core = %#v", updated.ToolNames, updated.CoreToolNames)
	}

	disabled := service.disabledCoreTools(ctx, session.ID)
	if !disabled["bash"] || len(disabled) != 1 {
		t.Fatalf("disabled core tools = %#v, want only bash", disabled)
	}
}

func TestCoreToolAssemblyOmitsExplicitlyDisabledTool(t *testing.T) {
	registry, err := NewCodingToolRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	assembly := AssembleToolSpecsWithSources(registry, registry.Specs(), map[string]string{"bash": "disabled"})
	for _, spec := range assembly.Specs {
		if spec.Name == "bash" {
			t.Fatalf("assembly exposed disabled bash: %#v", assembly.Specs)
		}
	}
	if !containsToolNames(toolSpecNames(assembly.Specs), "read", "edit", "write") {
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
