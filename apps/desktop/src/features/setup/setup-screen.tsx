import { useCallback, useEffect, useRef, useState } from "react";
import { Navigate } from "@tanstack/react-router";
import { X } from "lucide-react";
import { toast } from "sonner";
import { BrowserOpenURL, EventsOn } from "../../../bridge/runtime/runtime";

import type { domain } from "../../../bridge/go/models";
import anthropicIcon from "@/assets/icons/provider/anthropic.svg";
import googleIcon from "@/assets/icons/provider/google.svg";
import openAIIcon from "@/assets/icons/provider/openai.svg";
import syntheticIcon from "@/assets/icons/provider/synthetic.svg";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
 Dialog,
 DialogClose,
 DialogContent,
 DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
 Select,
 SelectContent,
 SelectItem,
 SelectTrigger,
 SelectValue,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";
import { useAppConfig, hasAppBridge } from "@/lib/app-config";
import {
 completePreviewOpenAIBrowserAuth,
 connectPreviewProvider,
 deletePreviewProviderAccount,
 setPreviewInitialized,
 startPreviewOpenAIBrowserAuth,
} from "@/lib/preview-state";
import type { CatalogState, ModelInfo, ProviderAccountInfo, ProviderConnectInput, ProviderInfo } from "@/lib/provider-catalog";
import { SetupLoadingSkeleton } from "@/features/setup/setup-loading-skeleton";
import {
 ProviderConnectionDialogs,
 ProviderIcon,
 type CustomProviderForm,
 type CustomProviderProtocol,
 type CustomProviderRow,
 type ProviderAuthMode,
 type ProviderChoice,
 type ProviderDialogStep,
} from "@/features/setup/provider-connection-dialogs";
import {
 connectProvider,
 deleteProviderAccount,
 refreshProviderModels,
 startProviderAuth,
 updateModelPreferences,
} from "@/services/aivo";

const capabilityPills = [
 "我能帮你整理文件",
 "我能帮你浏览总结",
 "我可以帮你使用电脑",
 "我能帮你操作 App",
 "我能搜集全网信息",
];

type SetupStep = "welcome" | "provider" | "project";
type ProviderValidationResult = {
 completed: boolean;
 start?: domain.ProviderAuthStartResult;
};
type AppConfigWithAuxiliary = domain.AppConfig & {
 auxiliaryModel?: domain.ModelRef;
};

const providerIconModules = import.meta.glob<string>("@/assets/icons/provider/*.svg", {
 eager: true,
 import: "default",
 query: "?url",
});

const primaryProviderIds = new Set(["openai", "anthropic", "google", "synthetic"]);

const providerChoices: ProviderChoice[] = [
 {
 id: "openai",
 name: "OpenAI",
 iconSrc: openAIIcon,
 },
 {
 id: "claude-code",
 name: "Claude Code",
 iconSrc: anthropicIcon,
 },
 {
 id: "gemini",
 name: "Gemini",
 iconSrc: googleIcon,
 },
 {
 id: "other",
 name: "其他",
 iconSrc: syntheticIcon,
 opensProviderPicker: true,
 },
 {
 id: "custom-api",
 name: "Custom API",
 iconSrc: syntheticIcon,
 },
];

const otherProviderChoices: ProviderChoice[] = Object.entries(providerIconModules)
 .map(([path, iconSrc]) => {
 const id = path.split("/").pop()?.replace(/\.svg$/, "") ?? "";
 return {
 id,
 name: providerDisplayName(id),
 iconSrc,
 custom: true,
 };
 })
 .filter((provider) => provider.id && !primaryProviderIds.has(provider.id))
 .sort((first, second) => first.name.localeCompare(second.name));

const providerProtocolDefaults: Record<string, CustomProviderProtocol> = {
 openai: "openai",
 "claude-code": "anthropic",
 anthropic: "anthropic",
 google: "google",
 gemini: "google",
 openrouter: "openrouter",
 "kimi-for-coding": "anthropic",
 minimax: "anthropic",
 "minimax-cn": "anthropic",
 "minimax-coding-plan": "anthropic",
 "minimax-cn-coding-plan": "anthropic",
 "perplexity-agent": "openai",
 vivgrid: "openai",
};

const providerBaseURLDefaults: Record<string, string> = {
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
 "cloudferro-sherlock": "https://api-sherlock.cloudferro.com/openai/v1/",
 "cloudflare-workers-ai": "https://api.cloudflare.com/client/v4/accounts/${CLOUDFLARE_ACCOUNT_ID}/ai/v1",
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
 inference: "https://inference.net/v1",
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
 togetherai: "https://api.together.xyz/v1",
 upstage: "https://api.upstage.ai/v1/solar",
 venice: "https://api.venice.ai/api/v1",
 vivgrid: "https://api.vivgrid.com/v1",
 vultr: "https://api.vultrinference.com/v1",
 wandb: "https://api.inference.wandb.ai/v1",
 xai: "https://api.x.ai/v1",
 xiaomi: "https://api.xiaomimimo.com/v1",
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
 baseten: "zai-org/GLM-4.7",
 berget: "zai-org/GLM-4.7",
 cerebras: "llama3.1-8b",
 chutes: "NousResearch/DeepHermes-3-Mistral-24B-Preview",
 clarifai: "arcee_ai/AFM/models/trinity-mini",
 "cloudferro-sherlock": "meta-llama/Llama-3.3-70B-Instruct",
 "cloudflare-workers-ai": "@cf/zai-org/glm-4.7-flash",
 cortecs: "minimax-m2.7",
 deepinfra: "Qwen/Qwen3.5-397B-A17B",
 deepseek: "deepseek-chat",
 digitalocean: "openai-gpt-4o-mini",
 dinference: "gpt-oss-120b",
 drun: "public/deepseek-r1",
 evroc: "Qwen/Qwen3-VL-30B-A3B-Instruct",
 fastrouter: "x-ai/grok-4",
 "fireworks-ai": "accounts/fireworks/models/glm-5p1",
 friendli: "Qwen/Qwen3-235B-A22B-Instruct-2507",
 "github-copilot": "gpt-5.1-codex-max",
 "github-models": "deepseek/deepseek-v3-0324",
 groq: "gemma2-9b-it",
 helicone: "mistral-nemo",
 huggingface: "Qwen/Qwen3.5-397B-A17B",
 iflowcn: "qwen3-coder-plus",
 inception: "mercury-edit-2",
 inference: "mistral/mistral-nemo-12b-instruct",
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
 nebius: "NousResearch/Hermes-4-70B",
 nova: "nova-2-lite-v1",
 "novita-ai": "deepseek/deepseek-r1-turbo",
 nvidia: "upstage/solar-10_7b-instruct",
 "ollama-cloud": "minimax-m2.7",
 opencode: "minimax-m2.7",
 "opencode-go": "minimax-m2.7",
 ovhcloud: "meta-llama-3_3-70b-instruct",
 perplexity: "sonar-pro",
 "perplexity-agent": "perplexity/sonar",
 poe: "topazlabs-co/topazlabs",
 "privatemode-ai": "gemma-3-27b",
 "qihang-ai": "claude-opus-4-5-20251101",
 "qiniu-ai": "qwen3-235b-a22b",
 requesty: "xai/grok-4-fast",
 scaleway: "qwen3-embedding-8b",
 siliconflow: "nex-agi/DeepSeek-V3.1-Nex-N1",
 "siliconflow-cn": "Kwaipilot/KAT-Dev",
 stackit: "Qwen/Qwen3-VL-Embedding-8B",
 stepfun: "step-3.5-flash-2603",
 submodel: "Qwen/Qwen3-235B-A22B-Instruct-2507",
 synthetic: "hf:meta-llama/Llama-3.1-405B-Instruct",
 "tencent-coding-plan": "kimi-k2.5",
 togetherai: "essentialai/Rnj-1-Instruct",
 upstage: "solar-pro2",
 vivgrid: "gpt-5.1-codex-max",
 vultr: "MiniMax-M2.5",
 wandb: "Qwen/Qwen3-30B-A3B-Instruct-2507",
 xai: "grok-2-1212",
 xiaomi: "mimo-v2.5-pro",
 zai: "glm-5v-turbo",
 "zai-coding-plan": "glm-4.7",
 zenmux: "deepseek/deepseek-chat",
 zhipuai: "glm-5v-turbo",
 "zhipuai-coding-plan": "glm-5v-turbo",
};

