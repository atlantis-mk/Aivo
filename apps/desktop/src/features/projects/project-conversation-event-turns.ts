import type { domain } from "../../../bridge/go/models";
import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";
import { parseTime } from "@/features/projects/project-time-model";
import { conversationAttachmentsFromEvent } from "./project-conversation-attachments";
import {
  appendAssistantPreamblePart,
  appendAssistantText,
  stripSessionAttachmentSummary,
} from "./project-conversation-text";
import {
  finalizeOpenTurnFromRuntime,
  finalizeSupersededOpenTurn,
} from "./project-conversation-runtime-turns";
import { attachSystemNotesToTurns } from "./project-conversation-system-notes";
import { groupToolCallsByTurnId } from "./project-conversation-tool-calls";
import { runtimeMetricsFromEventPayload } from "./project-session-runtime-stats";

export function hasRunningTurn(turns: ConversationTurn[]) {
  return turns.some((turn) => !turn.responseCompletedAt && !turn.stopped);
}

export function turnsFromEvents(
  events: domain.SessionEvent[],
  toolCalls: domain.ToolCall[] = [],
  runtimeTurns: domain.Turn[] = [],
): ConversationTurn[] {
  const messageEvents = events.filter(isConversationMessageEvent);
  const runtimeTurnByUserEventId = new Map(
    runtimeTurns
      .filter((turn) => turn.userEventId)
      .map((turn) => [turn.userEventId, turn]),
  );
  const runtimeTurnById = new Map(runtimeTurns.map((turn) => [turn.id, turn]));
  const turns: ConversationTurn[] = [];
  let current: ConversationTurn | null = null;
  const toolCallsByTurnId = groupToolCallsByTurnId(toolCalls);
  const getTurnToolCalls = (turnId?: string) =>
    turnId ? toolCallsByTurnId.get(turnId) ?? [] : [];

  for (const event of messageEvents) {
    if (event.role === "user" || event.type === "user_message") {
      if (current) {
        turns.push(finalizeSupersededOpenTurn(current, runtimeTurnById));
      }
      const submittedAt = parseTime(event.timeCreated);
      const runtimeTurn = runtimeTurnByUserEventId.get(event.id);
      const currentToolCalls = getTurnToolCalls(runtimeTurn?.id);
      const attachments = conversationAttachmentsFromEvent(event);
      current = {
        activityVisible: currentToolCalls.length > 0,
        assistantPreambles: [],
        attachments,
        id: event.id,
        prompt:
          attachments.length > 0
            ? stripSessionAttachmentSummary(event.content ?? "")
            : event.content ?? "",
        preToolText: "",
        responseCompletedAt: null,
        responseText: "",
        responseVisible: false,
        startedAt: submittedAt.getTime(),
        stopped: false,
        submittedAt,
        thinkingSeconds: 0,
        toolCalls: currentToolCalls,
        turnId: runtimeTurn?.id,
        userEventId: event.id,
      };
      continue;
    }

    if (!current) continue;
    if (event.type === "error") {
      const completedAt = parseTime(event.timeCreated);
      const runtimeTurn = event.turnId
        ? runtimeTurnById.get(event.turnId)
        : undefined;
      if (runtimeTurn?.status === "cancelled") {
        current = {
          ...current,
          stopped: true,
          thinkingSeconds: Math.max(
            0,
            Math.floor(
              (completedAt.getTime() - current.submittedAt.getTime()) / 1000,
            ),
          ),
          turnId: event.turnId,
        };
        turns.push(current);
        current = null;
        continue;
      }
      current = {
        ...current,
        activityVisible:
          current.activityVisible ||
          getTurnToolCalls(event.turnId).length > 0,
        responseCompletedAt: completedAt,
        responseText: event.content ?? "请求失败。",
        responseVisible: true,
        thinkingSeconds: Math.max(
          0,
          Math.floor(
            (completedAt.getTime() - current.submittedAt.getTime()) / 1000,
          ),
        ),
        toolCalls: getTurnToolCalls(event.turnId),
        turnId: event.turnId,
      };
      turns.push(current);
      current = null;
      continue;
    }
    if (isBeforeToolAssistantEvent(event)) {
      current = {
        ...current,
        activityVisible: true,
        assistantPreambles: appendAssistantPreamblePart(
          current.assistantPreambles,
          {
            id: event.id,
            text: event.content ?? "",
            timeCreated: event.timeCreated,
          },
        ),
        preToolText: appendAssistantText(
          current.preToolText,
          event.content ?? "",
        ),
        toolCalls: getTurnToolCalls(event.turnId),
        turnId: event.turnId,
      };
      continue;
    }
    const completedAt = parseTime(event.timeCreated);
    current = {
      ...current,
      activityVisible:
        current.activityVisible || getTurnToolCalls(event.turnId).length > 0,
      responseCompletedAt: completedAt,
      responseText: appendAssistantText(
        current.responseText,
        event.content ?? "",
      ),
      responseVisible: true,
      runtimeMetrics: runtimeMetricsFromEventPayload(event.payload),
      thinkingSeconds: Math.max(
        0,
        Math.floor(
          (completedAt.getTime() - current.submittedAt.getTime()) / 1000,
        ),
      ),
      toolCalls: getTurnToolCalls(event.turnId),
      turnId: event.turnId,
      assistantEventId: event.id,
    };
    turns.push(current);
    current = null;
  }

  if (current) turns.push(finalizeOpenTurnFromRuntime(current, runtimeTurnById));
  return attachSystemNotesToTurns(turns, events);
}

function isBeforeToolAssistantEvent(event: domain.SessionEvent) {
  return (
    event.type === "assistant_message" &&
    event.payload?.["phase"] === "before_tool"
  );
}

function isConversationMessageEvent(event: domain.SessionEvent) {
  if (event.visibility && event.visibility !== "normal") return false;
  return (
    event.type === "user_message" ||
    event.type === "assistant_message" ||
    event.type === "error"
  );
}
