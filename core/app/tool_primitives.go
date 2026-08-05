package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"aivo/core/domain"
)

const (
	primitiveImageMaxBytes  = 20 << 20
	primitiveImageMaxPixels = 40_000_000
	primitiveImageMaxSide   = 2048
	primitiveModelImageMax  = 8 << 20
	primitiveDiffMaxChars   = 20_000
)

type ReadTool struct {
	workspaceRoot string
	environment   ExecutionEnvironment
}

func NewReadTool(workspaceRoot string) *ReadTool { return &ReadTool{workspaceRoot: workspaceRoot} }

func (t *ReadTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name: "read", Description: "Read exactly one workspace-relative text or supported image file from the active execution environment. Text offsets are 1-based and bounded; this tool does not list directories, search, or mutate. An absolute path is valid only when it is an exact retained-output reference returned by Aivo.",
		Capability: "filesystem.read", RiskLevel: "low", Category: "filesystem", Toolsets: []string{"safe", "coding"}, RequiresWorkspace: true,
		ImplementationHash: executionEnvironmentHash(t.environment),
		InputSchema: map[string]any{
			"type": "object", "additionalProperties": false,
			"properties": map[string]any{
				"path":   map[string]any{"type": "string", "minLength": 1, "description": workspaceRelativePathDescription + " An absolute path is valid only for an exact retained-output reference returned by Aivo."},
				"offset": map[string]any{"type": "integer", "minimum": 1},
				"limit":  map[string]any{"type": "integer", "minimum": 1, "maximum": readFileMaxLines},
			},
			"required": []string{"path"},
		},
	}
}

func (t *ReadTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	if t.environment != nil {
		return t.environment.ExecutePrimitive(ctx, "read", args, execCtx)
	}
	var input struct {
		Path   string `json:"path"`
		Offset *int   `json:"offset"`
		Limit  *int   `json:"limit"`
	}
	if err := decodeStrictToolArgs(args, &input); err != nil {
		return primitiveError("read", "invalid_arguments", err)
	}
	if strings.TrimSpace(input.Path) == "" {
		return primitiveError("read", "invalid_arguments", errors.New("path is required"))
	}
	root := toolWorkspaceRoot(t.workspaceRoot, execCtx)
	target, err := primitiveExistingPath(root, input.Path)
	if err != nil {
		return primitiveError("read", "invalid_path", err)
	}
	if isSensitiveRelPath(input.Path) {
		return primitiveError("read", "sensitive_file", errors.New("refusing to read sensitive file"))
	}
	info, err := os.Stat(target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return primitiveError("read", "not_found", err)
		}
		return primitiveError("read", "read_failed", err)
	}
	if !info.Mode().IsRegular() {
		return primitiveError("read", "not_file", errors.New("path is not a regular file"))
	}
	if looksLikeSupportedImage(target) {
		if input.Offset != nil || input.Limit != nil {
			return primitiveError("read", "invalid_arguments", errors.New("offset and limit are not valid for images"))
		}
		return readPrimitiveImage(ctx, input.Path, target, info)
	}
	return readPrimitiveText(ctx, root, input.Path, target, input.Offset, input.Limit)
}

func readPrimitiveText(ctx context.Context, root, requested, target string, offset, limit *int) domain.ToolResult {
	var content string
	var truncated bool
	var next int
	var err error
	if offset != nil || limit != nil {
		content, truncated, next, err = readTextFileLines(ctx, target, offset, limit)
	} else {
		content, truncated, err = readTextFileLimited(ctx, target, readFileMaxChars)
	}
	if err != nil {
		return primitiveError("read", "unsupported_binary", err)
	}
	if truncated {
		if next > 0 {
			content += fmt.Sprintf("\n\n[truncated: continue with offset %d]", next)
		} else {
			content += fmt.Sprintf("\n\n[truncated: text exceeded %d characters]", readFileMaxChars)
		}
	}
	snapshot, snapErr := snapshotForFile(requested, target, lineRangeString(offset, limit, truncated, next), truncated)
	if snapErr != nil {
		return primitiveError("read", "read_failed", snapErr)
	}
	modelContent := withNestedProjectInstructions(content, root, target)
	return domain.ToolResult{
		Name: "read", OK: true, Content: content, ModelContent: modelContent, Truncated: truncated,
		Structured: map[string]any{"kind": "text", "path": canonicalEnvironmentPath(root, target), "nextOffset": next, "snapshot": snapshot},
	}
}

