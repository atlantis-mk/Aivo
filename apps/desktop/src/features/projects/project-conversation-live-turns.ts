import type { domain } from "../../../bridge/go/models";
import {
  sameToolCalls,
  type ConversationTurn,
} from "@/features/projects/conversation-timeline-model";
import {
  appendAssistantPreamblePart,
  appendAssistantText,
} from "./project-conversation-text";
import { finalizeOpenTurnFromRuntime } from "./project-conversation-runtime-turns";
import { mergeToolCallLists } from "./project-conversation-tool-calls";

export function mergeSingleToolCall(
  turns: ConversationTurn[],
  toolCall: domain.ToolCall,
) {
  if (!toolCall.id) return turns;
  let changed = false;
  const lastRunningTurnIndex = turns.findLastIndex(
    (turn) => !turn.stopped && !turn.responseCompletedAt,
  );
  const nextTurns = turns.map((turn, index) => {
    if (toolCall.turnId) {
      if (turn.turnId !== toolCall.turnId) return turn;
    } else if (index !== lastRunningTurnIndex) {
      return turn;
    }
    const nextToolCalls = mergeToolCallLists(turn.toolCalls, [toolCall]);
    const nextActivityVisible =
      turn.activityVisible || nextToolCalls.length > 0;
    if (
      sameToolCalls(turn.toolCalls, nextToolCalls) &&
      turn.activityVisible === nextActivityVisible
    ) {
      return turn;
    }
    changed = true;
    return {
      ...turn,
      activityVisible: nextActivityVisible,
      toolCalls: nextToolCalls,
      turnId: turn.turnId || toolCall.turnId,
    };
  });
  return changed ? nextTurns : turns;
}

export function moveOpenResponseTextToAssistantPreambleBeforeTool(
  turns: ConversationTurn[],
  toolCall: domain.ToolCall,
) {
  let changed = false;
  const lastRunningTurnIndex = turns.findLastIndex(
    (turn) => !turn.stopped && !turn.responseCompletedAt,
  );
  const nextTurns = turns.map((turn, index) => {
    if (turn.stopped || turn.responseCompletedAt || !turn.responseText.trim()) {
      return turn;
    }
    if (toolCall.turnId) {
      if (turn.turnId !== toolCall.turnId) return turn;
    } else if (index !== lastRunningTurnIndex) {
      return turn;
    }
    changed = true;
    return {
      ...turn,
      activityVisible: true,
      assistantPreambles: appendAssistantPreamblePart(
        turn.assistantPreambles,
        {
          id: `live-preamble:${toolCall.id}`,
          text: turn.responseText,
          timeCreated: toolCall.timeCreated || new Date().toISOString(),
        },
      ),
      preToolText: appendAssistantText(turn.preToolText, turn.responseText),
      responseText: "",
      responseVisible: false,
      turnId: turn.turnId || toolCall.turnId,
    };
  });
  return changed ? nextTurns : turns;
}

export function mergeRuntimeTurn(
  turns: ConversationTurn[],
  runtimeTurn: domain.Turn,
) {
  if (!runtimeTurn.id) return turns;

  const exactIndex = turns.findIndex(
    (turn) =>
      turn.turnId === runtimeTurn.id ||
      (Boolean(runtimeTurn.userEventId) &&
        turn.userEventId === runtimeTurn.userEventId),
  );
  const runtimeStartedAt = Date.parse(runtimeTurn.timeCreated || "");
  const targetIndex =
    exactIndex >= 0
      ? exactIndex
      : turns.findLastIndex(
          (turn) =>
            !turn.turnId &&
            !turn.stopped &&
            !turn.responseCompletedAt &&
            (!Number.isFinite(runtimeStartedAt) ||
              turn.submittedAt.getTime() <= runtimeStartedAt + 1_000),
        );
  if (targetIndex < 0) return turns;

  let changed = false;
  const nextTurns = turns.map((turn, index) => {
    if (index !== targetIndex) return turn;
    const finalized = finalizeOpenTurnFromRuntime(
      {
        ...turn,
        turnId: runtimeTurn.id,
      },
      new Map([[runtimeTurn.id, runtimeTurn]]),
    );
    if (finalized === turn) return turn;
    changed = true;
    return finalized;
  });
  return changed ? nextTurns : turns;
}
