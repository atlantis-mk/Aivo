import { sameToolCalls } from "@/features/projects/conversation-timeline-model";
import type { ConversationTimelineRow } from "@/features/projects/conversation-timeline-row-types";

export function sameTimelineRow(
  previous: ConversationTimelineRow,
  next: ConversationTimelineRow,
) {
  if (previous === next) return true;
  if (previous.type !== next.type || previous.key !== next.key) return false;

  switch (previous.type) {
    case "turn-gap":
      return previous.turnId === (next as typeof previous).turnId;
    case "user-message":
      return previous.turn === (next as typeof previous).turn;
    case "assistant-preamble":
      return (
        previous.turnId === (next as typeof previous).turnId &&
        previous.text === (next as typeof previous).text
      );
    case "tool-group": {
      const nextGroup = (next as typeof previous).group;
      return (
        previous.turnId === (next as typeof previous).turnId &&
        previous.group.description === nextGroup.description &&
        previous.group.id === nextGroup.id &&
        previous.group.kind === nextGroup.kind &&
        previous.group.title === nextGroup.title &&
        sameToolCalls(previous.group.calls, nextGroup.calls)
      );
    }
    case "tool-cluster": {
      const nextGroups = (next as typeof previous).groups;
      return (
        previous.turnId === (next as typeof previous).turnId &&
        previous.groups.length === nextGroups.length &&
        previous.groups.every((group, index) => {
          const nextGroup = nextGroups[index];
          return (
            nextGroup &&
            group.description === nextGroup.description &&
            group.id === nextGroup.id &&
            group.kind === nextGroup.kind &&
            group.title === nextGroup.title &&
            sameToolCalls(group.calls, nextGroup.calls)
          );
        })
      );
    }
    case "assistant-status":
      return (
        previous.turn === (next as typeof previous).turn &&
        previous.isExecuting === (next as typeof previous).isExecuting
      );
    case "assistant-response":
      return previous.turn === (next as typeof previous).turn;
    case "system-note":
      return (
        previous.turnId === (next as typeof previous).turnId &&
        previous.note === (next as typeof previous).note
      );
    case "thinking":
      return (
        previous.turnId === (next as typeof previous).turnId &&
        previous.actionHeading === (next as typeof previous).actionHeading &&
        previous.isExecuting === (next as typeof previous).isExecuting &&
        previous.showSkeleton === (next as typeof previous).showSkeleton
      );
    case "stopped":
      return (
        previous.turnId === (next as typeof previous).turnId &&
        previous.stoppedSeconds === (next as typeof previous).stoppedSeconds
      );
  }
}

export function timelineRowUsesAgentRuns(row: ConversationTimelineRow) {
  return row.type === "tool-group" && row.group.kind === "delegate";
}
