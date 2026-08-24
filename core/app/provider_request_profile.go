package app

import (
	"net/http"
	"strings"

	"aivo/core/domain"
)

var protectedRequestParamKeys = map[string]bool{
	"model":    true,
	"messages": true,
	"input":    true,
	"contents": true,
}

func applyRequestProfile(body map[string]any, profile domain.ProviderRequestProfile, provider domain.ProviderConfig, modelID string) {
	if len(body) == 0 {
		return
	}
	mergeRequestParams(body, profile.Params, false)
	if override, ok := matchingRequestOverride(profile.ModelOverrides, modelID); ok {
		mergeRequestParams(body, override.Params, false)
	}
	mergeRequestParams(body, provider.RequestParams, true)
}

func applyRequestProfileHeaders(req *http.Request, profile domain.ProviderRequestProfile, provider domain.ProviderConfig, modelID string) {
	applySafeRequestHeaders(req, profile.Headers)
	if override, ok := matchingRequestOverride(profile.ModelOverrides, modelID); ok {
		applySafeRequestHeaders(req, override.Headers)
	}
	applyProviderHeaders(req, provider.Headers)
}

func matchingRequestOverride(overrides map[string]domain.ProviderRequestOverride, modelID string) (domain.ProviderRequestOverride, bool) {
	if len(overrides) == 0 {
		return domain.ProviderRequestOverride{}, false
	}
	modelID = strings.ToLower(strings.TrimSpace(modelID))
	if modelID == "" {
		return domain.ProviderRequestOverride{}, false
	}
	if override, ok := overrides[modelID]; ok {
		return override, true
	}
	for pattern, override := range overrides {
		pattern = strings.ToLower(strings.TrimSpace(pattern))
		if pattern == "" {
			continue
		}
		if strings.HasSuffix(pattern, "*") && strings.HasPrefix(modelID, strings.TrimSuffix(pattern, "*")) {
			return override, true
		}
		if strings.HasPrefix(pattern, "*") && strings.HasSuffix(modelID, strings.TrimPrefix(pattern, "*")) {
			return override, true
		}
		if strings.Contains(modelID, pattern) {
			return override, true
		}
	}
	return domain.ProviderRequestOverride{}, false
}

func mergeRequestParams(dst map[string]any, src map[string]any, override bool) {
	for key, value := range src {
		key = strings.TrimSpace(key)
		if key == "" || protectedRequestParamKeys[key] {
			continue
		}
		value = cloneAnyValue(value)
		if existing, ok := dst[key]; ok {
			if override {
				if existingMap, ok := existing.(map[string]any); ok {
					if valueMap, ok := value.(map[string]any); ok {
						mergeRequestParams(existingMap, valueMap, true)
						continue
					}
				}
				dst[key] = value
			}
			continue
		}
		dst[key] = value
	}
}

func applySafeRequestHeaders(req *http.Request, headers map[string]string) {
	for key, value := range headers {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" || isCredentialHeader(key) {
			continue
		}
		req.Header.Set(key, value)
	}
}

func cloneRequestProfile(profile domain.ProviderRequestProfile) domain.ProviderRequestProfile {
	return domain.ProviderRequestProfile{
		Headers:        cloneHeaders(profile.Headers),
		Params:         cloneAnyMap(profile.Params),
		ModelOverrides: cloneRequestOverrides(profile.ModelOverrides),
	}
}

func cloneRequestOverrides(overrides map[string]domain.ProviderRequestOverride) map[string]domain.ProviderRequestOverride {
	if len(overrides) == 0 {
		return nil
	}
	out := make(map[string]domain.ProviderRequestOverride, len(overrides))
	for key, override := range overrides {
		out[key] = domain.ProviderRequestOverride{Headers: cloneHeaders(override.Headers), Params: cloneAnyMap(override.Params)}
	}
	return out
}

func cloneHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	out := make(map[string]string, len(headers))
	for key, value := range headers {
		out[key] = value
	}
	return out
}

func cloneAnyMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = cloneAnyValue(value)
	}
	return out
}

func cloneAnyValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneAnyMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = cloneAnyValue(typed[i])
		}
		return out
	default:
		return typed
	}
}
