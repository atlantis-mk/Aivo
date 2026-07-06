package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

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
	filesystemNamespaceDescription = "Workspace file tools. Use read_file only when the exact file path is known, list_files to inspect one directory, glob to find files by filename/path pattern, search_files to find text inside files, and write/edit/patch tools only for requested changes."
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
		Description:          "Read one UTF-8 text file inside the current workspace. Use this when the exact file path is known and you need file contents, such as source code, README files, config files, or project documentation. Do not use this to list directories or discover filenames; use list_files, glob, or search_files first. The path must be relative to the workspace root. Binary files, sensitive files, directories, and paths outside the workspace are rejected. Large whole-file reads are truncated. For large files, pass offset and limit to read a line range; paged output uses LINE|CONTENT.",
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
		return domain.ToolResult{
			Name:         "read_file",
			OK:           true,
			Content:      content,
			ModelContent: content,
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
	return domain.ToolResult{
		Name:         "read_file",
		OK:           true,
		Content:      content,
		ModelContent: content,
		Structured:   map[string]any{"snapshot": snapshot},
	}
}

type ListFilesTool struct {
	workspaceRoot string
}

func NewListFilesTool(workspaceRoot string) *ListFilesTool {
	return &ListFilesTool{workspaceRoot: workspaceRoot}
}

func (t *ListFilesTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:                 "list_files",
		Description:          "List entries in one workspace directory. Use this to inspect project structure when you do not yet know which file to read. Do not use this when you already know the target file; use read_file. For filename/path pattern matching use glob, and for content search use search_files. The optional path must be relative to the workspace root. Heavy generated directories such as .git, node_modules, vendor, dist, build, .next, and target are ignored, and workspace .gitignore rules are respected.",
		Namespace:            filesystemNamespace,
		NamespaceDescription: filesystemNamespaceDescription,
		Capability:           "filesystem.list",
		RiskLevel:            "low",
		Category:             "filesystem",
		Toolsets:             []string{"safe", "coding"},
		RequiresWorkspace:    true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string", "description": "Optional directory path relative to the workspace root. Defaults to the workspace root. Must point to a directory, not a file."},
			},
		},
	}
}

