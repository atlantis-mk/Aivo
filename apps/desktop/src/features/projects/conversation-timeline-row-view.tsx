import { memo } from "react";

import {
  sameTimelineRow,
  timelineRowUsesAgentRuns,
  type ConversationTimelineRow,
} from "@/features/projects/conversation-timeline-row-model";
import { sameAgentRuns } from "@/features/projects/conversation-timeline-subagent-model";
import { TimelineToolGroup } from "@/features/projects/conversation-timeline-tool-components";
import {
  AssistantPreamble,
  AssistantResponse,
  AssistantStatus,
  StoppedResponse,
  ThinkingResponse,
} from "./conversation-timeline-assistant-message";
import { SystemNoteRow, TimelineRowFrame } from "./conversation-timeline-frame";
import { UserMessageRow } from "./conversation-timeline-user-message";
import type { ConversationTimelineActions } from "./conversation-timeline-types";
import type { AgentRun } from "@/services/aivo";

export const ConversationTimelineRowView = memo(function ConversationTimelineRowView({
  actions,
  agentRuns,
  onOpenSession,
  row,
}: {
  actions: ConversationTimelineActions;
  agentRuns: AgentRun[];
  onOpenSession?: (sessionId: string) => void;
  row: ConversationTimelineRow;
}) {
  switch (row.type) {
    case "turn-gap":
      return <div aria-hidden="true" className="h-7" />;
    case "user-message":
      return <UserMessageRow actions={actions} turn={row.turn} />;
    case "assistant-preamble":
      return (
        <TimelineRowFrame role="assistant" turnId={row.turnId}>
          <AssistantPreamble text={row.text} />
        </TimelineRowFrame>
      );
    case "tool-group":
      return (
        <TimelineRowFrame role="assistant" turnId={row.turnId}>
          <TimelineToolGroup
            agentRuns={agentRuns}
            group={row.group}
            onOpenSession={onOpenSession}
          />
        </TimelineRowFrame>
      );
    case "assistant-status":
      return (
        <TimelineRowFrame role="assistant" turnId={row.turn.id}>
          <AssistantStatus
            actionHeading={undefined}
            completed={Boolean(row.turn.responseCompletedAt)}
            responseSeconds={row.turn.thinkingSeconds}
          />
        </TimelineRowFrame>
      );
    case "assistant-response":
      return (
        <TimelineRowFrame role="assistant" turnId={row.turn.id}>
          <AssistantResponse
            actions={actions}
            completedAt={row.turn.responseCompletedAt}
            responseText={row.turn.responseText}
            turn={row.turn}
          />
        </TimelineRowFrame>
      );
    case "system-note":
      return (
        <TimelineRowFrame role="assistant" turnId={row.turnId}>
          <SystemNoteRow note={row.note} />
        </TimelineRowFrame>
      );
    case "thinking":
      return (
        <TimelineRowFrame role="assistant" turnId={row.turnId}>
          <ThinkingResponse
            actionHeading={row.actionHeading}
            showSkeleton={row.showSkeleton}
          />
        </TimelineRowFrame>
      );
    case "stopped":
      return (
        <TimelineRowFrame role="assistant" turnId={row.turnId}>
          <StoppedResponse stoppedSeconds={row.stoppedSeconds} />
        </TimelineRowFrame>
      );
  }
}, areTimelineRowPropsEqual);

function areTimelineRowPropsEqual(
  previous: {
    actions: ConversationTimelineActions;
    agentRuns: AgentRun[];
    onOpenSession?: (sessionId: string) => void;
    row: ConversationTimelineRow;
  },
  next: {
    actions: ConversationTimelineActions;
    agentRuns: AgentRun[];
    onOpenSession?: (sessionId: string) => void;
    row: ConversationTimelineRow;
  },
) {
  return (
    sameTimelineRow(previous.row, next.row) &&
    previous.actions === next.actions &&
    previous.onOpenSession === next.onOpenSession &&
    (!timelineRowUsesAgentRuns(previous.row) ||
      sameAgentRuns(previous.agentRuns, next.agentRuns))
  );
}
