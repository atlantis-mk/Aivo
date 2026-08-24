package app

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"aivo/core/domain"
)

var errStopWalk = errors.New("stop walk")

const (
	workspaceRelativePathDescription  = `Workspace file path relative to the active workspace root, for example "notes.txt" or "src/app.go". Never pass the workspace root or an absolute workspace path.`
	workspaceRelativePathErrorMessage = `path must be relative to the active workspace root; remove the workspace-root prefix (for example, use "notes.txt")`
)

func workspaceRelativePathError() error {
	return errors.New(workspaceRelativePathErrorMessage)
}

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
		return "", workspaceRelativePathError()
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
