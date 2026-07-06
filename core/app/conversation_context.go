package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"aivo/core/domain"
)

const (
	modelContextSectionBudget = 24000
	modelTailMessageBudget    = 36000
	modelTailMessageLimit     = 40

	previewContextBudget      = 12000
	olderRecapMessageLimit    = 16
	recentToolContextLimit    = 12
	pendingPermissionLimit    = 8
	sectionDefaultMaxChars    = 4000
	sectionSummaryMaxChars    = 8000
	sectionToolMaxChars       = 5000
	messagePreviewMaxChars    = 1200
	messageFallbackMaxChars   = 700
	dynamicContextHeaderChars = 1200
)

const aivoConversationSystemPrompt = `You are Aivo, a coding assistant running inside a local desktop app.
Use the session context and recent conversation to answer the user. Treat tool results as authoritative. Be explicit about files changed, commands run, and verification status. If local context is insufficient, say what is missing instead of inventing details.`

const compactedContextNotice = `Reference-only background: the latest user message is the active task, and it wins over older context.`

type conversationContextOptions struct {
	CurrentInput       string
	IncludeCurrent     bool
	SectionBudget      int
	TailMessageBudget  int
	TailMessageLimit   int
	IncludeChatTail    bool
	IncludeToolContext bool
}

type conversationContextAssembly struct {
	Session           domain.Session
	Sections          []domain.ContextSection
	Messages          []domain.ChatMessage
	EstimatedTokens   int
	CharacterBudget   int
	TruncatedSections []string
}

type contextSectionCandidate struct {
	Name     string
	Content  string
	MaxChars int
	Required bool
}

func (s *Service) modelVisibleSessionHistory(ctx context.Context, sessionID string) ([]domain.ChatMessage, error) {
	assembly, err := s.assembleConversationContext(ctx, sessionID, conversationContextOptions{
		SectionBudget:      modelContextSectionBudget,
		TailMessageBudget:  modelTailMessageBudget,
		TailMessageLimit:   modelTailMessageLimit,
		IncludeChatTail:    true,
		IncludeToolContext: true,
	})
	if err != nil {
		return nil, err
	}
	return assembly.Messages, nil
}

func (s *Service) buildSessionContextSections(ctx context.Context, input domain.BuildSessionContextRequest) (domain.BuildSessionContextResult, error) {
	budget := input.CharacterBudget
	if budget <= 0 && input.MaxTokens > 0 {
		budget = input.MaxTokens * 4
	}
	if budget <= 0 {
		budget = previewContextBudget
	}
	assembly, err := s.assembleConversationContext(ctx, input.SessionID, conversationContextOptions{
		CurrentInput:       input.CurrentInput,
		IncludeCurrent:     true,
		SectionBudget:      budget,
		TailMessageBudget:  budget / 2,
		TailMessageLimit:   30,
		IncludeChatTail:    false,
		IncludeToolContext: true,
	})
	if err != nil {
		return domain.BuildSessionContextResult{}, err
	}
	return domain.BuildSessionContextResult{
		SessionID:         input.SessionID,
		Sections:          assembly.Sections,
		EstimatedTokens:   assembly.EstimatedTokens,
		CharacterBudget:   assembly.CharacterBudget,
		TruncatedSections: assembly.TruncatedSections,
	}, nil
}

