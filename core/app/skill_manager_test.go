package app

import (
	"context"
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
	if !strings.Contains(joined, "<available_skills>") || !strings.Contains(joined, "<name>code-review</name>") {
		t.Fatalf("context missing available skill guidance: %q", joined)
	}
	if !strings.Contains(joined, `<skill_content name="code-review"`) || !strings.Contains(joined, "# Skill: code-review") || !strings.Contains(joined, "Use this skill to review code.") {
		t.Fatalf("context missing loaded skill: %q", joined)
	}
	if strings.Contains(joined, "---\nname: code-review") {
		t.Fatalf("context should not include skill frontmatter: %q", joined)
	}
	if !strings.Contains(joined, "Base directory for this skill: "+filepath.Join(home, ".aivo", "skills", "code-review")) {
		t.Fatalf("context missing OpenCode skill base directory line: %q", joined)
	}
	if !strings.Contains(joined, "<skill_files>") || !strings.Contains(joined, "<file>"+filepath.Join(home, ".aivo", "skills", "code-review", "reference.md")+"</file>") {
		t.Fatalf("context missing sampled skill files: %q", joined)
	}
	registry, _ := service.toolsForWorkspace(session.ProjectPath)
	if registry == nil {
		t.Fatal("registry is nil")
	}
	if _, ok := registry.Get("skill"); !ok {
		t.Fatal("model tool should be registered as skill")
	}
	if _, ok := registry.Get("skill_load"); ok {
		t.Fatal("model tool should not be registered as skill_load")
	}
}

func TestSkillScanDetectsSameNameConflict(t *testing.T) {
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
	if scan.Conflicts != 1 {
		t.Fatalf("conflicts = %d, want 1; scan=%+v", scan.Conflicts, scan)
	}
	var conflict domain.SkillImportCandidate
	for _, candidate := range scan.Candidates {
		if candidate.Status == domain.SkillCandidateStatusConflict {
			conflict = candidate
		}
	}
	if conflict.ID == "" || conflict.ConflictID == "" {
		t.Fatalf("missing conflict candidate: %+v", scan.Candidates)
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
