package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestGitWorktreeLifecycleAndActiveSessionGuard(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	t.Setenv("HOME", t.TempDir())
	ctx := context.Background()
	repository := initGitWorktreeTestRepository(t)
	approvedRoot := filepath.Join(t.TempDir(), "managed")
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: repository})
	if err != nil {
		t.Fatal(err)
	}
	worktree, err := service.CreateGitWorktree(ctx, domain.CreateGitWorktreeInput{
		RepositoryPath: repository, ApprovedRoot: approvedRoot, Branch: "feature/worktree", SessionID: session.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if worktree.Status != domain.WorktreeStatusReady || !pathWithinRoot(canonicalizePotentialPath(approvedRoot), worktree.Path) {
		t.Fatalf("worktree = %#v", worktree)
	}
	items, err := service.ListGitWorktrees(ctx, domain.ListGitWorktreesInput{RepositoryPath: repository})
	if err != nil || len(items) < 2 {
		t.Fatalf("worktrees = %#v err = %v", items, err)
	}
	foundManaged, foundRepository := false, false
	for _, item := range items {
		foundManaged = foundManaged || item.ID == worktree.ID
		foundRepository = foundRepository || (!item.Managed && canonicalizePotentialPath(item.Path) == canonicalizePotentialPath(repository))
	}
	if !foundManaged || !foundRepository {
		t.Fatalf("managed/discovered worktrees = %#v", items)
	}
	if _, err := service.store.UpsertSessionExecutionState(ctx, domain.SessionExecutionState{SessionID: session.ID, Status: domain.ExecutionStatusRunning}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RemoveGitWorktree(ctx, domain.RemoveGitWorktreeInput{WorktreeID: worktree.ID, Confirmed: true}); err == nil || !strings.Contains(err.Error(), session.ID) {
		t.Fatalf("active removal error = %v", err)
	}
	if _, err := service.store.UpsertSessionExecutionState(ctx, domain.SessionExecutionState{SessionID: session.ID, Status: domain.ExecutionStatusIdle}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worktree.Path, "README.md"), []byte("changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	reset, err := service.ResetGitWorktree(ctx, domain.ResetGitWorktreeInput{WorktreeID: worktree.ID, Confirmed: true})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(worktree.Path, "README.md"))
	if err != nil || string(raw) != "initial\n" || reset.Dirty {
		t.Fatalf("reset content = %q dirty = %v err = %v", raw, reset.Dirty, err)
	}
	removed, err := service.RemoveGitWorktree(ctx, domain.RemoveGitWorktreeInput{WorktreeID: worktree.ID, Confirmed: true})
	if err != nil || removed.Status != domain.WorktreeStatusRemoved {
		t.Fatalf("removed = %#v err = %v", removed, err)
	}
	if _, err := os.Stat(worktree.Path); !os.IsNotExist(err) {
		t.Fatalf("removed worktree path still exists: %v", err)
	}
	reused, err := service.CreateGitWorktree(ctx, domain.CreateGitWorktreeInput{
		RepositoryPath: repository, ApprovedRoot: approvedRoot, Path: "reused-branch", Branch: "feature/worktree",
	})
	if err != nil {
		t.Fatalf("create worktree from existing branch: %v", err)
	}
	if _, err := service.RemoveGitWorktree(ctx, domain.RemoveGitWorktreeInput{WorktreeID: reused.ID, Confirmed: true}); err != nil {
		t.Fatal(err)
	}
}

func TestCreateGitWorktreeSupportsAutomaticBranchDetachedAndStartupCommand(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	t.Setenv("HOME", t.TempDir())
	ctx := context.Background()
	repository := initGitWorktreeTestRepository(t)
	approvedRoot := filepath.Join(t.TempDir(), "managed")
	automatic, err := service.CreateGitWorktree(ctx, domain.CreateGitWorktreeInput{
		RepositoryPath: repository, ApprovedRoot: approvedRoot, Name: "Release Check",
		StartupCommand: "printf initialized > .aivo-started", StartupConfirmed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(automatic.Branch, "aivo/release-check-") || !automatic.OwnsBranch || automatic.Detached {
		t.Fatalf("automatic worktree = %#v", automatic)
	}
	if raw, err := os.ReadFile(filepath.Join(automatic.Path, ".aivo-started")); err != nil || string(raw) != "initialized" {
		t.Fatalf("startup output = %q err = %v", raw, err)
	}
	if err := os.WriteFile(filepath.Join(automatic.Path, ".gitignore"), []byte("cache/\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(automatic.Path, "cache"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(automatic.Path, "cache", "ignored"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ResetGitWorktree(ctx, domain.ResetGitWorktreeInput{WorktreeID: automatic.ID, Clean: true, Confirmed: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(automatic.Path, "cache")); !os.IsNotExist(err) {
		t.Fatalf("ignored content survived reset: %v", err)
	}
	branch := automatic.Branch
	if _, err := service.RemoveGitWorktree(ctx, domain.RemoveGitWorktreeInput{WorktreeID: automatic.ID, Force: true, Confirmed: true}); err != nil {
		t.Fatal(err)
	}
	if gitBranchExists(ctx, repository, branch) {
		t.Fatalf("owned branch %q survived removal", branch)
	}

	detached, err := service.CreateGitWorktree(ctx, domain.CreateGitWorktreeInput{RepositoryPath: repository, ApprovedRoot: approvedRoot, Name: "inspection", Detached: true})
	if err != nil {
		t.Fatal(err)
	}
	if !detached.Detached || detached.Branch != "" || detached.OwnsBranch {
		t.Fatalf("detached worktree = %#v", detached)
	}
	if _, err := service.RemoveGitWorktree(ctx, domain.RemoveGitWorktreeInput{WorktreeID: detached.ID, Force: true, Confirmed: true}); err != nil {
		t.Fatal(err)
	}
}

func TestCreateGitWorktreeRequiresStartupConfirmationAndPreservesFailure(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	t.Setenv("HOME", t.TempDir())
	ctx := context.Background()
	repository := initGitWorktreeTestRepository(t)
	approvedRoot := filepath.Join(t.TempDir(), "managed")
	if _, err := service.CreateGitWorktree(ctx, domain.CreateGitWorktreeInput{RepositoryPath: repository, ApprovedRoot: approvedRoot, StartupCommand: "exit 1"}); err == nil || !strings.Contains(err.Error(), "confirmation") {
		t.Fatalf("startup confirmation error = %v", err)
	}
	failed, err := service.CreateGitWorktree(ctx, domain.CreateGitWorktreeInput{RepositoryPath: repository, ApprovedRoot: approvedRoot, StartupCommand: "echo failed && exit 7", StartupConfirmed: true})
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != domain.WorktreeStatusError || !strings.Contains(failed.Error, "failed") {
		t.Fatalf("failed startup worktree = %#v", failed)
	}
	items, err := service.ListGitWorktrees(ctx, domain.ListGitWorktreesInput{RepositoryPath: repository})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range items {
		if item.ID == failed.ID {
			found = item.Status == domain.WorktreeStatusError && strings.Contains(item.Error, "startup command")
		}
	}
	if !found {
		t.Fatalf("startup failure was not retained: %#v", items)
	}
	if _, err := service.RemoveGitWorktree(ctx, domain.RemoveGitWorktreeInput{WorktreeID: failed.ID, Force: true, Confirmed: true}); err != nil {
		t.Fatal(err)
	}
}

func TestCreateGitWorktreeRejectsNonGitUnsafePathAndMissingConfirmation(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	t.Setenv("HOME", t.TempDir())
	ctx := context.Background()
	if _, err := service.CreateGitWorktree(ctx, domain.CreateGitWorktreeInput{RepositoryPath: t.TempDir(), Branch: "feature"}); err == nil {
		t.Fatal("non-Git worktree creation succeeded")
	}
	repository := initGitWorktreeTestRepository(t)
	approvedRoot := filepath.Join(t.TempDir(), "managed")
	if _, err := service.CreateGitWorktree(ctx, domain.CreateGitWorktreeInput{
		RepositoryPath: repository, ApprovedRoot: approvedRoot, Path: "../escape", Branch: "unsafe",
	}); err == nil {
		t.Fatal("unsafe worktree path succeeded")
	}
	worktree, err := service.CreateGitWorktree(ctx, domain.CreateGitWorktreeInput{RepositoryPath: repository, ApprovedRoot: approvedRoot, Branch: "safe"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ResetGitWorktree(ctx, domain.ResetGitWorktreeInput{WorktreeID: worktree.ID}); err == nil || !strings.Contains(err.Error(), "confirmation") {
		t.Fatalf("reset confirmation error = %v", err)
	}
	if _, err := service.RemoveGitWorktree(ctx, domain.RemoveGitWorktreeInput{WorktreeID: worktree.ID}); err == nil || !strings.Contains(err.Error(), "confirmation") {
		t.Fatalf("remove confirmation error = %v", err)
	}
	if _, err := service.RemoveGitWorktree(ctx, domain.RemoveGitWorktreeInput{WorktreeID: worktree.ID, Confirmed: true}); err != nil {
		t.Fatal(err)
	}
}

func initGitWorktreeTestRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runGitWorktreeTestCommand(t, root, "init")
	runGitWorktreeTestCommand(t, root, "config", "user.email", "aivo@example.test")
	runGitWorktreeTestCommand(t, root, "config", "user.name", "Aivo Test")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("initial\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGitWorktreeTestCommand(t, root, "add", "README.md")
	runGitWorktreeTestCommand(t, root, "commit", "-m", "initial")
	return root
}

func runGitWorktreeTestCommand(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}
