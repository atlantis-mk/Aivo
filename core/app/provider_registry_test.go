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

func TestProviderRegistryDeclaresDefaultResponsesProviders(t *testing.T) {
	for _, providerID := range []string{"xai", "xiaomi", "deepseek", "openrouter", "groq", "lmstudio", "ovhcloud"} {
		t.Run(providerID, func(t *testing.T) {
			def, ok := providerDefinition(providerID)
			if !ok {
				t.Fatalf("provider definition %q missing", providerID)
			}
			if !def.DefaultResponsesAPI {
				t.Fatalf("%s DefaultResponsesAPI = false, want true", providerID)
			}
		})
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
		{id: "xiaomi", alias: "mimo", baseURL: "https://api.xiaomimimo.com/v1", env: "MIMO_API_KEY", defaultModel: "mimo-v2.5-pro", transport: TransportOpenAICompatible, refreshable: true, capability: "reasoning"},
		{id: "mistral", alias: "mistral-ai", baseURL: "https://api.mistral.ai/v1", env: "MISTRAL_API_KEY", defaultModel: "mistral-medium-latest", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "groq", alias: "groqcloud", baseURL: "https://api.groq.com/openai/v1", env: "GROQ_API_KEY", defaultModel: "openai/gpt-oss-120b", transport: TransportOpenAICompatible, refreshable: true, capability: "reasoning"},
		{id: "deepinfra", alias: "deep-infra", baseURL: "https://api.deepinfra.com/v1/openai", env: "DEEPINFRA_API_KEY", defaultModel: "Qwen/Qwen3-Coder-480B-A35B-Instruct-Turbo", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "cerebras", alias: "cerebras-ai", baseURL: "https://api.cerebras.ai/v1", env: "CEREBRAS_API_KEY", defaultModel: "zai-glm-4.7", transport: TransportOpenAICompatible, refreshable: true, capability: "reasoning"},
		{id: "baseten", alias: "baseten", baseURL: "https://inference.baseten.co/v1", env: "BASETEN_API_KEY", defaultModel: "custom-profile", transport: TransportOpenAICompatible, refreshable: true, capability: "streaming"},
		{id: "deepseek", alias: "deep-seek", baseURL: "https://api.deepseek.com/v1", env: "DEEPSEEK_API_KEY", defaultModel: "deepseek-chat", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "fireworks", alias: "fireworks-ai", baseURL: "https://api.fireworks.ai/inference/v1", env: "FIREWORKS_API_KEY", defaultModel: "accounts/fireworks/models/glm-5p2", transport: TransportOpenAICompatible, refreshable: true, capability: "reasoning"},
		{id: "together", alias: "togetherai", baseURL: "https://api.together.ai/v1", env: "TOGETHER_API_KEY", defaultModel: "MiniMaxAI/MiniMax-M3", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "alibaba-coding-plan", alias: "bailian-coding", baseURL: "https://coding-intl.dashscope.aliyuncs.com/v1", env: "ALIBABA_CODING_PLAN_API_KEY", defaultModel: "qwen3-coder-plus", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "alibaba-coding-plan-cn", alias: "alibaba-coding-plan-cn", baseURL: "https://coding.dashscope.aliyuncs.com/v1", env: "ALIBABA_CODING_PLAN_CN_API_KEY", defaultModel: "qwen3-coder-plus", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "alibaba", alias: "dashscope", baseURL: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1", env: "ALIBABA_API_KEY", defaultModel: "qwen3-235b-a22b", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "moonshotai", alias: "kimi-api", baseURL: "https://api.moonshot.ai/v1", env: "MOONSHOT_API_KEY", defaultModel: "kimi-k2-0905-preview", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "kimi-for-coding", alias: "kimi", baseURL: "https://api.kimi.com/coding/v1", env: "KIMI_FOR_CODING_API_KEY", defaultModel: "k2p6", transport: TransportAnthropicMessages, refreshable: true, capability: "reasoning"},
		{id: "minimax", alias: "minimax", baseURL: "https://api.minimax.io/anthropic/v1", env: "MINIMAX_API_KEY", defaultModel: "MiniMax-M2", transport: TransportAnthropicMessages, refreshable: true, capability: "reasoning"},
		{id: "siliconflow", alias: "siliconflow", baseURL: "https://api.siliconflow.com/v1", env: "SILICONFLOW_API_KEY", defaultModel: "nex-agi/DeepSeek-V3.1-Nex-N1", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "stepfun", alias: "stepfun", baseURL: "https://api.stepfun.com/v1", env: "STEPFUN_API_KEY", defaultModel: "step-3.5-flash-2603", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "cloudflare-workers-ai", alias: "cloudflare-workers-ai", baseURL: "https://api.cloudflare.com/client/v4/accounts/${CLOUDFLARE_ACCOUNT_ID}/ai/v1", env: "CLOUDFLARE_API_KEY", defaultModel: "@cf/zai-org/glm-4.7-flash", transport: TransportOpenAICompatible, refreshable: false, capability: "tools"},
		{id: "huggingface", alias: "hf", baseURL: "https://router.huggingface.co/v1", env: "HF_TOKEN", defaultModel: "Qwen/Qwen3.5-397B-A17B", transport: TransportOpenAICompatible, refreshable: false, capability: "tools"},
		{id: "friendli", alias: "friendli", baseURL: "https://api.friendli.ai/serverless/v1", env: "FRIENDLI_API_KEY", defaultModel: "Qwen/Qwen3-235B-A22B-Instruct-2507", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "novita-ai", alias: "novita-ai", baseURL: "https://api.novita.ai/openai", env: "NOVITA_API_KEY", defaultModel: "deepseek/deepseek-r1-turbo", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "nvidia", alias: "nvidia", baseURL: "https://integrate.api.nvidia.com/v1", env: "NVIDIA_API_KEY", defaultModel: "upstage/solar-10_7b-instruct", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "zai", alias: "z-ai", baseURL: "https://api.z.ai/api/paas/v4", env: "ZAI_API_KEY", defaultModel: "glm-5v-turbo", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "zai-coding-plan", alias: "zai-coding-plan", baseURL: "https://api.z.ai/api/coding/paas/v4", env: "ZAI_CODING_PLAN_API_KEY", defaultModel: "glm-4.7", transport: TransportOpenAICompatible, refreshable: true, capability: "reasoning"},
		{id: "zhipuai", alias: "zhipuai", baseURL: "https://open.bigmodel.cn/api/paas/v4", env: "ZHIPUAI_API_KEY", defaultModel: "glm-5v-turbo", transport: TransportOpenAICompatible, refreshable: true, capability: "vision"},
		{id: "ollama-cloud", alias: "ollama-cloud", baseURL: "https://ollama.com/v1", env: "OLLAMA_API_KEY", defaultModel: "minimax-m2.7", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "lmstudio", alias: "lm-studio", baseURL: "http://127.0.0.1:1234/v1", env: "LMSTUDIO_API_KEY", defaultModel: "openai/gpt-oss-20b", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "inference", alias: "inference-net", baseURL: "https://api.inference.net/v1", env: "INFERENCE_API_KEY", defaultModel: "glm-5.2", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "tencent-coding-plan", alias: "tencent-tokenhub", baseURL: "https://api.lkeap.cloud.tencent.com/coding/v3", env: "TENCENT_CODING_PLAN_API_KEY", defaultModel: "kimi-k2.6", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "scaleway", alias: "scaleway", baseURL: "https://api.scaleway.ai/v1", env: "SCW_SECRET_KEY", defaultModel: "qwen3.5-397b-a17b", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "stackit", alias: "stackit", baseURL: "https://api.openai-compat.model-serving.eu01.onstackit.cloud/v1", env: "STACKIT_API_KEY", defaultModel: "Qwen/Qwen3.6-27B", transport: TransportOpenAICompatible, refreshable: true, capability: "reasoning"},
		{id: "vultr", alias: "vultr", baseURL: "https://api.vultrinference.com/v1", env: "VULTR_INFERENCE_API_KEY", defaultModel: "kimi-k2-instruct", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "digitalocean", alias: "do-gradient", baseURL: "https://inference.do-ai.run/v1", env: "DIGITALOCEAN_TOKEN", defaultModel: "openai-gpt-4o-mini", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "ovhcloud", alias: "ovh", baseURL: "https://oai.endpoints.kepler.ai.cloud.ovh.net/v1", env: "OVHCLOUD_API_KEY", defaultModel: "gpt-oss-20b", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "helicone", alias: "helicone-ai-gateway", baseURL: "https://ai-gateway.helicone.ai/v1", env: "HELICONE_API_KEY", defaultModel: "gpt-4o-mini", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "clarifai", alias: "clarifai-openai", baseURL: "https://api.clarifai.com/v2/ext/openai/v1", env: "CLARIFAI_API_KEY", defaultModel: "arcee_ai/AFM/models/trinity-mini", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "cloudferro-sherlock", alias: "sherlock", baseURL: "https://api-sherlock.cloudferro.com/openai/v1", env: "CLOUDFERRO_SHERLOCK_API_KEY", defaultModel: "meta-llama/Llama-3.3-70B-Instruct", transport: TransportOpenAICompatible, refreshable: true, capability: "streaming"},
		{id: "upstage", alias: "upstage-solar", baseURL: "https://api.upstage.ai/v1/solar", env: "UPSTAGE_API_KEY", defaultModel: "solar-pro4", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "poe", alias: "poe", baseURL: "https://api.poe.com/v1", env: "POE_API_KEY", defaultModel: "GPT-5.4", transport: TransportOpenAICompatible, refreshable: true, capability: "web_search"},
		{id: "vivgrid", alias: "vivgrid", baseURL: "https://api.vivgrid.com/v1", env: "VIVGRID_API_KEY", defaultModel: "gpt-5.6-terra", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "perplexity-agent", alias: "perplexity-router-agent", baseURL: "https://api.perplexity.ai/v1", env: "PERPLEXITY_API_KEY", defaultModel: "openai/gpt-5.6-terra", transport: TransportOpenAIResponses, refreshable: true, capability: "web_search"},
		{id: "requesty", alias: "requesty", baseURL: "https://router.requesty.ai/v1", env: "REQUESTY_API_KEY", defaultModel: "anthropic/claude-sonnet-4-20250514", transport: TransportOpenAICompatible, refreshable: true, capability: "web_search"},
		{id: "nebius", alias: "nebius", baseURL: "https://api.tokenfactory.nebius.com/v1", env: "NEBIUS_API_KEY", defaultModel: "moonshotai/Kimi-K2.5", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "wandb", alias: "wandb-inference", baseURL: "https://api.inference.wandb.ai/v1", env: "WANDB_API_KEY", defaultModel: "openai/gpt-oss-20b", transport: TransportOpenAICompatible, refreshable: true, capability: "tools"},
		{id: "venice", alias: "venice", baseURL: "https://api.venice.ai/api/v1", env: "VENICE_API_KEY", defaultModel: "zai-org-glm-5", transport: TransportOpenAICompatible, refreshable: true, capability: "web_search"},
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

func TestProviderRegistryOVHDefaultsToResponsesWithoutHostedSearch(t *testing.T) {
	def, ok := providerDefinition("ovhcloud")
	if !ok {
		t.Fatal("ovhcloud provider definition missing")
	}
	if !def.DefaultResponsesAPI {
		t.Fatalf("DefaultResponsesAPI = false, want true")
	}
	if def.NativeHostedTools.WebSearch.Type != "" {
		t.Fatalf("NativeHostedTools = %+v, want no hosted web_search", def.NativeHostedTools)
	}
	if !def.ResponsesDefaults.DisableEncryptedReasoningInclude || !def.ResponsesDefaults.DisableReasoningSummary {
		t.Fatalf("ResponsesDefaults = %+v, want OpenAI-only defaults disabled", def.ResponsesDefaults)
	}
}

func TestProviderRegistryMarksRetiredGitHubModelsDeprecated(t *testing.T) {
	def, ok := providerDefinition("github-models")
	if !ok {
		t.Fatal("github-models provider definition missing")
	}
	if !def.Deprecated || def.ModelFetch != ModelFetchStatic {
		t.Fatalf("definition = %+v, want deprecated static provider", def)
	}
	if len(def.Models) != 1 || !def.Models[0].Deprecated || def.Models[0].Status != "deprecated" {
		t.Fatalf("models = %+v, want deprecated placeholder model", def.Models)
	}
	info := providerInfoFromDefinition(def)
	if !info.Deprecated || info.ModelRefresh == nil || info.ModelRefresh.Refreshable {
		t.Fatalf("info = %+v, want deprecated non-refreshable provider", info)
	}
}

func TestProviderRegistryLMStudioDefaultsToLocalNoAuthResponses(t *testing.T) {
	def, ok := providerDefinition("lmstudio")
	if !ok {
		t.Fatal("lmstudio provider definition missing")
	}
	if def.DefaultAuthType != AuthNone || len(def.AuthTypes) == 0 || def.AuthTypes[0] != AuthNone {
		t.Fatalf("AuthTypes = %+v DefaultAuthType = %q, want no-auth first", def.AuthTypes, def.DefaultAuthType)
	}
	if !def.DefaultResponsesAPI {
		t.Fatalf("DefaultResponsesAPI = false, want true")
	}
}

func TestProviderRegistryAlibabaHostedWebSearchIsModelGated(t *testing.T) {
	def, ok := providerDefinition("alibaba")
	if !ok {
		t.Fatal("alibaba provider definition missing")
	}
	if def.NativeHostedTools.WebSearch.Type != "web_search" || !def.ResponsesAPIForHostedTools {
		t.Fatalf("hosted web search metadata = %+v responsesForTools=%v", def.NativeHostedTools, def.ResponsesAPIForHostedTools)
	}
	defaultModel, ok := findModelInfo(def.Models, "qwen3-235b-a22b")
	if !ok || modelSupportsCapability(defaultModel, "web_search") {
		t.Fatalf("default model = %+v ok=%v, want no hosted web_search capability", defaultModel, ok)
	}
	searchModel, ok := findModelInfo(def.Models, "qwen3-max")
	if !ok || !modelSupportsCapability(searchModel, "web_search") {
		t.Fatalf("qwen3-max = %+v ok=%v, want hosted web_search capability", searchModel, ok)
	}
}

func TestProviderRegistryIncludesPresetProviderTemplates(t *testing.T) {
	tests := []struct {
		id           string
		baseURL      string
		env          string
		defaultModel string
		transport    TransportType
		refreshable  bool
	}{
		{id: "302ai", baseURL: "https://api.302.ai/v1", env: "AI_302_API_KEY", defaultModel: "qwen3-235b-a22b", transport: TransportOpenAICompatible, refreshable: true},
		{id: "bailing", baseURL: "https://api.tbox.cn/api/llm/v1/chat/completions", env: "BAILING_API_KEY", defaultModel: "Ring-1T", transport: TransportOpenAICompatible, refreshable: false},
		{id: "kimi-for-coding", baseURL: "https://api.kimi.com/coding/v1", env: "KIMI_FOR_CODING_API_KEY", defaultModel: "k2p6", transport: TransportAnthropicMessages, refreshable: true},
		{id: "minimax-cn-coding-plan", baseURL: "https://api.minimaxi.com/anthropic/v1", env: "MINIMAX_CN_CODING_PLAN_API_KEY", defaultModel: "MiniMax-M2", transport: TransportAnthropicMessages, refreshable: true},
		{id: "lmstudio", baseURL: "http://127.0.0.1:1234/v1", env: "LMSTUDIO_API_KEY", defaultModel: "openai/gpt-oss-20b", transport: TransportOpenAICompatible, refreshable: true},
		{id: "privatemode-ai", baseURL: "http://localhost:8080/v1", env: "PRIVATEMODE_AI_API_KEY", defaultModel: "gemma-3-27b", transport: TransportOpenAICompatible, refreshable: true},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			def, ok := providerDefinition(tt.id)
			if !ok {
				t.Fatalf("provider definition %q missing", tt.id)
			}
			if !def.BuiltIn || def.Transport != tt.transport || def.DefaultBaseURL != tt.baseURL || def.DefaultModelID != tt.defaultModel {
				t.Fatalf("definition = %+v, want shipped template metadata", def)
			}
			if len(def.APIKeyEnvVars) == 0 || def.APIKeyEnvVars[0] != tt.env {
				t.Fatalf("APIKeyEnvVars = %+v, want %q", def.APIKeyEnvVars, tt.env)
			}
			if providerModelRefreshable(def) != tt.refreshable {
				t.Fatalf("refreshable = %v, want %v", providerModelRefreshable(def), tt.refreshable)
			}
		})
	}
}

func TestProviderRegistryTemplatesDoNotOverrideSpecificDefinitions(t *testing.T) {
	if got := normalizeProviderID("togetherai"); got != "together" {
		t.Fatalf("normalizeProviderID(togetherai) = %q, want together", got)
	}
	def, ok := providerDefinition("togetherai")
	if !ok {
		t.Fatal("together provider definition missing through alias")
	}
	if def.ID != "together" || def.DefaultModelID != "MiniMaxAI/MiniMax-M3" {
		t.Fatalf("definition = %+v, want specific Together provider through alias", def)
	}
	def, ok = providerDefinition("fireworks-ai")
	if !ok {
		t.Fatal("fireworks provider definition missing through alias")
	}
	if def.ID != "fireworks" || def.DefaultModelID != "accounts/fireworks/models/glm-5p2" {
		t.Fatalf("definition = %+v, want specific Fireworks provider through alias", def)
	}
}

func TestGoogleDefaultRequestProfileDoesNotForceThinking(t *testing.T) {
	for _, providerID := range []string{"gemini", "google", "google-vertex"} {
		t.Run(providerID, func(t *testing.T) {
			def, ok := providerDefinition(providerID)
			if !ok {
				t.Fatalf("provider definition %q missing", providerID)
			}
			if len(def.RequestProfile.Params) != 0 || len(def.RequestProfile.ModelOverrides) != 0 {
				t.Fatalf("RequestProfile = %+v, want no default generation overrides", def.RequestProfile)
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
