package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"aivo/core/domain"
)

type skillStore interface {
	SaveSkill(context.Context, domain.SkillEntry) (domain.SkillEntry, error)
	GetSkill(context.Context, string) (domain.SkillEntry, error)
	GetSkillByName(context.Context, string, string) (domain.SkillEntry, error)
	ListSkills(context.Context, bool) ([]domain.SkillEntry, error)
	SetSkillEnabled(context.Context, string, bool) (domain.SkillEntry, error)
	DeleteSkill(context.Context, string) error
	SaveSkillSource(context.Context, domain.SkillSource) (domain.SkillSource, error)
	ListSkillSources(context.Context, string) ([]domain.SkillSource, error)
	SaveSkillImportCandidate(context.Context, domain.SkillImportCandidate) (domain.SkillImportCandidate, error)
	GetSkillImportCandidate(context.Context, string) (domain.SkillImportCandidate, error)
	ListSkillImportCandidates(context.Context, bool) ([]domain.SkillImportCandidate, error)
	MarkSkillImportCandidateStatus(context.Context, string, string, string, string) (domain.SkillImportCandidate, error)
}

type SkillManager struct {
	store skillStore
	home  string
}

type parsedSkill struct {
	Name        string
	Description string
	Metadata    map[string]string
	RootPath    string
	SkillPath   string
	Content     string
	ContentHash string
}

var skillNamePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func NewSkillManager(store any) *SkillManager {
	home, _ := os.UserHomeDir()
	sm := &SkillManager{home: home}
	if s, ok := store.(skillStore); ok {
		sm.store = s
	}
	return sm
}

func (m *SkillManager) ScanGlobal(ctx context.Context) (domain.SkillScanResult, error) {
	if m == nil || m.store == nil {
		return domain.SkillScanResult{}, errors.New("skill store is not configured")
	}
	return m.scan(ctx, globalSkillScanRoots(m.home), domain.SkillScopeGlobal)
}

func (m *SkillManager) ScanProject(ctx context.Context, workspaceRoot string) (domain.SkillScanResult, error) {
	if m == nil || m.store == nil {
		return domain.SkillScanResult{}, errors.New("skill store is not configured")
	}
	workspaceRoot = strings.TrimSpace(workspaceRoot)
	if workspaceRoot == "" {
		return domain.SkillScanResult{}, errors.New("workspaceRoot is required")
	}
	return m.scan(ctx, projectSkillScanRoots(workspaceRoot), domain.SkillScopeProject)
}

func (m *SkillManager) List(ctx context.Context, input domain.SkillListInput) (domain.SkillListResult, error) {
	if m == nil || m.store == nil {
		return domain.SkillListResult{}, errors.New("skill store is not configured")
	}
	if strings.TrimSpace(input.WorkspaceRoot) != "" {
		_, _ = m.ScanProject(ctx, input.WorkspaceRoot)
	} else {
		_, _ = m.ScanGlobal(ctx)
	}
	entries, err := m.store.ListSkills(ctx, input.IncludeDisabled)
	if err != nil {
		return domain.SkillListResult{}, err
	}
	result := domain.SkillListResult{Entries: entries}
	if input.IncludeCandidates {
		candidates, err := m.store.ListSkillImportCandidates(ctx, true)
		if err != nil {
			return domain.SkillListResult{}, err
		}
		result.Candidates = candidates
	}
	return result, nil
}

