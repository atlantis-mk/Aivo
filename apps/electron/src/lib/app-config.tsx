/* eslint-disable react-refresh/only-export-components */

import { useEffect, type ReactNode } from "react";
import { create } from "zustand";

import type { domain } from "@/types/codex-domain";
import type { CatalogState } from "@/lib/provider-catalog";
import { getPreviewAppConfig, getPreviewCatalog } from "@/lib/preview-state";
import { appNameFromConfig } from "@/lib/app-identity";
import { catalogWithCodexModels } from "@/lib/codex-model-catalog";

type AppConfigState = {
  config: domain.AppConfig | null;
  catalog: CatalogState | null;
  loading: boolean;
  error: string;
  setConfig: (config: domain.AppConfig) => void;
  setCatalog: (catalog: CatalogState) => void;
  setError: (error: string) => void;
  reload: () => Promise<void>;
};

function hasCodexDesktopBridge() {
  return Boolean(window.aivoDesktop?.codex);
}

async function getAppConfig() {
  const previewConfig = getPreviewAppConfig();
  if (!hasCodexDesktopBridge()) return previewConfig;

  try {
    const persistedConfig = await window.aivoDesktop.codex.readModelConfig();
    return {
      ...previewConfig,
      defaultModel:
        persistedConfig.model && persistedConfig.modelProvider
          ? {
              modelId: persistedConfig.model,
              providerId: persistedConfig.modelProvider,
            }
          : previewConfig.defaultModel,
      reasoningEffort:
        persistedConfig.reasoningEffort ?? previewConfig.reasoningEffort,
      serviceTier: persistedConfig.serviceTier ?? previewConfig.serviceTier,
    } as typeof previewConfig;
  } catch {
    return previewConfig;
  }
}

async function getProviderCatalog() {
  const catalog = getPreviewCatalog();
  if (!hasCodexDesktopBridge()) return catalog;

  try {
    const models = await window.aivoDesktop.codex.listCodexModels();
    return models.length > 0 ? catalogWithCodexModels(catalog, models) : catalog;
  } catch {
    return catalog;
  }
}

export const useAppConfig = create<AppConfigState>((set) => ({
  config: null,
  catalog: null,
  loading: true,
  error: "",
  setConfig: (config) => set({ config }),
  setCatalog: (catalog) => set({ catalog }),
  setError: (error) => set({ error }),
  reload: async () => {
    try {
      const [nextConfig, nextCatalog] = await Promise.all([
        getAppConfig(),
        getProviderCatalog(),
      ]);
      set({
        catalog: nextCatalog,
        config: nextConfig,
        error: "",
      });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loading: false });
    }
  },
}));

export function AppConfigProvider({ children }: { children: ReactNode }) {
  const config = useAppConfig((state) => state.config);
  const setConfig = useAppConfig((state) => state.setConfig);
  const setCatalog = useAppConfig((state) => state.setCatalog);
  const setError = useAppConfig((state) => state.setError);
  const reload = useAppConfig((state) => state.reload);

  useEffect(() => {
    document.title = appNameFromConfig(config);
  }, [config]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return children;
}

export { hasCodexDesktopBridge };
