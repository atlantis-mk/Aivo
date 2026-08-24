package app

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"aivo/core/domain"
	"aivo/core/infra/persistence"
)

func TestManagedAgentModesComposePersistAndProtectReferences(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	path := filepath.Join(t.TempDir(), "aivo.db")
	store, err := persistence.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store)
	ctx := context.Background()

	modes, err := service.ListAgentModes(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(modes) != 1 {
		t.Fatalf("fresh visible modes = %#v, want Assistant only", modes)
	}
	assistant := findAgentMode(t, modes, domain.AgentModeAssistant)
	if !assistant.BuiltIn || assistant.Overridden || assistant.Source != "builtin" {
		t.Fatalf("fresh Assistant origin = %#v", assistant)
	}

	custom, err := service.SaveAgentMode(ctx, domain.AgentModeDefinition{
		ID: "research", DisplayName: "Research", Description: "Read-only research",
		Prompt: "Investigate carefully.", Toolsets: []string{"coding", "web"}, PermissionScope: "read_only", Mode: "all",
	})
	if err != nil {
		t.Fatal(err)
	}
	if custom.BuiltIn || custom.Source != "user" || custom.Revision == "" {
		t.Fatalf("custom origin = %#v", custom)
	}
	if len(custom.Toolsets) != 1 || custom.Toolsets[0] != "safe" {
		t.Fatalf("custom runtime toolsets = %#v, want safe", custom.Toolsets)
	}
	rawCustom, err := json.Marshal(custom)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rawCustom), "toolsets") {
		t.Fatalf("managed JSON exposed toolsets: %s", rawCustom)
	}

	projectRoot := t.TempDir()
	writeRuntimeConfigTestFile(t, filepath.Join(projectRoot, ".aivo", "config.json"), `{
	  "agents": {"research": {"prompt": "Project-specific research.", "toolsets": ["web"], "permissionScope": "read_only"}}
}`)
	projectModes, err := service.ListAgentModesForProject(ctx, projectRoot, false)
	if err != nil {
		t.Fatal(err)
	}
	projectMode := findAgentMode(t, projectModes, "research")
	if projectMode.Prompt != "Project-specific research." || projectMode.Source != "project" || len(projectMode.Toolsets) != 1 || projectMode.Toolsets[0] != "web" {
		t.Fatalf("project overlay = %#v", projectMode)
	}

	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, AgentMode: "research", ProjectPath: projectRoot})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteAgentMode(ctx, "research"); err == nil || !strings.Contains(err.Error(), "referenced") {
		t.Fatalf("referenced delete error = %v", err)
	}
	if _, err := service.SetSessionAgentMode(ctx, domain.SetSessionAgentModeInput{SessionID: session.ID, Mode: domain.AgentModeAssistant}); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteAgentMode(ctx, "research"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetAgentMode(ctx, "research"); err == nil {
		t.Fatal("deleted custom mode remained in catalog")
	}

	assistant.Prompt = "Customized built-in Assistant prompt."
	assistant.Toolsets = []string{"safe"}
	updated, err := service.SaveAgentMode(ctx, assistant)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.BuiltIn || !updated.Overridden || updated.Prompt != assistant.Prompt || len(updated.Toolsets) != 1 || updated.Toolsets[0] != "*" {
		t.Fatalf("updated built-in = %#v", updated)
	}
	if err := service.DeleteAgentMode(ctx, domain.AgentModeAssistant); err != nil {
		t.Fatal(err)
	}
	restored, err := service.GetAgentMode(ctx, domain.AgentModeAssistant)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Overridden || restored.Prompt == assistant.Prompt {
		t.Fatalf("restored built-in = %#v", restored)
	}
	child, err := service.SaveAgentMode(ctx, domain.AgentModeDefinition{
		ID: "persisted_child", DisplayName: "Persisted child", Prompt: "Handle one bounded task.", Mode: "subagent",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveAgentMode(ctx, domain.AgentModeDefinition{
		ID: "persisted", DisplayName: "Persisted", Prompt: "Survive restart.", Toolsets: []string{"coding"}, Mode: "all", Subagents: []string{child.ID},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveAgentMode(ctx, domain.AgentModeDefinition{ID: domain.AgentModeSummary, DisplayName: "Summary", Prompt: "unsafe"}); err == nil {
		t.Fatal("protected worker mode was editable")
	}

	service.Shutdown()
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := persistence.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	restarted := NewService(reopened)
	defer restarted.Shutdown()
	if _, err := restarted.GetAgentMode(ctx, domain.AgentModeAssistant); err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.GetAgentMode(ctx, "research"); err == nil {
		t.Fatal("deleted mode reappeared after restart")
	}
	persisted, err := restarted.GetAgentMode(ctx, "persisted")
	if err != nil || persisted.Prompt != "Survive restart." || len(persisted.Toolsets) != 1 || persisted.Toolsets[0] != "safe" || len(persisted.Subagents) != 1 || persisted.Subagents[0] != child.ID {
		t.Fatalf("persisted mode after restart = %#v err = %v", persisted, err)
	}
}

func TestCreateAgentPromptPersistsRoleAndSubagentAssociations(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()

	child, err := service.SaveAgentMode(ctx, domain.AgentModeDefinition{
		ID: "creation_child", DisplayName: "Creation child", Prompt: "Handle one bounded task.", Mode: "subagent",
	})
	if err != nil {
		t.Fatal(err)
	}
	created, err := service.CreateAgentPrompt(ctx, domain.CreateAgentPromptInput{
		ID: "creation_parent", Title: "Creation parent", Body: "Coordinate bounded work.",
		Mode: "primary", Subagents: []string{child.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Mode != "primary" || len(created.Subagents) != 1 || created.Subagents[0] != child.ID {
		t.Fatalf("created Agent associations = %#v", created)
	}
	if _, err := service.GetPromptDocument(ctx, "agent.creation_parent"); err != nil {
		t.Fatalf("created Agent prompt missing: %v", err)
	}

	if _, err := service.CreateAgentPrompt(ctx, domain.CreateAgentPromptInput{
		ID: "invalid_creation_parent", Title: "Invalid parent", Body: "Invalid association owner.",
		Mode: "subagent", Subagents: []string{child.ID},
	}); err == nil || !strings.Contains(err.Error(), "cannot associate") {
		t.Fatalf("subagent-only association error = %v", err)
	}
	if _, err := service.GetPromptDocument(ctx, "agent.invalid_creation_parent"); err == nil {
		t.Fatal("failed Agent creation left its prompt behind")
	}
}

func TestManagedAgentModeValidation(t *testing.T) {
	store, err := persistence.Open(filepath.Join(t.TempDir(), "aivo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := NewService(store)
	defer service.Shutdown()
	cases := []domain.AgentModeDefinition{
		{ID: "Bad ID", DisplayName: "Bad", Prompt: "Prompt"},
		{ID: "blank", DisplayName: "", Prompt: "Prompt"},
		{ID: "blank_prompt", DisplayName: "Blank", Prompt: ""},
		{ID: "bad_role", DisplayName: "Bad", Prompt: "Prompt", Mode: "worker"},
		{ID: "bad_scope", DisplayName: "Bad", Prompt: "Prompt", PermissionScope: "root"},
	}
	for _, definition := range cases {
		if _, err := service.SaveAgentMode(context.Background(), definition); err == nil {
			t.Fatalf("invalid definition saved: %#v", definition)
		}
	}
}

func TestManagedAgentModeAssociationsPersistAndProtectTargets(t *testing.T) {
	store, err := persistence.Open(filepath.Join(t.TempDir(), "aivo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := NewService(store)
	defer service.Shutdown()
	ctx := context.Background()

	child, err := service.SaveAgentMode(ctx, domain.AgentModeDefinition{
		ID: "research_child", DisplayName: "Research child", Prompt: "Research one bounded question.", Mode: "subagent",
	})
	if err != nil {
		t.Fatal(err)
	}
	parent, err := service.SaveAgentMode(ctx, domain.AgentModeDefinition{
		ID: "orchestrator", DisplayName: "Orchestrator", Prompt: "Delegate independent research when useful.", Mode: "primary",
		Subagents: []string{child.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(parent.Subagents) != 1 || parent.Subagents[0] != child.ID {
		t.Fatalf("parent associations = %#v", parent.Subagents)
	}
	if err := service.DeleteAgentMode(ctx, child.ID); err == nil || !strings.Contains(err.Error(), "another agent mode") {
		t.Fatalf("associated target delete error = %v", err)
	}
	child.Mode = "primary"
	if _, err := service.SaveAgentMode(ctx, child); err == nil || !strings.Contains(err.Error(), "unavailable subagent") {
		t.Fatalf("referenced role-change error = %v", err)
	}
	parent.Subagents = nil
	if _, err := service.SaveAgentMode(ctx, parent); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteAgentMode(ctx, child.ID); err != nil {
		t.Fatal(err)
	}
}

func TestManagedAgentModeAssociationValidation(t *testing.T) {
	store, err := persistence.Open(filepath.Join(t.TempDir(), "aivo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := NewService(store)
	defer service.Shutdown()
	ctx := context.Background()

	cases := []domain.AgentModeDefinition{
		{ID: "self", DisplayName: "Self", Prompt: "Prompt", Subagents: []string{"self"}},
		{ID: "duplicate", DisplayName: "Duplicate", Prompt: "Prompt", Subagents: []string{domain.AgentModeReview, domain.AgentModeReview}},
		{ID: "missing", DisplayName: "Missing", Prompt: "Prompt", Subagents: []string{"not_configured"}},
		{ID: "child_owner", DisplayName: "Child owner", Prompt: "Prompt", Mode: "subagent", Subagents: []string{domain.AgentModeReview}},
		{ID: "hidden_target", DisplayName: "Hidden target", Prompt: "Prompt", Subagents: []string{domain.AgentModePlanner}},
	}
	for _, definition := range cases {
		if _, err := service.SaveAgentMode(ctx, definition); err == nil {
			t.Fatalf("invalid association saved: %#v", definition)
		}
	}
}

func findAgentMode(t *testing.T, modes []domain.AgentModeDefinition, id string) domain.AgentModeDefinition {
	t.Helper()
	for _, mode := range modes {
		if mode.ID == id {
			return mode
		}
	}
	t.Fatalf("agent mode %q not found in %#v", id, modes)
	return domain.AgentModeDefinition{}
}
