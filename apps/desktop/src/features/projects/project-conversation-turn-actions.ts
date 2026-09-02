import type { Dispatch, SetStateAction } from "react";
import { toast } from "sonner";

import {
  getTurnElapsedSeconds,
  type ConversationTurn,
} from "@/features/projects/conversation-timeline-model";
import type { LoadConversationTurnsOptions } from "@/features/projects/project-conversation-turn-loader";
import { providerSupportsServiceTier } from "@/features/projects/project-model-options";
import { hasAppBridge } from "@/lib/app-config";
import {
  cancelSessionTurn,
  deleteSessionEvent,
  listSessions,
  retrySessionTurn,
  updateSessionEvent,
  type AgentModeId,
} from "@/services/aivo";
import type { domain } from "../../../bridge/go/models";

export function useProjectConversationTurnActions({
  activeModelRef,
  activeSessionIdRef,
  agentMode,
  hasPendingTurn,
  loadConversationTurns,
  pendingStopRequestedRef,
  reasoningEffort,
  refreshPendingPermissionRequests,
  serviceTier,
  setConversationRunning,
  setSessions,
  setTurns,
  turns,
}: {
  activeModelRef: domain.ModelRef | undefined;
  activeSessionIdRef: { current: string };
  agentMode: AgentModeId;
  hasPendingTurn: boolean;
  loadConversationTurns: (
    sessionId: string,
    options?: LoadConversationTurnsOptions,
  ) => Promise<void>;
  pendingStopRequestedRef: { current: boolean };
  reasoningEffort: string;
  refreshPendingPermissionRequests: (sessionId?: string) => Promise<void>;
  serviceTier: string;
  setConversationRunning: (sessionId: string, running: boolean) => void;
  setSessions: Dispatch<SetStateAction<domain.Session[]>>;
  setTurns: Dispatch<SetStateAction<ConversationTurn[]>>;
  turns: ConversationTurn[];
}) {
  async function stopPendingTurn() {
    const sessionId = activeSessionIdRef.current;
    const turnToStop = [...turns]
      .reverse()
      .find((turn) => !turn.responseCompletedAt && !turn.stopped);
    setConversationRunning(sessionId, false);
    setTurns((currentTurns) =>
      currentTurns.map((turn) =>
        turn.responseCompletedAt || turn.stopped
          ? turn
          : {
              ...turn,
              stopped: true,
              thinkingSeconds: getTurnElapsedSeconds(turn),
            },
      ),
    );
    if (!hasAppBridge()) return;
    if (!turnToStop?.turnId) {
      pendingStopRequestedRef.current = true;
      return;
    }
    try {
      await cancelSessionTurn({
        turnId: turnToStop.turnId,
        reason: "User stopped generation",
      } as domain.CancelTurnRequest);
      void refreshPendingPermissionRequests(sessionId);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  }

  async function editConversationUserMessage(turn: ConversationTurn) {
    if (!hasAppBridge() || !activeSessionIdRef.current || !turn.userEventId) {
      return;
    }
    const nextContent = window.prompt("编辑消息", turn.prompt);
    if (nextContent === null) return;
    const content = nextContent.trim();
    if (!content) {
      toast.error("消息不能为空");
      return;
    }
    try {
      await updateSessionEvent({ eventId: turn.userEventId, content });
      await loadConversationTurns(activeSessionIdRef.current);
      setSessions((await listSessions(50)) ?? []);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  }

  async function deleteConversationTurn(turn: ConversationTurn) {
    if (!hasAppBridge() || !activeSessionIdRef.current) return;
    const eventIds = [turn.userEventId, turn.assistantEventId].filter(
      Boolean,
    ) as string[];
    if (eventIds.length === 0) return;
    try {
      await Promise.all(
        eventIds.map((eventId) => deleteSessionEvent({ eventId })),
      );
      await loadConversationTurns(activeSessionIdRef.current);
      setSessions((await listSessions(50)) ?? []);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  }

  async function deleteConversationAssistantMessage(turn: ConversationTurn) {
    if (
      !hasAppBridge() ||
      !activeSessionIdRef.current ||
      !turn.assistantEventId
    ) {
      return;
    }
    try {
      await deleteSessionEvent({ eventId: turn.assistantEventId });
      await loadConversationTurns(activeSessionIdRef.current);
      setSessions((await listSessions(50)) ?? []);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  }

  async function retryConversationTurn(turn: ConversationTurn) {
    const sessionId = activeSessionIdRef.current;
    if (!hasAppBridge() || !sessionId || !turn.turnId || hasPendingTurn) return;
    try {
      setConversationRunning(sessionId, true);
      await retrySessionTurn({
        sessionId,
        turnId: turn.turnId,
        model: activeModelRef,
        agentMode,
        reasoningEffort,
        serviceTier:
          activeModelRef &&
          providerSupportsServiceTier(activeModelRef.providerId)
            ? serviceTier
            : "default",
      });
      await loadConversationTurns(sessionId, { snapToBottomAfterLoad: true });
      void refreshPendingPermissionRequests(sessionId);
      setSessions((await listSessions(50)) ?? []);
    } catch (err) {
      setConversationRunning(sessionId, false);
      toast.error(err instanceof Error ? err.message : String(err));
    }
  }

  return {
    deleteConversationAssistantMessage,
    deleteConversationTurn,
    editConversationUserMessage,
    retryConversationTurn,
    stopPendingTurn,
  };
}
