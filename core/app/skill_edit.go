package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"aivo/core/domain"
)

const (
	skillDescriptionMaxBytes = 1024
	skillContentMaxBytes     = 262144
)

func (m *SkillManager) GetForEdit(ctx context.Context, skillID string) (domain.SkillEditResult, error) {
	if m == nil || m.store == nil {
		return domain.SkillEditResult{}, errors.New("skill store is not configured")
	}
	skill, err := m.store.GetSkill(ctx, strings.TrimSpace(skillID))
	if err != nil {
		return domain.SkillEditResult{}, err
	}
	path, err := m.editableSkillPath(skill)
	if err != nil {
		return domain.SkillEditResult{}, err
	}
	lock := lockForFile(path)
	lock.Lock()
	defer lock.Unlock()
	return m.skillEditSnapshot(ctx, skill)
}

func (m *SkillManager) Update(ctx context.Context, input domain.SkillUpdateInput) (domain.SkillEditResult, error) {
	if m == nil || m.store == nil {
		return domain.SkillEditResult{}, errors.New("skill store is not configured")
	}
	input.SkillID = strings.TrimSpace(input.SkillID)
	input.Description = strings.TrimSpace(input.Description)
	input.ExpectedContentHash = strings.TrimSpace(input.ExpectedContentHash)
	if input.SkillID == "" {
		return domain.SkillEditResult{}, errors.New("skillId is required")
	}
	if input.Description == "" {
		return domain.SkillEditResult{}, errors.New("skill description is required")
	}
	if len([]byte(input.Description)) > skillDescriptionMaxBytes {
		return domain.SkillEditResult{}, fmt.Errorf("skill description exceeds %d bytes", skillDescriptionMaxBytes)
	}
	if len([]byte(input.Content)) > skillContentMaxBytes {
		return domain.SkillEditResult{}, errors.New("skill content exceeds 262144 bytes")
	}
	if input.ExpectedContentHash == "" {
		return domain.SkillEditResult{}, errors.New("expectedContentHash is required")
	}

	skill, err := m.store.GetSkill(ctx, input.SkillID)
	if err != nil {
		return domain.SkillEditResult{}, err
	}
	path, err := m.editableSkillPath(skill)
	if err != nil {
		return domain.SkillEditResult{}, err
	}
	lock := lockForFile(path)
	lock.Lock()
	defer lock.Unlock()

	current, err := parseSkillDirectory(skill.RootPath)
	if err != nil {
		return domain.SkillEditResult{}, err
	}
	if current.Name != skill.Name {
		return domain.SkillEditResult{}, errors.New("managed skill name no longer matches its stored identity")
	}
	if current.ContentHash != input.ExpectedContentHash {
		return domain.SkillEditResult{}, errors.New("skill changed since the editor was opened; reload before saving")
	}
	oldRaw, err := os.ReadFile(path)
	if err != nil {
		return domain.SkillEditResult{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return domain.SkillEditResult{}, err
	}
	metadata := current.Metadata
	if skill.Source == domain.SkillSourceAivoSystem {
		metadata = make(map[string]string, len(current.Metadata)+1)
		for key, value := range current.Metadata {
			metadata[key] = value
		}
		metadata["aivo.user_modified"] = "true"
	}
	raw := marshalSkillMarkdown(skill.Name, input.Description, metadata, input.Content)
	if err := atomicReplaceFile(path, raw, info.Mode().Perm()); err != nil {
		return domain.SkillEditResult{}, err
	}

	updated, err := parseSkillDirectory(skill.RootPath)
	if err != nil {
		_ = atomicReplaceFile(path, oldRaw, info.Mode().Perm())
		return domain.SkillEditResult{}, err
	}
	saved, err := m.store.SaveSkill(ctx, domain.SkillEntry{
		ID: skill.ID, Name: updated.Name, Description: updated.Description, Scope: skill.Scope, Source: skill.Source,
		RootPath: updated.RootPath, SkillPath: updated.SkillPath, ContentHash: updated.ContentHash, Enabled: skill.Enabled,
		Metadata: updated.Metadata, TimeCreated: skill.TimeCreated, TimeUpdated: domain.NowString(time.Now()),
	})
	if err != nil {
		if rollbackErr := atomicReplaceFile(path, oldRaw, info.Mode().Perm()); rollbackErr != nil {
			return domain.SkillEditResult{}, fmt.Errorf("save skill metadata: %w; restore skill file: %v", err, rollbackErr)
		}
		return domain.SkillEditResult{}, err
	}
	return domain.SkillEditResult{Skill: saved, Content: updated.Content}, nil
}

func (m *SkillManager) skillEditSnapshot(ctx context.Context, skill domain.SkillEntry) (domain.SkillEditResult, error) {
	parsed, err := parseSkillDirectory(skill.RootPath)
	if err != nil {
		return domain.SkillEditResult{}, err
	}
	if parsed.Name != skill.Name {
		return domain.SkillEditResult{}, errors.New("managed skill name no longer matches its stored identity")
	}
	if parsed.ContentHash != skill.ContentHash || parsed.Description != skill.Description {
		skill, err = m.store.SaveSkill(ctx, domain.SkillEntry{
			ID: skill.ID, Name: parsed.Name, Description: parsed.Description, Scope: skill.Scope, Source: skill.Source,
			RootPath: parsed.RootPath, SkillPath: parsed.SkillPath, ContentHash: parsed.ContentHash, Enabled: skill.Enabled,
			Metadata: parsed.Metadata, TimeCreated: skill.TimeCreated, TimeUpdated: domain.NowString(time.Now()),
		})
		if err != nil {
			return domain.SkillEditResult{}, err
		}
	}
	return domain.SkillEditResult{Skill: skill, Content: parsed.Content}, nil
}

func (m *SkillManager) editableSkillPath(skill domain.SkillEntry) (string, error) {
	if isSystemSkillSource(skill.Source) {
		return "", errors.New("system skills are read-only and updated by the Host")
	}
	managedRoot := filepath.Join(m.home, ".aivo", "skills")
	root := filepath.Clean(strings.TrimSpace(skill.RootPath))
	path := filepath.Clean(strings.TrimSpace(skill.SkillPath))
	if skill.Source != domain.SkillSourceAivo || root == "." || path == "." || !skillPathWithin(root, managedRoot) {
		return "", errors.New("only Aivo-managed skills can be edited")
	}
	if !samePath(path, filepath.Join(root, "SKILL.md")) {
		return "", errors.New("managed skill path is invalid")
	}
	managedResolved, err := filepath.EvalSymlinks(managedRoot)
	if err != nil {
		return "", err
	}
	rootResolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	pathResolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	if !skillPathWithin(rootResolved, managedResolved) || !skillPathWithin(pathResolved, managedResolved) {
		return "", errors.New("managed skill path resolves outside Aivo storage")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", errors.New("managed SKILL.md must be a regular file")
	}
	return path, nil
}

func marshalSkillMarkdown(name string, description string, metadata map[string]string, content string) []byte {
	quotedDescription, _ := json.Marshal(strings.TrimSpace(description))
	var builder strings.Builder
	builder.WriteString("---\nname: ")
	builder.WriteString(name)
	builder.WriteString("\ndescription: ")
	builder.Write(quotedDescription)
	builder.WriteByte('\n')
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		if key != "name" && key != "description" && !strings.HasPrefix(key, "metadata.") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		value, _ := json.Marshal(metadata[key])
		builder.WriteString(key)
		builder.WriteString(": ")
		builder.Write(value)
		builder.WriteByte('\n')
	}
	nestedKeys := make([]string, 0)
	for key := range metadata {
		if strings.HasPrefix(key, "metadata.") {
			nestedKey := strings.TrimPrefix(key, "metadata.")
			if nestedKey != "" {
				nestedKeys = append(nestedKeys, nestedKey)
			}
		}
	}
	sort.Strings(nestedKeys)
	if len(nestedKeys) > 0 {
		builder.WriteString("metadata:\n")
		for _, key := range nestedKeys {
			value, _ := json.Marshal(metadata["metadata."+key])
			builder.WriteString("  ")
			builder.WriteString(key)
			builder.WriteString(": ")
			builder.Write(value)
			builder.WriteByte('\n')
		}
	}
	builder.WriteString("---\n")
	if strings.TrimSpace(content) != "" {
		builder.WriteByte('\n')
		builder.WriteString(strings.TrimSpace(content))
		builder.WriteByte('\n')
	}
	return []byte(builder.String())
}
