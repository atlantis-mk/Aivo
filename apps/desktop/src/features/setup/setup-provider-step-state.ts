import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";

import type { domain } from "../../../bridge/go/models";
import { EventsOn } from "../../../bridge/runtime/runtime";
import {
  customProviderFormFor,
  customProviderFormIsValid,
  emptyCustomProviderForm,
  isCustomProviderChoice,
} from "@/features/providers/custom-provider-form";
import { normalizeProviderAuthUpdatedPayload } from "@/features/providers/provider-events";
import type {
  CustomProviderForm,
  ProviderAuthMode,
  ProviderChoice,
  ProviderDialogStep,
} from "@/features/providers/provider-types";
import {
  canRefreshProviderModels,
  catalogDefaultModelForProvider,
  defaultModelForProvider,
  providerRefreshInput,
} from "@/features/setup/setup-provider-models";
import { modelSelectionAfterCatalogRefresh } from "@/features/setup/setup-model-refresh-selection";
import {
  modelsForActiveProvider,
  oauthReady,
  oauthStatusFromAuthStart,
  selectedModelIdForProvider,
  shouldShowModelSelect,
  successfulOpenAIStatus,
} from "@/features/setup/setup-provider-step-selectors";
import { hasAppBridge } from "@/lib/app-config";
import type {
  CatalogState,
  ProviderAccountInfo,
  ProviderConnectInput,
} from "@/lib/provider-catalog";

export type ProviderValidationResult = {
  completed: boolean;
  start?: domain.ProviderAuthStartResult;
};

type UseSetupProviderStepStateParams = {
  catalog: CatalogState | null;
  onContinue: (
    provider: ProviderChoice,
    authMode: ProviderAuthMode,
    apiKey?: string,
    customProvider?: CustomProviderForm,
    selectedModelId?: string,
  ) => Promise<boolean>;
  onRefreshModels: (input: ProviderConnectInput) => Promise<CatalogState | null>;
  onResetValidation: () => void;
  onValidate: (
    provider: ProviderChoice,
    authMode: ProviderAuthMode,
    callbackInput?: string,
    apiKey?: string,
  ) => Promise<ProviderValidationResult>;
  providerValidated: boolean;
  saving: boolean;
};

