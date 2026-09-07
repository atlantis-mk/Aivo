import type { LoadConversationTurnsOptions } from "@/features/projects/project-conversation-turn-loader";
import { useProjectToolActivitySessionActions } from "@/features/projects/project-tool-activity-session-actions";
import {
  useProjectToolActivityRuntimeState,
  useProjectToolActivitySessionPersistence,
} from "@/features/projects/project-tool-activity-runtime-state";

export function useProjectWorkspaceToolActivityController({
  activeSessionId,
  activeSessionIdRef,
  loadConversationTurns,
}: {
  activeSessionId: string;
  activeSessionIdRef: { current: string };
  loadConversationTurns: (
    sessionId: string,
    options?: LoadConversationTurnsOptions,
  ) => Promise<void>;
}) {
  const {
    activeToolActivityTabId,
    activeToolActivityTabIdRef,
    closedToolActivityItemIdsRef,
    isRightSidebarOpen,
    isRightSidebarOpenRef,
    setActiveToolActivityTabId,
    setRightSidebarOpen,
    setToolActivityTabs,
    toolActivitySessionStatesRef,
    toolActivityTabs,
    toolActivityTabsRef,
  } = useProjectToolActivityRuntimeState();
  const {
    applyToolActivityFileState,
    closeToolActivityTab,
    mergeToolActivityFromCall,
    restoreToolActivitySessionState,
    saveCurrentToolActivitySessionState,
  } = useProjectToolActivitySessionActions({
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
  });

  useProjectToolActivitySessionPersistence({
    activeSessionId,
    activeToolActivityTabId,
    activeToolActivityTabIdRef,
    closedToolActivityItemIdsRef,
    isRightSidebarOpen,
    isRightSidebarOpenRef,
    toolActivitySessionStatesRef,
    toolActivityTabs,
    toolActivityTabsRef,
  });

  return {
    activeToolActivityTabId,
    applyToolActivityFileState,
    closeToolActivityTab,
    closedToolActivityItemIdsRef,
    isRightSidebarOpen,
    mergeToolActivityFromCall,
    restoreToolActivitySessionState,
    saveCurrentToolActivitySessionState,
    setActiveToolActivityTabId,
    setRightSidebarOpen,
    setToolActivityTabs,
    toolActivityTabs,
  };
}
