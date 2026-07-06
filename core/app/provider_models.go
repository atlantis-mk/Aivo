package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
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

func parseCodexModels(raw []byte, providerID string) ([]domain.ModelInfo, string, error) {
	var body codexModelsResponse
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, "", errors.New("provider model response could not be parsed")
	}
	items := make([]codexModelInfo, 0, len(body.Models))
	for _, model := range body.Models {
		if strings.TrimSpace(model.Visibility) != "" && model.Visibility != "list" {
			continue
		}
		id := firstNonEmpty(model.Slug, model.ID)
		if strings.TrimSpace(id) == "" {
			continue
		}
		items = append(items, model)
	}
	sort.SliceStable(items, func(i, j int) bool {
		leftID := firstNonEmpty(items[i].Slug, items[i].ID)
		rightID := firstNonEmpty(items[j].Slug, items[j].ID)
		if compareModelVersion(leftID, rightID) != 0 {
			return compareModelVersion(leftID, rightID) > 0
		}
		return items[i].Priority < items[j].Priority
	})
	models := make([]domain.ModelInfo, 0, len(items))
	for _, item := range items {
		id := firstNonEmpty(item.Slug, item.ID)
		models = append(models, domain.ModelInfo{
			ID:         id,
			ProviderID: providerID,
			Name:       firstNonEmpty(item.DisplayName, item.Name, id),
		})
	}
	return finalizeParsedModels(models, "")
}

type openAIModelsResponse struct {
	Data []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
	} `json:"data"`
}

func parseOpenAICompatibleModels(raw []byte, providerID string) ([]domain.ModelInfo, string, error) {
	var body openAIModelsResponse
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, "", errors.New("provider model response could not be parsed")
	}
	models := make([]domain.ModelInfo, 0, len(body.Data))
	for _, item := range body.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		models = append(models, domain.ModelInfo{
			ID:         id,
			ProviderID: providerID,
			Name:       firstNonEmpty(item.DisplayName, item.Name, id),
		})
	}
	sortModelsForProvider(providerID, models)
	return finalizeParsedModels(models, "")
}

type googleModelsResponse struct {
	Models []struct {
		Name                       string   `json:"name"`
		DisplayName                string   `json:"displayName"`
		SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
	} `json:"models"`
}

func parseGoogleModels(raw []byte, providerID string) ([]domain.ModelInfo, string, error) {
	var body googleModelsResponse
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, "", errors.New("provider model response could not be parsed")
	}
	models := make([]domain.ModelInfo, 0, len(body.Models))
	for _, item := range body.Models {
		id := strings.TrimPrefix(strings.TrimSpace(item.Name), "models/")
		if id == "" || !supportsGoogleGenerateContent(item.SupportedGenerationMethods) {
			continue
		}
		models = append(models, domain.ModelInfo{
			ID:         id,
			ProviderID: providerID,
			Name:       firstNonEmpty(item.DisplayName, id),
		})
	}
	return finalizeParsedModels(models, "")
}

