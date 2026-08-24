import type { domain } from "../../../bridge/go/models";
import type { TurnRuntimeMetrics } from "@/features/projects/project-session-runtime-stats";

export type ConversationTurn = {
  activityVisible: boolean;
  assistantPreambles?: ConversationAssistantTextPart[];
  attachments?: ConversationUserAttachment[];
  id: string;
  prompt: string;
  preToolText: string;
  responseText: string;
  toolCalls: domain.ToolCall[];
  turnId?: string;
  userEventId?: string;
  assistantEventId?: string;
  startedAt: number;
  submittedAt: Date;
  thinkingSeconds: number;
  pausedMilliseconds?: number;
  pauseStartedAt?: number | null;
  responseCompletedAt: Date | null;
  responseVisible: boolean;
  runtimeMetrics?: TurnRuntimeMetrics;
  stopped: boolean;
  systemNotes?: ConversationSystemNote[];
};

export type ConversationUserAttachment = {
  id: string;
  name: string;
  mimeType: string;
  kind: "image" | "file";
  previewUrl?: string;
  size?: number;
};

export type ConversationAssistantTextPart = {
  id: string;
  text: string;
  timeCreated?: string;
};

export type ConversationSystemNote = {
  id: string;
  content: string;
  payload?: Record<string, unknown>;
  timeCreated?: string;
};

export function getTurnElapsedSeconds(
  turn: Pick<
    ConversationTurn,
    "pausedMilliseconds" | "pauseStartedAt" | "startedAt"
  >,
  now = Date.now(),
) {
  const pausedMilliseconds =
    (turn.pausedMilliseconds ?? 0) +
    (turn.pauseStartedAt ? Math.max(0, now - turn.pauseStartedAt) : 0);
  return Math.max(
    0,
    Math.floor((now - turn.startedAt - pausedMilliseconds) / 1000),
  );
}

export function sameToolCalls(a: domain.ToolCall[], b: domain.ToolCall[]) {
  if (a === b) return true;
  if (a.length !== b.length) return false;
  return a.every((item, index) => {
    const other = b[index];
    return (
      other &&
      item.id === other.id &&
      item.status === other.status &&
      item.resultSummary === other.resultSummary &&
      (item.result === other.result ||
        stableToolCallResultKey(item.result) ===
          stableToolCallResultKey(other.result)) &&
      item.error === other.error &&
      item.timeUpdated === other.timeUpdated
    );
  });
}

function stableToolCallResultKey(value: unknown): string {
  if (!value || typeof value !== "object") return "";
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}
