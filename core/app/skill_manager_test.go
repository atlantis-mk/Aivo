package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aivo/core/domain"
	"aivo/core/infra/persistence"
)

func TestSkillScanImportLoadAndContext(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()

	sourceRoot := filepath.Join(home, ".claude", "skills", "code-review")
	writeSkill(t, sourceRoot, "code-review", "Review code changes", "Use this skill to review code.")
	if err := os.WriteFile(filepath.Join(sourceRoot, "reference.md"), []byte("reference"), 0o644); err != nil {
		t.Fatal(err)
	}

	scan, err := service.ScanGlobalSkills(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if scan.Scanned != 1 || len(scan.Candidates) != 1 {
		t.Fatalf("scan = %+v, want one candidate", scan)
	}

	imported, err := service.ImportSkill(ctx, domain.SkillImportInput{CandidateID: scan.Candidates[0].ID, TargetScope: domain.SkillScopeGlobal})
	if err != nil {
		t.Fatal(err)
	}
	if imported.Name != "code-review" || !imported.Enabled {
		t.Fatalf("imported = %+v", imported)
	}
	if _, err := os.Stat(filepath.Join(sourceRoot, "SKILL.md")); err != nil {
		t.Fatalf("source skill should remain in place: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".aivo", "skills", "code-review", "SKILL.md")); err != nil {
		t.Fatalf("managed skill copy missing: %v", err)
	}

	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.LoadSkillIntoSession(ctx, domain.LoadSkillIntoSessionInput{SessionID: session.ID, SkillID: imported.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.LoadSkillIntoSession(ctx, domain.LoadSkillIntoSessionInput{SessionID: session.ID, SkillID: imported.ID}); err == nil {
		t.Fatal("duplicate skill load should fail without reload")
	}
	result, err := service.BuildSessionContext(ctx, domain.BuildSessionContextRequest{SessionID: session.ID, CharacterBudget: 8000})
	if err != nil {
		t.Fatal(err)
	}
	joined := contextSectionsText(result.Sections)
	if strings.Contains(joined, "<available_skills>") || strings.Contains(joined, "skill_guidance") {
		t.Fatalf("context should not expose the available skill catalog: %q", joined)
	}
	if !strings.Contains(joined, `<skill_content name="code-review"`) || !strings.Contains(joined, "# Skill: code-review") || !strings.Contains(joined, "Use this skill to review code.") {
		t.Fatalf("context missing loaded skill: %q", joined)
	}
	if strings.Contains(joined, "---\nname: code-review") {
		t.Fatalf("context should not include skill frontmatter: %q", joined)
	}
	if !strings.Contains(joined, "Skill directory: "+filepath.Join(home, ".aivo", "skills", "code-review")) {
		t.Fatalf("context missing OpenCode skill base directory line: %q", joined)
	}
	if !strings.Contains(joined, "<skill_resources>") || !strings.Contains(joined, "<file>reference.md</file>") {
		t.Fatalf("context missing sampled skill files: %q", joined)
	}
	registry, _ := service.toolsForWorkspace(session.ProjectPath)
	if registry == nil {
		t.Fatal("registry is nil")
	}
	if _, ok := registry.Get("skill"); ok {
		t.Fatal("skill is a Host context protocol and must not be a model execution tool")
	}
	if _, ok := registry.Get("skill_load"); ok {
		t.Fatal("model tool should not be registered as skill_load")
	}
}

func TestSkillListDoesNotScanImplicitly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()

	writeSkill(t, filepath.Join(home, ".claude", "skills", "list-skill"), "list-skill", "List test", "Use this skill.")

	before, err := service.ListSkills(ctx, domain.SkillListInput{IncludeCandidates: true, IncludeIgnored: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Entries) != 0 || len(before.Candidates) != 0 {
		t.Fatalf("list before scan = %+v, want empty", before)
	}
	if _, err := service.ScanGlobalSkills(ctx); err != nil {
		t.Fatal(err)
	}
	after, err := service.ListSkills(ctx, domain.SkillListInput{IncludeCandidates: true, IncludeIgnored: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Candidates) != 1 || after.Candidates[0].Name != "list-skill" {
		t.Fatalf("list after scan = %+v, want scanned candidate", after)
	}
}

func TestManagedSkillEditSavesDescriptionAndInstructions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()

	root := filepath.Join(home, ".aivo", "skills", "editable-skill")
	writeSkill(t, root, "editable-skill", "Original description", "Original instructions.")
	if _, err := service.ScanGlobalSkills(ctx); err != nil {
		t.Fatal(err)
	}
	list, err := service.ListSkills(ctx, domain.SkillListInput{IncludeDisabled: true})
	if err != nil || len(list.Entries) != 1 {
		t.Fatalf("skills = %+v, err = %v", list.Entries, err)
	}

	editor, err := service.GetManagedSkillForEdit(ctx, list.Entries[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.UpdateManagedSkill(ctx, domain.SkillUpdateInput{
		SkillID: editor.Skill.ID, Description: "Updated description", Content: "Use the updated workflow.", ExpectedContentHash: editor.Skill.ContentHash,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Skill.Name != "editable-skill" || updated.Skill.Description != "Updated description" || updated.Content != "Use the updated workflow." {
		t.Fatalf("updated = %+v", updated)
	}
	if updated.Skill.ContentHash == editor.Skill.ContentHash {
		t.Fatal("content hash did not change")
	}
	raw, err := os.ReadFile(filepath.Join(root, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, `description: "Updated description"`) || !strings.Contains(text, "Use the updated workflow.") {
		t.Fatalf("saved SKILL.md = %q", text)
	}
}

func TestManagedSkillEditRejectsStaleRevisionAndOutsidePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()

	root := filepath.Join(home, ".aivo", "skills", "stale-skill")
	writeSkill(t, root, "stale-skill", "Original", "Original instructions.")
	if _, err := service.ScanGlobalSkills(ctx); err != nil {
		t.Fatal(err)
	}
	list, err := service.ListSkills(ctx, domain.SkillListInput{IncludeDisabled: true})
	if err != nil || len(list.Entries) != 1 {
		t.Fatalf("skills = %+v, err = %v", list.Entries, err)
	}
	editor, err := service.GetManagedSkillForEdit(ctx, list.Entries[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	writeSkill(t, root, "stale-skill", "Changed elsewhere", "Newer instructions.")
	if _, err := service.UpdateManagedSkill(ctx, domain.SkillUpdateInput{
		SkillID: editor.Skill.ID, Description: "Overwrite", Content: "Stale editor content.", ExpectedContentHash: editor.Skill.ContentHash,
	}); err == nil || !strings.Contains(err.Error(), "changed since") {
		t.Fatalf("stale update error = %v", err)
	}
	raw, _ := os.ReadFile(filepath.Join(root, "SKILL.md"))
	if !strings.Contains(string(raw), "Newer instructions.") || strings.Contains(string(raw), "Stale editor content.") {
		t.Fatalf("stale update changed file: %q", string(raw))
	}

	outsideRoot := filepath.Join(home, "outside", "outside-skill")
	writeSkill(t, outsideRoot, "outside-skill", "Outside", "Outside instructions.")
	outside, err := service.ensureSkillManager().store.SaveSkill(ctx, domain.SkillEntry{
		ID: "outside-skill", Name: "outside-skill", Description: "Outside", Scope: domain.SkillScopeGlobal,
		Source: domain.SkillSourceAivo, RootPath: outsideRoot, SkillPath: filepath.Join(outsideRoot, "SKILL.md"), Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetManagedSkillForEdit(ctx, outside.ID); err == nil || !strings.Contains(err.Error(), "Aivo-managed") {
		t.Fatalf("outside path error = %v", err)
	}
}

func TestSkillScanIgnoresSameNameContentConflicts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()

	writeSkill(t, filepath.Join(home, ".aivo", "skills", "dup-skill"), "dup-skill", "Managed", "Managed content.")
	if _, err := service.ScanGlobalSkills(ctx); err != nil {
		t.Fatal(err)
	}
	writeSkill(t, filepath.Join(home, ".agents", "skills", "dup-skill"), "dup-skill", "External", "Different content.")
	scan, err := service.ScanGlobalSkills(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if scan.Conflicts != 0 {
		t.Fatalf("conflicts = %d, want 0; scan=%+v", scan.Conflicts, scan)
	}
	var ignored domain.SkillImportCandidate
	for _, candidate := range scan.Candidates {
		if candidate.Status == domain.SkillCandidateStatusIgnored {
			ignored = candidate
		}
	}
	if ignored.ID == "" || ignored.ConflictID == "" {
		t.Fatalf("missing ignored conflict candidate: %+v", scan.Candidates)
	}
}

func TestSkillIgnoreByNamePersistsAcrossIncrementalScans(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()

	writeSkill(t, filepath.Join(home, ".claude", "skills", "ignore-me"), "ignore-me", "Ignore Claude", "Claude content.")
	writeSkill(t, filepath.Join(home, ".agents", "skills", "ignore-me"), "ignore-me", "Ignore Agents", "Agents content.")
	if _, err := service.ScanGlobalSkills(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := service.IgnoreSkillCandidatesByName(ctx, domain.SkillIgnoreCandidatesInput{Name: "ignore-me"}); err != nil {
		t.Fatal(err)
	}
	scan, err := service.ScanGlobalSkills(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(scan.Candidates) != 2 {
		t.Fatalf("scan candidates = %+v, want 2", scan.Candidates)
	}
	for _, candidate := range scan.Candidates {
		if candidate.Status != domain.SkillCandidateStatusIgnored {
			t.Fatalf("candidate = %+v, want ignored", candidate)
		}
	}
	list, err := service.ListSkills(ctx, domain.SkillListInput{IncludeCandidates: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Candidates) != 0 {
		t.Fatalf("non-ignored candidate list = %+v, want empty", list.Candidates)
	}
	withIgnored, err := service.ListSkills(ctx, domain.SkillListInput{IncludeCandidates: true, IncludeIgnored: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(withIgnored.Candidates) != 2 {
		t.Fatalf("ignored candidate list = %+v, want 2", withIgnored.Candidates)
	}
}

func TestSkillToolOnlyListsAndLoadsImportedSkills(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()

	writeSkill(t, filepath.Join(home, ".claude", "skills", "tool-skill"), "tool-skill", "Tool import", "Loaded by tool.")
	if _, err := service.ScanGlobalSkills(ctx); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	tool := NewSkillLoadTool(service)
	listed := tool.Execute(ctx, json.RawMessage(`{"mode":"list"}`), domain.ToolExecutionContext{SessionID: session.ID, ToolCallID: "skill-list"})
	if !listed.OK || strings.Contains(listed.ModelContent, "tool-skill") {
		t.Fatalf("list result = %+v, pending skill must not be visible to the model", listed)
	}
	blockedPending := tool.Execute(ctx, json.RawMessage(`{"mode":"activate","names":["tool-skill"]}`), domain.ToolExecutionContext{SessionID: session.ID, ToolCallID: "skill-pending"})
	if !blockedPending.OK || !strings.Contains(blockedPending.Content, "no_match") {
		t.Fatalf("pending load result = %+v, pending skill must not be loadable by the model", blockedPending)
	}
	candidates, err := service.ListSkills(ctx, domain.SkillListInput{IncludeCandidates: true})
	if err != nil || len(candidates.Candidates) != 1 {
		t.Fatalf("candidates = %+v, err = %v", candidates, err)
	}
	if _, err := service.ImportSkill(ctx, domain.SkillImportInput{CandidateID: candidates.Candidates[0].ID}); err != nil {
		t.Fatal(err)
	}
	listed = tool.Execute(ctx, json.RawMessage(`{"mode":"list"}`), domain.ToolExecutionContext{SessionID: session.ID, ToolCallID: "skill-list-imported"})
	if !listed.OK || !strings.Contains(listed.ModelContent, "<name>tool-skill</name>") || strings.Contains(listed.ModelContent, "Loaded by tool.") {
		t.Fatalf("list result = %+v, want imported catalog metadata without skill body", listed)
	}
	load := tool.Execute(ctx, json.RawMessage(`{"mode":"activate","names":["tool-skill"]}`), domain.ToolExecutionContext{SessionID: session.ID, ToolCallID: "skill-call-1"})
	if !load.OK || !strings.Contains(load.ModelContent, "Loaded by tool.") {
		t.Fatalf("load result = %+v, want imported and loaded", load)
	}
	duplicate := tool.Execute(ctx, json.RawMessage(`{"mode":"activate","names":["tool-skill"]}`), domain.ToolExecutionContext{SessionID: session.ID, ToolCallID: "skill-call-duplicate"})
	if !duplicate.OK || !strings.Contains(duplicate.ModelContent, "already_active") || strings.Contains(duplicate.ModelContent, "Loaded by tool.") {
		t.Fatalf("duplicate activation = %+v, want deduplicated activation without reinjection", duplicate)
	}
	list, err := service.ListSkills(ctx, domain.SkillListInput{IncludeCandidates: true, IncludeDisabled: true, IncludeIgnored: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Entries) != 1 || list.Entries[0].Name != "tool-skill" {
		t.Fatalf("entries = %+v, want imported tool-skill", list.Entries)
	}
	if len(list.Candidates) != 1 || list.Candidates[0].Status != domain.SkillCandidateStatusImported {
		t.Fatalf("candidates = %+v, want imported candidate", list.Candidates)
	}
	if _, err := service.SetSkillEnabled(ctx, domain.SkillEnabledInput{SkillID: list.Entries[0].ID, Enabled: false}); err != nil {
		t.Fatal(err)
	}
	nextSession, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	blocked := tool.Execute(ctx, json.RawMessage(`{"mode":"activate","names":["tool-skill"]}`), domain.ToolExecutionContext{SessionID: nextSession.ID, ToolCallID: "skill-call-3"})
	if !blocked.OK || !strings.Contains(blocked.Content, "no_match") {
		t.Fatalf("blocked load = %+v, want disabled skill omitted from the resolvable catalog", blocked)
	}
	if _, err := service.SetSkillEnabled(ctx, domain.SkillEnabledInput{SkillID: list.Entries[0].ID, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	reloaded := tool.Execute(ctx, json.RawMessage(`{"mode":"activate","names":["tool-skill"]}`), domain.ToolExecutionContext{SessionID: nextSession.ID, ToolCallID: "skill-call-5"})
	if !reloaded.OK {
		t.Fatalf("reload result = %+v, want ok", reloaded)
	}
}

func TestSkillToolDoesNotImportIgnoredCandidate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()

	writeSkill(t, filepath.Join(home, ".claude", "skills", "ignored-tool"), "ignored-tool", "Ignored tool", "Should not load.")
	if _, err := service.ScanGlobalSkills(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := service.IgnoreSkillCandidatesByName(ctx, domain.SkillIgnoreCandidatesInput{Name: "ignored-tool"}); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	result := NewSkillLoadTool(service).Execute(ctx, json.RawMessage(`{"mode":"activate","names":["ignored-tool"]}`), domain.ToolExecutionContext{SessionID: session.ID, ToolCallID: "skill-call-ignored"})
	if !result.OK || !strings.Contains(result.Content, "no_match") {
		t.Fatalf("result = %+v, want ignored candidate omitted from the resolvable catalog", result)
	}
}

func TestSkillToolResolvesWithAuxiliarySelector(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()

	writeSkill(t, filepath.Join(home, ".claude", "skills", "pdf-workflow"), "pdf-workflow", "Create and inspect PDF documents", "Follow the PDF workflow.")
	writeSkill(t, filepath.Join(home, ".claude", "skills", "code-review"), "code-review", "Review source code", "Follow the code review workflow.")
	scan, err := service.ScanGlobalSkills(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range scan.Candidates {
		if _, err := service.ImportSkill(ctx, domain.SkillImportInput{CandidateID: candidate.ID}); err != nil {
			t.Fatal(err)
		}
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	called := false
	resolver := func(_ context.Context, request SkillResolveRequest) (SkillResolveDecision, error) {
		called = true
		if request.Intent != "create a PDF report" || len(request.Candidates) != 2 {
			t.Fatalf("resolve request = %+v", request)
		}
		return SkillResolveDecision{Names: []string{"pdf-workflow", "invented-skill"}, Reason: "PDF workflow matched"}, nil
	}
	result := NewSkillLoadTool(service, resolver).Execute(ctx, json.RawMessage(`{"mode":"discover","intent":"create a PDF report"}`), domain.ToolExecutionContext{SessionID: session.ID, ToolCallID: "skill-discover"})
	if !called || !result.OK || !strings.Contains(result.ModelContent, "<name>pdf-workflow</name>") || strings.Contains(result.ModelContent, "Follow the PDF workflow.") || strings.Contains(result.ModelContent, "invented-skill") {
		t.Fatalf("discover result = %+v", result)
	}
	active, err := service.GetSessionActiveSkills(ctx, session.ID)
	if err != nil || len(active.Skills) != 0 {
		t.Fatalf("discovery must not activate skills: %+v, err = %v", active, err)
	}
	activated := NewSkillLoadTool(service, resolver).Execute(ctx, json.RawMessage(`{"mode":"activate","names":["pdf-workflow"]}`), domain.ToolExecutionContext{SessionID: session.ID, ToolCallID: "skill-activate"})
	if !activated.OK || !strings.Contains(activated.ModelContent, "Follow the PDF workflow.") {
		t.Fatalf("activate result = %+v", activated)
	}
	active, err = service.GetSessionActiveSkills(ctx, session.ID)
	if err != nil || len(active.Skills) != 1 || active.Skills[0].Name != "pdf-workflow" {
		t.Fatalf("active skills = %+v, err = %v", active, err)
	}
}

func TestSkillScanDiscoversCodexSkillsAsCandidates(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()

	writeSkill(t, filepath.Join(home, ".codex", "skills", "codex-review"), "codex-review", "Review with Codex conventions", "Follow Codex conventions.")

	scan, err := service.ScanGlobalSkills(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if scan.Scanned != 1 || len(scan.Candidates) != 1 {
		t.Fatalf("scan = %+v, want one Codex candidate", scan)
	}
	candidate := scan.Candidates[0]
	if candidate.Source != domain.SkillSourceCodex || candidate.Scope != domain.SkillScopeGlobal || candidate.Status != domain.SkillCandidateStatusPending {
		t.Fatalf("candidate = %+v, want pending global Codex candidate", candidate)
	}
}

func TestSkillScanDiscoversProjectCodexSkillsAsCandidates(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()
	workspaceRoot := t.TempDir()

	writeSkill(t, filepath.Join(workspaceRoot, ".codex", "skills", "project-codex"), "project-codex", "Project Codex workflow", "Follow project Codex workflow.")

	scan, err := service.ScanProjectSkills(ctx, domain.SkillScanInput{WorkspaceRoot: workspaceRoot})
	if err != nil {
		t.Fatal(err)
	}
	if scan.Scanned != 1 || len(scan.Candidates) != 1 {
		t.Fatalf("scan = %+v, want one project Codex candidate", scan)
	}
	candidate := scan.Candidates[0]
	if candidate.Source != domain.SkillSourceCodex || candidate.Scope != domain.SkillScopeProject || candidate.Status != domain.SkillCandidateStatusPending {
		t.Fatalf("candidate = %+v, want pending project Codex candidate", candidate)
	}
}

func TestParseSkillRejectsInvalidFrontmatter(t *testing.T) {
	root := filepath.Join(t.TempDir(), "bad-skill")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "SKILL.md"), []byte("# Missing frontmatter\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := parseSkillDirectory(root); err == nil {
		t.Fatal("parseSkillDirectory should reject missing frontmatter")
	}
}

func TestParseSkillMarkdownFoldedDescription(t *testing.T) {
	raw := "---\nname: folded-description\ndescription: >\n  Turn an idea into a goal package.\n  Build a reviewed execution plan.\nmetadata: test\n---\n\n# Instructions\n"
	name, description, metadata, _, err := parseSkillMarkdown(raw)
	if err != nil {
		t.Fatal(err)
	}
	if name != "folded-description" {
		t.Fatalf("name = %q", name)
	}
	if description != "Turn an idea into a goal package. Build a reviewed execution plan." {
		t.Fatalf("description = %q", description)
	}
	if metadata["metadata"] != "test" {
		t.Fatalf("metadata = %+v", metadata)
	}
}

func newSkillTestService(t *testing.T) (*Service, func()) {
	t.Helper()
	store, err := persistence.Open(filepath.Join(t.TempDir(), "aivo.db"))
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store)
	return service, func() { _ = store.Close() }
}

func writeSkill(t *testing.T, root string, name string, description string, body string) {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\n\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(root, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func contextSectionsText(sections []domain.ContextSection) string {
	var builder strings.Builder
	for _, section := range sections {
		builder.WriteString(section.Content)
		builder.WriteString("\n")
	}
	return builder.String()
}
