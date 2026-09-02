package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"aivo/core/domain"
)

func (s *Service) callChatGPTCodex(ctx context.Context, provider domain.ProviderConfig, model domain.ModelRef, modelInfo domain.ModelInfo, credential llmCredential, requestProfile domain.ProviderRequestProfile, messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string, serviceTier string, onDelta func(string), onToolDelta func(domain.ChatToolCall)) (domain.ChatResponse, error) {
	access, accountID, err := s.validOpenAIAccessToken(ctx, credential)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	body := responsesRequestBody(model.ModelID, messages, tools, reasoningEffort, serviceTier)
	applyRequestProfile(body, requestProfile, provider, model.ModelID)
	applyCodexRequestCapabilities(body, modelInfo, tools, reasoningEffort, serviceTier)
	raw, err := json.Marshal(body)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatGPTCodexResponsesURL, bytes.NewReader(raw))
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	applyRequestProfileHeaders(req, requestProfile, provider, model.ModelID)
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("User-Agent", openAIUserAgent)
	if codexModelUsesResponsesLite(modelInfo) {
		req.Header.Set("x-openai-internal-codex-responses-lite", "true")
	}
	if accountID != "" {
		req.Header.Set("ChatGPT-Account-Id", accountID)
	}
	return doLLMRequest(req, onDelta, onToolDelta)
}

func (s *Service) validOpenAIAccessToken(ctx context.Context, credential llmCredential) (string, string, error) {
	if credential.AccessToken != "" && !isExpired(credential.ExpiresAt, s.now()) {
		return credential.AccessToken, credential.AccountID, nil
	}
	if credential.Refresh == "" {
		return "", "", errors.New("OpenAI OAuth refresh token is missing")
	}
	tokens, err := refreshOpenAIToken(ctx, credential.Refresh)
	if err != nil {
		return "", "", err
	}
	accountID := extractOpenAIAccountID(tokens)
	if accountID == "" {
		accountID = credential.AccountID
	}
	expiresIn := tokens.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	if credential.AuthRecord != nil {
		next := *credential.AuthRecord
		next.AccessToken = tokens.AccessToken
		next.RefreshToken = firstNonEmpty(tokens.RefreshToken, credential.Refresh)
		next.ExpiresAt = domain.NowString(s.now().Add(time.Duration(expiresIn) * time.Second))
		next.AccountID = accountID
		next.UpdatedAt = domain.NowString(s.now())
		if err := s.saveProviderAuth(ctx, next); err != nil {
			return "", "", err
		}
	}
	return tokens.AccessToken, accountID, nil
}

func refreshOpenAIToken(ctx context.Context, refreshToken string) (openAITokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", openAIClientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIIssuer+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return openAITokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return doOpenAITokenRequest(req)
}

func isExpired(raw string, now time.Time) bool {
	if strings.TrimSpace(raw) == "" {
		return true
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return true
	}
	return !expiresAt.After(now.Add(30 * time.Second))
}

func callOpenAICompatible(ctx context.Context, provider domain.ProviderConfig, model domain.ModelRef, credential llmCredential, requestProfile domain.ProviderRequestProfile, messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string, serviceTier string, onDelta func(string), onToolDelta func(domain.ChatToolCall)) (domain.ChatResponse, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURLFor(provider.ID)
	}
	if baseURL == "" {
		return domain.ChatResponse{}, fmt.Errorf("base URL is not configured for provider %q", provider.ID)
	}
	var endpoint string
	var body map[string]any
	usesResponsesAPI := providerUsesResponsesAPI(provider, tools)
	if usesResponsesAPI {
		endpoint = providerResponsesBaseURL(provider, baseURL) + "/responses"
		body = responsesRequestBodyWithoutDefaults(model.ModelID, messages, tools, reasoningEffort, serviceTier)
	} else {
		endpoint = openAICompatibleChatCompletionsURL(baseURL)
		body = chatCompletionsRequestBody(model.ModelID, messages, tools)
	}
	applyProviderNativeWebSearchOptions(body, provider, tools)
	applyRequestProfile(body, requestProfile, provider, model.ModelID)
	if usesResponsesAPI {
		applyOpenAICompatibleResponsesRequestDefaults(body, provider)
	} else {
		applyOpenAIChatCompletionsRequestDefaults(body, reasoningEffort)
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	applyRequestProfileHeaders(req, requestProfile, provider, model.ModelID)
	if credential.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+credential.APIKey)
	}
	return doLLMRequest(req, onDelta, onToolDelta)
}

