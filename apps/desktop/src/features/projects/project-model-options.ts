import type { ModelInfo, ProviderInfo } from "@/lib/provider-catalog";
import type { PermissionMode } from "@/services/aivo";
import type { domain } from "../../../bridge/go/models";

export type WebSearchMode = "disabled" | "cached" | "indexed" | "live";

export const webSearchModeOptions: Array<{
  mode: WebSearchMode;
  label: string;
  description: string;
}> = [
  {
    mode: "live",
    label: "实时搜索",
    description: "允许 provider 使用实时互联网搜索",
  },
  {
    mode: "indexed",
    label: "索引搜索",
    description: "仅允许 provider 使用索引网络访问",
  },
  {
    mode: "cached",
    label: "缓存搜索",
    description: "禁用实时外网访问，仅保留缓存搜索能力",
  },
  {
    mode: "disabled",
    label: "关闭搜索",
    description: "不向 provider 暴露 Web Search",
  },
];

export type ModelOption = ModelInfo & {
  providerName: string;
};

export function getActiveProvider(
  config: domain.AppConfig | null,
  providers: ProviderInfo[],
  selectedProviderId = "",
) {
  const providerId =
    selectedProviderId ||
    config?.provider?.id ||
    config?.defaultModel?.providerId ||
    providers[0]?.id ||
    "";
  return (
    providers.find((provider) => provider.id === providerId) ??
    providers[0] ??
    null
  );
}

export function getConnectedModelProviders(
  config: domain.AppConfig | null,
  providers: ProviderInfo[],
  connectedProviders: ProviderInfo[],
) {
  if (connectedProviders.length > 0) return connectedProviders;
  const configuredProviderIds = new Set(
    [config?.provider?.id, config?.defaultModel?.providerId].filter(Boolean),
  );
  return providers.filter(
    (provider) => provider.connected || configuredProviderIds.has(provider.id),
  );
}

export function getModelOptions(
  provider: ProviderInfo | null,
  catalogModels: ModelInfo[],
) {
  if (!provider) return [];
  const providerModels = provider.models?.length
    ? provider.models
    : catalogModels.filter((model) => model.providerId === provider.id);
  const seen = new Set<string>();
  return providerModels.filter((model) => {
    if (!model.id || seen.has(model.id)) return false;
    seen.add(model.id);
    return true;
  });
}

export function getAllModelOptions(
  providers: ProviderInfo[],
  catalogModels: ModelInfo[],
): ModelOption[] {
  const out: ModelOption[] = [];
  const seen = new Set<string>();
  for (const provider of providers) {
    const models = getModelOptions(provider, catalogModels);
    for (const model of models) {
      const normalizedId = normalizeModelId(provider.id, model.id);
      const key = modelOptionKey(provider.id, normalizedId);
      if (seen.has(key)) continue;
      seen.add(key);
      out.push({
        ...model,
        id: normalizedId,
        providerId: provider.id,
        providerName: provider.name,
      });
    }
  }
  return out;
}

export function getDefaultModelId(
  config: domain.AppConfig | null,
  provider: ProviderInfo | null,
  modelOptions: ModelInfo[],
) {
  if (!provider) return "";
  if (
    config?.defaultModel?.providerId === provider.id &&
    config.defaultModel.modelId
  ) {
    const modelId = normalizeModelId(provider.id, config.defaultModel.modelId);
    if (
      modelOptions.length === 0 ||
      modelOptions.some((model) => model.id === modelId)
    )
      return modelId;
  }
  if (config?.provider?.id === provider.id && config.provider.model) {
    const modelId = normalizeModelId(provider.id, config.provider.model);
    if (
      modelOptions.length === 0 ||
      modelOptions.some((model) => model.id === modelId)
    )
      return modelId;
  }
  return provider.defaultModelId || modelOptions[0]?.id || "";
}

export function getModelLabel(modelOptions: ModelInfo[], modelId: string) {
  const normalizedModelId = normalizeModelId(
    modelOptions[0]?.providerId,
    modelId,
  );
  return (
    modelOptions.find((model) => model.id === normalizedModelId)?.name ||
    normalizedModelId
  );
}

export function formatModelTriggerLabel(modelLabel: string) {
  return modelLabel
    .replace(/^GPT-/i, "")
    .replace(/^Claude\s+/i, "")
    .replace(/^Gemini\s+/i, "");
}

export function normalizeModelId(
  providerId: string | undefined,
  modelId: string,
) {
  if (providerId === "openai" && modelId === "gpt-5-codex") return "gpt-5.5";
  return modelId;
}

export function normalizeReasoningEffort(effort: string | undefined) {
  if (
    effort === "none" ||
    effort === "minimal" ||
    effort === "low" ||
    effort === "medium" ||
    effort === "high" ||
    effort === "xhigh" ||
    effort === "max" ||
    effort === "ultra"
  )
    return effort;
  if (effort === "低") return "low";
  if (effort === "中") return "medium";
  if (effort === "高") return "high";
  if (effort === "超高") return "ultra";
  return "medium";
}

export function reasoningEffortLabel(effort: string) {
  switch (normalizeReasoningEffort(effort)) {
    case "none":
      return "无";
    case "minimal":
      return "最小";
    case "low":
      return "低";
    case "high":
      return "高";
    case "ultra":
      return "超高";
    case "xhigh":
      return "特高";
    case "max":
      return "最大";
    default:
      return "中";
  }
}

export function normalizeServiceTier(serviceTier: string | undefined) {
  if (serviceTier === "priority" || serviceTier === "fast") return "priority";
  if (serviceTier === "flex") return "flex";
  return "default";
}

export function serviceTierLabel(serviceTier: string) {
  switch (normalizeServiceTier(serviceTier)) {
    case "priority":
      return "快速";
    case "flex":
      return "弹性";
    default:
      return "标准";
  }
}

export function normalizePermissionMode(
  mode: string | undefined,
): PermissionMode {
  if (mode === "request_approval" || mode === "full_access") {
    return mode;
  }
  return "request_approval";
}

export function normalizeWebSearchMode(
  mode: string | undefined,
): WebSearchMode {
  if (
    mode === "disabled" ||
    mode === "cached" ||
    mode === "indexed" ||
    mode === "live"
  ) {
    return mode;
  }
  return "live";
}

export function webSearchModeLabel(mode: string | undefined) {
  switch (normalizeWebSearchMode(mode)) {
    case "disabled":
      return "关闭";
    case "cached":
      return "缓存";
    case "indexed":
      return "索引";
    default:
      return "实时";
  }
}

export function providerSupportsServiceTier(providerId: string | undefined) {
  return providerId === "openai";
}

export function groupModelOptionsByProvider(models: ModelOption[]) {
  const groups: Array<{
    providerId: string;
    providerName: string;
    models: ModelOption[];
  }> = [];
  const indexes = new Map<string, number>();
  for (const model of models) {
    let index = indexes.get(model.providerId);
    if (index === undefined) {
      index = groups.length;
      indexes.set(model.providerId, index);
      groups.push({
        providerId: model.providerId,
        providerName: model.providerName,
        models: [],
      });
    }
    groups[index].models.push(model);
  }
  return groups;
}

export function modelOptionKey(providerId: string, modelId: string) {
  return `${providerId}:${normalizeModelId(providerId, modelId)}`;
}