export function SetupScreen() {
 const { catalog, config, loading, setCatalog, setConfig, setError, error } = useAppConfig();
 const [saving, setSaving] = useState(false);
 const [step, setStep] = useState<SetupStep>("welcome");
 const [providerValidated, setProviderValidated] = useState(false);

 if (loading) return <SetupLoadingSkeleton />;

 async function completeProviderDialog(
 provider: ProviderChoice,
 authMode: ProviderAuthMode,
 apiKey?: string,
 customProvider?: CustomProviderForm,
 selectedModelId?: string,
 selectedAuxiliaryModelId?: string,
 ) {
 setSaving(true);
 setError("");
 const isCustomProvider = isCustomProviderChoice(provider) && customProvider;
 const customModel = customProvider?.models.find((model) => model.name.trim());
 const input: ProviderConnectInput = {
 providerId: isCustomProvider ? customProvider.providerId.trim() : provider.id,
 name: isCustomProvider ? customProvider.displayName.trim() : provider.name,
 type: isCustomProvider ? customProvider.protocol : providerTypeFor(provider.id),
 baseUrl: isCustomProvider ? customProvider.baseUrl.trim() : defaultBaseURLForProvider(provider.id),
 apiKeyEnv: isCustomProvider || (provider.id === "openai" && authMode !== "api-key") ? undefined : credentialReferenceFor(provider.id),
 apiKey: isCustomProvider ? customProvider.apiKey.trim() : authMode === "api-key" ? apiKey?.trim() : undefined,
 modelId: isCustomProvider ? customModel?.name.trim() : selectedModelId?.trim() || defaultModelForProvider(provider.id, catalog),
 method: authMode,
 headers: isCustomProvider ? headersFromRows(customProvider.headers) : undefined,
 };
 const auxiliaryModelId = selectedAuxiliaryModelId?.trim() || input.modelId;
 const auxiliaryModel =
 auxiliaryModelId && input.providerId
 ? ({ providerId: input.providerId, modelId: auxiliaryModelId } as domain.ModelRef)
 : undefined;
 try {
 if (hasAppBridge()) {
 try {
 const refreshedCatalog = await refreshProviderModels(input as unknown as domain.ProviderConnectInput);
 setCatalog(refreshedCatalog);
 if (!isCustomProvider) {
 input.modelId = catalogDefaultModelForProvider(refreshedCatalog, input.providerId) || input.modelId;
 }
 } catch {
 // Refresh is opportunistic; connecting with the configured/default model remains valid.
 }
 const nextCatalog = await connectProvider(input as unknown as domain.ProviderConnectInput);
 setCatalog(nextCatalog);
 const nextConfig = {
 initialized: true,
 provider: {
 id: input.providerId,
 type: input.type,
 baseUrl: input.baseUrl,
 apiKeyEnv: input.apiKeyEnv,
 headers: input.headers,
 model: input.modelId,
 },
 defaultModel: { providerId: input.providerId, modelId: input.modelId },
 auxiliaryModel,
 } as AppConfigWithAuxiliary;
 if (auxiliaryModel) {
 const savedConfig = await updateModelPreferences({ auxiliaryModel } as domain.ModelPreferencesInput & {
 auxiliaryModel: domain.ModelRef;
 });
 setConfig(savedConfig as AppConfigWithAuxiliary);
 } else {
 setConfig(nextConfig);
 }
 } else {
 const next = connectPreviewProvider(input);
 (next.config as AppConfigWithAuxiliary).auxiliaryModel = auxiliaryModel;
 setPreviewInitialized(next.config);
 setCatalog(next.catalog);
 setConfig(next.config);
 }
 return true;
 } catch (err) {
 setError(err instanceof Error ? err.message : String(err));
 return false;
 } finally {
 setSaving(false);
 }
 }

 async function validateProvider(
 provider: ProviderChoice,
 authMode: ProviderAuthMode,
 callbackInput?: string,
 apiKey?: string,
 ): Promise<ProviderValidationResult> {
 setError("");
 if (provider.id === "openai" && (authMode === "oauth-browser" || authMode === "oauth-headless")) {
 try {
 if (hasAppBridge()) {
 const pending = await startProviderAuth({ providerId: "openai", method: authMode });
 if (pending.url) await openExternalURL(pending.url);
 return { completed: false, start: pending };
 }
 if (authMode === "oauth-browser") {
 if (!callbackInput?.trim()) {
 const pending = await startPreviewOpenAIBrowserAuth();
 if (pending.authUrl) await openExternalURL(pending.authUrl);
 return {
 completed: false,
 start: {
 providerId: "openai",
 method: "oauth-browser",
 status: pending.status,
 url: pending.authUrl,
 instructions: pending.instructions,
 expiresAt: pending.expiresAt,
 } as domain.ProviderAuthStartResult,
 };
 }
 const next = await completePreviewOpenAIBrowserAuth(callbackInput);
 setCatalog(next.catalog);
 setConfig(next.config);
 setProviderValidated(true);
 return { completed: true };
 }
 throw new Error("Headless OAuth requires the Aivo desktop bridge.");
 } catch (err) {
 setError(err instanceof Error ? err.message : String(err));
 return { completed: false };
 }
 }

 if (authMode === "api-key" && !apiKey?.trim()) {
 setError("请输入 API Key。");
 return { completed: false };
 }
 setProviderValidated(true);
 return { completed: true };
 }

 async function removeProviderAccount(accountId: string) {
 setError("");
 try {
 if (hasAppBridge()) {
 setCatalog(await deleteProviderAccount(accountId));
 } else {
 setCatalog(deletePreviewProviderAccount(accountId));
 }
 } catch (err) {
 setError(err instanceof Error ? err.message : String(err));
 }
 }

 async function saveConnectedAccountModels(providerId: string, modelId: string, auxiliaryModelId: string) {
 const model = { providerId, modelId } as domain.ModelRef;
 const auxiliaryModel = { providerId, modelId: auxiliaryModelId } as domain.ModelRef;
 try {
 if (hasAppBridge()) {
 const savedConfig = await updateModelPreferences({ model, auxiliaryModel } as domain.ModelPreferencesInput & {
 auxiliaryModel: domain.ModelRef;
 });
 setConfig(savedConfig as AppConfigWithAuxiliary);
 } else {
 const nextConfig = {
 ...(config ?? {}),
 initialized: true,
 defaultModel: model,
 auxiliaryModel,
 provider: {
 ...((config?.provider ?? {}) as domain.ProviderConfig),
 id: providerId,
 type: config?.provider?.type || providerTypeFor(providerId),
 baseUrl: config?.provider?.baseUrl || defaultBaseURLForProvider(providerId),
 model: modelId,
 },
 } as AppConfigWithAuxiliary;
 setPreviewInitialized(nextConfig);
 setConfig(nextConfig);
 }
 setCatalog(updateCatalogDefaultModel(catalog, providerId, modelId));
 setError("");
 } catch (err) {
 setError(err instanceof Error ? err.message : String(err));
 }
 }

 if (step === "project") {
 return <Navigate to="/projects/chat" replace />;
 }

 return (
 <main className="min-h-dvh overflow-x-hidden bg-background text-foreground">
 <div className="window-title-drag-region" />
 {step === "welcome" ? (
 <WelcomeStep onNext={() => setStep("provider")} />
 ) : step === "provider" ? (
 <ProviderStep
 config={config as AppConfigWithAuxiliary | null}
 error={error}
 onContinue={completeProviderDialog}
 onNextPage={() => setStep("project")}
 onResetValidation={() => setProviderValidated(false)}
 onValidate={validateProvider}
 catalog={catalog}
 connectedAccounts={catalog?.providers.flatMap((provider) => provider.accounts ?? []) ?? []}
 onRemoveAccount={removeProviderAccount}
 onSaveAccountModels={saveConnectedAccountModels}
 onRefreshModels={async (input) => {
 if (!hasAppBridge()) return catalog;
 try {
 const nextCatalog = await refreshProviderModels(input as unknown as domain.ProviderConnectInput);
 setCatalog(nextCatalog);
 setError("");
 return nextCatalog;
 } catch {
 return catalog;
 }
 }}
 providerValidated={providerValidated}
 saving={saving}
 />
 ) : null}
 </main>
 );
}

