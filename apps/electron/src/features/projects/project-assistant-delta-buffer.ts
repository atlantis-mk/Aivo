import {
  useCallback,
  useRef,
  type Dispatch,
  type SetStateAction,
} from "react";

import {
  getTurnElapsedSeconds,
  type ConversationTurn,
} from "@/features/projects/conversation-timeline-model";
import type { PendingAssistantDelta } from "@/features/projects/project-workspace-state-model";

export function useProjectAssistantDeltaBuffer(
  setTurns: Dispatch<SetStateAction<ConversationTurn[]>>,
) {
  const pendingDeltaRef = useRef<PendingAssistantDelta | null>(null);
  const animationFrameRef = useRef(0);

  const flush = useCallback(() => {
    if (animationFrameRef.current) {
      window.cancelAnimationFrame(animationFrameRef.current);
      animationFrameRef.current = 0;
    }

    const pending = pendingDeltaRef.current;
    pendingDeltaRef.current = null;
    if (!pending?.text) return;

    setTurns((currentTurns) => {
      const index = currentTurns.findLastIndex(
        (turn) => !turn.stopped && !turn.responseCompletedAt,
      );
      if (index < 0) return currentTurns;
      return currentTurns.map((turn, turnIndex) => {
        if (turnIndex !== index) return turn;
        return {
          ...turn,
          responseText: turn.responseText + pending.text,
          responseVisible: true,
          thinkingSeconds: getTurnElapsedSeconds(turn),
          turnId: turn.turnId || pending.turnId,
        };
      });
    });
  }, [setTurns]);

  const cancel = useCallback(() => {
    if (animationFrameRef.current) {
      window.cancelAnimationFrame(animationFrameRef.current);
      animationFrameRef.current = 0;
    }
    pendingDeltaRef.current = null;
  }, []);

  const enqueue = useCallback(
    (payload: { delta: string; sessionId?: string; turnId?: string }) => {
      if (!payload.delta || !payload.sessionId) return;

      const current = pendingDeltaRef.current;
      if (
        current &&
        (current.sessionId !== payload.sessionId ||
          current.turnId !== payload.turnId)
      ) {
        flush();
      }

      pendingDeltaRef.current = {
        sessionId: payload.sessionId,
        text: `${pendingDeltaRef.current?.text ?? ""}${payload.delta}`,
        turnId: payload.turnId,
      };

      if (animationFrameRef.current) return;
      animationFrameRef.current = window.requestAnimationFrame(flush);
    },
    [flush],
  );

  return { cancel, enqueue, flush };
}
