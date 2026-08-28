package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"aivo/core/domain"
)

const (
	readFileMaxChars               = 16000
	readFileMaxLines               = 2000
	readFileDefaultLineLimit       = 500
	listFilesMaxResults            = 500
	globMaxResults                 = 100
	searchMaxMatches               = 100
	filesystemNamespace            = "functions"
	filesystemNamespaceDescription = "Workspace file tools. Use read_file only when the exact file path is known, ls to inspect one directory, find to match files by filename/path pattern, grep to find text inside files, and write/edit/patch tools only for requested changes."
)

var ignoredToolDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "dist": true, "build": true, ".next": true, "target": true,
}

type ReadFileTool struct {
	workspaceRoot string
}

func NewReadFileTool(workspaceRoot string) *ReadFileTool {
	return &ReadFileTool{workspaceRoot: workspaceRoot}
}

func (t *ReadFileTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:                 "read_file",
		Description:          "Read one UTF-8 text file inside the current workspace. Use this when the exact file path is known and you need file contents, such as source code, README files, config files, or project documentation. Do not use this to list directories or discover filenames; use ls, find, or grep first. The path must be relative to the workspace root. Binary files, sensitive files, directories, and paths outside the workspace are rejected. Large whole-file reads are truncated. For large files, pass offset and limit to read a line range; paged output uses LINE|CONTENT.",
		Namespace:            filesystemNamespace,
		NamespaceDescription: filesystemNamespaceDescription,
		Capability:           "filesystem.read",
		RiskLevel:            "low",
		Category:             "filesystem",
		Toolsets:             []string{"safe", "coding"},
		RequiresWorkspace:    true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":   map[string]any{"type": "string", "description": "Exact file path relative to the workspace root. Must point to a text file, not a directory. Do not pass absolute paths or paths containing '..'."},
				"offset": map[string]any{"type": "integer", "description": "Optional 1-based line number to start reading from. When omitted, the tool returns the file from the beginning subject to the character limit.", "minimum": 1},
				"limit":  map[string]any{"type": "integer", "description": "Optional maximum number of lines to read when using offset/limit pagination. Defaults to 500 when offset is provided; max 2000.", "minimum": 1, "maximum": readFileMaxLines},
			},
			"required": []string{"path"},
		},
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	var input struct {
		Path   string `json:"path"`
		Offset *int   `json:"offset"`
		Limit  *int   `json:"limit"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolError("read_file", err)
	}
	path, err := safeJoin(toolWorkspaceRoot(t.workspaceRoot, execCtx), input.Path)
	if err != nil {
		return toolError("read_file", err)
	}
	if isSensitiveRelPath(input.Path) {
		return toolError("read_file", errors.New("refusing to read sensitive file"))
	}
	info, err := os.Stat(path)
	if err != nil {
		return toolError("read_file", err)
	}
	if info.IsDir() {
		return toolError("read_file", errors.New("refusing to read directory"))
	}
	if input.Offset != nil || input.Limit != nil {
		content, truncated, next, err := readTextFileLines(ctx, path, input.Offset, input.Limit)
		if err != nil {
			return toolError("read_file", err)
		}
		lineRange := lineRangeString(input.Offset, input.Limit, truncated, next)
		snapshot, _, snapErr := readFileSnapshot(input.Path, path, lineRange, truncated)
		if snapErr != nil {
			return toolError("read_file", snapErr)
		}
		if truncated {
			content += fmt.Sprintf("\n\n[truncated: call read_file again with offset %d to continue]", next)
		}
		modelContent := withNestedProjectInstructions(content, toolWorkspaceRoot(t.workspaceRoot, execCtx), path)
		return domain.ToolResult{
			Name:         "read_file",
			OK:           true,
			Content:      content,
			ModelContent: modelContent,
			Structured:   map[string]any{"snapshot": snapshot},
		}
	}
	content, truncated, err := readTextFileLimited(ctx, path, readFileMaxChars)
	if err != nil {
		return toolError("read_file", err)
	}
	snapshot, _, err := readFileSnapshot(input.Path, path, "all", truncated)
	if err != nil {
		return toolError("read_file", err)
	}
	if truncated {
		content += fmt.Sprintf("\n\n[truncated: file exceeded %d characters]", readFileMaxChars)
	}
	modelContent := withNestedProjectInstructions(content, toolWorkspaceRoot(t.workspaceRoot, execCtx), path)
	return domain.ToolResult{
		Name:         "read_file",
		OK:           true,
		Content:      content,
		ModelContent: modelContent,
		Structured:   map[string]any{"snapshot": snapshot},
	}
}

func withNestedProjectInstructions(content, workspaceRoot, targetPath string) string {
	instructions := nestedProjectInstructionsForTarget(workspaceRoot, targetPath)
	if instructions == "" {
		return content
	}
	return content + "\n\n<project_instructions_for_file>\n" + instructions + "\n</project_instructions_for_file>"
}

type ListFilesTool struct {
	workspaceRoot string
	environment   ExecutionEnvironment
}

func NewListFilesTool(workspaceRoot string) *ListFilesTool {
	return &ListFilesTool{workspaceRoot: workspaceRoot}
}

func (t *ListFilesTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:                 "ls",
		Description:          "List entries in one workspace directory. Use this to inspect project structure when you do not yet know which file to read. Do not use this when you already know the target file; use read. For filename/path pattern matching use find, and for content search use grep. The optional path must be relative to the workspace root. Hidden files and directories whose names start with . are skipped unless includeHidden is true. Heavy generated directories such as .git, node_modules, vendor, dist, build, .next, and target are ignored, and workspace .gitignore rules are respected.",
		Namespace:            filesystemNamespace,
		NamespaceDescription: filesystemNamespaceDescription,
		Capability:           "filesystem.list",
		RiskLevel:            "low",
		Category:             "filesystem",
		Toolsets:             []string{"safe", "coding"},
		RequiresWorkspace:    true,
		ImplementationHash:   executionEnvironmentHash(t.environment),
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":          map[string]any{"type": "string", "description": "Optional directory path relative to the workspace root. Defaults to the workspace root. Must point to a directory, not a file."},
				"includeHidden": map[string]any{"type": "boolean", "description": "Include files and directories whose names start with . Defaults to false. Generated directories such as .git are still ignored."},
			},
		},
	}
}

func (t *ListFilesTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	if t.environment != nil {
		return t.environment.ExecutePrimitive(ctx, "ls", args, execCtx)
	}
	var input struct {
		Path          string `json:"path"`
		IncludeHidden *bool  `json:"includeHidden"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &input); err != nil {
			return toolError("ls", err)
		}
	}
	workspaceRoot := toolWorkspaceRoot(t.workspaceRoot, execCtx)
	path, err := safeJoin(workspaceRoot, input.Path)
	if err != nil {
		return toolError("ls", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return toolError("ls", err)
	}
	if !info.IsDir() {
		return toolError("ls", errors.New("path must be a directory"))
	}
	var files []string
	truncated := false
	includeHidden := input.IncludeHidden != nil && *input.IncludeHidden
	ignore := loadWorkspaceIgnore(ctx, workspaceRoot)
	err = filepath.WalkDir(path, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if shouldSkipWorkspaceEntry(workspaceRoot, current, entry, ignore) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !includeHidden && shouldSkipHiddenListEntry(workspaceRoot, current, entry) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(workspaceRoot, current)
		if err != nil {
			return nil
		}
		files = append(files, filepath.ToSlash(rel))
		if len(files) >= listFilesMaxResults {
			truncated = true
			return errStopWalk
		}
		return nil
	})
	if errors.Is(err, errStopWalk) {
		err = nil
	}
	if err != nil {
		return toolError("ls", err)
	}
	sort.Strings(files)
	content := strings.Join(files, "\n")
	if truncated {
		content += fmt.Sprintf("\n\n[truncated: showing first %d files]", listFilesMaxResults)
	}
	return domain.ToolResult{Name: "ls", OK: true, Content: content}
}