function WelcomeStep({ onNext }: { onNext: () => void }) {
 return (
 <section className="flex min-h-dvh items-center justify-center bg-background px-5 py-16">
 <div className="flex w-full max-w-[1100px] flex-col items-center text-center">
 <h1 className="text-3xl font-extrabold leading-9 tracking-normal text-foreground sm:text-4xl sm:leading-10">
 你好，我是 Aivo
 </h1>
 <p className="mt-6 text-xl leading-7 text-foreground">
 为你 24 小时随时在线
 </p>

 <div className="mt-12 flex flex-wrap items-center justify-center gap-x-5 gap-y-3 sm:mt-16 min-[1180px]:flex-nowrap">
 {capabilityPills.map((pill) => (
 <Badge key={pill} className="h-9 px-5 text-sm" variant="secondary">
 {pill}
 </Badge>
 ))}
 </div>

 <Button
 className="mt-16 h-12 rounded-full px-8 text-base sm:mt-24"
 onClick={onNext}
 size="lg"
 >
 下一步
 </Button>
 </div>
 </section>
 );
}

function ProviderStep({
 catalog,
 config,
 error,
 onContinue,
 onNextPage,
 onResetValidation,
 onRefreshModels,
 onValidate,
 connectedAccounts,
 onRemoveAccount,
 onSaveAccountModels,
 providerValidated,
 saving,
}: {
 catalog: CatalogState | null;
 config: AppConfigWithAuxiliary | null;
 error: string;
 onContinue: (
 provider: ProviderChoice,
 authMode: ProviderAuthMode,
 apiKey?: string,
 customProvider?: CustomProviderForm,
 selectedModelId?: string,
 selectedAuxiliaryModelId?: string,
 ) => Promise<boolean>;
 onNextPage: () => void;
 onResetValidation: () => void;
 onRefreshModels: (input: ProviderConnectInput) => Promise<CatalogState | null>;
 onValidate: (
 provider: ProviderChoice,
 authMode: ProviderAuthMode,
 callbackInput?: string,
 apiKey?: string,
 ) => Promise<ProviderValidationResult>;
 connectedAccounts: ProviderAccountInfo[];
 onRemoveAccount: (accountId: string) => Promise<void>;
 onSaveAccountModels: (providerId: string, modelId: string, auxiliaryModelId: string) => Promise<void>;
 providerValidated: boolean;
 saving: boolean;
}) {
 const [activeProvider, setActiveProvider] = useState<ProviderChoice | null>(null);
 const [providerDialogStep, setProviderDialogStep] = useState<ProviderDialogStep>("details");
 const [authMode, setAuthMode] = useState<ProviderAuthMode>("api-key");
 const [oauthStarted, setOauthStarted] = useState(false);
 const [oauthStartResult, setOauthStartResult] = useState<domain.ProviderAuthStartResult | null>(null);
 const [oauthStatus, setOauthStatus] = useState<domain.ProviderAuthStatus | null>(null);
 const [callbackInput, setCallbackInput] = useState("");
 const [apiKey, setApiKey] = useState("");
 const [customProviderForm, setCustomProviderForm] = useState<CustomProviderForm>(() => emptyCustomProviderForm());
 const [otherProviderPickerOpen, setOtherProviderPickerOpen] = useState(false);
 const [otherProviderSearch, setOtherProviderSearch] = useState("");
 const [selectedModelId, setSelectedModelId] = useState("");
 const [selectedAuxiliaryModelId, setSelectedAuxiliaryModelId] = useState("");
 const [settingsAccount, setSettingsAccount] = useState<ProviderAccountInfo | null>(null);
 const [authSuccessMessage, setAuthSuccessMessage] = useState("");
 const authSuccessNotifiedRef = useRef(false);

 function openProvider(provider: ProviderChoice) {
 if (provider.opensProviderPicker) {
 setOtherProviderSearch("");
 setOtherProviderPickerOpen(true);
 return;
 }
 onResetValidation();
 setOauthStarted(false);
 setOauthStartResult(null);
 setOauthStatus(null);
 setAuthSuccessMessage("");
 authSuccessNotifiedRef.current = false;
 setCallbackInput("");
 setApiKey("");
 const nextForm = customProviderFormFor(provider);
 const nextAuthMode = provider.id === "openai" ? "oauth-browser" : "api-key";
 setCustomProviderForm(nextForm);
 setAuthMode(nextAuthMode);
 const nextModelId = catalogDefaultModelForProvider(catalog, provider.id) || defaultModelForProvider(provider.id, catalog);
 setSelectedModelId(nextModelId);
 setSelectedAuxiliaryModelId(defaultAuxiliaryModelForProvider(catalog, provider.id, nextModelId));
 setProviderDialogStep(provider.id === "openai" ? "options" : "details");
 setActiveProvider(provider);
 void refreshModelsForProvider(provider, nextForm, nextAuthMode, "");
 }

 const closeProvider = useCallback(() => {
 onResetValidation();
 setOauthStarted(false);
 setOauthStartResult(null);
 setOauthStatus(null);
 setAuthSuccessMessage("");
 authSuccessNotifiedRef.current = false;
 setCallbackInput("");
 setApiKey("");
 setCustomProviderForm(emptyCustomProviderForm());
 setSelectedModelId("");
 setSelectedAuxiliaryModelId("");
 setProviderDialogStep("details");
 setActiveProvider(null);
 }, [onResetValidation]);

 function selectOtherProvider(provider: ProviderChoice) {
 setOtherProviderPickerOpen(false);
 openProvider(provider);
 }

 function selectOpenAIAuthMode(nextMode: ProviderAuthMode) {
 onResetValidation();
 setOauthStarted(false);
 setOauthStartResult(null);
 setOauthStatus(null);
 setAuthSuccessMessage("");
 authSuccessNotifiedRef.current = false;
 setCallbackInput("");
 setApiKey("");
 setAuthMode(nextMode);
 setProviderDialogStep("details");
 if (activeProvider) void refreshModelsForProvider(activeProvider, customProviderForm, nextMode, "");
 }

 function resetAuthMode(nextMode: ProviderAuthMode) {
 onResetValidation();
 setOauthStarted(false);
 setOauthStartResult(null);
 setOauthStatus(null);
 setAuthSuccessMessage("");
 authSuccessNotifiedRef.current = false;
 setCallbackInput("");
 setApiKey("");
 setAuthMode(nextMode);
 if (activeProvider) void refreshModelsForProvider(activeProvider, customProviderForm, nextMode, "");
 }

 const refreshModelsForProvider = useCallback(async (
 provider: ProviderChoice,
 form: CustomProviderForm,
 mode: ProviderAuthMode,
 key: string,
 force = false,
 ) => {
 if (!hasAppBridge()) return;
 if (!force && !canRefreshProviderModels(provider, form, mode, key, catalog)) return;
 const input = providerRefreshInput(provider, form, mode, key, selectedModelId);
 const nextCatalog = await onRefreshModels(input);
 const refreshedDefault = catalogDefaultModelForProvider(nextCatalog, input.providerId);
 if (refreshedDefault) {
 setSelectedModelId(refreshedDefault);
 }
 setSelectedAuxiliaryModelId(defaultAuxiliaryModelForProvider(nextCatalog, input.providerId, refreshedDefault || input.modelId || ""));
 }, [catalog, onRefreshModels, selectedModelId]);

 const markOpenAIAuthorized = useCallback((provider: ProviderChoice) => {
 setAuthSuccessMessage("OpenAI 授权已完成");
 setOauthStatus((current) => ({
 providerId: "openai",
 method: current?.method || authMode,
 status: "success",
 accountId: current?.accountId,
 instructions: current?.instructions,
 userCode: current?.userCode,
 } as domain.ProviderAuthStatus));
 if (!authSuccessNotifiedRef.current) {
 authSuccessNotifiedRef.current = true;
 toast.success("OpenAI 授权已完成");
 }
 void refreshModelsForProvider(provider, customProviderForm, authMode, apiKey, true);
 }, [apiKey, authMode, customProviderForm, refreshModelsForProvider]);

 async function validateActiveProvider() {
 if (!activeProvider) return { completed: false } satisfies ProviderValidationResult;
 const result = await onValidate(activeProvider, authMode, oauthStarted ? callbackInput : undefined, apiKey);
 if (result.start) {
 setOauthStartResult(result.start);
 setOauthStatus({
 providerId: result.start.providerId,
 method: result.start.method,
 status: result.start.status,
 instructions: result.start.instructions,
 userCode: result.start.userCode,
 } as domain.ProviderAuthStatus);
 }
 if (activeProvider.id === "openai" && (authMode === "oauth-browser" || authMode === "oauth-headless") && !result.completed) {
 setOauthStarted(true);
 }
 return result;
 }

 async function completeActiveProvider() {
 if (!activeProvider) return;
 if (isCustomProviderChoice(activeProvider) && !customProviderFormIsValid(customProviderForm)) {
 return;
 }
 const completed = await onContinue(
 activeProvider,
 authMode,
 apiKey,
 isCustomProviderChoice(activeProvider) ? customProviderForm : undefined,
 effectiveSelectedModelId(),
 effectiveSelectedAuxiliaryModelId(),
 );
 if (completed) closeProvider();
 }

 function effectiveSelectedModelId() {
 if (!activeProvider) return selectedModelId;
 if (selectedModelId && activeProviderModels.some((model) => model.id === selectedModelId)) return selectedModelId;
 return catalogDefaultModelForProvider(catalog, activeProvider.id) || activeProviderModels[0]?.id || selectedModelId;
 }

 function effectiveSelectedAuxiliaryModelId() {
 if (!activeProvider) return selectedAuxiliaryModelId;
 if (selectedAuxiliaryModelId && activeProviderModels.some((model) => model.id === selectedAuxiliaryModelId)) {
 return selectedAuxiliaryModelId;
 }
 return defaultAuxiliaryModelForProvider(catalog, activeProvider.id, effectiveSelectedModelId());
 }

 async function submitActiveProvider() {
 if (!activeProvider) return;
 if (activeProvider.id === "openai" && (authMode === "oauth-browser" || authMode === "oauth-headless")) {
 if (providerValidated || oauthStatus?.status === "success") {
 await completeActiveProvider();
 return;
 }
 const result = await validateActiveProvider();
 if (result.completed) {
 setOauthStarted(true);
 markOpenAIAuthorized(activeProvider);
 }
 return;
 }
 await completeActiveProvider();
 }

 function submitDisabled() {
 if (!activeProvider) return true;
 if (saving) return true;
 if (isCustomProviderChoice(activeProvider)) return !customProviderFormIsValid(customProviderForm);
 if (authMode === "api-key") return !apiKey.trim();
 if (authMode === "oauth-browser" && oauthStarted && !hasAppBridge()) return !callbackInput.trim();
 return false;
 }

 useEffect(() => {
 if (!hasAppBridge()) return;
 return EventsOn("provider_auth.updated", (...payloads: unknown[]) => {
 const status = normalizeProviderAuthUpdatedPayload(payloads);
 if (!status || status.providerId !== "openai") return;
 void window.aivo?.focusWindow?.();
 setOauthStarted(true);
 setOauthStatus(status);
 if (status.status === "failed") {
 return;
 }
 if (status.status !== "success") return;
 if (activeProvider) {
 markOpenAIAuthorized(activeProvider);
 }
 });
 }, [activeProvider, markOpenAIAuthorized]);

 const activeProviderModels =
 activeProvider && !isCustomProviderChoice(activeProvider)
 ? modelsForProvider(catalog, activeProvider.id)
 : [];
 const activeProviderModelValue = effectiveSelectedModelId();
 const showModelSelect =
 activeProviderModels.length > 0 &&
 Boolean(
 activeProvider &&
 (activeProvider.id !== "openai" ||
 authMode === "api-key" ||
 providerValidated ||
 oauthStatus?.status === "success"),
 );
 const oauthReady = providerValidated || oauthStatus?.status === "success";

 return (
 <section className="relative flex min-h-dvh justify-center bg-background px-5 py-20 sm:pt-28">
 <div className="flex w-full max-w-[1032px] flex-col items-center gap-6 text-center">
 <h1 className="max-w-[680px] text-3xl font-extrabold leading-9 tracking-normal text-foreground sm:text-4xl sm:leading-10">
 连接你的执行能力
 </h1>

 <div className="grid w-full max-w-[880px] grid-cols-2 gap-3 sm:grid-cols-3 sm:gap-4 min-[980px]:grid-cols-5">
 {providerChoices.map((provider) => (
 <ProviderChoiceCard
 key={provider.id}
 active={activeProvider?.id === provider.id}
 onClick={() => openProvider(provider)}
 provider={provider}
 />
 ))}
 </div>

 <ConnectedAccountsBar
 accounts={connectedAccounts}
 onAccountClick={setSettingsAccount}
 onRemoveAccount={onRemoveAccount}
 />
 {!activeProvider ? (
 <Button
 className="h-12 rounded-full px-8 text-base"
 onClick={onNextPage}
 size="lg"
 >
 继续
 </Button>
 ) : null}
 </div>

 <OtherProviderPickerDialog
 onOpenChange={setOtherProviderPickerOpen}
 onSelect={selectOtherProvider}
 open={otherProviderPickerOpen}
 providers={otherProviderChoices}
 search={otherProviderSearch}
 onSearchChange={setOtherProviderSearch}
 />

 <ProviderConnectionDialogs
 activeProvider={activeProvider}
 apiKey={apiKey}
 authMode={authMode}
 authSuccessMessage={authSuccessMessage}
 callbackInput={callbackInput}
 customProviderForm={customProviderForm}
 error={error}
 models={activeProviderModels}
 oauthReady={oauthReady}
 oauthStarted={oauthStarted}
 oauthStartResult={oauthStartResult}
 oauthStatus={oauthStatus}
 onApiKeyChange={setApiKey}
 onCallbackInputChange={setCallbackInput}
 onClose={closeProvider}
 onCustomProviderFormChange={setCustomProviderForm}
 onProviderDialogStepChange={setProviderDialogStep}
 onResetAuthMode={resetAuthMode}
 onSelectOpenAIAuthMode={selectOpenAIAuthMode}
 onSelectedModelIdChange={setSelectedModelId}
 onSelectedAuxiliaryModelIdChange={setSelectedAuxiliaryModelId}
 onSubmit={submitActiveProvider}
 providerDialogStep={providerDialogStep}
 saving={saving}
 selectedModelId={activeProviderModelValue}
 selectedAuxiliaryModelId={effectiveSelectedAuxiliaryModelId()}
 showModelSelect={showModelSelect}
 submitDisabled={submitDisabled()}
 />
 <ConnectedAccountModelDialog
 account={settingsAccount}
 catalog={catalog}
 config={config}
 onClose={() => setSettingsAccount(null)}
 onSave={onSaveAccountModels}
 />
 </section>
 );
}

