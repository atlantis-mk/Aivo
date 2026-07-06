package app

import (
	"context"
	"errors"
	"net/url"
	"os"
	"sort"
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
	"pplx":               "perplexity",
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
	def.RequestProfile = cloneRequestProfile(def.RequestProfile)
	return def
}

func providerDefinitions() []ProviderDefinition {
	return defaultProviderRegistry.Definitions()
}

func builtInProviderDefinitions() []ProviderDefinition {
	defs := []ProviderDefinition{
		{
			ID: "openai", DisplayName: "OpenAI", Description: "OpenAI API or ChatGPT account models.",
			Transport: TransportOpenAIResponses, AuthTypes: []AuthType{AuthOAuthBrowser, AuthOAuthDevice, AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.openai.com/v1", BaseURLEnvVar: "OPENAI_BASE_URL", APIKeyEnvVars: []string{"OPENAI_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "gpt-5.5", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{
				Params: map[string]any{"store": false, "stream": true},
				ModelOverrides: map[string]domain.ProviderRequestOverride{
					"gpt-5-chat-latest":   {Params: map[string]any{"reasoning": map[string]any{"effort": "medium", "summary": "auto"}, "include": []any{"reasoning.encrypted_content"}}},
					"gpt-5.1-chat-latest": {Params: map[string]any{"reasoning": map[string]any{"effort": "medium", "summary": "auto"}, "include": []any{"reasoning.encrypted_content"}}},
					"gpt-5.2-chat-latest": {Params: map[string]any{"reasoning": map[string]any{"effort": "medium", "summary": "auto"}, "include": []any{"reasoning.encrypted_content"}}},
					"gpt-5-search-api":    {Params: map[string]any{"reasoning": map[string]any{"effort": "none", "summary": "auto"}, "include": []any{"reasoning.encrypted_content"}}},
				},
			},
			Models: []domain.ModelInfo{
				model("openai", "gpt-5.5", "GPT-5.5", true, 400000, []string{"tools", "reasoning", "streaming", "web_search", "code_interpreter", "file_search", "remote_mcp"}),
				model("openai", "gpt-5.4", "GPT-5.4", false, 400000, []string{"tools", "reasoning", "streaming", "web_search", "code_interpreter", "file_search", "remote_mcp"}),
				model("openai", "gpt-5.4-mini", "GPT-5.4-Mini", false, 400000, []string{"tools", "reasoning", "streaming", "web_search", "code_interpreter", "file_search", "remote_mcp"}),
				model("openai", "gpt-5.3-codex-spark", "GPT-5.3-Codex-Spark", false, 400000, []string{"tools", "reasoning", "streaming", "web_search", "code_interpreter", "file_search", "remote_mcp"}),
			},
		},
		{
			ID: "azure-openai", DisplayName: "Azure OpenAI", Description: "Azure OpenAI / Microsoft Foundry v1 OpenAI-compatible endpoint.",
			Aliases: []string{"azure", "azure-ai", "azure-openai-v1"}, Transport: TransportAzureOpenAI,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://YOUR-RESOURCE-NAME.openai.azure.com/openai/v1", BaseURLEnvVar: "AZURE_OPENAI_BASE_URL", APIKeyEnvVars: []string{"AZURE_OPENAI_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "gpt-5.5", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"store": false, "stream": true}},
			Models: []domain.ModelInfo{
				model("azure-openai", "gpt-5.5", "GPT-5.5 deployment", true, 400000, []string{"tools", "reasoning", "streaming", "web_search", "code_interpreter", "file_search"}),
				model("azure-openai", "gpt-5.4", "GPT-5.4 deployment", false, 400000, []string{"tools", "reasoning", "streaming", "web_search", "code_interpreter", "file_search"}),
				model("azure-openai", "gpt-5.4-mini", "GPT-5.4 Mini deployment", false, 400000, []string{"tools", "reasoning", "streaming", "web_search", "code_interpreter", "file_search"}),
				model("azure-openai", "gpt-5.3-codex", "GPT-5.3 Codex deployment", false, 400000, []string{"tools", "reasoning", "streaming", "web_search", "code_interpreter", "file_search"}),
			},
		},
		{
			ID: "claude-code", DisplayName: "Claude Code", Description: "Anthropic Messages API compatible Claude coding models.",
			Transport: TransportAnthropicMessages, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.anthropic.com/v1", BaseURLEnvVar: "ANTHROPIC_BASE_URL", APIKeyEnvVars: []string{"ANTHROPIC_API_KEY", "ANTHROPIC_TOKEN", "CLAUDE_CODE_OAUTH_TOKEN"},
			ModelFetch: ModelFetchAnthropic, DefaultModelID: "claude-sonnet-4", BuiltIn: true,
			RequestProfile: anthropicDefaultRequestProfile(),
			Models:         []domain.ModelInfo{model("claude-code", "claude-sonnet-4", "Claude Sonnet 4", true, 200000, []string{"tools", "reasoning", "streaming", "web_search", "web_fetch"})},
		},
		{
			ID: "anthropic", DisplayName: "Anthropic", Description: "Anthropic Messages API.",
			Transport: TransportAnthropicMessages, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.anthropic.com/v1", BaseURLEnvVar: "ANTHROPIC_BASE_URL", APIKeyEnvVars: []string{"ANTHROPIC_API_KEY", "ANTHROPIC_TOKEN"},
			ModelFetch: ModelFetchAnthropic, DefaultModelID: "claude-sonnet-4", BuiltIn: true,
			RequestProfile: anthropicDefaultRequestProfile(),
			Models:         []domain.ModelInfo{model("anthropic", "claude-sonnet-4", "Claude Sonnet 4", true, 200000, []string{"tools", "reasoning", "streaming", "web_search", "web_fetch"})},
		},
		{
			ID: "gemini", DisplayName: "Gemini", Description: "Google Gemini API.",
			Transport: TransportGoogleGemini, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://generativelanguage.googleapis.com/v1beta", BaseURLEnvVar: "GEMINI_BASE_URL", APIKeyEnvVars: []string{"GEMINI_API_KEY", "GOOGLE_API_KEY"},
			ModelFetch: ModelFetchGoogle, DefaultModelID: "gemini-2.5-pro", BuiltIn: true,
			RequestProfile: googleDefaultRequestProfile(),
			Models: []domain.ModelInfo{
				model("gemini", "gemini-2.5-pro", "Gemini 2.5 Pro", true, 1000000, []string{"tools", "reasoning", "streaming", "web_search", "web_fetch", "code_execution", "file_search"}),
				model("gemini", "gemini-2.5-flash", "Gemini 2.5 Flash", false, 1000000, []string{"tools", "reasoning", "streaming", "web_search", "web_fetch", "code_execution", "file_search"}),
			},
		},
		{
			ID: "google", DisplayName: "Google", Description: "Google Generative Language API.",
			Transport: TransportGoogleGemini, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://generativelanguage.googleapis.com/v1beta", BaseURLEnvVar: "GOOGLE_BASE_URL", APIKeyEnvVars: []string{"GOOGLE_API_KEY", "GEMINI_API_KEY"},
			ModelFetch: ModelFetchGoogle, DefaultModelID: "gemini-2.5-pro", BuiltIn: true,
			RequestProfile: googleDefaultRequestProfile(),
			Models:         []domain.ModelInfo{model("google", "gemini-2.5-pro", "Gemini 2.5 Pro", true, 1000000, []string{"tools", "reasoning", "streaming", "web_search", "web_fetch", "code_execution", "file_search"})},
		},
		{
			ID: "google-vertex", DisplayName: "Google Vertex", Description: "Google Vertex AI Gemini publisher models.",
			Aliases: []string{"vertex", "vertex-ai", "google-vertex-ai"}, Transport: TransportGoogleVertex,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://us-central1-aiplatform.googleapis.com/v1/projects/YOUR_PROJECT_ID/locations/us-central1/publishers/google",
			BaseURLEnvVar:  "GOOGLE_VERTEX_BASE_URL", APIKeyEnvVars: []string{"GOOGLE_VERTEX_ACCESS_TOKEN", "VERTEX_ACCESS_TOKEN", "GOOGLE_OAUTH_ACCESS_TOKEN"},
			ModelFetch: ModelFetchStatic, DefaultModelID: "gemini-2.5-pro", BuiltIn: true,
			RequestProfile: googleDefaultRequestProfile(),
			Models: []domain.ModelInfo{
				model("google-vertex", "gemini-2.5-pro", "Gemini 2.5 Pro", true, 1000000, []string{"tools", "reasoning", "streaming", "web_search", "web_fetch", "code_execution", "file_search"}),
				model("google-vertex", "gemini-2.5-flash", "Gemini 2.5 Flash", false, 1000000, []string{"tools", "reasoning", "streaming", "web_search", "web_fetch", "code_execution", "file_search"}),
				model("google-vertex", "gemini-3-pro-preview", "Gemini 3 Pro Preview", false, 1000000, []string{"tools", "reasoning", "streaming", "web_search", "web_fetch", "code_execution", "file_search"}),
			},
		},
		{
			ID: "amazon-bedrock", DisplayName: "Amazon Bedrock", Description: "AWS Bedrock Runtime Converse API.",
			Aliases: []string{"aws", "bedrock", "aws-bedrock"}, Transport: TransportBedrockConverse,
			AuthTypes: []AuthType{AuthAWSSDK}, DefaultAuthType: AuthAWSSDK,
			DefaultBaseURL: "https://bedrock-runtime.us-east-1.amazonaws.com", BaseURLEnvVar: "BEDROCK_BASE_URL",
			ModelFetch: ModelFetchStatic, DefaultModelID: "anthropic.claude-sonnet-4-20250514-v1:0", BuiltIn: true,
			Models: []domain.ModelInfo{
				model("amazon-bedrock", "anthropic.claude-sonnet-4-20250514-v1:0", "Claude Sonnet 4", true, 200000, []string{"tools", "reasoning"}),
				model("amazon-bedrock", "anthropic.claude-3-7-sonnet-20250219-v1:0", "Claude 3.7 Sonnet", false, 200000, []string{"tools", "reasoning"}),
				model("amazon-bedrock", "amazon.nova-pro-v1:0", "Amazon Nova Pro", false, 300000, []string{"tools"}),
			},
		},
		{
			ID: "openrouter", DisplayName: "OpenRouter", Description: "OpenAI-compatible routing provider.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://openrouter.ai/api/v1", BaseURLEnvVar: "OPENROUTER_BASE_URL", APIKeyEnvVars: []string{"OPENROUTER_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "openai/gpt-5-codex", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{
				Headers: map[string]string{"HTTP-Referer": "https://aivo.local", "X-Title": "Aivo"},
				Params:  map[string]any{"stream": true},
				ModelOverrides: map[string]domain.ProviderRequestOverride{
					"openai/gpt":         {Params: map[string]any{"reasoning": map[string]any{"effort": "medium"}}},
					"z-ai/glm-5.2":       {Params: map[string]any{"reasoning": map[string]any{"effort": "high"}}},
					"grok-3-mini":        {Params: map[string]any{"reasoning": map[string]any{"effort": "low"}}},
					"anthropic/claude-*": {Params: map[string]any{"reasoning": map[string]any{"effort": "none"}}},
				},
			},
			Models: []domain.ModelInfo{model("openrouter", "openai/gpt-5-codex", "GPT-5 Codex", true, 400000, []string{"tools", "reasoning", "streaming"})},
		},
		{
			ID: "xai", DisplayName: "xAI", Description: "xAI Grok models through the OpenAI-compatible API.",
			Aliases: []string{"grok", "x-ai", "x.ai"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.x.ai/v1", BaseURLEnvVar: "XAI_BASE_URL", APIKeyEnvVars: []string{"XAI_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "grok-4.3", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				modelWithOutput("xai", "grok-4.3", "Grok 4.3", true, 1000000, 0, []string{"tools", "vision", "reasoning", "streaming", "web_search", "x_search", "code_interpreter", "file_search", "remote_mcp"}),
				model("xai", "grok-build-0.1", "Grok Build 0.1", false, 256000, []string{"tools", "streaming"}),
			},
		},
		{
			ID: "mistral", DisplayName: "Mistral AI", Description: "Mistral La Plateforme chat models.",
			Aliases: []string{"mistral-ai", "la-plateforme"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.mistral.ai/v1", BaseURLEnvVar: "MISTRAL_BASE_URL", APIKeyEnvVars: []string{"MISTRAL_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "mistral-medium-latest", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				modelWithOutput("mistral", "mistral-medium-latest", "Mistral Medium", true, 256000, 0, []string{"tools", "vision", "streaming"}),
				modelWithOutput("mistral", "devstral-latest", "Devstral", false, 256000, 0, []string{"tools", "streaming"}),
				modelWithOutput("mistral", "codestral-latest", "Codestral", false, 128000, 0, []string{"tools", "streaming"}),
				modelWithOutput("mistral", "magistral-medium-latest", "Magistral Medium", false, 128000, 0, []string{"reasoning", "streaming"}),
			},
		},
		{
			ID: "groq", DisplayName: "Groq", Description: "GroqCloud OpenAI-compatible high-speed inference.",
			Aliases: []string{"groqcloud"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.groq.com/openai/v1", BaseURLEnvVar: "GROQ_BASE_URL", APIKeyEnvVars: []string{"GROQ_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "openai/gpt-oss-120b", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				modelWithOutput("groq", "openai/gpt-oss-120b", "GPT OSS 120B", true, 131072, 65536, []string{"tools", "reasoning", "streaming"}),
				modelWithOutput("groq", "openai/gpt-oss-20b", "GPT OSS 20B", false, 131072, 65536, []string{"tools", "reasoning", "streaming"}),
				modelWithOutput("groq", "qwen/qwen3.6-27b", "Qwen3.6 27B", false, 131072, 32768, []string{"tools", "reasoning", "streaming"}),
			},
		},
		{
			ID: "deepinfra", DisplayName: "DeepInfra", Description: "DeepInfra OpenAI-compatible model catalog.",
			Aliases: []string{"deep-infra"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.deepinfra.com/v1/openai", BaseURLEnvVar: "DEEPINFRA_BASE_URL", APIKeyEnvVars: []string{"DEEPINFRA_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "Qwen/Qwen3-Coder-480B-A35B-Instruct-Turbo", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				modelWithOutput("deepinfra", "Qwen/Qwen3-Coder-480B-A35B-Instruct-Turbo", "Qwen3 Coder 480B Turbo", true, 256000, 0, []string{"tools", "reasoning", "streaming"}),
				model("deepinfra", "deepseek-ai/DeepSeek-V3.1", "DeepSeek V3.1", false, 128000, []string{"tools", "streaming"}),
				model("deepinfra", "openai/gpt-oss-120b", "GPT OSS 120B", false, 128000, []string{"tools", "reasoning", "streaming"}),
			},
		},
		{
			ID: "cerebras", DisplayName: "Cerebras", Description: "Cerebras Inference OpenAI-compatible endpoint.",
			Aliases: []string{"cerebras-ai"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.cerebras.ai/v1", BaseURLEnvVar: "CEREBRAS_BASE_URL", APIKeyEnvVars: []string{"CEREBRAS_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "zai-glm-4.7", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				modelWithOutput("cerebras", "zai-glm-4.7", "Z.ai GLM 4.7", true, 128000, 8192, []string{"tools", "reasoning", "streaming"}),
				modelWithOutput("cerebras", "gpt-oss-120b", "GPT OSS 120B", false, 128000, 8192, []string{"tools", "reasoning", "streaming"}),
				modelWithOutput("cerebras", "gemma-4-31b", "Gemma 4 31B", false, 128000, 8192, []string{"tools", "vision", "streaming"}),
			},
		},
		{
			ID: "together", DisplayName: "Together AI", Description: "Together AI serverless OpenAI-compatible inference.",
			Aliases: []string{"together-ai"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.together.ai/v1", BaseURLEnvVar: "TOGETHER_BASE_URL", APIKeyEnvVars: []string{"TOGETHER_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "moonshotai/Kimi-K2.5", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("together", "moonshotai/Kimi-K2.5", "Kimi K2.5", true, 256000, []string{"tools", "streaming"}),
				model("together", "zai-org/GLM-5", "GLM-5", false, 256000, []string{"tools", "reasoning", "streaming"}),
				model("together", "openai/gpt-oss-120b", "GPT OSS 120B", false, 128000, []string{"tools", "reasoning", "streaming"}),
			},
		},
		{
			ID: "perplexity", DisplayName: "Perplexity", Description: "Perplexity Sonar search-grounded chat models.",
			Aliases: []string{"pplx"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.perplexity.ai", BaseURLEnvVar: "PERPLEXITY_BASE_URL", APIKeyEnvVars: []string{"PERPLEXITY_API_KEY"},
			ModelFetch: ModelFetchStatic, DefaultModelID: "sonar-pro", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("perplexity", "sonar-pro", "Sonar Pro", true, 200000, []string{"tools", "search", "web_search", "streaming"}),
				model("perplexity", "sonar", "Sonar", false, 127000, []string{"tools", "search", "web_search", "streaming"}),
				model("perplexity", "sonar-reasoning-pro", "Sonar Reasoning Pro", false, 128000, []string{"tools", "search", "web_search", "reasoning", "streaming"}),
				model("perplexity", "sonar-deep-research", "Sonar Deep Research", false, 128000, []string{"search", "web_search", "reasoning", "streaming"}),
			},
		},
		{
			ID: "github-copilot", DisplayName: "GitHub Copilot", Description: "GitHub Copilot Chat completions endpoint using a Copilot bearer token.",
			Aliases: []string{"copilot", "github-copilot-api"}, Transport: TransportGitHubCopilot,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.githubcopilot.com", BaseURLEnvVar: "GITHUB_COPILOT_BASE_URL", APIKeyEnvVars: []string{"GITHUB_COPILOT_TOKEN", "COPILOT_TOKEN", "GH_COPILOT_TOKEN"},
			ModelFetch: ModelFetchStatic, DefaultModelID: "gpt-5.1-codex-max", BuiltIn: true,
			RequestProfile: githubCopilotRequestProfile(),
			Models: []domain.ModelInfo{
				model("github-copilot", "gpt-5.1-codex-max", "GPT-5.1 Codex Max", true, 400000, []string{"tools", "reasoning", "streaming"}),
				model("github-copilot", "gpt-5.1-codex", "GPT-5.1 Codex", false, 400000, []string{"tools", "reasoning", "streaming"}),
				model("github-copilot", "gpt-4.1", "GPT-4.1", false, 1000000, []string{"tools", "streaming"}),
				model("github-copilot", "claude-sonnet-4.5", "Claude Sonnet 4.5", false, 200000, []string{"tools", "reasoning", "streaming"}),
			},
		},
		{
			ID: "custom-api", DisplayName: "Custom API", Description: "User-configured OpenAI, Anthropic, Google, or compatible endpoint.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey, AuthNone}, DefaultAuthType: AuthAPIKey,
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "custom-profile", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models:         []domain.ModelInfo{model("custom-api", "custom-profile", "Custom profile", true, 0, []string{"streaming"})},
		},
	}
	return defs
}

func model(providerID, id, name string, recommended bool, context int, capabilities []string) domain.ModelInfo {
	modalities := []string{"text"}
	if containsString(capabilities, "vision") {
		modalities = append(modalities, "image")
	}
	var reasoningControls []string
	if containsString(capabilities, "reasoning") {
		reasoningControls = []string{"effort"}
	}
	return domain.ModelInfo{
		ID: id, ProviderID: providerID, Name: name, Recommended: recommended, ContextLength: context,
		Capabilities: capabilities, Streaming: containsString(capabilities, "streaming"), ToolSupport: containsString(capabilities, "tools"),
		ReasoningControls: reasoningControls, Status: "active", Modalities: modalities,
	}
}

func modelWithOutput(providerID, id, name string, recommended bool, context int, output int, capabilities []string) domain.ModelInfo {
	model := model(providerID, id, name, recommended, context, capabilities)
	model.OutputLimit = output
	return model
}

func anthropicDefaultRequestProfile() domain.ProviderRequestProfile {
	return domain.ProviderRequestProfile{
		Headers: map[string]string{"anthropic-version": "2023-06-01"},
		Params:  map[string]any{"max_tokens": 4096, "stream": true},
	}
}

func googleDefaultRequestProfile() domain.ProviderRequestProfile {
	return domain.ProviderRequestProfile{
		ModelOverrides: map[string]domain.ProviderRequestOverride{
			"gemini-2.5-pro":   {Params: map[string]any{"generationConfig": map[string]any{"thinkingConfig": map[string]any{"includeThoughts": true, "thinkingBudget": 16000}}}},
			"gemini-2.5-flash": {Params: map[string]any{"generationConfig": map[string]any{"thinkingConfig": map[string]any{"includeThoughts": true, "thinkingBudget": 16000}}}},
			"gemini-3-pro-preview": {Params: map[string]any{
				"generationConfig": map[string]any{"thinkingConfig": map[string]any{"includeThoughts": true, "thinkingLevel": "low"}},
			}},
			"gemini-3-flash-preview": {Params: map[string]any{
				"generationConfig": map[string]any{"thinkingConfig": map[string]any{"includeThoughts": true, "thinkingLevel": "minimal"}},
			}},
			"gemini-3.1-flash-image-preview": {Params: map[string]any{
				"generationConfig": map[string]any{"thinkingConfig": map[string]any{"includeThoughts": true, "thinkingLevel": "minimal"}},
			}},
			"gemini-3-pro-image-preview": {Params: map[string]any{
				"generationConfig": map[string]any{"thinkingConfig": map[string]any{"includeThoughts": true, "thinkingLevel": "high"}},
			}},
		},
	}
}

func githubCopilotRequestProfile() domain.ProviderRequestProfile {
	return domain.ProviderRequestProfile{
		Headers: map[string]string{
			"Copilot-Integration-Id": "vscode-chat",
			"Editor-Plugin-Version":  "aivo/0.0.0",
			"Editor-Version":         "Aivo/0.0.0",
			"OpenAI-Intent":          "conversation-panel",
		},
		Params: map[string]any{"stream": true},
	}
}

func providerDefinition(providerID string) (ProviderDefinition, bool) {
	return defaultProviderRegistry.Definition(providerID)
}

func (s *Service) normalizeProviderID(providerID string) string {
	if s != nil && s.providers != nil {
		return s.providers.Normalize(providerID)
	}
	return normalizeProviderID(providerID)
}

func (s *Service) providerDefinitions() []ProviderDefinition {
	if s != nil && s.providers != nil {
		return s.providers.Definitions()
	}
	return providerDefinitions()
}

func (s *Service) providerDefinition(providerID string) (ProviderDefinition, bool) {
	if s != nil && s.providers != nil {
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

func providerInfoFromDefinition(def ProviderDefinition) domain.ProviderInfo {
	env := ""
	if len(def.APIKeyEnvVars) > 0 {
		env = def.APIKeyEnvVars[0]
	}
	models := append([]domain.ModelInfo(nil), def.Models...)
	if len(models) == 0 && def.DefaultModelID != "" {
		models = []domain.ModelInfo{model(def.ID, def.DefaultModelID, def.DefaultModelID, true, 0, nil)}
	}
	return domain.ProviderInfo{
		ID: def.ID, Name: def.DisplayName, Description: def.Description, Type: string(def.Transport), BaseURL: def.DefaultBaseURL,
		BuiltIn: def.BuiltIn, Custom: !def.BuiltIn, Experimental: def.Experimental, Deprecated: def.Deprecated, Environment: env,
		DefaultModelID: def.DefaultModelID, Models: models, AuthMethods: authMethodsForDefinition(def),
		ModelRefresh: &domain.ProviderModelRefresh{
			Strategy: string(def.ModelFetch), Status: "idle", ModelCount: len(models), Refreshable: providerModelRefreshable(def),
			ParserType: parserTypeForModelFetch(def.ModelFetch), Endpoint: modelEndpointForDefinition(def),
		},
		Profile: &domain.ProviderProfile{
			ID: def.ID, DisplayName: def.DisplayName, ProviderType: string(def.Transport), InteractiveAuth: providerSupportsInteractiveAuth(def),
			ModelFetch: string(def.ModelFetch),
			ParserType: parserTypeForModelFetch(def.ModelFetch), ModelEndpoint: modelEndpointForDefinition(def), MessageShape: string(def.Transport),
			SupportedExtras: supportedExtrasForTransport(def.Transport),
			RequestProfile:  requestProfilePointer(def.RequestProfile),
		},
	}
}

func requestProfilePointer(profile domain.ProviderRequestProfile) *domain.ProviderRequestProfile {
	if len(profile.Headers) == 0 && len(profile.Params) == 0 && len(profile.ModelOverrides) == 0 {
		return nil
	}
	cloned := cloneRequestProfile(profile)
	return &cloned
}

func authMethodsForDefinition(def ProviderDefinition) []domain.ProviderAuthMethod {
	var methods []domain.ProviderAuthMethod
	seen := map[string]bool{}
	add := func(id, label, description string, stable bool) {
		if seen[id] {
			return
		}
		seen[id] = true
		methods = append(methods, domain.ProviderAuthMethod{ID: id, Label: label, Stable: stable, Available: true, Description: description})
	}
	for _, authType := range def.AuthTypes {
		switch authType {
		case AuthOAuthBrowser:
			if def.ID == "openai" {
				add("oauth-browser", "ChatGPT Pro/Plus (browser)", "OpenAI browser OAuth with PKCE and localhost callback.", false)
			} else {
				add("oauth-browser", "OAuth browser", "Browser OAuth with localhost callback.", false)
			}
		case AuthOAuthDevice:
			if def.ID == "openai" {
				add("oauth-headless", "ChatGPT Pro/Plus (headless)", "OpenAI device authorization flow.", false)
			} else {
				add("oauth-headless", "OAuth device code", "Device authorization flow for headless environments.", false)
			}
		case AuthAPIKey:
			add("api-key", "API Key", "", true)
			if len(def.APIKeyEnvVars) > 0 {
				add("env", "Credential reference", strings.Join(def.APIKeyEnvVars, ", "), true)
			}
		case AuthNone:
			add("none", "No credential", "Use for local or unauthenticated compatible endpoints.", true)
		case AuthAWSSDK:
			add("aws-sdk", "AWS SDK", "Resolve credentials from the AWS SDK chain.", true)
		case AuthExternalProcess:
			add("external-process", "External process", "Resolve credentials from an external provider process.", false)
		}
	}
	return methods
}

func defaultProviders() []domain.ProviderInfo {
	defs := providerDefinitions()
	out := make([]domain.ProviderInfo, 0, len(defs))
	for _, def := range defs {
		out = append(out, providerInfoFromDefinition(def))
	}
	return out
}

func defaultEnvFor(providerID string) string {
	if def, ok := providerDefinition(providerID); ok && len(def.APIKeyEnvVars) > 0 {
		return def.APIKeyEnvVars[0]
	}
	return ""
}

func defaultEnvCandidatesFor(providerID string) []string {
	if def, ok := providerDefinition(providerID); ok {
		return append([]string(nil), def.APIKeyEnvVars...)
	}
	return nil
}

func defaultModelFor(providerID string) string {
	if def, ok := providerDefinition(providerID); ok {
		return def.DefaultModelID
	}
	return "default"
}

func defaultBaseURLFor(providerID string) string {
	if def, ok := providerDefinition(providerID); ok {
		return def.DefaultBaseURL
	}
	return ""
}

func providerTypeFor(providerID string) string {
	if def, ok := providerDefinition(providerID); ok {
		return string(def.Transport)
	}
	return normalizeProviderID(providerID)
}

func (s *Service) ResolveModelRoute(ctx context.Context, cfg domain.AppConfig, requestedModel *domain.ModelRef) (ResolvedModelRoute, error) {
	provider, modelRef := resolveActiveProvider(cfg)
	if requestedModel != nil && strings.TrimSpace(requestedModel.ModelID) != "" {
		requestedProviderID := s.normalizeProviderID(requestedModel.ProviderID)
		if requestedProviderID != "" && requestedProviderID != s.normalizeProviderID(provider.ID) {
			provider = s.providerConfigForModelRequest(cfg, requestedProviderID, strings.TrimSpace(requestedModel.ModelID))
			modelRef = domain.ModelRef{ProviderID: provider.ID, ModelID: provider.Model}
		} else {
			modelRef.ModelID = strings.TrimSpace(requestedModel.ModelID)
			if provider.ID == "" {
				provider.ID = normalizeProviderID(requestedModel.ProviderID)
			}
			modelRef.ProviderID = provider.ID
			provider.Model = modelRef.ModelID
		}
	}
	provider.ID = s.normalizeProviderID(provider.ID)
	if provider.ID == "" {
		return ResolvedModelRoute{}, errors.New("provider is not configured")
	}
	def := s.providerDefinitionForConfig(provider)
	provider.Type = firstNonEmpty(provider.Type, string(def.Transport))
	if provider.BaseURL == "" {
		provider.BaseURL = def.DefaultBaseURL
	}
	if provider.APIKeyEnv == "" && len(def.APIKeyEnvVars) > 0 {
		provider.APIKeyEnv = def.APIKeyEnvVars[0]
	}
	if provider.Model == "" {
		provider.Model = firstNonEmpty(modelRef.ModelID, def.DefaultModelID)
	}
	modelRef = domain.ModelRef{ProviderID: provider.ID, ModelID: normalizeModelIDForProvider(provider.ID, provider.Model)}
	if modelRef.ModelID == "" {
		return ResolvedModelRoute{}, errors.New("model is not configured")
	}
	transport := inferTransport(provider.ID, provider.Type, provider.BaseURL)
	credential, err := s.resolveCredentialWithDefinition(ctx, provider, def)
	if err != nil {
		return ResolvedModelRoute{}, err
	}
	return ResolvedModelRoute{Provider: provider, Model: modelRef, Definition: def, Transport: transport, BaseURL: provider.BaseURL, Credential: credential}, nil
}

func inferTransport(providerID string, providerType string, baseURL string) TransportType {
	providerID = normalizeProviderID(providerID)
	providerType = strings.TrimSpace(strings.ToLower(providerType))
	if def, ok := providerDefinition(providerID); ok {
		if detected := inferTransportFromURL(baseURL); detected != "" && providerID == "custom-api" {
			return detected
		}
		return def.Transport
	}
	switch providerType {
	case string(TransportAzureOpenAI), "azure", "azure-openai":
		return TransportAzureOpenAI
	case string(TransportOpenAIResponses), "openai", "codex_responses":
		return TransportOpenAIResponses
	case string(TransportAnthropicMessages), "anthropic", "claude":
		return TransportAnthropicMessages
	case string(TransportGoogleGemini), "google", "gemini":
		return TransportGoogleGemini
	case string(TransportGoogleVertex), "google-vertex", "vertex", "vertex-ai":
		return TransportGoogleVertex
	case string(TransportBedrockConverse), "bedrock", "aws":
		return TransportBedrockConverse
	case string(TransportGitHubCopilot), "github-copilot", "copilot":
		return TransportGitHubCopilot
	case string(TransportOpenAIChat):
		return TransportOpenAIChat
	case string(TransportOpenAICompatible), "openai-compatible", "":
		if detected := inferTransportFromURL(baseURL); detected != "" {
			return detected
		}
		return TransportOpenAICompatible
	default:
		if detected := inferTransportFromURL(baseURL); detected != "" {
			return detected
		}
		return TransportOpenAICompatible
	}
}

func inferTransportFromURL(raw string) TransportType {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	host := strings.ToLower(parsed.Hostname())
	path := strings.ToLower(strings.TrimRight(parsed.Path, "/"))
	switch {
	case strings.HasSuffix(host, ".openai.azure.com") || strings.HasSuffix(host, ".services.ai.azure.com"):
		return TransportAzureOpenAI
	case host == "api.openai.com" || host == "api.x.ai":
		return TransportOpenAIResponses
	case host == "api.anthropic.com" || strings.HasSuffix(path, "/anthropic") || strings.HasSuffix(path, "/anthropic/v1"):
		return TransportAnthropicMessages
	case host == "api.kimi.com" && strings.Contains(path, "/coding"):
		return TransportAnthropicMessages
	case strings.Contains(host, "generativelanguage.googleapis.com"):
		return TransportGoogleGemini
	case strings.Contains(host, "aiplatform.googleapis.com") && strings.Contains(path, "/publishers/google"):
		return TransportGoogleVertex
	case strings.HasPrefix(host, "bedrock-runtime.") && strings.HasSuffix(host, ".amazonaws.com"):
		return TransportBedrockConverse
	case host == "api.githubcopilot.com" || host == "api.individual.githubcopilot.com" || host == "api.business.githubcopilot.com":
		return TransportGitHubCopilot
	default:
		return ""
	}
}

func (s *Service) resolveCredentialWithDefinition(ctx context.Context, provider domain.ProviderConfig, def ProviderDefinition) (llmCredential, error) {
	auth, err := s.store.LoadProviderAuth(ctx, provider.ID)
	if err != nil {
		return llmCredential{}, err
	}
	if auth != nil {
		resolvedAuth, err := s.resolveProviderAuthSecrets(ctx, *auth)
		if err != nil {
			return llmCredential{}, err
		}
		auth = &resolvedAuth
		if isOAuthMethod(auth.Method) {
			if auth.AccessToken == "" && auth.RefreshToken == "" {
				return llmCredential{}, errors.New("OAuth credentials are missing")
			}
			return llmCredential{Method: auth.Method, AccessToken: auth.AccessToken, Refresh: auth.RefreshToken, ExpiresAt: auth.ExpiresAt, AccountID: auth.AccountID, AuthRecord: auth}, nil
		}
		if strings.TrimSpace(auth.APIKey) != "" {
			return llmCredential{Method: auth.Method, APIKey: strings.TrimSpace(auth.APIKey), AuthRecord: auth}, nil
		}
	}
	for _, envName := range credentialEnvCandidates(provider, def) {
		if value := lookupEnv(strings.TrimSpace(envName)); value != "" {
			return llmCredential{Method: "env", APIKey: value}, nil
		}
	}
	if strings.TrimSpace(provider.APIKey) != "" {
		return llmCredential{Method: "api-key", APIKey: strings.TrimSpace(provider.APIKey)}, nil
	}
	if providerAllowsNoCredential(provider, def) {
		return llmCredential{Method: "none"}, nil
	}
	return llmCredential{}, errors.New("credentials are not configured for provider " + provider.ID)
}

func credentialEnvCandidates(provider domain.ProviderConfig, def ProviderDefinition) []string {
	var out []string
	out = append(out, splitEnvCandidates(provider.APIKeyEnv)...)
	out = append(out, def.APIKeyEnvVars...)
	seen := map[string]bool{}
	filtered := out[:0]
	for _, item := range out {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		filtered = append(filtered, item)
	}
	return filtered
}

func splitEnvCandidates(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	fields := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' || r == '|' || r == ' ' || r == '\n' || r == '\t' })
	var out []string
	for _, field := range fields {
		if field = strings.TrimSpace(field); field != "" {
			out = append(out, field)
		}
	}
	return out
}

var lookupEnv = func(name string) string { return strings.TrimSpace(os.Getenv(name)) }

func providerAllowsNoCredential(provider domain.ProviderConfig, def ProviderDefinition) bool {
	if def.DefaultAuthType == AuthAWSSDK || containsAuthType(def.AuthTypes, AuthAWSSDK) {
		return true
	}
	if def.DefaultAuthType == AuthNone || containsAuthType(def.AuthTypes, AuthNone) {
		return true
	}
	base := strings.TrimSpace(provider.BaseURL)
	parsed, err := url.Parse(base)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func containsAuthType(items []AuthType, target AuthType) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func providerModelRefreshable(def ProviderDefinition) bool {
	return def.ModelFetch != ModelFetchStatic && def.ModelFetch != ModelFetchDisabled
}

func providerSupportsInteractiveAuth(def ProviderDefinition) bool {
	return containsAuthType(def.AuthTypes, AuthOAuthBrowser) || containsAuthType(def.AuthTypes, AuthOAuthDevice) || containsAuthType(def.AuthTypes, AuthExternalProcess)
}

func parserTypeForModelFetch(strategy ModelFetchStrategy) string {
	switch strategy {
	case ModelFetchAnthropic:
		return "anthropic"
	case ModelFetchGoogle:
		return "google"
	case ModelFetchOpenAICodexAccount:
		return "openai-codex"
	case ModelFetchOpenAICompatible:
		return "openai-compatible"
	default:
		return string(strategy)
	}
}

func modelEndpointForDefinition(def ProviderDefinition) string {
	if def.DefaultBaseURL == "" || def.ModelFetch == ModelFetchDisabled || def.ModelFetch == ModelFetchStatic {
		return ""
	}
	return strings.TrimRight(def.DefaultBaseURL, "/") + "/models"
}

func supportedExtrasForTransport(transport TransportType) []string {
	switch transport {
	case TransportOpenAIResponses:
		return []string{"reasoning", "service_tier", "tools", "streaming"}
	case TransportAzureOpenAI:
		return []string{"reasoning", "tools", "streaming"}
	case TransportAnthropicMessages:
		return []string{"thinking", "tools", "streaming"}
	case TransportGoogleGemini, TransportGoogleVertex:
		return []string{"thinking", "tools", "streaming"}
	case TransportGitHubCopilot:
		return []string{"reasoning", "tools", "streaming"}
	default:
		return []string{"tools", "streaming"}
	}
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func sortProviderInfo(providers []domain.ProviderInfo) {
	sort.SliceStable(providers, func(i, j int) bool {
		if providers[i].BuiltIn != providers[j].BuiltIn {
			return providers[i].BuiltIn
		}
		return providers[i].Name < providers[j].Name
	})
}

func safeProviderError(err error) string {
	if err == nil {
		return ""
	}
	text := strings.TrimSpace(err.Error())
	if text == "" {
		return "provider validation failed"
	}
	lower := strings.ToLower(text)
	if strings.Contains(lower, "unauthorized") || strings.Contains(lower, "forbidden") || strings.Contains(lower, "authentication failed") {
		return "authentication failed"
	}
	if strings.Contains(lower, "timed out") || strings.Contains(lower, "timeout") {
		return "provider request timed out"
	}
	if strings.Contains(lower, "no such host") || strings.Contains(lower, "connection refused") {
		return "provider endpoint could not be reached"
	}
	if len(text) > 240 {
		text = text[:240]
	}
	return redactProviderSecretFragments(text)
}

func redactProviderSecretFragments(text string) string {
	fields := strings.Fields(text)
	for i, field := range fields {
		lower := strings.ToLower(field)
		if strings.Contains(lower, "sk-") || strings.Contains(lower, "token") || strings.Contains(lower, "key=") || strings.Contains(lower, "authorization") {
			fields[i] = "[redacted]"
		}
	}
	return strings.Join(fields, " ")
}
