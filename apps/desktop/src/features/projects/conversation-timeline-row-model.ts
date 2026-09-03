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
    const isExecuting = hasVisibleToolCalls || turn.activityVisible;
    const hasPreambleText = preambleParts.some((part) => part.text.trim());

    if (turn.stopped) {
      pushTurnActivityRows(rows, turn, hasVisibleToolCalls, preambleParts, toolGroups);
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
        hasToolActivity: hasVisibleToolCalls,
        isExecuting,
        key: `assistant-status:${turn.id}`,
        turn,
        type: "assistant-status",
      });
      pushTurnActivityRows(rows, turn, hasVisibleToolCalls, preambleParts, toolGroups);
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

    pushTurnActivityRows(rows, turn, hasVisibleToolCalls, preambleParts, toolGroups);
    rows.push({
      actionHeading: hasVisibleToolCalls
        ? undefined
        : toolActionHeading(toolGroups),
      isExecuting,
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

function pushTurnActivityRows(
  rows: ConversationTimelineRow[],
  turn: ConversationTurn,
  hasVisibleToolCalls: boolean,
  preambleParts: ConversationAssistantTextPart[],
  toolGroups: ToolCallGroup[],
) {
  pushAssistantPreambles(
    rows,
    turn,
    preambleParts,
    hasVisibleToolCalls && Boolean(turn.responseCompletedAt),
  );
  if (hasVisibleToolCalls) pushToolActivityRows(rows, turn, toolGroups);
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

function pushAssistantPreambles(
  rows: ConversationTimelineRow[],
  turn: ConversationTurn,
  preambleParts: ConversationAssistantTextPart[],
  hideWhenToolsCollapsed = false,
) {
  for (const part of preambleParts) {
    rows.push({
      hideWhenToolsCollapsed,
      key: `assistant-preamble:${turn.id}:${part.id}`,
      text: part.text,
      turnId: turn.id,
      type: "assistant-preamble",
    });
  }
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

function pushToolActivityRows(
  rows: ConversationTimelineRow[],
  turn: ConversationTurn,
  toolGroups: ToolCallGroup[],
) {
  let clusteredGroups: ToolCallGroup[] = [];
  let lastDescription = "";
  const pushCluster = () => {
    const firstGroup = clusteredGroups[0];
    if (!firstGroup) return;
    rows.push({
      groups: clusteredGroups,
      isCompleted: Boolean(turn.responseCompletedAt),
      key: `tool-cluster:${turn.id}:${firstGroup.id}`,
      turnId: turn.id,
      type: "tool-cluster",
    });
    clusteredGroups = [];
  };

  for (const group of toolGroups) {
    const description = group.description?.trim() ?? "";
    if (description && description !== lastDescription) {
      pushCluster();
      rows.push({
        hideWhenToolsCollapsed: Boolean(turn.responseCompletedAt),
        key: `assistant-preamble:${turn.id}:${group.id}`,
        text: description,
        turnId: turn.id,
        type: "assistant-preamble",
      });
      lastDescription = description;
    }

    if (group.kind !== "delegate") {
      clusteredGroups.push(group);
      continue;
    }
    pushCluster();
    rows.push({
      group,
      isCompleted: Boolean(turn.responseCompletedAt),
      key: `tool-group:${turn.id}:${group.id}`,
      turnId: turn.id,
      type: "tool-group",
    });
  }
  pushCluster();
}