function ProviderChoiceCard({
 active,
 onClick,
 provider,
}: {
 active: boolean;
 onClick: () => void;
 provider: ProviderChoice;
}) {
 return (
 <button
 className={cn(
 "flex h-28 min-w-0 flex-col items-center justify-center gap-3 rounded-2xl border p-4 text-center",
 active
 ? "border-primary bg-accent"
 : "border-border bg-card",
 )}
 onClick={onClick}
 type="button"
 >
 <ProviderIcon provider={provider} size="lg" />
 <span className="w-full min-w-0 truncate text-sm font-bold leading-4 text-foreground">
 {provider.name}
 </span>
 </button>
 );
}

function OtherProviderPickerDialog({
 onOpenChange,
 onSearchChange,
 onSelect,
 open,
 providers,
 search,
}: {
 onOpenChange: (open: boolean) => void;
 onSearchChange: (search: string) => void;
 onSelect: (provider: ProviderChoice) => void;
 open: boolean;
 providers: ProviderChoice[];
 search: string;
}) {
 const normalizedSearch = search.trim().toLowerCase();
 const filteredProviders = normalizedSearch
 ? providers.filter((provider) => {
 return (
 provider.name.toLowerCase().includes(normalizedSearch) ||
 provider.id.toLowerCase().includes(normalizedSearch)
 );
 })
 : providers;

 return (
 <Dialog open={open} onOpenChange={onOpenChange}>
 <DialogContent className="sm:max-w-lg" showCloseButton={false}>
 <div className="flex flex-col gap-4">
 <div className="flex items-center justify-between gap-3">
 <DialogTitle>选择提供商</DialogTitle>
 <DialogClose asChild>
 <Button aria-label="关闭" size="icon" variant="ghost">
 <X className="size-5" />
 </Button>
 </DialogClose>
 </div>

 <Input
 aria-label="搜索提供商"
 onChange={(event) => onSearchChange(event.target.value)}
 placeholder="搜索 provider"
 value={search}
 />

 <ScrollArea className="max-h-[min(52vh,420px)] pr-3">
 <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
 {filteredProviders.map((provider) => (
 <button
 className="flex items-center gap-2 rounded-lg border bg-background px-3 py-2 text-left text-sm  transition-colors hover:bg-muted"
 key={provider.id}
 onClick={() => onSelect(provider)}
 type="button"
 >
 <ProviderIcon provider={provider} size="sm" />
 <span className="min-w-0 truncate">{provider.name}</span>
 </button>
 ))}
 </div>
 </ScrollArea>

 {filteredProviders.length === 0 ? (
 <div className="rounded-lg border border-dashed py-8 text-center text-sm text-muted-foreground">
 没有匹配的提供商
 </div>
 ) : null}
 </div>
 </DialogContent>
 </Dialog>
 );
}

