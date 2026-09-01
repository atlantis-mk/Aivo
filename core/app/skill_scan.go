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
	"gopkg.in/yaml.v3"

	"aivo/core/domain"
)

func (m *SkillManager) scan(ctx context.Context, roots []skillScanRoot, scope string) (domain.SkillScanResult, error) {
	result := domain.SkillScanResult{}
	now := domain.NowString(time.Now())
	ignoredNames := map[string]bool{}
	if candidates, err := m.store.ListSkillImportCandidates(ctx, true); err == nil {
		for _, candidate := range candidates {
			if candidate.Status == domain.SkillCandidateStatusIgnored {
				ignoredNames[normalizeSkillName(candidate.Name)] = true
			}
		}
	}
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
			if previous, err := m.store.GetSkillImportCandidate(ctx, candidate.ID); err == nil && previous.ID != "" {
				candidate.Status = previous.Status
				candidate.ConflictID = previous.ConflictID
				candidate.Error = previous.Error
			}
			if ignoredNames[normalizeSkillName(parsed.Name)] {
				candidate.Status = domain.SkillCandidateStatusIgnored
			}
			if existing, err := m.store.GetSkillByName(ctx, parsed.Name, scope); err == nil && existing.ID != "" {
				if existing.ContentHash == parsed.ContentHash {
					_, _ = m.store.SaveSkillSource(ctx, domain.SkillSource{
						ID: sourceIDForPath(existing.ID, parsed.SkillPath), SkillID: existing.ID, Source: root.Source, Scope: scope,
						RootPath: parsed.RootPath, SkillPath: parsed.SkillPath, ContentHash: parsed.ContentHash, LastSeenAt: now,
					})
					if candidate.Status != domain.SkillCandidateStatusIgnored {
						candidate.Status = domain.SkillCandidateStatusImported
						candidate.ConflictID = ""
						candidate.Error = ""
					}
				} else {
					candidate.Status = domain.SkillCandidateStatusIgnored
					candidate.ConflictID = existing.ID
					candidate.Error = "skill name already exists with different content"
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
		{Path: filepath.Join(home, ".codex", "skills"), Source: domain.SkillSourceCodex},
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
		{Path: filepath.Join(root, ".codex", "skills"), Source: domain.SkillSourceCodex},
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
	if err := validateSkillDirectoryFiles(root); err != nil {
		return parsedSkill{}, err
	}
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
	fm, content, ok := splitSkillMarkdownFrontmatter(rest)
	if !ok {
		return "", "", nil, "", errors.New("SKILL.md frontmatter is not closed")
	}
	var values map[string]any
	if err := yaml.Unmarshal([]byte(fm), &values); err != nil {
		return "", "", nil, "", fmt.Errorf("SKILL.md frontmatter is invalid YAML: %w", err)
	}
	name := strings.TrimSpace(stringFromSkillFrontmatter(values["name"]))
	description := strings.TrimSpace(stringFromSkillFrontmatter(values["description"]))
	if name == "" || len(name) > 64 || !skillNamePattern.MatchString(name) {
		return "", "", nil, "", errors.New("skill name is invalid")
	}
	if description == "" {
		return "", "", nil, "", errors.New("skill description is required")
	}
	if len(description) > 1024 {
		return "", "", nil, "", errors.New("skill description exceeds 1024 bytes")
	}
	metadata := map[string]string{}
	for key, value := range values {
		switch key {
		case "name", "description":
			continue
		case "metadata":
			nested, ok := stringMapFromSkillFrontmatter(value)
			if !ok {
				return "", "", nil, "", errors.New("skill metadata must be a string map")
			}
			for nestedKey, nestedValue := range nested {
				metadata["metadata."+nestedKey] = nestedValue
			}
		case "license", "compatibility", "allowed-tools":
			metadata[key] = strings.TrimSpace(stringFromSkillFrontmatter(value))
		default:
			metadata[key] = strings.TrimSpace(stringFromSkillFrontmatter(value))
		}
	}
	return name, description, metadata, content, nil
}

func splitSkillMarkdownFrontmatter(rest string) (string, string, bool) {
	start := 0
	for start <= len(rest) {
		next := strings.IndexByte(rest[start:], '\n')
		lineEnd := len(rest)
		afterLine := len(rest)
		if next >= 0 {
			lineEnd = start + next
			afterLine = lineEnd + 1
		}
		if rest[start:lineEnd] == "---" {
			return rest[:start], strings.TrimSpace(rest[afterLine:]), true
		}
		if next < 0 {
			break
		}
		start = afterLine
	}
	return "", "", false
}

func stringFromSkillFrontmatter(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return ""
	default:
		return fmt.Sprint(typed)
	}
}

func stringMapFromSkillFrontmatter(value any) (map[string]string, bool) {
	raw, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	out := make(map[string]string, len(raw))
	for key, item := range raw {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, false
		}
		out[key] = strings.TrimSpace(stringFromSkillFrontmatter(item))
	}
	return out, true
}

func validateSkillDirectoryFiles(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s: skill directories cannot contain symlinks", path)
		}
		if !d.IsDir() && d.Type() != 0 {
			return fmt.Errorf("%s: skill directories can contain only directories and regular files", path)
		}
		return nil
	})
}

func hashSkillDirectory(root string) (string, error) {
	var files []string
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s: skill directories cannot contain symlinks", path)
		}
		if d.IsDir() {
			return nil
		}
		if d.Type() != 0 {
			return fmt.Errorf("%s: skill directories can contain only directories and regular files", path)
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
