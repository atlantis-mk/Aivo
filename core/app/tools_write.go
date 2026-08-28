package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"aivo/core/domain"
)

const (
	gitOutputMaxChars     = 20000
	maxDirectWriteLines   = 150
	maxDirectEditArgLines = 150
)

type WriteFileTool struct {
	workspaceRoot string
}

func NewWriteFileTool(workspaceRoot string) *WriteFileTool {
	return &WriteFileTool{workspaceRoot: workspaceRoot}
}

func (t *WriteFileTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:                 "write_file",
		Description:          "Create one new file or completely overwrite one existing file inside the current workspace. Hard limit: content must be 150 lines or fewer; calls over 150 lines are rejected. Use this only for short, complete file writes when the full final content is known. Do not use this for small targeted edits to an existing file; use edit_file. Creates parent directories automatically. This tool requires permission approval unless a saved approval covers the target path.",
		Namespace:            filesystemNamespace,
		NamespaceDescription: filesystemNamespaceDescription,
		Capability:           "filesystem.write",
		RiskLevel:            "high",
		Category:             "filesystem",
		Toolsets:             []string{"coding"},
		RequiresWorkspace:    true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":         map[string]any{"type": "string", "description": "Target file path relative to the workspace root. Parent directories are created automatically. Must not be absolute or escape the workspace."},
				"content":      map[string]any{"type": "string", "description": "Complete final file content to write. Existing file content is fully replaced. Maximum 150 lines."},
				"expectedHash": map[string]any{"type": "string", "description": "Optional sha256 from a previous read_file snapshot. When provided, write_file refuses to overwrite the file if it changed."},
			},
			"required": []string{"path", "content"},
		},
	}
}

func (t *WriteFileTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	input, err := parseWriteFileArgs(args)
	if err != nil {
		return toolError("write_file", err)
	}
	if err := requireTextLineLimit("write_file", "content", input.Content, maxDirectWriteLines); err != nil {
		return toolError("write_file", err)
	}
	workspaceRoot := toolWorkspaceRoot(t.workspaceRoot, execCtx)
	target, err := safeTargetForWrite(workspaceRoot, input.Path)
	if err != nil {
		return toolError("write_file", err)
	}
	if err := ctx.Err(); err != nil {
		return toolError("write_file", err)
	}
	existed := true
	if _, err := os.Stat(target); errors.Is(err, os.ErrNotExist) {
		existed = false
	} else if err != nil {
		return toolError("write_file", err)
	}
	oldText := ""
	baseHash := ""
	if existed {
		oldRaw, err := os.ReadFile(target)
		if err != nil {
			return toolError("write_file", err)
		}
		oldText = string(oldRaw)
		if snap, err := snapshotForBytes(input.Path, target, oldRaw, "all", false); err == nil {
			baseHash = snap.SHA256
		}
	}
	expectedHash := firstNonEmpty(input.ExpectedHash, preparedExpectedHash(execCtx.ToolCallID, input.Path))
	if err := writeFileIfUnchanged(target, input.Path, expectedHash, []byte(input.Content), 0o600); err != nil {
		return toolError("write_file", err)
	}
	currentHash, _, _ := fileHashIfExists(target)
	action := "Wrote"
	if !existed {
		action = "Created"
	}
	changeType := "write"
	if !existed {
		changeType = "add"
	}
	additions, deletions := countLineDelta(oldText, input.Content)
	file := domain.ToolResultFile{
		Path:        cleanPatchPath(input.Path),
		FullPath:    fullWorkspacePath(workspaceRoot, input.Path),
		Type:        changeType,
		Additions:   additions,
		Deletions:   deletions,
		Diff:        simpleFileDiff(cleanPatchPath(input.Path), cleanPatchPath(input.Path), oldText, input.Content),
		BaseHash:    baseHash,
		CurrentHash: currentHash,
	}
	return domain.ToolResult{
		Name:       "write_file",
		OK:         true,
		Content:    fmt.Sprintf("%s file: %s", action, cleanPatchPath(input.Path)),
		Files:      []domain.ToolResultFile{file},
		Structured: map[string]any{"files": []domain.ToolResultFile{file}},
	}
}

type EditFileTool struct {
	workspaceRoot string
}

func NewEditFileTool(workspaceRoot string) *EditFileTool {
	return &EditFileTool{workspaceRoot: workspaceRoot}
}

