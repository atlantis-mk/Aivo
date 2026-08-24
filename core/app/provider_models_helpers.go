package app

import (
	"errors"
	"net"

	"aivo/core/domain"
)

func isOAuthCredential(credential llmCredential) bool {
	return credential.Method == "oauth-browser" || credential.Method == "oauth-headless" || credential.Method == "oauth"
}

func isGoogleProvider(provider domain.ProviderConfig) bool {
	return inferTransport(provider.ID, provider.Type, provider.BaseURL) == TransportGoogleGemini
}

func isAnthropicProvider(provider domain.ProviderConfig) bool {
	return inferTransport(provider.ID, provider.Type, provider.BaseURL) == TransportAnthropicMessages
}

func isTimeoutError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