func providerUsesResponsesAPI(provider domain.ProviderConfig, tools []domain.ToolSpec) bool {
	providerID := providerDefinitionIDForConfig(provider)
	def, ok := providerDefinition(providerID)
	if ok && def.Transport == TransportOpenAIResponses {
		return true
	}
	if ok && def.DefaultResponsesAPI {
		return true
	}
	return ok && def.ResponsesAPIForHostedTools && hasResponsesHostedTool(tools)
}

func providerResponsesBaseURL(provider domain.ProviderConfig, baseURL string) string {
	providerID := providerDefinitionIDForConfig(provider)
	def, ok := providerDefinition(providerID)
	if !ok || strings.TrimSpace(def.ResponsesBaseURL) == "" {
		return baseURL
	}
	configuredBaseURL := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	defaultBaseURL := strings.TrimRight(strings.TrimSpace(def.DefaultBaseURL), "/")
	if configuredBaseURL != "" && configuredBaseURL != defaultBaseURL {
		return baseURL
	}
	return strings.TrimRight(strings.TrimSpace(def.ResponsesBaseURL), "/")
}

func providerDefinitionIDForConfig(provider domain.ProviderConfig) string {
	for _, candidate := range []string{provider.ID, provider.Type} {
		providerID := normalizeProviderID(candidate)
		if providerID == "" {
			continue
		}
		if _, ok := providerDefinition(providerID); ok {
			return providerID
		}
	}
	return normalizeProviderID(firstNonEmpty(provider.ID, provider.Type))
}

func openAICompatibleChatCompletionsURL(baseURL string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(strings.ToLower(trimmed), "/chat/completions") {
		return trimmed
	}
	return trimmed + "/chat/completions"
}

func hasResponsesHostedTool(tools []domain.ToolSpec) bool {
	for _, tool := range tools {
		if tool.Hosted == nil {
			continue
		}
		switch tool.Hosted.Type {
		case "web_search", "web_search_preview", "x_search", "code_interpreter", "file_search", "mcp":
			return true
		}
	}
	return false
}

func applyProviderNativeWebSearchOptions(body map[string]any, provider domain.ProviderConfig, tools []domain.ToolSpec) {
	providerID := providerDefinitionIDForConfig(provider)
	for _, tool := range tools {
		if tool.Hosted == nil {
			continue
		}
		switch providerID {
		case "perplexity":
			if tool.Hosted.Type != "perplexity_search" {
				continue
			}
			if len(tool.Hosted.AllowedDomains) > 0 {
				body["search_domain_filter"] = append([]string(nil), tool.Hosted.AllowedDomains...)
			}
			if size := strings.TrimSpace(tool.Hosted.SearchContextSize); size != "" {
				body["web_search_options"] = map[string]any{"search_context_size": size}
			}
			return
		case "perplexity-agent":
			if tool.Hosted.Type != "web_search" {
				continue
			}
			applyPerplexityAgentWebSearchFilters(body)
			return
		case "requesty":
			if tool.Hosted.Type != "web_search" {
				continue
			}
			serializedTools := requestToolsFromBody(body)
			serializedTools = append(serializedTools, map[string]any{"type": "web_search"})
			body["tools"] = serializedTools
			body["tool_choice"] = "auto"
			return
		case "venice":
			if tool.Hosted.Type != "venice_web_search" {
				continue
			}
			params := ensureProviderParamsMap(body, "venice_parameters")
			params["enable_web_search"] = "auto"
			return
		}
	}
}

