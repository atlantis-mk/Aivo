package app

import (
	"errors"
	"strings"

	"aivo/core/domain"
)

type TransportType string

const (
	TransportOpenAIResponses   TransportType = "openai_responses"
	TransportOpenAIChat        TransportType = "openai_chat"
	TransportOpenAICompatible  TransportType = "openai_compatible"
	TransportAzureOpenAI       TransportType = "azure_openai"
	TransportAnthropicMessages TransportType = "anthropic_messages"
	TransportGoogleGemini      TransportType = "google_gemini"
	TransportGoogleVertex      TransportType = "google_vertex"
	TransportBedrockConverse   TransportType = "bedrock_converse"
	TransportGitHubCopilot     TransportType = "github_copilot"
	TransportExternalProcess   TransportType = "external_process"
)

type AuthType string

const (
	AuthAPIKey          AuthType = "api_key"
	AuthOAuthBrowser    AuthType = "oauth_browser"
	AuthOAuthDevice     AuthType = "oauth_device"
	AuthExternalProcess AuthType = "external_process"
	AuthAWSSDK          AuthType = "aws_sdk"
	AuthNone            AuthType = "none"
)

type ModelFetchStrategy string

const (
	ModelFetchStatic             ModelFetchStrategy = "static"
	ModelFetchOpenAICompatible   ModelFetchStrategy = "openai_compatible"
	ModelFetchOpenAICodexAccount ModelFetchStrategy = "openai_codex_account"
	ModelFetchAnthropic          ModelFetchStrategy = "anthropic"
	ModelFetchMistral            ModelFetchStrategy = "mistral"
	ModelFetchOpenRouter         ModelFetchStrategy = "openrouter"
	ModelFetchCerebras           ModelFetchStrategy = "cerebras"
	ModelFetchGoogle             ModelFetchStrategy = "google"
	ModelFetchDisabled           ModelFetchStrategy = "disabled"
)

type ProviderDefinition struct {
	ID              string
	DisplayName     string
	Description     string
	Aliases         []string
	Transport       TransportType
	AuthTypes       []AuthType
	DefaultAuthType AuthType
	DefaultBaseURL  string
	BaseURLEnvVar   string
	APIKeyEnvVars   []string
	ModelFetch      ModelFetchStrategy
	DefaultModelID  string
	Models          []domain.ModelInfo
	RequestProfile  domain.ProviderRequestProfile
	BuiltIn         bool
	Experimental    bool
	Deprecated      bool
	Command         string
	Args            []string
}

type ProviderRegistry struct {
	definitions map[string]ProviderDefinition
	aliases     map[string]string
	order       []string
}

type ResolvedModelRoute struct {
	Provider   domain.ProviderConfig
	Model      domain.ModelRef
	Definition ProviderDefinition
	Transport  TransportType
	BaseURL    string
	Credential llmCredential
}

var builtInProviderAliases = map[string]string{
	"claude":             "anthropic",
	"claude-code":        "claude-code",
	"gemini":             "gemini",
	"google-ai":          "google",
	"google-generative":  "google",
	"vertex":             "google-vertex",
	"vertex-ai":          "google-vertex",
	"google-vertex":      "google-vertex",
	"google-vertex-ai":   "google-vertex",
	"openai-api":         "openai",
	"azure":              "azure-openai",
	"azure-ai":           "azure-openai",
	"azure-openai":       "azure-openai",
	"azure-openai-v1":    "azure-openai",
	"openai-compatible":  "custom-api",
	"custom":             "custom-api",
	"custom-api":         "custom-api",
	"open-router":        "openrouter",
	"grok":               "xai",
	"x-ai":               "xai",
	"x.ai":               "xai",
	"mistral-ai":         "mistral",
	"la-plateforme":      "mistral",
	"groqcloud":          "groq",
	"deep-infra":         "deepinfra",
	"cerebras-ai":        "cerebras",
	"together-ai":        "together",
	"togetherai":         "together",
	"pplx":               "perplexity",
	"fireworks-ai":       "fireworks",
	"kimi":               "kimi-for-coding",
	"kimi-coding":        "kimi-for-coding",
	"moonshot":           "moonshotai",
	"zhipu":              "zai",
	"glm":                "zai",
	"z-ai":               "zai",
	"deep-seek":          "deepseek",
	"dashscope":          "alibaba",
	"qwen":               "alibaba",
	"alibaba-cloud":      "alibaba",
	"aws":                "amazon-bedrock",
	"bedrock":            "amazon-bedrock",
	"aws-bedrock":        "amazon-bedrock",
	"copilot":            "github-copilot",
	"github-copilot":     "github-copilot",
	"github-copilot-api": "github-copilot",
	"hf":                 "huggingface",
	"hugging-face":       "huggingface",
	"lm-studio":          "lmstudio",
	"ollama":             "custom-api",
}

