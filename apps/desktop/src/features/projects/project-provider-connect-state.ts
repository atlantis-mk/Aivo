import { useState } from "react";

import type { domain } from "../../../bridge/go/models";
import {
  customProviderFormFor,
  customProviderFormIsValid,
  emptyCustomProviderForm,
} from "@/features/providers/custom-provider-form";
import type {
  CustomProviderForm,
  ProviderAuthMode,
  ProviderDialogStep,
} from "@/features/providers/provider-types";
import { hasAppBridge } from "@/lib/app-config";
import type { CatalogState, ProviderInfo } from "@/lib/provider-catalog";
import { connectPreviewProvider } from "@/lib/preview-state";
import { connectProvider, refreshProviderModels } from "@/services/aivo";
import type { ModelOption } from "@/features/projects/project-model-options";
import {
  catalogDefaultModelForProvider,
  defaultModelForProvider,
  defaultModelFromCatalog,
  modelOptionFromCatalog,
  type ProviderPickOption,
} from "@/features/projects/project-provider-picker-model";
import { providerConnectInputFromState } from "@/features/projects/project-provider-connect-model";
import { useProviderConnectDerivedState } from "@/features/projects/project-provider-connect-derived-state";
import { useOpenAIProviderAuthState } from "@/features/projects/project-provider-openai-auth-state";