func (t *EditFileTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:                 "edit_file",
		Description:          "Replace exact text in one existing workspace file. Hard limit: oldString and newString must each be 150 lines or fewer; calls over 150 lines are rejected. Use this only for focused single-file edits after reading the relevant current file content. When read_file returned a snapshot for this file, pass expectedHash so Aivo can edit without rereading and can reject stale writes safely. oldString must match the file exactly and must include enough surrounding context to be unique. If oldString appears multiple times, either provide more context or set replaceAll=true only when every occurrence should change. Do not use this to create files; use write_file. This tool requires permission approval unless a saved approval covers the target path.",
		Namespace:            filesystemNamespace,
		NamespaceDescription: filesystemNamespaceDescription,
		Capability:           "filesystem.write",
		RiskLevel:            "high",
		Category:             "filesystem",
		Toolsets:             []string{"coding"},
		RequiresWorkspace:    true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":         map[string]any{"type": "string", "description": "Existing file path relative to the workspace root. Must not be absolute or escape the workspace."},
				"oldString":    map[string]any{"type": "string", "description": "Exact text currently in the file. Must not be empty. Include surrounding lines when needed so the match is unique. Maximum 150 lines."},
				"newString":    map[string]any{"type": "string", "description": "Replacement text. Use an empty string to delete the matched text. Must differ from oldString. Maximum 150 lines."},
				"replaceAll":   map[string]any{"type": "boolean", "description": "Replace every exact occurrence. Defaults to false; set true only when all occurrences are intentionally identical edits."},
				"expectedHash": map[string]any{"type": "string", "description": "Optional sha256 from read_file.snapshot.sha256. When provided, edit_file refuses to write if the file changed since that snapshot."},
				"snapshotId":   map[string]any{"type": "string", "description": "Optional read_file snapshotId for traceability."},
			},
			"required": []string{"path", "oldString", "newString"},
		},
	}
}

func (t *EditFileTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	input, err := parseEditFileArgs(args)
	if err != nil {
		return toolError("edit_file", err)
	}
	if err := requireTextLineLimit("edit_file", "oldString", input.OldString, maxDirectEditArgLines); err != nil {
		return toolError("edit_file", err)
	}
	if err := requireTextLineLimit("edit_file", "newString", input.NewString, maxDirectEditArgLines); err != nil {
		return toolError("edit_file", err)
	}
	workspaceRoot := toolWorkspaceRoot(t.workspaceRoot, execCtx)
	target, err := safeTargetForWrite(workspaceRoot, input.Path)
	if err != nil {
		return toolError("edit_file", err)
	}
	oldRaw, err := os.ReadFile(target)
	if err != nil {
		return toolError("edit_file", fmt.Errorf("failed to read file: %s", cleanPatchPath(input.Path)))
	}
	baseHash := ""
	if snapshot, err := snapshotForBytes(input.Path, target, oldRaw, "all", false); err == nil {
		baseHash = snapshot.SHA256
	}
	expectedHash := firstNonEmpty(input.ExpectedHash, preparedExpectedHash(execCtx.ToolCallID, input.Path))
	if strings.TrimSpace(expectedHash) != "" && expectedHash != baseHash {
		return toolError("edit_file", staleFileError{Path: cleanPatchPath(input.Path), ExpectedHash: expectedHash, CurrentHash: baseHash})
	}
	oldText := string(oldRaw)
	replacements := strings.Count(oldText, input.OldString)
	if replacements == 0 {
		return toolError("edit_file", errors.New("oldString was not found in the file"))
	}
	if replacements > 1 && !input.ReplaceAll {
		return toolError("edit_file", errors.New("oldString appears multiple times; provide more context or set replaceAll=true"))
	}
	newText := strings.Replace(oldText, input.OldString, input.NewString, 1)
	if input.ReplaceAll {
		newText = strings.ReplaceAll(oldText, input.OldString, input.NewString)
	}
	if err := ctx.Err(); err != nil {
		return toolError("edit_file", err)
	}
	if err := writeFileIfUnchanged(target, input.Path, firstNonEmpty(expectedHash, baseHash), []byte(newText), 0o600); err != nil {
		return toolError("edit_file", err)
	}
	currentHash, _, _ := fileHashIfExists(target)
	additions, deletions := countLineDelta(oldText, newText)
	diff := simpleFileDiff(cleanPatchPath(input.Path), cleanPatchPath(input.Path), oldText, newText)
	file := domain.ToolResultFile{
		Path:        cleanPatchPath(input.Path),
		FullPath:    fullWorkspacePath(workspaceRoot, input.Path),
		Type:        "edit",
		Additions:   additions,
		Deletions:   deletions,
		Diff:        diff,
		BaseHash:    baseHash,
		CurrentHash: currentHash,
	}
	return domain.ToolResult{
		Name:    "edit_file",
		OK:      true,
		Content: fmt.Sprintf("Edited file: %s\nReplacements: %d", cleanPatchPath(input.Path), replacements),
		Files:   []domain.ToolResultFile{file},
		Structured: map[string]any{
			"snapshotId": input.SnapshotID,
			"files":      []domain.ToolResultFile{file},
		},
	}
}

