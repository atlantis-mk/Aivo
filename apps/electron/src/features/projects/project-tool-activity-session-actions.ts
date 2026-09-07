import { useCallback, useEffect } from "react";
import type { Dispatch, SetStateAction } from "react";
import { toast } from "sonner";

import { onDesktopEvent } from "@/lib/desktop-events";
import {
  buildToolActivitySessionState,
  closeToolActivityTabState,
  resolveActiveToolActivityTabId,
  toolActivityToolCallKey,
  visibleToolActivityTabs,
  type ToolActivitySessionState,
} from "@/features/projects/project-tool-activity-session-model";
import type { LoadConversationTurnsOptions } from "@/features/projects/project-conversation-turn-loader";
import { annotateToolActivityTabsWithTurnDiff } from "@/features/projects/project-tool-activity-turn-diff";
import { SHOULD_AUTO_OPEN_TOOL_ACTIVITY_SIDEBAR } from "@/features/projects/project-workspace-state-model";
import {
  annotateToolActivityTabsWithFileStates,
  appendShellOutputToTabs,
  toolActivityTabsFromToolCall,
  toolActivityTabsFromToolCalls,
  upsertToolActivityTabs,
  type ToolActivityFileTab,
  type ToolActivityTab,
} from "@/features/projects/tool-activity-model";
import type { domain } from "@/types/codex-domain";

export function useProjectToolActivitySessionActions({
  activeSessionIdRef,
  activeToolActivityTabIdRef,
  closedToolActivityItemIdsRef,
  isRightSidebarOpenRef,
  loadConversationTurns,
  setActiveToolActivityTabId,
  setRightSidebarOpen,
  setToolActivityTabs,
  toolActivitySessionStatesRef,
  toolActivityTabsRef,
}: {
  activeSessionIdRef: { current: string };
  activeToolActivityTabIdRef: { current: string };
  closedToolActivityItemIdsRef: { current: Set<string> };
  isRightSidebarOpenRef: { current: boolean };
  loadConversationTurns: (
    sessionId: string,
    options?: LoadConversationTurnsOptions,
  ) => Promise<void>;
  setActiveToolActivityTabId: Dispatch<SetStateAction<string>>;
  setRightSidebarOpen: Dispatch<SetStateAction<boolean>>;
  setToolActivityTabs: Dispatch<SetStateAction<ToolActivityTab[]>>;
  toolActivitySessionStatesRef: {
    current: Map<string, ToolActivitySessionState>;
  };
  toolActivityTabsRef: { current: ToolActivityTab[] };
}) {
  function saveCurrentToolActivitySessionState(
    sessionId = activeSessionIdRef.current,
  ) {
    if (!sessionId) return;
    toolActivitySessionStatesRef.current.set(
      sessionId,
      buildToolActivitySessionState({
        activeTabId: activeToolActivityTabIdRef.current,
        closedItemIds: closedToolActivityItemIdsRef.current,
        isOpen: isRightSidebarOpenRef.current,
        tabs: toolActivityTabsRef.current,
      }),
    );
  }

  function restoreToolActivitySessionState(sessionId: string) {
    const savedState = toolActivitySessionStatesRef.current.get(sessionId);
    const tabs = upsertToolActivityTabs([], savedState?.tabs ?? []);
    closedToolActivityItemIdsRef.current = new Set(
      savedState?.closedItemIds ?? [],
    );
    setToolActivityTabs(tabs);
    setActiveToolActivityTabId(
      resolveActiveToolActivityTabId({
        activeTabId: savedState?.activeTabId || "",
        tabs,
      }),
    );
    setRightSidebarOpen(
      Boolean(
        SHOULD_AUTO_OPEN_TOOL_ACTIVITY_SIDEBAR &&
          savedState?.isOpen &&
          tabs.length > 0,
      ),
    );
  }

  const mergeToolActivityFromCall = useCallback(
    (toolCall: domain.ToolCall) => {
      const nextTabs = visibleToolActivityTabs(
        toolActivityTabsFromToolCall(toolCall),
        closedToolActivityItemIdsRef.current,
      );
      if (nextTabs.length === 0) return;
      setToolActivityTabs((currentTabs) =>
        upsertToolActivityTabs(currentTabs, nextTabs),
      );
      setActiveToolActivityTabId(nextTabs.at(-1)?.id ?? "");
      if (SHOULD_AUTO_OPEN_TOOL_ACTIVITY_SIDEBAR) {
        setRightSidebarOpen(true);
      }
    },
    [
      closedToolActivityItemIdsRef,
      setActiveToolActivityTabId,
      setRightSidebarOpen,
      setToolActivityTabs,
    ],
  );

  async function refreshToolActivityTabs(
    sessionId = activeSessionIdRef.current,
  ) {
    void sessionId;
  }

  async function applyToolActivityFileState(
    tab: ToolActivityFileTab,
    targetState: "before" | "after",
  ) {
    void tab;
    void targetState;
  }

  const closeToolActivityTab = useCallback(
    (tabId: string) => {
      setToolActivityTabs((currentTabs) => {
        const nextState = closeToolActivityTabState({
          tabId,
          tabs: currentTabs,
        });
        for (const key of nextState.closedKeys) {
          closedToolActivityItemIdsRef.current.add(key);
        }
        setActiveToolActivityTabId((currentId) => {
          return nextState.nextActiveTabId(currentId);
        });
        if (nextState.shouldCloseSidebar) {
          setRightSidebarOpen(false);
        }
        return nextState.nextTabs;
      });
    },
    [
      closedToolActivityItemIdsRef,
      setActiveToolActivityTabId,
      setRightSidebarOpen,
      setToolActivityTabs,
    ],
  );

  return {
    applyToolActivityFileState,
    closeToolActivityTab,
    mergeToolActivityFromCall,
    restoreToolActivitySessionState,
    saveCurrentToolActivitySessionState,
  };
}
