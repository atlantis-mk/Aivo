import type { domain } from "../../../bridge/go/models";
import {
  headersFromRows,
  isCustomProviderChoice,
} from "@/features/providers/custom-provider-form";
import {
  credentialReferenceFor,
  defaultBaseURLForProvider,
  providerTypeFor,
} from "@/features/providers/provider-defaults";
import type {
  CustomProviderForm,
  ProviderAuthMode,
  ProviderChoice,
} from "@/features/providers/provider-types";
import {
  defaultModelForProvider,
  type AppConfigWithAuxiliary,
} from "@/features/setup/setup-provider-models";
import {
  modelPreferencesWithAuxiliary,
  modelRefForProvider,
} from "@/features/setup/setup-provider-bridge-inputs";
import type { CatalogState, ProviderConnectInput } from "@/lib/provider-catalog";

export function providerConnectionDraft({
  apiKey,
  authMode,
  catalog,
  customProvider,
  provider,
  selectedAuxiliaryModelId,
  selectedModelId,
}: {
  apiKey?: string;
  authMode: ProviderAuthMode;
  catalog: CatalogState | null;
  customProvider?: CustomProviderForm;
  provider: ProviderChoice;
  selectedAuxiliaryModelId?: string;
  selectedModelId?: string;
}) {
  const isCustomProvider = isCustomProviderChoice(provider) && customProvider;
  const customModel = customProvider?.models.find((model) => model.name.trim());
  const input: ProviderConnectInput = {
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
        ? apiKey?.trim()
        : undefined,
    modelId: isCustomProvider
      ? customModel?.name.trim()
      : selectedModelId?.trim() || defaultModelForProvider(provider.id, catalog),
    method: authMode,
    headers: isCustomProvider ? headersFromRows(customProvider.headers) : undefined,
  };
  const auxiliaryModelId = selectedAuxiliaryModelId?.trim() || input.modelId;
  const auxiliaryModel =
    auxiliaryModelId && input.providerId
      ? modelRefForProvider(input.providerId, auxiliaryModelId)
      : undefined;

  return {
    auxiliaryModel,
    input,
    isCustomProvider,
  };
}

export function connectedProviderConfig(
  input: ProviderConnectInput,
  auxiliaryModel: domain.ModelRef | undefined,
) {
  return {
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
}

export function modelPreferencesForProvider(
  providerId: string,
  modelId: string,
  auxiliaryModelId: string,
) {
  return modelPreferencesWithAuxiliary(
    modelRefForProvider(providerId, modelId),
    modelRefForProvider(providerId, auxiliaryModelId),
  );
}

export function previewConfigForConnectedAccountModels({
  auxiliaryModel,
  config,
  model,
}: {
  auxiliaryModel: domain.ModelRef;
  config: domain.AppConfig | null;
  model: domain.ModelRef;
}) {
  return {
    ...(config ?? {}),
    initialized: true,
    defaultModel: model,
    auxiliaryModel,
    provider: {
      ...((config?.provider ?? {}) as domain.ProviderConfig),
      id: model.providerId,
      type: config?.provider?.type || providerTypeFor(model.providerId),
      baseUrl:
        config?.provider?.baseUrl || defaultBaseURLForProvider(model.providerId),
      model: model.modelId,
    },
  } as AppConfigWithAuxiliary;
}
