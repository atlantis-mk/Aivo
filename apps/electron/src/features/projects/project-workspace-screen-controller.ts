import { useEffect } from "react";
import { toast } from "sonner";

import { useProjectAgentRuntimeState } from "@/features/projects/project-agent-runtime-state";
import { useProjectAssistantDeltaBuffer } from "@/features/projects/project-assistant-delta-buffer";
import { useProjectComposerTransitionState } from "@/features/projects/project-composer-transition-state";
import { useProjectConversationRuntimeState } from "@/features/projects/project-conversation-runtime-state";
import { useProjectConversationScroll } from "@/features/projects/project-conversation-scroll-state";
import { useProjectConversationTurnActions } from "@/features/projects/project-conversation-turn-actions";
import { useProjectConversationTurnLoader } from "@/features/projects/project-conversation-turn-loader";
import { useProjectInteractionRequestState } from "@/features/projects/project-interaction-request-state";
import {
  getProjectConversationViewState,
  getProjectWorkspacePanelViewState,
} from "@/features/projects/project-workspace-view-state";
import { useProjectWorkspacePreferencesState } from "@/features/projects/project-workspace-preferences-state";
import { useProjectWorkspaceRouteState } from "@/features/projects/project-workspace-route-state";
import { useProjectWorkspaceScreenEffects } from "@/features/projects/project-workspace-screen-effects";
import { useProjectWorkspaceSessionState } from "@/features/projects/project-workspace-session-state";
import { useProjectWorkspaceModelComposerController } from "@/features/projects/project-workspace-model-composer-controller";
import { buildProjectWorkspaceScreenViewProps } from "@/features/projects/project-workspace-screen-view-props";
import type { ProjectWorkspaceScreenViewProps } from "@/features/projects/project-workspace-screen-view";
import { useProjectWorkspaceSidebarController } from "@/features/projects/project-workspace-sidebar-controller";
import { useProjectWorkspaceScreenState } from "@/features/projects/project-workspace-screen-state";
import { useProjectWorkspaceToolActivityController } from "@/features/projects/project-workspace-tool-activity-controller";
import { useProjectWorkspaceUiActions } from "@/features/projects/project-workspace-ui-actions";
import { useAppConfig } from "@/lib/app-config";
import { getProviderCatalogForProject } from "@/services/aivo";

