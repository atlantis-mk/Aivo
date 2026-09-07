import type { Dispatch, SetStateAction } from "react";
import { toast } from "sonner";

import {
  getTurnElapsedSeconds,
  type ConversationTurn,
} from "@/features/projects/conversation-timeline-model";
import type { LoadConversationTurnsOptions } from "@/features/projects/project-conversation-turn-loader";
import { hasCodexDesktopBridge } from "@/lib/app-config";
import type { AgentModeId } from "@/services/aivo";
import type { domain } from "@/types/codex-domain";

export function useProjectConversationTurnActions({
  activeModelRef,
  activeSessionIdRef,
  agentMode,
  hasPendingTurn,
  loadConversationTurns,
  pendingStopRequestedRef,
  reasoningEffort,
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
    if (!turnToStop?.turnId) {
      pendingStopRequestedRef.current = true;
      return;
    }
    if (hasCodexDesktopBridge()) {
      try {
        await window.aivoDesktop.codex.interruptTurn({
          threadId: sessionId,
          turnId: turnToStop.turnId,
        });
      } catch (err) {
        toast.error(err instanceof Error ? err.message : String(err));
      }
      return;
    }
  }

  async function editConversationUserMessage(turn: ConversationTurn) {
    void turn;
  }

  async function deleteConversationTurn(turn: ConversationTurn) {
    void turn;
  }

  async function deleteConversationAssistantMessage(turn: ConversationTurn) {
    void turn;
  }

  async function retryConversationTurn(turn: ConversationTurn) {
    void turn;
  }

  return {
    deleteConversationAssistantMessage,
    deleteConversationTurn,
    editConversationUserMessage,
    retryConversationTurn,
    stopPendingTurn,
  };
}