type GitStatusTool struct {
	workspaceRoot string
}

func NewGitStatusTool(workspaceRoot string) *GitStatusTool {
	return &GitStatusTool{workspaceRoot: workspaceRoot}
}

func (t *GitStatusTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:                 "git_status",
		Description:          "Show read-only git status for the current workspace using `git status --short --branch`. This tool is only injected when the workspace is inside an initialized Git work tree. Use it to check branch and changed-file state before or after edits. It does not stage, commit, or modify files.",
		Namespace:            filesystemNamespace,
		NamespaceDescription: filesystemNamespaceDescription,
		Capability:           "git.read",
		RiskLevel:            "low",
		Category:             "git",
		Toolsets:             []string{"safe", "coding", "git"},
		RequiresWorkspace:    true,
		InputSchema: map[string]any{
			"type":        "object",
			"description": "No parameters.",
			"properties":  map[string]any{},
		},
	}
}

func (t *GitStatusTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	root := toolWorkspaceRoot(t.workspaceRoot, execCtx)
	out, err := gitCommandOutput(ctx, root, "status", "--short", "--branch")
	if err != nil {
		return toolError("git_status", err)
	}
	if strings.TrimSpace(out) == "" {
		out = "working tree clean"
	}
	return domain.ToolResult{Name: "git_status", OK: true, Content: bounded(out, gitOutputMaxChars)}
}

type GitDiffTool struct {
	workspaceRoot string
}

func NewGitDiffTool(workspaceRoot string) *GitDiffTool {
	return &GitDiffTool{workspaceRoot: workspaceRoot}
}

func (t *GitDiffTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:                 "git_diff",
		Description:          "Show the current unstaged git diff for the workspace. This tool is only injected when the workspace is inside an initialized Git work tree. Use it to inspect local file changes; it does not show staged-only changes and does not modify files. Optional path filters must be relative to the workspace root.",
		Namespace:            filesystemNamespace,
		NamespaceDescription: filesystemNamespaceDescription,
		Capability:           "git.read",
		RiskLevel:            "low",
		Category:             "git",
		Toolsets:             []string{"safe", "coding", "git"},
		RequiresWorkspace:    true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string", "description": "Optional file or directory path filter relative to the workspace root. Must not be absolute or escape the workspace."},
			},
		},
	}
}

func (t *GitDiffTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	var input struct {
		Path string `json:"path"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &input); err != nil {
			return toolError("git_diff", err)
		}
	}
	root := toolWorkspaceRoot(t.workspaceRoot, execCtx)
	gitArgs := []string{"diff", "--"}
	if strings.TrimSpace(input.Path) != "" {
		if _, err := safeJoin(root, input.Path); err != nil {
			return toolError("git_diff", err)
		}
		gitArgs = append(gitArgs, input.Path)
	}
	out, err := gitCommandOutput(ctx, root, gitArgs...)
	if err != nil {
		return toolError("git_diff", err)
	}
	if strings.TrimSpace(out) == "" {
		out = "no unstaged diff"
	}
	truncated := len(out) > gitOutputMaxChars
	content := bounded(out, gitOutputMaxChars)
	if truncated {
		content += fmt.Sprintf("\n\n[truncated: output exceeded %d characters]", gitOutputMaxChars)
	}
	return domain.ToolResult{Name: "git_diff", OK: true, Content: content, Truncated: truncated, OriginalSize: len(out)}
}
