import { type CSSProperties } from "react";

import {
  CodexApprovalDock,
  useCodexApprovalRequests,
} from "@/features/projects/project-codex-approval-dock";
import {
  CodexUserInputDock,
  useCodexUserInputRequests,
} from "@/features/projects/project-codex-user-input-dock";
import { QuestionRequestDock } from "@/features/projects/project-interaction-docks";
import type { ProjectWorkspaceMainContentProps } from "@/features/projects/project-workspace-main-content-model";
import {
  ProjectComposerDropOverlay,
  ProjectWorkspaceEmptyPrompt,
} from "@/features/projects/project-workspace-chat-overlays";
import { ProjectWorkspaceComposerFrame } from "@/features/projects/project-workspace-composer-frame";
import { ProjectConversationViewport } from "@/features/projects/project-workspace-conversation-view";

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
  hasPendingQuestionRequest,
  hasPendingTurn,
  hasPausedTurn,
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
  onPromptPaste,
  onPromptPasteRemove,
  onReasoningEffortSelect,
  onRemoveAttachment,
  onRemoveQueuedPrompt,
  onScrollToBottom,
  onServiceTierSelect,
  onSteerQueuedPrompt,
  onSubmit,
  pendingQuestionRequest,
  permissionMode,
  project,
  projectPath,
  projects,
  prompt,
  queuedPrompts,
  promptResourceReferences,
  pendingPromptPastes,
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
  const {
    approvalAction,
    requests: codexApprovalRequests,
    resolve: resolveCodexApproval,
    setApprovalAction,
  } = useCodexApprovalRequests();
  const { requests: codexUserInputRequests, resolve: resolveCodexUserInput } =
    useCodexUserInputRequests();
  const hasPendingCodexApproval = codexApprovalRequests.length > 0;
  const hasPendingCodexUserInput = codexUserInputRequests.length > 0;

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
          activeSessionId={activeSessionId}
          agentRuns={agentRuns}
          contentRef={contentRef}
          handlers={viewportHandlers}
          hasTurns={hasTurns}
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

        {hasPendingCodexApproval && showConversationLayout ? (
          <CodexApprovalDock
            approvalAction={approvalAction}
            onApprovalActionSelect={setApprovalAction}
            requests={codexApprovalRequests}
            resolve={resolveCodexApproval}
          />
        ) : hasPendingCodexUserInput && showConversationLayout ? (
          <CodexUserInputDock
            requests={codexUserInputRequests}
            resolve={resolveCodexUserInput}
          />
        ) : hasPendingQuestionRequest &&
          pendingQuestionRequest &&
          showConversationLayout ? (
          <QuestionRequestDock request={pendingQuestionRequest} />
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
            onOpenToolActivationDialog={onOpenToolActivationDialog}
            onPermissionModeSelect={onPermissionModeSelect}
            onProjectAdd={onProjectAdd}
            onProjectClear={onProjectClear}
            onProjectSelect={onProjectSelect}
            onPromptChange={onPromptChange}
            onPromptMentionRemove={onPromptMentionRemove}
            onPromptMentionSelect={onPromptMentionSelect}
            onPromptPaste={onPromptPaste}
            onPromptPasteRemove={onPromptPasteRemove}
            onReasoningEffortSelect={onReasoningEffortSelect}
            onRemoveAttachment={onRemoveAttachment}
            onRemoveQueuedPrompt={onRemoveQueuedPrompt}
            onScrollToBottom={onScrollToBottom}
            onServiceTierSelect={onServiceTierSelect}
            onSteerQueuedPrompt={onSteerQueuedPrompt}
            onSubmit={onSubmit}
            pending={hasPendingTurn}
            hasPausedTurn={hasPausedTurn}
            permissionMode={permissionMode}
            prompt={prompt}
            queuedPrompts={queuedPrompts}
            promptResourceReferences={promptResourceReferences}
            pendingPromptPastes={pendingPromptPastes}
            project={project}
            projectPath={projectPath}
            projects={projects}
            reasoningEffort={reasoningEffort}
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
    </div>
  );
}
