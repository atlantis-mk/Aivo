package app

import (
	"context"

	"aivo/core/domain"
)

func (s *Service) ResolveModelRoutes(ctx context.Context, cfg domain.AppConfig, requestedModel *domain.ModelRef) ([]ResolvedModelRoute, error) {
	primary, err := s.ResolveModelRoute(ctx, cfg, requestedModel)
	if err != nil {
		return nil, err
	}
	routes := []ResolvedModelRoute{primary}
	if requestedModel != nil {
		return routes, nil
	}
	seen := map[string]bool{modelRouteKey(primary.Model): true}
	for _, fallback := range cfg.FallbackModels {
		if fallback.ProviderID == "" || fallback.ModelID == "" {
			continue
		}
		key := modelRouteKey(fallback)
		if seen[key] {
			continue
		}
		route, err := s.ResolveModelRoute(ctx, cfg, &fallback)
		if err != nil {
			continue
		}
		seen[modelRouteKey(route.Model)] = true
		routes = append(routes, route)
	}
	return routes, nil
}

func modelRouteKey(model domain.ModelRef) string {
	return normalizeProviderID(model.ProviderID) + "\x00" + normalizeModelIDForProvider(model.ProviderID, model.ModelID)
}

func fallbackAllowed(err error, emittedOutput bool) bool {
	if err == nil || emittedOutput {
		return false
	}
	classified := classifyProviderError(err)
	switch classified.Class {
	case providerErrorBadRequest, providerErrorUnsupported:
		return false
	default:
		return true
	}
}
