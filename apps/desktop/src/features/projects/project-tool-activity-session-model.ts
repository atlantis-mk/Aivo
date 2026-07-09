import type { ToolActivityTab } from "@/features/projects/tool-activity-model";

export type ToolActivitySessionState = {
  activeTabId: string;
  browserInitialUrls: Record<string, string>;
  browserTabIds: string[];
  closedItemIds: string[];
  isOpen: boolean;
  tabs: ToolActivityTab[];
};

export function buildToolActivitySessionState({
  activeTabId,
  browserInitialUrls,
  browserTabIds,
  closedItemIds,
  isOpen,
  tabs,
}: {
  activeTabId: string;
  browserInitialUrls: Record<string, string>;
  browserTabIds: string[];
  closedItemIds: Set<string>;
  isOpen: boolean;
  tabs: ToolActivityTab[];
}): ToolActivitySessionState {
  const visibleTabs = visibleToolActivityTabs(tabs, closedItemIds);
  return {
    activeTabId: resolveActiveToolActivityTabId({
      activeTabId,
      browserTabIds,
      tabs: visibleTabs,
    }),
    browserInitialUrls,
    browserTabIds,
    closedItemIds: [...closedItemIds],
    isOpen: isOpen && (visibleTabs.length > 0 || browserTabIds.length > 0),
    tabs: visibleTabs,
  };
}

export function visibleToolActivityTabs(
  tabs: ToolActivityTab[],
  closedItemIds: Set<string>,
) {
  return tabs.filter((tab) => !toolActivityTabIsClosed(tab, closedItemIds));
}

export function resolveActiveToolActivityTabId({
  activeTabId,
  browserTabIds,
  tabs,
}: {
  activeTabId: string;
  browserTabIds: string[];
  tabs: ToolActivityTab[];
}) {
  return tabs.some((tab) => tab.id === activeTabId) ||
    browserTabIds.includes(activeTabId)
    ? activeTabId
    : browserTabIds.at(-1) || tabs.at(-1)?.id || "";
}

export function closeToolActivityTabState({
  browserTabIds,
  tabId,
  tabs,
}: {
  browserTabIds: string[];
  tabId: string;
  tabs: ToolActivityTab[];
}) {
  const closedTab = tabs.find((tab) => tab.id === tabId);
  const nextTabs = tabs.filter((tab) => tab.id !== tabId);
  return {
    closedKeys: closedTab ? toolActivityCloseKeys(closedTab) : [],
    nextActiveTabId: (currentId: string) =>
      currentId === tabId
        ? browserTabIds.at(-1) || nextTabs.at(-1)?.id || ""
        : currentId,
    nextTabs,
    shouldCloseSidebar: nextTabs.length === 0 && browserTabIds.length === 0,
  };
}

export function toolActivityTabIsClosed(
  tab: ToolActivityTab,
  closedItemIds: Set<string>,
) {
  return toolActivityCloseKeys(tab).some((key) => closedItemIds.has(key));
}

export function toolActivityCloseKeys(tab: ToolActivityTab) {
  if (tab.kind === "command") {
    const toolCallIds = [
      tab.toolCallId,
      ...tab.entries.map((entry) => entry.toolCallId),
    ].filter(Boolean);
    return [...new Set(toolCallIds.map(toolActivityToolCallKey))];
  }
  return [toolActivityTabKey(tab.id)];
}

export function toolActivityToolCallKey(toolCallId: string) {
  return `tool-call:${toolCallId}`;
}

function toolActivityTabKey(tabId: string) {
  return `tab:${tabId}`;
}
