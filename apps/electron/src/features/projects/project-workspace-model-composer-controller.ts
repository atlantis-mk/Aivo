import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type Dispatch,
  type SetStateAction,
} from "react";
import { toast } from "sonner";

import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";
import type { LoadConversationTurnsOptions } from "@/features/projects/project-conversation-turn-loader";
import { useProjectComposerAttachmentState } from "@/features/projects/project-composer-attachment-state";
import { useProjectModelRuntimeState } from "@/features/projects/project-model-runtime-state";
import {
  addPromptMentionReference,
  removePromptMentionReference,
  type PromptMentionReference,
} from "@/features/projects/project-prompt-mention-model";
import { useProjectSubmitPromptAction } from "@/features/projects/project-submit-prompt-action";
import { parsePromptSubmission } from "@/features/projects/project-prompt-editor-model";
import { codexResourceInputs } from "@/codex-composer-resources";
import {
  createPromptPaste,
  type PendingPromptPaste,
} from "@/features/projects/project-prompt-paste";
import { useProjectWorkspacePreferenceActions } from "@/features/projects/project-workspace-preference-actions";
import { hasCodexDesktopBridge } from "@/lib/app-config";
import type { CatalogState } from "@/lib/provider-catalog";
import type { AgentModeId, AgentRun } from "@/services/aivo";
import type { domain } from "@/types/codex-domain";

export type QueuedPrompt = {
  id: string;
  text: string;
  references?: PromptMentionReference[];
};

