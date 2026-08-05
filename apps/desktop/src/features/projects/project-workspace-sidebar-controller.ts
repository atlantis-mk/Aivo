import type { Dispatch, SetStateAction } from "react";

import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";
import type { LoadConversationTurnsOptions } from "@/features/projects/project-conversation-turn-loader";
import { useProjectSidebarActions } from "@/features/projects/project-sidebar-actions";
import { useProjectSidebarConversationState } from "@/features/projects/project-workspace-derived-state";
import type { ToolActivityTab } from "@/features/projects/tool-activity-model";
import type { domain } from "../../../bridge/go/models";

type StringListUpdater = string[] | ((current: string[]) => string[]);

export function useProjectWorkspaceSidebarController({
  activeSessionIdRef,
  archivedConversationIds,
  captureComposerTransitionStart,
  clearPendingPermissionCountForSession,
  closedToolActivityItemIdsRef,
  hasTurns,
  recentProjects,
  loadConversationTurns,
  refreshPendingPermissionRequests,
  resetConversationScroll,
  restoreToolActivitySessionState,
  saveCurrentToolActivitySessionState,
  selectedProjectPath,
  sessions,
  setActiveSessionId,
  setActiveToolActivityTabId,
  setArchivedConversationIds,
  setConversationRunning,
  setOpeningConversationFromEmpty,
  setPinnedConversationIds,
  setPrompt,
  setRecentProjects,
  setRevealingHistoryConversation,
  setRightSidebarOpen,
  setSelectedProjectPath,
  setSessions,
  setToolActivityTabs,
  setTurns,
  sidebarConversationSelectionRef,
}: {
  activeSessionIdRef: { current: string };
  archivedConversationIds: string[];
  captureComposerTransitionStart: () => void;
  clearPendingPermissionCountForSession: (sessionId: string) => void;
  closedToolActivityItemIdsRef: { current: Set<string> };
  hasTurns: boolean;
  recentProjects: domain.AssistantProject[];
  loadConversationTurns: (
    sessionId: string,
    options?: LoadConversationTurnsOptions,
  ) => Promise<void>;
  refreshPendingPermissionRequests: (sessionId?: string) => Promise<void>;
  resetConversationScroll: () => void;
  restoreToolActivitySessionState: (sessionId: string) => void;
  saveCurrentToolActivitySessionState: () => void;
  selectedProjectPath: string;
  sessions: domain.Session[];
  setActiveSessionId: Dispatch<SetStateAction<string>>;
  setActiveToolActivityTabId: Dispatch<SetStateAction<string>>;
  setArchivedConversationIds: (updater: StringListUpdater) => void;
  setConversationRunning: (sessionId: string, running: boolean) => void;
  setOpeningConversationFromEmpty: Dispatch<SetStateAction<boolean>>;
  setPinnedConversationIds: (updater: StringListUpdater) => void;
  setPrompt: Dispatch<SetStateAction<string>>;
  setRecentProjects: Dispatch<SetStateAction<domain.AssistantProject[]>>;
  setRevealingHistoryConversation: Dispatch<SetStateAction<boolean>>;
  setRightSidebarOpen: Dispatch<SetStateAction<boolean>>;
  setSelectedProjectPath: Dispatch<SetStateAction<string>>;
  setSessions: Dispatch<SetStateAction<domain.Session[]>>;
  setToolActivityTabs: Dispatch<SetStateAction<ToolActivityTab[]>>;
  setTurns: Dispatch<SetStateAction<ConversationTurn[]>>;
  sidebarConversationSelectionRef: { current: number };
}) {
  const { projectConversationGroups, visibleSessions } =
    useProjectSidebarConversationState({
      archivedConversationIds,
      recentProjects,
      selectedProjectPath,
      sessions,
    });
  const sidebarActions = useProjectSidebarActions({
    activeSessionIdRef,
    captureComposerTransitionStart,
    clearPendingPermissionCountForSession,
    closedToolActivityItemIdsRef,
    hasTurns,
    loadConversationTurns,
    refreshPendingPermissionRequests,
    resetConversationScroll,
    restoreToolActivitySessionState,
    saveCurrentToolActivitySessionState,
    selectedProjectPath,
    sessions,
    setActiveSessionId,
    setActiveToolActivityTabId,
    setArchivedConversationIds,
    setConversationRunning,
    setOpeningConversationFromEmpty,
    setPinnedConversationIds,
    setPrompt,
    setRecentProjects,
    setRevealingHistoryConversation,
    setRightSidebarOpen,
    setSelectedProjectPath,
    setSessions,
    setToolActivityTabs,
    setTurns,
    sidebarConversationSelectionRef,
  });

  return {
    ...sidebarActions,
    projectConversationGroups,
    visibleSessions,
  };
}
