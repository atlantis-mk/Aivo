package app

import (
	"net/url"
	"strings"

	"aivo/core/domain"
)

func providerTemplateDefinitions() []ProviderDefinition {
	return []ProviderDefinition{
		providerTemplateDefinition("302ai", "302.AI", TransportOpenAICompatible, "https://api.302.ai/v1", "AI_302_API_KEY", "qwen3-235b-a22b"),
		providerTemplateDefinition("abacus", "Abacus", TransportOpenAICompatible, "https://routellm.abacus.ai/v1", "ABACUS_API_KEY", "gpt-5.1-codex-max"),
		providerTemplateDefinition("aihubmix", "AIHubMix", TransportOpenAICompatible, "https://aihubmix.com/v1", "AIHUBMIX_API_KEY", "custom-profile"),
		providerTemplateDefinition("alibaba", "Alibaba", TransportOpenAICompatible, "https://dashscope-intl.aliyuncs.com/compatible-mode/v1", "ALIBABA_API_KEY", "qwen3-235b-a22b"),
		providerTemplateDefinition("alibaba-cn", "Alibaba CN", TransportOpenAICompatible, "https://dashscope.aliyuncs.com/compatible-mode/v1", "ALIBABA_CN_API_KEY", "qwen3-235b-a22b"),
		providerTemplateDefinition("alibaba-coding-plan", "Alibaba Coding Plan", TransportOpenAICompatible, "https://coding-intl.dashscope.aliyuncs.com/v1", "ALIBABA_CODING_PLAN_API_KEY", "qwen3-coder-plus"),
		providerTemplateDefinition("alibaba-coding-plan-cn", "Alibaba Coding Plan CN", TransportOpenAICompatible, "https://coding.dashscope.aliyuncs.com/v1", "ALIBABA_CODING_PLAN_CN_API_KEY", "qwen3-coder-plus"),
		providerTemplateDefinition("bailing", "Bailing", TransportOpenAICompatible, "https://api.tbox.cn/api/llm/v1/chat/completions", "BAILING_API_KEY", "Ring-1T"),
		providerTemplateDefinition("berget", "Berget", TransportOpenAICompatible, "https://api.berget.ai/v1", "BERGET_API_KEY", "zai-org/GLM-4.7"),
		providerTemplateDefinition("chutes", "Chutes", TransportOpenAICompatible, "https://llm.chutes.ai/v1", "CHUTES_API_KEY", "NousResearch/DeepHermes-3-Mistral-24B-Preview"),
		providerTemplateDefinition("clarifai", "Clarifai", TransportOpenAICompatible, "https://api.clarifai.com/v2/ext/openai/v1", "CLARIFAI_API_KEY", "arcee_ai/AFM/models/trinity-mini"),
		providerTemplateDefinition("cloudferro-sherlock", "Cloudferro Sherlock", TransportOpenAICompatible, "https://api-sherlock.cloudferro.com/openai/v1", "CLOUDFERRO_SHERLOCK_API_KEY", "meta-llama/Llama-3.3-70B-Instruct"),
		providerTemplateDefinition("cloudflare-workers-ai", "Cloudflare Workers AI", TransportOpenAICompatible, "https://api.cloudflare.com/client/v4/accounts/${CLOUDFLARE_ACCOUNT_ID}/ai/v1", "CLOUDFLARE_WORKERS_AI_API_KEY", "@cf/zai-org/glm-4.7-flash"),
		providerTemplateDefinition("cortecs", "Cortecs", TransportOpenAICompatible, "https://api.cortecs.ai/v1", "CORTECS_API_KEY", "minimax-m2.7"),
		providerTemplateDefinition("digitalocean", "DigitalOcean", TransportOpenAICompatible, "https://inference.do-ai.run/v1", "DIGITALOCEAN_TOKEN", "openai-gpt-4o-mini"),
		providerTemplateDefinition("dinference", "Dinference", TransportOpenAICompatible, "https://api.dinference.com/v1", "DINFERENCE_API_KEY", "gpt-oss-120b"),
		providerTemplateDefinition("drun", "D.run", TransportOpenAICompatible, "https://chat.d.run/v1", "DRUN_API_KEY", "public/deepseek-r1"),
		providerTemplateDefinition("evroc", "Evroc", TransportOpenAICompatible, "https://models.think.evroc.com/v1", "EVROC_API_KEY", "Qwen/Qwen3-VL-30B-A3B-Instruct"),
		providerTemplateDefinition("fastrouter", "FastRouter", TransportOpenAICompatible, "https://go.fastrouter.ai/api/v1", "FASTROUTER_API_KEY", "x-ai/grok-4"),
		providerTemplateDefinition("friendli", "Friendli", TransportOpenAICompatible, "https://api.friendli.ai/serverless/v1", "FRIENDLI_API_KEY", "Qwen/Qwen3-235B-A22B-Instruct-2507"),
		providerTemplateDefinition("github-models", "GitHub Models", TransportOpenAICompatible, "https://models.github.ai/inference", "GITHUB_MODELS_API_KEY", "github-models-retired"),
		providerTemplateDefinition("helicone", "Helicone", TransportOpenAICompatible, "https://ai-gateway.helicone.ai/v1", "HELICONE_API_KEY", "gpt-4o-mini"),
		providerTemplateDefinition("huggingface", "Hugging Face", TransportOpenAICompatible, "https://router.huggingface.co/v1", "HUGGINGFACE_API_KEY", "Qwen/Qwen3.5-397B-A17B"),
		providerTemplateDefinition("iflowcn", "iFlow CN", TransportOpenAICompatible, "https://apis.iflow.cn/v1", "IFLOWCN_API_KEY", "qwen3-coder-plus"),
		providerTemplateDefinition("inception", "Inception", TransportOpenAICompatible, "https://api.inceptionlabs.ai/v1", "INCEPTION_API_KEY", "mercury-edit-2"),
		providerTemplateDefinition("inference", "Inference.net", TransportOpenAICompatible, "https://api.inference.net/v1", "INFERENCE_API_KEY", "glm-5.2"),
		providerTemplateDefinition("io-net", "io.net", TransportOpenAICompatible, "https://api.intelligence.io.solutions/api/v1", "IO_NET_API_KEY", "Intel/Qwen3-Coder-480B-A35B-Instruct-int4-mixed-ar"),
		providerTemplateDefinition("jiekou", "Jiekou", TransportOpenAICompatible, "https://api.jiekou.ai/openai", "JIEKOU_API_KEY", "gpt-5.1-codex-max"),
		providerTemplateDefinition("kilo", "Kilo", TransportOpenAICompatible, "https://api.kilo.ai/api/gateway", "KILO_API_KEY", "rekaai/reka-edge"),
		providerTemplateDefinition("kimi-for-coding", "Kimi For Coding", TransportAnthropicMessages, "https://api.kimi.com/coding/v1", "KIMI_FOR_CODING_API_KEY", "k2p6"),
		providerTemplateDefinition("kuae-cloud-coding-plan", "Kuae Cloud Coding Plan", TransportOpenAICompatible, "https://coding-plan-endpoint.kuaecloud.net/v1", "KUAE_CLOUD_CODING_PLAN_API_KEY", "GLM-4.7"),
		providerTemplateDefinition("llama", "Llama", TransportOpenAICompatible, "https://api.llama.com/compat/v1", "LLAMA_API_KEY", "llama-3.3-70b-instruct"),
		providerTemplateDefinition("lmstudio", "LM Studio", TransportOpenAICompatible, "http://127.0.0.1:1234/v1", "LMSTUDIO_API_KEY", "openai/gpt-oss-20b"),
		providerTemplateDefinition("lucidquery", "LucidQuery", TransportOpenAICompatible, "https://lucidquery.com/api/v1", "LUCIDQUERY_API_KEY", "lucidnova-rf1-100b"),
		providerTemplateDefinition("meganova", "MegaNova", TransportOpenAICompatible, "https://api.meganova.ai/v1", "MEGANOVA_API_KEY", "Qwen/Qwen3-235B-A22B-Instruct-2507"),
		providerTemplateDefinition("minimax", "MiniMax", TransportAnthropicMessages, "https://api.minimax.io/anthropic/v1", "MINIMAX_API_KEY", "MiniMax-M2"),
		providerTemplateDefinition("minimax-cn", "MiniMax CN", TransportAnthropicMessages, "https://api.minimaxi.com/anthropic/v1", "MINIMAX_CN_API_KEY", "MiniMax-M2"),
		providerTemplateDefinition("minimax-cn-coding-plan", "MiniMax CN Coding Plan", TransportAnthropicMessages, "https://api.minimaxi.com/anthropic/v1", "MINIMAX_CN_CODING_PLAN_API_KEY", "MiniMax-M2"),
		providerTemplateDefinition("minimax-coding-plan", "MiniMax Coding Plan", TransportAnthropicMessages, "https://api.minimax.io/anthropic/v1", "MINIMAX_CODING_PLAN_API_KEY", "MiniMax-M2"),
		providerTemplateDefinition("moark", "Moark", TransportOpenAICompatible, "https://moark.com/v1", "MOARK_API_KEY", "GLM-4.7"),
		providerTemplateDefinition("modelscope", "ModelScope", TransportOpenAICompatible, "https://api-inference.modelscope.cn/v1", "MODELSCOPE_API_KEY", "Qwen/Qwen3-30B-A3B-Thinking-2507"),
		providerTemplateDefinition("moonshotai", "Moonshot AI", TransportOpenAICompatible, "https://api.moonshot.ai/v1", "MOONSHOTAI_API_KEY", "kimi-k2-0905-preview"),
		providerTemplateDefinition("moonshotai-cn", "Moonshot AI CN", TransportOpenAICompatible, "https://api.moonshot.cn/v1", "MOONSHOTAI_CN_API_KEY", "kimi-k2-thinking"),
		providerTemplateDefinition("morph", "Morph", TransportOpenAICompatible, "https://api.morphllm.com/v1", "MORPH_API_KEY", "auto"),
		providerTemplateDefinition("nano-gpt", "Nano GPT", TransportOpenAICompatible, "https://nano-gpt.com/api/v1", "NANO_GPT_API_KEY", "glm-4-flash"),
		providerTemplateDefinition("nebius", "Nebius", TransportOpenAICompatible, "https://api.tokenfactory.nebius.com/v1", "NEBIUS_API_KEY", "moonshotai/Kimi-K2.5"),
		providerTemplateDefinition("novita-ai", "Novita AI", TransportOpenAICompatible, "https://api.novita.ai/openai", "NOVITA_AI_API_KEY", "deepseek/deepseek-r1-turbo"),
		providerTemplateDefinition("nvidia", "NVIDIA", TransportOpenAICompatible, "https://integrate.api.nvidia.com/v1", "NVIDIA_API_KEY", "upstage/solar-10_7b-instruct"),
		providerTemplateDefinition("ollama-cloud", "Ollama Cloud", TransportOpenAICompatible, "https://ollama.com/v1", "OLLAMA_CLOUD_API_KEY", "minimax-m2.7"),
		providerTemplateDefinition("opencode", "opencode", TransportOpenAICompatible, "https://opencode.ai/zen/v1", "OPENCODE_API_KEY", "minimax-m2.7"),
		providerTemplateDefinition("opencode-go", "opencode Go", TransportOpenAICompatible, "https://opencode.ai/zen/go/v1", "OPENCODE_GO_API_KEY", "minimax-m2.7"),
		providerTemplateDefinition("ovhcloud", "OVHcloud", TransportOpenAICompatible, "https://oai.endpoints.kepler.ai.cloud.ovh.net/v1", "OVHCLOUD_API_KEY", "gpt-oss-20b"),
		providerTemplateDefinition("perplexity-agent", "Perplexity Agent", TransportOpenAIResponses, "https://api.perplexity.ai/v1", "PERPLEXITY_AGENT_API_KEY", "openai/gpt-5.6-terra"),
		providerTemplateDefinition("poe", "Poe", TransportOpenAICompatible, "https://api.poe.com/v1", "POE_API_KEY", "GPT-5.4"),
		providerTemplateDefinition("privatemode-ai", "PrivateMode AI", TransportOpenAICompatible, "http://localhost:8080/v1", "PRIVATEMODE_AI_API_KEY", "gemma-3-27b"),
		providerTemplateDefinition("qihang-ai", "Qihang AI", TransportOpenAICompatible, "https://api.qhaigc.net/v1", "QIHANG_AI_API_KEY", "claude-opus-4-5-20251101"),
		providerTemplateDefinition("qiniu-ai", "Qiniu AI", TransportOpenAICompatible, "https://api.qnaigc.com/v1", "QINIU_AI_API_KEY", "qwen3-235b-a22b"),
		providerTemplateDefinition("requesty", "Requesty", TransportOpenAICompatible, "https://router.requesty.ai/v1", "REQUESTY_API_KEY", "anthropic/claude-sonnet-4-20250514"),
		providerTemplateDefinition("scaleway", "Scaleway", TransportOpenAICompatible, "https://api.scaleway.ai/v1", "SCALEWAY_API_KEY", "qwen3.5-397b-a17b"),
		providerTemplateDefinition("siliconflow", "SiliconFlow", TransportOpenAICompatible, "https://api.siliconflow.com/v1", "SILICONFLOW_API_KEY", "nex-agi/DeepSeek-V3.1-Nex-N1"),
		providerTemplateDefinition("siliconflow-cn", "SiliconFlow CN", TransportOpenAICompatible, "https://api.siliconflow.cn/v1", "SILICONFLOW_CN_API_KEY", "Kwaipilot/KAT-Dev"),
		providerTemplateDefinition("stackit", "STACKIT", TransportOpenAICompatible, "https://api.openai-compat.model-serving.eu01.onstackit.cloud/v1", "STACKIT_API_KEY", "Qwen/Qwen3.6-27B"),
		providerTemplateDefinition("stepfun", "StepFun", TransportOpenAICompatible, "https://api.stepfun.com/v1", "STEPFUN_API_KEY", "step-3.5-flash-2603"),
		providerTemplateDefinition("submodel", "Submodel", TransportOpenAICompatible, "https://llm.submodel.ai/v1", "SUBMODEL_API_KEY", "Qwen/Qwen3-235B-A22B-Instruct-2507"),
		providerTemplateDefinition("synthetic", "Synthetic", TransportOpenAICompatible, "https://api.synthetic.new/openai/v1", "SYNTHETIC_API_KEY", "hf:meta-llama/Llama-3.1-405B-Instruct"),
		providerTemplateDefinition("tencent-coding-plan", "Tencent Coding Plan", TransportOpenAICompatible, "https://api.lkeap.cloud.tencent.com/coding/v3", "TENCENT_CODING_PLAN_API_KEY", "kimi-k2.6"),
		providerTemplateDefinition("upstage", "Upstage", TransportOpenAICompatible, "https://api.upstage.ai/v1/solar", "UPSTAGE_API_KEY", "solar-pro4"),
		providerTemplateDefinition("venice", "Venice", TransportOpenAICompatible, "https://api.venice.ai/api/v1", "VENICE_API_KEY", "zai-org-glm-5"),
		providerTemplateDefinition("vivgrid", "Vivgrid", TransportOpenAICompatible, "https://api.vivgrid.com/v1", "VIVGRID_API_KEY", "gpt-5.6-terra"),
		providerTemplateDefinition("vultr", "Vultr", TransportOpenAICompatible, "https://api.vultrinference.com/v1", "VULTR_API_KEY", "kimi-k2-instruct"),
		providerTemplateDefinition("wandb", "Weights & Biases", TransportOpenAICompatible, "https://api.inference.wandb.ai/v1", "WANDB_API_KEY", "openai/gpt-oss-20b"),
		providerTemplateDefinition("zai", "Z.ai", TransportOpenAICompatible, "https://api.z.ai/api/paas/v4", "ZAI_API_KEY", "glm-5v-turbo"),
		providerTemplateDefinition("zai-coding-plan", "Z.ai Coding Plan", TransportOpenAICompatible, "https://api.z.ai/api/coding/paas/v4", "ZAI_CODING_PLAN_API_KEY", "glm-4.7"),
		providerTemplateDefinition("zenmux", "ZenMux", TransportOpenAICompatible, "https://zenmux.ai/api/v1", "ZENMUX_API_KEY", "deepseek/deepseek-chat"),
		providerTemplateDefinition("zhipuai", "Zhipu AI", TransportOpenAICompatible, "https://open.bigmodel.cn/api/paas/v4", "ZHIPUAI_API_KEY", "glm-5v-turbo"),
		providerTemplateDefinition("zhipuai-coding-plan", "Zhipu AI Coding Plan", TransportOpenAICompatible, "https://open.bigmodel.cn/api/coding/paas/v4", "ZHIPUAI_CODING_PLAN_API_KEY", "glm-5v-turbo"),
	}
}

