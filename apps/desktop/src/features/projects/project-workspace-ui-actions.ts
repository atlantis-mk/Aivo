import { useCallback } from "react";
import type { Dispatch, SetStateAction } from "react";

import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";
import { useStableConversationTimelineHandlers } from "@/features/projects/project-conversation-timeline-handlers";
import type { ToolActivityFileTab } from "@/features/projects/tool-activity-types";
import type { domain } from "../../../bridge/go/models";

export function useProjectWorkspaceUiActions({
  activeParentSessionId,
  addComposerProject,
  applyToolActivityFileState,
  deleteConversationAssistantMessage,
  deleteConversationTurn,
  editConversationUserMessage,
  navigateToProjectChat,
  openConversationById,
  retryConversationTurn,
  selectSidebarConversation,
  setExtensionSettingsDrawerOpen,
  setToolActivationDialogOpen,
  startNewConversation,
  startNewProjectConversation,
}: {
  activeParentSessionId: string;
  addComposerProject: (rootPath?: string) => Promise<void>;
  applyToolActivityFileState: (
    tab: ToolActivityFileTab,
    targetState: "before" | "after",
  ) => Promise<void>;
  deleteConversationAssistantMessage: (
    turn: ConversationTurn,
  ) => Promise<void>;
  deleteConversationTurn: (turn: ConversationTurn) => Promise<void>;
  editConversationUserMessage: (turn: ConversationTurn) => Promise<void>;
  navigateToProjectChat: () => void;
  openConversationById: (sessionId: string) => Promise<void>;
  retryConversationTurn: (turn: ConversationTurn) => Promise<void>;
  selectSidebarConversation: (session: domain.Session) => Promise<void>;
  setExtensionSettingsDrawerOpen: Dispatch<SetStateAction<boolean>>;
  setToolActivationDialogOpen: Dispatch<SetStateAction<boolean>>;
  startNewConversation: () => void;
  startNewProjectConversation: (projectPath: string) => void;
}) {
  const startChatConversation = useCallback(() => {
    startNewConversation();
    navigateToProjectChat();
  }, [navigateToProjectChat, startNewConversation]);

  const startProjectChatConversation = useCallback(
    (projectPath: string) => {
      startNewProjectConversation(projectPath);
      navigateToProjectChat();
    },
    [navigateToProjectChat, startNewProjectConversation],
  );

  const selectChatConversation = useCallback(
    (session: domain.Session) => {
      navigateToProjectChat();
      void selectSidebarConversation(session);
    },
    [navigateToProjectChat, selectSidebarConversation],
  );

  const openExtensionSettingsDrawer = useCallback(() => {
    setExtensionSettingsDrawerOpen(true);
  }, [setExtensionSettingsDrawerOpen]);

  const openToolActivationDialog = useCallback(() => {
    setToolActivationDialogOpen(true);
  }, [setToolActivationDialogOpen]);

  const openParentSession = useCallback(() => {
    void openConversationById(activeParentSessionId);
  }, [activeParentSessionId, openConversationById]);

  const addProjectToComposer = useCallback((rootPath?: string) => {
    void addComposerProject(rootPath);
  }, [addComposerProject]);

  const applyToolActivityFileStateFromSidebar = useCallback(
    (tab: ToolActivityFileTab, targetState: "before" | "after") => {
      void applyToolActivityFileState(tab, targetState);
    },
    [applyToolActivityFileState],
  );
  const conversationTimelineHandlers = useStableConversationTimelineHandlers({
    onDeleteAssistantMessage: (turn) => {
      void deleteConversationAssistantMessage(turn);
    },
    onDeleteTurn: (turn) => {
      void deleteConversationTurn(turn);
    },
    onEditUserMessage: (turn) => {
      void editConversationUserMessage(turn);
    },
    onOpenSession: (sessionId) => {
      void openConversationById(sessionId);
    },
    onRetryTurn: (turn) => {
      void retryConversationTurn(turn);
    },
  });

  return {
    addProjectToComposer,
    applyToolActivityFileStateFromSidebar,
    conversationTimelineHandlers,
    openParentSession,
    openExtensionSettingsDrawer,
    openToolActivationDialog,
    selectChatConversation,
    startChatConversation,
    startProjectChatConversation,
  };
}
