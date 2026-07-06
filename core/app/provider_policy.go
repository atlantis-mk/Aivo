package app

import (
	"time"

	"aivo/core/domain"
)

func defaultProviderRuntimePolicy() domain.ProviderRuntimePolicy {
	enableFallback := true
	bufferStreamingFallback := true
	return domain.ProviderRuntimePolicy{
		EnableFallback:           &enableFallback,
		BufferStreamingFallback:  &bufferStreamingFallback,
		MaxRetries:               1,
		RetryBaseDelayMs:         100,
		RateLimitCooldownSeconds: 30,
	}
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

func providerPolicyBool(value *bool) bool {
	return value == nil || *value
}

func retryDelay(policy domain.ProviderRuntimePolicy, attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}
	return time.Duration(attempt*policy.RetryBaseDelayMs) * time.Millisecond
}
