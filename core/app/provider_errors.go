package app

import (
	"strings"
)

func safeProviderError(err error) string {
	if err == nil {
		return ""
	}
	text := strings.TrimSpace(err.Error())
	if text == "" {
		return "provider validation failed"
	}
	lower := strings.ToLower(text)
	if strings.Contains(lower, "unauthorized") || strings.Contains(lower, "forbidden") || strings.Contains(lower, "authentication failed") {
		return "authentication failed"
	}
	if strings.Contains(lower, "timed out") || strings.Contains(lower, "timeout") {
		return "provider request timed out"
	}
	if strings.Contains(lower, "no such host") || strings.Contains(lower, "connection refused") {
		return "provider endpoint could not be reached"
	}
	if len(text) > 240 {
		text = text[:240]
	}
	return redactProviderSecretFragments(text)
}

func redactProviderSecretFragments(text string) string {
	fields := strings.Fields(text)
	for i, field := range fields {
		lower := strings.ToLower(field)
		if strings.Contains(lower, "sk-") || strings.Contains(lower, "token") || strings.Contains(lower, "key=") || strings.Contains(lower, "authorization") {
			fields[i] = "[redacted]"
		}
	}
	return strings.Join(fields, " ")
}