func readPrimitiveImage(ctx context.Context, requested, target string, info os.FileInfo) domain.ToolResult {
	if info.Size() > primitiveImageMaxBytes {
		return primitiveError("read", "image_too_large", fmt.Errorf("image exceeds %d bytes", primitiveImageMaxBytes))
	}
	if err := ctx.Err(); err != nil {
		return primitiveError("read", "cancelled", err)
	}
	raw, err := os.ReadFile(target)
	if err != nil {
		return primitiveError("read", "read_failed", err)
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return primitiveError("read", "unsupported_image", err)
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > primitiveImageMaxPixels {
		return primitiveError("read", "image_too_large", errors.New("decoded image dimensions exceed the pixel limit"))
	}
	mimeType := "image/" + format
	modelRaw := raw
	modelWidth, modelHeight := config.Width, config.Height
	if config.Width > primitiveImageMaxSide || config.Height > primitiveImageMaxSide || len(raw) > primitiveModelImageMax {
		decoded, _, decodeErr := image.Decode(bytes.NewReader(raw))
		if decodeErr != nil {
			return primitiveError("read", "unsupported_image", decodeErr)
		}
		modelWidth, modelHeight = scaledImageSize(config.Width, config.Height, primitiveImageMaxSide)
		resized := resizeNearest(decoded, modelWidth, modelHeight)
		var output bytes.Buffer
		if err := png.Encode(&output, resized); err != nil {
			return primitiveError("read", "image_encode_failed", err)
		}
		modelRaw = output.Bytes()
		mimeType = "image/png"
	}
	if len(modelRaw) > primitiveModelImageMax {
		return primitiveError("read", "image_too_large", errors.New("resized image exceeds the model payload limit"))
	}
	digest := sha256.Sum256(raw)
	path := filepath.ToSlash(filepath.Clean(requested))
	attachment := domain.MessageAttachment{Name: filepath.Base(path), MIMEType: mimeType, Kind: "image", Data: base64.StdEncoding.EncodeToString(modelRaw), Size: int64(len(modelRaw))}
	return domain.ToolResult{
		Name: "read", OK: true, Content: fmt.Sprintf("Read image %s (%dx%d)", path, config.Width, config.Height), ModelContent: fmt.Sprintf("Image from %s (%dx%d)", path, config.Width, config.Height), ModelAttachments: []domain.MessageAttachment{attachment},
		Structured: map[string]any{"kind": "image", "path": path, "mimeType": mimeType, "width": config.Width, "height": config.Height, "modelWidth": modelWidth, "modelHeight": modelHeight, "bytes": len(raw), "sha256": hex.EncodeToString(digest[:])},
	}
}

type primitiveEdit struct {
	OldText string `json:"oldText"`
	NewText string `json:"newText"`
}
type EditTool struct {
	workspaceRoot string
	environment   ExecutionEnvironment
}

func NewEditTool(workspaceRoot string) *EditTool { return &EditTool{workspaceRoot: workspaceRoot} }

func (t *EditTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name: "edit", Description: "Atomically apply exact, unique, non-overlapping replacements to one workspace-relative text file snapshot. Matching is byte-for-byte and never fuzzy.",
		Capability: "filesystem.write", RiskLevel: "high", Category: "filesystem", Toolsets: []string{"coding"}, RequiresWorkspace: true,
		ImplementationHash: executionEnvironmentHash(t.environment),
		InputSchema: map[string]any{
			"type": "object", "additionalProperties": false,
			"properties": map[string]any{
				"path":  map[string]any{"type": "string", "minLength": 1, "description": workspaceRelativePathDescription},
				"edits": map[string]any{"type": "array", "minItems": 1, "items": map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"oldText": map[string]any{"type": "string", "minLength": 1}, "newText": map[string]any{"type": "string"}}, "required": []string{"oldText", "newText"}}},
			},
			"required": []string{"path", "edits"},
		},
	}
}

