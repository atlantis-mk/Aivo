package app

import "strings"

func isProviderExecutionUnavailable(err error) bool {
	if err == nil {
		return false
	}
	text := err.Error()
	return strings.Contains(text, "provider is not configured") ||
		strings.Contains(text, "model is not configured") ||
		strings.Contains(text, "credentials are not configured") ||
		strings.Contains(text, "credentials are missing") ||
		strings.Contains(text, "base URL is not configured")
}

func deterministicModelUnavailableFallback(userText string) string {
	text := strings.TrimSpace(userText)
	if text == "" {
		return "I recorded your request. Configure a model provider to generate a full assistant response."
	}
	if len(text) > 120 {
		text = text[:120]
	}
	return "I recorded your request and saved it to this session. Configure a model provider to continue with an AI-generated response. Request: " + text
}
