package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"aivo/core/domain"
)

const (
	turnFileTargetBefore = "before"
	turnFileTargetAfter  = "after"
)

type turnFileChange struct {
	domain.SessionTurnDiffFile
	beforeText string
	afterText  string
}

func (s *Service) GetSessionTurnDiff(ctx context.Context, input domain.GetSessionTurnDiffRequest) (domain.SessionTurnDiff, error) {
	turn, cc, changes, err := s.sessionTurnFileChanges(ctx, input.SessionID, input.TurnID)
	if err != nil {
		return domain.SessionTurnDiff{}, err
	}
	files := make([]domain.SessionTurnDiffFile, 0, len(changes))
	var diff strings.Builder
	for _, change := range changes {
		annotated := s.annotateTurnFileChange(cc.ProjectPath, change)
		files = append(files, annotated.SessionTurnDiffFile)
		if strings.TrimSpace(annotated.Diff) != "" {
			diff.WriteString(annotated.Diff)
			if !strings.HasSuffix(annotated.Diff, "\n") {
				diff.WriteByte('\n')
			}
		}
	}
	return domain.SessionTurnDiff{SessionID: turn.SessionID, TurnID: turn.ID, Files: files, Diff: diff.String()}, nil
}

func (s *Service) ApplySessionTurnFileState(ctx context.Context, input domain.ApplySessionTurnFileStateRequest) (domain.SessionTurnDiff, error) {
	targetState := strings.TrimSpace(input.TargetState)
	if targetState == "" {
		targetState = turnFileTargetBefore
	}
	if targetState != turnFileTargetBefore && targetState != turnFileTargetAfter {
		return domain.SessionTurnDiff{}, errors.New("targetState must be before or after")
	}
	turn, cc, changes, err := s.sessionTurnFileChanges(ctx, input.SessionID, input.TurnID)
	if err != nil {
		return domain.SessionTurnDiff{}, err
	}
	var selected []turnFileChange
	for _, change := range changes {
		if !turnFileChangeMatches(change, input.ToolCallID, input.Path) {
			continue
		}
		selected = append(selected, s.annotateTurnFileChange(cc.ProjectPath, change))
	}
	if len(selected) == 0 {
		return domain.SessionTurnDiff{}, errors.New("no matching file changes for turn")
	}
	for _, change := range selected {
		if err := applyTurnFileChangeState(ctx, cc.ProjectPath, change, targetState); err != nil {
			return domain.SessionTurnDiff{}, err
		}
	}
	_ = s.appendSessionTurnFileStateEvent(ctx, turn, selected, targetState)
	return s.GetSessionTurnDiff(ctx, domain.GetSessionTurnDiffRequest{SessionID: turn.SessionID, TurnID: turn.ID})
}

func (s *Service) appendSessionTurnFileStateEvent(ctx context.Context, turn domain.Turn, changes []turnFileChange, targetState string) error {
	if s == nil || s.store == nil || len(changes) == 0 {
		return nil
	}
	action := "reverted"
	if targetState == turnFileTargetAfter {
		action = "restored"
	}
	files := make([]map[string]any, 0, len(changes))
	paths := make([]string, 0, len(changes))
	for _, change := range changes {
		path := change.effectivePath()
		if path == "" {
			path = change.Path
		}
		paths = append(paths, path)
		files = append(files, map[string]any{
			"path":       path,
			"type":       change.Type,
			"toolCallId": change.ToolCallID,
			"toolName":   change.ToolName,
			"additions":  change.Additions,
			"deletions":  change.Deletions,
		})
	}
	content := fmt.Sprintf("File changes %s for %d file(s): %s", action, len(paths), strings.Join(paths, ", "))
	_, err := s.AppendEvent(ctx, domain.AppendEventRequest{
		SessionID:  turn.SessionID,
		TurnID:     turn.ID,
		Type:       domain.EventTypeSystemNote,
		Role:       domain.EventRoleSystem,
		Visibility: domain.EventVisibilityNormal,
		Content:    content,
		Payload:    map[string]any{"kind": "turn_file_state", "targetState": targetState, "action": action, "files": files},
		TokenCount: 0,
	})
	return err
}

