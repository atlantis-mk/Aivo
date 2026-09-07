import type { Dispatch, SetStateAction } from "react";
import { toast } from "sonner";

import {
  getTurnElapsedSeconds,
  type ConversationTurn,
} from "@/features/projects/conversation-timeline-model";
import type { LoadConversationTurnsOptions } from "@/features/projects/project-conversation-turn-loader";
import {
  attachmentKindLabel,
  composerAttachmentToConversationAttachment,
  formatAttachmentOnlyPrompt,
  modelSupportsAttachment,
  type ComposerAttachment,
} from "@/features/projects/project-composer-attachments";
import { providerSupportsServiceTier } from "@/features/projects/project-model-options";
import {
  activePromptMentionReferences,
  type PromptMentionReference,
} from "@/features/projects/project-prompt-mention-model";
import {
  expandPromptPastes,
  promptWithPasteSummaries,
  type PendingPromptPaste,
} from "@/features/projects/project-prompt-paste";
import { consumePendingToolActivation } from "@/features/projects/project-tool-activation-scope";
import { hasCodexDesktopBridge } from "@/lib/app-config";
import { listCodexSessions } from "@/services/codex-thread-service";
import { upsertSession } from "@/features/projects/project-conversation-permissions";
import type { ModelInfo } from "@/lib/provider-catalog";
import type { PermissionMode } from "@/codex-app-server";
import {
  cancelSessionTurn,
  createSession,
  invokeCommand,
  listCommandCatalog,
  parseCommandArgumentLine,
  listSessions,
  setSessionAgentMode,
  setSessionActiveTools,
  submitSessionMessage,
  type AgentModeId,
} from "@/services/aivo";
import { domain } from "@/types/codex-domain";

