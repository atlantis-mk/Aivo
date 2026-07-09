package app

import (
	"context"
	"net/http"
	"time"

	"aivo/core/domain"
)

const chatGPTCodexModelsURL = "https://chatgpt.com/backend-api/codex/models?client_version=0.0.0"

var providerModelHTTPClient = &http.Client{Timeout: 5 * time.Second}

func providerConfigForRefresh(input domain.ProviderConnectInput) (domain.ProviderConfig, error) {
	provider, _, err := providerConfigFromInput(input)
	return provider, err
}

func (s *Service) fetchProviderModels(ctx context.Context, provider domain.ProviderConfig) ([]domain.ModelInfo, string, error) {
	credential, err := s.resolveCredential(ctx, provider)
	if err != nil {
		return nil, "", err
	}
	if provider.ID == "openai" && isOAuthCredential(credential) {
		return s.fetchChatGPTCodexModels(ctx, provider, credential)
	}
	if isAnthropicProvider(provider) {
		return fetchAnthropicModels(ctx, provider, credential)
	}
	if isGoogleProvider(provider) {
		return fetchGoogleModels(ctx, provider, credential)
	}
	return fetchOpenAICompatibleModels(ctx, provider, credential)
}

type codexModelsResponse struct {
	Models []codexModelInfo `json:"models"`
}

type codexModelInfo struct {
	ID             string  `json:"id"`
	Slug           string  `json:"slug"`
	Name           string  `json:"name"`
	DisplayName    string  `json:"display_name"`
	Visibility     string  `json:"visibility"`
	Priority       float64 `json:"priority"`
	SupportedInAPI *bool   `json:"supported_in_api"`
}

type openAIModelsResponse struct {
	Data []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
	} `json:"data"`
}

type googleModelsResponse struct {
	Models []struct {
		Name                       string   `json:"name"`
		DisplayName                string   `json:"displayName"`
		SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
	} `json:"models"`
}
