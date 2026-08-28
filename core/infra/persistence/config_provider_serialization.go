package persistence

import (
	"encoding/json"
	"strings"

	"aivo/core/domain"
)

func encodeHeaders(headers map[string]string) string {
	if len(headers) == 0 {
		return ""
	}
	raw, err := json.Marshal(headers)
	if err != nil {
		return ""
	}
	return string(raw)
}

func decodeHeaders(raw string) map[string]string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var headers map[string]string
	if err := json.Unmarshal([]byte(raw), &headers); err != nil {
		return nil
	}
	return headers
}

func encodeModelRefs(models []domain.ModelRef) string {
	if len(models) == 0 {
		return ""
	}
	raw, err := json.Marshal(models)
	if err != nil {
		return ""
	}
	return string(raw)
}

func encodeModelRef(model *domain.ModelRef) string {
	if model == nil || strings.TrimSpace(model.ProviderID) == "" || strings.TrimSpace(model.ModelID) == "" {
		return ""
	}
	return encodeModelRefs([]domain.ModelRef{*model})
}

func decodeModelRef(raw string) *domain.ModelRef {
	models := decodeModelRefs(raw)
	if len(models) == 0 {
		return nil
	}
	model := models[0]
	return &model
}

func decodeModelRefs(raw string) []domain.ModelRef {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var models []domain.ModelRef
	if err := json.Unmarshal([]byte(raw), &models); err != nil {
		return nil
	}
	return models
}

func encodeProviderRuntimePolicy(policy domain.ProviderRuntimePolicy) string {
	raw, err := json.Marshal(policy)
	if err != nil {
		return ""
	}
	return string(raw)
}

func decodeProviderRuntimePolicy(raw string) domain.ProviderRuntimePolicy {
	if strings.TrimSpace(raw) == "" {
		return defaultProviderRuntimePolicy()
	}
	var policy domain.ProviderRuntimePolicy
	if err := json.Unmarshal([]byte(raw), &policy); err != nil {
		return defaultProviderRuntimePolicy()
	}
	return normalizeProviderRuntimePolicy(policy)
}

func encodeWebSearchConfig(config domain.WebSearchConfig) string {
	config = normalizeWebSearchConfig(config)
	raw, err := json.Marshal(config)
	if err != nil {
		return ""
	}
	return string(raw)
}

func decodeWebSearchConfig(raw string) domain.WebSearchConfig {
	if strings.TrimSpace(raw) == "" {
		return defaultWebSearchConfig()
	}
	var config domain.WebSearchConfig
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return defaultWebSearchConfig()
	}
	return normalizeWebSearchConfig(config)
}

func encodeNativeToolsConfig(config domain.NativeToolsConfig) string {
	config = normalizeNativeToolsConfig(config)
	raw, err := json.Marshal(config)
	if err != nil {
		return ""
	}
	return string(raw)
}

func decodeNativeToolsConfig(raw string) domain.NativeToolsConfig {
	if strings.TrimSpace(raw) == "" {
		return domain.NativeToolsConfig{}
	}
	var config domain.NativeToolsConfig
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return domain.NativeToolsConfig{}
	}
	return normalizeNativeToolsConfig(config)
}

func normalizeNativeToolsConfig(config domain.NativeToolsConfig) domain.NativeToolsConfig {
	config.CodeExecution.FileIDs = normalizeIDList(config.CodeExecution.FileIDs)
	config.FileSearch.VectorStoreIDs = normalizeIDList(config.FileSearch.VectorStoreIDs)
	if len(config.RemoteMCP) > 0 {
		out := make([]domain.NativeMCPToolConfig, 0, len(config.RemoteMCP))
		seen := map[string]bool{}
		for _, server := range config.RemoteMCP {
			server.ServerURL = strings.TrimSpace(server.ServerURL)
			server.ServerLabel = strings.TrimSpace(server.ServerLabel)
			server.AllowedTools = normalizeIDList(server.AllowedTools)
			if !server.Enabled || server.ServerURL == "" || server.ServerLabel == "" {
				continue
			}
			key := server.ServerURL + "\x00" + server.ServerLabel
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, server)
		}
		config.RemoteMCP = out
	}
	return config
}

func normalizeIDList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func normalizeWebSearchConfig(config domain.WebSearchConfig) domain.WebSearchConfig {
	defaults := defaultWebSearchConfig()
	switch strings.TrimSpace(config.Mode) {
	case domain.WebSearchModeDisabled, domain.WebSearchModeCached, domain.WebSearchModeIndexed, domain.WebSearchModeLive:
	default:
		config.Mode = defaults.Mode
	}
	switch strings.TrimSpace(config.Route) {
	case domain.WebSearchRouteAuto, domain.WebSearchRouteLocal, domain.WebSearchRouteProvider:
	default:
		config.Route = defaults.Route
	}
	if strings.TrimSpace(config.LocalProvider) == "" {
		config.LocalProvider = defaults.LocalProvider
	}
	switch strings.TrimSpace(config.SearchContextSize) {
	case "", "low", "medium", "high":
	default:
		config.SearchContextSize = ""
	}
	config.AllowedDomains = normalizeDomainFilters(config.AllowedDomains)
	if config.UserLocation != nil && strings.TrimSpace(config.UserLocation.Type) == "" {
		config.UserLocation.Type = "approximate"
	}
	return config
}

func normalizeDomainFilters(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(strings.ToLower(value))
		value = strings.TrimPrefix(value, "http://")
		value = strings.TrimPrefix(value, "https://")
		value = strings.Trim(value, "/")
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func normalizeProviderRuntimePolicy(policy domain.ProviderRuntimePolicy) domain.ProviderRuntimePolicy {
	defaults := defaultProviderRuntimePolicy()
	if policy.EnableFallback == nil && policy.BufferStreamingFallback == nil && policy.MaxRetries == 0 && policy.RetryBaseDelayMs == 0 && policy.RateLimitCooldownSeconds == 0 {
		return defaults
	}
	if policy.EnableFallback == nil {
		policy.EnableFallback = defaults.EnableFallback
	}
	if policy.BufferStreamingFallback == nil {
		policy.BufferStreamingFallback = defaults.BufferStreamingFallback
	}
	if policy.MaxRetries < 0 {
		policy.MaxRetries = 0
	}
	if policy.MaxRetries > 5 {
		policy.MaxRetries = 5
	}
	if policy.RetryBaseDelayMs <= 0 {
		policy.RetryBaseDelayMs = defaults.RetryBaseDelayMs
	}
	if policy.RetryBaseDelayMs > 5000 {
		policy.RetryBaseDelayMs = 5000
	}
	if policy.RateLimitCooldownSeconds <= 0 {
		policy.RateLimitCooldownSeconds = defaults.RateLimitCooldownSeconds
	}
	if policy.RateLimitCooldownSeconds > 3600 {
		policy.RateLimitCooldownSeconds = 3600
	}
	return policy
}

func encodeModels(models []domain.ModelInfo) string {
	if len(models) == 0 {
		return "[]"
	}
	raw, err := json.Marshal(models)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func decodeModels(raw string) []domain.ModelInfo {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var models []domain.ModelInfo
	if err := json.Unmarshal([]byte(raw), &models); err != nil {
		return nil
	}
	return models
}