func (t *EditTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	if t.environment != nil {
		return t.environment.ExecutePrimitive(ctx, "edit", args, execCtx)
	}
	var input struct {
		Path  string          `json:"path"`
		Edits []primitiveEdit `json:"edits"`
	}
	if err := decodeStrictToolArgs(args, &input); err != nil {
		return primitiveError("edit", "invalid_arguments", err)
	}
	if strings.TrimSpace(input.Path) == "" || len(input.Edits) == 0 {
		return primitiveError("edit", "invalid_arguments", errors.New("path and at least one edit are required"))
	}
	root := toolWorkspaceRoot(t.workspaceRoot, execCtx)
	target, err := safeTargetForWrite(root, input.Path)
	if err != nil {
		return primitiveError("edit", "invalid_path", err)
	}
	lock := lockForFile(target)
	lock.Lock()
	defer lock.Unlock()
	if err := ctx.Err(); err != nil {
		return primitiveError("edit", "cancelled", err)
	}
	original, err := os.ReadFile(target)
	if err != nil {
		return primitiveError("edit", "read_failed", err)
	}
	if bytes.IndexByte(original, 0) >= 0 {
		return primitiveError("edit", "unsupported_binary", errors.New("refusing to edit a binary file"))
	}
	baseHash := sha256.Sum256(original)
	if expected := preparedExpectedHash(execCtx.ToolCallID, input.Path); expected != "" && expected != hex.EncodeToString(baseHash[:]) {
		return primitiveError("edit", "file_changed", staleFileError{Path: cleanPatchPath(input.Path), ExpectedHash: expected, CurrentHash: hex.EncodeToString(baseHash[:])})
	}
	type replacement struct {
		start, end int
		newText    string
	}
	replacements := make([]replacement, 0, len(input.Edits))
	seen := map[string]bool{}
	for _, item := range input.Edits {
		if item.OldText == "" {
			return primitiveError("edit", "invalid_arguments", errors.New("oldText must not be empty"))
		}
		if seen[item.OldText] {
			return primitiveError("edit", "duplicate_edit", errors.New("duplicate oldText entries are not allowed"))
		}
		seen[item.OldText] = true
		count := bytes.Count(original, []byte(item.OldText))
		if count == 0 {
			return primitiveError("edit", "no_match", errors.New("oldText was not found"))
		}
		if count != 1 {
			return primitiveError("edit", "multiple_matches", errors.New("oldText must occur exactly once"))
		}
		start := bytes.Index(original, []byte(item.OldText))
		replacements = append(replacements, replacement{start: start, end: start + len(item.OldText), newText: item.NewText})
	}
	sort.Slice(replacements, func(i, j int) bool { return replacements[i].start < replacements[j].start })
	for index := 1; index < len(replacements); index++ {
		if replacements[index].start < replacements[index-1].end {
			return primitiveError("edit", "overlapping_edits", errors.New("edit ranges overlap"))
		}
	}
	result := append([]byte(nil), original...)
	for index := len(replacements) - 1; index >= 0; index-- {
		item := replacements[index]
		next := make([]byte, 0, len(result)-(item.end-item.start)+len(item.newText))
		next = append(next, result[:item.start]...)
		next = append(next, item.newText...)
		next = append(next, result[item.end:]...)
		result = next
	}
	if err := ctx.Err(); err != nil {
		return primitiveError("edit", "cancelled", err)
	}
	if err := atomicReplaceFile(target, result, 0o600); err != nil {
		return primitiveError("edit", "write_failed", err)
	}
	return primitiveMutationResult("edit", root, input.Path, original, result, false, len(input.Edits))
}

type WriteTool struct {
	workspaceRoot string
	environment   ExecutionEnvironment
}

func NewWriteTool(workspaceRoot string) *WriteTool { return &WriteTool{workspaceRoot: workspaceRoot} }

func (t *WriteTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name: "write", Description: "Atomically create or completely overwrite one workspace-relative text file, creating missing parent directories.",
		Capability: "filesystem.write", RiskLevel: "high", Category: "filesystem", Toolsets: []string{"coding"}, RequiresWorkspace: true,
		ImplementationHash: executionEnvironmentHash(t.environment),
		InputSchema: map[string]any{
			"type": "object", "additionalProperties": false,
			"properties": map[string]any{"path": map[string]any{"type": "string", "minLength": 1, "description": workspaceRelativePathDescription}, "content": map[string]any{"type": "string"}},
			"required":   []string{"path", "content"},
		},
	}
}

