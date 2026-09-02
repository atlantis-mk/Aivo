import {
  getTurnElapsedSeconds,
  sameToolCalls,
  type ConversationTurn,
} from "@/features/projects/conversation-timeline-model";
import { mergeToolCallLists } from "./project-conversation-tool-calls";
import { stripSessionAttachmentSummary } from "./project-conversation-text";

export function applyPendingTurnMetadata(
  turns: ConversationTurn[],
  options: { pendingTurnId?: string; pendingStartedAt?: number },
) {
  if (!options.pendingTurnId || !options.pendingStartedAt || turns.length === 0)
    return turns;
  const pendingTurnId = options.pendingTurnId;
  const pendingStartedAt = options.pendingStartedAt;
  const lastIndex = turns.length - 1;
  return turns.map((turn, index) => {
    if (index !== lastIndex || !turn.responseVisible) return turn;
    return {
      ...turn,
      id: turn.id || pendingTurnId,
      startedAt: pendingStartedAt,
      thinkingSeconds: getTurnElapsedSeconds({
        pausedMilliseconds: turn.pausedMilliseconds,
        pauseStartedAt: turn.pauseStartedAt,
        startedAt: pendingStartedAt,
      }),
    };
  });
}

export function mergeTurnPauseMetadata(
  nextTurns: ConversationTurn[],
  currentTurns: ConversationTurn[],
) {
  if (nextTurns.length === 0 || currentTurns.length === 0) return nextTurns;
  const currentByKey = new Map<string, ConversationTurn>();
  for (const turn of currentTurns) {
    for (const key of turnIdentityKeys(turn)) {
      currentByKey.set(key, turn);
    }
  }

  const now = Date.now();
  let changed = false;
  const mergedTurns = nextTurns.map((turn, index) => {
    const current =
      turnIdentityKeys(turn)
        .map((key) => currentByKey.get(key))
        .find(Boolean) ??
      fallbackCurrentTurnForMerge(turn, index, currentTurns);
    if (!current) return turn;

    let nextTurn = turn;

    if (current.toolCalls.length > 0) {
      const mergedToolCalls = mergeToolCallLists(
        current.toolCalls,
        turn.toolCalls,
      );
      if (!sameToolCalls(turn.toolCalls, mergedToolCalls)) {
        changed = true;
        nextTurn = {
          ...nextTurn,
          activityVisible: true,
          toolCalls: mergedToolCalls,
          turnId: nextTurn.turnId || current.turnId,
        };
      }
    }

    if (!nextTurn.preToolText.trim() && current.preToolText.trim()) {
      changed = true;
      nextTurn = {
        ...nextTurn,
        activityVisible: true,
        preToolText: current.preToolText,
      };
    }

    if (
      (nextTurn.attachments?.length ?? 0) === 0 &&
      (current.attachments?.length ?? 0) > 0
    ) {
      changed = true;
      nextTurn = {
        ...nextTurn,
        attachments: current.attachments,
        prompt: stripSessionAttachmentSummary(nextTurn.prompt),
      };
    }

    if (
      (nextTurn.assistantPreambles?.length ?? 0) === 0 &&
      (current.assistantPreambles?.length ?? 0) > 0
    ) {
      changed = true;
      nextTurn = {
        ...nextTurn,
        activityVisible: true,
        assistantPreambles: current.assistantPreambles,
      };
    }

    if (current.activityVisible && !nextTurn.activityVisible) {
      changed = true;
      nextTurn = {
        ...nextTurn,
        activityVisible: true,
      };
    }

    const pausedMilliseconds =
      (current.pausedMilliseconds ?? 0) +
      (current.pauseStartedAt ? Math.max(0, now - current.pauseStartedAt) : 0);
    if (pausedMilliseconds <= 0) return nextTurn;

    changed = true;
    return {
      ...nextTurn,
      pausedMilliseconds,
      pauseStartedAt: null,
      thinkingSeconds: Math.max(
        0,
        nextTurn.thinkingSeconds - Math.floor(pausedMilliseconds / 1000),
      ),
    };
  });

  return changed ? mergedTurns : nextTurns;
}

export function mergePreservedTurnAttachments(
  pendingTurnId: string | undefined,
  currentTurns: ConversationTurn[],
) {
  if (!pendingTurnId) return undefined;
  return currentTurns.find((turn) => turn.id === pendingTurnId)?.attachments;
}

function fallbackCurrentTurnForMerge(
  turn: ConversationTurn,
  index: number,
  currentTurns: ConversationTurn[],
) {
  const currentAtIndex = currentTurns[index];
  if (currentAtIndex && turnsCanShareLiveMetadata(turn, currentAtIndex)) {
    return currentAtIndex;
  }

  return currentTurns.find((currentTurn) =>
    turnsCanShareLiveMetadata(turn, currentTurn),
  );
}

function turnsCanShareLiveMetadata(
  nextTurn: ConversationTurn,
  currentTurn: ConversationTurn,
) {
  if (currentTurn.stopped || currentTurn.responseCompletedAt) return false;
  if (nextTurn.stopped || nextTurn.responseCompletedAt) return false;
  return true;
}

function turnIdentityKeys(turn: ConversationTurn) {
  return [
    turn.turnId && `turn:${turn.turnId}`,
    turn.userEventId && `user-event:${turn.userEventId}`,
    turn.id && `id:${turn.id}`,
  ].filter((key): key is string => Boolean(key));
}
