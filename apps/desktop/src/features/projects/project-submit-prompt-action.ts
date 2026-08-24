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
import { consumePendingToolActivation } from "@/features/projects/project-tool-activation-scope";
import { hasAppBridge } from "@/lib/app-config";
import type { ModelInfo } from "@/lib/provider-catalog";
import {
  cancelSessionTurn,
  createSession,
  invokeCommand,
  listCommandCatalog,
  parseCommandArgumentLine,
  listSessions,
  setPermissionMode,
  setSessionAgentMode,
  setSessionActiveTools,
  submitSessionMessage,
  type AgentModeId,
  type PermissionMode,
} from "@/services/aivo";
import type { domain } from "../../../bridge/go/models";

export function useProjectSubmitPromptAction({
  activeModelId,
  activeModelRef,
  activeSessionId,
  activeSessionIdRef,
  agentMode,
  composerAttachments,
  pendingActiveToolNames,
  hasPendingTurn,
  loadConversationTurns,
  modelOptions,
  pendingStopRequestedRef,
  permissionModeRef,
  prompt,
  promptResourceReferences,
  reasoningEffort,
  refreshPendingPermissionRequests,
  selectedProjectPath,
  serviceTier,
  setActiveSessionId,
  setCodingWorkspaceRoot,
  setComposerAttachments,
  setConversationRunning,
  setPrompt,
  setPromptResourceReferences,
  setPendingActiveToolNames,
  setSessions,
  setTurns,
}: {
  activeModelId: string;
  activeModelRef: domain.ModelRef | undefined;
  activeSessionId: string;
  activeSessionIdRef: { current: string };
  agentMode: AgentModeId;
  composerAttachments: ComposerAttachment[];
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
  promptResourceReferences: PromptMentionReference[];
  reasoningEffort: string;
  refreshPendingPermissionRequests: (sessionId?: string) => Promise<void>;
  selectedProjectPath: string;
  serviceTier: string;
  setActiveSessionId: Dispatch<SetStateAction<string>>;
  setCodingWorkspaceRoot: Dispatch<SetStateAction<string>>;
  setComposerAttachments: Dispatch<SetStateAction<ComposerAttachment[]>>;
  setConversationRunning: (sessionId: string, running: boolean) => void;
  setPrompt: Dispatch<SetStateAction<string>>;
  setPromptResourceReferences: Dispatch<SetStateAction<PromptMentionReference[]>>;
  setPendingActiveToolNames: Dispatch<SetStateAction<string[]>>;
  setSessions: Dispatch<SetStateAction<domain.Session[]>>;
  setTurns: Dispatch<SetStateAction<ConversationTurn[]>>;
}) {
  async function submitPrompt() {
    const nextPrompt = prompt.trim();
    if ((!nextPrompt && composerAttachments.length === 0) || hasPendingTurn) {
      return;
    }
    const activeModel = modelOptions.find((model) => model.id === activeModelId);
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
    const localTurnId = crypto.randomUUID();
    const startedAt = Date.now();
    const submittedAttachments = composerAttachments;
    const submittedTimelineAttachments = submittedAttachments.map(
      composerAttachmentToConversationAttachment,
    );
    const displayPrompt =
      nextPrompt || formatAttachmentOnlyPrompt(submittedAttachments);
    setTurns((currentTurns) => [
      ...currentTurns,
      {
        id: localTurnId,
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
    setComposerAttachments([]);
    if (!hasAppBridge()) {
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
      let sessionId = submittedSessionId;
      if (!sessionId) {
        const session = await createSession({
          type: "coding",
          source: "desktop",
          projectPath: submittedProjectPath,
          model: activeModelRef,
          agentMode,
        } as domain.CreateSessionRequest & { agentMode?: AgentModeId });
        sessionId = session.id;
        activeSessionIdRef.current = session.id;
        setActiveSessionId(session.id);
        setCodingWorkspaceRoot(submittedProjectPath);
        const pendingActivation = consumePendingToolActivation(
          pendingActiveToolNames,
        );
        setPendingActiveToolNames(pendingActivation.remainingToolNames);
        if (pendingActivation.appliedToolNames.length > 0) {
          await setSessionActiveTools(
            session.id,
            pendingActivation.appliedToolNames,
          );
        }
      }
      submittedSessionId = sessionId;
      let providerPrompt = nextPrompt;
      let providerAgentMode = agentMode;
      let providerModelRef = activeModelRef;
      const commandMatch = nextPrompt.match(/^\/([^\s]+)(?:\s+([\s\S]*))?$/);
      if (commandMatch) {
        const [, commandName, argumentLine = ""] = commandMatch;
        const catalog = await listCommandCatalog(submittedProjectPath);
        const command = catalog.find(
          (entry) => entry.name === commandName || entry.id === commandName,
        );
        if (!command) {
          throw new Error(`未知命令 /${commandName}`);
        }
        const tokens = parseCommandArgumentLine(argumentLine);
        const args: Record<string, string> = { ARGUMENTS: argumentLine };
        command.arguments?.forEach((argument, index) => {
          if (tokens[index] !== undefined) args[argument.name] = tokens[index];
        });
        const expanded = await invokeCommand({
          sessionId,
          projectPath: submittedProjectPath,
          commandId: command.id,
          arguments: args,
        });
        if (expanded.subtask) {
          setTurns((currentTurns) =>
            currentTurns.map((turn) =>
              turn.id === localTurnId
                ? {
                    ...turn,
                    responseCompletedAt: new Date(),
                    responseText:
                      expanded.response || "子任务已完成，但没有返回文本。",
                    responseVisible: true,
                    thinkingSeconds: getTurnElapsedSeconds({ startedAt }),
                  }
                : turn,
            ),
          );
          setSessions((await listSessions(50)) ?? []);
          return;
        }
        providerPrompt = expanded.prompt;
        if (expanded.agent) {
          providerAgentMode = expanded.agent;
          await setSessionAgentMode(sessionId, expanded.agent);
        }
        if (expanded.model) providerModelRef = expanded.model;
        if (expanded.toolsets?.length) {
          await setSessionActiveTools(sessionId, expanded.toolsets);
        }
      }
      await setPermissionMode(sessionId, permissionModeRef.current);
      setConversationRunning(sessionId, true);
      const run = await submitSessionMessage({
        sessionId,
        text: providerPrompt,
        attachments: submittedAttachments.map((attachment) => ({
          id: attachment.id,
          name: attachment.name,
          mimeType: attachment.mimeType,
          kind: attachment.kind,
          data: attachment.data,
          text: attachment.text,
          size: attachment.size,
        })),
        resourceReferences: submittedResourceReferences.map(
          ({ id, kind, rootPath }) => ({
            id,
            kind,
            rootPath,
          }),
        ),
        model: providerModelRef,
        agentMode: providerAgentMode,
        reasoningEffort,
        serviceTier:
          providerModelRef &&
          providerSupportsServiceTier(providerModelRef.providerId)
            ? serviceTier
            : "default",
      } as domain.SubmitSessionMessageRequest & {
        agentMode?: AgentModeId;
        attachments?: Array<{
          id: string;
          name: string;
          mimeType: string;
          kind: string;
          data: string;
          text?: string;
          size: number;
        }>;
        resourceReferences?: Array<{
          id: string;
          kind: PromptMentionReference["kind"];
          rootPath?: string;
        }>;
      });
      void refreshPendingPermissionRequests(sessionId);
      if (run.turn?.id || run.userEvent?.id) {
        setTurns((currentTurns) =>
          currentTurns.map((turn) =>
            turn.id === localTurnId
              ? {
                  ...turn,
                  turnId: run.turn?.id || turn.turnId,
                  userEventId: run.userEvent?.id || turn.userEventId,
                }
              : turn,
          ),
        );
        if (pendingStopRequestedRef.current) {
          pendingStopRequestedRef.current = false;
          await cancelSessionTurn({
            turnId: run.turn.id,
            reason: "User stopped generation",
          } as domain.CancelTurnRequest);
          void refreshPendingPermissionRequests(sessionId);
          setConversationRunning(sessionId, false);
          setSessions((await listSessions(50)) ?? []);
          return;
        }
      }
      // The streaming RPC returns the initially prepared turn while execution
      // continues in the background. Always reconcile once after binding the
      // real ids so an immediate failure cannot be lost before the SSE turn
      // event is associated with the optimistic local turn.
      await loadConversationTurns(sessionId, {
        pendingTurnId: localTurnId,
        pendingPrompt: displayPrompt,
        pendingAttachments: submittedTimelineAttachments,
        pendingStartedAt: startedAt,
        fallbackAssistantEvent: run.assistantEvent,
      });
      setSessions((await listSessions(50)) ?? []);
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
