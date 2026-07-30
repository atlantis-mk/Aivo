import type { CSSProperties } from "react";

import {
  PermissionApprovalDock,
  QuestionRequestDock,
} from "@/features/projects/project-interaction-docks";
import type { ProjectWorkspaceMainContentProps } from "@/features/projects/project-workspace-main-content-model";
import {
  ProjectComposerDropOverlay,
  ProjectEnvironmentSummaryAside,
  ProjectWorkspaceEmptyPrompt,
} from "@/features/projects/project-workspace-chat-overlays";
import { ProjectWorkspaceComposerFrame } from "@/features/projects/project-workspace-composer-frame";
import { ProjectConversationViewport } from "@/features/projects/project-workspace-conversation-view";

export function ProjectWorkspaceChatContent({
  activeSubagentRun,
  agentMode,
  agentModes,
  agentRuns,
  allModelOptions,
  attachments,
  canDockPinnedSummary,
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
  isPinnedSummaryOpen,
  isRevealingHistoryConversation,
  isSubagentSession,
  isVisibleTodoPlanComplete,
  mainRef,
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
  reasoningEffort,
  serviceTier,
  shouldShiftPinnedSummaryLayout,
  shouldShowEnvironmentSummaryPanel,
  shouldShowTodoFloatingStatus,
  showConversationLayout,
  showProjectPicker,
  showScrollToBottomButton,
  showServiceTier,
  todoItems,
  turns,
  viewportHandlers,
}: ProjectWorkspaceMainContentProps) {
  return (
    <div id="conversation-main" className="min-h-0 flex-1">
      <div
        className="relative h-full min-h-0 overflow-hidden px-4 sm:px-6"
        onDragEnter={onDragEnter}
        onDragLeave={onDragLeave}
        onDragOver={onDragOver}
        onDrop={onDrop}
        ref={mainRef}
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
          showConversationLayout={showConversationLayout}
        />

        <ProjectComposerDropOverlay active={isComposerDropActive} />

        <ProjectConversationViewport
          agentRuns={agentRuns}
          contentRef={contentRef}
          dockPinnedSummary={
            isPinnedSummaryOpen && shouldShiftPinnedSummaryLayout
          }
          handlers={viewportHandlers}
          hasTurns={hasTurns}
          reservePermissionDock={hasPendingInteractionRequest}
          revealFromHistory={isRevealingHistoryConversation}
          rootRef={messagesScrollRootRef}
          showConversationLayout={showConversationLayout}
          turns={turns}
        />

        {shouldShowEnvironmentSummaryPanel ? (
          <ProjectEnvironmentSummaryAside
            canDockPinnedSummary={canDockPinnedSummary}
            onOpenTools={onOpenToolActivationDialog}
          />
        ) : null}

        {showConversationLayout && (
          <div className="pointer-events-none absolute bottom-0 left-0 right-2.5 z-5 h-10 bg-background" />
        )}

        {hasPendingPermissionRequest && showConversationLayout ? (
          <PermissionApprovalDock
            dockPinnedSummary={
              isPinnedSummaryOpen && shouldShiftPinnedSummaryLayout
            }
            permissions={pendingPermissionRequests}
          />
        ) : hasPendingQuestionRequest &&
          pendingQuestionRequest &&
          showConversationLayout ? (
          <QuestionRequestDock
            dockPinnedSummary={
              isPinnedSummaryOpen && shouldShiftPinnedSummaryLayout
            }
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
            isPinnedSummaryOpen={isPinnedSummaryOpen}
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
            onPermissionModeSelect={onPermissionModeSelect}
            onProjectAdd={onProjectAdd}
            onProjectClear={onProjectClear}
            onProjectSelect={onProjectSelect}
            onPromptChange={onPromptChange}
            onReasoningEffortSelect={onReasoningEffortSelect}
            onRemoveAttachment={onRemoveAttachment}
            onScrollToBottom={onScrollToBottom}
            onServiceTierSelect={onServiceTierSelect}
            onSubmit={onSubmit}
            pending={hasPendingTurn}
            permissionMode={permissionMode}
            prompt={prompt}
            project={project}
            projectPath={projectPath}
            projects={projects}
            reasoningEffort={reasoningEffort}
            serviceTier={serviceTier}
            shouldShiftPinnedSummaryLayout={shouldShiftPinnedSummaryLayout}
            shouldShowTodoFloatingStatus={shouldShowTodoFloatingStatus}
            showConversationLayout={showConversationLayout}
            showProjectPicker={showProjectPicker}
            showScrollToBottomButton={showScrollToBottomButton}
            showServiceTier={showServiceTier}
            todoItems={todoItems}
          />
        )}
      </div>
    </div>
  );
}
