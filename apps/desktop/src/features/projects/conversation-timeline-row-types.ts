import type {
  ConversationSystemNote,
  ConversationTurn,
} from "@/features/projects/conversation-timeline-model";
import type { ToolCallGroup } from "@/features/projects/conversation-timeline-tool-model";

export type ConversationTimelineRow =
  | {
      type: "turn-gap";
      key: string;
      turnId: string;
    }
  | {
      type: "user-message";
      key: string;
      turn: ConversationTurn;
    }
  | {
      type: "assistant-preamble";
      key: string;
      text: string;
      turnId: string;
    }
  | {
      type: "tool-group";
      key: string;
      group: ToolCallGroup;
      turnId: string;
    }
  | {
      type: "assistant-status";
      key: string;
      turn: ConversationTurn;
    }
  | {
      type: "assistant-response";
      key: string;
      turn: ConversationTurn;
    }
  | {
      type: "system-note";
      key: string;
      note: ConversationSystemNote;
      turnId: string;
    }
  | {
      type: "thinking";
      actionHeading?: string;
      key: string;
      showSkeleton: boolean;
      turnId: string;
    }
  | {
      type: "stopped";
      key: string;
      stoppedSeconds: number;
      turnId: string;
    };
