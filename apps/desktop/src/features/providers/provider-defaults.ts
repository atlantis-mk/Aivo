export type ProviderProtocol =
  | "openai"
  | "openai-compatible"
  | "anthropic"
  | "google"
  | "openrouter";

export const providerProtocolDefaults: Record<string, ProviderProtocol> = {
  openai: "openai",
  "claude-code": "anthropic",
  anthropic: "anthropic",
  deepseek: "openai",
  google: "google",
  gemini: "google",
  groq: "openai",
  openrouter: "openrouter",
  xai: "openai",
  xiaomi: "openai",
  "volcengine-agent-plan": "openai",
  "kimi-for-coding": "anthropic",
  minimax: "anthropic",
  "minimax-cn": "anthropic",
  "minimax-coding-plan": "anthropic",
  "minimax-cn-coding-plan": "anthropic",
  "perplexity-agent": "openai",
};

export const providerBaseURLDefaults: Record<string, string> = {
  "302ai": "https://api.302.ai/v1",
  abacus: "https://routellm.abacus.ai/v1",
  aihubmix: "https://aihubmix.com/v1",
  alibaba: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1",
  "alibaba-cn": "https://dashscope.aliyuncs.com/compatible-mode/v1",
  "alibaba-coding-plan": "https://coding-intl.dashscope.aliyuncs.com/v1",
  "alibaba-coding-plan-cn": "https://coding.dashscope.aliyuncs.com/v1",
  anthropic: "https://api.anthropic.com/v1",
  baseten: "https://inference.baseten.co/v1",
  bailing: "https://api.tbox.cn/api/llm/v1/chat/completions",
  berget: "https://api.berget.ai/v1",
  cerebras: "https://api.cerebras.ai/v1",
  chutes: "https://llm.chutes.ai/v1",
  clarifai: "https://api.clarifai.com/v2/ext/openai/v1",
  "cloudferro-sherlock": "https://api-sherlock.cloudferro.com/openai/v1",
  "cloudflare-workers-ai":
    "https://api.cloudflare.com/client/v4/accounts/${CLOUDFLARE_ACCOUNT_ID}/ai/v1",
  cortecs: "https://api.cortecs.ai/v1",
  deepinfra: "https://api.deepinfra.com/v1/openai",
  deepseek: "https://api.deepseek.com",
  digitalocean: "https://inference.do-ai.run/v1",
  dinference: "https://api.dinference.com/v1",
  drun: "https://chat.d.run/v1",
  evroc: "https://models.think.evroc.com/v1",
  fastrouter: "https://go.fastrouter.ai/api/v1",
  "fireworks-ai": "https://api.fireworks.ai/inference/v1/",
  friendli: "https://api.friendli.ai/serverless/v1",
  "github-copilot": "https://api.githubcopilot.com",
  "github-models": "https://models.github.ai/inference",
  google: "https://generativelanguage.googleapis.com/v1beta",
  groq: "https://api.groq.com/openai/v1",
  helicone: "https://ai-gateway.helicone.ai/v1",
  huggingface: "https://router.huggingface.co/v1",
  iflowcn: "https://apis.iflow.cn/v1",
  inception: "https://api.inceptionlabs.ai/v1/",
  inference: "https://api.inference.net/v1",
  "io-net": "https://api.intelligence.io.solutions/api/v1",
  jiekou: "https://api.jiekou.ai/openai",
  kilo: "https://api.kilo.ai/api/gateway",
  "kimi-for-coding": "https://api.kimi.com/coding/v1",
  "kuae-cloud-coding-plan": "https://coding-plan-endpoint.kuaecloud.net/v1",
  llama: "https://api.llama.com/compat/v1/",
  lmstudio: "http://127.0.0.1:1234/v1",
  lucidquery: "https://lucidquery.com/api/v1",
  meganova: "https://api.meganova.ai/v1",
  minimax: "https://api.minimax.io/anthropic/v1",
  "minimax-cn": "https://api.minimaxi.com/anthropic/v1",
  "minimax-coding-plan": "https://api.minimax.io/anthropic/v1",
  "minimax-cn-coding-plan": "https://api.minimaxi.com/anthropic/v1",
  modelscope: "https://api-inference.modelscope.cn/v1",
  moark: "https://moark.com/v1",
  moonshotai: "https://api.moonshot.ai/v1",
  "moonshotai-cn": "https://api.moonshot.cn/v1",
  morph: "https://api.morphllm.com/v1",
  "nano-gpt": "https://nano-gpt.com/api/v1",
  nebius: "https://api.tokenfactory.nebius.com/v1",
  "novita-ai": "https://api.novita.ai/openai",
  nvidia: "https://integrate.api.nvidia.com/v1",
  "ollama-cloud": "https://ollama.com/v1",
  opencode: "https://opencode.ai/zen/v1",
  "opencode-go": "https://opencode.ai/zen/go/v1",
  openrouter: "https://openrouter.ai/api/v1",
  ovhcloud: "https://oai.endpoints.kepler.ai.cloud.ovh.net/v1",
  perplexity: "https://api.perplexity.ai",
  "perplexity-agent": "https://api.perplexity.ai/v1",
  poe: "https://api.poe.com/v1",
  "privatemode-ai": "http://localhost:8080/v1",
  "qihang-ai": "https://api.qhaigc.net/v1",
  "qiniu-ai": "https://api.qnaigc.com/v1",
  requesty: "https://router.requesty.ai/v1",
  scaleway: "https://api.scaleway.ai/v1",
  siliconflow: "https://api.siliconflow.com/v1",
  "siliconflow-cn": "https://api.siliconflow.cn/v1",
  stackit: "https://api.openai-compat.model-serving.eu01.onstackit.cloud/v1",
  stepfun: "https://api.stepfun.com/v1",
  submodel: "https://llm.submodel.ai/v1",
  synthetic: "https://api.synthetic.new/openai/v1",
  "tencent-coding-plan": "https://api.lkeap.cloud.tencent.com/coding/v3",
  togetherai: "https://api.together.ai/v1",
  upstage: "https://api.upstage.ai/v1/solar",
  venice: "https://api.venice.ai/api/v1",
  vivgrid: "https://api.vivgrid.com/v1",
  vultr: "https://api.vultrinference.com/v1",
  wandb: "https://api.inference.wandb.ai/v1",
  xai: "https://api.x.ai/v1",
  xiaomi: "https://api.xiaomimimo.com/v1",
  "volcengine-agent-plan": "https://ark.cn-beijing.volces.com/api/plan/v3",
  zai: "https://api.z.ai/api/paas/v4",
  "zai-coding-plan": "https://api.z.ai/api/coding/paas/v4",
  zenmux: "https://zenmux.ai/api/v1",
  zhipuai: "https://open.bigmodel.cn/api/paas/v4",
  "zhipuai-coding-plan": "https://open.bigmodel.cn/api/coding/paas/v4",
};

