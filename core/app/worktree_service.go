package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"aivo/core/domain"
)

type gitWorktreeStore interface {
	SaveGitWorktree(context.Context, domain.GitWorktree) error
	GetGitWorktree(context.Context, string) (domain.GitWorktree, error)
	ListGitWorktrees(context.Context, string, bool) ([]domain.GitWorktree, error)
	UpdateGitWorktreeStatus(context.Context, string, string, string, string, string) (domain.GitWorktree, error)
	ActiveSessionIDsForProjectPath(context.Context, string) ([]string, error)
}

var worktreePathPartPattern = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func (s *Service) CreateGitWorktree(ctx context.Context, input domain.CreateGitWorktreeInput) (domain.GitWorktree, error) {
	store, err := s.gitWorktreeStore()
	if err != nil {
		return domain.GitWorktree{}, err
	}
	repositoryRoot, err := resolveGitRepositoryRoot(ctx, input.RepositoryPath)
	if err != nil {
		return domain.GitWorktree{}, err
	}
	if strings.TrimSpace(input.StartupCommand) != "" && !input.StartupConfirmed {
		return domain.GitWorktree{}, errors.New("worktree startup command requires explicit confirmation")
	}
	branch := strings.TrimSpace(input.Branch)
	if input.Detached && branch != "" {
		return domain.GitWorktree{}, errors.New("detached worktree cannot specify a branch")
	}
	if !input.Detached && branch == "" {
		branch = nextAutomaticWorktreeBranch(ctx, repositoryRoot, input.Name)
	}
	if branch != "" {
		if _, err := runGitArgs(ctx, repositoryRoot, "check-ref-format", "--branch", branch); err != nil {
			return domain.GitWorktree{}, fmt.Errorf("invalid worktree branch: %w", err)
		}
	}
	pathName := branch
	if pathName == "" {
		pathName = firstNonEmpty(strings.TrimSpace(input.Name), "detached-"+strings.ReplaceAll(uuid.NewString()[:8], "-", ""))
	}
	approvedRoot, targetPath, err := resolveManagedWorktreePath(repositoryRoot, input.ApprovedRoot, input.Path, pathName)
	if err != nil {
		return domain.GitWorktree{}, err
	}
	if info, statErr := os.Lstat(targetPath); statErr == nil {
		return domain.GitWorktree{}, fmt.Errorf("worktree path already exists: %s (%s)", targetPath, info.Mode())
	} else if !os.IsNotExist(statErr) {
		return domain.GitWorktree{}, statErr
	}
	if err := os.MkdirAll(approvedRoot, 0o700); err != nil {
		return domain.GitWorktree{}, fmt.Errorf("create approved worktree root: %w", err)
	}
	baseRef := strings.TrimSpace(input.BaseRef)
	if baseRef == "" {
		baseRef = defaultWorktreeBaseRef(ctx, repositoryRoot)
	}
	args := []string{"worktree", "add"}
	branchExists := branch != "" && gitBranchExists(ctx, repositoryRoot, branch)
	if input.Detached {
		args = append(args, "--detach", targetPath, baseRef)
	} else if branchExists {
		args = append(args, targetPath, branch)
	} else {
		args = append(args, "-b", branch, targetPath, baseRef)
	}
	if _, err := runGitArgs(ctx, repositoryRoot, args...); err != nil {
		return domain.GitWorktree{}, fmt.Errorf("create git worktree: %w", err)
	}
	now := domain.NowString(s.now())
	worktree := domain.GitWorktree{
		ID: uuid.NewString(), RepositoryRoot: repositoryRoot, Path: targetPath, Branch: branch, BaseRef: baseRef,
		Head: strings.TrimSpace(gitOutput(ctx, targetPath, "rev-parse", "HEAD")), Status: domain.WorktreeStatusReady,
		Managed: true, OwnsBranch: branch != "" && !branchExists, Detached: input.Detached, Dirty: gitWorktreeDirty(ctx, targetPath), TimeCreated: now, TimeUpdated: now,
	}
	if command := strings.TrimSpace(input.StartupCommand); command != "" {
		startupCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		output, startupErr := runWorktreeStartupCommand(startupCtx, targetPath, command)
		cancel()
		if startupErr != nil {
			worktree.Status = domain.WorktreeStatusError
			worktree.Error = "startup command: " + bounded(firstNonEmpty(strings.TrimSpace(output), startupErr.Error()), 4000)
		}
	}
	if err := store.SaveGitWorktree(ctx, worktree); err != nil {
		_, _ = runGitArgs(ctx, repositoryRoot, "worktree", "remove", "--force", targetPath)
		return domain.GitWorktree{}, err
	}
	if strings.TrimSpace(input.SessionID) != "" {
		if _, err := s.CreateOrUpdateCodingContext(ctx, input.SessionID, targetPath); err != nil {
			return domain.GitWorktree{}, fmt.Errorf("bind session to worktree: %w", err)
		}
	}
	return worktree, nil
}

