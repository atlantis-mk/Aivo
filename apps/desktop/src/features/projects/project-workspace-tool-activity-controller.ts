import { useProjectBuiltinBrowserState } from "@/features/projects/project-builtin-browser-state";
import type { LoadConversationTurnsOptions } from "@/features/projects/project-conversation-turn-loader";
import type { ProjectWorkspacePage } from "@/features/projects/project-workspace-derived-state";
import { useProjectToolActivitySessionActions } from "@/features/projects/project-tool-activity-session-actions";
import {
  useProjectToolActivityRuntimeState,
  useProjectToolActivitySessionPersistence,
} from "@/features/projects/project-tool-activity-runtime-state";

export function useProjectWorkspaceToolActivityController({
  activeProjectPage,
  activeSessionId,
  activeSessionIdRef,
  loadConversationTurns,
  navigateToProjectChat,
}: {
  activeProjectPage: ProjectWorkspacePage;
  activeSessionId: string;
  activeSessionIdRef: { current: string };
  loadConversationTurns: (
    sessionId: string,
    options?: LoadConversationTurnsOptions,
  ) => Promise<void>;
  navigateToProjectChat: () => void;
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
  } = useProjectBuiltinBrowserState({
    activeProjectPage,
    activeToolActivityTabId,
    activeToolActivityTabIdRef,
    isRightSidebarOpen,
    navigateToChat: navigateToProjectChat,
    setActiveToolActivityTabId,
    setRightSidebarOpen,
    toolActivityTabsRef,
  });
  const {
    applyToolActivityFileState,
    closeToolActivityTab,
    mergeToolActivityFromCall,
    restoreToolActivitySessionState,
    saveCurrentToolActivitySessionState,
  } = useProjectToolActivitySessionActions({
    activeSessionIdRef,
    activeToolActivityTabIdRef,
    builtinBrowserInitialUrlsRef,
    builtinBrowserTabIdsRef,
    closedToolActivityItemIdsRef,
    isRightSidebarOpenRef,
    loadConversationTurns,
    setActiveToolActivityTabId,
    setBuiltinBrowserInitialUrls,
    setBuiltinBrowserTabIds,
    setRightSidebarOpen,
    setToolActivityTabs,
    toolActivitySessionStatesRef,
    toolActivityTabsRef,
  });

  useProjectToolActivitySessionPersistence({
    activeSessionId,
    activeToolActivityTabId,
    activeToolActivityTabIdRef,
    builtinBrowserInitialUrls,
    builtinBrowserTabIds,
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
    builtinBrowserInitialUrls,
    builtinBrowserReadyTokens,
    builtinBrowserTabIds,
    closeBuiltinBrowser,
    closeToolActivityTab,
    closedToolActivityItemIdsRef,
    handleBuiltinBrowserReady,
    isBrowserRevealReady,
    isRightSidebarOpen,
    mergeToolActivityFromCall,
    openBuiltinBrowser,
    restoreToolActivitySessionState,
    saveCurrentToolActivitySessionState,
    setActiveToolActivityTabId,
    setRightSidebarOpen,
    setToolActivityTabs,
    toolActivityTabs,
  };
}
