package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type patchOperationType string

const (
	patchAdd    patchOperationType = "add"
	patchUpdate patchOperationType = "update"
	patchDelete patchOperationType = "delete"
)

type patchLine struct {
	prefix byte
	text   string
}

type patchHunk struct {
	lines []patchLine
}

type patchOperation struct {
	op       patchOperationType
	path     string
	movePath string
	hunks    []patchHunk
}

type patchFileChange struct {
	Path        string         `json:"path"`
	MovePath    string         `json:"movePath,omitempty"`
	Type        string         `json:"type"`
	Additions   int            `json:"additions"`
	Deletions   int            `json:"deletions"`
	Diff        string         `json:"diff,omitempty"`
	BaseHash    string         `json:"baseHash,omitempty"`
	CurrentHash string         `json:"currentHash,omitempty"`
	op          patchOperation `json:"-"`
}

func parseApplyPatchText(raw string) ([]patchOperation, error) {
	lines := strings.Split(strings.ReplaceAll(strings.ReplaceAll(raw, "\r\n", "\n"), "\r", "\n"), "\n")
	start := -1
	end := -1
	for i, line := range lines {
		switch strings.TrimSpace(line) {
		case "*** Begin Patch":
			start = i
		case "*** End Patch":
			end = i
			break
		}
		if end >= 0 {
			break
		}
	}
	if start < 0 || end < 0 || end <= start {
		return nil, errors.New("patch must include *** Begin Patch and *** End Patch")
	}
	var ops []patchOperation
	var current *patchOperation
	var hunk *patchHunk
	flushHunk := func() {
		if current != nil && hunk != nil {
			current.hunks = append(current.hunks, *hunk)
		}
		hunk = nil
	}
	flushOp := func() {
		if current != nil {
			flushHunk()
			ops = append(ops, *current)
		}
		current = nil
	}
	for i := start + 1; i < end; i++ {
		line := lines[i]
		switch {
		case strings.HasPrefix(line, "*** Add File: "):
			flushOp()
			current = &patchOperation{op: patchAdd, path: cleanPatchPath(strings.TrimPrefix(line, "*** Add File: "))}
			hunk = &patchHunk{}
		case strings.HasPrefix(line, "*** Update File: "):
			flushOp()
			current = &patchOperation{op: patchUpdate, path: cleanPatchPath(strings.TrimPrefix(line, "*** Update File: "))}
		case strings.HasPrefix(line, "*** Delete File: "):
			flushOp()
			ops = append(ops, patchOperation{op: patchDelete, path: cleanPatchPath(strings.TrimPrefix(line, "*** Delete File: "))})
		case strings.HasPrefix(line, "*** Move to: "):
			if current == nil || current.op != patchUpdate {
				return nil, errors.New("move header must follow an update file header")
			}
			current.movePath = cleanPatchPath(strings.TrimPrefix(line, "*** Move to: "))
		case strings.HasPrefix(line, "@@"):
			if current == nil {
				return nil, errors.New("hunk appears before a file operation")
			}
			flushHunk()
			hunk = &patchHunk{}
		default:
			if current == nil {
				if strings.TrimSpace(line) == "" {
					continue
				}
				return nil, fmt.Errorf("unexpected patch line outside file operation: %s", line)
			}
			if hunk == nil {
				hunk = &patchHunk{}
			}
			if line == `\ No newline at end of file` {
				continue
			}
			if line == "" {
				hunk.lines = append(hunk.lines, patchLine{prefix: ' ', text: ""})
				continue
			}
			switch line[0] {
			case '+', '-', ' ':
				hunk.lines = append(hunk.lines, patchLine{prefix: line[0], text: line[1:]})
			default:
				hunk.lines = append(hunk.lines, patchLine{prefix: ' ', text: line})
			}
		}
	}
	flushOp()
	if len(ops) == 0 {
		return nil, errors.New("patch rejected: empty patch")
	}
	for _, op := range ops {
		if op.path == "" || op.path == "." {
			return nil, errors.New("patch operation has empty path")
		}
		if op.op != patchDelete && len(op.hunks) == 0 {
			return nil, fmt.Errorf("%s %s has no hunks", op.op, op.path)
		}
	}
	return ops, nil
}

func buildPatchChanges(workspaceRoot string, patchText string) ([]patchFileChange, error) {
	ops, err := parseApplyPatchText(patchText)
	if err != nil {
		return nil, err
	}
	changes := make([]patchFileChange, 0, len(ops))
	for _, op := range ops {
		if _, err := safeTargetForWrite(workspaceRoot, op.path); err != nil {
			return nil, err
		}
		if op.movePath != "" {
			if _, err := safeTargetForWrite(workspaceRoot, op.movePath); err != nil {
				return nil, err
			}
		}
		target := op.path
		if op.movePath != "" {
			target = op.movePath
		}
		changeType := string(op.op)
		if op.movePath != "" {
			changeType = "move"
		}
		change := patchFileChange{Path: op.path, MovePath: op.movePath, Type: changeType, op: op}
		var oldText, newText string
		switch op.op {
		case patchAdd:
			newText = addFileText(op)
			targetPath := filepath.Join(workspaceRoot, filepath.FromSlash(op.path))
			if hash, exists, hashErr := fileHashIfExists(targetPath); hashErr == nil && exists {
				change.CurrentHash = hash
			}
		case patchUpdate:
			sourcePath := filepath.Join(workspaceRoot, filepath.FromSlash(op.path))
			raw, readErr := os.ReadFile(sourcePath)
			if readErr != nil {
				err = readErr
			}
			oldText = string(raw)
			if err != nil {
				return nil, fmt.Errorf("failed to read file to update: %s", op.path)
			}
			if snapshot, snapErr := snapshotForBytes(op.path, sourcePath, raw, "all", false); snapErr == nil {
				change.BaseHash = snapshot.SHA256
				change.CurrentHash = snapshot.SHA256
			}
			newText, err = applyUpdateHunks(oldText, op.hunks)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", op.path, err)
			}
		case patchDelete:
			sourcePath := filepath.Join(workspaceRoot, filepath.FromSlash(op.path))
			raw, readErr := os.ReadFile(sourcePath)
			if readErr != nil {
				err = readErr
			}
			oldText = string(raw)
			if err != nil {
				return nil, fmt.Errorf("failed to read file to delete: %s", op.path)
			}
			if snapshot, snapErr := snapshotForBytes(op.path, sourcePath, raw, "all", false); snapErr == nil {
				change.BaseHash = snapshot.SHA256
				change.CurrentHash = snapshot.SHA256
			}
		}
		change.Additions, change.Deletions = countLineDelta(oldText, newText)
		change.Diff = simpleFileDiff(op.path, target, oldText, newText)
		changes = append(changes, change)
	}
	return changes, nil
}