func (s *Service) ListGitWorktrees(ctx context.Context, input domain.ListGitWorktreesInput) ([]domain.GitWorktree, error) {
	store, err := s.gitWorktreeStore()
	if err != nil {
		return nil, err
	}
	repositoryRoot := ""
	if strings.TrimSpace(input.RepositoryPath) != "" {
		repositoryRoot, err = resolveGitRepositoryRoot(ctx, input.RepositoryPath)
		if err != nil {
			return nil, err
		}
	}
	items, err := store.ListGitWorktrees(ctx, repositoryRoot, input.IncludeRemoved)
	if err != nil {
		return nil, err
	}
	for index := range items {
		items[index], _ = s.refreshGitWorktree(ctx, store, items[index])
	}
	if repositoryRoot != "" && !input.IncludeRemoved {
		discovered, discoveryErr := discoverGitWorktrees(ctx, repositoryRoot, store)
		if discoveryErr != nil {
			return nil, discoveryErr
		}
		known := make(map[string]bool, len(items))
		for _, item := range items {
			known[filepath.Clean(item.Path)] = true
		}
		for _, item := range discovered {
			if !known[filepath.Clean(item.Path)] {
				items = append(items, item)
			}
		}
	}
	return items, nil
}

func (s *Service) ResetGitWorktree(ctx context.Context, input domain.ResetGitWorktreeInput) (domain.GitWorktree, error) {
	if !input.Confirmed {
		return domain.GitWorktree{}, errors.New("worktree reset requires explicit confirmation")
	}
	store, err := s.gitWorktreeStore()
	if err != nil {
		return domain.GitWorktree{}, err
	}
	worktree, err := s.validManagedWorktree(ctx, store, input.WorktreeID)
	if err != nil {
		return domain.GitWorktree{}, err
	}
	if err := ensureWorktreeInactive(ctx, store, &worktree); err != nil {
		return domain.GitWorktree{}, err
	}
	target := strings.TrimSpace(input.TargetRef)
	if target == "" {
		target = firstNonEmpty(strings.TrimSpace(worktree.BaseRef), defaultWorktreeBaseRef(ctx, worktree.RepositoryRoot))
	}
	if _, err := runGitArgs(ctx, worktree.Path, "reset", "--hard", target); err != nil {
		return domain.GitWorktree{}, fmt.Errorf("reset git worktree: %w", err)
	}
	if input.Clean {
		if _, err := runGitArgs(ctx, worktree.Path, "clean", "-ffdx"); err != nil {
			return domain.GitWorktree{}, fmt.Errorf("clean git worktree: %w", err)
		}
	}
	if _, err := runGitArgs(ctx, worktree.Path, "submodule", "sync", "--recursive"); err != nil {
		return domain.GitWorktree{}, fmt.Errorf("sync worktree submodules: %w", err)
	}
	if _, err := runGitArgs(ctx, worktree.Path, "submodule", "update", "--init", "--recursive", "--force"); err != nil {
		return domain.GitWorktree{}, fmt.Errorf("restore worktree submodules: %w", err)
	}
	if _, err := runGitArgs(ctx, worktree.Path, "submodule", "foreach", "--recursive", "git reset --hard && git clean -ffdx"); err != nil {
		return domain.GitWorktree{}, fmt.Errorf("clean worktree submodules: %w", err)
	}
	worktree.Error = ""
	worktree.Status = domain.WorktreeStatusReady
	return s.refreshGitWorktree(ctx, store, worktree)
}