export function useProjectWorkspaceScreenController(): ProjectWorkspaceScreenViewProps {
  const {
    activeSessionId,
    isOpeningConversationFromEmpty,
    isRevealingHistoryConversation,
    prompt,
    extensionSettingsDrawerOpen,
    recentProjects,
    selectedProjectPath,
    sessions,
    setActiveSessionId,
    setOpeningConversationFromEmpty,
    setExtensionSettingsDrawerOpen,
    setPrompt,
    setRecentProjects,
    setRevealingHistoryConversation,
    setSelectedProjectPath,
    setSessions,
    setToolActivationDialogOpen,
    setTurns,
    toolActivationDialogOpen,
    turns,
  } = useProjectWorkspaceScreenState();
  const {
    archivedConversationIds,
    pendingActiveToolNames,
    hiddenTodoPlanKeys,
    pinnedConversationIds,
    setArchivedConversationIds,
    setPendingActiveToolNames,
    setHiddenTodoPlanKeyForSession,
    setPinnedConversationIds,
  } = useProjectWorkspacePreferencesState();
  const { catalog, config, setCatalog, setConfig } = useAppConfig();
  const { activeProjectPage, navigateToProjectChat } =
    useProjectWorkspaceRouteState();
  const {
    cancel: cancelPendingAssistantDelta,
    enqueue: enqueueAssistantDelta,
    flush: flushPendingAssistantDelta,
  } = useProjectAssistantDeltaBuffer(setTurns);
  const {
    activeSessionIdRef,
    pendingStopRequestedRef,
    runningConversationIds,
    setConversationRunning,
    sidebarConversationSelectionRef,
  } = useProjectConversationRuntimeState({
    activeSessionId,
    flushPendingAssistantDelta,
  });
  const { hasPendingTurn, hasTurns, lastTurnStateKey, showConversationLayout } =
    getProjectConversationViewState({
      isOpeningConversationFromEmpty,
      turns,
    });
  const {
    activeSession,
    activeWorkspaceRoot,
    composerProject,
    composerProjectPath,
    conversationTitle,
    hiddenTodoPlanKey,
    setCodingWorkspaceRoot,
  } = useProjectWorkspaceSessionState({
    activeSessionId,
    hiddenTodoPlanKeys,
    recentProjects,
    selectedProjectPath,
    sessions,
    turns,
  });
  const {
    activeParentSessionId,
    activeRunningSubagentRun,
    activeSubagentRun,
    agentMode,
    agentModes,
    agentRuns,
    hideCompletedTodoPlan,
    isSubagentSession,
    isVisibleTodoPlanComplete,
    refreshAgentRuntimeState,
    setAgentMode,
    setTodoItems,
    shouldShowTodoFloatingStatus,
    visibleTodoPlanItems,
  } = useProjectAgentRuntimeState({
    activeSession,
    activeSessionId,
    activeSessionIdRef,
    activeWorkspaceRoot,
    hiddenTodoPlanKey,
    setHiddenTodoPlanKeyForSession,
  });
  const {
    captureComposerTransitionStart,
    composerBottom,
    composerBottomSm,
    composerFrameRef,
    composerHeight,
    emptyComposerTop,
    handleComposerHeightChange,
    setComposerExtraHeight,
    stopComposerTransition,
  } = useProjectComposerTransitionState({
    activeSessionId,
    showConversationLayout,
  });
  const {
    contentRef: messagesContentRef,
    handleScrollToBottomButtonClick,
    prepareConversationReveal,
    resetConversationScroll,
    rootRef: messagesScrollRootRef,
    showScrollToBottomButton,
    stopForceScrollToBottom,
  } = useProjectConversationScroll({
    composerHeight,
    hasTurns,
    lastTurnStateKey,
    showConversationLayout,
    turnCount: turns.length,
  });
  const loadConversationTurns = useProjectConversationTurnLoader({
    activeSessionIdRef,
    prepareConversationReveal,
    setConversationRunning,
    setTurns,
    turns,
  });
  const {
    applyToolActivityFileState,
    closedToolActivityItemIdsRef,
    mergeToolActivityFromCall,
    restoreToolActivitySessionState,
    saveCurrentToolActivitySessionState,
    setActiveToolActivityTabId,
    setRightSidebarOpen,
    setToolActivityTabs,
  } = useProjectWorkspaceToolActivityController({
    activeSessionId,
    activeSessionIdRef,
    loadConversationTurns,
  });
  const { pendingQuestionRequests, refreshPendingQuestionRequests } =
    useProjectInteractionRequestState({
      activeSessionId,
      activeSessionIdRef,
    });
  const { hasPendingInteractionRequest, hasPendingQuestionRequest } =
    getProjectWorkspacePanelViewState({
      pendingQuestionRequests,
    });

  const {
    cancelActiveSubagentRun,
    activeModelId,
    activeModelRef,
    addComposerAttachmentFiles,
    allModelOptions,
    composerAttachments,
    handleComposerDragEnter,
    handleComposerDragLeave,
    handleComposerDragOver,
    handleComposerDrop,
    isComposerDropActive,
    modelOptions,
    permissionMode,
    reasoningEffort,
    removeComposerAttachment,
    removePromptMention,
    queuedPrompts,
    changePrompt,
    handlePromptPaste,
    removePromptPaste,
    removeQueuedPrompt,
    pendingPromptPastes,
    selectAgentMode,
    selectModel,
    selectPermissionMode,
    selectPromptMention,
    steerQueuedPrompt,
    promptResourceReferences,
    selectReasoningEffort,
    selectServiceTier,
    serviceTier,
    submitPrompt,
  } = useProjectWorkspaceModelComposerController({
    activeRunningSubagentRun,
    activeSessionId,
    activeSessionIdRef,
    agentMode,
    catalog,
    config,
    pendingActiveToolNames,
    hasPendingTurn,
    loadConversationTurns,
    pendingStopRequestedRef,
    prompt,
    refreshAgentRuntimeState,
    selectedProjectPath,
    setActiveSessionId,
    setAgentMode,
    setCodingWorkspaceRoot,
    setConfig,
    setConversationRunning,
    setPrompt,
    setPendingActiveToolNames,
    setSessions,
    setTurns,
    turns,
  });
  const hasComposerSubmitInput =
    Boolean(prompt.trim()) ||
    composerAttachments.length > 0 ||
    pendingPromptPastes.length > 0;
  const {
    addComposerProject,
    archiveConversation,
    clearComposerProject,
    hideSidebarProject,
    openConversationById,
    projectConversationGroups,
    refreshRecentProjects,
    selectComposerProject,
    selectSidebarConversation,
    startNewConversation,
    startNewProjectConversation,
    togglePinnedConversation,
    visibleSessions,
  } = useProjectWorkspaceSidebarController({
    activeSessionIdRef,
    archivedConversationIds,
    captureComposerTransitionStart,
    closedToolActivityItemIdsRef,
    hasTurns,
    loadConversationTurns,
    recentProjects,
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

  const {
    deleteConversationAssistantMessage,
    deleteConversationTurn,
    editConversationUserMessage,
    retryConversationTurn,
    stopPendingTurn,
  } = useProjectConversationTurnActions({
    activeModelRef,
    activeSessionIdRef,
    agentMode,
    hasPendingTurn,
    loadConversationTurns,
    pendingStopRequestedRef,
    reasoningEffort,
    serviceTier,
    setConversationRunning,
    setSessions,
    setTurns,
    turns,
  });

  useProjectWorkspaceScreenEffects({
    activeSessionIdRef,
    activeWorkspaceRoot,
    cancelPendingAssistantDelta,
    enqueueAssistantDelta,
    flushPendingAssistantDelta,
    loadConversationTurns,
    mergeToolActivityFromCall,
    refreshAgentRuntimeState,
    refreshPendingQuestionRequests,
    refreshRecentProjects,
    setConversationRunning,
    setSessions,
    setTodoItems,
    setTurns,
    stopComposerTransition,
    stopForceScrollToBottom,
  });
  const {
    addProjectToComposer,
    conversationTimelineHandlers,
    openParentSession,
    openToolActivationDialog,
    selectChatConversation,
    startChatConversation,
    startProjectChatConversation,
  } = useProjectWorkspaceUiActions({
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
  });
  return buildProjectWorkspaceScreenViewProps({
    dialogs: {
      activeSessionId,
      pendingActiveToolNames,
      onPendingActiveToolNamesChange: setPendingActiveToolNames,
      onExtensionSettingsDrawerOpenChange: setExtensionSettingsDrawerOpen,
      extensionSettingsDrawerOpen,
      onToolActivationOpenChange: setToolActivationDialogOpen,
      toolActivationDialogOpen,
      turns,
      workspaceRoot: activeWorkspaceRoot,
    },
    leftSidebar: {
      activeConversationId: activeSessionId,
      activeProjectPage,
      conversations: visibleSessions,
      archiveConversation,
      onHideProject: hideSidebarProject,
      onNewConversation: startChatConversation,
      onNewProjectConversation: startProjectChatConversation,
      onSelectConversation: selectChatConversation,
      onTogglePinnedConversation: togglePinnedConversation,
      pinnedConversationIds,
      projectGroups: projectConversationGroups,
      runningConversationIds,
      selectedProjectPath,
    },
    topBar: {
      activeProjectPage,
      conversationTitle,
      hasConversation: Boolean(activeSessionId),
    },
    mainTopBar: {
      activeProjectPage,
      conversationTitle,
      hasConversation: Boolean(activeSessionId),
      repositoryPath: composerProjectPath,
      sessionId: activeSessionId,
    },
    main: {
      activeProjectPage,
      activeSessionId,
      activeModelRef,
      activeRunningSubagentRun,
      activeSubagentRun,
      agentMode,
      agentModes,
      agentRuns,
      allModelOptions,
      attachments: composerAttachments,
      composerBottom,
      composerBottomSm,
      composerFrameRef,
      composerHeight,
      contentRef: messagesContentRef,
      emptyComposerTop,
      hasPendingInteractionRequest,
      hasPendingQuestionRequest,
      hasPendingTurn,
      hasTurns,
      isComposerDropActive,
      isRevealingHistoryConversation,
      isSubagentSession,
      isVisibleTodoPlanComplete,
      messagesScrollRootRef,
      modelId: activeModelId,
      modelOptions,
      onAddAttachments: addComposerAttachmentFiles,
      onAgentModeSelect: selectAgentMode,
      onBackToParentSession: openParentSession,
      cancelActiveSubagentRun,
      onDragEnter: handleComposerDragEnter,
      onDragLeave: handleComposerDragLeave,
      onDragOver: handleComposerDragOver,
      onDrop: (event) => handleComposerDrop(event, addProjectToComposer),
      onExtraHeightChange: setComposerExtraHeight,
      onHeightChange: handleComposerHeightChange,
      onHideCompletedTodoPlan: hideCompletedTodoPlan,
      onModelSelect: selectModel,
      onOpenToolActivationDialog: openToolActivationDialog,
      onPermissionModeSelect: selectPermissionMode,
      onProjectAdd: addProjectToComposer,
      onProjectClear: clearComposerProject,
      onProjectSelect: selectComposerProject,
      onPromptChange: changePrompt,
      onPromptMentionRemove: removePromptMention,
      onPromptMentionSelect: selectPromptMention,
      onPromptPaste: handlePromptPaste,
      onPromptPasteRemove: removePromptPaste,
      onReasoningEffortSelect: selectReasoningEffort,
      onRemoveAttachment: removeComposerAttachment,
      onRemoveQueuedPrompt: removeQueuedPrompt,
      onScrollToBottom: handleScrollToBottomButtonClick,
      onServiceTierSelect: selectServiceTier,
      onSteerQueuedPrompt: (id) => {
        void steerQueuedPrompt(id);
      },
      onSubmit:
        hasPendingTurn && !hasComposerSubmitInput
          ? stopPendingTurn
          : submitPrompt,
      pendingQuestionRequests,
      permissionMode,
      project: composerProject,
      projectPath: composerProjectPath,
      projects: recentProjects,
      prompt,
      queuedPrompts,
      promptResourceReferences,
      pendingPromptPastes,
      reasoningEffort,
      serviceTier,
      shouldShowTodoFloatingStatus,
      showConversationLayout,
      showScrollToBottomButton,
      todoItems: visibleTodoPlanItems,
      turns,
      viewportHandlers: conversationTimelineHandlers,
      workspaceRoot: activeWorkspaceRoot,
    },
  });
}
