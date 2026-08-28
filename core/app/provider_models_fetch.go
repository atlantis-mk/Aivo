package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"aivo/core/domain"
)

func (s *Service) fetchChatGPTCodexModels(ctx context.Context, provider domain.ProviderConfig, credential llmCredential) ([]domain.ModelInfo, string, error) {
	access, accountID, err := s.validOpenAIAccessToken(ctx, credential)
	if err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, chatGPTCodexModelsURL, nil)
	if err != nil {
		return nil, "", err
	}
	applyProviderHeaders(req, provider.Headers)
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", openAIUserAgent)
	if accountID != "" {
		req.Header.Set("ChatGPT-Account-Id", accountID)
	}
	raw, err := doProviderModelsRequest(req)
	if err != nil {
		return nil, "", err
	}
	models, defaultModel, err := parseCodexModels(raw, provider.ID)
	if err != nil {
		return nil, "", err
	}
	return models, defaultModel, nil
}

func fetchOpenAICompatibleModels(ctx context.Context, provider domain.ProviderConfig, credential llmCredential) ([]domain.ModelInfo, string, error) {
	endpoint, err := providerModelsEndpoint(provider.BaseURL)
	if err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", err
	}
	applyProviderHeaders(req, provider.Headers)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", openAIUserAgent)
	if credential.APIKey != "" {
		if usesAzureAPIKeyHeader(provider) {
			req.Header.Set("api-key", credential.APIKey)
		} else {
			req.Header.Set("Authorization", "Bearer "+credential.APIKey)
		}
	}
	raw, err := doProviderModelsRequest(req)
	if err != nil {
		return nil, "", err
	}
	models, defaultModel, err := parseOpenAICompatibleModels(raw, provider.ID)
	if err != nil {
		return nil, "", err
	}
	if preferred := preferredDefaultModel(provider.ID, models); preferred != "" {
		defaultModel = preferred
		markRecommended(models, defaultModel)
	}
	return models, defaultModel, nil
}

func fetchMistralModels(ctx context.Context, provider domain.ProviderConfig, credential llmCredential) ([]domain.ModelInfo, string, error) {
	endpoint, err := providerModelsEndpoint(provider.BaseURL)
	if err != nil {
		return nil, "", err
	}
	raw, err := fetchBearerModelCatalog(ctx, provider, credential, endpoint)
	if err != nil {
		return nil, "", err
	}
	models, defaultModel, err := parseMistralModels(raw, provider.ID)
	return finalizeProviderModelCatalog(provider.ID, models, defaultModel, err)
}

func fetchOpenRouterModels(ctx context.Context, provider domain.ProviderConfig, credential llmCredential) ([]domain.ModelInfo, string, error) {
	endpoint, err := providerModelsEndpoint(provider.BaseURL)
	if err != nil {
		return nil, "", err
	}
	raw, err := fetchBearerModelCatalog(ctx, provider, credential, endpoint)
	if err != nil {
		return nil, "", err
	}
	models, defaultModel, err := parseOpenRouterModels(raw, provider.ID)
	return finalizeProviderModelCatalog(provider.ID, models, defaultModel, err)
}

func fetchCerebrasModels(ctx context.Context, provider domain.ProviderConfig, credential llmCredential) ([]domain.ModelInfo, string, error) {
	raw, err := fetchBearerModelCatalog(ctx, provider, credential, cerebrasPublicModelsURL)
	if err != nil {
		return nil, "", err
	}
	models, defaultModel, err := parseCerebrasModels(raw, provider.ID)
	return finalizeProviderModelCatalog(provider.ID, models, defaultModel, err)
}

func fetchBearerModelCatalog(ctx context.Context, provider domain.ProviderConfig, credential llmCredential, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	applyProviderHeaders(req, provider.Headers)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", openAIUserAgent)
	if credential.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+credential.APIKey)
	}
	return doProviderModelsRequest(req)
}

func finalizeProviderModelCatalog(providerID string, models []domain.ModelInfo, defaultModel string, err error) ([]domain.ModelInfo, string, error) {
	if err != nil {
		return nil, "", err
	}
	if preferred := preferredDefaultModel(providerID, models); preferred != "" {
		defaultModel = preferred
		markRecommended(models, defaultModel)
	}
	return models, defaultModel, nil
}

func usesAzureAPIKeyHeader(provider domain.ProviderConfig) bool {
	if normalizeProviderID(provider.ID) == "azure-openai" || provider.Type == string(TransportAzureOpenAI) {
		return true
	}
	parsed, err := url.Parse(strings.TrimSpace(provider.BaseURL))
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return strings.HasSuffix(host, ".openai.azure.com") || strings.HasSuffix(host, ".services.ai.azure.com")
}

func fetchAnthropicModels(ctx context.Context, provider domain.ProviderConfig, credential llmCredential) ([]domain.ModelInfo, string, error) {
	if credential.APIKey == "" {
		return nil, "", fmt.Errorf("credentials are not configured for provider %q", provider.ID)
	}
	endpoint, err := providerModelsEndpoint(provider.BaseURL)
	if err != nil {
		return nil, "", err
	}
	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil {
		return nil, "", err
	}
	query := parsedEndpoint.Query()
	query.Set("limit", "1000")
	parsedEndpoint.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedEndpoint.String(), nil)
	if err != nil {
		return nil, "", err
	}
	applyProviderHeaders(req, provider.Headers)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", openAIUserAgent)
	req.Header.Set("x-api-key", credential.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	raw, err := doProviderModelsRequest(req)
	if err != nil {
		return nil, "", err
	}
	models, defaultModel, err := parseAnthropicModels(raw, provider.ID)
	if err != nil {
		return nil, "", err
	}
	if preferred := preferredDefaultModel(provider.ID, models); preferred != "" {
		defaultModel = preferred
		markRecommended(models, defaultModel)
	}
	return models, defaultModel, nil
}

func fetchGoogleModels(ctx context.Context, provider domain.ProviderConfig, credential llmCredential) ([]domain.ModelInfo, string, error) {
	endpoint, err := providerModelsEndpoint(provider.BaseURL)
	if err != nil {
		return nil, "", err
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, "", err
	}
	if credential.APIKey != "" {
		query := parsed.Query()
		query.Set("key", credential.APIKey)
		parsed.RawQuery = query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, "", err
	}
	applyProviderHeaders(req, provider.Headers)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", openAIUserAgent)
	raw, err := doProviderModelsRequest(req)
	if err != nil {
		return nil, "", err
	}
	models, defaultModel, err := parseGoogleModels(raw, provider.ID)
	if err != nil {
		return nil, "", err
	}
	if preferred := preferredDefaultModel(provider.ID, models); preferred != "" {
		defaultModel = preferred
		markRecommended(models, defaultModel)
	}
	return models, defaultModel, nil
}

func doProviderModelsRequest(req *http.Request) ([]byte, error) {
	resp, err := providerModelHTTPClient.Do(req)
	if err != nil {
		if isTimeoutError(err) {
			return nil, errors.New("provider model refresh timed out")
		}
		return nil, err
	}
	defer resp.Body.Close()
	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, errors.New("authentication failed while refreshing models")
	}
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		return nil, errors.New("provider model endpoint is not supported")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("provider model refresh failed: %s", resp.Status)
	}
	if readErr != nil {
		return nil, readErr
	}
	return raw, nil
}

func providerModelsEndpoint(baseURL string) (string, error) {
	clean := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if clean == "" {
		return "", errors.New("provider base URL is required")
	}
	return clean + "/models", nil
}
