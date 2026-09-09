import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";

export function getProjectConversationViewState({
  isOpeningConversationFromEmpty,
  turns,
}: {
  isOpeningConversationFromEmpty: boolean;
  turns: ConversationTurn[];
}) {
  const hasTurns = turns.length > 0;
  const lastTurn = turns.at(-1);
  return {
    hasPendingTurn: turns.some(
      (turn) => !turn.responseCompletedAt && !turn.stopped,
    ),
    hasPausedTurn: lastTurn?.stopped === true,
    hasTurns,
    lastTurnStateKey: lastTurn ? buildConversationTurnStateKey(turns) : "empty",
    showConversationLayout: hasTurns || isOpeningConversationFromEmpty,
  };
}

export function getProjectWorkspacePanelViewState({
  pendingQuestionRequests,
}: {
  pendingQuestionRequests: readonly unknown[];
}) {
  const hasPendingQuestionRequest = pendingQuestionRequests.length > 0;
  const hasPendingInteractionRequest = hasPendingQuestionRequest;

  return {
    hasPendingInteractionRequest,
    hasPendingQuestionRequest,
  };
}

function buildConversationTurnStateKey(turns: ConversationTurn[]) {
  const lastTurn = turns.at(-1);
  if (!lastTurn) return "empty";
  return [
    turns.length,
    lastTurn.responseVisible ? "visible" : "hidden",
    lastTurn.stopped ? "stopped" : "running",
    lastTurn.responseText.length,
    lastTurn.preToolText.length,
    lastTurn.assistantPreambles
      ?.map((part) => `${part.id}:${part.text.length}`)
      .join("|") ?? "",
    lastTurn.toolCalls.length,
    lastTurn.toolCalls
      .map((toolCall) =>
        [
          toolCall.id,
          toolCall.status,
          toolCall.timeUpdated ?? "",
          toolCall.resultSummary?.length ?? 0,
          toolCall.error?.length ?? 0,
        ].join(":"),
      )
      .join("|"),
  ].join("-");
}