func finalizeParsedModels(models []domain.ModelInfo, preferredDefault string) ([]domain.ModelInfo, string, error) {
	if len(models) == 0 {
		return nil, "", errors.New("provider model response did not include any models")
	}
	seen := map[string]bool{}
	out := make([]domain.ModelInfo, 0, len(models))
	for _, model := range models {
		id := strings.TrimSpace(model.ID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		model.ID = id
		model.Name = firstNonEmpty(model.Name, id)
		out = append(out, model)
	}
	if len(out) == 0 {
		return nil, "", errors.New("provider model response did not include any models")
	}
	defaultModel := preferredDefault
	if defaultModel == "" {
		defaultModel = out[0].ID
	}
	markRecommended(out, defaultModel)
	return out, defaultModel, nil
}

func sortModelsForProvider(providerID string, models []domain.ModelInfo) {
	if providerID != "openai" {
		return
	}
	sort.SliceStable(models, func(i, j int) bool {
		if compareModelVersion(models[i].ID, models[j].ID) != 0 {
			return compareModelVersion(models[i].ID, models[j].ID) > 0
		}
		return models[i].ID < models[j].ID
	})
}

func compareModelVersion(left string, right string) int {
	leftParts := modelVersionParts(left)
	rightParts := modelVersionParts(right)
	for i := 0; i < len(leftParts) || i < len(rightParts); i++ {
		leftValue := 0
		rightValue := 0
		if i < len(leftParts) {
			leftValue = leftParts[i]
		}
		if i < len(rightParts) {
			rightValue = rightParts[i]
		}
		if leftValue > rightValue {
			return 1
		}
		if leftValue < rightValue {
			return -1
		}
	}
	return 0
}

func modelVersionParts(modelID string) []int {
	lower := strings.ToLower(modelID)
	index := strings.Index(lower, "gpt-")
	if index < 0 {
		return nil
	}
	rest := lower[index+len("gpt-"):]
	parts := []int{}
	current := 0
	hasDigit := false
	for _, ch := range rest {
		if ch >= '0' && ch <= '9' {
			current = current*10 + int(ch-'0')
			hasDigit = true
			continue
		}
		if hasDigit {
			parts = append(parts, current)
			current = 0
			hasDigit = false
		}
		if ch != '.' && ch != '-' {
			break
		}
	}
	if hasDigit {
		parts = append(parts, current)
	}
	return parts
}

func (s *Service) rememberRefreshedModels(ctx context.Context, provider domain.ProviderConfig, name string, models []domain.ModelInfo, defaultModel string) {
	s.modelRefreshMu.Lock()
	defer s.modelRefreshMu.Unlock()
	providerID := provider.ID
	copied := append([]domain.ModelInfo(nil), models...)
	s.refreshedModels[providerID] = copied
	s.refreshedDefault[providerID] = defaultModel
	s.refreshedInfo[providerID] = domain.ProviderInfo{
		ID:             providerID,
		Name:           firstNonEmpty(strings.TrimSpace(name), providerID),
		Type:           provider.Type,
		BaseURL:        provider.BaseURL,
		BuiltIn:        false,
		Custom:         true,
		Environment:    provider.APIKeyEnv,
		DefaultModelID: defaultModel,
		Models:         copied,
		AuthMethods:    providerAuthMethods(providerID, provider.APIKeyEnv),
	}
	def := s.providerDefinitionForConfig(provider)
	_ = s.store.SaveProviderModelCache(ctx, domain.ProviderModelCache{
		ProviderID: providerID, Models: copied, DefaultModel: defaultModel, Strategy: string(def.ModelFetch),
		ParserType: parserTypeForModelFetch(def.ModelFetch), Endpoint: modelEndpointForDefinition(def), CacheSource: "remote",
		Status: "ready", RefreshedAt: domain.NowString(s.now()), UpdatedAt: domain.NowString(s.now()),
	})
}

func (s *Service) applyRefreshedProviderModels(providers []domain.ProviderInfo) []domain.ProviderInfo {
	s.modelRefreshMu.Lock()
	defer s.modelRefreshMu.Unlock()
	seen := map[string]bool{}
	for i := range providers {
		seen[providers[i].ID] = true
		models := s.refreshedModels[providers[i].ID]
		if len(models) == 0 {
			continue
		}
		providers[i].Models = append([]domain.ModelInfo(nil), models...)
		if defaultModel := s.refreshedDefault[providers[i].ID]; defaultModel != "" {
			providers[i].DefaultModelID = defaultModel
		}
	}
	for providerID, provider := range s.refreshedInfo {
		if seen[providerID] {
			continue
		}
		providers = append(providers, provider)
	}
	return providers
}

func (s *Service) defaultModelFor(providerID string) string {
	s.modelRefreshMu.Lock()
	defer s.modelRefreshMu.Unlock()
	if model := s.refreshedDefault[providerID]; model != "" {
		return model
	}
	return defaultModelFor(providerID)
}

func preferredDefaultModel(providerID string, models []domain.ModelInfo) string {
	preferred := defaultModelFor(providerID)
	if preferred == "" {
		return ""
	}
	for _, model := range models {
		if model.ID == preferred {
			return preferred
		}
	}
	return ""
}

func markRecommended(models []domain.ModelInfo, defaultModel string) {
	for i := range models {
		models[i].Recommended = models[i].ID == defaultModel
	}
}

func supportsGoogleGenerateContent(methods []string) bool {
	if len(methods) == 0 {
		return true
	}
	for _, method := range methods {
		if method == "generateContent" {
			return true
		}
	}
	return false
}

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