func (s *Service) assembleConversationContext(ctx context.Context, sessionID string, opts conversationContextOptions) (conversationContextAssembly, error) {
	session, err := s.store.GetRuntimeSession(ctx, sessionID)
	if err != nil {
		return conversationContextAssembly{}, err
	}
	events, err := s.store.ListSessionEvents(ctx, sessionID, false, 500)
	if err != nil {
		return conversationContextAssembly{}, err
	}
	chat := chatMessagesFromEvents(events)
	tail := selectRecentChatTail(chat, opts.TailMessageLimit, opts.TailMessageBudget)
	older := chat[:len(chat)-len(tail)]

	summary, _ := s.store.LatestSummary(ctx, session.ID)
	checkpoint, _ := s.store.LatestCheckpoint(ctx, session.ID)
	codingContext, _ := s.store.GetCodingContext(ctx, session.ID)
	var tools []domain.ToolCall
	if opts.IncludeToolContext {
		tools, _ = s.store.ListToolCalls(ctx, session.ID)
	}
	pendingPermissions, _ := s.store.ListPermissionRequests(ctx, session.ID, domain.PermissionRequestStatusPending)

	sections := s.contextSectionCandidates(session, summary, checkpoint, codingContext, tools, pendingPermissions, older, tail, opts)
	applied := applyConversationSectionBudget(sections, opts.SectionBudget)
	messages := []domain.ChatMessage{
		{Role: domain.EventRoleSystem, Text: aivoConversationSystemPrompt},
	}
	if text := renderDynamicContext(applied.Sections); text != "" {
		messages = append(messages, domain.ChatMessage{Role: domain.EventRoleSystem, Text: text})
	}
	if opts.IncludeChatTail {
		messages = append(messages, tail...)
	}
	return conversationContextAssembly{
		Session:           session,
		Sections:          applied.Sections,
		Messages:          messages,
		EstimatedTokens:   (applied.UsedChars + chatMessagesCharLen(tail)) / 4,
		CharacterBudget:   applied.CharacterBudget,
		TruncatedSections: applied.TruncatedSections,
	}, nil
}

func (s *Service) contextSectionCandidates(
	session domain.Session,
	summary *domain.SessionSummary,
	checkpoint *domain.SessionCheckpoint,
	cc domain.CodingContext,
	tools []domain.ToolCall,
	pendingPermissions []domain.PermissionRequest,
	older []domain.ChatMessage,
	tail []domain.ChatMessage,
	opts conversationContextOptions,
) []contextSectionCandidate {
	sections := []contextSectionCandidate{
		{Name: "context_policy", Content: compactedContextNotice, MaxChars: dynamicContextHeaderChars, Required: true},
		{Name: "session", Content: renderSessionMetadata(session), MaxChars: sectionDefaultMaxChars, Required: true},
	}
	if text := strings.TrimSpace(session.SystemPromptSnapshot); text != "" {
		sections = append(sections, contextSectionCandidate{Name: "system_prompt_snapshot", Content: text, MaxChars: sectionSummaryMaxChars, Required: true})
	}
	if text := strings.TrimSpace(session.Goal); text != "" {
		sections = append(sections, contextSectionCandidate{Name: "goal", Content: text, MaxChars: sectionDefaultMaxChars, Required: true})
	}
	if summary != nil {
		sections = append(sections, contextSectionCandidate{Name: "latest_summary", Content: renderSummaryForContext(*summary), MaxChars: sectionSummaryMaxChars, Required: true})
	} else if len(older) > 0 {
		sections = append(sections, contextSectionCandidate{Name: "deterministic_older_recap", Content: renderOlderConversationRecap(older, olderRecapMessageLimit), MaxChars: sectionSummaryMaxChars, Required: true})
	}
	if opts.IncludeCurrent && strings.TrimSpace(opts.CurrentInput) != "" {
		sections = append(sections, contextSectionCandidate{Name: "current_user_input", Content: strings.TrimSpace(opts.CurrentInput), MaxChars: sectionDefaultMaxChars, Required: true})
	}
	if !opts.IncludeChatTail && len(tail) > 0 {
		sections = append(sections, contextSectionCandidate{Name: "recent_events", Content: renderChatMessagesForPreview(tail), MaxChars: sectionSummaryMaxChars, Required: true})
	}
	if checkpoint != nil {
		sections = append(sections, contextSectionCandidate{Name: "latest_checkpoint", Content: renderCheckpointForContext(*checkpoint), MaxChars: sectionSummaryMaxChars, Required: true})
	}
	if cc.SessionID != "" {
		sections = append(sections, contextSectionCandidate{Name: "coding_context", Content: renderCodingContextForModel(cc), MaxChars: sectionSummaryMaxChars})
	}
	if len(pendingPermissions) > 0 {
		sections = append(sections, contextSectionCandidate{Name: "pending_permissions", Content: renderPendingPermissions(pendingPermissions, pendingPermissionLimit), MaxChars: sectionToolMaxChars, Required: true})
	}
	if len(tools) > 0 {
		if snapshots := renderRecentFileSnapshots(tools, recentToolContextLimit); snapshots != "" {
			sections = append(sections, contextSectionCandidate{Name: "recent_file_snapshots", Content: snapshots, MaxChars: sectionToolMaxChars})
		}
		sections = append(sections, contextSectionCandidate{Name: "recent_tool_results", Content: renderRecentToolsForContext(tools, recentToolContextLimit), MaxChars: sectionToolMaxChars})
	}
	return sections
}