const providerModelDefaults: Record<string, string> = {
  openai: "gpt-5.5",
  anthropic: "claude-sonnet-4",
  "claude-code": "claude-sonnet-4",
  google: "gemini-2.5-pro",
  gemini: "gemini-2.5-pro",
  openrouter: "openai/gpt-5-codex",
  "302ai": "qwen3-235b-a22b",
  abacus: "gpt-5.1-codex-max",
  alibaba: "qwen3-235b-a22b",
  "alibaba-cn": "qwen3-235b-a22b",
  "alibaba-coding-plan": "qwen3-coder-plus",
  "alibaba-coding-plan-cn": "qwen3-coder-plus",
  bailing: "Ring-1T",
  baseten: "custom-profile",
  berget: "zai-org/GLM-4.7",
  cerebras: "zai-glm-4.7",
  chutes: "NousResearch/DeepHermes-3-Mistral-24B-Preview",
  clarifai: "arcee_ai/AFM/models/trinity-mini",
  "cloudferro-sherlock": "meta-llama/Llama-3.3-70B-Instruct",
  "cloudflare-workers-ai": "@cf/zai-org/glm-4.7-flash",
  cortecs: "minimax-m2.7",
  deepinfra: "Qwen/Qwen3-Coder-480B-A35B-Instruct-Turbo",
  deepseek: "deepseek-chat",
  digitalocean: "openai-gpt-4o-mini",
  dinference: "gpt-oss-120b",
  drun: "public/deepseek-r1",
  evroc: "Qwen/Qwen3-VL-30B-A3B-Instruct",
  fastrouter: "x-ai/grok-4",
  "fireworks-ai": "accounts/fireworks/models/glm-5p2",
  friendli: "Qwen/Qwen3-235B-A22B-Instruct-2507",
  "github-copilot": "gpt-5.1-codex-max",
  "github-models": "github-models-retired",
  groq: "openai/gpt-oss-120b",
  helicone: "gpt-4o-mini",
  huggingface: "Qwen/Qwen3.5-397B-A17B",
  iflowcn: "qwen3-coder-plus",
  inception: "mercury-edit-2",
  inference: "glm-5.2",
  "io-net": "Intel/Qwen3-Coder-480B-A35B-Instruct-int4-mixed-ar",
  jiekou: "gpt-5.1-codex-max",
  kilo: "rekaai/reka-edge",
  "kimi-for-coding": "k2p6",
  "kuae-cloud-coding-plan": "GLM-4.7",
  llama: "llama-3.3-70b-instruct",
  lmstudio: "openai/gpt-oss-20b",
  lucidquery: "lucidnova-rf1-100b",
  meganova: "Qwen/Qwen3-235B-A22B-Instruct-2507",
  minimax: "MiniMax-M2",
  "minimax-cn": "MiniMax-M2",
  "minimax-coding-plan": "MiniMax-M2",
  "minimax-cn-coding-plan": "MiniMax-M2",
  moark: "GLM-4.7",
  modelscope: "Qwen/Qwen3-30B-A3B-Thinking-2507",
  moonshotai: "kimi-k2-0905-preview",
  "moonshotai-cn": "kimi-k2-thinking",
  morph: "auto",
  "nano-gpt": "glm-4-flash",
  nebius: "moonshotai/Kimi-K2.5",
  nova: "nova-2-lite-v1",
  "novita-ai": "deepseek/deepseek-r1-turbo",
  nvidia: "upstage/solar-10_7b-instruct",
  "ollama-cloud": "minimax-m2.7",
  opencode: "minimax-m2.7",
  "opencode-go": "minimax-m2.7",
  ovhcloud: "gpt-oss-20b",
  perplexity: "sonar-pro",
  "perplexity-agent": "openai/gpt-5.6-terra",
  poe: "GPT-5.4",
  "privatemode-ai": "gemma-3-27b",
  "qihang-ai": "claude-opus-4-5-20251101",
  "qiniu-ai": "qwen3-235b-a22b",
  requesty: "anthropic/claude-sonnet-4-20250514",
  scaleway: "qwen3.5-397b-a17b",
  siliconflow: "nex-agi/DeepSeek-V3.1-Nex-N1",
  "siliconflow-cn": "Kwaipilot/KAT-Dev",
  stackit: "Qwen/Qwen3.6-27B",
  stepfun: "step-3.5-flash-2603",
  submodel: "Qwen/Qwen3-235B-A22B-Instruct-2507",
  synthetic: "hf:meta-llama/Llama-3.1-405B-Instruct",
  "tencent-coding-plan": "kimi-k2.6",
  togetherai: "MiniMaxAI/MiniMax-M3",
  upstage: "solar-pro4",
  venice: "zai-org-glm-5",
  vivgrid: "gpt-5.6-terra",
  vultr: "kimi-k2-instruct",
  wandb: "openai/gpt-oss-20b",
  xai: "grok-4.3",
  xiaomi: "mimo-v2.5-pro",
  "volcengine-agent-plan": "ark-code-latest",
  zai: "glm-5v-turbo",
  "zai-coding-plan": "glm-4.7",
  zenmux: "deepseek/deepseek-chat",
  zhipuai: "glm-5v-turbo",
  "zhipuai-coding-plan": "glm-5v-turbo",
};