func (t *ListFilesTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	var input struct {
		Path string `json:"path"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &input); err != nil {
			return toolError("list_files", err)
		}
	}
	workspaceRoot := toolWorkspaceRoot(t.workspaceRoot, execCtx)
	path, err := safeJoin(workspaceRoot, input.Path)
	if err != nil {
		return toolError("list_files", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return toolError("list_files", err)
	}
	if !info.IsDir() {
		return toolError("list_files", errors.New("path must be a directory"))
	}
	var files []string
	truncated := false
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
		return toolError("list_files", err)
	}
	sort.Strings(files)
	content := strings.Join(files, "\n")
	if truncated {
		content += fmt.Sprintf("\n\n[truncated: showing first %d files]", listFilesMaxResults)
	}
	return domain.ToolResult{Name: "list_files", OK: true, Content: content}
}

type GlobTool struct {
	workspaceRoot string
}

func NewGlobTool(workspaceRoot string) *GlobTool {
	return &GlobTool{workspaceRoot: workspaceRoot}
}

func (t *GlobTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:                 "glob",
		Description:          "Find workspace files by filename or path glob pattern. Use this when you know a path pattern, extension, or filename fragment, such as **/*.go, src/**/*.tsx, or **/README*. Do not use this to search file contents; use search_files. Do not use this to read file contents; use read_file after choosing a result. The optional path limits matching to one workspace-relative directory. Generated directories, sensitive files, and paths ignored by .gitignore are skipped.",
		Namespace:            filesystemNamespace,
		NamespaceDescription: filesystemNamespaceDescription,
		Capability:           "filesystem.glob",
		RiskLevel:            "low",
		Category:             "filesystem",
		Toolsets:             []string{"safe", "coding"},
		RequiresWorkspace:    true,
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
	var input struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolError("glob", err)
	}
	pattern := filepath.ToSlash(strings.TrimSpace(input.Pattern))
	if pattern == "" {
		return toolError("glob", errors.New("pattern is required"))
	}
	if filepath.IsAbs(pattern) || strings.HasPrefix(filepath.Clean(pattern), "../") || filepath.Clean(pattern) == ".." {
		return toolError("glob", errors.New("pattern must be relative"))
	}
	matcher, err := compileGlobMatcher(pattern)
	if err != nil {
		return toolError("glob", err)
	}
	workspaceRoot := toolWorkspaceRoot(t.workspaceRoot, execCtx)
	searchRoot, err := safeJoin(workspaceRoot, input.Path)
	if err != nil {
		return toolError("glob", err)
	}
	info, err := os.Stat(searchRoot)
	if err != nil {
		return toolError("glob", err)
	}
	if !info.IsDir() {
		return toolError("glob", errors.New("path must be a directory"))
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
		return toolError("glob", err)
	}
	sort.Strings(files)
	content := "No files found"
	if len(files) > 0 {
		content = strings.Join(files, "\n")
		if truncated {
			content += fmt.Sprintf("\n\n[truncated: showing first %d files]", globMaxResults)
		}
	}
	return domain.ToolResult{Name: "glob", OK: true, Content: content}
}

type SearchFilesTool struct {
	workspaceRoot string
}

func NewSearchFilesTool(workspaceRoot string) *SearchFilesTool {
	return &SearchFilesTool{workspaceRoot: workspaceRoot}
}

func (t *SearchFilesTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:                 "search_files",
		Description:          "Search for a plain-text query inside workspace text files. Use this to locate definitions, references, config keys, error messages, or documentation snippets before reading specific files. Do not use this to find filenames by pattern; use glob. Do not use this to list directories; use list_files. Use path to restrict search to one directory or file, fileGlob to restrict searched filenames, and limit to control result size. Query must be non-empty. Generated directories, binary files, sensitive files, and paths ignored by .gitignore are skipped.",
		Namespace:            filesystemNamespace,
		NamespaceDescription: filesystemNamespaceDescription,
		Capability:           "filesystem.search",
		RiskLevel:            "low",
		Category:             "filesystem",
		Toolsets:             []string{"safe", "coding"},
		RequiresWorkspace:    true,
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
	var input struct {
		Query    string `json:"query"`
		Path     string `json:"path"`
		FileGlob string `json:"fileGlob"`
		Limit    int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolError("search_files", err)
	}
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return toolError("search_files", errors.New("query is required"))
	}
	limit := input.Limit
	if limit <= 0 || limit > searchMaxMatches {
		limit = searchMaxMatches
	}
	workspaceRoot := toolWorkspaceRoot(t.workspaceRoot, execCtx)
	searchRoot, err := safeJoin(workspaceRoot, input.Path)
	if err != nil {
		return toolError("search_files", err)
	}
	info, err := os.Stat(searchRoot)
	if err != nil {
		return toolError("search_files", err)
	}
	var globMatcher *regexp.Regexp
	if strings.TrimSpace(input.FileGlob) != "" {
		pattern := filepath.ToSlash(strings.TrimSpace(input.FileGlob))
		if filepath.IsAbs(pattern) || strings.HasPrefix(filepath.Clean(pattern), "../") || filepath.Clean(pattern) == ".." {
			return toolError("search_files", errors.New("fileGlob must be relative"))
		}
		globMatcher, err = compileGlobMatcher(pattern)
		if err != nil {
			return toolError("search_files", err)
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
		return toolError("search_files", err)
	}
	content := strings.Join(matches, "\n")
	if len(matches) >= limit {
		content += fmt.Sprintf("\n\n[truncated: showing first %d matches]", limit)
	}
	return domain.ToolResult{Name: "search_files", OK: true, Content: content}
}

var errStopWalk = errors.New("stop walk")

func compileGlobMatcher(pattern string) (*regexp.Regexp, error) {
	pattern = filepath.ToSlash(strings.TrimSpace(pattern))
	expanded := expandGlobBraces(pattern)
	parts := make([]string, 0, len(expanded))
	for _, item := range expanded {
		parts = append(parts, globPatternRegex(item))
	}
	return regexp.Compile("^(?:" + strings.Join(parts, "|") + ")$")
}

func expandGlobBraces(pattern string) []string {
	start := strings.IndexByte(pattern, '{')
	if start < 0 {
		return []string{pattern}
	}
	end := strings.IndexByte(pattern[start+1:], '}')
	if end < 0 {
		return []string{pattern}
	}
	end += start + 1
	options := strings.Split(pattern[start+1:end], ",")
	if len(options) == 0 {
		return []string{pattern}
	}
	out := make([]string, 0, len(options))
	for _, option := range options {
		out = append(out, pattern[:start]+option+pattern[end+1:])
	}
	return out
}

func globPatternRegex(pattern string) string {
	var b strings.Builder
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				i++
				if i+1 < len(pattern) && pattern[i+1] == '/' {
					i++
					b.WriteString("(?:.*/)?")
				} else {
					b.WriteString(".*")
				}
				continue
			}
			b.WriteString("[^/]*")
		case '?':
			b.WriteString("[^/]")
		case '/', '.',
			'+', '(', ')', '|', '^', '$', '[', ']', '{', '}', '\\':
			b.WriteByte('\\')
			b.WriteByte(pattern[i])
		default:
			b.WriteByte(pattern[i])
		}
	}
	return b.String()
}

func safeJoin(workspaceRoot string, relPath string) (string, error) {
	root, err := filepath.Abs(strings.TrimSpace(workspaceRoot))
	if err != nil {
		return "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		relPath = "."
	}
	if filepath.IsAbs(relPath) {
		return "", errors.New("path must be relative to workspace root")
	}
	clean := filepath.Clean(relPath)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", errors.New("path escapes workspace root")
	}
	target := filepath.Join(root, clean)
	realTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		parent := filepath.Dir(target)
		realParent, parentErr := filepath.EvalSymlinks(parent)
		if parentErr != nil {
			return "", err
		}
		realTarget = filepath.Join(realParent, filepath.Base(target))
	}
	rel, err := filepath.Rel(root, realTarget)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", errors.New("path escapes workspace root")
	}
	return realTarget, nil
}

func isSensitiveRelPath(path string) bool {
	path = filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
	base := filepath.Base(path)
	if base == ".env" || strings.HasPrefix(base, ".env.") {
		return true
	}
	if path == ".git/config" || strings.HasPrefix(path, ".ssh/") || strings.Contains(path, "/.ssh/") {
		return true
	}
	lower := strings.ToLower(base)
	return strings.Contains(lower, "private_key") || strings.HasSuffix(lower, ".pem") || strings.HasSuffix(lower, ".key")
}

func readTextFileLimited(ctx context.Context, path string, maxChars int) (string, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	defer file.Close()
	var buf bytes.Buffer
	tmp := make([]byte, 4096)
	for buf.Len() <= maxChars {
		if err := ctx.Err(); err != nil {
			return "", false, err
		}
		n, readErr := file.Read(tmp)
		if n > 0 {
			buf.Write(tmp[:n])
			if bytes.IndexByte(tmp[:n], 0) >= 0 {
				return "", false, errors.New("refusing to read binary file")
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return "", false, readErr
		}
	}
	content := buf.String()
	if !utf8.ValidString(content) {
		return "", false, errors.New("refusing to read binary file")
	}
	if len(content) > maxChars {
		return content[:maxChars], true, nil
	}
	return content, false, nil
}

func readTextFileLines(ctx context.Context, path string, offset *int, limit *int) (string, bool, int, error) {
	start := 1
	if offset != nil {
		start = *offset
	}
	if start < 1 {
		return "", false, 0, errors.New("offset must be at least 1")
	}
	count := readFileDefaultLineLimit
	if limit != nil {
		count = *limit
	}
	if count < 1 {
		return "", false, 0, errors.New("limit must be at least 1")
	}
	if count > readFileMaxLines {
		return "", false, 0, fmt.Errorf("limit must be at most %d", readFileMaxLines)
	}
	file, err := os.Open(path)
	if err != nil {
		return "", false, 0, err
	}
	defer file.Close()
	probe := make([]byte, 1024)
	n, _ := file.Read(probe)
	if bytes.IndexByte(probe[:n], 0) >= 0 || !validUTF8Probe(probe[:n]) {
		return "", false, 0, errors.New("refusing to read binary file")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", false, 0, err
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var lines []string
	lineNo := 0
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return "", false, 0, err
		}
		lineNo++
		if lineNo < start {
			continue
		}
		if len(lines) >= count {
			return strings.Join(lines, "\n"), true, lineNo, nil
		}
		lines = append(lines, fmt.Sprintf("%d|%s", lineNo, scanner.Text()))
	}
	if err := scanner.Err(); err != nil {
		return "", false, 0, err
	}
	return strings.Join(lines, "\n"), false, 0, nil
}

func searchTextFile(path string, query string, remaining int) ([]string, error) {
	if remaining <= 0 {
		return nil, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	probe := make([]byte, 1024)
	n, _ := file.Read(probe)
	if bytes.IndexByte(probe[:n], 0) >= 0 || !validUTF8Probe(probe[:n]) {
		return nil, errors.New("binary file")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var matches []string
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		if strings.Contains(line, query) {
			matches = append(matches, fmt.Sprintf("%d:%s", lineNo, strings.TrimSpace(line)))
			if len(matches) >= remaining {
				break
			}
		}
	}
	return matches, scanner.Err()
}

func toolError(name string, err error) domain.ToolResult {
	message := err.Error()
	var stale staleFileError
	if errors.As(err, &stale) {
		return domain.ToolResult{
			Name:  name,
			OK:    false,
			Error: message,
			ToolError: &domain.ToolError{
				Code:    "stale_file",
				Message: message,
				Retry:   true,
			},
			Structured: map[string]any{
				"stale":        true,
				"path":         stale.Path,
				"expectedHash": stale.ExpectedHash,
				"currentHash":  stale.CurrentHash,
			},
		}
	}
	return domain.ToolResult{Name: name, OK: false, Error: message, ToolError: &domain.ToolError{Code: "tool_error", Message: message}}
}

func validUTF8Probe(sample []byte) bool {
	if utf8.Valid(sample) {
		return true
	}
	for trim := 1; trim < utf8.UTFMax && trim < len(sample); trim++ {
		if utf8.Valid(sample[:len(sample)-trim]) {
			return true
		}
	}
	return false
}

func mustRel(root string, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

func toolWorkspaceRoot(defaultRoot string, execCtx domain.ToolExecutionContext) string {
	if strings.TrimSpace(execCtx.WorkspaceRoot) != "" {
		return execCtx.WorkspaceRoot
	}
	return defaultRoot
}
