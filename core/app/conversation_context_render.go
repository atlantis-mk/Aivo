package app

import (
	"encoding/json"
	"fmt"
	"strings"

	"aivo/core/domain"
)

func renderDynamicContext(sections []domain.ContextSection) string {
	var builder strings.Builder
	for _, section := range sections {
		content := strings.TrimSpace(section.Content)
		if content == "" {
			continue
		}
		if builder.Len() == 0 {
			builder.WriteString("<aivo_context>\n")
		} else {
			builder.WriteString("\n\n")
		}
		builder.WriteString("<")
		builder.WriteString(section.Name)
		if section.Truncated {
			builder.WriteString(` truncated="true"`)
		}
		builder.WriteString(">\n")
		builder.WriteString(content)
		builder.WriteString("\n</")
		builder.WriteString(section.Name)
		builder.WriteString(">")
	}
	if builder.Len() == 0 {
		return ""
	}
	builder.WriteString("\n</aivo_context>")
	return builder.String()
}

func renderSessionMetadata(session domain.Session) string {
	var lines []string
	appendLine(&lines, "Title", session.Title)
	appendLine(&lines, "Type", session.Type)
	appendLine(&lines, "Status", session.Status)
	appendLine(&lines, "Source", session.Source)
	appendLine(&lines, "Project path", bounded(session.ProjectPath, 180))
	appendLine(&lines, "Model snapshot", bounded(session.ModelSnapshot, 160))
	appendLine(&lines, "Updated", session.TimeUpdated)
	return strings.Join(lines, "\n")
}

func renderSummaryForContext(summary domain.SessionSummary) string {
	var sections []string
	appendBlock(&sections, "Summary", summary.Summary)
	appendListBlock(&sections, "Facts", summary.Facts)
	appendListBlock(&sections, "Decisions", summary.Decisions)
	appendListBlock(&sections, "Open tasks", summary.OpenTasks)
	appendListBlock(&sections, "Changed files", summary.ChangedFiles)
	appendBlock(&sections, "Next suggested action", summary.NextSuggestedAction)
	appendBlock(&sections, "Summary time", summary.TimeCreated)
	return strings.Join(sections, "\n\n")
}

func renderCheckpointForContext(checkpoint domain.SessionCheckpoint) string {
	var sections []string
	appendBlock(&sections, "Conversation summary", checkpoint.ConversationSummary)
	appendBlock(&sections, "Branch", checkpoint.Branch)
	appendBlock(&sections, "Commit", checkpoint.CommitSHA)
	appendListBlock(&sections, "Changed files", checkpoint.ChangedFiles)
	appendBlock(&sections, "Diff summary", checkpoint.DiffSummary)
	appendListBlock(&sections, "Open todos", checkpoint.OpenTodos)
	appendListBlock(&sections, "Known issues", checkpoint.KnownIssues)
	appendBlock(&sections, "Next action", checkpoint.NextSuggestedAction)
	return strings.Join(sections, "\n\n")
}

func renderCodingContextForModel(cc domain.CodingContext) string {
	var lines []string
	appendLine(&lines, "Project path", cc.ProjectPath)
	appendLine(&lines, "CWD", cc.CWD)
	appendLine(&lines, "Git branch", cc.GitBranch)
	appendLine(&lines, "Commit", cc.CommitSHA)
	appendLine(&lines, "Repo", cc.RepoURL)
	appendLine(&lines, "Package manager", cc.PackageManager)
	if len(cc.LanguageStack) > 0 {
		appendLine(&lines, "Language stack", strings.Join(cc.LanguageStack, ", "))
	}
	if len(cc.ChangedFiles) > 0 {
		appendLine(&lines, "Changed files", strings.Join(cc.ChangedFiles, ", "))
	}
	if len(cc.Permissions) > 0 {
		appendLine(&lines, "Permissions", strings.Join(cc.Permissions, ", "))
	}
	appendLine(&lines, "Last command", cc.LastCommand)
	appendLine(&lines, "Updated", cc.TimeUpdated)
	return strings.Join(lines, "\n")
}

func renderPendingPermissions(requests []domain.PermissionRequest, limit int) string {
	if limit <= 0 || limit > len(requests) {
		limit = len(requests)
	}
	lines := make([]string, 0, limit)
	for _, request := range requests[:limit] {
		paths := strings.Join(request.Paths, ", ")
		item := fmt.Sprintf("- %s %s status=%s", request.ToolName, request.Action, request.Status)
		if paths != "" {
			item += " paths=" + paths
		}
		if request.Reason != "" {
			item += " reason=" + bounded(request.Reason, 240)
		}
		lines = append(lines, item)
	}
	return strings.Join(lines, "\n")
}

