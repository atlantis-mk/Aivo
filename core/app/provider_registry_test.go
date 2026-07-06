package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestProviderDefinitionExposesProductionMetadata(t *testing.T) {
	def, ok := providerDefinition("openai")
	if !ok {
		t.Fatal("openai provider definition missing")
	}
	if def.Transport != TransportOpenAIResponses {
		t.Fatalf("Transport = %q, want %q", def.Transport, TransportOpenAIResponses)
	}
	if def.ModelFetch != ModelFetchOpenAICompatible {
		t.Fatalf("ModelFetch = %q, want %q", def.ModelFetch, ModelFetchOpenAICompatible)
	}
	info := providerInfoFromDefinition(def)
	if info.Profile == nil || info.Profile.MessageShape != string(TransportOpenAIResponses) {
		t.Fatalf("Profile = %+v, want responses message shape", info.Profile)
	}
	if !info.Profile.InteractiveAuth {
		t.Fatalf("Profile.InteractiveAuth = false, want true")
	}
	if info.ModelRefresh == nil || !info.ModelRefresh.Refreshable || info.ModelRefresh.ParserType == "" {
		t.Fatalf("ModelRefresh = %+v, want refreshable parser metadata", info.ModelRefresh)
	}
	if len(info.Models) == 0 || !info.Models[0].Streaming || !info.Models[0].ToolSupport {
		t.Fatalf("Models = %+v, want capability metadata", info.Models)
	}
}

func TestProviderRegistryIncludesOpenAICompatibleProviderCoverage(t *testing.T) {
	tests := []struct {
		id           string
		alias        string
		baseURL      string
		env          string
		defaultModel string
		transport    TransportType
		refreshable  bool
		capability   string
	}{
		{id: "azure-openai", alias: "azure", baseURL: "https://YOUR-RESOURCE-NAME.openai.azure.com/openai/v1", env: "AZURE_OPENAI_API_KEY", defaultModel: "gpt-5.5", transport: TransportAzureOpenAI, refreshable: true, capability: "reasoning"},
		{id: "xai", alias: "grok", baseURL: "https://api.x.ai/v1", env: "XAI_API_KEY", defaultModel: "grok-4.3", transport: TransportOpenAICompatible, refreshable: true, capability: "vision"},
		{id: "mistral", alias: "mistral-ai", baseURL: "https://api.mistral.ai/v1", env: "MISTRAL_API_KEY", defaultModel: "mistral-medium-latest", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "groq", alias: "groqcloud", baseURL: "https://api.groq.com/openai/v1", env: "GROQ_API_KEY", defaultModel: "openai/gpt-oss-120b", transport: TransportOpenAICompatible, refreshable: true, capability: "reasoning"},
		{id: "deepinfra", alias: "deep-infra", baseURL: "https://api.deepinfra.com/v1/openai", env: "DEEPINFRA_API_KEY", defaultModel: "Qwen/Qwen3-Coder-480B-A35B-Instruct-Turbo", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "cerebras", alias: "cerebras-ai", baseURL: "https://api.cerebras.ai/v1", env: "CEREBRAS_API_KEY", defaultModel: "zai-glm-4.7", transport: TransportOpenAICompatible, refreshable: true, capability: "reasoning"},
		{id: "together", alias: "together-ai", baseURL: "https://api.together.ai/v1", env: "TOGETHER_API_KEY", defaultModel: "moonshotai/Kimi-K2.5", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "perplexity", alias: "pplx", baseURL: "https://api.perplexity.ai", env: "PERPLEXITY_API_KEY", defaultModel: "sonar-pro", transport: TransportOpenAICompatible, refreshable: false, capability: "search"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			if got := normalizeProviderID(tt.alias); got != tt.id {
				t.Fatalf("normalizeProviderID(%q) = %q, want %q", tt.alias, got, tt.id)
			}
			def, ok := providerDefinition(tt.id)
			if !ok {
				t.Fatalf("provider definition %q missing", tt.id)
			}
			if def.Transport != tt.transport {
				t.Fatalf("Transport = %q, want %q", def.Transport, tt.transport)
			}
			if !def.BuiltIn || def.DefaultBaseURL != tt.baseURL || def.DefaultModelID != tt.defaultModel {
				t.Fatalf("definition = %+v, want built-in base/default", def)
			}
			if len(def.APIKeyEnvVars) == 0 || def.APIKeyEnvVars[0] != tt.env {
				t.Fatalf("APIKeyEnvVars = %+v, want primary %q", def.APIKeyEnvVars, tt.env)
			}
			info := providerInfoFromDefinition(def)
			if info.ModelRefresh == nil || info.ModelRefresh.Refreshable != tt.refreshable {
				t.Fatalf("ModelRefresh = %+v, want refreshable=%v", info.ModelRefresh, tt.refreshable)
			}
			model, ok := findModelInfo(info.Models, tt.defaultModel)
			if !ok || !model.Streaming || !modelSupportsCapability(model, tt.capability) {
				t.Fatalf("model = %+v ok=%v, want streaming and capability %q", model, ok, tt.capability)
			}
		})
	}
}

