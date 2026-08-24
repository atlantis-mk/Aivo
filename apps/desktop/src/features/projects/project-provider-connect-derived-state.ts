import { useMemo } from "react";

import type {
  CustomProviderForm,
  ProviderAuthMode,
} from "@/features/providers/provider-types";
import type { ProviderInfo } from "@/lib/provider-catalog";
import {
  providerPickerOptions,
  type ProviderPickOption,
} from "@/features/projects/project-provider-picker-model";
import {
  activeProviderModelValue as resolveActiveProviderModelValue,
  activeProviderModelsFor,
  providerSubmitDisabled,
  shouldShowProviderModelSelect,
} from "@/features/projects/project-provider-connect-model";

export function useProviderConnectDerivedState({
  apiKey,
  authMode,
  catalogProviders,
  customProviderForm,
  oauthReady,
  query,
  selectedModelId,
  selectedProvider,
  submitting,
}: {
  apiKey: string;
  authMode: ProviderAuthMode;
  catalogProviders: ProviderInfo[];
  customProviderForm: CustomProviderForm;
  oauthReady: boolean;
  query: string;
  selectedModelId: string;
  selectedProvider: ProviderPickOption | null;
  submitting: boolean;
}) {
  const providerOptions = useMemo(
    () => providerPickerOptions(catalogProviders),
    [catalogProviders],
  );
  const normalizedQuery = query.trim().toLowerCase();
  const filteredProviders = normalizedQuery
    ? providerOptions.filter((provider) =>
        `${provider.name} ${provider.id} ${provider.type}`
          .toLowerCase()
          .includes(normalizedQuery),
      )
    : providerOptions;
  const activeProviderModels = activeProviderModelsFor(
    catalogProviders,
    selectedProvider,
  );
  const activeProviderModelValue = resolveActiveProviderModelValue({
    activeProviderModels,
    catalogProviders,
    selectedModelId,
    selectedProvider,
  });
  const showModelSelect = shouldShowProviderModelSelect({
    activeProviderModels,
    authMode,
    oauthReady,
    selectedProvider,
  });
  const submitDisabled = providerSubmitDisabled({
    apiKey,
    authMode,
    customProviderForm,
    selectedProvider,
    submitting,
  });

  return {
    activeProviderModelValue,
    activeProviderModels,
    filteredProviders,
    showModelSelect,
    submitDisabled,
  };
}