func applyPerplexityAgentWebSearchFilters(body map[string]any) {
	if body == nil {
		return
	}
	for _, tool := range requestToolsFromBody(body) {
		if tool["type"] != "web_search" {
			continue
		}
		filters, ok := tool["filters"].(map[string]any)
		if !ok {
			continue
		}
		if domains, ok := filters["allowed_domains"]; ok {
			filters["search_domain_filter"] = domains
			delete(filters, "allowed_domains")
		}
	}
}

func requestToolsFromBody(body map[string]any) []map[string]any {
	if body == nil {
		return nil
	}
	switch typed := body["tools"].(type) {
	case []map[string]any:
		return append([]map[string]any(nil), typed...)
	case []any:
		tools := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			if tool, ok := item.(map[string]any); ok {
				tools = append(tools, tool)
			}
		}
		return tools
	default:
		return nil
	}
}

func ensureProviderParamsMap(body map[string]any, key string) map[string]any {
	if body == nil {
		return map[string]any{}
	}
	switch typed := body[key].(type) {
	case map[string]any:
		return typed
	case map[string]string:
		params := make(map[string]any, len(typed))
		for k, v := range typed {
			params[k] = v
		}
		body[key] = params
		return params
	default:
		params := map[string]any{}
		body[key] = params
		return params
	}
}

func callAzureOpenAI(ctx context.Context, provider domain.ProviderConfig, model domain.ModelRef, credential llmCredential, requestProfile domain.ProviderRequestProfile, messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string, serviceTier string, onDelta func(string), onToolDelta func(domain.ChatToolCall)) (domain.ChatResponse, error) {
	if credential.APIKey == "" {
		return domain.ChatResponse{}, fmt.Errorf("credentials are not configured for provider %q", provider.ID)
	}
	baseURL := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURLFor(provider.ID)
	}
	if baseURL == "" {
		return domain.ChatResponse{}, fmt.Errorf("base URL is not configured for provider %q", provider.ID)
	}
	body := responsesRequestBody(model.ModelID, messages, tools, reasoningEffort, serviceTier)
	applyRequestProfile(body, requestProfile, provider, model.ModelID)
	applyOpenAIResponsesRequestDefaults(body)
	raw, err := json.Marshal(body)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/responses", bytes.NewReader(raw))
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	applyRequestProfileHeaders(req, requestProfile, provider, model.ModelID)
	req.Header.Set("api-key", credential.APIKey)
	return doLLMRequest(req, onDelta, onToolDelta)
}

func callAnthropic(ctx context.Context, provider domain.ProviderConfig, model domain.ModelRef, credential llmCredential, requestProfile domain.ProviderRequestProfile, messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string, onDelta func(string), onToolDelta func(domain.ChatToolCall)) (domain.ChatResponse, error) {
	if credential.APIKey == "" {
		return domain.ChatResponse{}, fmt.Errorf("credentials are not configured for provider %q", provider.ID)
	}
	baseURL := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURLFor(provider.ID)
	}
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	body := anthropicRequestBody(model.ModelID, messages, tools, reasoningEffort)
	applyRequestProfile(body, requestProfile, provider, model.ModelID)
	raw, err := json.Marshal(body)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/messages", bytes.NewReader(raw))
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	applyRequestProfileHeaders(req, requestProfile, provider, model.ModelID)
	req.Header.Set("x-api-key", credential.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	return doLLMRequest(req, onDelta, onToolDelta)
}

func callGoogle(ctx context.Context, provider domain.ProviderConfig, model domain.ModelRef, credential llmCredential, requestProfile domain.ProviderRequestProfile, messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string, onDelta func(string), onToolDelta func(domain.ChatToolCall)) (domain.ChatResponse, error) {
	if credential.APIKey == "" {
		return domain.ChatResponse{}, fmt.Errorf("credentials are not configured for provider %q", provider.ID)
	}
	baseURL := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURLFor(provider.ID)
	}
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	endpoint := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse&key=%s", baseURL, url.PathEscape(model.ModelID), url.QueryEscape(credential.APIKey))
	body := googleRequestBody(model.ModelID, messages, tools, reasoningEffort)
	applyRequestProfile(body, requestProfile, provider, model.ModelID)
	raw, err := json.Marshal(body)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	applyRequestProfileHeaders(req, requestProfile, provider, model.ModelID)
	return doLLMRequest(req, onDelta, onToolDelta)
}