export function useProjectWorkspaceModelComposerController({
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
}: {
  activeRunningSubagentRun?: AgentRun;
  activeSessionId: string;
  activeSessionIdRef: { current: string };
  agentMode: AgentModeId;
  catalog: CatalogState | null;
  config: domain.AppConfig | null;
  pendingActiveToolNames: string[];
  hasPendingTurn: boolean;
  loadConversationTurns: (
    sessionId: string,
    options?: LoadConversationTurnsOptions,
  ) => Promise<void>;
  pendingStopRequestedRef: { current: boolean };
  prompt: string;
  refreshAgentRuntimeState: (sessionId?: string) => Promise<void>;
  selectedProjectPath: string;
  setActiveSessionId: Dispatch<SetStateAction<string>>;
  setAgentMode: Dispatch<SetStateAction<AgentModeId>>;
  setCodingWorkspaceRoot: Dispatch<SetStateAction<string>>;
  setConfig: (config: domain.AppConfig) => void;
  setConversationRunning: (sessionId: string, running: boolean) => void;
  setPrompt: Dispatch<SetStateAction<string>>;
  setPendingActiveToolNames: Dispatch<SetStateAction<string[]>>;
  setSessions: Dispatch<SetStateAction<domain.Session[]>>;
  setTurns: Dispatch<SetStateAction<ConversationTurn[]>>;
  turns: ConversationTurn[];
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
  const permissionModeRef = useRef(permissionMode);
  const [queuedPrompts, setQueuedPrompts] = useState<QueuedPrompt[]>([]);
  const [promptResourceReferences, setPromptResourceReferences] = useState<
    PromptMentionReference[]
  >([]);
  const [pendingPromptPastes, setPendingPromptPastes] = useState<
    PendingPromptPaste[]
  >([]);
  useEffect(() => {
    permissionModeRef.current = permissionMode;
  }, [permissionMode]);
  useEffect(() => {
    setPromptResourceReferences([]);
    setPendingPromptPastes([]);
  }, [activeSessionId]);
  const changePrompt = useCallback(
    (nextPrompt: string) => {
      setPrompt(nextPrompt);
    },
    [setPrompt],
  );
  const handlePromptPaste = useCallback(
    (pastedText: string, sourcePath?: string) => {
      const result = createPromptPaste({ pastedText, sourcePath });
      if (!result) return null;

      setPendingPromptPastes((current) => [...current, result.paste]);
      return result;
    },
    [],
  );
  const removePromptPaste = useCallback((id: string) => {
    setPendingPromptPastes((current) =>
      current.filter((paste) => paste.id !== id),
    );
  }, []);

  const queuePrompt = useCallback((text: string, references: PromptMentionReference[] = []) => {
    setQueuedPrompts((currentPrompts) => [
      ...currentPrompts,
      { id: crypto.randomUUID(), text, references },
    ]);
  }, []);
  const selectPromptMention = useCallback(
    (reference: PromptMentionReference) => {
      setPromptResourceReferences((current) =>
        addPromptMentionReference(current, reference),
      );
    },
    [],
  );
  const removePromptMention = useCallback(
    (reference: PromptMentionReference) => {
      setPromptResourceReferences((current) =>
        removePromptMentionReference(current, reference),
      );
    },
    [],
  );
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
    permissionModeRef,
    reasoningEffort,
    refreshAgentRuntimeState,
    serviceTier,
    setAgentMode,
    setLocalPermissionMode,
    setConfig,
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
    defaultWorkspacePath:
      (config as { initialWorkspacePath?: string } | null)
        ?.initialWorkspacePath ?? "",
    pendingActiveToolNames,
    hasPendingTurn,
    loadConversationTurns,
    modelOptions,
    pendingStopRequestedRef,
    permissionModeRef,
    prompt,
    queuePrompt,
    promptResourceReferences,
    pendingPromptPastes,
    reasoningEffort,
    selectedProjectPath,
    serviceTier,
    setActiveSessionId,
    setCodingWorkspaceRoot,
    setComposerAttachments,
    setConversationRunning,
    setPrompt,
    setPromptResourceReferences,
    setPendingPromptPastes,
    setPendingActiveToolNames,
    setSessions,
    setTurns,
    turns,
  });

  const submitPromptRef = useRef(submitPrompt);
  const dispatchingQueuedPromptRef = useRef(false);
  useEffect(() => {
    submitPromptRef.current = submitPrompt;
  }, [submitPrompt]);
  useEffect(() => {
    if (hasPendingTurn || dispatchingQueuedPromptRef.current) return;
    const queuedPrompt = queuedPrompts[0];
    if (!queuedPrompt) return;

    dispatchingQueuedPromptRef.current = true;
    setQueuedPrompts((currentPrompts) => currentPrompts.slice(1));
    void submitPromptRef.current(queuedPrompt.text, queuedPrompt.references).finally(() => {
      dispatchingQueuedPromptRef.current = false;
    });
  }, [hasPendingTurn, queuedPrompts]);

  const removeQueuedPrompt = useCallback((id: string) => {
    setQueuedPrompts((currentPrompts) =>
      currentPrompts.filter((prompt) => prompt.id !== id),
    );
  }, []);

  const steerQueuedPrompt = useCallback(
    async (id: string) => {
      const queuedPrompt = queuedPrompts.find((prompt) => prompt.id === id);
      if (!queuedPrompt) return;

      const activeTurn = [...turns]
        .reverse()
        .find(
          (turn) => turn.turnId && !turn.responseCompletedAt && !turn.stopped,
        );
      if (!hasCodexDesktopBridge() || !activeSessionId || !activeTurn?.turnId) {
        toast.error("当前没有可以调整方向的回合。");
        return;
      }

      const localTurnId = crypto.randomUUID();
      const startedAt = Date.now();
      setTurns((currentTurns) =>
        currentTurns.map((turn) =>
          turn.turnId === activeTurn.turnId
            ? { ...turn, hasSteeredInput: true }
            : turn,
        ),
      );
      setQueuedPrompts((currentPrompts) =>
        currentPrompts.filter((prompt) => prompt.id !== id),
      );
      setTurns((currentTurns) => [
        ...currentTurns,
        {
          id: localTurnId,
          model: undefined,
          modelProvider: undefined,
          activityVisible: false,
          assistantPreambles: [],
          attachments: [],
          prompt: parsePromptSubmission(queuedPrompt.text, queuedPrompt.references ?? []).text,
          preToolText: "",
          responseText: "",
          responseCompletedAt: null,
          responseVisible: false,
          startedAt,
          submittedAt: new Date(),
          steerTargetTurnId: activeTurn.turnId,
          steered: true,
          stopped: false,
          thinkingSeconds: 0,
          toolCalls: [],
        },
      ]);

      try {
        await window.aivoDesktop.codex.steerTurn({
          resourceInputs: codexResourceInputs(parsePromptSubmission(queuedPrompt.text, queuedPrompt.references ?? []).references),
          clientUserMessageId: localTurnId,
          expectedTurnId: activeTurn.turnId,
          text: parsePromptSubmission(queuedPrompt.text, queuedPrompt.references ?? []).text,
          threadId: activeSessionId,
        });
      } catch (error) {
        setQueuedPrompts((currentPrompts) => [queuedPrompt, ...currentPrompts]);
        setTurns((currentTurns) =>
          currentTurns.map((turn) =>
            turn.id === localTurnId
              ? {
                  ...turn,
                  responseCompletedAt: new Date(),
                  responseText:
                    error instanceof Error ? error.message : String(error),
                  responseVisible: true,
                  thinkingSeconds: 0,
                  toolCalls: [],
                }
              : turn,
          ),
        );
      }
    },
    [activeSessionId, queuedPrompts, setTurns, turns],
  );

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
    removePromptMention,
    changePrompt,
    handlePromptPaste,
    removePromptPaste,
    pendingPromptPastes,
    selectAgentMode,
    selectModel,
    selectPermissionMode,
    selectPromptMention,
    promptResourceReferences,
    selectReasoningEffort,
    selectServiceTier,
    serviceTier,
    submitPrompt,
    queuedPrompts,
    removeQueuedPrompt,
    steerQueuedPrompt,
  };
}