const providerDisplayNames: Record<string, string> = {
  "302ai": "302.AI",
  aihubmix: "AIHubMix",
  "alibaba-cn": "Alibaba CN",
  "alibaba-coding-plan-cn": "Alibaba Coding Plan CN",
  "alibaba-coding-plan": "Alibaba Coding Plan",
  alibaba: "Alibaba",
  anthropic: "Anthropic",
  "amazon-bedrock": "Amazon Bedrock",
  "azure-cognitive-services": "Azure Cognitive Services",
  "claude-code": "Claude Code",
  "cloudflare-ai-gateway": "Cloudflare AI Gateway",
  "cloudflare-workers-ai": "Cloudflare Workers AI",
  "fireworks-ai": "Fireworks AI",
  gemini: "Gemini",
  "github-copilot": "GitHub Copilot",
  "github-models": "GitHub Models",
  "google-vertex-anthropic": "Google Vertex Anthropic",
  "google-vertex": "Google Vertex",
  google: "Google",
  "io-net": "io.net",
  lmstudio: "LM Studio",
  "minimax-cn-coding-plan": "MiniMax CN Coding Plan",
  "minimax-cn": "MiniMax CN",
  "minimax-coding-plan": "MiniMax Coding Plan",
  minimax: "MiniMax",
  "moonshotai-cn": "Moonshot AI CN",
  moonshotai: "Moonshot AI",
  "nano-gpt": "Nano GPT",
  "novita-ai": "Novita AI",
  "ollama-cloud": "Ollama Cloud",
  "opencode-go": "opencode Go",
  opencode: "opencode",
  openai: "OpenAI",
  openrouter: "OpenRouter",
  "perplexity-agent": "Perplexity Agent",
  qwen: "Qwen",
  "qwen-cn": "Qwen CN",
  "qiniu-ai": "Qiniu AI",
  "siliconflow-cn": "SiliconFlow CN",
  siliconflow: "SiliconFlow",
  stackit: "STACKIT",
  synthetic: "Synthetic",
  "tencent-coding-plan": "Tencent Coding Plan",
  "volcengine-agent-plan": "火山方舟 Agent Plan",
  togetherai: "Together AI",
  v0: "v0",
  wandb: "Weights & Biases",
};