export function useProjectSubmitPromptAction({
  activeModelId,
  activeModelRef,
  activeSessionId,
  activeSessionIdRef,
  agentMode,
  composerAttachments,
  defaultWorkspacePath,
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
}: {
  activeModelId: string;
  activeModelRef: domain.ModelRef | undefined;
  activeSessionId: string;
  activeSessionIdRef: { current: string };
  agentMode: AgentModeId;
  composerAttachments: ComposerAttachment[];
  defaultWorkspacePath: string;
  pendingActiveToolNames: string[];
  hasPendingTurn: boolean;
  loadConversationTurns: (
    sessionId: string,
    options?: LoadConversationTurnsOptions,
  ) => Promise<void>;
  modelOptions: ModelInfo[];
  pendingStopRequestedRef: { current: boolean };
  permissionModeRef: { current: PermissionMode };
  prompt: string;
  queuePrompt: (text: string) => void;
  promptResourceReferences: PromptMentionReference[];
  pendingPromptPastes: PendingPromptPaste[];
  reasoningEffort: string;
  selectedProjectPath: string;
  serviceTier: string;
  setActiveSessionId: Dispatch<SetStateAction<string>>;
  setCodingWorkspaceRoot: Dispatch<SetStateAction<string>>;
  setComposerAttachments: Dispatch<SetStateAction<ComposerAttachment[]>>;
  setConversationRunning: (sessionId: string, running: boolean) => void;
  setPrompt: Dispatch<SetStateAction<string>>;
  setPromptResourceReferences: Dispatch<
    SetStateAction<PromptMentionReference[]>
  >;
  setPendingPromptPastes: Dispatch<SetStateAction<PendingPromptPaste[]>>;
  setPendingActiveToolNames: Dispatch<SetStateAction<string[]>>;
  setSessions: Dispatch<SetStateAction<domain.Session[]>>;
  setTurns: Dispatch<SetStateAction<ConversationTurn[]>>;
  turns: ConversationTurn[];
}) {
  async function submitPrompt(promptOverride?: string) {
    const nextPrompt =
      promptOverride ?? expandPromptPastes(prompt, pendingPromptPastes).trim();
    if (!nextPrompt && composerAttachments.length === 0) {
      return;
    }
    const activeModel = modelOptions.find(
      (model) => model.id === activeModelId,
    );
    const submittedResourceReferences = activePromptMentionReferences(
      promptResourceReferences,
    );
    const submittedProjectPath =
      submittedResourceReferences.find(
        (reference) => reference.kind === "project",
      )?.rootPath || selectedProjectPath;
    const unsupportedAttachment = composerAttachments.find(
      (attachment) =>
        !modelSupportsAttachment(
          activeModelRef,
          activeModel,
          attachment.kind,
          attachment.mimeType,
          attachment.name,
        ),
    );
    if (unsupportedAttachment) {
      toast.error(
        `当前模型不支持${attachmentKindLabel(unsupportedAttachment.kind, unsupportedAttachment.mimeType)}：${unsupportedAttachment.name}`,
      );
      return;
    }
    if (hasCodexDesktopBridge() && composerAttachments.length > 0) {
      toast.error("当前 Codex 桌面对话暂不支持发送附件。");
      return;
    }
    const localTurnId = crypto.randomUUID();
    const startedAt = Date.now();
    const submittedAttachments = composerAttachments;
    const submittedTimelineAttachments = submittedAttachments.map(
      composerAttachmentToConversationAttachment,
    );
    const displayPrompt =
      (promptOverride ??
        promptWithPasteSummaries(prompt, pendingPromptPastes)) ||
      formatAttachmentOnlyPrompt(submittedAttachments);
    if (hasPendingTurn) {
      queuePrompt(displayPrompt);
      setPrompt("");
      setPromptResourceReferences([]);
      setPendingPromptPastes([]);
      setComposerAttachments([]);
      return;
    }
    setTurns((currentTurns) => [
      ...currentTurns,
      {
        id: localTurnId,
        model: activeModelRef?.modelId,
        modelProvider: activeModelRef?.providerId,
        activityVisible: false,
        assistantPreambles: [],
        attachments: submittedTimelineAttachments,
        prompt: displayPrompt,
        preToolText: "",
        responseText: "",
        responseCompletedAt: null,
        responseVisible: false,
        startedAt,
        submittedAt: new Date(),
        stopped: false,
        thinkingSeconds: 0,
        toolCalls: [],
      },
    ]);
    setPrompt("");
    setPromptResourceReferences([]);
    setPendingPromptPastes([]);
    setComposerAttachments([]);
    if (!hasCodexDesktopBridge()) {
      setTurns((currentTurns) =>
        currentTurns.map((turn) =>
          turn.id === localTurnId
            ? {
                ...turn,
                responseCompletedAt: new Date(),
                responseText:
                  "当前运行环境未连接 Aivo 后端，无法发送真实 provider 请求。",
                responseVisible: true,
                thinkingSeconds: getTurnElapsedSeconds({ startedAt }),
                toolCalls: [],
              }
            : turn,
        ),
      );
      return;
    }
    let submittedSessionId = activeSessionId;
    try {
      let threadId = submittedSessionId;
      if (!threadId) {
        const workspacePath = submittedProjectPath || defaultWorkspacePath;
        const thread = await window.aivoDesktop.codex.startThread({
          cwd: workspacePath || undefined,
          model: activeModelRef?.modelId || activeModelId || undefined,
          modelProvider: activeModelRef?.providerId || undefined,
          permissionMode: permissionModeRef.current,
        });
        threadId = thread.threadId;
        activeSessionIdRef.current = threadId;
        setActiveSessionId(threadId);
        setCodingWorkspaceRoot(workspacePath);
        const now = new Date().toISOString();
        const optimisticSession = new domain.Session({
          id: threadId,
          type: "coding",
          status: "inProgress",
          source: "codex",
          title: "新对话",
          projectPath: workspacePath,
          model: activeModelRef
            ? {
                providerId: activeModelRef.providerId,
                modelId: activeModelRef.modelId,
              }
            : undefined,
          timeCreated: now,
          timeUpdated: now,
        });
        setSessions((currentSessions) =>
          upsertSession(currentSessions, optimisticSession),
        );
        const fetchedSessions = await listCodexSessions(50);
        setSessions(
          fetchedSessions.some((session) => session.id === threadId)
            ? fetchedSessions
            : [optimisticSession, ...fetchedSessions],
        );
      }
      submittedSessionId = threadId;
      const turn = await window.aivoDesktop.codex.startTurn({
        model: activeModelRef?.modelId || activeModelId || undefined,
        modelProvider: activeModelRef?.providerId || undefined,
        permissionMode: permissionModeRef.current,
        text: nextPrompt,
        threadId,
      });
      setConversationRunning(threadId, true);
      setTurns((currentTurns) =>
        currentTurns.map((currentTurn) =>
          currentTurn.id === localTurnId
            ? { ...currentTurn, turnId: turn.turnId }
            : currentTurn,
        ),
      );
      return;
    } catch (err) {
      setComposerAttachments((current) => [
        ...submittedAttachments,
        ...current,
      ]);
      setConversationRunning(submittedSessionId, false);
      setTurns((currentTurns) =>
        currentTurns.map((turn) =>
          turn.id === localTurnId
            ? {
                ...turn,
                responseCompletedAt: new Date(),
                responseText: err instanceof Error ? err.message : String(err),
                responseVisible: true,
                thinkingSeconds: getTurnElapsedSeconds({ startedAt }),
                toolCalls: [],
              }
            : turn,
        ),
      );
    }
  }

  return { submitPrompt };
}
