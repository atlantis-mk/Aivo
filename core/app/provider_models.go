package app

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"aivo/core/domain"
)

const chatGPTCodexModelsURL = "https://chatgpt.com/backend-api/codex/models?client_version=0.0.0"
const cerebrasPublicModelsURL = "https://api.cerebras.ai/public/v1/models"

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
	definition := s.providerDefinitionForConfig(provider)
	switch definition.ModelFetch {
	case ModelFetchAnthropic:
		return fetchAnthropicModels(ctx, provider, credential)
	case ModelFetchMistral:
		return fetchMistralModels(ctx, provider, credential)
	case ModelFetchOpenRouter:
		return fetchOpenRouterModels(ctx, provider, credential)
	case ModelFetchCerebras:
		if definition.BuiltIn && definition.ID == "cerebras" {
			return fetchCerebrasModels(ctx, provider, credential)
		}
		return fetchOpenAICompatibleModels(ctx, provider, credential)
	case ModelFetchGoogle:
		return fetchGoogleModels(ctx, provider, credential)
	default:
		return fetchOpenAICompatibleModels(ctx, provider, credential)
	}
}

type codexModelsResponse struct {
	Models []codexModelInfo `json:"models"`
}

type codexModelInfo struct {
	ID                       string          `json:"id"`
	Slug                     string          `json:"slug"`
	Name                     string          `json:"name"`
	DisplayName              string          `json:"display_name"`
	Description              string          `json:"description"`
	Visibility               string          `json:"visibility"`
	Priority                 float64         `json:"priority"`
	SupportedInAPI           *bool           `json:"supported_in_api"`
	ContextWindow            int             `json:"context_window"`
	MaxContextWindow         int             `json:"max_context_window"`
	AutoCompactTokenLimit    int             `json:"auto_compact_token_limit"`
	DefaultReasoningLevel    json.RawMessage `json:"default_reasoning_level"`
	SupportedReasoningLevels []struct {
		Effort      string `json:"effort"`
		Description string `json:"description"`
	} `json:"supported_reasoning_levels"`
	SupportVerbosity     *bool           `json:"support_verbosity"`
	DefaultVerbosity     json.RawMessage `json:"default_verbosity"`
	AdditionalSpeedTiers []string        `json:"additional_speed_tiers"`
	ServiceTiers         []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"service_tiers"`
	DefaultServiceTier          string          `json:"default_service_tier"`
	InputModalities             []string        `json:"input_modalities"`
	SupportsParallelToolCalls   *bool           `json:"supports_parallel_tool_calls"`
	SupportsImageDetailOriginal *bool           `json:"supports_image_detail_original"`
	UseResponsesLite            *bool           `json:"use_responses_lite"`
	ShellType                   json.RawMessage `json:"shell_type"`
	WebSearchToolType           json.RawMessage `json:"web_search_tool_type"`
	SupportsSearchTool          *bool           `json:"supports_search_tool"`
}

type openAIModelsResponse struct {
	Data []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
	} `json:"data"`
}

type anthropicModelsResponse struct {
	Data []struct {
		ID             string                     `json:"id"`
		DisplayName    string                     `json:"display_name"`
		Capabilities   map[string]json.RawMessage `json:"capabilities"`
		MaxInputTokens int                        `json:"max_input_tokens"`
		MaxTokens      int                        `json:"max_tokens"`
	} `json:"data"`
}

type mistralModelsResponse struct {
	Data []struct {
		ID               string                     `json:"id"`
		Name             string                     `json:"name"`
		MaxContextLength int                        `json:"max_context_length"`
		Archived         bool                       `json:"archived"`
		Capabilities     map[string]json.RawMessage `json:"capabilities"`
	} `json:"data"`
}

type openRouterModelsResponse struct {
	Data []struct {
		ID                  string    `json:"id"`
		Name                string    `json:"name"`
		ContextLength       int       `json:"context_length"`
		SupportedParameters *[]string `json:"supported_parameters"`
		Architecture        struct {
			InputModalities []string `json:"input_modalities"`
		} `json:"architecture"`
		TopProvider struct {
			MaxCompletionTokens int `json:"max_completion_tokens"`
		} `json:"top_provider"`
	} `json:"data"`
}

type cerebrasModelsResponse struct {
	Data []struct {
		ID           string                     `json:"id"`
		Name         string                     `json:"name"`
		Deprecated   bool                       `json:"deprecated"`
		Preview      bool                       `json:"preview"`
		Capabilities map[string]json.RawMessage `json:"capabilities"`
		Limits       struct {
			MaxContextLength    int `json:"max_context_length"`
			MaxCompletionTokens int `json:"max_completion_tokens"`
		} `json:"limits"`
	} `json:"data"`
}

type googleModelsResponse struct {
	Models []struct {
		Name                       string   `json:"name"`
		DisplayName                string   `json:"displayName"`
		SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
	} `json:"models"`
}