func (s *Service) sessionTurnFileChanges(ctx context.Context, sessionID string, turnID string) (domain.Turn, domain.CodingContext, []turnFileChange, error) {
	turnID = strings.TrimSpace(turnID)
	if turnID == "" {
		return domain.Turn{}, domain.CodingContext{}, nil, errors.New("turnId is required")
	}
	turn, err := s.store.GetTurn(ctx, turnID)
	if err != nil {
		return domain.Turn{}, domain.CodingContext{}, nil, err
	}
	if strings.TrimSpace(sessionID) != "" && strings.TrimSpace(sessionID) != turn.SessionID {
		return domain.Turn{}, domain.CodingContext{}, nil, errors.New("turn does not belong to session")
	}
	cc, err := s.store.GetCodingContext(ctx, turn.SessionID)
	if err != nil || strings.TrimSpace(cc.ProjectPath) == "" {
		return domain.Turn{}, domain.CodingContext{}, nil, errors.New("turn has no coding workspace")
	}
	toolCalls, err := s.store.ListToolCalls(ctx, turn.SessionID)
	if err != nil {
		return domain.Turn{}, domain.CodingContext{}, nil, err
	}
	var changes []turnFileChange
	for _, call := range toolCalls {
		if call.TurnID != turn.ID || call.Status != domain.ToolCallStatusSuccess {
			continue
		}
		changes = append(changes, turnFileChangesFromToolCall(call)...)
	}
	return turn, cc, changes, nil
}

func turnFileChangesFromToolCall(call domain.ToolCall) []turnFileChange {
	files := toolResultFilesFromMap(call.Result)
	changes := make([]turnFileChange, 0, len(files))
	for _, file := range files {
		before, after, ok := restoreTextsFromSimpleDiff(file.Diff)
		if !ok {
			continue
		}
		changes = append(changes, turnFileChange{
			SessionTurnDiffFile: domain.SessionTurnDiffFile{
				ToolCallID:  call.ID,
				ToolName:    call.Name,
				Path:        cleanPatchPath(file.Path),
				MovePath:    cleanOptionalPath(file.MovePath),
				Type:        strings.TrimSpace(file.Type),
				Additions:   file.Additions,
				Deletions:   file.Deletions,
				Diff:        file.Diff,
				BaseHash:    file.BaseHash,
				CurrentHash: file.CurrentHash,
				TimeUpdated: call.TimeUpdated,
			},
			beforeText: before,
			afterText:  after,
		})
	}
	return changes
}

func toolResultFilesFromMap(result map[string]any) []domain.ToolResultFile {
	if len(result) == 0 {
		return nil
	}
	for _, key := range []string{"files"} {
		if files := decodeToolResultFiles(result[key]); len(files) > 0 {
			return files
		}
	}
	if structured, _ := result["structured"].(map[string]any); structured != nil {
		return decodeToolResultFiles(structured["files"])
	}
	return nil
}

func decodeToolResultFiles(value any) []domain.ToolResultFile {
	switch typed := value.(type) {
	case []domain.ToolResultFile:
		return typed
	case []any:
		files := make([]domain.ToolResultFile, 0, len(typed))
		for _, item := range typed {
			if file, ok := decodeToolResultFile(item); ok {
				files = append(files, file)
			}
		}
		return files
	default:
		return nil
	}
}

func decodeToolResultFile(value any) (domain.ToolResultFile, bool) {
	switch typed := value.(type) {
	case domain.ToolResultFile:
		return typed, true
	case map[string]any:
		return domain.ToolResultFile{
			Path:         stringFromAny(typed["path"]),
			FullPath:     stringFromAny(typed["fullPath"]),
			MovePath:     stringFromAny(typed["movePath"]),
			MoveFullPath: stringFromAny(typed["moveFullPath"]),
			Type:         stringFromAny(typed["type"]),
			Additions:    intFromAny(typed["additions"]),
			Deletions:    intFromAny(typed["deletions"]),
			Diff:         stringFromAny(typed["diff"]),
			BaseHash:     stringFromAny(typed["baseHash"]),
			CurrentHash:  stringFromAny(typed["currentHash"]),
		}, true
	default:
		return domain.ToolResultFile{}, false
	}
}