func shouldSkipHiddenListEntry(workspaceRoot string, current string, entry os.DirEntry) bool {
	if current == workspaceRoot {
		return false
	}
	name := strings.TrimSpace(entry.Name())
	return strings.HasPrefix(name, ".") && name != "." && name != ".."
}

type GlobTool struct {
	workspaceRoot string
	environment   ExecutionEnvironment
}

func NewGlobTool(workspaceRoot string) *GlobTool {
	return &GlobTool{workspaceRoot: workspaceRoot}
}

func (t *GlobTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:                 "find",
		Description:          "Find workspace files by filename or path glob pattern. Use this when you know a path pattern, extension, or filename fragment, such as **/*.go, src/**/*.tsx, or **/README*. Do not use this to search file contents; use grep. Do not use this to read file contents; use read after choosing a result. The optional path limits matching to one workspace-relative directory. Generated directories, sensitive files, and paths ignored by .gitignore are skipped.",
		Namespace:            filesystemNamespace,
		NamespaceDescription: filesystemNamespaceDescription,
		Capability:           "filesystem.glob",
		RiskLevel:            "low",
		Category:             "filesystem",
		Toolsets:             []string{"safe", "coding"},
		RequiresWorkspace:    true,
		ImplementationHash:   executionEnvironmentHash(t.environment),
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{"type": "string", "description": "Relative glob pattern to match file paths under path, such as **/*.go, src/**/*.tsx, or **/README*. Must not be absolute or escape the workspace."},
				"path":    map[string]any{"type": "string", "description": "Optional directory path relative to the workspace root. Defaults to the workspace root and must point to a directory."},
			},
			"required": []string{"pattern"},
		},
	}
}

