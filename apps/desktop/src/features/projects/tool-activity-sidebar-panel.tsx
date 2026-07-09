import { useCallback, useEffect, useState } from "react";

import { BuiltinBrowserSidebar } from "@/features/projects/builtin-browser-sidebar";
import { ToolActivityDetail } from "@/features/projects/tool-activity-detail";
import { ToolActivityEmptyState } from "@/features/projects/tool-activity-sidebar-empty-state";
import { ToolActivitySidebarTabs } from "@/features/projects/tool-activity-sidebar-tabs";
import type { ToolActivitySidebarProps } from "@/features/projects/tool-activity-sidebar-model";

export function ToolActivitySidebar({
  activeTabId,
  browserInitialUrls = {},
  browserReadyTokens = {},
  browserTabIds = [],
  browserVisible = true,
  tabs,
  onActiveTabChange,
  onApplyFileState,
  onBrowserReady,
  onCloseBrowser,
  onCloseTab,
  onOpenBrowser,
}: ToolActivitySidebarProps) {
  const activeBrowserTabId = browserTabIds.includes(activeTabId ?? "")
    ? activeTabId ?? ""
    : "";
  const activeTab =
    activeBrowserTabId
      ? null
      : tabs.find((tab) => tab.id === activeTabId) ?? tabs.at(-1) ?? null;
  const hasTabs = tabs.length > 0 || browserTabIds.length > 0;
  const [browserStates, setBrowserStates] = useState<
    Record<string, AivoBrowserState>
  >({});
  const handleBrowserStateChange = useCallback(
    (browserTabId: string, browserState: AivoBrowserState) => {
      setBrowserStates((currentStates) => {
        if (currentStates[browserTabId] === browserState) {
          return currentStates;
        }
        return {
          ...currentStates,
          [browserTabId]: browserState,
        };
      });
    },
    [],
  );
  const handleActiveBrowserReady = useCallback(() => {
    if (!activeBrowserTabId) return;
    onBrowserReady?.(activeBrowserTabId);
  }, [activeBrowserTabId, onBrowserReady]);

  useEffect(() => {
    const browserTabIdSet = new Set(browserTabIds);
    setBrowserStates((currentStates) => {
      const nextStates = Object.fromEntries(
        Object.entries(currentStates).filter(([browserTabId]) =>
          browserTabIdSet.has(browserTabId),
        ),
      );
      return Object.keys(nextStates).length === Object.keys(currentStates).length
        ? currentStates
        : nextStates;
    });
  }, [browserTabIds]);

  return (
    <section
      className="relative z-[70] flex h-full min-h-0 flex-col bg-background"
      data-app-no-drag
    >
      {hasTabs ? (
        <div className="flex h-9 shrink-0 items-center pl-2 pr-(--project-right-topbar-actions-width)">
          <ToolActivitySidebarTabs
            activeTabId={activeTabId}
            browserStates={browserStates}
            browserTabIds={browserTabIds}
            onActiveTabChange={onActiveTabChange}
            onCloseBrowser={onCloseBrowser}
            onCloseTab={onCloseTab}
            onOpenBrowser={onOpenBrowser}
            tabs={tabs}
          />
        </div>
      ) : null}
      {hasTabs ? (
        <div className="min-h-0 flex-1 overflow-hidden">
          {activeBrowserTabId ? (
            <BuiltinBrowserSidebar
              browserTabId={activeBrowserTabId}
              initialUrl={browserInitialUrls[activeBrowserTabId]}
              onClose={() => onCloseBrowser?.(activeBrowserTabId)}
              onReady={handleActiveBrowserReady}
              onStateChange={(browserState) =>
                handleBrowserStateChange(activeBrowserTabId, browserState)
              }
              readyToken={browserReadyTokens[activeBrowserTabId]}
              visible={browserVisible}
            />
          ) : activeTab ? (
            <ToolActivityDetail
              onApplyFileState={onApplyFileState}
              tab={activeTab}
            />
          ) : (
            <ToolActivityEmptyState onOpenBrowser={onOpenBrowser} />
          )}
        </div>
      ) : (
        <ToolActivityEmptyState onOpenBrowser={onOpenBrowser} />
      )}
    </section>
  );
}
