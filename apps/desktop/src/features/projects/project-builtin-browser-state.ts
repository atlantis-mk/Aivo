import { useCallback, useEffect, useRef, useState } from "react";
import type { Dispatch, SetStateAction } from "react";

import { BUILTIN_BROWSER_TAB_ID } from "@/features/projects/tool-activity-sidebar";
import { BROWSER_REVEAL_AFTER_PANEL_MS } from "@/features/projects/project-workspace-state-model";
import type { ToolActivityTab } from "@/features/projects/tool-activity-model";

export function useProjectBuiltinBrowserState({
  activeProjectPage,
  activeToolActivityTabId,
  activeToolActivityTabIdRef,
  isRightSidebarOpen,
  navigateToChat,
  setActiveToolActivityTabId,
  setRightSidebarOpen,
  toolActivityTabsRef,
}: {
  activeProjectPage: "chat" | "plugins";
  activeToolActivityTabId: string;
  activeToolActivityTabIdRef: { current: string };
  isRightSidebarOpen: boolean;
  navigateToChat: () => void;
  setActiveToolActivityTabId: Dispatch<SetStateAction<string>>;
  setRightSidebarOpen: Dispatch<SetStateAction<boolean>>;
  toolActivityTabsRef: { current: ToolActivityTab[] };
}) {
  const [isBrowserRevealReady, setBrowserRevealReady] = useState(false);
  const [builtinBrowserInitialUrls, setBuiltinBrowserInitialUrls] = useState<
    Record<string, string>
  >({});
  const [builtinBrowserReadyTokens, setBuiltinBrowserReadyTokens] = useState<
    Record<string, number>
  >({});
  const [builtinBrowserTabIds, setBuiltinBrowserTabIds] = useState<string[]>([]);
  const builtinBrowserInitialUrlsRef = useRef<Record<string, string>>({});
  const pendingBuiltinBrowserReadyRef = useRef(new Map<string, Set<() => void>>());
  const builtinBrowserTabIdsRef = useRef<string[]>([]);

  useEffect(() => {
    builtinBrowserInitialUrlsRef.current = builtinBrowserInitialUrls;
    builtinBrowserTabIdsRef.current = builtinBrowserTabIds;
  }, [builtinBrowserInitialUrls, builtinBrowserTabIds]);

  useEffect(() => {
    if (activeProjectPage !== "chat" || !isRightSidebarOpen) {
      setBrowserRevealReady(false);
      return;
    }

    setBrowserRevealReady(false);
    const timeout = window.setTimeout(() => {
      requestAnimationFrame(() => setBrowserRevealReady(true));
    }, BROWSER_REVEAL_AFTER_PANEL_MS);
    return () => window.clearTimeout(timeout);
  }, [activeProjectPage, isRightSidebarOpen]);

  useEffect(() => {
    const browser = window.aivo?.browser;
    if (!browser || builtinBrowserTabIds.length === 0) return;

    const visibleBrowserTabId =
      activeProjectPage === "chat" &&
      isRightSidebarOpen &&
      isBrowserRevealReady &&
      builtinBrowserTabIds.includes(activeToolActivityTabId)
        ? activeToolActivityTabId
        : "";

    for (const browserTabId of builtinBrowserTabIds) {
      if (browserTabId === visibleBrowserTabId) continue;
      void browser.setVisible(browserTabId, false).catch(() => undefined);
    }
  }, [
    activeProjectPage,
    activeToolActivityTabId,
    builtinBrowserTabIds,
    isBrowserRevealReady,
    isRightSidebarOpen,
  ]);

  const waitForBuiltinBrowserReady = useCallback((tabId: string) => {
    return new Promise<void>((resolve) => {
      let settled = false;
      const resolvers =
        pendingBuiltinBrowserReadyRef.current.get(tabId) ??
        new Set<() => void>();
      const finish = () => {
        if (settled) return;
        settled = true;
        window.clearTimeout(timeout);
        resolvers.delete(finish);
        if (resolvers.size === 0) {
          pendingBuiltinBrowserReadyRef.current.delete(tabId);
        }
        resolve();
      };
      const timeout = window.setTimeout(finish, 1800);
      resolvers.add(finish);
      pendingBuiltinBrowserReadyRef.current.set(tabId, resolvers);
    });
  }, []);

  const handleBuiltinBrowserReady = useCallback((tabId: string) => {
    const resolvers = pendingBuiltinBrowserReadyRef.current.get(tabId);
    if (!resolvers) return;
    for (const resolve of [...resolvers]) {
      resolve();
    }
  }, []);

  const openBuiltinBrowser = useCallback(
    (targetUrl?: string, requestedTabId?: string) => {
      const nextTabId =
        requestedTabId?.trim() ||
        (builtinBrowserTabIdsRef.current.length === 0
          ? BUILTIN_BROWSER_TAB_ID
          : `${BUILTIN_BROWSER_TAB_ID}-${Date.now().toString(36)}-${Math.random()
              .toString(36)
              .slice(2, 8)}`);
      const ready = waitForBuiltinBrowserReady(nextTabId);
      if (targetUrl) {
        setBuiltinBrowserInitialUrls((currentInitialUrls) => ({
          ...currentInitialUrls,
          [nextTabId]: targetUrl,
        }));
      }
      setBuiltinBrowserReadyTokens((currentTokens) => ({
        ...currentTokens,
        [nextTabId]: (currentTokens[nextTabId] ?? 0) + 1,
      }));
      setBuiltinBrowserTabIds((currentTabIds) =>
        currentTabIds.includes(nextTabId)
          ? currentTabIds
          : [...currentTabIds, nextTabId],
      );
      setActiveToolActivityTabId(nextTabId);
      setRightSidebarOpen(true);
      if (activeProjectPage !== "chat") {
        navigateToChat();
      }
      return ready;
    },
    [
      activeProjectPage,
      navigateToChat,
      setActiveToolActivityTabId,
      setRightSidebarOpen,
      waitForBuiltinBrowserReady,
    ],
  );

  useEffect(() => {
    const unsubscribe = window.aivo?.browser?.onOpenRequest?.((payload) => {
      const tabId = payload?.tabId?.trim() || BUILTIN_BROWSER_TAB_ID;
      return openBuiltinBrowser(payload?.url || undefined, tabId);
    });
    return () => unsubscribe?.();
  }, [openBuiltinBrowser]);

  const closeBuiltinBrowser = useCallback(
    (tabId = activeToolActivityTabIdRef.current) => {
      const browserTabId = builtinBrowserTabIdsRef.current.includes(tabId)
        ? tabId
        : builtinBrowserTabIdsRef.current.at(-1) || "";
      if (!browserTabId) return;
      const browser = window.aivo?.browser;
      if (browser) {
        void browser.close(browserTabId).catch(() => undefined);
      }
      setBuiltinBrowserTabIds((currentTabIds) => {
        const nextTabIds = currentTabIds.filter((id) => id !== browserTabId);
        setBuiltinBrowserInitialUrls((currentInitialUrls) => {
          if (!(browserTabId in currentInitialUrls)) return currentInitialUrls;
          const { [browserTabId]: _closedInitialUrl, ...nextInitialUrls } =
            currentInitialUrls;
          return nextInitialUrls;
        });
        setBuiltinBrowserReadyTokens((currentTokens) => {
          if (!(browserTabId in currentTokens)) return currentTokens;
          const { [browserTabId]: _closedReadyToken, ...nextTokens } =
            currentTokens;
          return nextTokens;
        });
        pendingBuiltinBrowserReadyRef.current.delete(browserTabId);
        setActiveToolActivityTabId((currentId) => {
          if (currentId !== browserTabId) return currentId;
          return (
            nextTabIds.at(-1) || toolActivityTabsRef.current.at(-1)?.id || ""
          );
        });
        if (
          nextTabIds.length === 0 &&
          toolActivityTabsRef.current.length === 0
        ) {
          setRightSidebarOpen(false);
        }
        return nextTabIds;
      });
    },
    [
      activeToolActivityTabIdRef,
      setActiveToolActivityTabId,
      setRightSidebarOpen,
      toolActivityTabsRef,
    ],
  );

  return {
    builtinBrowserInitialUrls,
    builtinBrowserInitialUrlsRef,
    builtinBrowserReadyTokens,
    builtinBrowserTabIds,
    builtinBrowserTabIdsRef,
    closeBuiltinBrowser,
    handleBuiltinBrowserReady,
    isBrowserRevealReady,
    openBuiltinBrowser,
    setBuiltinBrowserInitialUrls,
    setBuiltinBrowserTabIds,
  };
}
