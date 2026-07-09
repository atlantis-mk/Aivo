package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"aivo/core/domain"
)

func (m *SkillManager) scan(ctx context.Context, roots []skillScanRoot, scope string) (domain.SkillScanResult, error) {
	result := domain.SkillScanResult{}
	now := domain.NowString(time.Now())
	for _, root := range roots {
		for _, dir := range discoverSkillDirectories(root.Path) {
			result.Scanned++
			parsed, err := parseSkillDirectory(dir)
			if err != nil {
				result.Errors = append(result.Errors, err.Error())
				continue
			}
			if root.Source == domain.SkillSourceAivo {
				entry, err := m.upsertManagedSkill(ctx, parsed, scope, now)
				if err != nil {
					result.Errors = append(result.Errors, err.Error())
					continue
				}
				result.Entries = append(result.Entries, entry)
				result.Imported++
				continue
			}
			candidate := domain.SkillImportCandidate{
				ID: candidateIDForPath(parsed.SkillPath), Name: parsed.Name, Description: parsed.Description, Scope: scope, Source: root.Source,
				RootPath: parsed.RootPath, SkillPath: parsed.SkillPath, ContentHash: parsed.ContentHash,
				Status: domain.SkillCandidateStatusPending, LastSeenAt: now,
			}
			if existing, err := m.store.GetSkillByName(ctx, parsed.Name, scope); err == nil && existing.ID != "" {
				if existing.ContentHash == parsed.ContentHash {
					_, _ = m.store.SaveSkillSource(ctx, domain.SkillSource{
						ID: sourceIDForPath(existing.ID, parsed.SkillPath), SkillID: existing.ID, Source: root.Source, Scope: scope,
						RootPath: parsed.RootPath, SkillPath: parsed.SkillPath, ContentHash: parsed.ContentHash, LastSeenAt: now,
					})
					candidate.Status = domain.SkillCandidateStatusImported
				} else {
					candidate.Status = domain.SkillCandidateStatusConflict
					candidate.ConflictID = existing.ID
					result.Conflicts++
				}
			}
			saved, err := m.store.SaveSkillImportCandidate(ctx, candidate)
			if err != nil {
				result.Errors = append(result.Errors, err.Error())
				continue
			}
			result.Candidates = append(result.Candidates, saved)
		}
	}
	sort.Slice(result.Entries, func(i, j int) bool { return result.Entries[i].Name < result.Entries[j].Name })
	sort.Slice(result.Candidates, func(i, j int) bool { return result.Candidates[i].Name < result.Candidates[j].Name })
	return result, nil
}

func (m *SkillManager) upsertManagedSkill(ctx context.Context, parsed parsedSkill, scope string, now string) (domain.SkillEntry, error) {
	existing, _ := m.store.GetSkillByName(ctx, parsed.Name, scope)
	id := existing.ID
	created := existing.TimeCreated
	if id == "" {
		id = uuid.NewString()
		created = now
	}
	return m.store.SaveSkill(ctx, domain.SkillEntry{
		ID: id, Name: parsed.Name, Description: parsed.Description, Scope: scope, Source: domain.SkillSourceAivo,
		RootPath: parsed.RootPath, SkillPath: parsed.SkillPath, ContentHash: parsed.ContentHash, Enabled: existing.ID == "" || existing.Enabled,
		Metadata: parsed.Metadata, TimeCreated: created, TimeUpdated: now,
	})
}

func (m *SkillManager) managedSkillRoot(scope string, name string, sourceRoot string) string {
	base := filepath.Join(m.home, ".aivo", "skills")
	if scope == domain.SkillScopeProject {
		sum := sha256.Sum256([]byte(filepath.Clean(sourceRoot)))
		base = filepath.Join(base, "projects", hex.EncodeToString(sum[:6]))
	}
	return filepath.Join(base, normalizeSkillName(name))
}

