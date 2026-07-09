import type { Dispatch, SetStateAction } from "react";

import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";
import type { LoadConversationTurnsOptions } from "@/features/projects/project-conversation-turn-loader";
import { useProjectComposerAttachmentState } from "@/features/projects/project-composer-attachment-state";
import { useProjectModelRuntimeState } from "@/features/projects/project-model-runtime-state";
import { useProjectSubmitPromptAction } from "@/features/projects/project-submit-prompt-action";
import { useProjectWorkspacePreferenceActions } from "@/features/projects/project-workspace-preference-actions";
import type { CatalogState } from "@/lib/provider-catalog";
import type {
  AgentModeId,
  AgentRun,
} from "@/services/aivo";
import type { domain } from "../../../bridge/go/models";

export function useProjectWorkspaceModelComposerController({
  activeRunningSubagentRun,
  activeSessionId,
  activeSessionIdRef,
  agentMode,
  catalog,
  config,
  defaultActiveToolNames,
  hasPendingTurn,
  loadConversationTurns,
  pendingStopRequestedRef,
  prompt,
  refreshAgentRuntimeState,
  refreshPendingPermissionRequests,
  selectedProjectPath,
  setActiveSessionId,
  setAgentMode,
  setCodingWorkspaceRoot,
  setConversationRunning,
  setPrompt,
  setSessions,
  setTurns,
}: {
  activeRunningSubagentRun?: AgentRun;
  activeSessionId: string;
  activeSessionIdRef: { current: string };
  agentMode: AgentModeId;
  catalog: CatalogState | null;
  config: domain.AppConfig | null;
  defaultActiveToolNames: string[];
  hasPendingTurn: boolean;
  loadConversationTurns: (
    sessionId: string,
    options?: LoadConversationTurnsOptions,
  ) => Promise<void>;
  pendingStopRequestedRef: { current: boolean };
  prompt: string;
  refreshAgentRuntimeState: (sessionId?: string) => Promise<void>;
  refreshPendingPermissionRequests: (
    sessionId?: string,
  ) => Promise<void>;
  selectedProjectPath: string;
  setActiveSessionId: Dispatch<SetStateAction<string>>;
  setAgentMode: Dispatch<SetStateAction<AgentModeId>>;
  setCodingWorkspaceRoot: Dispatch<SetStateAction<string>>;
  setConversationRunning: (sessionId: string, running: boolean) => void;
  setPrompt: Dispatch<SetStateAction<string>>;
  setSessions: Dispatch<SetStateAction<domain.Session[]>>;
  setTurns: Dispatch<SetStateAction<ConversationTurn[]>>;
}) {
  const {
    activeModelId,
    activeModelRef,
    allModelOptions,
    modelOptions,
    permissionMode,
    reasoningEffort,
    serviceTier,
    setLocalPermissionMode,
    setReasoningEffort,
    setSelectedModelId,
    setSelectedProviderId,
    setServiceTier,
  } = useProjectModelRuntimeState({
    activeSessionId,
    catalog,
    config,
  });
  const {
    addFiles: addComposerAttachmentFiles,
    attachments: composerAttachments,
    handleDragEnter: handleComposerDragEnter,
    handleDragLeave: handleComposerDragLeave,
    handleDragOver: handleComposerDragOver,
    handleDrop: handleComposerDrop,
    isDropActive: isComposerDropActive,
    removeAttachment: removeComposerAttachment,
    setAttachments: setComposerAttachments,
  } = useProjectComposerAttachmentState({
    activeModelId,
    activeModelRef,
    modelOptions,
  });
  const {
    cancelActiveSubagentRun,
    selectAgentMode,
    selectModel,
    selectPermissionMode,
    selectReasoningEffort,
    selectServiceTier,
  } = useProjectWorkspacePreferenceActions({
    activeModelRef,
    activeRunningSubagentRun,
    activeSessionId,
    activeSessionIdRef,
    reasoningEffort,
    refreshAgentRuntimeState,
    serviceTier,
    setAgentMode,
    setLocalPermissionMode,
    setReasoningEffort,
    setSelectedModelId,
    setSelectedProviderId,
    setServiceTier,
    setSessions,
  });
  const { submitPrompt } = useProjectSubmitPromptAction({
    activeModelId,
    activeModelRef,
    activeSessionId,
    activeSessionIdRef,
    agentMode,
    composerAttachments,
    defaultActiveToolNames,
    hasPendingTurn,
    loadConversationTurns,
    modelOptions,
    pendingStopRequestedRef,
    permissionMode,
    prompt,
    reasoningEffort,
    refreshPendingPermissionRequests,
    selectedProjectPath,
    serviceTier,
    setActiveSessionId,
    setCodingWorkspaceRoot,
    setComposerAttachments,
    setConversationRunning,
    setPrompt,
    setSessions,
    setTurns,
  });

  return {
    activeModelId,
    activeModelRef,
    addComposerAttachmentFiles,
    allModelOptions,
    cancelActiveSubagentRun,
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
    selectAgentMode,
    selectModel,
    selectPermissionMode,
    selectReasoningEffort,
    selectServiceTier,
    serviceTier,
    submitPrompt,
  };
}
