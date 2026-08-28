import type { domain } from "../../../bridge/go/models";
import { isCustomProviderChoice } from "@/features/providers/custom-provider-form";
import type {
  ProviderAuthMode,
  ProviderChoice,
} from "@/features/providers/provider-types";
import {
  catalogDefaultModelForProvider,
  modelsForProvider,
} from "@/features/setup/setup-provider-models";
import type { CatalogState, ModelInfo } from "@/lib/provider-catalog";

export function modelsForActiveProvider(
  catalog: CatalogState | null,
  provider: ProviderChoice | null,
) {
  if (!provider || isCustomProviderChoice(provider)) return [];
  return modelsForProvider(catalog, provider.id);
}

export function selectedModelIdForProvider({
  catalog,
  models,
  provider,
  selectedModelId,
}: {
  catalog: CatalogState | null;
  models: ModelInfo[];
  provider: ProviderChoice | null;
  selectedModelId: string;
}) {
  if (!provider) return selectedModelId;
  if (selectedModelId.trim()) {
    return selectedModelId;
  }
  return (
    catalogDefaultModelForProvider(catalog, provider.id) ||
    models[0]?.id ||
    selectedModelId
  );
}

export function oauthReady(
  providerValidated: boolean,
  oauthStatus: domain.ProviderAuthStatus | null,
) {
  return providerValidated || oauthStatus?.status === "success";
}

export function shouldShowModelSelect({
  authMode,
  models,
  oauthStatus,
  provider,
  providerValidated,
}: {
  authMode: ProviderAuthMode;
  models: ModelInfo[];
  oauthStatus: domain.ProviderAuthStatus | null;
  provider: ProviderChoice | null;
  providerValidated: boolean;
}) {
  return (
    models.length > 0 &&
    Boolean(
      provider &&
        (provider.id !== "openai" ||
          authMode === "api-key" ||
          providerValidated ||
          oauthStatus?.status === "success"),
    )
  );
}

export function oauthStatusFromAuthStart(start: domain.ProviderAuthStartResult) {
  return {
    providerId: start.providerId,
    method: start.method,
    status: start.status,
    instructions: start.instructions,
    userCode: start.userCode,
  } as domain.ProviderAuthStatus;
}

export function successfulOpenAIStatus(
  current: domain.ProviderAuthStatus | null,
  authMode: ProviderAuthMode,
) {
  return {
    providerId: "openai",
    method: current?.method || authMode,
    status: "success",
    accountId: current?.accountId,
    instructions: current?.instructions,
    userCode: current?.userCode,
  } as domain.ProviderAuthStatus;
}
