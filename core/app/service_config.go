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
	if cfg.Provider != nil {
		provider := cloneProviderConfig(*cfg.Provider)
		if saved, ok := s.savedProviderConfig(ctx, provider.ID); ok {
			provider = mergeMissingProviderConfig(provider, saved)
		}
		provider = s.providerConfigWithDefaults(provider)
		cfg.Provider = &provider
	}
	normalizeLegacyConfigModels(&cfg)
	cfg.Persistence.Configured = true
	cfg.Persistence.JournalEnabled = true
	cfg.Persistence.DualWriteValidation = true
	cfg.ProviderPolicy = normalizeProviderRuntimePolicy(cfg.ProviderPolicy)
	runtime := loadEffectiveRuntimeConfig("")
	cfg.Runtime = runtime.Config
	if len(runtime.Sources) > 0 {
		cfg.ConfigPath = runtime.Sources[len(runtime.Sources)-1].Path
	}
	if cfg.Persistence.ReadPath == "" {
		cfg.Persistence.ReadPath = "sqlite"
	}
	defaultWorkspacePath, err := managedWorkspaceRoot()
	if err != nil {
		return domain.AppConfig{}, err
	}
	cfg.DefaultInitialWorkspacePath = defaultWorkspacePath
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
	permissionMode, err := normalizePermissionMode(cfg.DefaultPermissionMode)
	if err != nil {
		cfg.DefaultPermissionMode = domain.PermissionModeRequestApproval
	} else {
		cfg.DefaultPermissionMode = permissionMode
	}
}

func normalizeReasoningEffort(effort string) string {
	switch strings.TrimSpace(strings.ToLower(effort)) {
	case "none", "minimal", "low", "medium", "high", "xhigh", "max", "ultra":
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
	case "flex":
		return "flex"
	case "default", "":
		return "default"
	default:
		return "default"
	}
}

func providerSupportsServiceTier(providerID string) bool {
	return strings.TrimSpace(providerID) == "openai"
}
