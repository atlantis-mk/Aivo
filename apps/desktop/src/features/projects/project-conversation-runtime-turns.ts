import {
  getTurnElapsedSeconds,
  type ConversationTurn,
} from "@/features/projects/conversation-timeline-model";
import { parseTime } from "@/features/projects/project-time-model";
import type { domain } from "../../../bridge/go/models";

export function finalizeSupersededOpenTurn(
  turn: ConversationTurn,
  runtimeTurnById: Map<string, domain.Turn>,
) {
  const finalizedTurn = finalizeOpenTurnFromRuntime(turn, runtimeTurnById);
  if (finalizedTurn !== turn) return finalizedTurn;

  return {
    ...turn,
    stopped: true,
    thinkingSeconds: getTurnElapsedSeconds(turn),
  };
}

export function finalizeOpenTurnFromRuntime(
  turn: ConversationTurn,
  runtimeTurnById: Map<string, domain.Turn>,
) {
  const runtimeTurn = turn.turnId
    ? runtimeTurnById.get(turn.turnId)
    : undefined;
  if (!runtimeTurn || runtimeTurn.status === "running") return turn;

  const completedAt = parseTime(
    runtimeTurn.timeCompleted || runtimeTurn.timeUpdated,
  );
  const thinkingSeconds = Math.max(
    0,
    Math.floor((completedAt.getTime() - turn.submittedAt.getTime()) / 1000),
  );

  if (runtimeTurn.status === "cancelled") {
    return {
      ...turn,
      stopped: true,
      thinkingSeconds,
    };
  }

  if (runtimeTurn.status === "failed") {
    return {
      ...turn,
      responseCompletedAt: completedAt,
      responseText: runtimeTurn.error || "请求失败。",
      responseVisible: true,
      thinkingSeconds,
    };
  }

  if (runtimeTurn.status === "completed") {
    return {
      ...turn,
      responseCompletedAt: completedAt,
      responseVisible: true,
      thinkingSeconds,
    };
  }

  return turn;
}
