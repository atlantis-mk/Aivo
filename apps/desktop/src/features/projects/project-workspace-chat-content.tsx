import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
} from "react";

import {
  PermissionApprovalDock,
  QuestionRequestDock,
} from "@/features/projects/project-interaction-docks";
import type { ProjectWorkspaceMainContentProps } from "@/features/projects/project-workspace-main-content-model";
import {
  ProjectComposerDropOverlay,
  ProjectWorkspaceEmptyPrompt,
} from "@/features/projects/project-workspace-chat-overlays";
import { ProjectWorkspaceComposerFrame } from "@/features/projects/project-workspace-composer-frame";
import { ProjectConversationViewport } from "@/features/projects/project-workspace-conversation-view";
import { constructConversationTimelineRows } from "@/features/projects/conversation-timeline-row-model";
import type { ToolCallActivity } from "@/features/projects/conversation-timeline-tool-model";
import { ConversationToolInspector } from "@/features/projects/conversation-tool-inspector";
import {
  deriveSessionRuntimeStats,
  formatSessionRuntimeStatsValue,
} from "@/features/projects/project-session-runtime-stats";
import {
  compactSessionContext,
  getSessionRuntimeStats,
  type SessionRuntimeStats,
} from "@/services/aivo";
import { toast } from "sonner";

