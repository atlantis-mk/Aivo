import { useCallback, useEffect } from "react";
import type { Dispatch, SetStateAction } from "react";
import { toast } from "sonner";

import { EventsOn } from "../../../bridge/runtime/runtime";
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
import { normalizeShellOutputPayload } from "@/features/projects/project-event-payloads";
import { hasAppBridge } from "@/lib/app-config";
import {
  applySessionTurnFileState,
  getSessionTurnDiff,
  listSessionToolCalls,
} from "@/services/aivo";
import type { domain } from "../../../bridge/go/models";

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
      if (toolCall.turnId && toolCall.status === "success" && hasAppBridge()) {
        void getSessionTurnDiff({
          sessionId: toolCall.sessionId,
          turnId: toolCall.turnId,
        })
          .then((diff) => {
            setToolActivityTabs((currentTabs) =>
              annotateToolActivityTabsWithFileStates(
                currentTabs,
                diff.files,
              ),
            );
          })
          .catch(() => undefined);
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
    if (!hasAppBridge() || !sessionId) return;
    const toolCalls = (await listSessionToolCalls(sessionId).catch(
      () => [] as domain.ToolCall[],
    )) ?? [];
    const baseTabs = visibleToolActivityTabs(
      toolActivityTabsFromToolCalls(toolCalls),
      closedToolActivityItemIdsRef.current,
    );
    const tabs = await annotateToolActivityTabsWithTurnDiff(sessionId, baseTabs);
    setToolActivityTabs(tabs);
    setActiveToolActivityTabId((currentId) =>
      resolveActiveToolActivityTabId({
        activeTabId: currentId,
        tabs,
      }),
    );
    setRightSidebarOpen((current) => current && tabs.length > 0);
  }

  async function applyToolActivityFileState(
    tab: ToolActivityFileTab,
    targetState: "before" | "after",
  ) {
    const sessionId = activeSessionIdRef.current;
    if (!hasAppBridge() || !sessionId || !tab.turnId) return;
    try {
      await applySessionTurnFileState({
        sessionId,
        turnId: tab.turnId,
        toolCallId: tab.toolCallId,
        path: tab.relativePath || tab.path,
        targetState,
      });
      await refreshToolActivityTabs(sessionId);
      await loadConversationTurns(sessionId);
      toast.success(targetState === "before" ? "已回滚文件改动" : "已恢复文件改动");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  }

  useEffect(() => {
    if (!hasAppBridge()) return;
    return EventsOn("shell.output", (...payloads: unknown[]) => {
      const payload = normalizeShellOutputPayload(payloads);
      if (
        !payload.toolCallId ||
        payload.sessionId !== activeSessionIdRef.current
      ) {
        return;
      }
      if (
        closedToolActivityItemIdsRef.current.has(
          toolActivityToolCallKey(payload.toolCallId),
        )
      ) {
        return;
      }
      setToolActivityTabs((currentTabs) =>
        appendShellOutputToTabs(currentTabs, payload),
      );
      setActiveToolActivityTabId(
        `command:shell:${payload.sessionId || "current"}`,
      );
      if (SHOULD_AUTO_OPEN_TOOL_ACTIVITY_SIDEBAR) {
        setRightSidebarOpen(true);
      }
    });
  }, [
    activeSessionIdRef,
    closedToolActivityItemIdsRef,
    setActiveToolActivityTabId,
    setRightSidebarOpen,
    setToolActivityTabs,
  ]);

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