type appliedConversationSections struct {
	Sections          []domain.ContextSection
	UsedChars         int
	CharacterBudget   int
	TruncatedSections []string
}

func applyConversationSectionBudget(candidates []contextSectionCandidate, budget int) appliedConversationSections {
	if budget <= 0 {
		budget = modelContextSectionBudget
	}
	out := make([]domain.ContextSection, 0, len(candidates))
	used := 0
	var truncated []string
	for _, candidate := range candidates {
		content := strings.TrimSpace(candidate.Content)
		if content == "" {
			continue
		}
		if candidate.MaxChars <= 0 {
			candidate.MaxChars = sectionDefaultMaxChars
		}
		content = bounded(content, candidate.MaxChars)
		remaining := budget - used
		if remaining <= 0 {
			if candidate.Required {
				out = append(out, domain.ContextSection{Name: candidate.Name, Content: "", Truncated: true})
			}
			truncated = append(truncated, candidate.Name)
			continue
		}
		section := domain.ContextSection{Name: candidate.Name, Content: content}
		if len(content) > remaining {
			section.Content = content[:remaining]
			section.Truncated = true
			truncated = append(truncated, candidate.Name)
		}
		used += len(section.Content)
		out = append(out, section)
	}
	return appliedConversationSections{Sections: out, UsedChars: used, CharacterBudget: budget, TruncatedSections: truncated}
}

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

func chatMessagesFromEvents(events []domain.SessionEvent) []domain.ChatMessage {
	messages := make([]domain.ChatMessage, 0, len(events))
	for _, event := range events {
		if event.Type != domain.EventTypeUserMessage && event.Type != domain.EventTypeAssistantMessage {
			continue
		}
		role := strings.TrimSpace(event.Role)
		if role == "" {
			if event.Type == domain.EventTypeUserMessage {
				role = domain.EventRoleUser
			} else {
				role = domain.EventRoleAssistant
			}
		}
		messages = append(messages, domain.ChatMessage{Role: role, Text: event.Content})
	}
	return messages
}

func selectRecentChatTail(messages []domain.ChatMessage, limit int, charBudget int) []domain.ChatMessage {
	if len(messages) == 0 {
		return nil
	}
	if limit <= 0 || limit > len(messages) {
		limit = len(messages)
	}
	if charBudget <= 0 {
		return messages[len(messages)-limit:]
	}
	start := len(messages)
	used := 0
	for start > 0 && len(messages)-start < limit {
		nextLen := chatMessageCharLen(messages[start-1])
		if used > 0 && used+nextLen > charBudget {
			break
		}
		used += nextLen
		start--
	}
	if start == len(messages) {
		start = len(messages) - 1
	}
	return messages[start:]
}

func chatMessageCharLen(message domain.ChatMessage) int {
	total := len(message.Role) + len(message.Text)
	for _, call := range message.ToolCalls {
		total += len(call.ID) + len(call.Name) + len(call.Arguments)
	}
	total += len(message.ToolCallID) + len(message.Name)
	return total
}

func chatMessagesCharLen(messages []domain.ChatMessage) int {
	total := 0
	for _, message := range messages {
		total += chatMessageCharLen(message)
	}
	return total
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
