import type { domain } from "../../../bridge/go/models";
import {
  customProviderFormIsValid,
  headersFromRows,
} from "@/features/providers/custom-provider-form";
import type {
  CustomProviderForm,
  ProviderAuthMode,
} from "@/features/providers/provider-types";
import {
  providerBaseURLDefaults,
  providerProtocolForProvider,
} from "@/features/providers/provider-defaults";
import type { ProviderInfo } from "@/lib/provider-catalog";
import {
  catalogDefaultModelForProvider,
  defaultModelForProvider,
  type ProviderPickOption,
} from "@/features/projects/project-provider-picker-model";

export function activeProviderModelsFor(
  catalogProviders: ProviderInfo[],
  selectedProvider: ProviderPickOption | null,
) {
  if (!selectedProvider || selectedProvider.id === "custom-api") return [];
  return (
    catalogProviders.find((provider) => provider.id === selectedProvider.id)
      ?.models ?? selectedProvider.models
  );
}

export function activeProviderModelValue({
  activeProviderModels,
  catalogProviders,
  selectedModelId,
  selectedProvider,
}: {
  activeProviderModels: ProviderPickOption["models"];
  catalogProviders: ProviderInfo[];
  selectedModelId: string;
  selectedProvider: ProviderPickOption | null;
}) {
  return (
    selectedModelId ||
    (selectedProvider
      ? catalogDefaultModelForProvider(catalogProviders, selectedProvider.id) ||
        activeProviderModels[0]?.id ||
        ""
      : "")
  );
}

export function shouldShowProviderModelSelect({
  activeProviderModels,
  authMode,
  oauthReady,
  selectedProvider,
}: {
  activeProviderModels: ProviderPickOption["models"];
  authMode: ProviderAuthMode;
  oauthReady: boolean;
  selectedProvider: ProviderPickOption | null;
}) {
  return (
    activeProviderModels.length > 0 &&
    Boolean(
      selectedProvider &&
        (selectedProvider.id !== "openai" ||
          authMode === "api-key" ||
          oauthReady),
    )
  );
}

export function providerSubmitDisabled({
  apiKey,
  authMode,
  customProviderForm,
  selectedProvider,
  submitting,
}: {
  apiKey: string;
  authMode: ProviderAuthMode;
  customProviderForm: CustomProviderForm;
  selectedProvider: ProviderPickOption | null;
  submitting: boolean;
}) {
  return (
    submitting ||
    !selectedProvider ||
    (selectedProvider.id === "custom-api"
      ? !customProviderFormIsValid(customProviderForm)
      : authMode === "api-key"
        ? !apiKey.trim()
        : false)
  );
}

export function providerConnectInputFromState({
  apiKey,
  authMode,
  catalogProviders,
  customProviderForm,
  selectedModelId,
  selectedProvider,
}: {
  apiKey: string;
  authMode: ProviderAuthMode;
  catalogProviders: ProviderInfo[];
  customProviderForm: CustomProviderForm;
  selectedModelId: string;
  selectedProvider: ProviderPickOption;
}) {
  const isCustom = selectedProvider.id === "custom-api";
  const customModel = customProviderForm.models.find((model) =>
    model.name.trim(),
  );
  const providerId = isCustom
    ? customProviderForm.providerId.trim()
    : selectedProvider.id;
  const providerName = isCustom
    ? customProviderForm.displayName.trim()
    : selectedProvider.name;
  const baseUrl = isCustom
    ? customProviderForm.baseUrl.trim()
    : selectedProvider.baseUrl ||
      providerBaseURLDefaults[selectedProvider.id] ||
      "";
  const modelId = isCustom
    ? customModel?.name.trim()
    : selectedModelId ||
      catalogDefaultModelForProvider(catalogProviders, selectedProvider.id) ||
      defaultModelForProvider(selectedProvider.id);
  return {
    input: {
      providerId,
      name: providerName || providerId,
      type: isCustom
        ? customProviderForm.protocol
        : selectedProvider.type ||
          providerProtocolForProvider(selectedProvider.id),
      baseUrl,
      apiKey: isCustom
        ? customProviderForm.apiKey.trim()
        : authMode === "api-key"
          ? apiKey.trim()
          : undefined,
      modelId: modelId || "default",
      method: selectedProvider.id === "openai" ? authMode : "api-key",
      headers: isCustom
        ? headersFromRows(customProviderForm.headers)
        : undefined,
    } as domain.ProviderConnectInput,
    providerId,
  };
}
