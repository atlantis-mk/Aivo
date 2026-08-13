import { useCallback } from "react";
import type { Dispatch, SetStateAction } from "react";
import { toast } from "sonner";

import {
  projectIsUserSelectable,
  upsertRecentProject,
} from "@/features/projects/project-sidebar-model";
import type { LoadConversationTurnsOptions } from "@/features/projects/project-conversation-turn-loader";
import { OPEN_CONVERSATION_FROM_EMPTY_DELAY } from "@/features/projects/project-workspace-state-model";
import { hasAppBridge } from "@/lib/app-config";
import {
  archiveSession,
  listRecentProjects,
  listSessions,
  scanProjectSkills,
  selectProjectDirectory,
  setProjectSidebarHidden,
  upsertProject,
} from "@/services/aivo";
import type { ToolActivityTab } from "@/features/projects/tool-activity-model";
import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";
import type { domain } from "../../../bridge/go/models";

type StringListUpdater = string[] | ((current: string[]) => string[]);

export function useProjectSidebarActions({
  activeSessionIdRef,
  captureComposerTransitionStart,
  clearPendingPermissionCountForSession,
  closedToolActivityItemIdsRef,
  hasTurns,
  loadConversationTurns,
  refreshPendingPermissionRequests,
  restoreToolActivitySessionState,
  saveCurrentToolActivitySessionState,
  resetConversationScroll,
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
  captureComposerTransitionStart: () => void;
  clearPendingPermissionCountForSession: (sessionId: string) => void;
  closedToolActivityItemIdsRef: { current: Set<string> };
  hasTurns: boolean;
  loadConversationTurns: (
    sessionId: string,
    options?: LoadConversationTurnsOptions,
  ) => Promise<void>;
  refreshPendingPermissionRequests: (sessionId?: string) => Promise<void>;
  restoreToolActivitySessionState: (sessionId: string) => void;
  saveCurrentToolActivitySessionState: () => void;
  resetConversationScroll: () => void;
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
  const refreshRecentProjects = useCallback(async () => {
    if (!hasAppBridge()) return;
    try {
      const projects = await listRecentProjects(20);
      setRecentProjects((projects ?? []).filter(projectIsUserSelectable));
    } catch {
      setRecentProjects([]);
    }
  }, [setRecentProjects]);

  function scanProjectInBackground(projectPath: string) {
    if (!hasAppBridge() || !projectPath) return;
    void scanProjectSkills(projectPath).catch(() => undefined);
  }

  function startNewConversation({
    preservePrompt = false,
  }: { preservePrompt?: boolean } = {}) {
    saveCurrentToolActivitySessionState();
    sidebarConversationSelectionRef.current += 1;
    resetConversationScroll();
    setOpeningConversationFromEmpty(false);
    setRevealingHistoryConversation(false);
    if (!preservePrompt) {
      setPrompt("");
    }
    setTurns([]);
    activeSessionIdRef.current = "";
    setActiveSessionId("");
    closedToolActivityItemIdsRef.current = new Set();
    setToolActivityTabs([]);
    setActiveToolActivityTabId("");
    setRightSidebarOpen(false);
  }

  function selectComposerProject(project: domain.AssistantProject) {
    if (activeSessionIdRef.current) {
      startNewConversation({ preservePrompt: true });
    }
    setSelectedProjectPath(project.rootPath);
    scanProjectInBackground(project.rootPath);
  }

  function clearComposerProject() {
    if (activeSessionIdRef.current) {
      startNewConversation({ preservePrompt: true });
    }
    setSelectedProjectPath("");
  }

  async function addComposerProject(selectedRootPath?: string) {
    if (!hasAppBridge()) return;
    try {
      const rootPath = selectedRootPath || await selectProjectDirectory();
      if (!rootPath) return;
      const project = await upsertProject(rootPath);
      if (projectIsUserSelectable(project)) {
        setRecentProjects((currentProjects) =>
          upsertRecentProject(currentProjects, project),
        );
      }
      if (activeSessionIdRef.current) {
        startNewConversation({ preservePrompt: true });
      }
      setSelectedProjectPath(rootPath);
      scanProjectInBackground(rootPath);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "选择项目失败");
    }
  }

  async function openConversation(session: domain.Session) {
    if (!hasAppBridge()) return;

    const isDifferentSession = session.id !== activeSessionIdRef.current;
    if (isDifferentSession) {
      saveCurrentToolActivitySessionState();
    }
    const shouldAnimateFromEmpty = !hasTurns && isDifferentSession;
    const selectionId = sidebarConversationSelectionRef.current + 1;
    sidebarConversationSelectionRef.current = selectionId;

    if (shouldAnimateFromEmpty) {
      captureComposerTransitionStart();
      setOpeningConversationFromEmpty(true);
      setRevealingHistoryConversation(false);
    } else {
      setRevealingHistoryConversation(false);
    }

    activeSessionIdRef.current = session.id;
    setActiveSessionId(session.id);
    restoreToolActivitySessionState(session.id);
    scanProjectInBackground(session.projectPath || "");

    try {
      if (shouldAnimateFromEmpty) {
        await delay(OPEN_CONVERSATION_FROM_EMPTY_DELAY);
      }
      if (sidebarConversationSelectionRef.current !== selectionId) {
        return;
      }
      setRevealingHistoryConversation(isDifferentSession);
      await loadConversationTurns(session.id, {
        snapToBottomAfterLoad: isDifferentSession,
      });
      await refreshPendingPermissionRequests(session.id);
    } finally {
      if (
        shouldAnimateFromEmpty &&
        sidebarConversationSelectionRef.current === selectionId
      ) {
        setOpeningConversationFromEmpty(false);
      }
    }
  }

  async function openConversationById(sessionId: string) {
    if (!hasAppBridge() || !sessionId) return;
    let session = sessions.find((candidate) => candidate.id === sessionId);
    if (!session) {
      const nextSessions = (await listSessions(50)) ?? [];
      setSessions(nextSessions);
      session = nextSessions.find((candidate) => candidate.id === sessionId);
    }
    if (!session) {
      toast.error("找不到子代理会话");
      return;
    }
    await openConversation(session);
  }

  async function selectSidebarConversation(session: domain.Session) {
    await openConversation(session);
  }

  function startNewProjectConversation(projectPath: string) {
    if (!projectPath) return;
    startNewConversation();
    setSelectedProjectPath(projectPath);
    scanProjectInBackground(projectPath);
  }

  async function hideSidebarProject(projectPath: string) {
    if (!projectPath) return;
    setRecentProjects((currentProjects) =>
      currentProjects.filter((project) => project.rootPath !== projectPath),
    );
    if (!activeSessionIdRef.current && selectedProjectPath === projectPath) {
      setSelectedProjectPath("");
    }
    if (!hasAppBridge()) return;
    try {
      await setProjectSidebarHidden(projectPath, true);
      await refreshRecentProjects();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "移除项目失败");
      await refreshRecentProjects();
    }
  }

  function togglePinnedConversation(sessionId: string) {
    setPinnedConversationIds((currentIds) =>
      currentIds.includes(sessionId)
        ? currentIds.filter((id) => id !== sessionId)
        : [sessionId, ...currentIds],
    );
  }

  async function archiveConversation(sessionId: string) {
    setArchivedConversationIds((currentIds) =>
      currentIds.includes(sessionId) ? currentIds : [sessionId, ...currentIds],
    );
    setPinnedConversationIds((currentIds) =>
      currentIds.filter((id) => id !== sessionId),
    );
    clearPendingPermissionCountForSession(sessionId);
    setConversationRunning(sessionId, false);
    if (sessionId === activeSessionIdRef.current) {
      startNewConversation();
    }

    if (!hasAppBridge()) return;
    try {
      await archiveSession(sessionId);
    } catch {
      setArchivedConversationIds((currentIds) =>
        currentIds.filter((id) => id !== sessionId),
      );
    }
  }

  return {
    addComposerProject,
    archiveConversation,
    clearComposerProject,
    hideSidebarProject,
    openConversationById,
    refreshRecentProjects,
    selectComposerProject,
    selectSidebarConversation,
    startNewConversation,
    startNewProjectConversation,
    togglePinnedConversation,
  };
}

function delay(ms: number) {
  return new Promise((resolve) => window.setTimeout(resolve, ms));
}
