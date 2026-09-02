package app

import (
	"context"
	"strings"

	"aivo/core/domain"
)

func (s *Service) UpdateModelPreferences(ctx context.Context, input domain.ModelPreferencesInput) (domain.AppConfig, error) {
	cfg, err := s.AppConfig(ctx)
	if err != nil {
		return domain.AppConfig{}, err
	}
	if input.Model != nil {
		providerID := strings.TrimSpace(input.Model.ProviderID)
		modelID := strings.TrimSpace(input.Model.ModelID)
		if providerID != "" && modelID != "" {
			modelID = normalizeModelIDForProvider(providerID, modelID)
			cfg.DefaultModel = &domain.ModelRef{ProviderID: providerID, ModelID: modelID}
			provider := s.providerConfigForModelRequest(ctx, cfg, providerID, modelID)
			cfg.Provider = &provider
		}
	}
	if input.AuxiliaryModel != nil {
		auxiliaryModel := normalizeOptionalModelRef(*input.AuxiliaryModel)
		cfg.AuxiliaryModel = auxiliaryModel
	}
	if input.FallbackModels != nil {
		cfg.FallbackModels = normalizeFallbackModels(input.FallbackModels, cfg.DefaultModel)
	}
	if input.ProviderPolicy != nil {
		cfg.ProviderPolicy = normalizeProviderRuntimePolicy(*input.ProviderPolicy)
	}
	if strings.TrimSpace(input.ReasoningEffort) != "" {
		cfg.ReasoningEffort = normalizeReasoningEffort(input.ReasoningEffort)
	}
	if strings.TrimSpace(input.ServiceTier) != "" {
		if cfg.DefaultModel != nil && providerSupportsServiceTier(cfg.DefaultModel.ProviderID) {
			cfg.ServiceTier = normalizeServiceTier(input.ServiceTier)
		} else {
			cfg.ServiceTier = "default"
		}
	}
	if strings.TrimSpace(input.DefaultPermissionMode) != "" {
		mode, err := normalizePermissionMode(input.DefaultPermissionMode)
		if err != nil {
			return domain.AppConfig{}, err
		}
		cfg.DefaultPermissionMode = mode
	}
	if input.WebSearch != nil {
		cfg.WebSearch = normalizeWebSearchRuntimeConfig(*input.WebSearch)
	}
	if input.NativeTools != nil {
		cfg.NativeTools = normalizeNativeToolsRuntimeConfig(*input.NativeTools)
	}
	if cfg.ReasoningEffort == "" {
		cfg.ReasoningEffort = "medium"
	}
	if cfg.ServiceTier == "" {
		cfg.ServiceTier = "default"
	}
	if err := s.store.SaveConfig(ctx, cfg); err != nil {
		return domain.AppConfig{}, err
	}
	return s.AppConfig(ctx)
}

func normalizeOptionalModelRef(model domain.ModelRef) *domain.ModelRef {
	providerID := normalizeProviderID(model.ProviderID)
	modelID := normalizeModelIDForProvider(providerID, strings.TrimSpace(model.ModelID))
	if providerID == "" || modelID == "" {
		return nil
	}
	return &domain.ModelRef{ProviderID: providerID, ModelID: modelID}
}

func normalizeFallbackModels(models []domain.ModelRef, primary *domain.ModelRef) []domain.ModelRef {
	out := make([]domain.ModelRef, 0, len(models))
	seen := map[string]bool{}
	if primary != nil {
		seen[normalizeProviderID(primary.ProviderID)+"\x00"+normalizeModelIDForProvider(primary.ProviderID, primary.ModelID)] = true
	}
	for _, model := range models {
		providerID := normalizeProviderID(model.ProviderID)
		modelID := normalizeModelIDForProvider(providerID, strings.TrimSpace(model.ModelID))
		if providerID == "" || modelID == "" {
			continue
		}
		key := providerID + "\x00" + modelID
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, domain.ModelRef{ProviderID: providerID, ModelID: modelID})
	}
	return out
}

func isOAuthMethod(method string) bool {
	switch method {
	case "oauth", "oauth-browser", "browser", "oauth-headless", "headless":
		return true
	default:
		return false
	}
}