func normalizeProviderID(providerID string) string {
	return defaultProviderRegistry.Normalize(providerID)
}

func NewProviderRegistry(definitions []ProviderDefinition, aliases map[string]string) *ProviderRegistry {
	registry := &ProviderRegistry{definitions: map[string]ProviderDefinition{}, aliases: map[string]string{}}
	for key, value := range builtInProviderAliases {
		registry.aliases[key] = value
	}
	for key, value := range aliases {
		registry.aliases[normalizeProviderKey(key)] = normalizeProviderKey(value)
	}
	for _, def := range definitions {
		_ = registry.RegisterDefinition(def)
	}
	return registry
}

func NewDefaultProviderRegistry() *ProviderRegistry {
	return NewProviderRegistry(builtInProviderDefinitions(), nil)
}

var defaultProviderRegistry = NewDefaultProviderRegistry()

func (r *ProviderRegistry) RegisterDefinition(def ProviderDefinition) error {
	if r == nil {
		return errors.New("provider registry is nil")
	}
	id := normalizeProviderKey(def.ID)
	if id == "" {
		return errors.New("provider id is required")
	}
	def.ID = id
	if strings.TrimSpace(def.DisplayName) == "" {
		def.DisplayName = id
	}
	if def.Transport == "" {
		def.Transport = TransportOpenAICompatible
	}
	if len(def.AuthTypes) == 0 {
		def.AuthTypes = []AuthType{AuthAPIKey}
	}
	if def.DefaultAuthType == "" {
		def.DefaultAuthType = def.AuthTypes[0]
	}
	if def.ModelFetch == "" {
		def.ModelFetch = ModelFetchOpenAICompatible
	}
	if def.DefaultModelID == "" && len(def.Models) > 0 {
		def.DefaultModelID = def.Models[0].ID
	}
	if def.DefaultModelID == "" {
		def.DefaultModelID = "default"
	}
	for i := range def.Models {
		if def.Models[i].ProviderID == "" {
			def.Models[i].ProviderID = id
		}
	}
	if _, exists := r.definitions[id]; !exists {
		r.order = append(r.order, id)
	}
	r.definitions[id] = cloneProviderDefinition(def)
	for _, alias := range def.Aliases {
		if normalized := normalizeProviderKey(alias); normalized != "" {
			r.aliases[normalized] = id
		}
	}
	return nil
}

func (r *ProviderRegistry) Normalize(providerID string) string {
	key := strings.TrimSpace(strings.ToLower(providerID))
	if key == "" {
		return ""
	}
	if r != nil {
		if alias, ok := r.aliases[key]; ok {
			return alias
		}
	}
	if alias, ok := builtInProviderAliases[key]; ok {
		return alias
	}
	return key
}

func (r *ProviderRegistry) Definition(providerID string) (ProviderDefinition, bool) {
	if r == nil {
		return ProviderDefinition{}, false
	}
	normalized := r.Normalize(providerID)
	def, ok := r.definitions[normalized]
	if !ok {
		return ProviderDefinition{}, false
	}
	return cloneProviderDefinition(def), true
}