function ConnectedAccountsBar({
 accounts,
 onAccountClick,
 onRemoveAccount,
}: {
 accounts: ProviderAccountInfo[];
 onAccountClick: (account: ProviderAccountInfo) => void;
 onRemoveAccount: (accountId: string) => Promise<void>;
}) {
 if (accounts.length === 0) {
 return null;
 }

 return (
 <div className="flex w-full max-w-[880px] flex-wrap items-center justify-center gap-2">
 {accounts.map((account) => (
 <div
 key={account.id}
 className="inline-flex max-w-full items-center gap-1.5"
 >
 <Button
 className="max-w-full rounded-full px-3"
 onClick={() => onAccountClick(account)}
 size="sm"
 type="button"
 variant="secondary"
 >
 <span className="min-w-0 truncate">{accountTypeLabel(account)} {accountDisplayName(account)} 已连接</span>
 </Button>
 <Button
 aria-label={`删除 ${accountDisplayName(account)}`}
 className="rounded-full"
 onClick={(event) => {
 event.stopPropagation();
 void onRemoveAccount(account.id);
 }}
 size="icon"
 type="button"
 variant="ghost"
 >
 <X className="size-3" />
 </Button>
 </div>
 ))}
 </div>
 );
}

function ConnectedAccountModelDialog({
 account,
 catalog,
 config,
 onClose,
 onSave,
}: {
 account: ProviderAccountInfo | null;
 catalog: CatalogState | null;
 config: AppConfigWithAuxiliary | null;
 onClose: () => void;
 onSave: (providerId: string, modelId: string, auxiliaryModelId: string) => Promise<void>;
}) {
 const provider = account ? catalog?.providers.find((item) => item.id === account.providerId) : undefined;
 const modelOptions = provider ? modelOptionsForConnectedProvider(provider, catalog) : [];
 const defaultModelId = provider ? currentDefaultModelForProvider(config, provider, modelOptions) : "";
 const defaultAuxiliaryModelId = provider
 ? currentAuxiliaryModelForProvider(config, provider, modelOptions, defaultModelId)
 : "";
 const [modelId, setModelId] = useState(defaultModelId);
 const [auxiliaryModelId, setAuxiliaryModelId] = useState(defaultAuxiliaryModelId);
 const [saving, setSaving] = useState(false);

 useEffect(() => {
 setModelId(defaultModelId);
 setAuxiliaryModelId(defaultAuxiliaryModelId);
 }, [defaultModelId, defaultAuxiliaryModelId, account?.id]);

 async function handleSave() {
 if (!account || !modelId || !auxiliaryModelId) return;
 setSaving(true);
 try {
 await onSave(account.providerId, modelId, auxiliaryModelId);
 onClose();
 } finally {
 setSaving(false);
 }
 }

 return (
 <Dialog open={Boolean(account)} onOpenChange={(open) => {
 if (!open) onClose();
 }}>
 <DialogContent className="sm:max-w-md" showCloseButton={false}>
 <div className="flex flex-col gap-4">
 <div className="flex min-w-0 items-center justify-between gap-3">
 <DialogTitle className="min-w-0 truncate">{provider?.name || account?.providerId || "Provider"} 模型设置</DialogTitle>
 <DialogClose asChild>
 <Button aria-label="关闭" size="icon" type="button" variant="ghost">
 <X className="size-5" />
 </Button>
 </DialogClose>
 </div>

 <div className="grid gap-3 sm:grid-cols-2">
 <ConnectedModelSelect
 label="主模型"
 models={modelOptions}
 onValueChange={setModelId}
 value={modelId}
 />
 <ConnectedModelSelect
 label="辅助模型"
 models={modelOptions}
 onValueChange={setAuxiliaryModelId}
 value={auxiliaryModelId}
 />
 </div>

 <div className="flex justify-end gap-2">
 <DialogClose asChild>
 <Button type="button" variant="secondary">取消</Button>
 </DialogClose>
 <Button disabled={!modelId || !auxiliaryModelId || saving} onClick={handleSave} type="button">
 {saving ? "保存中" : "保存"}
 </Button>
 </div>
 </div>
 </DialogContent>
 </Dialog>
 );
}

