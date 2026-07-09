import {
  type ConversationAssistantTextPart,
  type ConversationTurn,
} from "@/features/projects/conversation-timeline-model";
import type { ConversationTimelineRow } from "@/features/projects/conversation-timeline-row-types";
import {
  filterVisibleToolCalls,
  groupToolCalls,
  toolActionHeading,
  type ToolCallGroup,
} from "@/features/projects/conversation-timeline-tool-model";

export type { ConversationTimelineRow } from "@/features/projects/conversation-timeline-row-types";
export {
  sameTimelineRow,
  timelineRowUsesAgentRuns,
} from "@/features/projects/conversation-timeline-row-equality";

export function constructConversationTimelineRows(
  turns: ConversationTurn[],
): ConversationTimelineRow[] {
  return turns.flatMap((turn, index) => {
    const rows: ConversationTimelineRow[] = [];
    if (index > 0) {
      rows.push({
        key: `turn-gap:${turn.id}`,
        turnId: turn.id,
        type: "turn-gap",
      });
    }

    rows.push({
      key: `user-message:${turn.id}`,
      turn,
      type: "user-message",
    });

    const preambleParts = assistantPreambleParts(turn);
    const toolGroups = groupToolCalls(
      filterVisibleToolCalls(turn.toolCalls),
      preambleParts,
    );
    const hasVisibleToolCalls = toolGroups.length > 0;
    const hasPreambleText = preambleParts.some((part) => part.text.trim());
    pushAssistantActivityRows(rows, turn, preambleParts, toolGroups);

    if (turn.stopped) {
      rows.push({
        key: `stopped:${turn.id}`,
        stoppedSeconds: turn.thinkingSeconds,
        turnId: turn.id,
        type: "stopped",
      });
      pushSystemNotes(rows, turn);
      return rows;
    }

    if (turn.responseVisible || turn.responseText.trim()) {
      rows.push({
        key: `assistant-status:${turn.id}`,
        turn,
        type: "assistant-status",
      });
      if (turn.responseText.trim()) {
        rows.push({
          key: `assistant-response:${turn.id}`,
          turn,
          type: "assistant-response",
        });
      }
      pushSystemNotes(rows, turn);
      return rows;
    }

    rows.push({
      actionHeading: hasVisibleToolCalls
        ? undefined
        : toolActionHeading(toolGroups),
      key: `thinking:${turn.id}`,
      showSkeleton:
        !turn.activityVisible && !hasPreambleText && !hasVisibleToolCalls,
      turnId: turn.id,
      type: "thinking",
    });
    pushSystemNotes(rows, turn);
    return rows;
  });
}

function assistantPreambleParts(
  turn: ConversationTurn,
): ConversationAssistantTextPart[] {
  const parts = (turn.assistantPreambles ?? []).filter((part) =>
    part.text.trim(),
  );
  if (parts.length > 0) return parts;
  if (!turn.preToolText.trim()) return [];
  return [
    {
      id: `legacy-preamble:${turn.id}`,
      text: turn.preToolText,
      timeCreated: undefined,
    },
  ];
}

function pushAssistantActivityRows(
  rows: ConversationTimelineRow[],
  turn: ConversationTurn,
  preambleParts: ConversationAssistantTextPart[],
  toolGroups: ToolCallGroup[],
) {
  if (preambleParts.length === 0) {
    pushToolGroups(rows, turn, toolGroups);
    return;
  }

  const activityItems = [
    ...preambleParts.map((part, index) => ({
      index,
      key: `assistant-preamble:${turn.id}:${part.id}`,
      kind: "text" as const,
      part,
      sortTime: timelineSortTime(part.timeCreated, index),
    })),
    ...toolGroups.map((group, index) => ({
      index,
      group,
      key: `tool-group:${turn.id}:${group.id}`,
      kind: "tool" as const,
      sortTime: timelineSortTime(group.timeCreated, preambleParts.length + index),
    })),
  ].toSorted((a, b) => {
    const timeDelta = a.sortTime - b.sortTime;
    if (timeDelta !== 0) return timeDelta;
    if (a.kind !== b.kind) return a.kind === "text" ? -1 : 1;
    return a.index - b.index;
  });

  for (const item of activityItems) {
    if (item.kind === "text") {
      rows.push({
        key: item.key,
        text: item.part.text,
        turnId: turn.id,
        type: "assistant-preamble",
      });
      continue;
    }
    rows.push({
      group: item.group,
      key: item.key,
      turnId: turn.id,
      type: "tool-group",
    });
  }
}

function timelineSortTime(value: string | undefined, fallbackOffset: number) {
  if (!value) return Number.MIN_SAFE_INTEGER + fallbackOffset;
  const time = Date.parse(value);
  if (Number.isNaN(time)) return Number.MIN_SAFE_INTEGER + fallbackOffset;
  return time;
}

function pushSystemNotes(rows: ConversationTimelineRow[], turn: ConversationTurn) {
  for (const note of turn.systemNotes ?? []) {
    if (!note.content.trim()) continue;
    rows.push({
      key: `system-note:${turn.id}:${note.id}`,
      note,
      turnId: turn.turnId ?? turn.id,
      type: "system-note",
    });
  }
}

function pushToolGroups(
  rows: ConversationTimelineRow[],
  turn: ConversationTurn,
  toolGroups: ToolCallGroup[],
) {
  for (const group of toolGroups) {
    rows.push({
      group,
      key: `tool-group:${turn.id}:${group.id}`,
      turnId: turn.id,
      type: "tool-group",
    });
  }
}
