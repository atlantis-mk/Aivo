import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";

export const CONVERSATION_OPEN_ANIMATION_MS = 520;
export const OPEN_CONVERSATION_FROM_EMPTY_DELAY =
  CONVERSATION_OPEN_ANIMATION_MS + 60;
export const EMPTY_COMPOSER_VERTICAL_OFFSET = 8;
export const MARKDOWN_CONTENT_RESIZE_EVENT = "aivo-markdown-content-resize";
export const SCROLL_BOTTOM_SENTINEL = 9_999_999;
export const FORCE_BOTTOM_FRAME_COUNT = 18;
export const SCROLL_BOTTOM_ANIMATION_MS = 220;
export const SHOW_SCROLL_TO_BOTTOM_DISTANCE = 96;
export const PROJECT_PANEL_TRANSITION_MS = 200;
export const SHOULD_MOUNT_TOOL_ACTIVITY_SIDEBAR = true;
export const SHOULD_AUTO_OPEN_TOOL_ACTIVITY_SIDEBAR = true;

export type PendingAssistantDelta = {
  sessionId: string;
  text: string;
  turnId?: string;
};

export type ConversationTimelineHandlerRefs = {
  onDeleteAssistantMessage: (turn: ConversationTurn) => void;
  onDeleteTurn: (turn: ConversationTurn) => void;
  onEditUserMessage: (turn: ConversationTurn) => void;
  onOpenSession: (sessionId: string) => void;
  onRetryTurn: (turn: ConversationTurn) => void;
};