func (s *Service) RemoveGitWorktree(ctx context.Context, input domain.RemoveGitWorktreeInput) (domain.GitWorktree, error) {
	if !input.Confirmed {
		return domain.GitWorktree{}, errors.New("worktree removal requires explicit confirmation")
	}
	store, err := s.gitWorktreeStore()
	if err != nil {
		return domain.GitWorktree{}, err
	}
	worktree, err := s.validManagedWorktree(ctx, store, input.WorktreeID)
	if err != nil {
		return domain.GitWorktree{}, err
	}
	if err := ensureWorktreeInactive(ctx, store, &worktree); err != nil {
		return domain.GitWorktree{}, err
	}
	args := []string{"worktree", "remove"}
	if input.Force {
		args = append(args, "--force")
	}
	args = append(args, worktree.Path)
	if _, err := runGitArgs(ctx, worktree.RepositoryRoot, args...); err != nil {
		return domain.GitWorktree{}, fmt.Errorf("remove git worktree: %w", err)
	}
	removed, err := store.UpdateGitWorktreeStatus(ctx, worktree.ID, domain.WorktreeStatusRemoved, worktree.Head, worktree.Branch, "")
	if err != nil {
		return domain.GitWorktree{}, err
	}
	deleteBranch := worktree.OwnsBranch
	if input.DeleteBranch != nil {
		deleteBranch = *input.DeleteBranch
	}
	if deleteBranch && strings.TrimSpace(worktree.Branch) != "" {
		if _, branchErr := runGitArgs(ctx, worktree.RepositoryRoot, "branch", "-D", worktree.Branch); branchErr != nil {
			return removed, fmt.Errorf("worktree removed but delete branch: %w", branchErr)
		}
	}
	return removed, nil
}

func (s *Service) BindSessionToGitWorktree(ctx context.Context, input domain.BindSessionGitWorktreeInput) (domain.CodingContext, error) {
	if strings.TrimSpace(input.SessionID) == "" {
		return domain.CodingContext{}, errors.New("sessionId is required")
	}
	store, err := s.gitWorktreeStore()
	if err != nil {
		return domain.CodingContext{}, err
	}
	worktree, err := s.validManagedWorktree(ctx, store, input.WorktreeID)
	if err != nil {
		return domain.CodingContext{}, err
	}
	return s.CreateOrUpdateCodingContext(ctx, input.SessionID, worktree.Path)
}

func (s *Service) gitWorktreeStore() (gitWorktreeStore, error) {
	store, ok := s.store.(gitWorktreeStore)
	if !ok {
		return nil, errors.New("git worktree persistence is unavailable")
	}
	return store, nil
}

func (s *Service) validManagedWorktree(ctx context.Context, store gitWorktreeStore, id string) (domain.GitWorktree, error) {
	worktree, err := store.GetGitWorktree(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.GitWorktree{}, err
	}
	if !worktree.Managed || worktree.Status == domain.WorktreeStatusRemoved {
		return domain.GitWorktree{}, errors.New("worktree is not an active Aivo-managed worktree")
	}
	root, err := resolveGitRepositoryRoot(ctx, worktree.RepositoryRoot)
	if err != nil || root != filepath.Clean(worktree.RepositoryRoot) {
		return domain.GitWorktree{}, errors.New("worktree repository ownership is invalid")
	}
	common, err := runGitArgs(ctx, worktree.Path, "rev-parse", "--git-common-dir")
	if err != nil {
		return domain.GitWorktree{}, errors.New("worktree path is no longer registered with Git")
	}
	commonPath := strings.TrimSpace(common)
	if !filepath.IsAbs(commonPath) {
		commonPath = filepath.Join(worktree.Path, commonPath)
	}
	expected := filepath.Join(worktree.RepositoryRoot, ".git")
	resolvedCommon, _ := filepath.Abs(commonPath)
	resolvedExpected, _ := filepath.Abs(expected)
	if filepath.Clean(resolvedCommon) != filepath.Clean(resolvedExpected) {
		return domain.GitWorktree{}, errors.New("worktree belongs to a different repository")
	}
	return worktree, nil
}