function ConnectedModelSelect({
 label,
 models,
 onValueChange,
 value,
}: {
 label: string;
 models: ModelInfo[];
 onValueChange: (value: string) => void;
 value: string;
}) {
 return (
 <div className="flex flex-col gap-1.5 text-left">
 <label className="text-sm">{label}</label>
 <Select onValueChange={onValueChange} value={value}>
 <SelectTrigger>
 <SelectValue />
 </SelectTrigger>
 <SelectContent>
 {models.map((model) => (
 <SelectItem key={model.id} value={model.id}>
 {model.name || model.id}
 </SelectItem>
 ))}
 </SelectContent>
 </Select>
 </div>
 );
}

function emptyCustomProviderForm(): CustomProviderForm {
 return {
 providerId: "",
 displayName: "",
 protocol: "openai-compatible",
 baseUrl: "",
 apiKey: "",
 models: [emptyCustomRow()],
 headers: [emptyCustomRow()],
 };
}

function customProviderFormFor(provider: ProviderChoice): CustomProviderForm {
 if (!isCustomProviderChoice(provider)) return emptyCustomProviderForm();
 const defaultModel = knownDefaultModelForProvider(provider.id);
 return {
 ...emptyCustomProviderForm(),
 providerId: provider.id === "custom-api" ? "" : provider.id,
 displayName: provider.id === "custom-api" ? "" : provider.name,
 protocol: protocolForProvider(provider.id),
 baseUrl: providerBaseURLDefaults[provider.id] ?? "",
 models: [defaultModel ? { ...emptyCustomRow(), name: defaultModel } : emptyCustomRow()],
 };
}

