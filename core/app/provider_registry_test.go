package app

import (
	"context"
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

func TestDeclaredCapabilityProvidersUseDedicatedCatalogParsers(t *testing.T) {
	want := map[string]ModelFetchStrategy{
		"anthropic":  ModelFetchAnthropic,
		"mistral":    ModelFetchMistral,
		"openrouter": ModelFetchOpenRouter,
		"cerebras":   ModelFetchCerebras,
	}
	for providerID, strategy := range want {
		definition, ok := providerDefinition(providerID)
		if !ok {
			t.Fatalf("provider definition %q missing", providerID)
		}
		if definition.ModelFetch != strategy || !modelFetchDeclaresCapabilities(definition.ModelFetch) {
			t.Fatalf("%s model fetch = %q, want declared-capability strategy %q", providerID, definition.ModelFetch, strategy)
		}
		if parserTypeForModelFetch(strategy) == "openai-compatible" {
			t.Fatalf("%s still uses lossy generic parser", providerID)
		}
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
	auth            map[string]domain.ProviderAuthRecord
	authByID        map[string]domain.ProviderAuthRecord
	config          *domain.AppConfig
	providers       []domain.ProviderConfig
	modelCaches     map[string]domain.ProviderModelCache
	health          map[string]domain.ProviderHealth
	callEvents      []domain.ProviderCallEvent
	mcpDiagnostics  []domain.MCPDiagnostic
	mcpServers      []domain.MCPServerConfig
	mcpTools        map[string][]domain.MCPToolRecord
	savedAuth       *domain.ProviderAuthRecord
	savedCache      *domain.ProviderModelCache
	savedValidation *domain.ProviderValidationResult
	savedHealth     *domain.ProviderHealth
}