func (s *Service) refreshGitWorktree(ctx context.Context, store gitWorktreeStore, worktree domain.GitWorktree) (domain.GitWorktree, error) {
	if worktree.Status == domain.WorktreeStatusRemoved {
		return worktree, nil
	}
	if _, err := os.Stat(worktree.Path); err != nil {
		status := domain.WorktreeStatusMissing
		updated, updateErr := store.UpdateGitWorktreeStatus(ctx, worktree.ID, status, worktree.Head, worktree.Branch, err.Error())
		return updated, updateErr
	}
	head, headErr := runGitArgs(ctx, worktree.Path, "rev-parse", "HEAD")
	branch, branchErr := runGitArgs(ctx, worktree.Path, "branch", "--show-current")
	if headErr != nil || branchErr != nil {
		updated, updateErr := store.UpdateGitWorktreeStatus(ctx, worktree.ID, domain.WorktreeStatusError, "", "", firstNonEmpty(errorText(headErr), errorText(branchErr)))
		return updated, updateErr
	}
	status, errText := domain.WorktreeStatusReady, ""
	if strings.HasPrefix(worktree.Error, "startup command:") {
		status, errText = domain.WorktreeStatusError, worktree.Error
	}
	updated, err := store.UpdateGitWorktreeStatus(ctx, worktree.ID, status, strings.TrimSpace(head), strings.TrimSpace(branch), errText)
	updated.Dirty = gitWorktreeDirty(ctx, worktree.Path)
	updated.ActiveSessions, _ = store.ActiveSessionIDsForProjectPath(ctx, worktree.Path)
	return updated, err
}

func ensureWorktreeInactive(ctx context.Context, store gitWorktreeStore, worktree *domain.GitWorktree) error {
	ids, err := store.ActiveSessionIDsForProjectPath(ctx, worktree.Path)
	if err != nil {
		return err
	}
	worktree.ActiveSessions = ids
	if len(ids) > 0 {
		return fmt.Errorf("worktree has active session execution: %s", strings.Join(ids, ", "))
	}
	return nil
}

func resolveGitRepositoryRoot(ctx context.Context, path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("repositoryPath is required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	root, err := runGitArgs(ctx, absolute, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", errors.New("path is not a Git repository")
	}
	return filepath.Clean(strings.TrimSpace(root)), nil
}

func resolveManagedWorktreePath(repositoryRoot string, approvedRootInput string, targetInput string, branch string) (string, string, error) {
	approvedRoot := strings.TrimSpace(approvedRootInput)
	if approvedRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", err
		}
		repoName := worktreePathPartPattern.ReplaceAllString(filepath.Base(repositoryRoot), "-")
		approvedRoot = filepath.Join(home, ".aivo", "worktrees", repoName)
	}
	approvedRoot, err := filepath.Abs(approvedRoot)
	if err != nil {
		return "", "", err
	}
	approvedRoot = canonicalizePotentialPath(approvedRoot)
	target := strings.TrimSpace(targetInput)
	if target == "" {
		target = filepath.Join(approvedRoot, worktreePathPartPattern.ReplaceAllString(branch, "-"))
	} else if !filepath.IsAbs(target) {
		target = filepath.Join(approvedRoot, target)
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return "", "", err
	}
	target = canonicalizePotentialPath(target)
	if target == approvedRoot || !pathWithinRoot(approvedRoot, target) {
		return "", "", errors.New("worktree path must be below the approved root")
	}
	if pathWithinRoot(repositoryRoot, target) {
		return "", "", errors.New("worktree path must not be inside the source repository")
	}
	return filepath.Clean(approvedRoot), filepath.Clean(target), nil
}