func (r *ProviderRegistry) Definitions() []ProviderDefinition {
	if r == nil {
		return nil
	}
	out := make([]ProviderDefinition, 0, len(r.order))
	seen := map[string]bool{}
	for _, id := range r.order {
		if def, ok := r.definitions[id]; ok {
			out = append(out, cloneProviderDefinition(def))
			seen[id] = true
		}
	}
	for id, def := range r.definitions {
		if seen[id] {
			continue
		}
		out = append(out, cloneProviderDefinition(def))
	}
	return out
}

func normalizeProviderKey(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func cloneProviderDefinition(def ProviderDefinition) ProviderDefinition {
	def.Aliases = append([]string(nil), def.Aliases...)
	def.AuthTypes = append([]AuthType(nil), def.AuthTypes...)
	def.APIKeyEnvVars = append([]string(nil), def.APIKeyEnvVars...)
	def.Models = append([]domain.ModelInfo(nil), def.Models...)
	def.Args = append([]string(nil), def.Args...)
	def.RequestProfile = cloneRequestProfile(def.RequestProfile)
	return def
}

func providerDefinition(providerID string) (ProviderDefinition, bool) {
	return defaultProviderRegistry.Definition(providerID)
}

func (s *Service) normalizeProviderID(providerID string) string {
	if s != nil && s.providers != nil {
		s.providersMu.RLock()
		defer s.providersMu.RUnlock()
		return s.providers.Normalize(providerID)
	}
	return normalizeProviderID(providerID)
}

func (s *Service) providerDefinitions() []ProviderDefinition {
	if s != nil && s.providers != nil {
		s.providersMu.RLock()
		defer s.providersMu.RUnlock()
		return s.providers.Definitions()
	}
	return providerDefinitions()
}

func (s *Service) providerDefinition(providerID string) (ProviderDefinition, bool) {
	if s != nil && s.providers != nil {
		s.providersMu.RLock()
		defer s.providersMu.RUnlock()
		return s.providers.Definition(providerID)
	}
	return providerDefinition(providerID)
}

func (s *Service) providerDefinitionForConfig(provider domain.ProviderConfig) ProviderDefinition {
	if def, ok := s.providerDefinition(provider.ID); ok {
		return def
	}
	return providerDefinitionForConfig(provider)
}

func (s *Service) defaultProviders() []domain.ProviderInfo {
	defs := s.providerDefinitions()
	out := make([]domain.ProviderInfo, 0, len(defs))
	for _, def := range defs {
		out = append(out, providerInfoFromDefinition(def))
	}
	return out
}

func (s *Service) defaultModelForProvider(providerID string) string {
	if def, ok := s.providerDefinition(providerID); ok {
		return def.DefaultModelID
	}
	return "default"
}

func providerDefinitionForConfig(provider domain.ProviderConfig) ProviderDefinition {
	if def, ok := providerDefinition(provider.ID); ok {
		return def
	}
	transport := inferTransport(provider.ID, provider.Type, provider.BaseURL)
	fetch := ModelFetchOpenAICompatible
	switch transport {
	case TransportAnthropicMessages:
		fetch = ModelFetchAnthropic
	case TransportGoogleGemini:
		fetch = ModelFetchGoogle
	case TransportOpenAIResponses:
		fetch = ModelFetchOpenAICompatible
	}
	return ProviderDefinition{
		ID: strings.TrimSpace(provider.ID), DisplayName: strings.TrimSpace(provider.ID), Transport: transport,
		AuthTypes: []AuthType{AuthAPIKey, AuthNone}, DefaultAuthType: AuthAPIKey,
		DefaultBaseURL: strings.TrimSpace(provider.BaseURL), APIKeyEnvVars: splitEnvCandidates(provider.APIKeyEnv),
		ModelFetch: fetch, DefaultModelID: firstNonEmpty(provider.Model, "default"),
	}
}
