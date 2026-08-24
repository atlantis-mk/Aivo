package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"aivo/core/domain"
)

func (s *Service) updateUntitledSession(ctx context.Context, sessionID, content string) error {
	session, err := s.store.GetRuntimeSession(ctx, sessionID)
	if err != nil || !isLegacyUntitledSessionTitle(session.Title, content) {
		return err
	}
	_, err = s.updateSessionTitle(ctx, sessionID, fallbackSessionTitle(content), false)
	return err
}

func (s *Service) ensureGeneratedSessionTitle(ctx context.Context, sessionID string, model *domain.ModelRef) {
	session, err := s.store.GetRuntimeSession(ctx, sessionID)
	if err != nil {
		return
	}
	events, err := s.store.ListSessionEvents(ctx, sessionID, false, 20)
	if err != nil {
		return
	}
	var userText string
	var userCount int
	for _, event := range events {
		if event.Type != domain.EventTypeUserMessage || strings.TrimSpace(event.Content) == "" {
			continue
		}
		userCount++
		userText = event.Content
	}
	if userCount != 1 || !isDefaultSessionTitle(session.Title, userText) {
		return
	}
	title := ""
	if s.titleGenerator != nil && model != nil && strings.TrimSpace(model.ModelID) != "" {
		generated, err := s.titleGenerator(ctx, userText, model)
		if err != nil {
			fmt.Printf("session title generation failed for %s/%s: %v\n", model.ProviderID, model.ModelID, err)
		}
		title = cleanGeneratedSessionTitle(generated)
	}
	if title == "" {
		return
	}
	_, _ = s.updateSessionTitle(context.Background(), sessionID, title, true)
}

func (s *Service) generateSessionTitle(ctx context.Context, userText string, model *domain.ModelRef) (string, error) {
	systemPrompt, systemErr := s.renderManagedPrompt("auxiliary.title.system", nil)
	userPrompt, userErr := s.renderManagedPrompt("auxiliary.title.user", map[string]string{"content": strings.TrimSpace(userText)})
	if systemErr != nil || userErr != nil {
		return "", errors.New("title prompts are unavailable")
	}
	messages := []domain.ChatMessage{
		{Role: "system", Text: systemPrompt},
		{Role: "user", Text: userPrompt},
	}
	var failures []string
	for _, titleModel := range s.resolveAuxiliaryModels(ctx, model) {
		title, _, err := s.GenerateChatReply(ctx, messages, &titleModel, "medium", "default")
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s/%s: %v", titleModel.ProviderID, titleModel.ModelID, err))
			continue
		}
		title = cleanGeneratedSessionTitle(title)
		if title != "" {
			return title, nil
		}
		failures = append(failures, fmt.Sprintf("%s/%s: empty title", titleModel.ProviderID, titleModel.ModelID))
	}
	if len(failures) > 0 {
		return "", errors.New(strings.Join(failures, "; "))
	}
	return "", nil
}

func (s *Service) generatedSummary(ctx context.Context, input domain.CreateSummaryRequest) string {
	events, previous, err := s.summaryEvents(ctx, input)
	if err != nil || len(events) == 0 {
		return ""
	}
	transcript := renderEventsForSummary(events)
	if previous != nil && strings.TrimSpace(previous.Summary) != "" {
		transcript = "Previous durable summary:\n" + bounded(previous.Summary, 4000) + "\n\nNew conversation range:\n" + transcript
	}
	if strings.TrimSpace(transcript) == "" {
		return ""
	}
	systemPrompt, systemErr := s.renderManagedPrompt("auxiliary.summary.system", nil)
	userPrompt, userErr := s.renderManagedPrompt("auxiliary.summary.user", map[string]string{"content": transcript})
	if systemErr != nil || userErr != nil {
		return ""
	}
	for _, model := range s.configuredAuxiliaryModels(ctx) {
		summary, _, err := s.GenerateChatReply(ctx, []domain.ChatMessage{
			{Role: "system", Text: systemPrompt},
			{Role: "user", Text: userPrompt},
		}, &model, "medium", "default")
		if err == nil && strings.TrimSpace(summary) != "" {
			return bounded(strings.TrimSpace(stripThinkBlocks(summary)), 4000)
		}
	}
	return ""
}