func (t *WriteTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	if t.environment != nil {
		return t.environment.ExecutePrimitive(ctx, "write", args, execCtx)
	}
	var input struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := decodeStrictToolArgs(args, &input); err != nil {
		return primitiveError("write", "invalid_arguments", err)
	}
	if strings.TrimSpace(input.Path) == "" {
		return primitiveError("write", "invalid_arguments", errors.New("path is required"))
	}
	root := toolWorkspaceRoot(t.workspaceRoot, execCtx)
	target, err := safeTargetForWrite(root, input.Path)
	if err != nil {
		return primitiveError("write", "invalid_path", err)
	}
	lock := lockForFile(target)
	lock.Lock()
	defer lock.Unlock()
	if err := ctx.Err(); err != nil {
		return primitiveError("write", "cancelled", err)
	}
	original, err := os.ReadFile(target)
	created := errors.Is(err, os.ErrNotExist)
	if err != nil && !created {
		return primitiveError("write", "read_failed", err)
	}
	currentHash := "<missing>"
	if !created {
		digest := sha256.Sum256(original)
		currentHash = hex.EncodeToString(digest[:])
	}
	if expected := preparedExpectedHash(execCtx.ToolCallID, input.Path); expected != "" && expected != currentHash {
		return primitiveError("write", "file_changed", staleFileError{Path: cleanPatchPath(input.Path), ExpectedHash: expected, CurrentHash: currentHash})
	}
	result := []byte(input.Content)
	if err := atomicReplaceFile(target, result, 0o600); err != nil {
		return primitiveError("write", "write_failed", err)
	}
	return primitiveMutationResult("write", root, input.Path, original, result, created, 1)
}

func primitiveMutationResult(name, root, requested string, before, after []byte, created bool, count int) domain.ToolResult {
	path := filepath.ToSlash(filepath.Clean(requested))
	diff := bounded(simpleFileDiff(path, path, string(before), string(after)), primitiveDiffMaxChars)
	additions, deletions := countLineDelta(string(before), string(after))
	digest := sha256.Sum256(after)
	changeType := name
	verb := "Updated"
	if created {
		changeType, verb = "add", "Created"
	}
	file := domain.ToolResultFile{Path: path, FullPath: fullWorkspacePath(root, path), Type: changeType, Additions: additions, Deletions: deletions, Diff: diff, CurrentHash: hex.EncodeToString(digest[:])}
	return domain.ToolResult{
		Name: name, OK: true, Content: fmt.Sprintf("%s %s (%d bytes)", verb, path, len(after)), ModelContent: fmt.Sprintf("%s %s (%d bytes)", verb, path, len(after)), Files: []domain.ToolResultFile{file},
		Structured: map[string]any{"path": path, "created": created, "bytes": len(after), "sha256": hex.EncodeToString(digest[:]), "editCount": count, "files": []domain.ToolResultFile{file}},
	}
}

func primitiveExistingPath(root, input string) (string, error) {
	if filepath.IsAbs(input) {
		if pathWithinRetainedOutputRoots(filepath.Clean(input)) {
			return validateRetainedOutputRef(input)
		}
		return "", workspaceRelativePathError()
	}
	return safeJoin(root, input)
}

func canonicalEnvironmentPath(root, target string) string {
	rel, err := filepath.Rel(root, target)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(filepath.Clean(target))
}

func decodeStrictToolArgs(raw json.RawMessage, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("multiple JSON values are not allowed")
	}
	return nil
}

func primitiveError(name, code string, err error) domain.ToolResult {
	return domain.ToolResult{Name: name, OK: false, Error: err.Error(), ModelContent: err.Error(), ToolError: &domain.ToolError{Code: code, Message: err.Error()}}
}

func looksLikeSupportedImage(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif":
		return true
	default:
		return false
	}
}

func scaledImageSize(width, height, maxSide int) (int, int) {
	if width <= maxSide && height <= maxSide {
		return width, height
	}
	if width >= height {
		return maxSide, maxInt(1, height*maxSide/width)
	}
	return maxInt(1, width*maxSide/height), maxSide
}

func resizeNearest(source image.Image, width, height int) *image.RGBA {
	destination := image.NewRGBA(image.Rect(0, 0, width, height))
	bounds := source.Bounds()
	for y := 0; y < height; y++ {
		sy := bounds.Min.Y + y*bounds.Dy()/height
		for x := 0; x < width; x++ {
			sx := bounds.Min.X + x*bounds.Dx()/width
			destination.Set(x, y, source.At(sx, sy))
		}
	}
	return destination
}
