import type { domain } from "../../../bridge/go/models";
import { BrowserOpenURL } from "../../../bridge/runtime/runtime";
import type {
  CustomProviderForm,
  ProviderAuthMode,
  ProviderChoice,
} from "@/features/providers/provider-types";
import {
  connectedProviderConfig,
  previewConfigForAuxiliaryModel,
  providerConnectionDraft,
} from "@/features/setup/setup-provider-action-builders";
import type { ProviderValidationResult } from "@/features/setup/setup-provider-step-state";
import {
  catalogDefaultModelForProvider,
  type AppConfigWithAuxiliary,
} from "@/features/setup/setup-provider-models";
import { catalogWithCodexModels } from "@/lib/codex-model-catalog";
import { modelSelectionAfterCatalogRefresh } from "@/features/setup/setup-model-refresh-selection";
import {
  auxiliaryModelPreference,
  modelRefForProvider,
  providerConnectInputForBridge,
} from "@/features/setup/setup-provider-bridge-inputs";
import { hasAppBridge, hasCodexDesktopBridge } from "@/lib/app-config";
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
  ) {
    setSaving(true);
    setError("");
    const { input, isCustomProvider } = providerConnectionDraft({
      apiKey,
      authMode,
      catalog,
      customProvider,
      provider,
      selectedModelId,
    });
    try {
      if (
        hasCodexDesktopBridge() &&
        authMode === "api-key" &&
        input.providerId !== "openai"
      ) {
        await window.aivoDesktop.codex.configureProvider({
          apiKey: input.apiKey ?? "",
          baseUrl: input.baseUrl ?? "",
          model: input.modelId ?? "",
          name: input.name ?? input.providerId,
          providerId: input.providerId,
        });
        const backendManagedInput = {
          ...input,
          apiKey: undefined,
          apiKeyEnv: undefined,
        };
        const next = connectPreviewProvider(backendManagedInput);
        let nextCatalog = next.catalog;
        try {
          const codexModels = await window.aivoDesktop.codex.listCodexModels();
          if (codexModels.length > 0) {
            nextCatalog = catalogWithCodexModels(next.catalog, codexModels);
          }
        } catch {
          // The selected provider remains usable when the optional Codex model refresh fails.
        }
        const nextConfig = connectedProviderConfig(
          backendManagedInput,
          configAuxiliaryModel(config),
        );
        setPreviewInitialized(nextConfig);
        setCatalog(nextCatalog);
        setConfig(nextConfig);
        return true;
      }
      if (hasAppBridge()) {
        try {
          const refreshedCatalog = await refreshProviderModels(
            providerConnectInputForBridge(input),
          );
          setCatalog(refreshedCatalog);
          if (!isCustomProvider) {
            input.modelId = modelSelectionAfterCatalogRefresh(
              input.modelId,
              catalogDefaultModelForProvider(refreshedCatalog, input.providerId) ?? "",
            );
          }
        } catch {
          // Refresh is opportunistic; connecting with the configured/default model remains valid.
        }
        const nextCatalog = await connectProvider(
          providerConnectInputForBridge(input),
        );
        setCatalog(nextCatalog);
        const existingAuxiliaryModel = configAuxiliaryModel(config);
        const nextConfig = connectedProviderConfig(input, existingAuxiliaryModel);
        if (!existingAuxiliaryModel && input.modelId) {
          const savedConfig = await updateModelPreferences(
            auxiliaryModelPreference(
              modelRefForProvider(input.providerId, input.modelId),
            ),
          );
          setConfig(savedConfig as AppConfigWithAuxiliary);
        } else {
          setConfig(nextConfig);
        }
      } else {
        const next = connectPreviewProvider(input);
        let nextCatalog = next.catalog;
        if (hasCodexDesktopBridge() && input.providerId === "openai") {
          try {
            nextCatalog = catalogWithCodexModels(
              next.catalog,
              await window.aivoDesktop.codex.listCodexModels(),
            );
          } catch {
            // The selected model remains valid even if a catalog refresh fails.
          }
        }
        const nextConfig =
          !configAuxiliaryModel(config) && input.modelId
            ? previewConfigForAuxiliaryModel(
                next.config,
                modelRefForProvider(input.providerId, input.modelId),
              )
            : next.config;
        setPreviewInitialized(nextConfig);
        setCatalog(nextCatalog);
        setConfig(nextConfig);
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
        if (hasCodexDesktopBridge()) {
          if (authMode !== "oauth-browser") {
            throw new Error("当前桌面版仅支持通过浏览器登录 ChatGPT。");
          }
          const account = await window.aivoDesktop.codex.getAccount();
          if (account.authMode === "chatgpt") {
            setProviderValidated(true);
            return { completed: true };
          }
          const pending = await window.aivoDesktop.codex.login();
          return {
            completed: false,
            start: {
              providerId: "openai",
              method: "oauth-browser",
              status: "pending",
              instructions: "浏览器已打开，请完成 ChatGPT 登录。",
              expiresAt: new Date(Date.now() + 10 * 60 * 1000).toISOString(),
              id: pending.loginId,
            } as domain.ProviderAuthStartResult,
          };
        }
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

  async function saveAuxiliaryModel(providerId: string, modelId: string) {
    const auxiliaryModel = modelRefForProvider(providerId, modelId);
    try {
      if (hasAppBridge()) {
        const savedConfig = await updateModelPreferences(
          auxiliaryModelPreference(auxiliaryModel),
        );
        setConfig(savedConfig as AppConfigWithAuxiliary);
      } else {
        const nextConfig = previewConfigForAuxiliaryModel(config, auxiliaryModel);
        setPreviewInitialized(nextConfig);
        setConfig(nextConfig);
      }
      setError("");
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      return false;
    }
  }

  async function refreshProviderCatalog(input: ProviderConnectInput) {
    if (hasCodexDesktopBridge() && input.providerId === "openai") {
      try {
        const codexModels = await window.aivoDesktop.codex.listCodexModels();
        if (!catalog || codexModels.length === 0) return catalog;

        const nextCatalog = catalogWithCodexModels(catalog, codexModels);
        setCatalog(nextCatalog);
        setError("");
        return nextCatalog;
      } catch (error) {
        setError(
          error instanceof Error ? error.message : "无法读取 Codex 模型列表。",
        );
        return catalog;
      }
    }

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
    saveAuxiliaryModel,
    validateProvider,
  };
}

function configAuxiliaryModel(config: domain.AppConfig | null) {
  return (config as AppConfigWithAuxiliary | null)?.auxiliaryModel;
}

async function openExternalURL(url: string) {
  await BrowserOpenURL(url);
}