func TestProviderRegistryIncludesAmazonBedrockConverse(t *testing.T) {
	if got := normalizeProviderID("bedrock"); got != "amazon-bedrock" {
		t.Fatalf("normalizeProviderID(bedrock) = %q, want amazon-bedrock", got)
	}
	def, ok := providerDefinition("amazon-bedrock")
	if !ok {
		t.Fatal("amazon-bedrock provider definition missing")
	}
	if def.Transport != TransportBedrockConverse {
		t.Fatalf("Transport = %q, want %q", def.Transport, TransportBedrockConverse)
	}
	if def.DefaultAuthType != AuthAWSSDK || len(def.AuthTypes) != 1 || def.AuthTypes[0] != AuthAWSSDK {
		t.Fatalf("AuthTypes = %+v DefaultAuthType = %q, want aws-sdk only", def.AuthTypes, def.DefaultAuthType)
	}
	if def.DefaultBaseURL != "https://bedrock-runtime.us-east-1.amazonaws.com" {
		t.Fatalf("DefaultBaseURL = %q", def.DefaultBaseURL)
	}
	info := providerInfoFromDefinition(def)
	if info.Profile == nil || info.Profile.MessageShape != string(TransportBedrockConverse) {
		t.Fatalf("Profile = %+v, want bedrock_converse message shape", info.Profile)
	}
	if info.ModelRefresh == nil || info.ModelRefresh.Refreshable {
		t.Fatalf("ModelRefresh = %+v, want static model list", info.ModelRefresh)
	}
	model, ok := findModelInfo(info.Models, def.DefaultModelID)
	if !ok || !modelSupportsCapability(model, "tools") || !modelSupportsCapability(model, "reasoning") {
		t.Fatalf("model = %+v ok=%v, want tools and reasoning", model, ok)
	}
}

func TestProviderRegistryIncludesGoogleVertex(t *testing.T) {
	if got := normalizeProviderID("vertex"); got != "google-vertex" {
		t.Fatalf("normalizeProviderID(vertex) = %q, want google-vertex", got)
	}
	def, ok := providerDefinition("google-vertex")
	if !ok {
		t.Fatal("google-vertex provider definition missing")
	}
	if def.Transport != TransportGoogleVertex {
		t.Fatalf("Transport = %q, want %q", def.Transport, TransportGoogleVertex)
	}
	if def.DefaultBaseURL != "https://us-central1-aiplatform.googleapis.com/v1/projects/YOUR_PROJECT_ID/locations/us-central1/publishers/google" {
		t.Fatalf("DefaultBaseURL = %q", def.DefaultBaseURL)
	}
	if len(def.APIKeyEnvVars) == 0 || def.APIKeyEnvVars[0] != "GOOGLE_VERTEX_ACCESS_TOKEN" {
		t.Fatalf("APIKeyEnvVars = %+v, want GOOGLE_VERTEX_ACCESS_TOKEN first", def.APIKeyEnvVars)
	}
	info := providerInfoFromDefinition(def)
	if info.Profile == nil || info.Profile.MessageShape != string(TransportGoogleVertex) {
		t.Fatalf("Profile = %+v, want google_vertex message shape", info.Profile)
	}
	if info.ModelRefresh == nil || info.ModelRefresh.Refreshable {
		t.Fatalf("ModelRefresh = %+v, want static model list", info.ModelRefresh)
	}
	model, ok := findModelInfo(info.Models, def.DefaultModelID)
	if !ok || !model.Streaming || !modelSupportsCapability(model, "tools") || !modelSupportsCapability(model, "reasoning") {
		t.Fatalf("model = %+v ok=%v, want streaming tools and reasoning", model, ok)
	}
}

func TestProviderRegistryIncludesGitHubCopilot(t *testing.T) {
	if got := normalizeProviderID("copilot"); got != "github-copilot" {
		t.Fatalf("normalizeProviderID(copilot) = %q, want github-copilot", got)
	}
	def, ok := providerDefinition("github-copilot")
	if !ok {
		t.Fatal("github-copilot provider definition missing")
	}
	if def.Transport != TransportGitHubCopilot {
		t.Fatalf("Transport = %q, want %q", def.Transport, TransportGitHubCopilot)
	}
	if def.DefaultBaseURL != "https://api.githubcopilot.com" {
		t.Fatalf("DefaultBaseURL = %q", def.DefaultBaseURL)
	}
	if len(def.APIKeyEnvVars) == 0 || def.APIKeyEnvVars[0] != "GITHUB_COPILOT_TOKEN" {
		t.Fatalf("APIKeyEnvVars = %+v, want GITHUB_COPILOT_TOKEN first", def.APIKeyEnvVars)
	}
	if def.RequestProfile.Headers["Copilot-Integration-Id"] == "" {
		t.Fatalf("RequestProfile.Headers = %+v, want Copilot integration header", def.RequestProfile.Headers)
	}
	info := providerInfoFromDefinition(def)
	if info.Profile == nil || info.Profile.MessageShape != string(TransportGitHubCopilot) {
		t.Fatalf("Profile = %+v, want github_copilot message shape", info.Profile)
	}
	if info.ModelRefresh == nil || info.ModelRefresh.Refreshable {
		t.Fatalf("ModelRefresh = %+v, want static model list", info.ModelRefresh)
	}
	model, ok := findModelInfo(info.Models, def.DefaultModelID)
	if !ok || !model.Streaming || !modelSupportsCapability(model, "tools") || !modelSupportsCapability(model, "reasoning") {
		t.Fatalf("model = %+v ok=%v, want streaming tools and reasoning", model, ok)
	}
}

func TestInferTransportFromCustomBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		want    TransportType
	}{
		{name: "openai responses", baseURL: "https://api.openai.com/v1", want: TransportOpenAIResponses},
		{name: "azure openai v1", baseURL: "https://team.openai.azure.com/openai/v1", want: TransportAzureOpenAI},
		{name: "anthropic suffix", baseURL: "https://gateway.example.com/anthropic/v1", want: TransportAnthropicMessages},
		{name: "kimi coding", baseURL: "https://api.kimi.com/coding/v1", want: TransportAnthropicMessages},
		{name: "google", baseURL: "https://generativelanguage.googleapis.com/v1beta", want: TransportGoogleGemini},
		{name: "google vertex", baseURL: "https://us-central1-aiplatform.googleapis.com/v1/projects/team/locations/us-central1/publishers/google", want: TransportGoogleVertex},
		{name: "github copilot", baseURL: "https://api.githubcopilot.com", want: TransportGitHubCopilot},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inferTransport("custom-api", "openai-compatible", tt.baseURL); got != tt.want {
				t.Fatalf("inferTransport() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveModelRouteUsesDefinitionEnvCandidates(t *testing.T) {
	oldLookup := lookupEnv
	defer func() { lookupEnv = oldLookup }()
	lookupEnv = func(name string) string {
		if name == "ANTHROPIC_TOKEN" {
			return "token-value"
		}
		return ""
	}
	service := NewService(&memoryProviderStore{})
	cfg := domain.AppConfig{Provider: &domain.ProviderConfig{ID: "anthropic", Model: "claude-sonnet-4"}}

	route, err := service.ResolveModelRoute(context.Background(), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if route.Transport != TransportAnthropicMessages {
		t.Fatalf("Transport = %q, want anthropic", route.Transport)
	}
	if route.Credential.Method != "env" || route.Credential.APIKey != "token-value" {
		t.Fatalf("Credential = %+v, want env token", route.Credential)
	}
	if route.Provider.APIKeyEnv != "ANTHROPIC_API_KEY" {
		t.Fatalf("APIKeyEnv = %q, want primary env reference", route.Provider.APIKeyEnv)
	}
}

func TestResolveModelRouteNormalizesProviderAlias(t *testing.T) {
	service := NewService(&memoryProviderStore{})
	cfg := domain.AppConfig{DefaultModel: &domain.ModelRef{ProviderID: "claude", ModelID: "claude-sonnet-4"}}

	route, err := service.ResolveModelRoute(context.Background(), cfg, nil)
	if err != nil && !strings.Contains(err.Error(), "credentials are not configured") {
		t.Fatal(err)
	}
	if route.Provider.ID != "" {
		t.Fatalf("route should not resolve without credentials, got %+v", route)
	}

	oldLookup := lookupEnv
	defer func() { lookupEnv = oldLookup }()
	lookupEnv = func(name string) string {
		if name == "ANTHROPIC_API_KEY" {
			return "anthropic-key"
		}
		return ""
	}
	route, err = service.ResolveModelRoute(context.Background(), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if route.Provider.ID != "anthropic" {
		t.Fatalf("Provider.ID = %q, want anthropic", route.Provider.ID)
	}
}

func TestCatalogIncludesPersistedCustomProviders(t *testing.T) {
	service := NewService(&memoryProviderStore{providers: []domain.ProviderConfig{{
		ID:        "team-proxy",
		Type:      string(TransportAnthropicMessages),
		BaseURL:   "https://proxy.example.com/anthropic/v1",
		APIKeyEnv: "TEAM_PROXY_KEY",
		Model:     "claude-sonnet-4-proxy",
	}}})

	catalog, err := service.Catalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var found *domain.ProviderInfo
	for i := range catalog.Providers {
		if catalog.Providers[i].ID == "team-proxy" {
			found = &catalog.Providers[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("persisted provider missing from catalog: %+v", catalog.Providers)
	}
	if found.Type != string(TransportAnthropicMessages) || found.BaseURL != "https://proxy.example.com/anthropic/v1" {
		t.Fatalf("provider = %+v, want persisted transport/baseURL", found)
	}
	if found.DefaultModelID != "claude-sonnet-4-proxy" || !modelListContains(found.Models, "claude-sonnet-4-proxy") {
		t.Fatalf("models = %+v default=%q, want persisted model", found.Models, found.DefaultModelID)
	}
	if found.Profile == nil || found.Profile.MessageShape != string(TransportAnthropicMessages) {
		t.Fatalf("Profile = %+v, want anthropic profile", found.Profile)
	}
}

func TestCatalogIncludesProviderHealth(t *testing.T) {
	service := NewService(&memoryProviderStore{health: map[string]domain.ProviderHealth{
		"openai": {
			ProviderID:       "openai",
			Status:           "degraded",
			LastErrorClass:   "rate_limit",
			LastErrorMessage: "too many requests",
			LastHTTPStatus:   429,
			FailureCount:     1,
			UpdatedAt:        "2026-01-01T00:00:00Z",
		},
	}})

	catalog, err := service.Catalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var found *domain.ProviderInfo
	for i := range catalog.Providers {
		if catalog.Providers[i].ID == "openai" {
			found = &catalog.Providers[i]
			break
		}
	}
	if found == nil || found.Health == nil {
		t.Fatalf("openai health missing from catalog: %+v", found)
	}
	if found.Health.Status != "degraded" || found.Health.LastErrorClass != "rate_limit" || found.Health.LastHTTPStatus != 429 {
		t.Fatalf("health = %+v, want degraded rate_limit", found.Health)
	}
}

func TestValidateProviderRefreshesModelsAndPersistsCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Fatalf("path = %q, want /models", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want bearer key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"team-model","name":"Team Model"}]}`))
	}))
	defer server.Close()
	store := &memoryProviderStore{}
	service := NewService(store)

	result, err := service.ValidateProvider(context.Background(), domain.ProviderConnectInput{
		ProviderID: "custom-api",
		Type:       string(TransportOpenAICompatible),
		BaseURL:    server.URL,
		APIKey:     "test-key",
		ModelID:    "team-model",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready || result.Status != "ready" || result.DefaultModel != "team-model" || result.ModelCount != 1 {
		t.Fatalf("validation result = %+v, want ready team-model", result)
	}
	if store.savedCache == nil || store.savedCache.ProviderID != "custom-api" || len(store.savedCache.Models) != 1 {
		t.Fatalf("saved cache = %+v, want model cache", store.savedCache)
	}
	if store.savedValidation == nil || !store.savedValidation.Ready {
		t.Fatalf("saved validation = %+v, want ready validation", store.savedValidation)
	}
}

func TestConnectProviderStoresAPIKeyAsSecretReference(t *testing.T) {
	store := &memoryProviderStore{}
	secrets := NewMemorySecretStore()
	service := NewService(store)
	service.SetSecretStore(secrets)

	_, err := service.ConnectProvider(context.Background(), domain.ProviderConnectInput{
		ProviderID: "custom-api",
		Type:       string(TransportOpenAICompatible),
		BaseURL:    "http://127.0.0.1:1234/v1",
		ModelID:    "local-model",
		Method:     "api-key",
		APIKey:     "super-secret-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	if store.savedAuth == nil {
		t.Fatal("auth was not saved")
	}
	if store.savedAuth.APIKey != "" {
		t.Fatalf("saved APIKey = %q, want empty plaintext", store.savedAuth.APIKey)
	}
	if store.savedAuth.APIKeyRef == "" {
		t.Fatalf("saved APIKeyRef is empty: %+v", store.savedAuth)
	}
	resolved, err := service.resolveProviderAuthSecrets(context.Background(), *store.savedAuth)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.APIKey != "super-secret-key" {
		t.Fatalf("resolved APIKey = %q, want secret", resolved.APIKey)
	}
}

func TestResolveCredentialSupportsLegacyPlaintextAuth(t *testing.T) {
	service := NewService(&memoryProviderStore{auth: map[string]domain.ProviderAuthRecord{
		"custom-api": {ProviderID: "custom-api", Method: "api-key", APIKey: "legacy-key"},
	}})
	service.SetSecretStore(NewMemorySecretStore())

	credential, err := service.resolveCredentialWithDefinition(context.Background(), domain.ProviderConfig{
		ID: "custom-api", Type: string(TransportOpenAICompatible), BaseURL: "http://127.0.0.1:1234/v1",
	}, providerDefinitionForConfig(domain.ProviderConfig{ID: "custom-api"}))
	if err != nil {
		t.Fatal(err)
	}
	if credential.APIKey != "legacy-key" {
		t.Fatalf("APIKey = %q, want legacy key", credential.APIKey)
	}
}

func TestDeleteProviderAccountDeletesSecretReferences(t *testing.T) {
	secrets := NewMemorySecretStore()
	auth := domain.ProviderAuthRecord{
		ID:         "auth-1",
		ProviderID: "custom-api",
		Method:     "api-key",
		APIKeyRef:  "provider-auth/custom-api/api-key/default/api-key",
	}
	if err := secrets.Put(context.Background(), auth.APIKeyRef, "secret"); err != nil {
		t.Fatal(err)
	}
	service := NewService(&memoryProviderStore{authByID: map[string]domain.ProviderAuthRecord{"auth-1": auth}})
	service.SetSecretStore(secrets)

	if _, err := service.DeleteProviderAccount(context.Background(), "auth-1"); err != nil {
		t.Fatal(err)
	}
	value, err := secrets.Get(context.Background(), auth.APIKeyRef)
	if err != nil {
		t.Fatal(err)
	}
	if value != "" {
		t.Fatalf("secret value = %q, want deleted", value)
	}
}

type memoryProviderStore struct {
	auth              map[string]domain.ProviderAuthRecord
	authByID          map[string]domain.ProviderAuthRecord
	config            *domain.AppConfig
	providers         []domain.ProviderConfig
	modelCaches       map[string]domain.ProviderModelCache
	health            map[string]domain.ProviderHealth
	callEvents        []domain.ProviderCallEvent
	plugins           []domain.PluginInstall
	pluginDiagnostics []domain.PluginDiagnostic
	mcpServers        []domain.MCPServerConfig
	mcpTools          map[string][]domain.MCPToolRecord
	savedAuth         *domain.ProviderAuthRecord
	savedCache        *domain.ProviderModelCache
	savedValidation   *domain.ProviderValidationResult
	savedHealth       *domain.ProviderHealth
}

func (m *memoryProviderStore) LoadProviderAuth(_ context.Context, providerID string) (*domain.ProviderAuthRecord, error) {
	if m.auth == nil {
		return nil, nil
	}
	auth, ok := m.auth[providerID]
	if !ok {
		return nil, nil
	}
	return &auth, nil
}

func (m *memoryProviderStore) GetProviderAuth(_ context.Context, id string) (*domain.ProviderAuthRecord, error) {
	if m.authByID == nil {
		return nil, nil
	}
	auth, ok := m.authByID[id]
	if !ok {
		return nil, nil
	}
	return &auth, nil
}

func (m *memoryProviderStore) LoadConfig(context.Context) (domain.AppConfig, error) {
	if m.config != nil {
		return *m.config, nil
	}
	return domain.AppConfig{}, nil
}
func (m *memoryProviderStore) SaveConfig(_ context.Context, cfg domain.AppConfig) error {
	m.config = &cfg
	return nil
}
func (m *memoryProviderStore) SaveProvider(_ context.Context, provider domain.ProviderConfig) error {
	for i := range m.providers {
		if m.providers[i].ID == provider.ID {
			m.providers[i] = provider
			return nil
		}
	}
	m.providers = append(m.providers, provider)
	return nil
}
func (m *memoryProviderStore) ListProviders(context.Context) ([]domain.ProviderConfig, error) {
	return append([]domain.ProviderConfig(nil), m.providers...), nil
}
func (m *memoryProviderStore) DeleteProvider(_ context.Context, providerID string) error {
	next := m.providers[:0]
	for _, provider := range m.providers {
		if provider.ID != providerID {
			next = append(next, provider)
		}
	}
	m.providers = next
	delete(m.auth, providerID)
	delete(m.modelCaches, providerID)
	delete(m.health, providerID)
	return nil
}
func (m *memoryProviderStore) SaveProviderModelCache(_ context.Context, cache domain.ProviderModelCache) error {
	m.savedCache = &cache
	if m.modelCaches == nil {
		m.modelCaches = map[string]domain.ProviderModelCache{}
	}
	m.modelCaches[cache.ProviderID] = cache
	return nil
}
func (m *memoryProviderStore) LoadProviderModelCache(_ context.Context, providerID string) (*domain.ProviderModelCache, error) {
	if m.modelCaches == nil {
		return nil, nil
	}
	cache, ok := m.modelCaches[providerID]
	if !ok {
		return nil, nil
	}
	return &cache, nil
}
func (m *memoryProviderStore) ListProviderModelCaches(context.Context) ([]domain.ProviderModelCache, error) {
	if m.modelCaches == nil {
		return nil, nil
	}
	out := make([]domain.ProviderModelCache, 0, len(m.modelCaches))
	for _, cache := range m.modelCaches {
		out = append(out, cache)
	}
	return out, nil
}
func (m *memoryProviderStore) SaveProviderValidation(_ context.Context, result domain.ProviderValidationResult) error {
	m.savedValidation = &result
	return nil
}
func (m *memoryProviderStore) LoadProviderValidation(context.Context, string) (*domain.ProviderValidationResult, error) {
	return nil, nil
}
func (m *memoryProviderStore) SaveProviderHealth(_ context.Context, health domain.ProviderHealth) error {
	m.savedHealth = &health
	if m.health == nil {
		m.health = map[string]domain.ProviderHealth{}
	}
	m.health[health.ProviderID] = health
	return nil
}
func (m *memoryProviderStore) LoadProviderHealth(_ context.Context, providerID string) (*domain.ProviderHealth, error) {
	if m.health == nil {
		return nil, nil
	}
	health, ok := m.health[providerID]
	if !ok {
		return nil, nil
	}
	return &health, nil
}
func (m *memoryProviderStore) ListProviderHealth(context.Context) ([]domain.ProviderHealth, error) {
	if m.health == nil {
		return nil, nil
	}
	out := make([]domain.ProviderHealth, 0, len(m.health))
	for _, health := range m.health {
		out = append(out, health)
	}
	return out, nil
}
func (m *memoryProviderStore) SaveProviderCallEvent(_ context.Context, event domain.ProviderCallEvent) error {
	m.callEvents = append(m.callEvents, event)
	return nil
}
func (m *memoryProviderStore) ListProviderCallEvents(_ context.Context, providerID string, limit int) ([]domain.ProviderCallEvent, error) {
	var out []domain.ProviderCallEvent
	for _, event := range m.callEvents {
		if providerID == "" || event.ProviderID == providerID {
			out = append(out, event)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
func (m *memoryProviderStore) SaveProviderAuth(_ context.Context, auth domain.ProviderAuthRecord) error {
	m.savedAuth = &auth
	if m.auth == nil {
		m.auth = map[string]domain.ProviderAuthRecord{}
	}
	m.auth[auth.ProviderID] = auth
	if m.authByID == nil {
		m.authByID = map[string]domain.ProviderAuthRecord{}
	}
	if auth.ID != "" {
		m.authByID[auth.ID] = auth
	}
	return nil
}
func (m *memoryProviderStore) ListProviderAuths(_ context.Context, providerID string) ([]domain.ProviderAuthRecord, error) {
	var out []domain.ProviderAuthRecord
	if m.auth != nil {
		if auth, ok := m.auth[providerID]; ok {
			out = append(out, auth)
		}
	}
	if m.authByID != nil {
		seen := map[string]bool{}
		for _, auth := range out {
			seen[auth.ID] = true
		}
		for _, auth := range m.authByID {
			if auth.ProviderID == providerID && !seen[auth.ID] {
				out = append(out, auth)
			}
		}
	}
	return out, nil
}
func (m *memoryProviderStore) DeleteProviderAuth(_ context.Context, id string) error {
	delete(m.authByID, id)
	return nil
}
func (m *memoryProviderStore) UpsertProject(context.Context, string) (domain.AssistantProject, error) {
	return domain.AssistantProject{}, nil
}
func (m *memoryProviderStore) SetProjectSidebarHidden(context.Context, string, bool) (domain.AssistantProject, error) {
	return domain.AssistantProject{}, nil
}
func (m *memoryProviderStore) ListProjects(context.Context, int) ([]domain.AssistantProject, error) {
	return nil, nil
}
func (m *memoryProviderStore) CreateRuntimeSession(context.Context, domain.CreateSessionRequest) (domain.Session, error) {
	return domain.Session{}, nil
}
func (m *memoryProviderStore) GetRuntimeSession(context.Context, string) (domain.Session, error) {
	return domain.Session{}, nil
}
func (m *memoryProviderStore) ListRuntimeSessions(context.Context, domain.ListSessionsRequest) ([]domain.Session, error) {
	return nil, nil
}
func (m *memoryProviderStore) UpdateRuntimeSession(context.Context, domain.UpdateSessionRequest) (domain.Session, error) {
	return domain.Session{}, nil
}
func (m *memoryProviderStore) SetRuntimeSessionStatus(context.Context, string, string) (domain.Session, error) {
	return domain.Session{}, nil
}
func (m *memoryProviderStore) SetRuntimeSessionAgentMode(context.Context, string, string) (domain.Session, error) {
	return domain.Session{}, nil
}
func (m *memoryProviderStore) AppendSessionEvent(context.Context, domain.SessionEvent) error {
	return nil
}
func (m *memoryProviderStore) GetSessionEvent(context.Context, string) (domain.SessionEvent, error) {
	return domain.SessionEvent{}, nil
}
func (m *memoryProviderStore) ListSessionEvents(context.Context, string, bool, int) ([]domain.SessionEvent, error) {
	return nil, nil
}
func (m *memoryProviderStore) UpdateSessionEvent(context.Context, domain.UpdateSessionEventRequest) (domain.SessionEvent, error) {
	return domain.SessionEvent{}, nil
}
func (m *memoryProviderStore) SetSessionEventVisibility(context.Context, string, string) (domain.SessionEvent, error) {
	return domain.SessionEvent{}, nil
}
func (m *memoryProviderStore) HideSessionTurnEvents(context.Context, string) error {
	return nil
}
func (m *memoryProviderStore) StartTurn(context.Context, domain.Turn) error { return nil }
func (m *memoryProviderStore) GetTurn(context.Context, string) (domain.Turn, error) {
	return domain.Turn{}, nil
}
func (m *memoryProviderStore) UpdateTurnStatus(context.Context, string, string, string) (domain.Turn, error) {
	return domain.Turn{}, nil
}
func (m *memoryProviderStore) ListTurns(context.Context, string, int) ([]domain.Turn, error) {
	return nil, nil
}
func (m *memoryProviderStore) SaveToolCall(context.Context, domain.ToolCall) error { return nil }
func (m *memoryProviderStore) ListToolCalls(context.Context, string) ([]domain.ToolCall, error) {
	return nil, nil
}
func (m *memoryProviderStore) UpsertSessionExecutionState(_ context.Context, state domain.SessionExecutionState) (domain.SessionExecutionState, error) {
	return state, nil
}
func (m *memoryProviderStore) GetSessionExecutionState(_ context.Context, sessionID string) (domain.SessionExecutionState, error) {
	return domain.SessionExecutionState{SessionID: sessionID, Status: domain.ExecutionStatusIdle}, nil
}
func (m *memoryProviderStore) CreatePendingSessionInput(_ context.Context, input domain.PendingSessionInput) (domain.PendingSessionInput, error) {
	return input, nil
}
func (m *memoryProviderStore) ListPendingSessionInputs(context.Context, string, string) ([]domain.PendingSessionInput, error) {
	return nil, nil
}
func (m *memoryProviderStore) UpdatePendingSessionInputStatus(_ context.Context, _ string, status string, promotedTurnID string) (domain.PendingSessionInput, error) {
	return domain.PendingSessionInput{Status: status, PromotedTurnID: promotedTurnID}, nil
}
func (m *memoryProviderStore) ListSessionEventsAfterCursor(context.Context, string, string, bool, int) ([]domain.SessionEvent, string, error) {
	return nil, "", nil
}
func (m *memoryProviderStore) MarkRunningToolCallsInterrupted(context.Context, string, string) (int, error) {
	return 0, nil
}
func (m *memoryProviderStore) CreatePermissionRequest(context.Context, domain.PermissionRequest) (domain.PermissionRequest, error) {
	return domain.PermissionRequest{}, nil
}
func (m *memoryProviderStore) GetPermissionRequest(context.Context, string) (domain.PermissionRequest, error) {
	return domain.PermissionRequest{}, nil
}
func (m *memoryProviderStore) ListPermissionRequests(context.Context, string, string) ([]domain.PermissionRequest, error) {
	return nil, nil
}
func (m *memoryProviderStore) UpdatePermissionRequest(context.Context, string, string, bool, string) (domain.PermissionRequest, error) {
	return domain.PermissionRequest{}, nil
}
func (m *memoryProviderStore) SavePermissionRule(context.Context, domain.PermissionRule) (domain.PermissionRule, error) {
	return domain.PermissionRule{}, nil
}
func (m *memoryProviderStore) ListPermissionRules(context.Context, string, string) ([]domain.PermissionRule, error) {
	return nil, nil
}
func (m *memoryProviderStore) CreateQuestionRequest(context.Context, domain.QuestionRequest) (domain.QuestionRequest, error) {
	return domain.QuestionRequest{}, nil
}
func (m *memoryProviderStore) GetQuestionRequest(context.Context, string) (domain.QuestionRequest, error) {
	return domain.QuestionRequest{}, nil
}
func (m *memoryProviderStore) ListQuestionRequests(context.Context, string, string) ([]domain.QuestionRequest, error) {
	return nil, nil
}
func (m *memoryProviderStore) UpdateQuestionRequest(context.Context, string, string, [][]string, string) (domain.QuestionRequest, error) {
	return domain.QuestionRequest{}, nil
}
func (m *memoryProviderStore) CreateSummary(context.Context, domain.SessionSummary) error { return nil }
func (m *memoryProviderStore) LatestSummary(context.Context, string) (*domain.SessionSummary, error) {
	return nil, nil
}
func (m *memoryProviderStore) CreateCheckpoint(context.Context, domain.SessionCheckpoint) error {
	return nil
}
func (m *memoryProviderStore) ListCheckpoints(context.Context, string, int) ([]domain.SessionCheckpoint, error) {
	return nil, nil
}
func (m *memoryProviderStore) LatestCheckpoint(context.Context, string) (*domain.SessionCheckpoint, error) {
	return nil, nil
}
func (m *memoryProviderStore) UpsertCodingContext(context.Context, domain.CodingContext) (domain.CodingContext, error) {
	return domain.CodingContext{}, nil
}
func (m *memoryProviderStore) GetCodingContext(context.Context, string) (domain.CodingContext, error) {
	return domain.CodingContext{}, nil
}
func (m *memoryProviderStore) LatestSessionByProject(context.Context, string) (*domain.Session, error) {
	return nil, nil
}
func (m *memoryProviderStore) ForkRuntimeSession(context.Context, domain.Session, domain.ForkSessionRequest) (domain.Session, error) {
	return domain.Session{}, nil
}
func (m *memoryProviderStore) SaveAgentRun(context.Context, domain.AgentRun) (domain.AgentRun, error) {
	return domain.AgentRun{}, nil
}
func (m *memoryProviderStore) ListAgentRuns(context.Context, domain.AgentRunListRequest) ([]domain.AgentRun, error) {
	return nil, nil
}
func (m *memoryProviderStore) GetAgentRun(context.Context, string) (domain.AgentRun, error) {
	return domain.AgentRun{}, nil
}
func (m *memoryProviderStore) ReplaceTodoItems(context.Context, domain.TodoListInput, []domain.TodoItem) ([]domain.TodoItem, error) {
	return nil, nil
}
func (m *memoryProviderStore) ListTodoItems(context.Context, domain.TodoListInput) ([]domain.TodoItem, error) {
	return nil, nil
}
func (m *memoryProviderStore) SaveScheduledJob(context.Context, domain.ScheduledJob) (domain.ScheduledJob, error) {
	return domain.ScheduledJob{}, nil
}
func (m *memoryProviderStore) GetScheduledJob(context.Context, string) (domain.ScheduledJob, error) {
	return domain.ScheduledJob{}, nil
}
func (m *memoryProviderStore) ListScheduledJobs(context.Context, domain.ScheduledJobListInput) ([]domain.ScheduledJob, error) {
	return nil, nil
}
func (m *memoryProviderStore) ListDueScheduledJobs(context.Context, string, int) ([]domain.ScheduledJob, error) {
	return nil, nil
}
func (m *memoryProviderStore) DeleteScheduledJob(context.Context, string) error {
	return nil
}

func (m *memoryProviderStore) SavePluginInstall(_ context.Context, plugin domain.PluginInstall) (domain.PluginInstall, error) {
	m.plugins = append(m.plugins, plugin)
	return plugin, nil
}

func (m *memoryProviderStore) GetPluginInstall(_ context.Context, id string) (domain.PluginInstall, error) {
	for _, plugin := range m.plugins {
		if plugin.ID == id {
			return plugin, nil
		}
	}
	return domain.PluginInstall{}, errors.New("plugin not found")
}

func (m *memoryProviderStore) ListPluginInstalls(_ context.Context, includeDisabled bool) ([]domain.PluginInstall, error) {
	out := make([]domain.PluginInstall, 0, len(m.plugins))
	for _, plugin := range m.plugins {
		if includeDisabled || plugin.Enabled {
			out = append(out, plugin)
		}
	}
	return out, nil
}

func (m *memoryProviderStore) SetPluginEnabled(_ context.Context, id string, enabled bool, status string, statusMessage string) (domain.PluginInstall, error) {
	for i, plugin := range m.plugins {
		if plugin.ID == id {
			plugin.Enabled = enabled
			plugin.Status = status
			plugin.Error = statusMessage
			m.plugins[i] = plugin
			return plugin, nil
		}
	}
	return domain.PluginInstall{}, errors.New("plugin not found")
}

func (m *memoryProviderStore) SavePluginDiagnostic(_ context.Context, diagnostic domain.PluginDiagnostic) (domain.PluginDiagnostic, error) {
	m.pluginDiagnostics = append(m.pluginDiagnostics, diagnostic)
	return diagnostic, nil
}

func (m *memoryProviderStore) ListPluginDiagnostics(_ context.Context, pluginID string, serverID string, limit int) ([]domain.PluginDiagnostic, error) {
	out := []domain.PluginDiagnostic{}
	for _, diagnostic := range m.pluginDiagnostics {
		if pluginID != "" && diagnostic.PluginID != pluginID {
			continue
		}
		if serverID != "" && diagnostic.ServerID != serverID {
			continue
		}
		out = append(out, diagnostic)
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *memoryProviderStore) SaveMCPServer(_ context.Context, server domain.MCPServerConfig) (domain.MCPServerConfig, error) {
	m.mcpServers = append(m.mcpServers, server)
	return server, nil
}

func (m *memoryProviderStore) GetMCPServer(_ context.Context, id string) (domain.MCPServerConfig, error) {
	for _, server := range m.mcpServers {
		if server.ID == id {
			return server, nil
		}
	}
	return domain.MCPServerConfig{}, errors.New("mcp server not found")
}

func (m *memoryProviderStore) ListMCPServers(_ context.Context, includeDisabled bool) ([]domain.MCPServerConfig, error) {
	out := make([]domain.MCPServerConfig, 0, len(m.mcpServers))
	for _, server := range m.mcpServers {
		if includeDisabled || server.Enabled {
			out = append(out, server)
		}
	}
	return out, nil
}

func (m *memoryProviderStore) SetMCPServerEnabled(_ context.Context, id string, enabled bool, status string, statusMessage string) (domain.MCPServerConfig, error) {
	for i, server := range m.mcpServers {
		if server.ID == id {
			server.Enabled = enabled
			server.Status = status
			server.Error = statusMessage
			m.mcpServers[i] = server
			return server, nil
		}
	}
	return domain.MCPServerConfig{}, errors.New("mcp server not found")
}

func (m *memoryProviderStore) DeleteMCPServer(_ context.Context, id string) error {
	next := m.mcpServers[:0]
	for _, server := range m.mcpServers {
		if server.ID != id {
			next = append(next, server)
		}
	}
	m.mcpServers = next
	delete(m.mcpTools, id)
	return nil
}

func (m *memoryProviderStore) ReplaceMCPTools(_ context.Context, serverID string, tools []domain.MCPToolRecord) error {
	if m.mcpTools == nil {
		m.mcpTools = map[string][]domain.MCPToolRecord{}
	}
	m.mcpTools[serverID] = append([]domain.MCPToolRecord(nil), tools...)
	return nil
}

func (m *memoryProviderStore) ListMCPTools(_ context.Context, serverID string) ([]domain.MCPToolRecord, error) {
	if strings.TrimSpace(serverID) != "" {
		return append([]domain.MCPToolRecord(nil), m.mcpTools[serverID]...), nil
	}
	out := []domain.MCPToolRecord{}
	for _, tools := range m.mcpTools {
		out = append(out, tools...)
	}
	return out, nil
}

func (m *memoryProviderStore) ReplaceMCPPrompts(context.Context, string, []domain.MCPPromptRecord) error {
	return nil
}

func (m *memoryProviderStore) ListMCPPrompts(context.Context, string) ([]domain.MCPPromptRecord, error) {
	return nil, nil
}

func (m *memoryProviderStore) ReplaceMCPResources(context.Context, string, []domain.MCPResourceRecord) error {
	return nil
}

func (m *memoryProviderStore) ListMCPResources(context.Context, string, bool) ([]domain.MCPResourceRecord, error) {
	return nil, nil
}