func providerTemplateDefinition(id, name string, transport TransportType, baseURL, apiKeyEnv, defaultModel string) ProviderDefinition {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	authTypes := []AuthType{AuthAPIKey}
	if isLoopbackProviderBaseURL(baseURL) {
		authTypes = []AuthType{AuthAPIKey, AuthNone}
	}
	definition := ProviderDefinition{
		ID: id, DisplayName: name, Description: "Preset provider template.",
		Transport: transport, AuthTypes: authTypes, DefaultAuthType: AuthAPIKey,
		DefaultBaseURL: baseURL, BaseURLEnvVar: providerTemplateEnvName(id, "BASE_URL"), APIKeyEnvVars: nonEmptyStrings(apiKeyEnv),
		ModelFetch: providerTemplateModelFetch(transport, baseURL), DefaultModelID: defaultModel, BuiltIn: true,
	}
	switch transport {
	case TransportAnthropicMessages:
		definition.RequestProfile = anthropicDefaultRequestProfile()
	case TransportGoogleGemini:
		definition.RequestProfile = googleDefaultRequestProfile()
	case TransportOpenAICompatible, TransportOpenAIResponses:
		definition.RequestProfile = domain.ProviderRequestProfile{Params: map[string]any{"stream": true}}
	}
	return definition
}

func providerTemplateModelFetch(transport TransportType, baseURL string) ModelFetchStrategy {
	if strings.HasSuffix(strings.ToLower(baseURL), "/chat/completions") {
		return ModelFetchStatic
	}
	switch transport {
	case TransportAnthropicMessages:
		return ModelFetchAnthropic
	case TransportGoogleGemini:
		return ModelFetchGoogle
	default:
		return ModelFetchOpenAICompatible
	}
}

func providerTemplateEnvName(providerID string, suffix string) string {
	prefix := strings.ToUpper(providerID)
	prefix = strings.NewReplacer("-", "_", ".", "_").Replace(prefix)
	if prefix == "302AI" {
		prefix = "AI_302"
	}
	return prefix + "_" + suffix
}

func isLoopbackProviderBaseURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	return isLoopbackHost(parsed.Hostname())
}
