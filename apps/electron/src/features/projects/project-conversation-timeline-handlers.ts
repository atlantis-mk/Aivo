import { useMemo, useRef } from "react";

import type { ConversationTimelineHandlerRefs } from "@/features/projects/project-workspace-state-model";

const noopConversationTimelineHandlers: ConversationTimelineHandlerRefs = {
  onDeleteAssistantMessage: () => undefined,
  onDeleteTurn: () => undefined,
  onEditUserMessage: () => undefined,
  onOpenSession: () => undefined,
  onRetryTurn: () => undefined,
};

export function useStableConversationTimelineHandlers(
  handlers: ConversationTimelineHandlerRefs,
) {
  const handlersRef = useRef<ConversationTimelineHandlerRefs>(
    noopConversationTimelineHandlers,
  );
  handlersRef.current = handlers;

  return useMemo<ConversationTimelineHandlerRefs>(
    () => ({
      onDeleteAssistantMessage: (turn) =>
        handlersRef.current.onDeleteAssistantMessage(turn),
      onDeleteTurn: (turn) => handlersRef.current.onDeleteTurn(turn),
      onEditUserMessage: (turn) =>
        handlersRef.current.onEditUserMessage(turn),
      onOpenSession: (sessionId) =>
        handlersRef.current.onOpenSession(sessionId),
      onRetryTurn: (turn) => handlersRef.current.onRetryTurn(turn),
    }),
    [],
  );
}
