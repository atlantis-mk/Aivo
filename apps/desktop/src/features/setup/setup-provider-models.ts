import type { domain } from "../../../bridge/go/models";
import {
  headersFromRows,
  isCustomProviderChoice,
} from "@/features/providers/custom-provider-form";
import {
  credentialReferenceFor,
  defaultBaseURLForProvider,
  knownDefaultModelForProvider,
  providerTypeFor,
} from "@/features/providers/provider-defaults";
import type {
  CustomProviderForm,
  ProviderAuthMode,
  ProviderChoice,
} from "@/features/providers/provider-types";
import { otherProviderChoices } from "@/features/setup/setup-provider-options";
import type {
  CatalogState,
  ModelInfo,
  ProviderConnectInput,
  ProviderInfo,
} from "@/lib/provider-catalog";

export type AppConfigWithAuxiliary = domain.AppConfig & {
  auxiliaryModel?: domain.ModelRef;
};

export function defaultModelForProvider(
  providerId: string,
  catalog: CatalogState | null = null,
) {
  const catalogDefault = catalogDefaultModelForProvider(catalog, providerId);
  if (catalogDefault) return catalogDefault;
  const defaultModel = knownDefaultModelForProvider(providerId);
  if (defaultModel) return defaultModel;
  if (
    providerId === "custom-api" ||
    otherProviderChoices.some((provider) => provider.id === providerId)
  ) {
    return "custom-profile";
  }
  return "gpt-5.5";
}

export function catalogDefaultModelForProvider(
  catalog: CatalogState | null | undefined,
  providerId: string,
) {
  const provider = catalog?.providers.find((item) => item.id === providerId);
  if (!provider) return "";
  if (provider.defaultModelId) return provider.defaultModelId;
  return (
    provider.models.find((model) => model.recommended)?.id ||
    provider.models[0]?.id ||
    ""
  );
}

export function currentDefaultModelForProvider(
  config: AppConfigWithAuxiliary | null,
  provider: ProviderInfo,
  modelOptions: ModelInfo[],
) {
  if (
    config?.defaultModel?.providerId === provider.id &&
    config.defaultModel.modelId
  ) {
    return modelInOptionsOrFirst(modelOptions, config.defaultModel.modelId);
  }
  if (config?.provider?.id === provider.id && config.provider.model) {
    return modelInOptionsOrFirst(modelOptions, config.provider.model);
  }
  return modelInOptionsOrFirst(
    modelOptions,
    provider.defaultModelId || modelOptions[0]?.id || "",
  );
}

export function currentAuxiliaryModelForProvider(
  config: AppConfigWithAuxiliary | null,
  provider: ProviderInfo,
  modelOptions: ModelInfo[],
  fallbackModelId: string,
) {
  if (
    config?.auxiliaryModel?.providerId === provider.id &&
    config.auxiliaryModel.modelId
  ) {
    return modelInOptionsOrFirst(modelOptions, config.auxiliaryModel.modelId);
  }
  return defaultAuxiliaryModelForProvider(
    { providers: [provider], models: modelOptions, connected: [] },
    provider.id,
    fallbackModelId,
  );
}

export function modelOptionsForConnectedProvider(
  provider: ProviderInfo,
  catalog: CatalogState | null,
) {
  const models = modelsForProvider(catalog, provider.id);
  if (models.length > 0) return models;
  const fallback =
    provider.defaultModelId || defaultModelForProvider(provider.id, catalog);
  return fallback ? [{ id: fallback, providerId: provider.id, name: fallback }] : [];
}

export function updateCatalogDefaultModel(
  catalog: CatalogState | null,
  providerId: string,
  modelId: string,
) {
  if (!catalog) return { providers: [], models: [], connected: [] };
  return {
    ...catalog,
    defaultModel: { providerId, modelId },
    providers: catalog.providers.map((provider) =>
      provider.id === providerId
        ? { ...provider, defaultModelId: modelId }
        : provider,
    ),
  };
}

export function defaultAuxiliaryModelForProvider(
  catalog: CatalogState | null | undefined,
  providerId: string,
  fallbackModelId: string,
) {
  const models = modelsForProvider(catalog, providerId);
  const priority = auxiliaryModelPriorityForProvider(providerId);
  for (const item of priority) {
    const match = models.find((model) => model.id.includes(item));
    if (match) return match.id;
  }
  return fallbackModelId || catalogDefaultModelForProvider(catalog, providerId);
}

export function canRefreshProviderModels(
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
        (customProvider.apiKey.trim() ||
          customProvider.protocol === "openai-compatible"),
    );
  }
  if (authMode === "api-key") {
    return Boolean(apiKey.trim() || catalogProviderHasCredential(catalog, provider.id));
  }
  if (
    provider.id === "openai" &&
    (authMode === "oauth-browser" || authMode === "oauth-headless")
  ) {
    return catalogProviderHasCredential(catalog, provider.id);
  }
  return catalogProviderHasCredential(catalog, provider.id);
}

export function providerRefreshInput(
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
    baseUrl: isCustomProvider
      ? customProvider.baseUrl.trim()
      : defaultBaseURLForProvider(provider.id),
    apiKeyEnv:
      isCustomProvider || (provider.id === "openai" && authMode !== "api-key")
        ? undefined
        : credentialReferenceFor(provider.id),
    apiKey: isCustomProvider
      ? customProvider.apiKey.trim()
      : authMode === "api-key"
        ? apiKey.trim()
        : undefined,
    modelId: isCustomProvider
      ? customModel?.name.trim()
      : selectedModelId || defaultModelForProvider(provider.id),
    method: authMode,
    headers: isCustomProvider ? headersFromRows(customProvider.headers) : undefined,
  };
}

function modelInOptionsOrFirst(modelOptions: ModelInfo[], modelId: string) {
  if (!modelId) return modelOptions[0]?.id || "";
  if (modelOptions.some((model) => model.id === modelId)) return modelId;
  return modelOptions[0]?.id || modelId;
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

export function modelsForProvider(
  catalog: CatalogState | null | undefined,
  providerId: string,
) {
  return (
    catalog?.providers.find((provider) => provider.id === providerId)?.models ?? []
  );
}

function catalogProviderHasCredential(
  catalog: CatalogState | null | undefined,
  providerId: string,
) {
  const catalogProvider = catalog?.providers.find(
    (provider) => provider.id === providerId,
  );
  return Boolean(
    catalogProvider?.auth?.connected ||
      catalogProvider?.connected ||
      catalogProvider?.accounts?.some(
        (account) => account.providerId === providerId,
      ),
  );
}