func callGoogleVertex(ctx context.Context, provider domain.ProviderConfig, model domain.ModelRef, credential llmCredential, requestProfile domain.ProviderRequestProfile, messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string, onDelta func(string), onToolDelta func(domain.ChatToolCall)) (domain.ChatResponse, error) {
	token := firstNonEmpty(credential.AccessToken, credential.APIKey)
	if token == "" {
		return domain.ChatResponse{}, fmt.Errorf("credentials are not configured for provider %q", provider.ID)
	}
	baseURL := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURLFor(provider.ID)
	}
	if baseURL == "" {
		return domain.ChatResponse{}, fmt.Errorf("base URL is not configured for provider %q", provider.ID)
	}
	endpoint := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse", baseURL, url.PathEscape(model.ModelID))
	body := googleRequestBody(model.ModelID, messages, tools, reasoningEffort)
	applyRequestProfile(body, requestProfile, provider, model.ModelID)
	raw, err := json.Marshal(body)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+token)
	applyRequestProfileHeaders(req, requestProfile, provider, model.ModelID)
	return doLLMRequest(req, onDelta, onToolDelta)
}

func callBedrockConverse(ctx context.Context, provider domain.ProviderConfig, model domain.ModelRef, requestProfile domain.ProviderRequestProfile, messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string, onDelta func(string), onToolDelta func(domain.ChatToolCall)) (domain.ChatResponse, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURLFor(provider.ID)
	}
	if baseURL == "" {
		return domain.ChatResponse{}, fmt.Errorf("base URL is not configured for provider %q", provider.ID)
	}
	body := bedrockConverseRequestBody(messages, tools, reasoningEffort)
	applyRequestProfile(body, requestProfile, provider, model.ModelID)
	raw, err := json.Marshal(body)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	endpoint := fmt.Sprintf("%s/model/%s/converse", baseURL, url.PathEscape(model.ModelID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	applyRequestProfileHeaders(req, requestProfile, provider, model.ModelID)
	if err := signBedrockRequest(req, raw); err != nil {
		return domain.ChatResponse{}, err
	}
	return doLLMRequest(req, onDelta, onToolDelta)
}

func callGitHubCopilot(ctx context.Context, provider domain.ProviderConfig, model domain.ModelRef, credential llmCredential, requestProfile domain.ProviderRequestProfile, messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string, onDelta func(string), onToolDelta func(domain.ChatToolCall)) (domain.ChatResponse, error) {
	token := firstNonEmpty(credential.AccessToken, credential.APIKey)
	if token == "" {
		return domain.ChatResponse{}, fmt.Errorf("credentials are not configured for provider %q", provider.ID)
	}
	baseURL := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURLFor(provider.ID)
	}
	if baseURL == "" {
		return domain.ChatResponse{}, fmt.Errorf("base URL is not configured for provider %q", provider.ID)
	}
	body := chatCompletionsRequestBody(model.ModelID, messages, tools)
	if effort := chatCompletionsReasoningEffort(reasoningEffort); effort != "" {
		body["reasoning_effort"] = effort
	}
	applyRequestProfile(body, requestProfile, provider, model.ModelID)
	raw, err := json.Marshal(body)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+token)
	applyRequestProfileHeaders(req, githubCopilotRequestProfile(), provider, model.ModelID)
	applyRequestProfileHeaders(req, requestProfile, provider, model.ModelID)
	return doLLMRequest(req, onDelta, onToolDelta)
}

func applyProviderHeaders(req *http.Request, headers map[string]string) {
	for key, value := range headers {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		req.Header.Set(key, value)
	}
}
