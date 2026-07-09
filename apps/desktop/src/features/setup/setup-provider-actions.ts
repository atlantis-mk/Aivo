import type { domain } from "../../../bridge/go/models";
import { BrowserOpenURL } from "../../../bridge/runtime/runtime";
import type {
  CustomProviderForm,
  ProviderAuthMode,
  ProviderChoice,
} from "@/features/providers/provider-types";
import {
  connectedProviderConfig,
  modelPreferencesForProvider,
  previewConfigForConnectedAccountModels,
  providerConnectionDraft,
} from "@/features/setup/setup-provider-action-builders";
import type { ProviderValidationResult } from "@/features/setup/setup-provider-step-state";
import {
  catalogDefaultModelForProvider,
  updateCatalogDefaultModel,
  type AppConfigWithAuxiliary,
} from "@/features/setup/setup-provider-models";
import {
  auxiliaryModelPreference,
  modelPreferencesWithAuxiliary,
  providerConnectInputForBridge,
} from "@/features/setup/setup-provider-bridge-inputs";
import { hasAppBridge } from "@/lib/app-config";
import {
  completePreviewOpenAIBrowserAuth,
  connectPreviewProvider,
  deletePreviewProviderAccount,
  setPreviewInitialized,
  startPreviewOpenAIBrowserAuth,
} from "@/lib/preview-state";
import type { CatalogState, ProviderConnectInput } from "@/lib/provider-catalog";
import {
  connectProvider,
  deleteProviderAccount,
  refreshProviderModels,
  startProviderAuth,
  updateModelPreferences,
} from "@/services/aivo";

type UseSetupProviderActionsParams = {
  catalog: CatalogState | null;
  config: domain.AppConfig | null;
  setCatalog: (catalog: CatalogState) => void;
  setConfig: (config: domain.AppConfig) => void;
  setError: (error: string) => void;
  setProviderValidated: (validated: boolean) => void;
  setSaving: (saving: boolean) => void;
};

export function useSetupProviderActions({
  catalog,
  config,
  setCatalog,
  setConfig,
  setError,
  setProviderValidated,
  setSaving,
}: UseSetupProviderActionsParams) {
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
    const { auxiliaryModel, input, isCustomProvider } = providerConnectionDraft({
      apiKey,
      authMode,
      catalog,
      customProvider,
      provider,
      selectedAuxiliaryModelId,
      selectedModelId,
    });
    try {
      if (hasAppBridge()) {
        try {
          const refreshedCatalog = await refreshProviderModels(
            providerConnectInputForBridge(input),
          );
          setCatalog(refreshedCatalog);
          if (!isCustomProvider) {
            input.modelId =
              catalogDefaultModelForProvider(refreshedCatalog, input.providerId) ||
              input.modelId;
          }
        } catch {
          // Refresh is opportunistic; connecting with the configured/default model remains valid.
        }
        const nextCatalog = await connectProvider(
          providerConnectInputForBridge(input),
        );
        setCatalog(nextCatalog);
        const nextConfig = connectedProviderConfig(input, auxiliaryModel);
        if (auxiliaryModel) {
          const savedConfig = await updateModelPreferences(
            auxiliaryModelPreference(auxiliaryModel),
          );
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
    if (
      provider.id === "openai" &&
      (authMode === "oauth-browser" || authMode === "oauth-headless")
    ) {
      try {
        if (hasAppBridge()) {
          const pending = await startProviderAuth({
            providerId: "openai",
            method: authMode,
          });
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

  async function saveConnectedAccountModels(
    providerId: string,
    modelId: string,
    auxiliaryModelId: string,
  ) {
    const { auxiliaryModel, model } = modelPreferencesForProvider(
      providerId,
      modelId,
      auxiliaryModelId,
    );
    try {
      if (hasAppBridge()) {
        const savedConfig = await updateModelPreferences(
          modelPreferencesWithAuxiliary(model, auxiliaryModel),
        );
        setConfig(savedConfig as AppConfigWithAuxiliary);
      } else {
        const nextConfig = previewConfigForConnectedAccountModels({
          auxiliaryModel,
          config,
          model,
        });
        setPreviewInitialized(nextConfig);
        setConfig(nextConfig);
      }
      setCatalog(updateCatalogDefaultModel(catalog, providerId, modelId));
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function refreshProviderCatalog(input: ProviderConnectInput) {
    if (!hasAppBridge()) return catalog;
    try {
      const nextCatalog = await refreshProviderModels(
        providerConnectInputForBridge(input),
      );
      setCatalog(nextCatalog);
      setError("");
      return nextCatalog;
    } catch {
      return catalog;
    }
  }

  return {
    completeProviderDialog,
    refreshProviderCatalog,
    removeProviderAccount,
    saveConnectedAccountModels,
    validateProvider,
  };
}

async function openExternalURL(url: string) {
  await BrowserOpenURL(url);
}
