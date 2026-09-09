import type { RefObject } from "react";

import type {
  ComposerAttachment,
  ComposerAttachmentInput,
} from "@/features/projects/project-composer-attachments";
import type { ModelOption } from "@/features/projects/project-model-options";
import type { PromptMentionReference } from "@/features/projects/project-prompt-mention-model";
import { ProjectComposerFloatingControls } from "@/features/projects/project-workspace-chat-overlays";
import { PromptComposer } from "@/features/projects/project-prompt-composer";
import type {
  PromptComposerProps,
  QueuedPrompt,
} from "@/features/projects/project-prompt-composer-types";
import { SubagentSessionActionBar } from "@/features/projects/project-workspace-top-bars";
import { cn } from "@/lib/utils";
import type { ModelInfo } from "@/lib/provider-catalog";
import type { PermissionMode } from "@/codex-app-server";
import type {
  AgentModeDefinition,
  AgentModeId,
  AgentRun,
  TodoItem,
} from "@/services/aivo";
import type { domain } from "@/types/codex-domain";

export function ProjectWorkspaceComposerFrame({
  activeSubagentRun,
  agentMode,
  agentModes,
  allModelOptions,
  attachments,
  composerFrameRef,
  isSubagentSession,
  isVisibleTodoPlanComplete,
  modelId,
  modelLabel,
  modelOptions,
  onAddAttachments,
  onAgentModeSelect,
  onBackToParentSession,
  onCancelSubagentRun,
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
  pending,
  hasPausedTurn,
  pendingPromptPastes,
  permissionMode,
  prompt,
  queuedPrompts,
  promptResourceReferences,
  project,
  projectPath,
  projects,
  reasoningEffort,
  serviceTier,
  shouldShowTodoFloatingStatus,
  showConversationLayout,
  showProjectPicker,
  showScrollToBottomButton,
  showServiceTier,
  todoItems,
}: {
  activeSubagentRun?: AgentRun;
  agentMode: AgentModeId;
  agentModes: AgentModeDefinition[];
  allModelOptions: ModelOption[];
  attachments: ComposerAttachment[];
  composerFrameRef: RefObject<HTMLDivElement | null>;
  isSubagentSession: boolean;
  isVisibleTodoPlanComplete: boolean;
  modelId: string;
  modelLabel: string;
  modelOptions: ModelInfo[];
  onAddAttachments: (files: ComposerAttachmentInput) => void;
  onAgentModeSelect: (mode: AgentModeId) => void;
  onBackToParentSession: () => void;
  onCancelSubagentRun?: () => void;
  onExtraHeightChange: (height: number) => void;
  onHeightChange: (height: number) => void;
  onHideCompletedTodoPlan: () => void;
  onModelSelect: (option: ModelOption) => void;
  onOpenToolActivationDialog: () => void;
  onPermissionModeSelect: (mode: PermissionMode) => void;
  onProjectAdd: (rootPath?: string) => void;
  onProjectClear: () => void;
  onProjectSelect: (project: domain.AssistantProject) => void;
  onPromptChange: (prompt: string) => void;
  onPromptMentionRemove: (reference: PromptMentionReference) => void;
  onPromptMentionSelect: (reference: PromptMentionReference) => void;
  onPromptPaste: PromptComposerProps["onPromptPaste"];
  onPromptPasteRemove: PromptComposerProps["onPromptPasteRemove"];
  onReasoningEffortSelect: (reasoningEffort: string) => void;
  onRemoveAttachment: (id: string) => void;
  onRemoveQueuedPrompt: (id: string) => void;
  onScrollToBottom: () => void;
  onServiceTierSelect: (serviceTier: string) => void;
  onSteerQueuedPrompt: (id: string) => void;
  pending: boolean;
  onSubmit: (promptOverride?: string) => void;
  hasPausedTurn: boolean;
  pendingPromptPastes: PromptComposerProps["pendingPromptPastes"];
  permissionMode: PermissionMode;
  prompt: string;
  queuedPrompts: QueuedPrompt[];
  promptResourceReferences: PromptMentionReference[];
  project: domain.AssistantProject | null;
  projectPath: string;
  projects: domain.AssistantProject[];
  reasoningEffort: string;
  serviceTier: string;
  shouldShowTodoFloatingStatus: boolean;
  showConversationLayout: boolean;
  showProjectPicker: boolean;
  showScrollToBottomButton: boolean;
  showServiceTier: boolean;
  todoItems: TodoItem[];
}) {
  return (
    <div
      ref={composerFrameRef}
      className={cn(
        "absolute left-1/2 z-30 w-[calc(100%-2rem)] max-w-[736px] -translate-x-1/2 sm:w-[calc(100%-48px)]",
        showConversationLayout
          ? "bottom-[var(--conversation-composer-bottom)] will-change-[bottom,transform] sm:bottom-[var(--conversation-composer-bottom-sm)]"
          : "bottom-[var(--conversation-composer-bottom)] will-change-[bottom,transform] sm:bottom-[var(--conversation-composer-bottom-sm)]",
        "transition-[bottom,transform,margin] duration-[520ms] ease-[cubic-bezier(0.22,1,0.36,1)]",
      )}
    >
      {showConversationLayout ? (
        <ProjectComposerFloatingControls
          isVisibleTodoPlanComplete={isVisibleTodoPlanComplete}
          onHideCompletedTodoPlan={onHideCompletedTodoPlan}
          onScrollToBottom={onScrollToBottom}
          shouldShowTodoFloatingStatus={shouldShowTodoFloatingStatus}
          showScrollToBottomButton={showScrollToBottomButton}
          todoItems={todoItems}
        />
      ) : null}
      {isSubagentSession ? (
        <SubagentSessionActionBar
          agentRun={activeSubagentRun}
          onBack={onBackToParentSession}
          onCancel={onCancelSubagentRun}
          onHeightChange={onHeightChange}
        />
      ) : (
        <PromptComposer
          onHeightChange={onHeightChange}
          onPromptChange={onPromptChange}
          onPromptMentionRemove={onPromptMentionRemove}
          onPromptMentionSelect={onPromptMentionSelect}
          onPromptPaste={onPromptPaste}
          onPromptPasteRemove={onPromptPasteRemove}
          onSubmit={onSubmit}
          pending={pending}
          hasPausedTurn={hasPausedTurn}
          pendingPromptPastes={pendingPromptPastes}
          prompt={prompt}
          queuedPrompts={queuedPrompts}
          promptResourceReferences={promptResourceReferences}
          modelId={modelId}
          modelLabel={modelLabel}
          modelOptions={modelOptions}
          allModelOptions={allModelOptions}
          onAddAttachments={onAddAttachments}
          onModelSelect={onModelSelect}
          onOpenToolActivationDialog={onOpenToolActivationDialog}
          onAgentModeSelect={onAgentModeSelect}
          onExtraHeightChange={onExtraHeightChange}
          onPermissionModeSelect={onPermissionModeSelect}
          onProjectAdd={onProjectAdd}
          onProjectClear={onProjectClear}
          onProjectSelect={onProjectSelect}
          onReasoningEffortSelect={onReasoningEffortSelect}
          onRemoveAttachment={onRemoveAttachment}
          onRemoveQueuedPrompt={onRemoveQueuedPrompt}
          onServiceTierSelect={onServiceTierSelect}
          onSteerQueuedPrompt={onSteerQueuedPrompt}
          permissionMode={permissionMode}
          project={project}
          projectPath={projectPath}
          projects={projects}
          agentMode={agentMode}
          agentModes={agentModes}
          attachments={attachments}
          reasoningEffort={reasoningEffort}
          serviceTier={serviceTier}
          showProjectPicker={showProjectPicker}
          showServiceTier={showServiceTier}
        />
      )}
    </div>
  );
}
