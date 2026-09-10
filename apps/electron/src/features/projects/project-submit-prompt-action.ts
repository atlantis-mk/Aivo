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
  promptWithTextAttachments,
  type ComposerAttachment,
} from "@/features/projects/project-composer-attachments";
import { providerSupportsServiceTier } from "@/features/projects/project-model-options";
import {
  type PromptMentionReference,
} from "@/features/projects/project-prompt-mention-model";
import { parsePromptSubmission } from "@/features/projects/project-prompt-editor-model";
import { codexResourceInputs } from "@/codex-composer-resources";
import {
  expandPromptPastes,
  promptPasteTextElements,
  promptPasteTitle,
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
  queuePrompt: (text: string, references?: PromptMentionReference[]) => void;
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
  async function submitPrompt(promptOverride?: string, referencesOverride?: PromptMentionReference[]) {
    const draft = promptOverride ?? prompt;
    const submitted = parsePromptSubmission(draft.trim(), referencesOverride ?? promptResourceReferences);
    const nextPrompt =
      pendingPromptPastes.length > 0
        ? expandPromptPastes(submitted.text, pendingPromptPastes)
        : submitted.text;
    if (!nextPrompt.trim() && composerAttachments.length === 0) {
      return;
    }
    const activeModel = modelOptions.find(
      (model) => model.id === activeModelId,
    );
    const submittedResourceReferences = submitted.references;
    const resourceInputs = codexResourceInputs(submittedResourceReferences);
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
    const unsupportedDesktopAttachment = hasCodexDesktopBridge()
      ? composerAttachments.find(
          (attachment) =>
            attachment.kind !== "image" &&
            attachment.kind !== "directory" &&
            typeof attachment.text !== "string",
        )
      : undefined;
    if (unsupportedDesktopAttachment) {
      toast.error(
        `当前 Codex 桌面对话支持图片和文本文件：${unsupportedDesktopAttachment.name} 无法发送。`,
      );
      return;
    }
    const localTurnId = crypto.randomUUID();
    const startedAt = Date.now();
    const submittedAttachments = composerAttachments;
    const submittedPastes = pendingPromptPastes.map((paste) => ({
      id: paste.id,
      name: promptPasteTitle(paste),
      preview: paste.text.trim().split("\n").find(Boolean)?.trim() || promptPasteTitle(paste),
      text: paste.text,
    }));
    const transportPrompt = hasCodexDesktopBridge()
      ? promptWithTextAttachments(nextPrompt, submittedAttachments)
      : nextPrompt;
    const textElements =
      hasCodexDesktopBridge()
        ? promptPasteTextElements(submitted.text, pendingPromptPastes)
        : [];
    const submittedTimelineAttachments = submittedAttachments.map(
      composerAttachmentToConversationAttachment,
    );
    const displayPrompt =
      submitted.text ||
      formatAttachmentOnlyPrompt(submittedAttachments);
    if (hasPendingTurn) {
      queuePrompt(
        promptWithTextAttachments(
          expandPromptPastes(draft, pendingPromptPastes),
          composerAttachments.filter(
            (attachment) => attachment.kind === "directory",
          ),
        ),
        submittedResourceReferences,
      );
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
        pastes: submittedPastes,
        prompt: displayPrompt,
        preToolText: "",
        responseText: "",
        responseCompletedAt: null,
        responseVisible: false,
        sessionId: activeSessionId,
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
        setTurns((currentTurns) =>
          currentTurns.map((currentTurn) =>
            currentTurn.id === localTurnId
              ? { ...currentTurn, sessionId: threadId }
              : currentTurn,
          ),
        );
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
      setConversationRunning(threadId, true);
      const turn = await window.aivoDesktop.codex.startTurn({
        resourceInputs,
        images: submittedAttachments
          .filter((attachment) => attachment.kind === "image")
          .map(({ data, mimeType }) => ({ data, mimeType })),
        model: activeModelRef?.modelId || activeModelId || undefined,
        modelProvider: activeModelRef?.providerId || undefined,
        permissionMode: permissionModeRef.current,
        text: transportPrompt,
        textElements,
        threadId,
      });
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
      setSessions((currentSessions) =>
        currentSessions.map((session) =>
          session.id === submittedSessionId
            ? domain.Session.createFrom({ ...session, status: "failed" })
            : session,
        ),
      );
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