function emptyCustomRow(): CustomProviderRow {
 return { id: crypto.randomUUID(), name: "", value: "" };
}

function customProviderFormIsValid(form: CustomProviderForm) {
 return Boolean(
 form.providerId.trim() &&
 form.displayName.trim() &&
 form.baseUrl.trim() &&
 form.models.some((model) => model.name.trim()),
 );
}

function headersFromRows(rows: CustomProviderRow[]) {
 return Object.fromEntries(
 rows
 .map((row) => [row.name.trim(), row.value.trim()] as const)
 .filter(([name, value]) => name && value),
 );
}

function isCustomProviderChoice(provider: ProviderChoice) {
 return provider.id === "custom-api";
}

function accountTypeLabel(account: ProviderAccountInfo) {
 if (account.providerId === "openai" && account.method === "oauth-browser") return "OpenAI Browser";
 if (account.providerId === "openai" && account.method === "oauth-headless") return "OpenAI Headless";
 if (account.method === "api-key") return `${providerNameForPrompt(account.providerId)} API Key`;
 return providerNameForPrompt(account.providerId);
}

function accountDisplayName(account: ProviderAccountInfo) {
 const displayName = account.displayName?.trim();
 const accountId = account.accountId?.trim();
 if (displayName) return displayName;
 return displayName || accountId || "默认账号";
}

function credentialReferenceFor(providerId: string) {
 if (providerId === "openai") return "OPENAI_API_KEY";
 if (providerId === "anthropic") return "ANTHROPIC_API_KEY";
 if (providerId === "claude-code") return "ANTHROPIC_API_KEY";
 if (providerId === "google") return "GEMINI_API_KEY";
 if (providerId === "gemini") return "GEMINI_API_KEY";
 if (providerId === "openrouter") return "OPENROUTER_API_KEY";
 return undefined;
}

function providerNameForPrompt(providerId: string) {
 if (providerId === "claude-code") return "Anthropic";
 if (providerId === "gemini") return "Google";
 if (providerId === "custom-api") return "Custom API";
 return providerDisplayName(providerId);
}

function providerDisplayName(providerId: string) {
 const knownNames: Record<string, string> = {
 "302ai": "302.AI",
 aihubmix: "AIHubMix",
 "alibaba-cn": "Alibaba CN",
 "alibaba-coding-plan-cn": "Alibaba Coding Plan CN",
 "alibaba-coding-plan": "Alibaba Coding Plan",
 alibaba: "Alibaba",
 anthropic: "Anthropic",
 "amazon-bedrock": "Amazon Bedrock",
 "azure-cognitive-services": "Azure Cognitive Services",
 "cloudflare-ai-gateway": "Cloudflare AI Gateway",
 "cloudflare-workers-ai": "Cloudflare Workers AI",
 "fireworks-ai": "Fireworks AI",
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
 "stackit": "STACKIT",
 "synthetic": "Synthetic",
 "tencent-coding-plan": "Tencent Coding Plan",
 "togetherai": "Together AI",
 "v0": "v0",
 "wandb": "Weights & Biases",
 };
 if (knownNames[providerId]) return knownNames[providerId];
 return providerId
 .split("-")
 .filter(Boolean)
 .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
 .join(" ");
}

function defaultModelForProvider(providerId: string, catalog: CatalogState | null = null) {
 const catalogDefault = catalogDefaultModelForProvider(catalog, providerId);
 if (catalogDefault) return catalogDefault;
 const defaultModel = knownDefaultModelForProvider(providerId);
 if (defaultModel) return defaultModel;
 if (providerId === "custom-api" || otherProviderChoices.some((provider) => provider.id === providerId)) return "custom-profile";
 return "gpt-5.5";
}

function catalogDefaultModelForProvider(catalog: CatalogState | null | undefined, providerId: string) {
 const provider = catalog?.providers.find((item) => item.id === providerId);
 if (!provider) return "";
 if (provider.defaultModelId) return provider.defaultModelId;
 return provider.models.find((model) => model.recommended)?.id || provider.models[0]?.id || "";
}

function currentDefaultModelForProvider(
 config: AppConfigWithAuxiliary | null,
 provider: ProviderInfo,
 modelOptions: ModelInfo[],
) {
 if (config?.defaultModel?.providerId === provider.id && config.defaultModel.modelId) {
 return modelInOptionsOrFirst(modelOptions, config.defaultModel.modelId);
 }
 if (config?.provider?.id === provider.id && config.provider.model) {
 return modelInOptionsOrFirst(modelOptions, config.provider.model);
 }
 return modelInOptionsOrFirst(modelOptions, provider.defaultModelId || modelOptions[0]?.id || "");
}

