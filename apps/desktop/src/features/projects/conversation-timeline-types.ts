import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";

export type ConversationTimelineActions = {
  onDeleteAssistantMessage?: (turn: ConversationTurn) => void;
  onDeleteTurn?: (turn: ConversationTurn) => void;
  onEditUserMessage?: (turn: ConversationTurn) => void;
  onRetryTurn?: (turn: ConversationTurn) => void;
};