func applyPatchChanges(workspaceRoot string, changes []patchFileChange) error {
	for _, change := range changes {
		op := change.op
		switch op.op {
		case patchAdd:
			target := filepath.Join(workspaceRoot, filepath.FromSlash(op.path))
			if _, err := os.Stat(target); err == nil {
				return fmt.Errorf("file already exists: %s", op.path)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
				return err
			}
			if err := writeFileIfUnchanged(target, op.path, "<missing>", []byte(addFileText(op)), 0o600); err != nil {
				return err
			}
		case patchUpdate:
			source := filepath.Join(workspaceRoot, filepath.FromSlash(op.path))
			oldRaw, err := os.ReadFile(source)
			if err != nil {
				return fmt.Errorf("failed to read file to update: %s", op.path)
			}
			next, err := applyUpdateHunks(string(oldRaw), op.hunks)
			if err != nil {
				return fmt.Errorf("%s: %w", op.path, err)
			}
			target := source
			if op.movePath != "" {
				target = filepath.Join(workspaceRoot, filepath.FromSlash(op.movePath))
			}
			expectedHash := change.BaseHash
			if op.movePath != "" {
				expectedHash = ""
			}
			if err := writeFileIfUnchanged(target, firstNonEmpty(op.movePath, op.path), expectedHash, []byte(next), 0o600); err != nil {
				return err
			}
			if op.movePath != "" {
				if err := removeFileIfUnchanged(source, op.path, change.BaseHash); err != nil {
					return err
				}
			}
		case patchDelete:
			if err := removeFileIfUnchanged(filepath.Join(workspaceRoot, filepath.FromSlash(op.path)), op.path, change.BaseHash); err != nil {
				return fmt.Errorf("failed to delete file: %s", op.path)
			}
		}
	}
	return nil
}

func addFileText(op patchOperation) string {
	var lines []string
	for _, hunk := range op.hunks {
		for _, line := range hunk.lines {
			if line.prefix != '+' {
				continue
			}
			lines = append(lines, line.text)
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func applyUpdateHunks(source string, hunks []patchHunk) (string, error) {
	current := source
	for _, hunk := range hunks {
		var oldLines []string
		var newLines []string
		for _, line := range hunk.lines {
			if line.prefix == ' ' || line.prefix == '-' {
				oldLines = append(oldLines, line.text)
			}
			if line.prefix == ' ' || line.prefix == '+' {
				newLines = append(newLines, line.text)
			}
		}
		oldBlock := strings.Join(oldLines, "\n")
		newBlock := strings.Join(newLines, "\n")
		if strings.Contains(current, oldBlock+"\n") {
			current = strings.Replace(current, oldBlock+"\n", newBlock+"\n", 1)
			continue
		}
		if strings.Contains(current, oldBlock) {
			current = strings.Replace(current, oldBlock, newBlock, 1)
			continue
		}
		return "", errors.New("hunk did not match file content")
	}
	return current, nil
}

func countLineDelta(oldText string, newText string) (int, int) {
	oldLines := splitComparableLines(oldText)
	newLines := splitComparableLines(newText)
	oldCounts := map[string]int{}
	for _, line := range oldLines {
		oldCounts[line]++
	}
	additions := 0
	for _, line := range newLines {
		if oldCounts[line] > 0 {
			oldCounts[line]--
		} else {
			additions++
		}
	}
	newCounts := map[string]int{}
	for _, line := range newLines {
		newCounts[line]++
	}
	deletions := 0
	for _, line := range oldLines {
		if newCounts[line] > 0 {
			newCounts[line]--
		} else {
			deletions++
		}
	}
	return additions, deletions
}

func splitComparableLines(text string) []string {
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

func simpleFileDiff(oldPath string, newPath string, oldText string, newText string) string {
	var b strings.Builder
	b.WriteString("--- " + oldPath + "\n")
	b.WriteString("+++ " + newPath + "\n")
	for _, line := range splitComparableLines(oldText) {
		b.WriteString("-" + line + "\n")
	}
	for _, line := range splitComparableLines(newText) {
		b.WriteString("+" + line + "\n")
	}
	return b.String()
}

func patchTouchedPathsFromText(patchText string) ([]string, error) {
	ops, err := parseApplyPatchText(patchText)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, op := range ops {
		addPatchPath(seen, op.path)
		if op.movePath != "" {
			addPatchPath(seen, op.movePath)
		}
	}
	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func cleanPatchPath(path string) string {
	return filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
}
