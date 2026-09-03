import { useCallback } from "react";
import type { Dispatch, SetStateAction } from "react";

import {
  getTurnElapsedSeconds,
  type ConversationTurn,
  type ConversationUserAttachment,
} from "@/features/projects/conversation-timeline-model";
import {
  applyPendingTurnMetadata,
  hasRunningTurn,
  mergePreservedTurnAttachments,
  mergeTurnPauseMetadata,
  toolCallsForTurn,
  turnsFromEvents,
} from "@/features/projects/project-conversation-events";
import { parseTime } from "@/features/projects/project-time-model";
import { codexToolCallFromItem } from "@/features/projects/project-codex-tool-calls";
import { runtimeMetricsFromEventPayload } from "@/features/projects/project-session-runtime-stats";
import {
  listSessionEvents,
  listSessionToolCalls,
  listSessionTurns,
} from "@/services/aivo";
import { listCodexThreadTurns } from "@/services/codex-thread-service";
import { hasCodexDesktopBridge } from "@/lib/app-config";
import type { domain } from "../../../bridge/go/models";

export type LoadConversationTurnsOptions = {
  pendingTurnId?: string;
  pendingPrompt?: string;
  pendingAttachments?: ConversationUserAttachment[];
  pendingStartedAt?: number;
  fallbackAssistantEvent?: domain.SessionEvent;
  snapToBottomAfterLoad?: boolean;
};

export function useProjectConversationTurnLoader({
  activeSessionIdRef,
  prepareConversationReveal,
  setConversationRunning,
  setTurns,
  turns,
}: {
  activeSessionIdRef: { current: string };
  prepareConversationReveal: (turnCount: number) => void;
  setConversationRunning: (sessionId: string, running: boolean) => void;
  setTurns: Dispatch<SetStateAction<ConversationTurn[]>>;
  turns: ConversationTurn[];
}) {
  return useCallback(
    async function loadConversationTurns(
      sessionId: string,
      options: LoadConversationTurnsOptions = {},
    ) {
      if (hasCodexDesktopBridge()) {
        const nextTurns = (await listCodexThreadTurns(sessionId)).map(
          (turn) => codexTurnToConversationTurn(turn, sessionId),
        );
        const hydratedTurns = mergeTurnPauseMetadata(
          applyPendingTurnMetadata(nextTurns, options),
          turns,
        );
        setConversationRunning(
          sessionId,
          hydratedTurns.some((turn) => !turn.responseCompletedAt && !turn.stopped),
        );
        if (activeSessionIdRef.current !== sessionId) return;
        if (options.snapToBottomAfterLoad) {
          prepareConversationReveal(hydratedTurns.length);
        }
        setTurns(hydratedTurns);
        return;
      }
      const [events, toolCalls, runtimeTurns] = await Promise.all([
        listSessionEvents(sessionId, false, 100),
        listSessionToolCalls(sessionId).catch(() => [] as domain.ToolCall[]),
        listSessionTurns(sessionId, 100).catch(() => [] as domain.Turn[]),
      ]);
      let nextTurns = turnsFromEvents(
        events ?? [],
        toolCalls ?? [],
        runtimeTurns ?? [],
      );
      if (
        nextTurns.length === 0 &&
        options.fallbackAssistantEvent &&
        options.pendingPrompt
      ) {
        const submittedAt = new Date(options.pendingStartedAt ?? Date.now());
        const completedAt = parseTime(
          options.fallbackAssistantEvent.timeCreated,
        );
        nextTurns = [
          {
            id: options.pendingTurnId ?? options.fallbackAssistantEvent.id,
            activityVisible: (toolCalls ?? []).some(
              (toolCall) =>
                toolCall.turnId === options.fallbackAssistantEvent?.turnId,
            ),
            assistantPreambles: [],
            prompt: options.pendingPrompt,
            attachments:
              options.pendingAttachments ??
              mergePreservedTurnAttachments(options.pendingTurnId, turns),
            preToolText: "",
            responseCompletedAt: completedAt,
            responseText: options.fallbackAssistantEvent.content ?? "",
            responseVisible: true,
            runtimeMetrics: runtimeMetricsFromEventPayload(
              options.fallbackAssistantEvent.payload,
            ),
            startedAt: submittedAt.getTime(),
            stopped: false,
            submittedAt,
            thinkingSeconds: getTurnElapsedSeconds({
              startedAt: options.pendingStartedAt ?? submittedAt.getTime(),
            }),
            toolCalls: toolCallsForTurn(
              toolCalls ?? [],
              options.fallbackAssistantEvent.turnId,
            ),
            turnId: options.fallbackAssistantEvent.turnId,
            assistantEventId: options.fallbackAssistantEvent.id,
          },
        ];
      }
      const hydratedTurns = mergeTurnPauseMetadata(
        applyPendingTurnMetadata(nextTurns, options),
        turns,
      );
      setConversationRunning(sessionId, hasRunningTurn(hydratedTurns));
      if (activeSessionIdRef.current !== sessionId) {
        return;
      }
      if (options.snapToBottomAfterLoad) {
        prepareConversationReveal(hydratedTurns.length);
      }
      setTurns(hydratedTurns);
    },
    [
      activeSessionIdRef,
      prepareConversationReveal,
      setConversationRunning,
      setTurns,
      turns,
    ],
  );
}

