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
import { runtimeMetricsFromEventPayload } from "@/features/projects/project-session-runtime-stats";
import {
  listSessionEvents,
  listSessionToolCalls,
  listSessionTurns,
} from "@/services/aivo";
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
