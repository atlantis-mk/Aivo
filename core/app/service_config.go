package app

import (
	"context"
	"strings"

	"aivo/core/domain"
)

func (s *Service) appConfig(ctx context.Context) (domain.AppConfig, error) {
	cfg, err := s.store.LoadConfig(ctx)
	if err != nil {
		return domain.AppConfig{}, err
	}
	normalizeLegacyConfigModels(&cfg)
	cfg.Persistence.Configured = true
	cfg.Persistence.JournalEnabled = true
	cfg.Persistence.DualWriteValidation = true
	cfg.ProviderPolicy = normalizeProviderRuntimePolicy(cfg.ProviderPolicy)
	if cfg.Persistence.ReadPath == "" {
		cfg.Persistence.ReadPath = "sqlite"
	}
	return cfg, nil
}

func normalizeLegacyConfigModels(cfg *domain.AppConfig) {
	if cfg == nil {
		return
	}
	if cfg.Provider != nil && cfg.Provider.ID == "openai" && cfg.Provider.Model == "gpt-5-codex" {
		cfg.Provider.Model = "gpt-5.5"
	}
	if cfg.DefaultModel != nil && cfg.DefaultModel.ProviderID == "openai" && cfg.DefaultModel.ModelID == "gpt-5-codex" {
		cfg.DefaultModel.ModelID = "gpt-5.5"
	}
	if cfg.AuxiliaryModel != nil && cfg.AuxiliaryModel.ProviderID == "openai" && cfg.AuxiliaryModel.ModelID == "gpt-5-codex" {
		cfg.AuxiliaryModel.ModelID = "gpt-5.5"
	}
	cfg.ReasoningEffort = normalizeReasoningEffort(cfg.ReasoningEffort)
	cfg.ServiceTier = normalizeServiceTier(cfg.ServiceTier)
}

func normalizeReasoningEffort(effort string) string {
	switch strings.TrimSpace(strings.ToLower(effort)) {
	case "low", "medium", "high", "ultra":
		return strings.TrimSpace(strings.ToLower(effort))
	case "低":
		return "low"
	case "中":
		return "medium"
	case "高":
		return "high"
	case "超高":
		return "ultra"
	default:
		return "medium"
	}
}

func normalizeServiceTier(serviceTier string) string {
	switch strings.TrimSpace(strings.ToLower(serviceTier)) {
	case "priority", "fast":
		return "priority"
	case "default", "":
		return "default"
	default:
		return "default"
	}
}

func providerSupportsServiceTier(providerID string) bool {
	return strings.TrimSpace(providerID) == "openai"
}