export function ProjectWorkspaceChatContent({
  activeSessionId,
  activeSubagentRun,
  agentMode,
  agentModes,
  agentRuns,
  allModelOptions,
  attachments,
  composerBottom,
  composerBottomSm,
  composerFrameRef,
  composerHeight,
  contentRef,
  emptyComposerTop,
  hasPendingInteractionRequest,
  hasPendingPermissionRequest,
  hasPendingQuestionRequest,
  hasPendingTurn,
  hasTurns,
  isComposerDropActive,
  isRevealingHistoryConversation,
  isSubagentSession,
  isVisibleTodoPlanComplete,
  messagesScrollRootRef,
  modelId,
  modelLabel,
  modelOptions,
  onAddAttachments,
  onAgentModeSelect,
  onBackToParentSession,
  onCancelSubagentRun,
  onDragEnter,
  onDragLeave,
  onDragOver,
  onDrop,
  onExtraHeightChange,
  onHeightChange,
  onHideCompletedTodoPlan,
  onModelSelect,
  onOpenToolActivationDialog,
  onPermissionModeSelect,
  onProjectAdd,
  onProjectClear,
  onProjectSelect,
  onPromptChange,
  onPromptMentionRemove,
  onPromptMentionSelect,
  onReasoningEffortSelect,
  onRemoveAttachment,
  onScrollToBottom,
  onServiceTierSelect,
  onSubmit,
  pendingPermissionRequests,
  pendingQuestionRequest,
  permissionMode,
  project,
  projectPath,
  projects,
  prompt,
  promptResourceReferences,
  reasoningEffort,
  serviceTier,
  shouldShowTodoFloatingStatus,
  showConversationLayout,
  showProjectPicker,
  showScrollToBottomButton,
  showServiceTier,
  todoItems,
  turns,
  viewportHandlers,
  workspaceRoot,
}: ProjectWorkspaceMainContentProps) {
  const [selectedToolActivityId, setSelectedToolActivityId] = useState("");
  const [contextCompactionPending, setContextCompactionPending] = useState(false);
  const compactContext = useCallback(async () => {
    if (!activeSessionId) {
      toast.info("请先开始会话，再压缩上下文");
      return;
    }
    if (hasPendingTurn) {
      toast.info("请等待当前回复完成后再压缩上下文");
      return;
    }
    if (contextCompactionPending) return;
    setContextCompactionPending(true);
    try {
      await compactSessionContext(activeSessionId);
      toast.success("上下文已压缩");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "压缩上下文失败");
    } finally {
      setContextCompactionPending(false);
    }
  }, [activeSessionId, contextCompactionPending, hasPendingTurn]);
  const toolActivities = useMemo(
    () =>
      constructConversationTimelineRows(turns).flatMap((row) => {
        if (row.type === "tool-cluster") {
          return [{ id: row.key, groups: row.groups }];
        }
        if (row.type === "tool-group" && row.group.kind !== "delegate") {
          return [{ id: row.key, groups: [row.group] }];
        }
        return [];
      }),
    [turns],
  );
  const [persistedRuntimeStats, setPersistedRuntimeStats] = useState<{
    sessionId: string;
    value?: SessionRuntimeStats;
  }>({ sessionId: "" });
  useEffect(() => {
    let cancelled = false;
    if (!activeSessionId) {
      setPersistedRuntimeStats({ sessionId: "" });
      return () => {
        cancelled = true;
      };
    }
    setPersistedRuntimeStats({ sessionId: activeSessionId });
    void getSessionRuntimeStats(activeSessionId)
      .then((value) => {
        if (!cancelled) setPersistedRuntimeStats({ sessionId: activeSessionId, value });
      })
      .catch(() => {
        if (!cancelled) setPersistedRuntimeStats({ sessionId: activeSessionId });
      });
    return () => {
      cancelled = true;
    };
  }, [activeSessionId, hasPendingTurn]);
  const runtimeStatsLine = useMemo(
    () =>
      showConversationLayout
        ? formatSessionRuntimeStatsValue(
            persistedRuntimeStats.sessionId === activeSessionId
              ? persistedRuntimeStats.value ?? deriveSessionRuntimeStats(turns)
              : deriveSessionRuntimeStats(turns),
          )
        : "",
    [activeSessionId, persistedRuntimeStats, showConversationLayout, turns],
  );
  const toolInspectorAutoOpenRef = useRef({
    observedToolCallIds: new Set(
      toolActivities.flatMap((activity) =>
        activity.groups.flatMap((group) =>
          group.calls.map((toolCall) => toolCall.id),
        ),
      ),
    ),
    sessionId: activeSessionId,
    suppressed: false,
  });
  const selectedToolActivity =
    toolActivities.find(
      (activity) => activity.id === selectedToolActivityId,
    ) ?? null;
  const openToolActivity = useCallback((activity: ToolCallActivity) => {
    setSelectedToolActivityId(activity.id);
  }, []);
  const closeToolActivity = useCallback(() => {
    toolInspectorAutoOpenRef.current.suppressed = true;
    setSelectedToolActivityId("");
  }, []);

  useEffect(() => {
    const currentToolCallIds = new Set(
      toolActivities.flatMap((activity) =>
        activity.groups.flatMap((group) =>
          group.calls.map((toolCall) => toolCall.id),
        ),
      ),
    );
    const autoOpenState = toolInspectorAutoOpenRef.current;

    if (autoOpenState.sessionId !== activeSessionId) {
      toolInspectorAutoOpenRef.current = {
        observedToolCallIds: currentToolCallIds,
        sessionId: activeSessionId,
        suppressed: false,
      };
      setSelectedToolActivityId("");
      if (
        hasPendingTurn &&
        !isRevealingHistoryConversation &&
        toolActivities.length > 0
      ) {
        setSelectedToolActivityId(toolActivities.at(-1)?.id ?? "");
      }
      return;
    }

    const latestNewActivity = toolActivities.findLast((activity) =>
      activity.groups.some((group) =>
        group.calls.some(
          (toolCall) => !autoOpenState.observedToolCallIds.has(toolCall.id),
        ),
      ),
    );
    autoOpenState.observedToolCallIds = currentToolCallIds;
    if (
      latestNewActivity &&
      hasPendingTurn &&
      !isRevealingHistoryConversation &&
      !autoOpenState.suppressed
    ) {
      setSelectedToolActivityId(latestNewActivity.id);
    }
  }, [
    activeSessionId,
    hasPendingTurn,
    isRevealingHistoryConversation,
    toolActivities,
  ]);

  return (
    <div
      id="conversation-main"
      className="relative flex min-h-0 flex-1 overflow-hidden"
    >
      <div
        className="relative h-full min-h-0 min-w-0 flex-1 overflow-hidden px-4 sm:px-6"
        onDragEnter={onDragEnter}
        onDragLeave={onDragLeave}
        onDragOver={onDragOver}
        onDrop={onDrop}
        style={
          {
            "--composer-height": `${composerHeight}px`,
            "--conversation-bottom-height": `${composerHeight}px`,
            "--conversation-composer-bottom": composerBottom,
            "--conversation-composer-bottom-sm": composerBottomSm,
            "--empty-composer-top": emptyComposerTop,
          } as CSSProperties
        }
      >
        <ProjectWorkspaceEmptyPrompt
          onPromptChange={onPromptChange}
          showConversationLayout={showConversationLayout}
        />

        <ProjectComposerDropOverlay active={isComposerDropActive} />

        <ProjectConversationViewport
          agentRuns={agentRuns}
          contentRef={contentRef}
          handlers={viewportHandlers}
          hasTurns={hasTurns}
          onOpenToolActivity={openToolActivity}
          reserveFloatingControls={
            shouldShowTodoFloatingStatus || showScrollToBottomButton
          }
          reservePermissionDock={hasPendingInteractionRequest}
          revealFromHistory={isRevealingHistoryConversation}
          rootRef={messagesScrollRootRef}
          showConversationLayout={showConversationLayout}
          turns={turns}
          workspaceRoot={workspaceRoot}
        />

        {showConversationLayout && (
          <div className="pointer-events-none absolute bottom-0 left-0 right-2.5 z-5 h-10 bg-background" />
        )}

        {hasPendingPermissionRequest && showConversationLayout ? (
          <PermissionApprovalDock
            permissions={pendingPermissionRequests}
          />
        ) : hasPendingQuestionRequest &&
          pendingQuestionRequest &&
          showConversationLayout ? (
          <QuestionRequestDock
            request={pendingQuestionRequest}
          />
        ) : (
          <ProjectWorkspaceComposerFrame
            activeSubagentRun={activeSubagentRun}
            agentMode={agentMode}
            agentModes={agentModes}
            allModelOptions={allModelOptions}
            attachments={attachments}
            composerFrameRef={composerFrameRef}
            isSubagentSession={isSubagentSession}
            isVisibleTodoPlanComplete={isVisibleTodoPlanComplete}
            modelId={modelId}
            modelLabel={modelLabel}
            modelOptions={modelOptions}
            onAddAttachments={onAddAttachments}
            onAgentModeSelect={onAgentModeSelect}
            onBackToParentSession={onBackToParentSession}
            onCancelSubagentRun={onCancelSubagentRun}
            onExtraHeightChange={onExtraHeightChange}
            onHeightChange={onHeightChange}
            onHideCompletedTodoPlan={onHideCompletedTodoPlan}
            onModelSelect={onModelSelect}
            onCompactContext={compactContext}
            onOpenToolActivationDialog={onOpenToolActivationDialog}
            onPermissionModeSelect={onPermissionModeSelect}
            onProjectAdd={onProjectAdd}
            onProjectClear={onProjectClear}
            onProjectSelect={onProjectSelect}
            onPromptChange={onPromptChange}
            onPromptMentionRemove={onPromptMentionRemove}
            onPromptMentionSelect={onPromptMentionSelect}
            onReasoningEffortSelect={onReasoningEffortSelect}
            onRemoveAttachment={onRemoveAttachment}
            onScrollToBottom={onScrollToBottom}
            onServiceTierSelect={onServiceTierSelect}
            onSubmit={onSubmit}
            pending={hasPendingTurn || contextCompactionPending}
            permissionMode={permissionMode}
            prompt={prompt}
            promptResourceReferences={promptResourceReferences}
            project={project}
            projectPath={projectPath}
            projects={projects}
            reasoningEffort={reasoningEffort}
            runtimeStatsLine={runtimeStatsLine}
            serviceTier={serviceTier}
            shouldShowTodoFloatingStatus={shouldShowTodoFloatingStatus}
            showConversationLayout={showConversationLayout}
            showProjectPicker={showProjectPicker}
            showScrollToBottomButton={showScrollToBottomButton}
            showServiceTier={showServiceTier}
            todoItems={todoItems}
          />
        )}
      </div>
      <ConversationToolInspector
        activity={selectedToolActivity}
        onClose={closeToolActivity}
      />
    </div>
  );
}