function currentAuxiliaryModelForProvider(
 config: AppConfigWithAuxiliary | null,
 provider: ProviderInfo,
 modelOptions: ModelInfo[],
 fallbackModelId: string,
) {
 if (config?.auxiliaryModel?.providerId === provider.id && config.auxiliaryModel.modelId) {
 return modelInOptionsOrFirst(modelOptions, config.auxiliaryModel.modelId);
 }
 return defaultAuxiliaryModelForProvider({ providers: [provider], models: modelOptions, connected: [] }, provider.id, fallbackModelId);
}

function modelInOptionsOrFirst(modelOptions: ModelInfo[], modelId: string) {
 if (!modelId) return modelOptions[0]?.id || "";
 if (modelOptions.some((model) => model.id === modelId)) return modelId;
 return modelOptions[0]?.id || modelId;
}

function modelOptionsForConnectedProvider(provider: ProviderInfo, catalog: CatalogState | null) {
 const models = modelsForProvider(catalog, provider.id);
 if (models.length > 0) return models;
 const fallback = provider.defaultModelId || defaultModelForProvider(provider.id, catalog);
 return fallback ? [{ id: fallback, providerId: provider.id, name: fallback }] : [];
}

function updateCatalogDefaultModel(catalog: CatalogState | null, providerId: string, modelId: string) {
 if (!catalog) return { providers: [], models: [], connected: [] };
 return {
 ...catalog,
 defaultModel: { providerId, modelId },
 providers: catalog.providers.map((provider) =>
 provider.id === providerId ? { ...provider, defaultModelId: modelId } : provider,
 ),
 };
}

function defaultAuxiliaryModelForProvider(catalog: CatalogState | null | undefined, providerId: string, fallbackModelId: string) {
 const models = modelsForProvider(catalog, providerId);
 const priority = auxiliaryModelPriorityForProvider(providerId);
 for (const item of priority) {
 const match = models.find((model) => model.id.includes(item));
 if (match) return match.id;
 }
 return fallbackModelId || catalogDefaultModelForProvider(catalog, providerId);
}

function auxiliaryModelPriorityForProvider(providerId: string) {
 if (providerId.startsWith("opencode")) return ["gpt-5.4-mini", "gpt-5-mini"];
 const priority = [
 "claude-haiku-4-5",
 "claude-haiku-4.5",
 "3-5-haiku",
 "3.5-haiku",
 "gemini-3-flash",
 "gemini-2.5-flash",
 "gpt-5.4-mini",
 "gpt-5-mini",
 ];
 if (providerId.startsWith("github-copilot")) {
 return ["gpt-5-mini", "claude-haiku-4.5", ...priority];
 }
 return priority;
}

function modelsForProvider(catalog: CatalogState | null | undefined, providerId: string) {
 return catalog?.providers.find((provider) => provider.id === providerId)?.models ?? [];
}

function canRefreshProviderModels(
 provider: ProviderChoice,
 customProvider: CustomProviderForm,
 authMode: ProviderAuthMode,
 apiKey: string,
 catalog: CatalogState | null,
) {
 if (isCustomProviderChoice(provider)) {
 return Boolean(
 customProvider.providerId.trim() &&
 customProvider.baseUrl.trim() &&
 (customProvider.apiKey.trim() || customProvider.protocol === "openai-compatible"),
 );
 }
 if (authMode === "api-key") {
 return Boolean(apiKey.trim() || catalogProviderHasCredential(catalog, provider.id));
 }
 if (provider.id === "openai" && (authMode === "oauth-browser" || authMode === "oauth-headless")) {
 return catalogProviderHasCredential(catalog, provider.id);
 }
 return catalogProviderHasCredential(catalog, provider.id);
}

function catalogProviderHasCredential(catalog: CatalogState | null | undefined, providerId: string) {
 const catalogProvider = catalog?.providers.find((provider) => provider.id === providerId);
 return Boolean(
 catalogProvider?.auth?.connected ||
 catalogProvider?.connected ||
 catalogProvider?.accounts?.some((account) => account.providerId === providerId),
 );
}

function providerRefreshInput(
 provider: ProviderChoice,
 customProvider: CustomProviderForm,
 authMode: ProviderAuthMode,
 apiKey: string,
 selectedModelId: string,
): ProviderConnectInput {
 const isCustomProvider = isCustomProviderChoice(provider);
 const customModel = customProvider.models.find((model) => model.name.trim());
 return {
 providerId: isCustomProvider ? customProvider.providerId.trim() : provider.id,
 name: isCustomProvider ? customProvider.displayName.trim() : provider.name,
 type: isCustomProvider ? customProvider.protocol : providerTypeFor(provider.id),
 baseUrl: isCustomProvider ? customProvider.baseUrl.trim() : defaultBaseURLForProvider(provider.id),
 apiKeyEnv: isCustomProvider || (provider.id === "openai" && authMode !== "api-key") ? undefined : credentialReferenceFor(provider.id),
 apiKey: isCustomProvider ? customProvider.apiKey.trim() : authMode === "api-key" ? apiKey.trim() : undefined,
 modelId: isCustomProvider ? customModel?.name.trim() : selectedModelId || defaultModelForProvider(provider.id),
 method: authMode,
 headers: isCustomProvider ? headersFromRows(customProvider.headers) : undefined,
 };
}

function knownDefaultModelForProvider(providerId: string) {
 return providerModelDefaults[providerId];
}

function defaultBaseURLForProvider(providerId: string) {
 return providerBaseURLDefaults[providerId];
}

function providerTypeFor(providerId: string) {
 return protocolForProvider(providerId);
}

function protocolForProvider(providerId: string): CustomProviderProtocol {
 return providerProtocolDefaults[providerId] ?? "openai-compatible";
}

function normalizeProviderAuthUpdatedPayload(payloads: unknown[]) {
 const first = payloads[0];
 if (!first || typeof first !== "object" || Array.isArray(first)) return null;
 const record = first as Record<string, unknown>;
 const status = record.status;
 if (!status || typeof status !== "object" || Array.isArray(status)) return null;
 const statusRecord = status as Record<string, unknown>;
 if (typeof statusRecord.providerId !== "string") return null;
 return statusRecord as unknown as domain.ProviderAuthStatus;
}

async function openExternalURL(url: string) {
 await BrowserOpenURL(url);
}