func (s *Service) summaryEvents(ctx context.Context, input domain.CreateSummaryRequest) ([]domain.SessionEvent, *domain.SessionSummary, error) {
	events, err := s.store.ListSessionEvents(ctx, input.SessionID, false, 500)
	if err != nil {
		return nil, nil, err
	}
	var previous *domain.SessionSummary
	if strings.TrimSpace(input.FromEventID) != "" {
		previous, _ = s.store.LatestSummary(ctx, input.SessionID)
	}
	if strings.TrimSpace(input.FromEventID) == "" && strings.TrimSpace(input.ToEventID) == "" {
		if len(events) > 80 {
			events = events[len(events)-80:]
		}
		return events, nil, nil
	}
	start := 0
	if input.FromEventID != "" {
		found := false
		for i, event := range events {
			if event.ID == input.FromEventID {
				start = i + 1
				found = true
				break
			}
		}
		if !found {
			return nil, previous, errors.New("summary start boundary was not found")
		}
	}
	end := len(events)
	if input.ToEventID != "" {
		found := false
		for i := start; i < len(events); i++ {
			if events[i].ID == input.ToEventID {
				end = i + 1
				found = true
				break
			}
		}
		if !found {
			return nil, previous, errors.New("summary end boundary was not found")
		}
	}
	return events[start:end], previous, nil
}

func renderEventsForSummary(events []domain.SessionEvent) string {
	parts := make([]string, 0, len(events))
	for _, event := range events {
		if event.Type == domain.EventTypeSummary || event.Payload["kind"] == "context_compacted" {
			continue
		}
		content := strings.TrimSpace(event.Content)
		if content == "" {
			continue
		}
		role := strings.TrimSpace(event.Role)
		if role == "" {
			role = event.Type
		}
		parts = append(parts, role+": "+bounded(content, 1200))
	}
	return bounded(strings.Join(parts, "\n\n"), 16000)
}

func (s *Service) configuredAuxiliaryModels(ctx context.Context) []domain.ModelRef {
	cfg, err := s.AppConfig(ctx)
	if err != nil || cfg.AuxiliaryModel == nil || strings.TrimSpace(cfg.AuxiliaryModel.ModelID) == "" {
		return nil
	}
	return s.resolveAuxiliaryModels(ctx, nil)
}

func (s *Service) resolveAuxiliaryModels(ctx context.Context, fallback *domain.ModelRef) []domain.ModelRef {
	cfg, err := s.AppConfig(ctx)
	if err == nil && cfg.AuxiliaryModel != nil && strings.TrimSpace(cfg.AuxiliaryModel.ModelID) != "" {
		auxiliary := *cfg.AuxiliaryModel
		models := []domain.ModelRef{auxiliary}
		for _, model := range s.resolveTitleModels(ctx, fallback) {
			if model != auxiliary {
				models = append(models, model)
			}
		}
		return models
	}
	return s.resolveTitleModels(ctx, fallback)
}

func (s *Service) resolveTitleModels(ctx context.Context, fallback *domain.ModelRef) []domain.ModelRef {
	if fallback == nil || strings.TrimSpace(fallback.ModelID) == "" {
		cfg, err := s.AppConfig(ctx)
		if err != nil || cfg.DefaultModel == nil || strings.TrimSpace(cfg.DefaultModel.ModelID) == "" {
			return nil
		}
		fallback = cfg.DefaultModel
	}
	fallbackModel := domain.ModelRef{
		ProviderID: strings.TrimSpace(fallback.ProviderID),
		ModelID:    strings.TrimSpace(fallback.ModelID),
	}
	catalog, err := s.Catalog(ctx)
	if err != nil {
		return []domain.ModelRef{fallbackModel}
	}
	for _, provider := range catalog.Providers {
		if provider.ID != fallbackModel.ProviderID || !provider.Connected {
			continue
		}
		if modelID := smallModelIDForTitleProvider(provider); modelID != "" {
			titleModel := domain.ModelRef{ProviderID: provider.ID, ModelID: modelID}
			if titleModel == fallbackModel {
				return []domain.ModelRef{fallbackModel}
			}
			return []domain.ModelRef{titleModel, fallbackModel}
		}
		break
	}
	return []domain.ModelRef{fallbackModel}
}

