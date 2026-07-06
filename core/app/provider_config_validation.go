package app

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"aivo/core/domain"
)

var envNamePattern = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

func providerConfigFromInput(input domain.ProviderConnectInput) (domain.ProviderConfig, ProviderDefinition, error) {
	return defaultProviderRegistry.providerConfigFromInput(input)
}

func (s *Service) providerConfigFromInput(input domain.ProviderConnectInput) (domain.ProviderConfig, ProviderDefinition, error) {
	registry := defaultProviderRegistry
	if s != nil && s.providers != nil {
		registry = s.providers
	}
	return registry.providerConfigFromInput(input)
}

func (r *ProviderRegistry) providerConfigFromInput(input domain.ProviderConnectInput) (domain.ProviderConfig, ProviderDefinition, error) {
	providerID := normalizeProviderID(input.ProviderID)
	if r != nil {
		providerID = r.Normalize(input.ProviderID)
	}
	if providerID == "" {
		return domain.ProviderConfig{}, ProviderDefinition{}, errors.New("provider is required")
	}
	def, known := r.Definition(providerID)
	baseURL := strings.TrimSpace(input.BaseURL)
	if baseURL == "" && known {
		baseURL = def.DefaultBaseURL
	}
	providerType := strings.TrimSpace(input.Type)
	if providerType == "" {
		if known {
			providerType = string(def.Transport)
		} else {
			providerType = string(inferTransport(providerID, "", baseURL))
		}
	}
	transport := inferTransport(providerID, providerType, baseURL)
	if providerType == "openai-compatible" {
		providerType = string(TransportOpenAICompatible)
	}
	if known {
		def.Transport = transport
	} else {
		def = providerDefinitionForConfig(domain.ProviderConfig{
			ID:      providerID,
			Type:    string(transport),
			BaseURL: baseURL,
			Model:   strings.TrimSpace(input.ModelID),
		})
	}
	apiKeyEnv := strings.TrimSpace(input.APIKeyEnv)
	if apiKeyEnv == "" && len(def.APIKeyEnvVars) > 0 {
		apiKeyEnv = def.APIKeyEnvVars[0]
	}
	cfg := domain.ProviderConfig{
		ID:            providerID,
		Type:          string(transport),
		BaseURL:       baseURL,
		APIKey:        strings.TrimSpace(input.APIKey),
		APIKeyEnv:     apiKeyEnv,
		Model:         strings.TrimSpace(input.ModelID),
		Headers:       normalizeHeaders(input.Headers),
		RequestParams: cloneAnyMap(input.RequestParams),
	}
	if cfg.Model == "" {
		cfg.Model = def.DefaultModelID
	}
	if err := validateProviderConfig(cfg, def, strings.TrimSpace(input.Method)); err != nil {
		return domain.ProviderConfig{}, ProviderDefinition{}, err
	}
	return cfg, def, nil
}

func validateProviderConfig(provider domain.ProviderConfig, def ProviderDefinition, method string) error {
	if strings.TrimSpace(provider.ID) == "" {
		return errors.New("provider is required")
	}
	if !validTransport(TransportType(provider.Type)) {
		return fmt.Errorf("unsupported provider transport: %s", provider.Type)
	}
	if err := validateProviderBaseURL(provider, def); err != nil {
		return err
	}
	if strings.TrimSpace(provider.Model) == "" {
		return errors.New("model is required")
	}
	if strings.TrimSpace(provider.APIKeyEnv) != "" {
		for _, envName := range splitEnvCandidates(provider.APIKeyEnv) {
			if !envNamePattern.MatchString(envName) {
				return fmt.Errorf("invalid credential environment variable: %s", envName)
			}
		}
	}
	if err := validateProviderHeaders(provider.Headers); err != nil {
		return err
	}
	if err := validateProviderRequestParams(provider.RequestParams); err != nil {
		return err
	}
	if method != "" {
		if err := validateProviderAuthMethod(def, method); err != nil {
			return err
		}
	}
	return nil
}

func validateProviderRequestParams(params map[string]any) error {
	if len(params) == 0 {
		return nil
	}
	if _, ok := params["model"]; ok {
		return errors.New("requestParams cannot override model")
	}
	if _, ok := params["messages"]; ok {
		return errors.New("requestParams cannot override messages")
	}
	if _, ok := params["input"]; ok {
		return errors.New("requestParams cannot override input")
	}
	if _, ok := params["contents"]; ok {
		return errors.New("requestParams cannot override contents")
	}
	return nil
}

func validateProviderBaseURL(provider domain.ProviderConfig, def ProviderDefinition) error {
	baseURL := strings.TrimSpace(provider.BaseURL)
	if baseURL == "" {
		if def.DefaultBaseURL != "" {
			return nil
		}
		if containsAuthType(def.AuthTypes, AuthNone) {
			return nil
		}
		return errors.New("provider base URL is required")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return errors.New("provider base URL must be an absolute URL")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return errors.New("provider base URL must use http or https")
	}
	if parsed.Scheme == "http" && !isLoopbackHost(parsed.Hostname()) {
		return errors.New("plain http provider base URL is only allowed for local endpoints")
	}
	if strings.Contains(parsed.RawQuery, "key=") || strings.Contains(strings.ToLower(parsed.RawQuery), "token=") {
		return errors.New("provider base URL must not contain credentials")
	}
	if parsed.User != nil {
		return errors.New("provider base URL must not contain user info")
	}
	return nil
}

func validateProviderHeaders(headers map[string]string) error {
	for key, value := range headers {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		if !validHeaderName(key) {
			return fmt.Errorf("invalid provider header name: %s", key)
		}
		if isCredentialHeader(key) {
			return fmt.Errorf("provider header %q must be configured as a credential, not a static header", key)
		}
	}
	return nil
}

func isCredentialHeader(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	return lower == "authorization" ||
		lower == "x-api-key" ||
		lower == "api-key" ||
		lower == "cookie" ||
		strings.Contains(lower, "token") ||
		strings.Contains(lower, "secret")
}

func validateProviderAuthMethod(def ProviderDefinition, method string) error {
	method = strings.TrimSpace(method)
	if method == "env" || method == "none" {
		return nil
	}
	switch method {
	case "api-key":
		if containsAuthType(def.AuthTypes, AuthAPIKey) {
			return nil
		}
	case "oauth", "oauth-browser", "browser":
		if containsAuthType(def.AuthTypes, AuthOAuthBrowser) {
			return nil
		}
	case "oauth-headless", "headless":
		if containsAuthType(def.AuthTypes, AuthOAuthDevice) {
			return nil
		}
	case "external-process":
		if containsAuthType(def.AuthTypes, AuthExternalProcess) {
			return nil
		}
	case "aws-sdk":
		if containsAuthType(def.AuthTypes, AuthAWSSDK) {
			return nil
		}
	}
	return fmt.Errorf("provider %q does not support auth method %q", def.ID, method)
}

func validTransport(transport TransportType) bool {
	switch transport {
	case TransportOpenAIResponses, TransportOpenAIChat, TransportOpenAICompatible, TransportAzureOpenAI, TransportAnthropicMessages, TransportGoogleGemini, TransportGoogleVertex, TransportBedrockConverse, TransportGitHubCopilot, TransportExternalProcess:
		return true
	default:
		return false
	}
}

func validHeaderName(name string) bool {
	return http.CanonicalHeaderKey(name) != "" && !strings.ContainsAny(name, " \t\r\n:")
}

func isLoopbackHost(host string) bool {
	host = strings.ToLower(strings.Trim(host, "[]"))
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
