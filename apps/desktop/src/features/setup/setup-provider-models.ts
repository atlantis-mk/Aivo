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
  ProviderConnectInput,
  ProviderInfo,
} from "@/lib/provider-catalog";

export type AppConfigWithAuxiliary = domain.AppConfig & {
  appName?: string;
  auxiliaryModel?: domain.ModelRef;
  initialWorkspacePath?: string;
  defaultInitialWorkspacePath?: string;
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