func (t *GlobTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	if t.environment != nil {
		return t.environment.ExecutePrimitive(ctx, "find", args, execCtx)
	}
	var input struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolError("find", err)
	}
	pattern := filepath.ToSlash(strings.TrimSpace(input.Pattern))
	if pattern == "" {
		return toolError("find", errors.New("pattern is required"))
	}
	if filepath.IsAbs(pattern) || strings.HasPrefix(filepath.Clean(pattern), "../") || filepath.Clean(pattern) == ".." {
		return toolError("find", errors.New("pattern must be relative"))
	}
	matcher, err := compileGlobMatcher(pattern)
	if err != nil {
		return toolError("find", err)
	}
	workspaceRoot := toolWorkspaceRoot(t.workspaceRoot, execCtx)
	searchRoot, err := safeJoin(workspaceRoot, input.Path)
	if err != nil {
		return toolError("find", err)
	}
	info, err := os.Stat(searchRoot)
	if err != nil {
		return toolError("find", err)
	}
	if !info.IsDir() {
		return toolError("find", errors.New("path must be a directory"))
	}
	var files []string
	truncated := false
	ignore := loadWorkspaceIgnore(ctx, workspaceRoot)
	err = filepath.WalkDir(searchRoot, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if shouldSkipWorkspaceEntry(workspaceRoot, current, entry, ignore) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		relToWorkspace := filepath.ToSlash(mustRel(workspaceRoot, current))
		if isSensitiveRelPath(relToWorkspace) {
			return nil
		}
		relToSearchRoot := filepath.ToSlash(mustRel(searchRoot, current))
		if !matcher.MatchString(relToSearchRoot) {
			return nil
		}
		files = append(files, relToWorkspace)
		if len(files) >= globMaxResults {
			truncated = true
			return errStopWalk
		}
		return nil
	})
	if errors.Is(err, errStopWalk) {
		err = nil
	}
	if err != nil {
		return toolError("find", err)
	}
	sort.Strings(files)
	content := "No files found"
	if len(files) > 0 {
		content = strings.Join(files, "\n")
		if truncated {
			content += fmt.Sprintf("\n\n[truncated: showing first %d files]", globMaxResults)
		}
	}
	return domain.ToolResult{Name: "find", OK: true, Content: content}
}

type SearchFilesTool struct {
	workspaceRoot string
	environment   ExecutionEnvironment
}

func NewSearchFilesTool(workspaceRoot string) *SearchFilesTool {
	return &SearchFilesTool{workspaceRoot: workspaceRoot}
}