func restoreTextsFromSimpleDiff(diff string) (string, string, bool) {
	diff = strings.ReplaceAll(strings.ReplaceAll(diff, "\r\n", "\n"), "\r", "\n")
	if strings.TrimSpace(diff) == "" {
		return "", "", false
	}
	var before []string
	var after []string
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ "):
			continue
		case strings.HasPrefix(line, "-"):
			before = append(before, strings.TrimPrefix(line, "-"))
		case strings.HasPrefix(line, "+"):
			after = append(after, strings.TrimPrefix(line, "+"))
		}
	}
	return linesToFileText(before), linesToFileText(after), true
}

func linesToFileText(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func (s *Service) annotateTurnFileChange(workspaceRoot string, change turnFileChange) turnFileChange {
	if strings.TrimSpace(change.MovePath) != "" {
		change.Reason = "move changes are not revertible yet"
		return change
	}
	targetPath := change.effectivePath()
	if strings.TrimSpace(targetPath) == "" {
		change.Reason = "missing path"
		return change
	}
	target, err := safeTargetForWrite(workspaceRoot, targetPath)
	if err != nil {
		change.Reason = err.Error()
		return change
	}
	currentHash, exists, err := fileHashIfExists(target)
	if err != nil {
		change.Reason = err.Error()
		return change
	}
	if exists {
		change.CurrentFileHash = currentHash
	}
	beforeMissing := change.Type == "add"
	afterMissing := change.Type == "delete"
	change.Revertible = currentFileMatchesState(currentHash, exists, change.afterText, afterMissing)
	change.Unrevertible = currentFileMatchesState(currentHash, exists, change.beforeText, beforeMissing)
	if !change.Revertible && !change.Unrevertible && change.Reason == "" {
		change.Reason = "file changed after this turn"
	}
	return change
}

func applyTurnFileChangeState(ctx context.Context, workspaceRoot string, change turnFileChange, targetState string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	switch targetState {
	case turnFileTargetBefore:
		if !change.Revertible {
			return fmt.Errorf("%s is not revertible: %s", change.effectivePath(), firstNonEmpty(change.Reason, "current content does not match turn output"))
		}
		return writeTurnFileState(workspaceRoot, change, change.beforeText, change.afterText, change.Type == "add", change.Type == "delete")
	case turnFileTargetAfter:
		if !change.Unrevertible {
			return fmt.Errorf("%s is not unrevertible: %s", change.effectivePath(), firstNonEmpty(change.Reason, "current content does not match pre-turn content"))
		}
		return writeTurnFileState(workspaceRoot, change, change.afterText, change.beforeText, change.Type == "delete", change.Type == "add")
	default:
		return errors.New("targetState must be before or after")
	}
}

func writeTurnFileState(workspaceRoot string, change turnFileChange, desiredText string, expectedText string, desiredMissing bool, expectedMissing bool) error {
	targetPath := change.effectivePath()
	target, err := safeTargetForWrite(workspaceRoot, targetPath)
	if err != nil {
		return err
	}
	expectedHash := hashText(expectedText)
	if expectedMissing {
		expectedHash = "<missing>"
	}
	if desiredMissing {
		return removeFileIfUnchanged(target, targetPath, expectedHash)
	}
	return writeFileIfUnchanged(target, targetPath, expectedHash, []byte(desiredText), 0o600)
}

func (change turnFileChange) effectivePath() string {
	if strings.TrimSpace(change.MovePath) != "" {
		return cleanPatchPath(change.MovePath)
	}
	return cleanPatchPath(change.Path)
}

func turnFileChangeMatches(change turnFileChange, toolCallID string, path string) bool {
	if strings.TrimSpace(toolCallID) != "" && strings.TrimSpace(toolCallID) != change.ToolCallID {
		return false
	}
	path = cleanPatchPath(path)
	if path == "." || path == "" {
		return true
	}
	return path == cleanPatchPath(change.Path) || path == cleanPatchPath(change.MovePath)
}

func currentFileMatchesState(currentHash string, exists bool, stateText string, stateMissing bool) bool {
	if stateMissing {
		return !exists
	}
	return exists && currentHash == hashText(stateText)
}

func hashText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func stringFromAny(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}

func intFromAny(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	default:
		return 0
	}
}

func cleanOptionalPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	return cleanPatchPath(path)
}
