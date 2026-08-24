package app

import (
	"context"
	"errors"
	"strings"

	"aivo/core/domain"
)

func (s *Service) promptRegistry() (*PromptRegistry, error) {
	if s == nil || s.prompts == nil || strings.TrimSpace(s.prompts.Root()) == "" {
		return nil, errors.New("prompt persistence is unavailable")
	}
	return s.prompts, nil
}

func (s *Service) ListPromptDocuments(context.Context) ([]domain.PromptDocument, error) {
	if s == nil || s.prompts == nil {
		return nil, errors.New("prompt catalog is unavailable")
	}
	return s.prompts.List(), nil
}

func (s *Service) GetPromptDocument(_ context.Context, id string) (domain.PromptDocument, error) {
	if s == nil || s.prompts == nil {
		return domain.PromptDocument{}, errors.New("prompt catalog is unavailable")
	}
	return s.prompts.Get(id)
}

func (s *Service) ValidatePromptDraft(_ context.Context, input domain.PromptDocumentInput) (domain.PromptValidationResult, error) {
	if s == nil || s.prompts == nil {
		return domain.PromptValidationResult{}, errors.New("prompt catalog is unavailable")
	}
	return s.prompts.Validate(input), nil
}

func (s *Service) SavePromptDocument(_ context.Context, input domain.PromptDocumentInput) (domain.PromptDocument, error) {
	registry, err := s.promptRegistry()
	if err != nil {
		return domain.PromptDocument{}, err
	}
	return registry.Save(input)
}

func (s *Service) ResetPromptDocument(_ context.Context, id string) (domain.PromptDocument, error) {
	registry, err := s.promptRegistry()
	if err != nil {
		return domain.PromptDocument{}, err
	}
	return registry.Reset(id)
}

func (s *Service) SetPromptDocumentEnabled(_ context.Context, input domain.PromptEnabledInput) (domain.PromptDocument, error) {
	registry, err := s.promptRegistry()
	if err != nil {
		return domain.PromptDocument{}, err
	}
	return registry.SetEnabled(input.ID, input.Enabled)
}

func (s *Service) DeletePromptDocument(ctx context.Context, id string) error {
	registry, err := s.promptRegistry()
	if err != nil {
		return err
	}
	document, err := registry.Get(id)
	if err != nil {
		return err
	}
	if document.Category == domain.PromptCategoryAgent {
		modeID := strings.TrimPrefix(document.ID, "agent.")
		if document.Deletable {
			return s.DeleteAgentMode(ctx, modeID)
		}
		return errors.New("built-in agent prompts cannot be deleted")
	}
	return registry.Delete(id)
}

func (s *Service) ReloadPromptCatalog(context.Context) ([]domain.PromptDocument, error) {
	registry, err := s.promptRegistry()
	if err != nil {
		return nil, err
	}
	if err := registry.Reload(); err != nil {
		return nil, err
	}
	return registry.List(), nil
}

func (s *Service) PromptDirectory(context.Context) (string, error) {
	registry, err := s.promptRegistry()
	if err != nil {
		return "", err
	}
	return registry.Root(), nil
}

func (s *Service) CreateAgentPrompt(ctx context.Context, input domain.CreateAgentPromptInput) (domain.AgentModeDefinition, error) {
	modeID, err := domain.NormalizeAgentMode(strings.TrimPrefix(strings.TrimSpace(input.ID), "agent."))
	if err != nil {
		return domain.AgentModeDefinition{}, err
	}
	promptID := "agent." + modeID
	registry, err := s.promptRegistry()
	if err != nil {
		return domain.AgentModeDefinition{}, err
	}
	if _, existingErr := registry.Get(promptID); existingErr == nil {
		return domain.AgentModeDefinition{}, errors.New("prompt id already exists")
	}
	if _, err := registry.Save(domain.PromptDocumentInput{ID: promptID, Category: domain.PromptCategoryAgent, Title: input.Title, Body: input.Body, Enabled: true}); err != nil {
		return domain.AgentModeDefinition{}, err
	}
	validated := registry.Validate(domain.PromptDocumentInput{ID: promptID, Category: domain.PromptCategoryAgent, Title: input.Title, Body: input.Body, Enabled: true})
	if !validated.Valid {
		_ = registry.Delete(promptID)
		return domain.AgentModeDefinition{}, errors.New("agent prompt must be valid before the agent mode is created")
	}
	mode, err := s.SaveAgentMode(ctx, domain.AgentModeDefinition{
		ID: modeID, DisplayName: input.Title, Description: input.Description,
		PromptID: promptID, Prompt: input.Body,
		PermissionScope: firstNonEmpty(input.PermissionScope, "read_only"),
		Mode:            firstNonEmpty(input.Mode, "all"), Subagents: append([]string(nil), input.Subagents...),
	})
	if err != nil {
		_ = registry.Delete(promptID)
		return domain.AgentModeDefinition{}, err
	}
	return mode, nil
}

func (s *Service) CreateQuickPrompt(_ context.Context, input domain.CreateQuickPromptInput) (domain.PromptDocument, error) {
	id := strings.TrimSpace(input.ID)
	if !strings.HasPrefix(id, "quick.") {
		id = "quick." + id
	}
	registry, err := s.promptRegistry()
	if err != nil {
		return domain.PromptDocument{}, err
	}
	if _, existingErr := registry.Get(id); existingErr == nil {
		return domain.PromptDocument{}, errors.New("prompt id already exists")
	}
	return registry.Save(domain.PromptDocumentInput{ID: id, Category: domain.PromptCategoryQuickPrompt, Title: input.Title, Body: input.Body, Enabled: true})
}

func (s *Service) ListPromptToolDescriptions(_ context.Context) ([]domain.PromptToolDescription, error) {
	registry, _ := s.toolsForWorkspace("")
	if registry == nil {
		return []domain.PromptToolDescription{}, nil
	}
	specs := registry.Specs()
	out := make([]domain.PromptToolDescription, 0, len(specs))
	for _, spec := range specs {
		out = append(out, domain.PromptToolDescription{Name: spec.Name, Description: spec.Description, Category: "tool", Source: "core"})
	}
	return out, nil
}
