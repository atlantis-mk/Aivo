package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"aivo/core/domain"
)

const (
	SkillsReadToolName = "skills_read"
	SkillsListToolName = "skills_list"

	skillPackageNamespace    = "aivo"
	skillReadPageMaxBytes    = 512 * 1024
	skillResourcePathMaxSize = 4096
)

type SkillsReadTool struct {
	service *Service
}

type SkillsListTool struct {
	service *Service
}

func NewSkillsReadTool(service *Service) *SkillsReadTool {
	return &SkillsReadTool{service: service}
}

func NewSkillsListTool(service *Service) *SkillsListTool {
	return &SkillsListTool{service: service}
}

func (t *SkillsReadTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:        SkillsReadToolName,
		Description: "Read one page from a skill. Pass its provided package directly; root aliases are resolved automatically. Omit resource to read SKILL.md; to read another file, use the same package and pass the file's complete skill:// identifier as resource. If the package is not provided, use skills_list to find it. Pass next_cursor back as cursor to continue.",
		Kind:        domain.ToolKindJSON,
		InputSchema: map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"package":  map[string]any{"type": "string", "description": "Package locator shown in the Skills catalog."},
				"resource": map[string]any{"type": "string", "description": "Optional complete skill:// resource identifier inside the same package. Omit to read SKILL.md."},
				"cursor":   map[string]any{"type": "string", "description": "Optional next_cursor from a previous skills_read response."},
			},
			"required": []string{"package"},
		},
		Capability: "skill.read", Category: "skill", RiskLevel: "low", Toolsets: []string{"safe", "coding"},
	}
}

func (t *SkillsReadTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	if t == nil || t.service == nil {
		return toolFailure(execCtx.ToolCallID, SkillsReadToolName, "skill_service_unavailable", "skill service is unavailable")
	}
	var input struct {
		Package  string `json:"package"`
		Resource string `json:"resource"`
		Cursor   string `json:"cursor"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolFailure(execCtx.ToolCallID, SkillsReadToolName, "invalid_arguments", "invalid skills_read arguments")
	}
	skill, err := t.service.resolveSkillPackage(ctx, execCtx.SessionID, execCtx.WorkspaceRoot, input.Package)
	if err != nil {
		return toolFailure(execCtx.ToolCallID, SkillsReadToolName, "skill_package_unavailable", err.Error())
	}
	resource, path, err := resolveSkillResourcePath(skill, input.Package, input.Resource)
	if err != nil {
		return toolFailure(execCtx.ToolCallID, SkillsReadToolName, "skill_resource_unavailable", err.Error())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return toolFailure(execCtx.ToolCallID, SkillsReadToolName, "skill_read_failed", err.Error())
	}
	response, err := pagedSkillReadResponse(resource, string(data), strings.TrimSpace(input.Cursor))
	if err != nil {
		return toolFailure(execCtx.ToolCallID, SkillsReadToolName, "invalid_cursor", err.Error())
	}
	raw, _ := json.MarshalIndent(response, "", "  ")
	return domain.ToolResult{Name: SkillsReadToolName, CallID: execCtx.ToolCallID, OK: true, Content: string(raw), ModelContent: string(raw), Structured: response}
}

func (t *SkillsListTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:        SkillsListToolName,
		Description: "List skills owned by the requested authority. Returns each skill's authority, package, and main_resource. Pass the package to skills_read, and pass next_cursor back as cursor to continue.",
		Kind:        domain.ToolKindJSON,
		InputSchema: map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"authority": map[string]any{"type": "string", "enum": []string{"orchestrator"}, "description": "Skill authority to list."},
				"cursor":    map[string]any{"type": "string", "description": "Optional next_cursor from a previous skills_list response."},
			},
		},
		Capability: "skill.list", Category: "skill", RiskLevel: "low", Toolsets: []string{"safe", "coding"},
	}
}

func (t *SkillsListTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	if t == nil || t.service == nil {
		return toolFailure(execCtx.ToolCallID, SkillsListToolName, "skill_service_unavailable", "skill service is unavailable")
	}
	var input struct {
		Authority string `json:"authority"`
		Cursor    string `json:"cursor"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &input); err != nil {
			return toolFailure(execCtx.ToolCallID, SkillsListToolName, "invalid_arguments", "invalid skills_list arguments")
		}
	}
	authority := strings.TrimSpace(input.Authority)
	if authority == "" {
		authority = "orchestrator"
	}
	if authority != "orchestrator" {
		return toolFailure(execCtx.ToolCallID, SkillsListToolName, "unsupported_authority", "only orchestrator skills are available")
	}
	skills, err := t.service.listModelVisibleSkills(ctx, execCtx.SessionID, execCtx.WorkspaceRoot)
	if err != nil {
		return toolFailure(execCtx.ToolCallID, SkillsListToolName, "skill_catalog_failed", err.Error())
	}
	start, err := parseSkillCursor(strings.TrimSpace(input.Cursor), len(skills))
	if err != nil {
		return toolFailure(execCtx.ToolCallID, SkillsListToolName, "invalid_cursor", err.Error())
	}
	listed := make([]map[string]any, 0, len(skills)-start)
	for _, skill := range skills[start:] {
		listed = append(listed, map[string]any{
			"authority":     "orchestrator",
			"package":       skillPackageLocator(skill),
			"name":          skill.Name,
			"description":   skill.Description,
			"main_resource": skillMainResourceLocator(skill),
		})
	}
	response := map[string]any{"skills": listed, "warnings": []string{}, "next_cursor": nil}
	raw, _ := json.MarshalIndent(response, "", "  ")
	return domain.ToolResult{Name: SkillsListToolName, CallID: execCtx.ToolCallID, OK: true, Content: string(raw), ModelContent: string(raw), Structured: response}
}