export function useSetupProviderStepState({
  catalog,
  onContinue,
  onRefreshModels,
  onResetValidation,
  onValidate,
  providerValidated,
  saving,
}: UseSetupProviderStepStateParams) {
  const [activeProvider, setActiveProvider] = useState<ProviderChoice | null>(null);
  const [providerDialogStep, setProviderDialogStep] =
    useState<ProviderDialogStep>("details");
  const [authMode, setAuthMode] = useState<ProviderAuthMode>("api-key");
  const [oauthStarted, setOauthStarted] = useState(false);
  const [oauthStartResult, setOauthStartResult] =
    useState<domain.ProviderAuthStartResult | null>(null);
  const [oauthStatus, setOauthStatus] =
    useState<domain.ProviderAuthStatus | null>(null);
  const [callbackInput, setCallbackInput] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [customProviderForm, setCustomProviderForm] = useState<CustomProviderForm>(
    () => emptyCustomProviderForm(),
  );
  const [otherProviderPickerOpen, setOtherProviderPickerOpen] = useState(false);
  const [otherProviderSearch, setOtherProviderSearch] = useState("");
  const [selectedModelId, setSelectedModelId] = useState("");
  const [settingsAccount, setSettingsAccount] =
    useState<ProviderAccountInfo | null>(null);
  const [authSuccessMessage, setAuthSuccessMessage] = useState("");
  const authSuccessNotifiedRef = useRef(false);

  const activeProviderModels = modelsForActiveProvider(catalog, activeProvider);

  function resetTransientAuthState() {
    setOauthStarted(false);
    setOauthStartResult(null);
    setOauthStatus(null);
    setAuthSuccessMessage("");
    authSuccessNotifiedRef.current = false;
    setCallbackInput("");
    setApiKey("");
  }

  function effectiveSelectedModelId() {
    return selectedModelIdForProvider({
      catalog,
      models: activeProviderModels,
      provider: activeProvider,
      selectedModelId,
    });
  }

  const refreshModelsForProvider = useCallback(
    async (
      provider: ProviderChoice,
      form: CustomProviderForm,
      mode: ProviderAuthMode,
      key: string,
      force = false,
    ) => {
      if (!hasAppBridge()) return;
      if (!force && !canRefreshProviderModels(provider, form, mode, key, catalog)) {
        return;
      }
      const input = providerRefreshInput(provider, form, mode, key, selectedModelId);
      const nextCatalog = await onRefreshModels(input);
      const refreshedDefault = catalogDefaultModelForProvider(
        nextCatalog,
        input.providerId,
      );
      if (refreshedDefault) {
        setSelectedModelId((current) =>
          modelSelectionAfterCatalogRefresh(current, refreshedDefault),
        );
      }
    },
    [catalog, onRefreshModels, selectedModelId],
  );

  function openProvider(provider: ProviderChoice) {
    if (provider.opensProviderPicker) {
      setOtherProviderSearch("");
      setOtherProviderPickerOpen(true);
      return;
    }
    onResetValidation();
    resetTransientAuthState();
    const nextForm = customProviderFormFor(provider);
    const nextAuthMode = provider.id === "openai" ? "oauth-browser" : "api-key";
    setCustomProviderForm(nextForm);
    setAuthMode(nextAuthMode);
    const nextModelId =
      catalogDefaultModelForProvider(catalog, provider.id) ||
      defaultModelForProvider(provider.id, catalog);
    setSelectedModelId(nextModelId);
    setProviderDialogStep(provider.id === "openai" ? "options" : "details");
    setActiveProvider(provider);
    void refreshModelsForProvider(provider, nextForm, nextAuthMode, "");
  }

  const closeProvider = useCallback(() => {
    onResetValidation();
    resetTransientAuthState();
    setCustomProviderForm(emptyCustomProviderForm());
    setSelectedModelId("");
    setProviderDialogStep("details");
    setActiveProvider(null);
  }, [onResetValidation]);

  function selectOtherProvider(provider: ProviderChoice) {
    setOtherProviderPickerOpen(false);
    openProvider(provider);
  }

  function selectOpenAIAuthMode(nextMode: ProviderAuthMode) {
    onResetValidation();
    resetTransientAuthState();
    setAuthMode(nextMode);
    setProviderDialogStep("details");
    if (activeProvider) {
      void refreshModelsForProvider(activeProvider, customProviderForm, nextMode, "");
    }
  }

  function resetAuthMode(nextMode: ProviderAuthMode) {
    onResetValidation();
    resetTransientAuthState();
    setAuthMode(nextMode);
    if (activeProvider) {
      void refreshModelsForProvider(activeProvider, customProviderForm, nextMode, "");
    }
  }

  const markOpenAIAuthorized = useCallback(
    (provider: ProviderChoice) => {
      setAuthSuccessMessage("OpenAI 授权已完成");
      setOauthStatus(
        (current) => successfulOpenAIStatus(current, authMode),
      );
      if (!authSuccessNotifiedRef.current) {
        authSuccessNotifiedRef.current = true;
        toast.success("OpenAI 授权已完成");
      }
      void refreshModelsForProvider(provider, customProviderForm, authMode, apiKey, true);
    },
    [apiKey, authMode, customProviderForm, refreshModelsForProvider],
  );

  async function validateActiveProvider() {
    if (!activeProvider) return { completed: false } satisfies ProviderValidationResult;
    const result = await onValidate(
      activeProvider,
      authMode,
      oauthStarted ? callbackInput : undefined,
      apiKey,
    );
    if (result.start) {
      setOauthStartResult(result.start);
      setOauthStatus(oauthStatusFromAuthStart(result.start));
    }
    if (
      activeProvider.id === "openai" &&
      (authMode === "oauth-browser" || authMode === "oauth-headless") &&
      !result.completed
    ) {
      setOauthStarted(true);
    }
    return result;
  }

  async function completeActiveProvider() {
    if (!activeProvider) return;
    if (
      isCustomProviderChoice(activeProvider) &&
      !customProviderFormIsValid(customProviderForm)
    ) {
      return;
    }
    const completed = await onContinue(
      activeProvider,
      authMode,
      apiKey,
      isCustomProviderChoice(activeProvider) ? customProviderForm : undefined,
      effectiveSelectedModelId(),
    );
    if (completed) closeProvider();
  }

  async function submitActiveProvider() {
    if (!activeProvider) return;
    if (
      activeProvider.id === "openai" &&
      (authMode === "oauth-browser" || authMode === "oauth-headless")
    ) {
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
    if (isCustomProviderChoice(activeProvider)) {
      return !customProviderFormIsValid(customProviderForm);
    }
    if (authMode === "api-key") return !apiKey.trim();
    if (authMode === "oauth-browser" && oauthStarted && !hasAppBridge()) {
      return !callbackInput.trim();
    }
    return false;
  }

  useEffect(() => {
    if (!hasAppBridge()) return;
    return EventsOn("provider_auth.updated", (...payloads: unknown[]) => {
      const status =
        normalizeProviderAuthUpdatedPayload<domain.ProviderAuthStatus>(payloads);
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

  const activeProviderModelValue = effectiveSelectedModelId();
  const oauthIsReady = oauthReady(providerValidated, oauthStatus);
  const showModelSelect = shouldShowModelSelect({
    authMode,
    models: activeProviderModels,
    oauthStatus,
    provider: activeProvider,
    providerValidated,
  });

  return {
    activeProvider,
    activeProviderModels,
    activeProviderModelValue,
    apiKey,
    authMode,
    authSuccessMessage,
    callbackInput,
    closeProvider,
    customProviderForm,
    oauthReady: oauthIsReady,
    oauthStarted,
    oauthStartResult,
    oauthStatus,
    openProvider,
    otherProviderPickerOpen,
    otherProviderSearch,
    providerDialogStep,
    resetAuthMode,
    selectOpenAIAuthMode,
    selectOtherProvider,
    setApiKey,
    setCallbackInput,
    setCustomProviderForm,
    setOtherProviderPickerOpen,
    setOtherProviderSearch,
    setProviderDialogStep,
    setSelectedModelId,
    setSettingsAccount,
    settingsAccount,
    showModelSelect,
    submitActiveProvider,
    submitDisabled: submitDisabled(),
  };
}
