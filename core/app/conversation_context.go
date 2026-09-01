package app

import (
	"context"
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

type conversationContextOptions struct {
	CurrentInput       string
	IncludeCurrent     bool
	SectionBudget      int
	TailMessageBudget  int
	TailMessageLimit   int
	IncludeChatTail    bool
	IncludeToolContext bool
	TargetPaths        []string
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
		TargetPaths:        input.TargetPaths,
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
	turns, err := s.store.ListTurns(ctx, sessionID, 500)
	if err != nil {
		return conversationContextAssembly{}, err
	}
	summary, _ := s.store.LatestSummary(ctx, session.ID)
	chatEvents := events
	if summary != nil {
		chatEvents = events[firstEventAfterSummary(events, summary):]
	}
	chat := chatMessagesFromEvents(chatEvents, turns)
	tail := selectRecentChatTail(chat, opts.TailMessageLimit, opts.TailMessageBudget)
	older := chat[:len(chat)-len(tail)]

	checkpoint, _ := s.store.LatestCheckpoint(ctx, session.ID)
	codingContext, _ := s.store.GetCodingContext(ctx, session.ID)
	var tools []domain.ToolCall
	if opts.IncludeToolContext {
		tools, _ = s.store.ListToolCalls(ctx, session.ID)
	}
	pendingPermissions, _ := s.store.ListPermissionRequests(ctx, session.ID, domain.PermissionRequestStatusPending)
	visibleSkills := s.visibleSkillsContext(ctx, session.ID)
	activeSkills := s.activeSkillsContext(ctx, session.ID)
	activeExtensionContexts := s.activeExtensionContextsContext(ctx, session.ID)
	if strings.TrimSpace(activeExtensionContexts) != "" {
		activeSkills = strings.TrimSpace(activeSkills + "\n\n" + activeExtensionContexts)
	}
	projectInstructions := resolveProjectInstructions(session.ProjectPath, opts.TargetPaths)
	configuredInstructions := resolveConfiguredRuntimeInstructions(ctx, session.ProjectPath)
	if configuredInstructions != "" {
		projectInstructions = strings.TrimSpace(projectInstructions + "\n\n" + configuredInstructions)
	}
	liveTerminals := s.liveSessionTerminalsContext(session, codingContext)

	sections := s.contextSectionCandidates(session, summary, checkpoint, codingContext, tools, pendingPermissions, visibleSkills, activeSkills, projectInstructions, liveTerminals, older, tail, opts)
	applied := applyConversationSectionBudget(sections, opts.SectionBudget)
	var messages []domain.ChatMessage
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
	visibleSkills string,
	activeSkills string,
	projectInstructions string,
	liveTerminals string,
	older []domain.ChatMessage,
	tail []domain.ChatMessage,
	opts conversationContextOptions,
) []contextSectionCandidate {
	contextPolicy, err := s.renderManagedPrompt("dynamic.context_policy", nil)
	if err != nil {
		contextPolicy = builtinPromptBody("dynamic.context_policy")
	}
	sections := []contextSectionCandidate{
		{Name: "context_policy", Content: contextPolicy, MaxChars: dynamicContextHeaderChars, Required: true},
		{Name: "session", Content: renderSessionMetadata(session), MaxChars: sectionDefaultMaxChars, Required: true},
	}
	if text := strings.TrimSpace(session.SystemPromptSnapshot); text != "" {
		sections = append(sections, contextSectionCandidate{Name: "system_prompt_snapshot", Content: text, MaxChars: sectionSummaryMaxChars, Required: true})
	}
	if text := strings.TrimSpace(session.Goal); text != "" {
		sections = append(sections, contextSectionCandidate{Name: "goal", Content: text, MaxChars: sectionDefaultMaxChars, Required: true})
	}
	if strings.TrimSpace(liveTerminals) != "" {
		sections = append(sections, contextSectionCandidate{Name: "live_terminals", Content: liveTerminals, MaxChars: sectionToolMaxChars, Required: true})
	}
	if strings.TrimSpace(projectInstructions) != "" {
		sections = append(sections, contextSectionCandidate{Name: "project_instructions", Content: projectInstructions, MaxChars: projectInstructionsMaxChars, Required: true})
	}
	if summary != nil {
		sections = append(sections, contextSectionCandidate{Name: "latest_summary", Content: renderSummaryForContext(*summary), MaxChars: sectionSummaryMaxChars, Required: true})
	} else if len(older) > 0 {
		lines := renderOlderConversationRecapMessages(older, olderRecapMessageLimit)
		recap, promptErr := s.renderManagedPrompt("dynamic.older_recap", map[string]string{"messages": lines})
		if promptErr != nil {
			recap = renderOlderConversationRecap(older, olderRecapMessageLimit)
		}
		sections = append(sections, contextSectionCandidate{Name: "deterministic_older_recap", Content: recap, MaxChars: sectionSummaryMaxChars, Required: true})
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
	if strings.TrimSpace(visibleSkills) != "" {
		sections = append(sections, contextSectionCandidate{Name: "available_skills", Content: visibleSkills, MaxChars: sectionSummaryMaxChars, Required: true})
	}
	if strings.TrimSpace(activeSkills) != "" {
		sections = append(sections, contextSectionCandidate{Name: "active_skills", Content: activeSkills, MaxChars: sectionSummaryMaxChars, Required: true})
	}
	if len(tools) > 0 {
		if snapshotLines := renderRecentFileSnapshotLines(tools, recentToolContextLimit); snapshotLines != "" {
			snapshots, promptErr := s.renderManagedPrompt("dynamic.file_snapshots", map[string]string{"snapshots": snapshotLines})
			if promptErr != nil {
				snapshots = renderRecentFileSnapshots(tools, recentToolContextLimit)
			}
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
