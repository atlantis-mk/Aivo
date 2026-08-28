package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	return fmt.Errorf("%s %s exceeds %d lines (%d lines)", toolName, fieldName, maxLines, lineCount)
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
