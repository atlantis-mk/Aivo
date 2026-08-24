package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"aivo/core/domain"
)

func safeTargetForWrite(workspaceRoot string, relPath string) (string, error) {
	root, err := filepath.Abs(strings.TrimSpace(workspaceRoot))
	if err != nil {
		return "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	clean := filepath.Clean(strings.TrimSpace(relPath))
	if clean == "" || clean == "." {
		return "", errors.New("path is required")
	}
	if filepath.IsAbs(clean) {
		return "", workspaceRelativePathError()
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", errors.New("path escapes workspace root")
	}
	target := filepath.Join(root, clean)
	probe := target
	for {
		if _, err := os.Lstat(probe); err == nil {
			realProbe, err := filepath.EvalSymlinks(probe)
			if err != nil {
				return "", err
			}
			rel, err := filepath.Rel(root, realProbe)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
				return "", errors.New("path escapes workspace root")
			}
			break
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return "", errors.New("path escapes workspace root")
		}
		probe = parent
	}
	return target, nil
}

func gitCommandOutput(ctx context.Context, workspaceRoot string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = workspaceRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", errors.New(message)
	}
	return string(output), nil
}

func extractPatchText(args json.RawMessage) (string, error) {
	raw := strings.TrimSpace(string(args))
	if raw == "" || raw == "{}" {
		return "", errors.New("patchText is required")
	}
	if strings.HasPrefix(raw, "*** Begin Patch") {
		return raw, nil
	}
	var input struct {
		PatchText string `json:"patchText"`
	}
	if err := json.Unmarshal(args, &input); err == nil {
		if strings.TrimSpace(input.PatchText) == "" {
			return "", errors.New("patchText is required")
		}
		return input.PatchText, nil
	}
	return "", errors.New("invalid apply_patch arguments")
}

type writeFileInput struct {
	Path         string `json:"path"`
	Content      string `json:"content"`
	ExpectedHash string `json:"expectedHash"`
}

func parseWriteFileArgs(args json.RawMessage) (writeFileInput, error) {
	var input writeFileInput
	if err := json.Unmarshal(args, &input); err != nil {
		return input, errors.New("invalid write_file arguments")
	}
	if strings.TrimSpace(input.Path) == "" {
		return input, errors.New("path is required")
	}
	return input, nil
}

type editFileInput struct {
	Path         string `json:"path"`
	OldString    string `json:"oldString"`
	NewString    string `json:"newString"`
	ReplaceAll   bool   `json:"replaceAll"`
	ExpectedHash string `json:"expectedHash"`
	SnapshotID   string `json:"snapshotId"`
}

func parseEditFileArgs(args json.RawMessage) (editFileInput, error) {
	var input editFileInput
	if err := json.Unmarshal(args, &input); err != nil {
		return input, errors.New("invalid edit_file arguments")
	}
	if strings.TrimSpace(input.Path) == "" {
		return input, errors.New("path is required")
	}
	if input.OldString == "" {
		return input, errors.New("oldString must not be empty")
	}
	if input.OldString == input.NewString {
		return input, errors.New("oldString and newString must differ")
	}
	return input, nil
}

func patchChangePaths(changes []patchFileChange) []string {
	seen := map[string]bool{}
	for _, change := range changes {
		addPatchPath(seen, change.Path)
		if change.MovePath != "" {
			addPatchPath(seen, change.MovePath)
		}
	}
	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func patchApplySummary(changes []patchFileChange) string {
	lines := []string{"Success. Updated the following files:"}
	for _, change := range changes {
		path := change.Path
		prefix := "M"
		switch change.Type {
		case "add":
			prefix = "A"
		case "delete":
			prefix = "D"
		case "move":
			prefix = "R"
			path = change.Path + " -> " + change.MovePath
		}
		lines = append(lines, prefix+" "+path)
	}
	return strings.Join(lines, "\n")
}

func patchChangeResultFiles(workspaceRoot string, changes []patchFileChange) []domain.ToolResultFile {
	files := make([]domain.ToolResultFile, 0, len(changes))
	for _, change := range changes {
		files = append(files, domain.ToolResultFile{
			Path:         change.Path,
			FullPath:     fullWorkspacePath(workspaceRoot, change.Path),
			MovePath:     change.MovePath,
			MoveFullPath: fullWorkspacePath(workspaceRoot, change.MovePath),
			Type:         change.Type,
			Additions:    change.Additions,
			Deletions:    change.Deletions,
			Diff:         change.Diff,
			BaseHash:     change.BaseHash,
			CurrentHash:  change.CurrentHash,
		})
	}
	return files
}

func fullWorkspacePath(workspaceRoot string, relPath string) string {
	root := strings.TrimSpace(workspaceRoot)
	path := strings.TrimSpace(relPath)
	if root == "" || path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.ToSlash(filepath.Clean(path))
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		absRoot = filepath.Clean(root)
	}
	return filepath.ToSlash(filepath.Join(absRoot, filepath.FromSlash(cleanPatchPath(path))))
}

func requireTextLineLimit(toolName string, fieldName string, value string, maxLines int) error {
	lineCount := countContentLines(value)
	if lineCount <= maxLines {
		return nil
	}
	return fmt.Errorf("%s %s exceeds %d lines (%d lines); use apply_patch for long content", toolName, fieldName, maxLines, lineCount)
}

func countContentLines(value string) int {
	if value == "" {
		return 0
	}
	lines := strings.Count(value, "\n")
	if strings.HasSuffix(value, "\n") {
		return lines
	}
	return lines + 1
}
