package app

import (
	"context"
	"strings"

	"aivo/core/domain"
)

func isSystemSkillSource(source string) bool {
	switch strings.TrimSpace(source) {
	case domain.SkillSourceAivoSystem, domain.SkillSourceCodexSystem:
		return true
	default:
		return false
	}
}

func skillEntryActions(skill domain.SkillEntry, canActivate bool) []string {
	actions := make([]string, 0, 4)
	if canActivate {
		actions = append(actions, domain.SkillActionActivate)
	}
	if isSystemSkillSource(skill.Source) {
		return actions
	}
	actions = append(actions, domain.SkillActionSetEnabled)
	if skill.Source == domain.SkillSourceAivo {
		actions = append(actions, domain.SkillActionEdit)
	}
	actions = append(actions, domain.SkillActionDelete)
	return actions
}

func (s *Service) skillAvailableForUse(ctx context.Context, skill domain.SkillEntry) bool {
	if !skill.Enabled {
		return false
	}
	if skill.Source == domain.SkillSourceCodexSystem && !s.codexOAuthConfigured(ctx) {
		return false
	}
	requiredTool := strings.TrimSpace(skill.Metadata["aivo.tool"])
	if requiredTool == "" {
		return true
	}
	cfg, err := s.AppConfig(ctx)
	if err != nil {
		return false
	}
	nativeConfig := normalizeNativeToolsRuntimeConfig(cfg.NativeTools)
	if nativeToolDisabled(nativeConfig, requiredTool) {
		return false
	}
	if requiredTool == "image_gen.imagegen" && nativeToolDisabled(nativeConfig, CodexImagegenToolName) {
		return false
	}
	return true
}

func (s *Service) decorateSkillEntry(ctx context.Context, skill domain.SkillEntry) domain.SkillEntry {
	skill.Actions = skillEntryActions(skill, s.skillAvailableForUse(ctx, skill))
	skill.SelectionGroup = skillSelectionGroup(skill)
	return skill
}

func (s *Service) decorateSkillEntries(ctx context.Context, skills []domain.SkillEntry) []domain.SkillEntry {
	out := make([]domain.SkillEntry, 0, len(skills))
	for _, skill := range skills {
		out = append(out, s.decorateSkillEntry(ctx, skill))
	}
	return out
}

func skillSelectionGroup(skill domain.SkillEntry) *domain.ToolSelectionGroup {
	return inferredSkillSelectionGroup(skill)
}

func inferredSkillSelectionGroup(skill domain.SkillEntry) *domain.ToolSelectionGroup {
	name := normalizeSkillName(skill.Name)
	if name == "hyperframes" || strings.HasPrefix(name, "hyperframes-") {
		return &domain.ToolSelectionGroup{
			ID:          "hyperframes",
			Name:        "HyperFrames",
			Description: "HyperFrames video authoring, animation, audio, keyframes, CLI, registry, creative, and composition workflow skills.",
		}
	}
	return nil
}