func (m *SkillManager) Import(ctx context.Context, input domain.SkillImportInput) (domain.SkillEntry, error) {
	if m == nil || m.store == nil {
		return domain.SkillEntry{}, errors.New("skill store is not configured")
	}
	candidate, err := m.store.GetSkillImportCandidate(ctx, input.CandidateID)
	if err != nil {
		return domain.SkillEntry{}, err
	}
	scope := normalizeSkillScope(firstNonEmpty(input.TargetScope, candidate.Scope))
	parsed, err := parseSkillDirectory(candidate.RootPath)
	if err != nil {
		_, _ = m.store.MarkSkillImportCandidateStatus(ctx, candidate.ID, domain.SkillCandidateStatusPending, candidate.ConflictID, err.Error())
		return domain.SkillEntry{}, err
	}
	destRoot := m.managedSkillRoot(scope, parsed.Name, candidate.RootPath)
	if samePath(parsed.RootPath, destRoot) {
		destRoot = parsed.RootPath
	} else if err := copyDirectory(parsed.RootPath, destRoot); err != nil {
		return domain.SkillEntry{}, err
	}
	managed, err := parseSkillDirectory(destRoot)
	if err != nil {
		return domain.SkillEntry{}, err
	}
	existing, existingErr := m.store.GetSkillByName(ctx, managed.Name, scope)
	if existingErr == nil && existing.ID != "" && existing.ContentHash != managed.ContentHash {
		_, _ = m.store.MarkSkillImportCandidateStatus(ctx, candidate.ID, domain.SkillCandidateStatusConflict, existing.ID, "skill name already exists with different content")
		return domain.SkillEntry{}, errors.New("skill name already exists with different content")
	}
	now := domain.NowString(time.Now())
	id := existing.ID
	created := existing.TimeCreated
	if id == "" {
		id = uuid.NewString()
		created = now
	}
	entry, err := m.store.SaveSkill(ctx, domain.SkillEntry{
		ID: id, Name: managed.Name, Description: managed.Description, Scope: scope, Source: domain.SkillSourceAivo,
		RootPath: managed.RootPath, SkillPath: managed.SkillPath, ContentHash: managed.ContentHash, Enabled: true,
		Metadata: managed.Metadata, TimeCreated: created, TimeUpdated: now,
	})
	if err != nil {
		return domain.SkillEntry{}, err
	}
	_, _ = m.store.SaveSkillSource(ctx, domain.SkillSource{
		ID: sourceIDForPath(entry.ID, candidate.SkillPath), SkillID: entry.ID, Source: candidate.Source, Scope: candidate.Scope,
		RootPath: candidate.RootPath, SkillPath: candidate.SkillPath, ContentHash: candidate.ContentHash, LastSeenAt: now,
	})
	_, _ = m.store.MarkSkillImportCandidateStatus(ctx, candidate.ID, domain.SkillCandidateStatusImported, "", "")
	return entry, nil
}

func (m *SkillManager) SetEnabled(ctx context.Context, input domain.SkillEnabledInput) (domain.SkillEntry, error) {
	if m == nil || m.store == nil {
		return domain.SkillEntry{}, errors.New("skill store is not configured")
	}
	return m.store.SetSkillEnabled(ctx, input.SkillID, input.Enabled)
}

func (m *SkillManager) Delete(ctx context.Context, skillID string) error {
	if m == nil || m.store == nil {
		return errors.New("skill store is not configured")
	}
	skill, err := m.store.GetSkill(ctx, skillID)
	if err != nil {
		return err
	}
	if err := m.store.DeleteSkill(ctx, skillID); err != nil {
		return err
	}
	managedRoot := filepath.Join(m.home, ".aivo", "skills")
	if strings.TrimSpace(skill.RootPath) != "" && skillPathWithin(skill.RootPath, managedRoot) {
		_ = os.RemoveAll(skill.RootPath)
	}
	return nil
}

func (m *SkillManager) Resolve(ctx context.Context, id string, name string, scope string) (domain.SkillEntry, error) {
	if m == nil || m.store == nil {
		return domain.SkillEntry{}, errors.New("skill store is not configured")
	}
	if strings.TrimSpace(id) != "" {
		return m.store.GetSkill(ctx, id)
	}
	return m.store.GetSkillByName(ctx, normalizeSkillName(name), normalizeSkillScope(scope))
}

func (m *SkillManager) ReadContent(skill domain.SkillEntry) (string, error) {
	path := strings.TrimSpace(skill.SkillPath)
	if path == "" {
		path = filepath.Join(skill.RootPath, "SKILL.md")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	_, _, _, content, err := parseSkillMarkdown(string(data))
	if err != nil {
		return "", err
	}
	return content, nil
}

func (m *SkillManager) SupportingFiles(skill domain.SkillEntry, limit int) ([]string, error) {
	if limit <= 0 {
		return nil, nil
	}
	root := strings.TrimSpace(skill.RootPath)
	if root == "" {
		root = filepath.Dir(strings.TrimSpace(skill.SkillPath))
	}
	if root == "" {
		return nil, nil
	}
	var files []string
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) == "SKILL.md" {
			return nil
		}
		files = append(files, path)
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Strings(files)
	if len(files) > limit {
		files = files[:limit]
	}
	return files, nil
}

type skillScanRoot struct {
	Path   string
	Source string
}