func smallModelIDForTitleProvider(provider domain.ProviderInfo) string {
	priority := []string{
		"claude-haiku-4-5",
		"claude-haiku-4.5",
		"3-5-haiku",
		"3.5-haiku",
		"gemini-3-flash",
		"gemini-2.5-flash",
		"gpt-5.4-mini",
		"gpt-5-mini",
	}
	if strings.HasPrefix(provider.ID, "opencode") {
		priority = []string{"gpt-5.4-mini", "gpt-5-mini"}
	}
	if strings.HasPrefix(provider.ID, "github-copilot") {
		priority = append([]string{"gpt-5-mini", "claude-haiku-4.5"}, priority...)
	}
	for _, item := range priority {
		if provider.ID == "amazon-bedrock" {
			if match := smallBedrockTitleModelID(provider.Models, item); match != "" {
				return match
			}
			continue
		}
		for _, model := range provider.Models {
			if strings.Contains(model.ID, item) {
				return model.ID
			}
		}
	}
	return ""
}

func smallBedrockTitleModelID(models []domain.ModelInfo, item string) string {
	var candidates []string
	for _, model := range models {
		if strings.Contains(model.ID, item) {
			candidates = append(candidates, model.ID)
		}
	}
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, "global.") {
			return candidate
		}
	}
	for _, candidate := range candidates {
		if !strings.HasPrefix(candidate, "global.") && !strings.HasPrefix(candidate, "us.") && !strings.HasPrefix(candidate, "eu.") {
			return candidate
		}
	}
	if len(candidates) > 0 {
		return candidates[0]
	}
	return ""
}

func (s *Service) updateSessionTitle(ctx context.Context, sessionID string, title string, verifyDefault bool) (domain.Session, error) {
	title = cleanGeneratedSessionTitle(title)
	if title == "" {
		return domain.Session{}, errors.New("title is empty")
	}
	if verifyDefault {
		current, err := s.store.GetRuntimeSession(ctx, sessionID)
		if err != nil {
			return domain.Session{}, err
		}
		events, err := s.store.ListSessionEvents(ctx, sessionID, false, 20)
		if err != nil {
			return domain.Session{}, err
		}
		firstUser := ""
		for _, event := range events {
			if event.Type == domain.EventTypeUserMessage && strings.TrimSpace(event.Content) != "" {
				firstUser = event.Content
				break
			}
		}
		if !isDefaultSessionTitle(current.Title, firstUser) {
			return current, nil
		}
	}
	updated, err := s.store.UpdateRuntimeSession(ctx, domain.UpdateSessionRequest{SessionID: sessionID, Title: title})
	if err == nil && s.onSessionUpdated != nil {
		s.onSessionUpdated(sessionID, &updated)
	}
	return updated, err
}

func isLegacyUntitledSessionTitle(title string, firstUserText string) bool {
	title = strings.TrimSpace(title)
	switch title {
	case "", "Untitled session", "生成第一版MVP":
		return true
	}
	first := fallbackSessionTitle(firstUserText)
	return first != "" && title == first
}

func isDefaultSessionTitle(title string, firstUserText string) bool {
	title = strings.TrimSpace(title)
	switch title {
	case "", "Untitled session", "New chat", "生成第一版MVP":
		return true
	}
	if strings.HasPrefix(title, "New session - ") {
		return true
	}
	first := fallbackSessionTitle(firstUserText)
	return first != "" && title == first
}

func fallbackSessionTitle(text string) string {
	return cleanGeneratedSessionTitle(bounded(strings.TrimSpace(text), 80))
}

func cleanGeneratedSessionTitle(text string) string {
	text = strings.TrimSpace(stripThinkBlocks(text))
	text = strings.Trim(text, "\"'`“”‘’")
	for _, line := range strings.Split(text, "\n") {
		title := strings.TrimSpace(line)
		title = strings.Trim(title, "\"'`“”‘’")
		if title == "" {
			continue
		}
		runes := []rune(title)
		if len(runes) > 50 {
			title = string(runes[:50])
		}
		return title
	}
	return ""
}

func stripThinkBlocks(text string) string {
	for {
		lower := strings.ToLower(text)
		start := strings.Index(lower, "<think>")
		if start < 0 {
			return text
		}
		rest := text[start+len("<think>"):]
		end := strings.Index(strings.ToLower(rest), "</think>")
		if end < 0 {
			return strings.TrimSpace(text[:start])
		}
		text = text[:start] + rest[end+len("</think>"):]
	}
}