export function providerDisplayName(providerId: string) {
  if (providerDisplayNames[providerId]) return providerDisplayNames[providerId];
  return providerId
    .split("-")
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

export function providerNameForPrompt(providerId: string) {
  if (providerId === "claude-code") return "Anthropic";
  if (providerId === "gemini") return "Google";
  if (providerId === "custom-api") return "Custom API";
  return providerDisplayName(providerId);
}

export function credentialReferenceFor(providerId: string) {
  if (providerId === "openai") return "OPENAI_API_KEY";
  if (providerId === "anthropic") return "ANTHROPIC_API_KEY";
  if (providerId === "claude-code") return "ANTHROPIC_API_KEY";
  if (providerId === "google") return "GEMINI_API_KEY";
  if (providerId === "gemini") return "GEMINI_API_KEY";
  if (providerId === "openrouter") return "OPENROUTER_API_KEY";
  if (providerId === "volcengine-agent-plan") return "ARK_API_KEY";
  return undefined;
}

export function knownDefaultModelForProvider(providerId: string) {
  return providerModelDefaults[providerId];
}

export function primaryDefaultModelForProvider(providerId: string) {
  if (providerId === "openai") return "gpt-5.5";
  if (providerId === "anthropic" || providerId === "claude-code") {
    return "claude-sonnet-4";
  }
  if (providerId === "google" || providerId === "gemini") {
    return "gemini-2.5-pro";
  }
  if (providerId === "openrouter") return "openai/gpt-5-codex";
  return "";
}

export function defaultBaseURLForProvider(providerId: string) {
  return providerBaseURLDefaults[providerId];
}

export function providerProtocolForProvider(providerId: string) {
  return providerProtocolDefaults[providerId] ?? "openai-compatible";
}

export function protocolForProvider(providerId: string) {
  return providerProtocolForProvider(providerId);
}

export function providerTypeFor(providerId: string) {
  return providerProtocolForProvider(providerId);
}
