package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	if err := os.MkdirAll(filepath.Join(sourceRoot, "references"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "references", "guide.md"), []byte("guide"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(sourceRoot, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "scripts", "check.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(sourceRoot, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "assets", "sample.txt"), []byte("asset"), 0o644); err != nil {
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
	for _, rel := range []string{"reference.md", filepath.Join("references", "guide.md"), filepath.Join("scripts", "check.sh"), filepath.Join("assets", "sample.txt")} {
		if _, err := os.Stat(filepath.Join(home, ".aivo", "skills", "code-review", rel)); err != nil {
			t.Fatalf("managed skill copy missing %s: %v", rel, err)
		}
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
	if !strings.Contains(joined, "<available_skills>") || !strings.Contains(joined, "<name>code-review</name>") {
		t.Fatalf("context missing explicit visible skill catalog: %q", joined)
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
		t.Fatal("legacy skill Host control tool is still registered")
	}
	if _, ok := registry.Get(SkillsListToolName); !ok {
		t.Fatal("skills_list package control tool should be registered for filtered Skill catalogs")
	}
	if _, ok := registry.Get(SkillsReadToolName); !ok {
		t.Fatal("skills_read package control tool should be registered for filtered Skill catalogs")
	}
	assembly := AssembleToolSpecsWithSources(registry, registry.SpecsForToolsets([]string{"safe", "coding"}), nil)
	for _, spec := range assembly.Specs {
		if spec.Name == SkillsListToolName || spec.Name == SkillsReadToolName {
			t.Fatalf("%s should not be visible before Host filters a Skill catalog", spec.Name)
		}
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
	if len(before.Candidates) != 0 {
		t.Fatalf("list before scan = %+v, want no scanned candidates", before)
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

func TestSkillListOmitsCandidatesDeletedFromDisk(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()

	sourceRoot := filepath.Join(home, ".agents", "skills", "deleted-candidate")
	writeSkill(t, sourceRoot, "deleted-candidate", "Deleted candidate", "Use this skill.")
	if _, err := service.ScanGlobalSkills(ctx); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(sourceRoot); err != nil {
		t.Fatal(err)
	}

	list, err := service.ListSkills(ctx, domain.SkillListInput{IncludeCandidates: true, IncludeIgnored: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Candidates) != 0 {
		t.Fatalf("candidates = %+v, want deleted source omitted", list.Candidates)
	}
}

func TestSkillListOmitsManagedSkillsDeletedFromDisk(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()

	root := filepath.Join(home, ".aivo", "skills", "deleted-managed")
	writeSkill(t, root, "deleted-managed", "Deleted managed", "Use this skill.")
	if _, err := service.ScanGlobalSkills(ctx); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}

	list, err := service.ListSkills(ctx, domain.SkillListInput{IncludeDisabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := skillEntryByName(list.Entries, "deleted-managed"); ok {
		t.Fatalf("entries = %+v, want deleted managed skill omitted", list.Entries)
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
	editable, ok := skillEntryByName(list.Entries, "editable-skill")
	if err != nil || !ok {
		t.Fatalf("skills = %+v, err = %v", list.Entries, err)
	}

	editor, err := service.GetManagedSkillForEdit(ctx, editable.ID)
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
	stale, ok := skillEntryByName(list.Entries, "stale-skill")
	if err != nil || !ok {
		t.Fatalf("skills = %+v, err = %v", list.Entries, err)
	}
	editor, err := service.GetManagedSkillForEdit(ctx, stale.ID)
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

func TestSkillImportConflictDoesNotOverwriteManagedDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()

	managedRoot := filepath.Join(home, ".aivo", "skills", "dup-skill")
	writeSkill(t, managedRoot, "dup-skill", "Managed", "Original managed content.")
	if err := os.MkdirAll(filepath.Join(managedRoot, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(managedRoot, "assets", "keep.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ScanGlobalSkills(ctx); err != nil {
		t.Fatal(err)
	}
	list, err := service.ListSkills(ctx, domain.SkillListInput{IncludeDisabled: true})
	managed, ok := skillEntryByName(list.Entries, "dup-skill")
	if err != nil || !ok {
		t.Fatalf("managed skills = %+v, err = %v", list.Entries, err)
	}

	sourceRoot := filepath.Join(home, ".agents", "skills", "dup-skill")
	writeSkill(t, sourceRoot, "dup-skill", "External", "Different external content.")
	source, err := parseSkillDirectory(sourceRoot)
	if err != nil {
		t.Fatal(err)
	}
	manager := service.ensureSkillManager()
	candidate, err := manager.store.SaveSkillImportCandidate(ctx, domain.SkillImportCandidate{
		ID: "manual-conflict", Name: source.Name, Description: source.Description, Scope: domain.SkillScopeGlobal, Source: domain.SkillSourceAgents,
		RootPath: source.RootPath, SkillPath: source.SkillPath, ContentHash: source.ContentHash, Status: domain.SkillCandidateStatusPending, LastSeenAt: domain.NowString(time.Now()),
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.ImportSkill(ctx, domain.SkillImportInput{CandidateID: candidate.ID}); err == nil || !strings.Contains(err.Error(), "different content") {
		t.Fatalf("import error = %v, want same-name content conflict", err)
	}
	raw, err := os.ReadFile(filepath.Join(managedRoot, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "Original managed content.") || strings.Contains(string(raw), "Different external content.") {
		t.Fatalf("managed SKILL.md was overwritten: %q", string(raw))
	}
	if _, err := os.Stat(filepath.Join(managedRoot, "assets", "keep.txt")); err != nil {
		t.Fatalf("managed supporting file was removed: %v", err)
	}
	after, err := manager.store.GetSkillImportCandidate(ctx, candidate.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Status != domain.SkillCandidateStatusIgnored || after.ConflictID != managed.ID {
		t.Fatalf("candidate after conflict = %+v", after)
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

func TestSkillsReadAndListOnlyExposeImportedSkills(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()

	writeSkill(t, filepath.Join(home, ".claude", "skills", "tool-skill"), "tool-skill", "Tool import", "Loaded by tool.")
	writeSkill(t, filepath.Join(home, ".claude", "skills", "hidden-skill"), "hidden-skill", "Hidden import", "Hidden by filtered catalog.")
	if _, err := service.ScanGlobalSkills(ctx); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	listTool := NewSkillsListTool(service)
	readTool := NewSkillsReadTool(service)
	listed := listTool.Execute(ctx, json.RawMessage(`{}`), domain.ToolExecutionContext{SessionID: session.ID, ToolCallID: "skills-list"})
	if !listed.OK || strings.Contains(listed.ModelContent, "tool-skill") {
		t.Fatalf("list result = %+v, pending skill must not be visible to the model", listed)
	}
	blockedPending := readTool.Execute(ctx, json.RawMessage(`{"package":"skill://aivo/tool-skill"}`), domain.ToolExecutionContext{SessionID: session.ID, ToolCallID: "skills-read-pending"})
	if blockedPending.OK || !strings.Contains(blockedPending.Error, "skill package is not available") {
		t.Fatalf("pending read result = %+v, pending skill must not be readable by the model", blockedPending)
	}
	candidates, err := service.ListSkills(ctx, domain.SkillListInput{IncludeCandidates: true})
	if err != nil || len(candidates.Candidates) != 2 {
		t.Fatalf("candidates = %+v, err = %v", candidates, err)
	}
	var imported domain.SkillEntry
	for _, candidate := range candidates.Candidates {
		skill, err := service.ImportSkill(ctx, domain.SkillImportInput{CandidateID: candidate.ID})
		if err != nil {
			t.Fatal(err)
		}
		if skill.Name == "tool-skill" {
			imported = skill
		}
	}
	if imported.ID == "" {
		t.Fatalf("imported candidates did not include tool-skill: %+v", candidates.Candidates)
	}
	if _, err := service.rememberVisibleSkills(ctx, session.ID, []domain.SkillEntry{imported}); err != nil {
		t.Fatal(err)
	}
	listed = listTool.Execute(ctx, json.RawMessage(`{"authority":"orchestrator"}`), domain.ToolExecutionContext{SessionID: session.ID, ToolCallID: "skills-list-imported"})
	if !listed.OK || !strings.Contains(listed.ModelContent, `"name": "tool-skill"`) || !strings.Contains(listed.ModelContent, `"package": "skill://aivo/tool-skill"`) || strings.Contains(listed.ModelContent, "hidden-skill") || strings.Contains(listed.ModelContent, "Loaded by tool.") {
		t.Fatalf("list result = %+v, want imported catalog metadata without skill body", listed)
	}
	hiddenRead := readTool.Execute(ctx, json.RawMessage(`{"package":"skill://aivo/hidden-skill"}`), domain.ToolExecutionContext{SessionID: session.ID, ToolCallID: "skills-read-hidden"})
	if hiddenRead.OK || !strings.Contains(hiddenRead.Error, "skill package is not available") {
		t.Fatalf("hidden read result = %+v, want filtered-out Skill unreadable", hiddenRead)
	}
	read := readTool.Execute(ctx, json.RawMessage(`{"package":"skill://aivo/tool-skill"}`), domain.ToolExecutionContext{SessionID: session.ID, ToolCallID: "skills-read-1"})
	if !read.OK || !strings.Contains(read.ModelContent, "Loaded by tool.") || !strings.Contains(read.ModelContent, `"resource": "skill://aivo/tool-skill/SKILL.md"`) {
		t.Fatalf("read result = %+v, want imported SKILL.md contents", read)
	}
	supportingFiles := map[string]string{
		"references/guide.md":       "supporting guidance\n",
		"scripts/build.sh":          "#!/bin/sh\n",
		"assets/icon.txt":           "asset payload\n",
		"fixtures/nested/input.txt": "nested fixture\n",
		"notes.md":                  "top-level note\n",
	}
	for relative, contents := range supportingFiles {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(imported.RootPath, relative)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(imported.RootPath, relative), []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
		readResource := readTool.Execute(ctx, mustJSONRaw(t, map[string]any{
			"package":  "skill://aivo/tool-skill",
			"resource": "skill://aivo/tool-skill/" + filepath.ToSlash(relative),
		}), domain.ToolExecutionContext{SessionID: session.ID, ToolCallID: "skills-read-" + strings.ReplaceAll(filepath.ToSlash(relative), "/", "-")})
		if !readResource.OK || !strings.Contains(readResource.ModelContent, strings.TrimSpace(contents)) || !strings.Contains(readResource.ModelContent, `"resource": "skill://aivo/tool-skill/`+filepath.ToSlash(relative)+`"`) {
			t.Fatalf("resource read result for %s = %+v, want supporting file contents", relative, readResource)
		}
	}
	active, err := service.GetSessionActiveSkills(ctx, session.ID)
	if err != nil || len(active.Skills) != 0 {
		t.Fatalf("skills_read must not persist activation: %+v, err = %v", active, err)
	}
	list, err := service.ListSkills(ctx, domain.SkillListInput{IncludeCandidates: true, IncludeDisabled: true, IncludeIgnored: true})
	if err != nil {
		t.Fatal(err)
	}
	toolSkill, ok := skillEntryByName(list.Entries, "tool-skill")
	if !ok {
		t.Fatalf("entries = %+v, want imported tool-skill", list.Entries)
	}
	if len(list.Candidates) != 2 {
		t.Fatalf("candidates = %+v, want imported candidates", list.Candidates)
	}
	for _, candidate := range list.Candidates {
		if candidate.Status != domain.SkillCandidateStatusImported {
			t.Fatalf("candidates = %+v, want imported candidates", list.Candidates)
		}
	}
	if _, err := service.SetSkillEnabled(ctx, domain.SkillEnabledInput{SkillID: toolSkill.ID, Enabled: false}); err != nil {
		t.Fatal(err)
	}
	nextSession, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	blocked := readTool.Execute(ctx, json.RawMessage(`{"package":"skill://aivo/tool-skill"}`), domain.ToolExecutionContext{SessionID: nextSession.ID, ToolCallID: "skills-read-disabled"})
	if blocked.OK || !strings.Contains(blocked.Error, "skill package is not available") {
		t.Fatalf("blocked read = %+v, want disabled skill omitted from the readable catalog", blocked)
	}
	if _, err := service.SetSkillEnabled(ctx, domain.SkillEnabledInput{SkillID: toolSkill.ID, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.rememberVisibleSkills(ctx, nextSession.ID, []domain.SkillEntry{toolSkill}); err != nil {
		t.Fatal(err)
	}
	reloaded := readTool.Execute(ctx, json.RawMessage(`{"package":"skill://aivo/tool-skill"}`), domain.ToolExecutionContext{SessionID: nextSession.ID, ToolCallID: "skills-read-reenabled"})
	if !reloaded.OK {
		t.Fatalf("reenabled read result = %+v, want ok", reloaded)
	}
}

func TestSkillsReadDoesNotImportIgnoredCandidate(t *testing.T) {
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
	result := NewSkillsReadTool(service).Execute(ctx, json.RawMessage(`{"package":"skill://aivo/ignored-tool"}`), domain.ToolExecutionContext{SessionID: session.ID, ToolCallID: "skills-read-ignored"})
	if result.OK || !strings.Contains(result.Error, "skill package is not available") {
		t.Fatalf("result = %+v, want ignored candidate omitted from the readable catalog", result)
	}
}

func TestLocalSkillResolveReturnsOnlyCatalogMatches(t *testing.T) {
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
	candidates, err := service.skillResolveCandidates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	candidateNames := map[string]bool{}
	for _, candidate := range candidates {
		candidateNames[candidate.Name] = true
	}
	if !candidateNames["pdf-workflow"] || !candidateNames["code-review"] || !candidateNames["skill-creator"] || !candidateNames["skill-installer"] {
		t.Fatalf("resolve candidates = %+v", candidates)
	}
	decision, err := localSkillResolve(ctx, SkillResolveRequest{Intent: "create a PDF report", Candidates: candidates})
	if err != nil {
		t.Fatal(err)
	}
	names := validateSkillResolveSelection(candidates, append(decision.Names, "invented-skill"), 0)
	if len(names) == 0 || names[0] != "pdf-workflow" {
		t.Fatalf("resolved names = %#v from decision %#v", names, decision)
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
	raw := "---\nname: folded-description\ndescription: >\n  Turn an idea into a goal package.\n  Build a reviewed execution plan.\nlicense: MIT\ncompatibility: Requires git.\nallowed-tools: Bash(git:*) Read\nmetadata:\n  author: mattpocock\n  version: \"1.0\"\n---\n\n# Instructions\n"
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
	if metadata["license"] != "MIT" || metadata["compatibility"] != "Requires git." || metadata["allowed-tools"] != "Bash(git:*) Read" {
		t.Fatalf("top-level metadata = %+v", metadata)
	}
	if metadata["metadata.author"] != "mattpocock" || metadata["metadata.version"] != "1.0" {
		t.Fatalf("metadata = %+v", metadata)
	}
}

func TestMarshalSkillMarkdownPreservesNestedMetadata(t *testing.T) {
	raw := marshalSkillMarkdown("meta-skill", "Updated description", map[string]string{
		"license":          "MIT",
		"metadata.author":  "Aivo",
		"metadata.version": "1",
	}, "Use this workflow.")
	text := string(raw)
	if !strings.Contains(text, "metadata:\n  author: \"Aivo\"\n  version: \"1\"") {
		t.Fatalf("marshaled metadata = %q", text)
	}
	name, description, metadata, content, err := parseSkillMarkdown(text)
	if err != nil {
		t.Fatal(err)
	}
	if name != "meta-skill" || description != "Updated description" || content != "Use this workflow." {
		t.Fatalf("parsed = %q %q %q", name, description, content)
	}
	if metadata["license"] != "MIT" || metadata["metadata.author"] != "Aivo" || metadata["metadata.version"] != "1" {
		t.Fatalf("metadata = %+v", metadata)
	}
}

func TestParseSkillMarkdownFollowsAgentSkillsSpecBounds(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "uppercase name", raw: "---\nname: Bad-Skill\ndescription: Valid description.\n---\n", want: "skill name is invalid"},
		{name: "consecutive hyphen", raw: "---\nname: bad--skill\ndescription: Valid description.\n---\n", want: "skill name is invalid"},
		{name: "too long name", raw: "---\nname: " + strings.Repeat("a", 65) + "\ndescription: Valid description.\n---\n", want: "skill name is invalid"},
		{name: "too long description", raw: "---\nname: long-description\ndescription: " + strings.Repeat("a", 1025) + "\n---\n", want: "skill description exceeds 1024 bytes"},
		{name: "scalar metadata", raw: "---\nname: scalar-metadata\ndescription: Valid description.\nmetadata: no\n---\n", want: "skill metadata must be a string map"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, _, _, err := parseSkillMarkdown(test.raw); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("parse error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestParseSkillDirectoryRejectsSymlinkResources(t *testing.T) {
	root := filepath.Join(t.TempDir(), "linked-skill")
	writeSkill(t, root, "linked-skill", "Reject symlink resources", "Use this skill.")
	target := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(target, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "references.md")); err != nil {
		t.Skipf("symlink unsupported on this platform: %v", err)
	}
	if _, err := parseSkillDirectory(root); err == nil || !strings.Contains(err.Error(), "symlinks") {
		t.Fatalf("parse error = %v, want symlink refusal", err)
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

func skillEntryByName(entries []domain.SkillEntry, name string) (domain.SkillEntry, bool) {
	for _, entry := range entries {
		if entry.Name == name {
			return entry, true
		}
	}
	return domain.SkillEntry{}, false
}

func contextSectionsText(sections []domain.ContextSection) string {
	var builder strings.Builder
	for _, section := range sections {
		builder.WriteString(section.Content)
		builder.WriteString("\n")
	}
	return builder.String()
}