function codexTurnToConversationTurn(
  turn: CodexThreadTurn,
  threadId: string,
): ConversationTurn {
  const startedAt = turn.startedAt ? Date.parse(turn.startedAt) : Date.now();
  const completedAt = turn.completedAt ? new Date(turn.completedAt) : null;
  const prompt = turn.items
    .filter(isCodexItemType("userMessage"))
    .flatMap((item) => textFromUserMessage(item.content))
    .join("\n");
  const assistantText = splitCodexAssistantText(turn.items);
  const toolCalls = turn.items.flatMap((item) => {
    const toolCall = codexToolCallFromItem({
      item,
      threadId,
      timeCreated: turn.startedAt ?? undefined,
      timeUpdated: turn.completedAt ?? turn.startedAt ?? undefined,
      turnId: turn.id,
    });
    return toolCall ? [toolCall] : [];
  });

  return {
    activityVisible: turn.status === "inProgress" || toolCalls.length > 0,
    assistantPreambles: assistantText.preambles,
    attachments: [],
    id: turn.id,
    preToolText: assistantText.preambles.map((part) => part.text).join("\n"),
    prompt,
    responseCompletedAt:
      completedAt ?? (turn.status === "inProgress" ? null : new Date(startedAt)),
    responseText: assistantText.responseText || turn.error || "",
    responseVisible: Boolean(assistantText.responseText || turn.error),
    startedAt,
    stopped: turn.status === "interrupted",
    submittedAt: new Date(startedAt),
    thinkingSeconds:
      typeof turn.durationMs === "number" && Number.isFinite(turn.durationMs)
        ? Math.max(0, Math.floor(turn.durationMs / 1000))
        : getTurnElapsedSeconds({ startedAt }, completedAt?.getTime()),
    toolCalls,
    turnId: turn.id,
  };
}

function splitCodexAssistantText(items: unknown[]) {
  const firstToolIndex = items.findIndex(isCodexToolItem);
  const agentMessages = items
    .map((item, index) => ({
      index,
      item,
      text: textFromAgentMessage(item),
    }))
    .filter((entry): entry is { index: number; item: unknown; text: string } =>
      Boolean(entry.text),
    );

  if (firstToolIndex < 0) {
    return {
      preambles: [],
      responseText: agentMessages.map((entry) => entry.text).join("\n"),
    };
  }

  const preambles = agentMessages
    .filter((entry) => entry.index < firstToolIndex)
    .map((entry, index) => ({
      id: `codex-preamble:${index}`,
      text: entry.text,
      timeCreated: undefined,
    }));
  return {
    preambles,
    responseText: agentMessages
      .filter((entry) => entry.index > firstToolIndex)
      .map((entry) => entry.text)
      .join("\n"),
  };
}

function isCodexToolItem(item: unknown) {
  const type = codexItemType(item);
  return (
    type === "commandExecution" ||
    type === "mcpToolCall" ||
    type === "webSearch"
  );
}

function textFromAgentMessage(item: unknown) {
  if (codexItemType(item) !== "agentMessage") return null;
  const text = (item as Record<string, unknown>).text;
  return typeof text === "string" && text.trim() ? text : null;
}

function codexItemType(item: unknown) {
  return typeof item === "object" && item !== null
    ? (item as Record<string, unknown>).type
    : undefined;
}

function isCodexItemType(type: string) {
  return (item: unknown): item is Record<string, unknown> =>
    typeof item === "object" &&
    item !== null &&
    (item as Record<string, unknown>).type === type;
}

function textFromUserMessage(content: unknown): string[] {
  if (!Array.isArray(content)) return [];
  return content.flatMap((item) =>
    typeof item === "object" && item !== null &&
    (item as Record<string, unknown>).type === "text"
      ? textArrayFromString((item as Record<string, unknown>).text)
      : [],
  );
}

function textArrayFromString(value: unknown): string[] {
  return typeof value === "string" ? [value] : [];
}