export function useProviderConnectState({
  catalogProviders,
  onConnected,
  onOpenChange,
  setCatalog,
  setConfig,
  setError,
}: {
  catalogProviders: ProviderInfo[];
  onConnected: (option: ModelOption | null) => Promise<void>;
  onOpenChange: (open: boolean) => void;
  setCatalog: (catalog: CatalogState) => void;
  setConfig: (config: domain.AppConfig) => void;
  setError: (error: string) => void;
}) {
  const [query, setQuery] = useState("");
  const [selectedProvider, setSelectedProvider] =
    useState<ProviderPickOption | null>(null);
  const [providerDialogStep, setProviderDialogStep] =
    useState<ProviderDialogStep>("details");
  const [authMode, setAuthMode] = useState<ProviderAuthMode>("api-key");
  const [callbackInput, setCallbackInput] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [customProviderForm, setCustomProviderForm] =
    useState<CustomProviderForm>(() => emptyCustomProviderForm());
  const [selectedModelId, setSelectedModelId] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [localError, setLocalError] = useState("");
  const {
    authSuccessMessage,
    oauthReady,
    oauthStarted,
    oauthStartResult,
    oauthStatus,
    resetOpenAIAuthState,
    startOrCheckOpenAIOAuth,
  } = useOpenAIProviderAuthState({
    authMode,
    selectedProvider,
    setLocalError,
  });
  const {
    activeProviderModelValue,
    activeProviderModels,
    filteredProviders,
    showModelSelect,
    submitDisabled,
  } = useProviderConnectDerivedState({
    apiKey,
    authMode,
    catalogProviders,
    customProviderForm,
    oauthReady,
    query,
    selectedModelId,
    selectedProvider,
    submitting,
  });

  function resetDialog() {
    setQuery("");
    setSelectedProvider(null);
    setProviderDialogStep("details");
    setAuthMode("api-key");
    resetOpenAIAuthState();
    setCallbackInput("");
    setApiKey("");
    setCustomProviderForm(emptyCustomProviderForm());
    setSelectedModelId("");
    setSubmitting(false);
    setLocalError("");
  }

  function selectProvider(provider: ProviderPickOption) {
    setSelectedProvider(provider);
    const nextAuthMode = provider.id === "openai" ? "oauth-browser" : "api-key";
    setAuthMode(nextAuthMode);
    setProviderDialogStep(provider.id === "openai" ? "options" : "details");
    resetOpenAIAuthState();
    setCallbackInput("");
    setApiKey("");
    setCustomProviderForm(
      customProviderFormFor(provider, {
        baseUrl: provider.baseUrl,
        fallbackModelId: provider.defaultModelId || "custom-profile",
      }),
    );
    setSelectedModelId(
      catalogDefaultModelForProvider(catalogProviders, provider.id) ||
        defaultModelForProvider(provider.id),
    );
    setLocalError("");
    onOpenChange(false);
  }

  function closeProviderDetails() {
    resetDialog();
  }

  function selectOpenAIAuthMode(nextMode: ProviderAuthMode) {
    resetOpenAIAuthState();
    setCallbackInput("");
    setApiKey("");
    setAuthMode(nextMode);
    setProviderDialogStep("details");
  }

  function resetAuthMode(nextMode: ProviderAuthMode) {
    resetOpenAIAuthState();
    setCallbackInput("");
    setApiKey("");
    setAuthMode(nextMode);
  }

  async function submitProvider() {
    if (!selectedProvider) return;
    const isCustom = selectedProvider.id === "custom-api";
    const { input, providerId } = providerConnectInputFromState({
      apiKey,
      authMode,
      catalogProviders,
      customProviderForm,
      selectedModelId,
      selectedProvider,
    });
    if (!providerId) {
      setLocalError("请输入 Provider ID。");
      return;
    }
    if (selectedProvider.id === "openai" && authMode !== "api-key") {
      setSubmitting(true);
      setLocalError("");
      try {
        const oauthReady = await startOrCheckOpenAIOAuth();
        if (!oauthReady) return;
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        setLocalError(message);
        setError(message);
        return;
      } finally {
        setSubmitting(false);
      }
    } else if (isCustom && !customProviderFormIsValid(customProviderForm)) {
      setLocalError("请填写完整的自定义 Provider 信息。");
      return;
    } else if (!isCustom && !apiKey.trim()) {
      setLocalError("请输入 API Key。");
      return;
    }
    setSubmitting(true);
    setLocalError("");
    setError("");
    try {
      let modelToSelect = input.modelId || "default";
      if (hasAppBridge()) {
        try {
          const refreshedCatalog = await refreshProviderModels(input);
          setCatalog(refreshedCatalog);
          modelToSelect =
            defaultModelFromCatalog(refreshedCatalog, providerId) ||
            modelToSelect;
          input.modelId = modelToSelect;
        } catch {
          // Connecting with the existing/default model is still useful if refresh fails.
        }
        const nextCatalog = await connectProvider(input);
        setCatalog(nextCatalog);
        const option = modelOptionFromCatalog(
          nextCatalog,
          providerId,
          input.modelId || modelToSelect,
        );
        resetDialog();
        await onConnected(option);
        return;
      }
      const next = connectPreviewProvider(input);
      setCatalog(next.catalog);
      setConfig(next.config);
      const option = modelOptionFromCatalog(
        next.catalog,
        providerId,
        modelToSelect,
      );
      resetDialog();
      await onConnected(option);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setLocalError(message);
      setError(message);
    } finally {
      setSubmitting(false);
    }
  }

  function handlePickerOpenChange(nextOpen: boolean) {
    onOpenChange(nextOpen);
    if (!nextOpen && !selectedProvider) resetDialog();
  }

  return {
    activeProviderModelValue,
    activeProviderModels,
    apiKey,
    authMode,
    authSuccessMessage,
    callbackInput,
    closeProviderDetails,
    customProviderForm,
    filteredProviders,
    handlePickerOpenChange,
    localError,
    oauthReady,
    oauthStarted,
    oauthStartResult,
    oauthStatus,
    providerDialogStep,
    query,
    resetAuthMode,
    selectOpenAIAuthMode,
    selectProvider,
    selectedProvider,
    setApiKey,
    setCallbackInput,
    setCustomProviderForm,
    setProviderDialogStep,
    setQuery,
    setSelectedModelId,
    showModelSelect,
    submitDisabled,
    submitProvider,
    submitting,
  };
}
