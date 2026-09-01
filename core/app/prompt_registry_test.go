package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestBuiltinPromptCatalogSatisfiesRegisteredContracts(t *testing.T) {
	registry, err := NewBuiltinPromptRegistry()
	if err != nil {
		t.Fatal(err)
	}
	for id, contract := range registry.contracts {
		entry, ok := registry.builtins[id]
		if !ok {
			t.Fatalf("registered prompt %s has no builtin document", id)
		}
		if entry.Category != contract.Category || entry.Revision == "" {
			t.Fatalf("builtin prompt %s = %#v, contract = %#v", id, entry, contract)
		}
	}
}

func TestBuiltinPromptCatalogKeepsOnlyAssistantAndRequiredAgentWorkers(t *testing.T) {
	registry, err := NewBuiltinPromptRegistry()
	if err != nil {
		t.Fatal(err)
	}
	wanted := map[string]bool{
		"agent.assistant": true, "agent.summary": true, "agent.title": true, "agent.scheduler_worker": true,
	}
	for _, document := range registry.List() {
		if document.Category != domain.PromptCategoryAgent {
			continue
		}
		if !wanted[document.ID] {
			t.Fatalf("retired built-in Agent prompt remained: %#v", document)
		}
		delete(wanted, document.ID)
	}
	if len(wanted) != 0 {
		t.Fatalf("required built-in Agent prompts missing: %#v", wanted)
	}
}

func TestHostResourcePromptsPreferComprehensiveDomainSelection(t *testing.T) {
	registry, err := NewBuiltinPromptRegistry()
	if err != nil {
		t.Fatal(err)
	}
	groups, err := registry.Render("auxiliary.host_resource_groups.system", nil)
	if err != nil {
		t.Fatal(err)
	}
	resources := builtinPromptBody("auxiliary.host_resources.system")
	for _, prompt := range []string{groups, resources} {
		for _, want := range []string{"Select every", "video production", "do not minimize to one entry"} {
			if !strings.Contains(prompt, want) {
				t.Fatalf("Host resource prompt missing %q: %s", want, prompt)
			}
		}
	}
}

func TestPromptRegistryRetiresManagedSubagentProtocol(t *testing.T) {
	root := t.TempDir()
	registry, err := NewPromptRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, document := range registry.List() {
		if document.ID == "protocol.subagents" {
			t.Fatalf("retired subagent protocol remained in builtin catalog: %#v", document)
		}
	}
	overridePath := filepath.Join(root, "overrides", "protocol", "protocol.subagents.md")
	if err := os.MkdirAll(filepath.Dir(overridePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(overridePath, []byte(`---
schema: aivo.prompt/v1
id: protocol.subagents
category: protocol
title: Associated Subagents
enabled: true
---
Former override with {{subagents}}.
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := registry.Reload(); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Get("protocol.subagents"); err == nil {
		t.Fatal("retired subagent protocol override became active")
	}
	if _, err := registry.Save(domain.PromptDocumentInput{
		ID: "protocol.subagents", Category: domain.PromptCategoryProtocol,
		Title: "Associated Subagents", Body: "Replacement {{subagents}}.", Enabled: true,
	}); err == nil || !strings.Contains(err.Error(), "retired") {
		t.Fatalf("retired subagent protocol save error = %v", err)
	}
	if _, err := os.Stat(overridePath); err != nil {
		t.Fatalf("retired override should remain recoverable on disk: %v", err)
	}
}

func TestPromptRegistryInvalidWorkingDraftKeepsLastValidRevisionAcrossRestart(t *testing.T) {
	root := t.TempDir()
	registry, err := NewPromptRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	valid, err := registry.Save(domain.PromptDocumentInput{
		ID: "auxiliary.title.user", Category: domain.PromptCategoryAuxiliary,
		Title: "Title input", Body: "Title this:\n{{content}}", Enabled: true,
	})
	if err != nil || valid.Status != "valid" {
		t.Fatalf("valid save = %#v, err = %v", valid, err)
	}
	activeRevision := valid.ActiveRevision
	invalid, err := registry.Save(domain.PromptDocumentInput{
		ID: "auxiliary.title.user", Category: domain.PromptCategoryAuxiliary,
		Title: "Broken title input", Body: "Missing the required variable", Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if invalid.Status != "invalid" || !invalid.Fallback || invalid.ActiveRevision != activeRevision {
		t.Fatalf("invalid draft = %#v", invalid)
	}
	restarted, err := NewPromptRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := restarted.Get("auxiliary.title.user")
	if err != nil || !recovered.Fallback || recovered.ActiveRevision != activeRevision {
		t.Fatalf("restarted document = %#v, err = %v", recovered, err)
	}
	rendered, err := restarted.Render("auxiliary.title.user", map[string]string{"content": "Aivo"})
	if err != nil || rendered != "Title this:\nAivo" {
		t.Fatalf("active render = %q, err = %v", rendered, err)
	}
}

func TestPromptRegistryResetRemovesOverrideAndRestoresBuiltin(t *testing.T) {
	registry, err := NewPromptRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Save(domain.PromptDocumentInput{ID: "quick.run_task", Category: domain.PromptCategoryQuickPrompt, Title: "Run", Body: "Custom run", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	document, err := registry.Reset("quick.run_task")
	if err != nil {
		t.Fatal(err)
	}
	if document.Origin != "builtin" || document.Body != builtinPromptBody("quick.run_task") {
		t.Fatalf("reset document = %#v", document)
	}
}

func TestPromptRegistryRejectsSymlinksAndUnsafeCategories(t *testing.T) {
	root := t.TempDir()
	registry, err := NewPromptRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(target, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "overrides", "agent", "linked.md")
	if err := os.MkdirAll(filepath.Dir(link), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := registry.Reload(); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("reload symlink error = %v", err)
	}
	if _, err := registry.Save(domain.PromptDocumentInput{ID: "quick.safe", Category: "../../outside", Title: "Safe", Body: "Safe", Enabled: true}); err == nil {
		t.Fatal("unsafe category was accepted")
	}
}

func TestPromptSnapshotIsImmutableAcrossReload(t *testing.T) {
	registry, err := NewPromptRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()
	if _, err := registry.Save(domain.PromptDocumentInput{ID: "quick.run_task", Category: domain.PromptCategoryQuickPrompt, Title: "Run", Body: "Next revision", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if before.Body("quick.run_task") == "Next revision" {
		t.Fatal("existing snapshot changed after catalog reload")
	}
	if registry.Snapshot().Body("quick.run_task") != "Next revision" {
		t.Fatal("new snapshot did not observe applied revision")
	}
}