func (t *SearchFilesTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:                 "grep",
		Description:          "Search for a plain-text query inside workspace text files. Use this to locate definitions, references, config keys, error messages, or documentation snippets before reading specific files. Do not use this to find filenames by pattern; use find. Do not use this to list directories; use ls. Use path to restrict search to one directory or file, fileGlob to restrict searched filenames, and limit to control result size. Query must be non-empty. Generated directories, binary files, sensitive files, and paths ignored by .gitignore are skipped.",
		Namespace:            filesystemNamespace,
		NamespaceDescription: filesystemNamespaceDescription,
		Capability:           "filesystem.search",
		RiskLevel:            "low",
		Category:             "filesystem",
		Toolsets:             []string{"safe", "coding"},
		RequiresWorkspace:    true,
		ImplementationHash:   executionEnvironmentHash(t.environment),
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query":    map[string]any{"type": "string", "description": "Non-empty literal text to search for inside files. This is a plain-text contains search, not a regular expression."},
				"path":     map[string]any{"type": "string", "description": "Optional file or directory path relative to the workspace root. Defaults to the whole workspace."},
				"fileGlob": map[string]any{"type": "string", "description": "Optional relative glob pattern for files to search, such as **/*.go, src/**/*.tsx, or **/README*. Matched against workspace-relative paths."},
				"limit":    map[string]any{"type": "integer", "description": "Maximum number of matches to return. Defaults to 100; max 100.", "minimum": 1, "maximum": searchMaxMatches},
			},
			"required": []string{"query"},
		},
	}
}

func (t *SearchFilesTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	if t.environment != nil {
		return t.environment.ExecutePrimitive(ctx, "grep", args, execCtx)
	}
	var input struct {
		Query    string `json:"query"`
		Path     string `json:"path"`
		FileGlob string `json:"fileGlob"`
		Limit    int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolError("grep", err)
	}
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return toolError("grep", errors.New("query is required"))
	}
	limit := input.Limit
	if limit <= 0 || limit > searchMaxMatches {
		limit = searchMaxMatches
	}
	workspaceRoot := toolWorkspaceRoot(t.workspaceRoot, execCtx)
	searchRoot, err := safeJoin(workspaceRoot, input.Path)
	if err != nil {
		return toolError("grep", err)
	}
	info, err := os.Stat(searchRoot)
	if err != nil {
		return toolError("grep", err)
	}
	var globMatcher *regexp.Regexp
	if strings.TrimSpace(input.FileGlob) != "" {
		pattern := filepath.ToSlash(strings.TrimSpace(input.FileGlob))
		if filepath.IsAbs(pattern) || strings.HasPrefix(filepath.Clean(pattern), "../") || filepath.Clean(pattern) == ".." {
			return toolError("grep", errors.New("fileGlob must be relative"))
		}
		globMatcher, err = compileGlobMatcher(pattern)
		if err != nil {
			return toolError("grep", err)
		}
	}
	var matches []string
	walkRoot := searchRoot
	if !info.IsDir() {
		walkRoot = filepath.Dir(searchRoot)
	}
	ignore := loadWorkspaceIgnore(ctx, workspaceRoot)
	err = filepath.WalkDir(walkRoot, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if shouldSkipWorkspaceEntry(workspaceRoot, current, entry, ignore) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if !info.IsDir() && current != searchRoot {
			return nil
		}
		if isSensitiveRelPath(mustRel(workspaceRoot, current)) {
			return nil
		}
		rel := filepath.ToSlash(mustRel(workspaceRoot, current))
		if globMatcher != nil && !globMatcher.MatchString(rel) {
			return nil
		}
		fileMatches, err := searchTextFile(current, query, limit-len(matches))
		if err != nil {
			return nil
		}
		for _, match := range fileMatches {
			matches = append(matches, rel+":"+match)
			if len(matches) >= limit {
				return errStopWalk
			}
		}
		return nil
	})
	if errors.Is(err, errStopWalk) {
		err = nil
	}
	if err != nil {
		return toolError("grep", err)
	}
	content := strings.Join(matches, "\n")
	if len(matches) >= limit {
		content += fmt.Sprintf("\n\n[truncated: showing first %d matches]", limit)
	}
	return domain.ToolResult{Name: "grep", OK: true, Content: content}
}
