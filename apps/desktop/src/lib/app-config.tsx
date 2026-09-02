/* eslint-disable react-refresh/only-export-components */

import { useEffect, type ReactNode } from "react";
import { create } from "zustand";

import { EventsOn } from "../../bridge/runtime/runtime";
import type { domain } from "../../bridge/go/models";
import type { CatalogState } from "@/lib/provider-catalog";
import { getPreviewAppConfig, getPreviewCatalog } from "@/lib/preview-state";
import { appNameFromConfig } from "@/lib/app-identity";
import { catalogWithCodexModels } from "@/lib/codex-model-catalog";
import { getAppConfig as loadAppConfig, getProviderCatalog as loadProviderCatalog } from "@/services/aivo";

type AppConfigState = {
  config: domain.AppConfig | null;
  catalog: CatalogState | null;
  loading: boolean;
  error: string;
  bridgeReady: boolean;
  bridgeResolved: boolean;
  setConfig: (config: domain.AppConfig) => void;
  setCatalog: (catalog: CatalogState) => void;
  setError: (error: string) => void;
  setBridgeReady: (ready: boolean) => void;
  setBridgeResolved: (resolved: boolean) => void;
  reload: () => Promise<void>;
};

function hasAppBridge() {
  return Boolean(window.aivo?.invoke);
}

/**
 * The migrated renderer can run against the current Electron/Codex shell without
 * loading the retired Go desktop bridge. Keep this separate from hasAppBridge:
 * the latter deliberately remains the compatibility check for legacy services.
 */
function hasCodexDesktopBridge() {
  return Boolean(window.aivoDesktop?.codex);
}

const BRIDGE_WAIT_TIMEOUT_MS = 3000;
const BRIDGE_POLL_INTERVAL_MS = 100;

async function waitForAppBridge(timeoutMs = BRIDGE_WAIT_TIMEOUT_MS) {
  if (hasAppBridge()) return true;

  const startedAt = Date.now();
  return new Promise<boolean>((resolve) => {
    const timer = window.setInterval(() => {
      if (hasAppBridge()) {
        window.clearInterval(timer);
        resolve(true);
        return;
      }
      if (Date.now() - startedAt >= timeoutMs) {
        window.clearInterval(timer);
        resolve(false);
      }
    }, BRIDGE_POLL_INTERVAL_MS);
  });
}

async function getAppConfig() {
  if (!hasAppBridge()) return getPreviewAppConfig();
  return loadAppConfig();
}

async function getProviderCatalog() {
  if (!hasAppBridge()) {
    const catalog = getPreviewCatalog();
    if (!hasCodexDesktopBridge()) return catalog;

    try {
      const account = await window.aivoDesktop.codex.getAccount();
      if (account.authMode !== "chatgpt") return catalog;
      return catalogWithCodexModels(
        catalog,
        await window.aivoDesktop.codex.listModels(),
      );
    } catch {
      return catalog;
    }
  }
  return loadProviderCatalog();
}

export const useAppConfig = create<AppConfigState>((set) => ({
  config: null,
  catalog: null,
  loading: true,
  error: "",
  bridgeReady: hasAppBridge(),
  // The migrated frontend runs with local preview state in the Electron/Codex
  // shell. Do not spend three seconds waiting for the retired Go bridge there.
  bridgeResolved: hasAppBridge() || hasCodexDesktopBridge(),
  setConfig: (config) => set({ config }),
  setCatalog: (catalog) => set({ catalog }),
  setError: (error) => set({ error }),
  setBridgeReady: (bridgeReady) => set({ bridgeReady }),
  setBridgeResolved: (bridgeResolved) => set({ bridgeResolved }),
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
  const bridgeReady = useAppConfig((state) => state.bridgeReady);
  const bridgeResolved = useAppConfig((state) => state.bridgeResolved);
  const setBridgeReady = useAppConfig((state) => state.setBridgeReady);
  const setBridgeResolved = useAppConfig((state) => state.setBridgeResolved);
  const setConfig = useAppConfig((state) => state.setConfig);
  const setCatalog = useAppConfig((state) => state.setCatalog);
  const setError = useAppConfig((state) => state.setError);
  const reload = useAppConfig((state) => state.reload);

  useEffect(() => {
    document.title = appNameFromConfig(config);
  }, [config]);

  useEffect(() => {
    if (bridgeReady || hasCodexDesktopBridge()) {
      setBridgeResolved(true);
      return;
    }

    const startedAt = Date.now();
    const timer = window.setInterval(() => {
      if (hasAppBridge()) {
        setBridgeReady(true);
        setBridgeResolved(true);
        window.clearInterval(timer);
        return;
      }

      if (Date.now() - startedAt >= BRIDGE_WAIT_TIMEOUT_MS) {
        setBridgeResolved(true);
        window.clearInterval(timer);
      }
    }, BRIDGE_POLL_INTERVAL_MS);

    return () => window.clearInterval(timer);
  }, [bridgeReady, setBridgeReady, setBridgeResolved]);

  useEffect(() => {
    if (!bridgeResolved) return;
    void Promise.resolve().then(reload);
  }, [bridgeResolved, bridgeReady, reload]);

  useEffect(() => {
    if (!bridgeReady) return;
    return EventsOn("config.changed", (nextConfig: domain.AppConfig) => {
      setConfig(nextConfig);
      setError("");
    });
  }, [bridgeReady, setConfig, setError]);

  useEffect(() => {
    if (!bridgeReady) return;
    return EventsOn("catalog.changed", (nextCatalog: CatalogState) => {
      setCatalog(nextCatalog);
      setError("");
    });
  }, [bridgeReady, setCatalog, setError]);

  return children;
}

export { hasAppBridge, hasCodexDesktopBridge, waitForAppBridge };
