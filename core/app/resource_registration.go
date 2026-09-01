package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"aivo/core/domain"
)

const (
	resourceRegistrationProposalTTL = 10 * time.Minute
	resourceRegistrationMaxPending  = 64
	skillResourceMaxFiles           = 128
	skillResourceMaxFileBytes       = 1024 * 1024
	skillResourceMaxTotalBytes      = 4 * 1024 * 1024
	skillsCLIInstallTimeout         = 2 * time.Minute
	skillsCLIOutputMaxBytes         = 32 * 1024
)

type resourceRegistrationError struct {
	code string
	err  error
}

func (e *resourceRegistrationError) Error() string {
	if e == nil || e.err == nil {
		return "resource registration failed"
	}
	return e.err.Error()
}

func (e *resourceRegistrationError) Unwrap() error { return e.err }

func newResourceRegistrationError(code, message string) error {
	return &resourceRegistrationError{code: code, err: errors.New(message)}
}

func resourceRegistrationErrorCode(err error) string {
	var registrationErr *resourceRegistrationError
	if errors.As(err, &registrationErr) && registrationErr.code != "" {
		return registrationErr.code
	}
	if errors.Is(err, context.Canceled) {
		return "cancelled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	return "resource_registration_failed"
}

type resourceRegistrationProposal struct {
	ID         string
	SessionID  string
	TurnID     string
	ToolCallID string
	Input      domain.ResourceRegistrationProposalInput
	Hash       string
	CreatedAt  time.Time
	ExpiresAt  time.Time
}

type resourceRegistrationProposalStore struct {
	mu       sync.Mutex
	commitMu sync.Mutex
	byCall   map[string]resourceRegistrationProposal
	now      func() time.Time
	ttl      time.Duration
	limit    int
}

func newResourceRegistrationProposalStore() *resourceRegistrationProposalStore {
	return &resourceRegistrationProposalStore{byCall: map[string]resourceRegistrationProposal{}, now: time.Now, ttl: resourceRegistrationProposalTTL, limit: resourceRegistrationMaxPending}
}

func (s *resourceRegistrationProposalStore) prepare(input domain.ResourceRegistrationProposalInput, execCtx domain.ToolExecutionContext) (resourceRegistrationProposal, error) {
	if s == nil || strings.TrimSpace(execCtx.SessionID) == "" || strings.TrimSpace(execCtx.TurnID) == "" || strings.TrimSpace(execCtx.ToolCallID) == "" {
		return resourceRegistrationProposal{}, newResourceRegistrationError("invalid_proposal_owner", "resource registration proposal requires an owning session, turn, and tool call")
	}
	now := s.now()
	hash := resourceRegistrationInputHash(input)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked(now)
	if current, ok := s.byCall[execCtx.ToolCallID]; ok {
		if current.SessionID == execCtx.SessionID && current.TurnID == execCtx.TurnID && current.Hash == hash {
			return current, nil
		}
		return resourceRegistrationProposal{}, newResourceRegistrationError("proposal_conflict", "tool call already owns a different resource registration proposal")
	}
	if len(s.byCall) >= s.limit {
		s.evictOldestLocked()
	}
	proposal := resourceRegistrationProposal{
		ID: uuid.NewString(), SessionID: execCtx.SessionID, TurnID: execCtx.TurnID, ToolCallID: execCtx.ToolCallID,
		Input: input, Hash: hash, CreatedAt: now, ExpiresAt: now.Add(s.ttl),
	}
	s.byCall[proposal.ToolCallID] = proposal
	return proposal, nil
}

func (s *resourceRegistrationProposalStore) consume(input domain.ResourceRegistrationProposalInput, execCtx domain.ToolExecutionContext) (resourceRegistrationProposal, error) {
	if s == nil {
		return resourceRegistrationProposal{}, newResourceRegistrationError("proposal_unavailable", "resource registration proposal store is unavailable")
	}
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked(now)
	proposal, ok := s.byCall[strings.TrimSpace(execCtx.ToolCallID)]
	if !ok {
		return resourceRegistrationProposal{}, newResourceRegistrationError("proposal_expired", "resource registration proposal is missing, expired, cancelled, or already consumed")
	}
	if proposal.SessionID != strings.TrimSpace(execCtx.SessionID) || proposal.TurnID != strings.TrimSpace(execCtx.TurnID) || proposal.Hash != resourceRegistrationInputHash(input) {
		return resourceRegistrationProposal{}, newResourceRegistrationError("proposal_mismatch", "resource registration confirmation does not match the exact approved proposal")
	}
	delete(s.byCall, proposal.ToolCallID)
	return proposal, nil
}

func (s *resourceRegistrationProposalStore) discard(toolCallID string) {
	if s == nil || strings.TrimSpace(toolCallID) == "" {
		return
	}
	s.mu.Lock()
	delete(s.byCall, strings.TrimSpace(toolCallID))
	s.mu.Unlock()
}

func (s *resourceRegistrationProposalStore) cleanupLocked(now time.Time) {
	for key, proposal := range s.byCall {
		if !proposal.ExpiresAt.After(now) {
			delete(s.byCall, key)
		}
	}
}

func (s *resourceRegistrationProposalStore) evictOldestLocked() {
	oldestKey := ""
	var oldest time.Time
	for key, proposal := range s.byCall {
		if oldestKey == "" || proposal.CreatedAt.Before(oldest) {
			oldestKey, oldest = key, proposal.CreatedAt
		}
	}
	delete(s.byCall, oldestKey)
}

func resourceRegistrationInputHash(input domain.ResourceRegistrationProposalInput) string {
	raw, _ := json.Marshal(input)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (s *Service) prepareResourceRegistrationPermission(ctx context.Context, name string, args json.RawMessage, execCtx domain.ToolExecutionContext) ([]string, map[string]any, bool, error) {
	if name != toolRegistrationResourceName {
		return nil, nil, false, newResourceRegistrationError("invalid_registration_tool", "unknown resource registration tool")
	}
	var input domain.ResourceRegistrationProposalInput
	if err := decodeStrictToolArgs(args, &input); err != nil {
		return nil, nil, false, newResourceRegistrationError("invalid_arguments", err.Error())
	}
	normalized, err := normalizeResourceRegistrationInput(input, false)
	if err != nil {
		return nil, nil, false, err
	}
	if s.resourceRegistrationProposals == nil {
		s.resourceRegistrationProposals = newResourceRegistrationProposalStore()
	}
	proposal, err := s.resourceRegistrationProposals.prepare(normalized, execCtx)
	if err != nil {
		return nil, nil, false, err
	}
	metadata := map[string]any{
		"registrationProposalId": proposal.ID,
		"registrationKind":       "resource",
		"registrationResource":   normalized.Kind,
		"registrationResourceId": normalized.ID,
		"registrationSource":     normalized.Source,
		"registrationName":       normalized.Skill,
		"registrationTarget":     resourceRegistrationTarget(normalized),
		"registrationScope":      normalized.Scope,
		"registrationFileCount":  len(normalized.Files),
		"registrationGlobal":     normalized.Scope == domain.SkillScopeGlobal,
		"registrationExpiresAt":  domain.NowString(proposal.ExpiresAt),
		"riskLevel":              "medium",
		"category":               "external_resource_registration",
		"rememberScope":          "never",
	}
	return nil, metadata, false, nil
}

func (s *Service) commitResourceRegistrationProposal(ctx context.Context, input domain.ResourceRegistrationProposalInput, execCtx domain.ToolExecutionContext) (domain.ResourceRegistrationResult, error) {
	normalized, err := normalizeResourceRegistrationInput(input, true)
	if err != nil {
		return domain.ResourceRegistrationResult{}, err
	}
	if s.resourceRegistrationProposals == nil {
		return domain.ResourceRegistrationResult{}, newResourceRegistrationError("proposal_unavailable", "resource registration proposal store is unavailable")
	}
	if _, err := s.resourceRegistrationProposals.consume(normalized, execCtx); err != nil {
		return domain.ResourceRegistrationResult{}, err
	}
	s.resourceRegistrationProposals.commitMu.Lock()
	defer s.resourceRegistrationProposals.commitMu.Unlock()
	files := normalized.Files
	if len(files) == 0 {
		entry, relFiles, err := s.installResourceSkillWithCLI(ctx, normalized)
		if err != nil {
			return domain.ResourceRegistrationResult{}, err
		}
		return domain.ResourceRegistrationResult{
			Kind: domain.SessionResourceSkill, ID: normalized.ID, Name: entry.Name, Scope: entry.Scope, Status: "installed",
			FileCount: len(relFiles), ContentHash: entry.ContentHash, SourceHash: entry.ContentHash, Files: relFiles,
		}, nil
	}
	metadata := map[string]string{
		"resource.id":       normalized.ID,
		"resource.url":      normalized.URL,
		"resource.provider": "inline",
	}
	entry, relFiles, err := s.ensureSkillManager().InstallFiles(ctx, domain.SkillScopeGlobal, files, metadata)
	if err != nil {
		return domain.ResourceRegistrationResult{}, newResourceRegistrationError("resource_install_failed", err.Error())
	}
	_, _ = s.ScanGlobalSkills(ctx)
	return domain.ResourceRegistrationResult{
		Kind: domain.SessionResourceSkill, ID: normalized.ID, Name: entry.Name, Scope: entry.Scope, Status: "installed",
		FileCount: len(relFiles), ContentHash: entry.ContentHash, SourceHash: entry.ContentHash, Files: relFiles,
	}, nil
}

func normalizeResourceRegistrationInput(input domain.ResourceRegistrationProposalInput, allowEmptyFiles bool) (domain.ResourceRegistrationProposalInput, error) {
	input.Kind = strings.TrimSpace(strings.ToLower(input.Kind))
	input.ID = strings.Trim(strings.TrimSpace(input.ID), "/")
	input.Source = strings.Trim(strings.TrimSpace(input.Source), "/")
	input.Skill = strings.Trim(strings.TrimSpace(input.Skill), "/")
	input.URL = strings.TrimSpace(input.URL)
	input.Scope = normalizeSkillScope(input.Scope)
	if input.Kind == "" {
		input.Kind = domain.SessionResourceSkill
	}
	if input.Kind != domain.SessionResourceSkill {
		return input, newResourceRegistrationError("unsupported_resource_kind", "resource registration currently supports kind=skill")
	}
	if input.Scope != domain.SkillScopeGlobal {
		return input, newResourceRegistrationError("unsupported_resource_scope", "conversational resource registration currently installs global Skills")
	}
	if input.ID == "" && input.Source != "" && input.Skill != "" {
		input.ID = input.Source + "/" + input.Skill
	}
	if input.ID == "" && input.URL != "" {
		id, source, skill, err := resourceLocatorFromURL(input.URL, input.Skill)
		if err != nil {
			return input, err
		}
		input.ID = id
		if input.Source == "" {
			input.Source = source
		}
		if input.Skill == "" {
			input.Skill = skill
		}
	}
	if input.Source == "" && input.ID != "" {
		parts := strings.Split(input.ID, "/")
		if len(parts) >= 3 {
			input.Source = strings.Join(parts[:len(parts)-1], "/")
			if input.Skill == "" {
				input.Skill = parts[len(parts)-1]
			}
		}
	}
	if input.Skill == "" && len(input.Files) > 0 {
		if skill, err := skillNameFromResourceFiles(input.Files); err == nil {
			input.Skill = skill
		}
	}
	if len(input.Files) == 0 && !allowEmptyFiles {
		if input.ID == "" {
			return input, newResourceRegistrationError("invalid_resource_id", "resource registration requires a skills.sh id, URL, or complete files snapshot")
		}
	}
	if input.ID != "" && len(input.ID) > 240 {
		return input, newResourceRegistrationError("invalid_resource_id", "resource id is too long")
	}
	files, err := normalizeSkillResourceFiles(input.Files)
	if err != nil {
		return input, err
	}
	input.Files = files
	return input, nil
}

func resourceLocatorFromURL(raw string, requestedSkill string) (id string, source string, skill string, err error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed == nil || parsed.Host == "" {
		return "", "", "", newResourceRegistrationError("invalid_resource_url", "resource URL is invalid")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
		return "", "", "", newResourceRegistrationError("invalid_resource_url", "resource URL cannot include user info, query, or fragment")
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "github.com" || host == "www.github.com" {
		return githubSkillLocatorFromURL(parsed, requestedSkill)
	}
	if !strings.EqualFold(parsed.Hostname(), "skills.sh") && !strings.EqualFold(parsed.Hostname(), "www.skills.sh") {
		return "", "", "", newResourceRegistrationError("invalid_resource_url", "resource URL must be on skills.sh or github.com")
	}
	parts := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	if len(parts) >= 4 && parts[0] == "api" && parts[1] == "v1" && parts[2] == "skills" {
		parts = parts[3:]
	}
	if len(parts) < 3 || parts[0] == "p" {
		return "", "", "", newResourceRegistrationError("invalid_resource_url", "resource URL must identify one skills.sh Skill")
	}
	unescaped := make([]string, 0, len(parts))
	for _, part := range parts {
		value, err := url.PathUnescape(part)
		if err != nil || strings.TrimSpace(value) == "" {
			return "", "", "", newResourceRegistrationError("invalid_resource_url", "resource URL path is invalid")
		}
		unescaped = append(unescaped, value)
	}
	source = strings.Join(unescaped[:len(unescaped)-1], "/")
	skill = unescaped[len(unescaped)-1]
	return strings.Join(unescaped, "/"), source, skill, nil
}

func githubSkillLocatorFromURL(parsed *url.URL, requestedSkill string) (string, string, string, error) {
	parts := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	if len(parts) < 2 {
		return "", "", "", newResourceRegistrationError("invalid_resource_url", "GitHub resource URL must identify a repository")
	}
	owner, err := url.PathUnescape(parts[0])
	if err != nil || strings.TrimSpace(owner) == "" {
		return "", "", "", newResourceRegistrationError("invalid_resource_url", "GitHub owner is invalid")
	}
	repo, err := url.PathUnescape(parts[1])
	if err != nil || strings.TrimSpace(repo) == "" {
		return "", "", "", newResourceRegistrationError("invalid_resource_url", "GitHub repository is invalid")
	}
	source := strings.TrimSuffix(owner+"/"+repo, ".git")
	skill := strings.Trim(strings.TrimSpace(requestedSkill), "/")
	if skill == "" && len(parts) >= 5 && parts[2] == "tree" {
		last, err := url.PathUnescape(parts[len(parts)-1])
		if err != nil || strings.TrimSpace(last) == "" {
			return "", "", "", newResourceRegistrationError("invalid_resource_url", "GitHub Skill path is invalid")
		}
		skill = strings.TrimSpace(last)
	}
	if skill == "" {
		return "", "", "", newResourceRegistrationError("ambiguous_resource_locator", "GitHub repository URLs can contain multiple Skills; provide the exact skill slug")
	}
	return source + "/" + skill, source, skill, nil
}

func skillNameFromResourceFiles(files []domain.ResourceRegistrationFile) (string, error) {
	for _, file := range files {
		if filepath.ToSlash(strings.TrimSpace(file.Path)) != "SKILL.md" {
			continue
		}
		name, _, _, _, err := parseSkillMarkdown(file.Contents)
		return name, err
	}
	return "", errors.New("SKILL.md is missing")
}

func normalizeSkillResourceFiles(files []domain.ResourceRegistrationFile) ([]domain.ResourceRegistrationFile, error) {
	if len(files) == 0 {
		return nil, nil
	}
	if len(files) > skillResourceMaxFiles {
		return nil, newResourceRegistrationError("resource_too_large", "skill file tree has too many files")
	}
	seen := map[string]bool{}
	total := 0
	out := make([]domain.ResourceRegistrationFile, 0, len(files))
	for _, file := range files {
		path := filepath.ToSlash(strings.TrimSpace(file.Path))
		clean := filepath.Clean(path)
		clean = filepath.ToSlash(clean)
		if clean == "." || clean == "" || filepath.IsAbs(path) || strings.HasPrefix(clean, "../") || clean == ".." {
			return nil, newResourceRegistrationError("invalid_resource_file", "skill file paths must be relative and stay inside the Skill directory")
		}
		if len(clean) > 240 {
			return nil, newResourceRegistrationError("invalid_resource_file", "skill file path is too long")
		}
		if strings.EqualFold(clean, "SKILL.md") {
			clean = "SKILL.md"
		}
		if seen[clean] {
			return nil, newResourceRegistrationError("invalid_resource_file", "skill file tree contains duplicate paths")
		}
		seen[clean] = true
		size := len([]byte(file.Contents))
		if size > skillResourceMaxFileBytes {
			return nil, newResourceRegistrationError("resource_too_large", "skill file exceeds the per-file size limit")
		}
		total += size
		if total > skillResourceMaxTotalBytes {
			return nil, newResourceRegistrationError("resource_too_large", "skill file tree exceeds the total size limit")
		}
		out = append(out, domain.ResourceRegistrationFile{Path: clean, Contents: file.Contents})
	}
	if !seen["SKILL.md"] {
		return nil, newResourceRegistrationError("invalid_resource_file", "skill file tree must include SKILL.md")
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func (s *Service) installResourceSkillWithCLI(ctx context.Context, input domain.ResourceRegistrationProposalInput) (domain.SkillEntry, []string, error) {
	manager := s.ensureSkillManager()
	packageRef, skillName, err := skillsCLIInstallLocator(input)
	if err != nil {
		return domain.SkillEntry{}, nil, err
	}
	if err := runSkillsCLIInstall(ctx, manager.home, packageRef, skillName); err != nil {
		return domain.SkillEntry{}, nil, err
	}
	sourceRoot, err := findAgentSkillInstallRoot(manager.home, skillName)
	if err != nil {
		return domain.SkillEntry{}, nil, err
	}
	if _, err := parseSkillDirectory(sourceRoot); err != nil {
		return domain.SkillEntry{}, nil, newResourceRegistrationError("resource_install_failed", err.Error())
	}
	if _, err := s.ScanGlobalSkills(ctx); err != nil {
		return domain.SkillEntry{}, nil, newResourceRegistrationError("resource_catalog_refresh_failed", err.Error())
	}
	candidate, err := manager.findImportCandidateForRootOrName(ctx, sourceRoot, skillName)
	if err != nil {
		return domain.SkillEntry{}, nil, err
	}
	entry, err := manager.Import(ctx, domain.SkillImportInput{CandidateID: candidate.ID, TargetScope: domain.SkillScopeGlobal})
	if err != nil {
		return domain.SkillEntry{}, nil, newResourceRegistrationError("resource_install_failed", err.Error())
	}
	if err := os.RemoveAll(sourceRoot); err != nil {
		return domain.SkillEntry{}, nil, newResourceRegistrationError("resource_cleanup_failed", err.Error())
	}
	if _, err := s.ScanGlobalSkills(ctx); err != nil {
		return domain.SkillEntry{}, nil, newResourceRegistrationError("resource_catalog_refresh_failed", err.Error())
	}
	files, err := listSkillRelativeFiles(entry.RootPath)
	if err != nil {
		return domain.SkillEntry{}, nil, newResourceRegistrationError("resource_install_failed", err.Error())
	}
	return entry, files, nil
}

func skillsCLIInstallLocator(input domain.ResourceRegistrationProposalInput) (string, string, error) {
	skill := strings.Trim(strings.TrimSpace(input.Skill), "/")
	source := strings.Trim(strings.TrimSpace(input.Source), "/")
	if skill == "" && input.ID != "" {
		parts := strings.Split(strings.Trim(input.ID, "/"), "/")
		if len(parts) >= 3 {
			source = strings.Join(parts[:len(parts)-1], "/")
			skill = parts[len(parts)-1]
		}
	}
	if skill == "" {
		return "", "", newResourceRegistrationError("invalid_resource_id", "skills CLI installation requires an exact Skill slug")
	}
	packageRef := source
	if input.URL != "" {
		parsed, err := url.Parse(input.URL)
		if err == nil && parsed != nil {
			host := strings.ToLower(parsed.Hostname())
			if host == "github.com" || host == "www.github.com" {
				packageRef = input.URL
			}
		}
	}
	if packageRef == "" {
		return "", "", newResourceRegistrationError("invalid_resource_id", "skills CLI installation requires a package source")
	}
	return packageRef, skill, nil
}

func runSkillsCLIInstall(ctx context.Context, home string, packageRef string, skillName string) error {
	commandCtx, cancel := context.WithTimeout(ctx, skillsCLIInstallTimeout)
	defer cancel()
	args := []string{
		"-y", "skills@latest", "add", packageRef,
		"-g", "--copy", "-y",
		"--skill", skillName,
		"--agent", "codex",
		"--full-depth",
	}
	cmd := exec.CommandContext(commandCtx, "npx", args...)
	cmd.Dir = home
	cmd.Env = resourceRegistrationCommandEnv(home)
	cmd.Stdin = strings.NewReader("")
	output := &boundedCommandOutput{limit: skillsCLIOutputMaxBytes}
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Run(); err != nil {
		if errors.Is(commandCtx.Err(), context.DeadlineExceeded) {
			return newResourceRegistrationError("resource_install_timeout", "skills CLI installation timed out")
		}
		detail := strings.TrimSpace(output.String())
		if detail != "" {
			return newResourceRegistrationError("skills_cli_failed", sanitizeMCPError(detail))
		}
		return newResourceRegistrationError("skills_cli_failed", sanitizeMCPError(err.Error()))
	}
	return nil
}

func resourceRegistrationCommandEnv(home string) []string {
	overrides := []struct {
		key   string
		value string
	}{
		{key: "HOME", value: home},
		{key: "CI", value: "1"},
		{key: "NO_COLOR", value: "1"},
		{key: "DISABLE_TELEMETRY", value: "1"},
	}
	env := append([]string{}, os.Environ()...)
	for _, override := range overrides {
		prefix := override.key + "="
		replaced := false
		for i, item := range env {
			if strings.HasPrefix(item, prefix) {
				env[i] = prefix + override.value
				replaced = true
			}
		}
		if !replaced {
			env = append(env, prefix+override.value)
		}
	}
	return env
}

type boundedCommandOutput struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *boundedCommandOutput) Write(p []byte) (int, error) {
	if b == nil {
		return len(p), nil
	}
	remaining := b.limit - b.buf.Len()
	if remaining > 0 {
		if len(p) > remaining {
			_, _ = b.buf.Write(p[:remaining])
			b.truncated = true
		} else {
			_, _ = b.buf.Write(p)
		}
	} else {
		b.truncated = true
	}
	return len(p), nil
}

func (b *boundedCommandOutput) String() string {
	if b == nil {
		return ""
	}
	out := b.buf.String()
	if b.truncated {
		out += "\n[truncated]"
	}
	return out
}

func findAgentSkillInstallRoot(home string, skillName string) (string, error) {
	root := filepath.Join(home, ".agents", "skills")
	direct := filepath.Join(root, skillName)
	if parsed, err := parseSkillDirectory(direct); err == nil && parsed.Name == skillName {
		return direct, nil
	}
	matches := []string{}
	for _, dir := range discoverSkillDirectories(root) {
		parsed, err := parseSkillDirectory(dir)
		if err != nil {
			continue
		}
		if parsed.Name == skillName {
			matches = append(matches, dir)
		}
	}
	sort.Strings(matches)
	switch len(matches) {
	case 0:
		return "", newResourceRegistrationError("resource_snapshot_unavailable", "skills CLI did not install the requested Skill into ~/.agents/skills")
	case 1:
		return matches[0], nil
	default:
		return "", newResourceRegistrationError("ambiguous_resource_locator", "skills CLI installed multiple Skills with the requested name")
	}
}

func (m *SkillManager) findImportCandidateForRootOrName(ctx context.Context, root string, name string) (domain.SkillImportCandidate, error) {
	candidates, err := m.store.ListSkillImportCandidates(ctx, true)
	if err != nil {
		return domain.SkillImportCandidate{}, newResourceRegistrationError("resource_catalog_refresh_failed", err.Error())
	}
	name = normalizeSkillName(name)
	for _, candidate := range candidates {
		if samePath(candidate.RootPath, root) {
			return candidate, nil
		}
	}
	for _, candidate := range candidates {
		if normalizeSkillName(candidate.Name) == name && candidate.Source == domain.SkillSourceAgents {
			return candidate, nil
		}
	}
	return domain.SkillImportCandidate{}, newResourceRegistrationError("resource_snapshot_unavailable", "installed Skill was not found in the refreshed Skill catalog")
}

func listSkillRelativeFiles(root string) ([]string, error) {
	files := []string{}
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 || d.Type() != 0 {
			return fmt.Errorf("%s: skill directories can contain only regular files", path)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func resourceRegistrationTarget(input domain.ResourceRegistrationProposalInput) string {
	if input.URL != "" {
		return input.URL
	}
	if input.ID != "" {
		return "skills.sh/" + input.ID
	}
	if input.Skill != "" {
		return input.Skill
	}
	return "inline Skill file snapshot"
}

func (m *SkillManager) InstallFiles(ctx context.Context, scope string, files []domain.ResourceRegistrationFile, metadata map[string]string) (domain.SkillEntry, []string, error) {
	if m == nil || m.store == nil {
		return domain.SkillEntry{}, nil, errors.New("skill store is not configured")
	}
	files, err := normalizeSkillResourceFiles(files)
	if err != nil {
		return domain.SkillEntry{}, nil, err
	}
	name, _, _, _, err := parseSkillMarkdown(skillFileContents(files, "SKILL.md"))
	if err != nil {
		return domain.SkillEntry{}, nil, err
	}
	scope = normalizeSkillScope(scope)
	managedRoot := filepath.Join(m.home, ".aivo", "skills")
	if err := os.MkdirAll(managedRoot, 0o755); err != nil {
		return domain.SkillEntry{}, nil, err
	}
	stagingParent, err := os.MkdirTemp(managedRoot, ".resource-install-")
	if err != nil {
		return domain.SkillEntry{}, nil, err
	}
	defer os.RemoveAll(stagingParent)
	stagedRoot := filepath.Join(stagingParent, name)
	for _, file := range files {
		target := filepath.Join(stagedRoot, filepath.FromSlash(file.Path))
		if !skillPathWithin(target, stagedRoot) && filepath.Clean(target) != filepath.Clean(stagedRoot) {
			return domain.SkillEntry{}, nil, errors.New("skill file path resolves outside staging directory")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return domain.SkillEntry{}, nil, err
		}
		if err := os.WriteFile(target, []byte(file.Contents), 0o644); err != nil {
			return domain.SkillEntry{}, nil, err
		}
	}
	parsed, err := parseSkillDirectory(stagedRoot)
	if err != nil {
		return domain.SkillEntry{}, nil, err
	}
	destRoot := m.managedSkillRoot(scope, parsed.Name, stagedRoot)
	existing, existingErr := m.store.GetSkillByName(ctx, parsed.Name, scope)
	if existingErr == nil && existing.ID != "" {
		if existing.ContentHash != parsed.ContentHash {
			return domain.SkillEntry{}, nil, errors.New("skill name already exists with different content")
		}
		return existing, resourceFilePaths(files), nil
	}
	if err := os.MkdirAll(filepath.Dir(destRoot), 0o755); err != nil {
		return domain.SkillEntry{}, nil, err
	}
	if _, err := os.Lstat(destRoot); err == nil {
		return domain.SkillEntry{}, nil, errors.New("managed skill directory already exists outside the catalog")
	}
	if err := os.Rename(stagedRoot, destRoot); err != nil {
		return domain.SkillEntry{}, nil, err
	}
	published, err := parseSkillDirectory(destRoot)
	if err != nil || published.ContentHash != parsed.ContentHash {
		_ = os.RemoveAll(destRoot)
		if err == nil {
			err = errors.New("published skill integrity does not match staged skill")
		}
		return domain.SkillEntry{}, nil, err
	}
	entryMetadata := make(map[string]string, len(published.Metadata)+len(metadata))
	for key, value := range published.Metadata {
		entryMetadata[key] = value
	}
	for key, value := range metadata {
		if strings.TrimSpace(value) != "" {
			entryMetadata[key] = value
		}
	}
	now := domain.NowString(time.Now())
	entry, err := m.store.SaveSkill(ctx, domain.SkillEntry{
		ID: uuid.NewString(), Name: published.Name, Description: published.Description, Scope: scope, Source: domain.SkillSourceAivo,
		RootPath: published.RootPath, SkillPath: published.SkillPath, ContentHash: published.ContentHash, Enabled: true,
		Metadata: entryMetadata, TimeCreated: now, TimeUpdated: now,
	})
	if err != nil {
		_ = os.RemoveAll(destRoot)
		return domain.SkillEntry{}, nil, err
	}
	return entry, resourceFilePaths(files), nil
}

func skillFileContents(files []domain.ResourceRegistrationFile, path string) string {
	for _, file := range files {
		if file.Path == path {
			return file.Contents
		}
	}
	return ""
}

func resourceFilePaths(files []domain.ResourceRegistrationFile) []string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	sort.Strings(paths)
	return paths
}