func canonicalizePotentialPath(path string) string {
	path = filepath.Clean(path)
	current := path
	var remainder []string
	for {
		if resolved, err := filepath.EvalSymlinks(current); err == nil {
			for index := len(remainder) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, remainder[index])
			}
			return filepath.Clean(resolved)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return path
		}
		remainder = append(remainder, filepath.Base(current))
		current = parent
	}
}

func gitBranchExists(ctx context.Context, repositoryRoot string, branch string) bool {
	_, err := runGitArgs(ctx, repositoryRoot, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	return err == nil
}

func nextAutomaticWorktreeBranch(ctx context.Context, repositoryRoot string, requestedName string) string {
	name := strings.Trim(worktreePathPartPattern.ReplaceAllString(strings.ToLower(strings.TrimSpace(requestedName)), "-"), "-._")
	if name == "" {
		name = "task"
	}
	for attempt := 0; attempt < 100; attempt++ {
		suffix := uuid.NewString()[:8]
		candidate := "aivo/" + name + "-" + suffix
		if !gitBranchExists(ctx, repositoryRoot, candidate) {
			return candidate
		}
	}
	return "aivo/task-" + uuid.NewString()
}

func defaultWorktreeBaseRef(ctx context.Context, repositoryRoot string) string {
	if remote := strings.TrimSpace(gitOutput(ctx, repositoryRoot, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD")); remote != "" {
		return remote
	}
	if branch := strings.TrimSpace(gitOutput(ctx, repositoryRoot, "branch", "--show-current")); branch != "" {
		return branch
	}
	return "HEAD"
}

func runWorktreeStartupCommand(ctx context.Context, cwd string, command string) (string, error) {
	process := exec.CommandContext(ctx, "/bin/sh", "-lc", command)
	process.Dir = cwd
	process.Env = append(os.Environ(), "AIVO_WORKTREE=1")
	output, err := process.CombinedOutput()
	return bounded(string(output), 4000), err
}

func discoverGitWorktrees(ctx context.Context, repositoryRoot string, store gitWorktreeStore) ([]domain.GitWorktree, error) {
	output, err := runGitArgs(ctx, repositoryRoot, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("list git worktrees: %w", err)
	}
	now := domain.NowString(time.Now())
	var result []domain.GitWorktree
	var current domain.GitWorktree
	flush := func() {
		if strings.TrimSpace(current.Path) == "" {
			return
		}
		sum := sha256.Sum256([]byte(filepath.Clean(current.Path)))
		current.ID = "git:" + hex.EncodeToString(sum[:8])
		current.RepositoryRoot = repositoryRoot
		if current.Status == "" {
			current.Status = domain.WorktreeStatusReady
		}
		current.Managed = false
		current.Dirty = gitWorktreeDirty(ctx, current.Path)
		current.ActiveSessions, _ = store.ActiveSessionIDsForProjectPath(ctx, current.Path)
		current.TimeCreated, current.TimeUpdated = now, now
		result = append(result, current)
		current = domain.GitWorktree{}
	}
	for _, line := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			flush()
			continue
		}
		key, value, _ := strings.Cut(line, " ")
		switch key {
		case "worktree":
			current.Path = filepath.Clean(strings.TrimSpace(value))
		case "HEAD":
			current.Head = strings.TrimSpace(value)
		case "branch":
			current.Branch = strings.TrimPrefix(strings.TrimSpace(value), "refs/heads/")
		case "detached":
			current.Detached = true
		case "prunable":
			current.Status = domain.WorktreeStatusMissing
			current.Error = strings.TrimSpace(value)
		}
	}
	flush()
	return result, nil
}

func gitWorktreeDirty(ctx context.Context, path string) bool {
	output, err := runGitArgs(ctx, path, "status", "--porcelain")
	return err == nil && strings.TrimSpace(output) != ""
}

func runGitArgs(ctx context.Context, cwd string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", append([]string{"-C", cwd}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w", bounded(strings.TrimSpace(string(output)), 2000), err)
	}
	return string(output), nil
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