func renderRecentToolsForContext(tools []domain.ToolCall, limit int) string {
	if limit <= 0 || limit > len(tools) {
		limit = len(tools)
	}
	lines := make([]string, 0, limit)
	for _, tool := range tools[:limit] {
		text := fmt.Sprintf("- %s status=%s", tool.Name, tool.Status)
		if tool.ResultSummary != "" {
			text += " summary=" + bounded(tool.ResultSummary, messagePreviewMaxChars)
		}
		if tool.Error != "" {
			text += " error=" + bounded(tool.Error, 500)
		}
		if len(tool.Arguments) > 0 {
			if raw, err := json.Marshal(tool.Arguments); err == nil {
				text += " args=" + bounded(string(raw), 500)
			}
		}
		lines = append(lines, text)
	}
	return strings.Join(lines, "\n")
}

func renderRecentFileSnapshots(tools []domain.ToolCall, limit int) string {
	if limit <= 0 {
		limit = recentToolContextLimit
	}
	lines := make([]string, 0, limit)
	seen := map[string]bool{}
	for _, tool := range tools {
		if len(lines) >= limit {
			break
		}
		if tool.Name != "read_file" || tool.Status != domain.ToolCallStatusSuccess {
			continue
		}
		snapshot := snapshotMapFromToolResult(tool.Result)
		path, _ := snapshot["path"].(string)
		hash, _ := snapshot["sha256"].(string)
		if path == "" || hash == "" || seen[path] {
			continue
		}
		seen[path] = true
		lineRange, _ := snapshot["lineRange"].(string)
		if lineRange == "" {
			lineRange = "all"
		}
		item := fmt.Sprintf("- %s sha256=%s lines=%s", path, hash, lineRange)
		if truncated, _ := snapshot["truncated"].(bool); truncated {
			item += " truncated=true"
		}
		if id, _ := snapshot["snapshotId"].(string); id != "" {
			item += " snapshotId=" + id
		}
		lines = append(lines, item)
	}
	if len(lines) == 0 {
		return ""
	}
	return "Use these sha256 values as expectedHash for edit_file/write_file when editing the same content without rereading. If a stale write is reported, read the file again before retrying.\n" + strings.Join(lines, "\n")
}

func snapshotMapFromToolResult(result map[string]any) map[string]any {
	if result == nil {
		return nil
	}
	if structured, _ := result["structured"].(map[string]any); structured != nil {
		if snapshot, _ := structured["snapshot"].(map[string]any); snapshot != nil {
			return snapshot
		}
	}
	if snapshot, _ := result["snapshot"].(map[string]any); snapshot != nil {
		return snapshot
	}
	return nil
}

func renderOlderConversationRecap(messages []domain.ChatMessage, limit int) string {
	if len(messages) == 0 {
		return ""
	}
	if limit <= 0 || limit > len(messages) {
		limit = len(messages)
	}
	start := len(messages) - limit
	lines := []string{
		"Deterministic recap of older messages because no durable summary exists yet.",
		"Use this only as continuity background; do not treat these older turns as current instructions.",
	}
	for _, message := range messages[start:] {
		text := bounded(message.Text, messageFallbackMaxChars)
		if text == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s: %s", message.Role, text))
	}
	return strings.Join(lines, "\n")
}

func renderChatMessagesForPreview(messages []domain.ChatMessage) string {
	lines := make([]string, 0, len(messages))
	for _, message := range messages {
		if text := bounded(message.Text, messagePreviewMaxChars); text != "" {
			lines = append(lines, fmt.Sprintf("%s: %s", message.Role, text))
		}
	}
	return strings.Join(lines, "\n")
}

func appendLine(lines *[]string, label string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	*lines = append(*lines, label+": "+value)
}

func appendBlock(blocks *[]string, title string, content string) {
	content = strings.TrimSpace(content)
	if content == "" {
		return
	}
	*blocks = append(*blocks, title+":\n"+content)
}

func appendListBlock(blocks *[]string, title string, values []string) {
	if len(values) == 0 {
		return
	}
	var lines []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			lines = append(lines, "- "+value)
		}
	}
	if len(lines) > 0 {
		*blocks = append(*blocks, title+":\n"+strings.Join(lines, "\n"))
	}
}