func globalSkillScanRoots(home string) []skillScanRoot {
	return []skillScanRoot{
		{Path: filepath.Join(home, ".aivo", "skills"), Source: domain.SkillSourceAivo},
		{Path: filepath.Join(home, ".claude", "skills"), Source: domain.SkillSourceClaude},
		{Path: filepath.Join(home, ".agents", "skills"), Source: domain.SkillSourceAgents},
		{Path: filepath.Join(home, ".config", "opencode", "skills"), Source: domain.SkillSourceOpenCode},
		{Path: filepath.Join(home, ".config", "opencode", "skill"), Source: domain.SkillSourceOpenCode},
		{Path: filepath.Join(home, ".opencode", "skills"), Source: domain.SkillSourceOpenCode},
		{Path: filepath.Join(home, ".opencode", "skill"), Source: domain.SkillSourceOpenCode},
	}
}

func projectSkillScanRoots(root string) []skillScanRoot {
	return []skillScanRoot{
		{Path: filepath.Join(root, ".aivo", "skills"), Source: domain.SkillSourceAivo},
		{Path: filepath.Join(root, ".claude", "skills"), Source: domain.SkillSourceClaude},
		{Path: filepath.Join(root, ".agents", "skills"), Source: domain.SkillSourceAgents},
		{Path: filepath.Join(root, ".opencode", "skills"), Source: domain.SkillSourceOpenCode},
		{Path: filepath.Join(root, ".opencode", "skill"), Source: domain.SkillSourceOpenCode},
	}
}

func discoverSkillDirectories(root string) []string {
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil
	}
	var dirs []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == "SKILL.md" {
			dirs = append(dirs, filepath.Dir(path))
		}
		return nil
	})
	sort.Strings(dirs)
	return dirs
}

func parseSkillDirectory(root string) (parsedSkill, error) {
	root = filepath.Clean(root)
	skillPath := filepath.Join(root, "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		return parsedSkill{}, err
	}
	name, description, metadata, content, err := parseSkillMarkdown(string(data))
	if err != nil {
		return parsedSkill{}, fmt.Errorf("%s: %w", skillPath, err)
	}
	dirName := filepath.Base(root)
	if name != dirName && !strings.Contains(root, string(filepath.Separator)+"projects"+string(filepath.Separator)) {
		return parsedSkill{}, fmt.Errorf("%s: skill name %q does not match directory %q", skillPath, name, dirName)
	}
	hash, err := hashSkillDirectory(root)
	if err != nil {
		return parsedSkill{}, err
	}
	return parsedSkill{Name: name, Description: description, Metadata: metadata, RootPath: root, SkillPath: skillPath, Content: content, ContentHash: hash}, nil
}

func parseSkillMarkdown(raw string) (string, string, map[string]string, string, error) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	if !strings.HasPrefix(raw, "---\n") {
		return "", "", nil, "", errors.New("SKILL.md must start with YAML frontmatter")
	}
	rest := strings.TrimPrefix(raw, "---\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", "", nil, "", errors.New("SKILL.md frontmatter is not closed")
	}
	fm := rest[:end]
	content := strings.TrimSpace(rest[end+len("\n---"):])
	values := map[string]string{}
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	name := normalizeSkillName(values["name"])
	description := strings.TrimSpace(values["description"])
	if name == "" || !skillNamePattern.MatchString(name) {
		return "", "", nil, "", errors.New("skill name is invalid")
	}
	if description == "" {
		return "", "", nil, "", errors.New("skill description is required")
	}
	metadata := map[string]string{}
	for key, value := range values {
		if key != "name" && key != "description" {
			metadata[key] = value
		}
	}
	return name, description, metadata, content, nil
}

func hashSkillDirectory(root string) (string, error) {
	var files []string
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	}); err != nil {
		return "", err
	}
	sort.Strings(files)
	h := sha256.New()
	for _, file := range files {
		rel, _ := filepath.Rel(root, file)
		_, _ = h.Write([]byte(rel))
		_, _ = h.Write([]byte{0})
		data, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		_, _ = h.Write(data)
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