func (s *Service) listModelVisibleSkills(ctx context.Context, sessionID string, workspaceRoot string) ([]domain.SkillEntry, error) {
	if strings.TrimSpace(sessionID) != "" {
		_, skills := s.visibleSkills(ctx, sessionID)
		return skills, nil
	}
	result, err := s.ListSkills(ctx, domain.SkillListInput{})
	if err != nil {
		return nil, err
	}
	byName := map[string]domain.SkillEntry{}
	for _, skill := range result.Entries {
		if isModelResolvableSkillEntry(skill) {
			byName[normalizeSkillName(skill.Name)] = skill
		}
	}
	skills := make([]domain.SkillEntry, 0, len(byName))
	for _, skill := range byName {
		skills = append(skills, skill)
	}
	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })
	return skills, nil
}

func (s *Service) resolveSkillPackage(ctx context.Context, sessionID string, workspaceRoot string, packageLocator string) (domain.SkillEntry, error) {
	packageLocator = strings.TrimSpace(packageLocator)
	if packageLocator == "" {
		return domain.SkillEntry{}, errors.New("package is required")
	}
	skills, err := s.listModelVisibleSkills(ctx, sessionID, workspaceRoot)
	if err != nil {
		return domain.SkillEntry{}, err
	}
	for _, skill := range skills {
		if packageLocator == skillPackageLocator(skill) || packageLocator == skill.Name || packageLocator == normalizeSkillName(skill.Name) {
			return skill, nil
		}
		if name, ok := skillNameFromPackageLocator(packageLocator); ok && name == normalizeSkillName(skill.Name) {
			return skill, nil
		}
	}
	return domain.SkillEntry{}, errors.New("skill package is not available")
}

func skillPackageLocator(skill domain.SkillEntry) string {
	return "skill://" + skillPackageNamespace + "/" + normalizeSkillName(skill.Name)
}

func skillMainResourceLocator(skill domain.SkillEntry) string {
	return skillPackageLocator(skill) + "/SKILL.md"
}

func skillNameFromPackageLocator(packageLocator string) (string, bool) {
	value := strings.TrimSpace(packageLocator)
	value = strings.TrimSuffix(value, "/")
	for _, prefix := range []string{"skill://" + skillPackageNamespace + "/", skillPackageNamespace + "/"} {
		if strings.HasPrefix(value, prefix) {
			name := strings.Trim(strings.TrimPrefix(value, prefix), "/")
			if name != "" && !strings.Contains(name, "/") {
				return normalizeSkillName(name), true
			}
		}
	}
	return "", false
}

func resolveSkillResourcePath(skill domain.SkillEntry, packageLocator string, resource string) (string, string, error) {
	root := strings.TrimSpace(skill.RootPath)
	if root == "" && strings.TrimSpace(skill.SkillPath) != "" {
		root = filepath.Dir(skill.SkillPath)
	}
	if root == "" {
		return "", "", errors.New("skill root is unavailable")
	}
	root = filepath.Clean(root)
	resource = strings.TrimSpace(resource)
	if resource == "" || resource == skillPackageLocator(skill) || resource == skillMainResourceLocator(skill) {
		path := strings.TrimSpace(skill.SkillPath)
		if path == "" {
			path = filepath.Join(root, "SKILL.md")
		}
		return skillMainResourceLocator(skill), filepath.Clean(path), nil
	}
	if len(resource) > skillResourcePathMaxSize {
		return "", "", errors.New("resource is too large")
	}
	packageRoot := strings.TrimSuffix(skillPackageLocator(skill), "/") + "/"
	if strings.HasPrefix(resource, "skill://") {
		if !strings.HasPrefix(resource, packageRoot) {
			return "", "", errors.New("resource is outside the requested skill package")
		}
		resource = strings.TrimPrefix(resource, packageRoot)
	} else if name, ok := skillNameFromPackageLocator(packageLocator); !ok || name != normalizeSkillName(skill.Name) {
		return "", "", errors.New("package does not match selected skill")
	}
	relative := filepath.Clean(filepath.FromSlash(strings.TrimLeft(resource, "/")))
	if relative == "." || filepath.IsAbs(relative) || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || relative == ".." {
		return "", "", errors.New("resource must stay inside the skill package")
	}
	path := filepath.Join(root, relative)
	cleanPath := filepath.Clean(path)
	if !skillPathWithin(cleanPath, root) {
		return "", "", errors.New("resource must stay inside the skill package")
	}
	return packageRoot + filepath.ToSlash(relative), cleanPath, nil
}

func pagedSkillReadResponse(resource string, contents string, cursor string) (map[string]any, error) {
	start, err := parseSkillCursor(cursor, len(contents))
	if err != nil {
		return nil, err
	}
	if !contentsIsBoundary(contents, start) {
		return nil, fmt.Errorf("cursor is invalid")
	}
	end := len(contents)
	if end-start > skillReadPageMaxBytes {
		end = start + skillReadPageMaxBytes
		for end > start && !contentsIsBoundary(contents, end) {
			end--
		}
	}
	var next any
	if end < len(contents) {
		next = strconv.Itoa(end)
	}
	return map[string]any{"resource": resource, "contents": contents[start:end], "next_cursor": next}, nil
}

func parseSkillCursor(cursor string, length int) (int, error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return 0, nil
	}
	start, err := strconv.Atoi(cursor)
	if err != nil || start < 0 || start > length {
		return 0, fmt.Errorf("cursor is invalid")
	}
	return start, nil
}

func contentsIsBoundary(value string, index int) bool {
	return index >= 0 && index <= len(value) && (index == len(value) || value[index]&0xC0 != 0x80)
}
