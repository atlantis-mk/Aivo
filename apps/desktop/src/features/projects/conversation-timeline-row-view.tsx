import { memo } from "react";

import {
  sameTimelineRow,
  timelineRowUsesAgentRuns,
  type ConversationTimelineRow,
} from "@/features/projects/conversation-timeline-row-model";
import { sameAgentRuns } from "@/features/projects/conversation-timeline-subagent-model";
import {
  TimelineToolCluster,
  TimelineToolGroup,
} from "@/features/projects/conversation-timeline-tool-components";
import type { ToolCallActivity } from "@/features/projects/conversation-timeline-tool-model";
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
  onOpenToolActivity,
  row,
  workspaceRoot,
}: {
  actions: ConversationTimelineActions;
  agentRuns: AgentRun[];
  onOpenSession?: (sessionId: string) => void;
  onOpenToolActivity?: (activity: ToolCallActivity) => void;
  row: ConversationTimelineRow;
  workspaceRoot: string;
}) {
  switch (row.type) {
    case "turn-gap":
      return <div aria-hidden="true" className="h-7" />;
    case "user-message":
      return <UserMessageRow actions={actions} turn={row.turn} />;
    case "assistant-preamble":
      return (
        <TimelineRowFrame role="assistant" turnId={row.turnId}>
          <AssistantPreamble text={row.text} workspaceRoot={workspaceRoot} />
        </TimelineRowFrame>
      );
    case "tool-group":
      return (
        <TimelineRowFrame role="assistant" turnId={row.turnId}>
          <TimelineToolGroup
            agentRuns={agentRuns}
            group={row.group}
            onOpenSession={onOpenSession}
            onOpenToolActivity={onOpenToolActivity}
          />
        </TimelineRowFrame>
      );
    case "tool-cluster":
      return (
        <TimelineRowFrame role="assistant" turnId={row.turnId}>
          <TimelineToolCluster
            activityId={row.key}
            groups={row.groups}
            onOpenToolActivity={onOpenToolActivity}
          />
        </TimelineRowFrame>
      );
    case "assistant-status":
      return (
        <TimelineRowFrame role="assistant" turnId={row.turn.id}>
          <AssistantStatus
            actionHeading={undefined}
            completed={Boolean(row.turn.responseCompletedAt)}
            isExecuting={row.isExecuting}
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
            workspaceRoot={workspaceRoot}
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
            isExecuting={row.isExecuting}
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
    onOpenToolActivity?: (activity: ToolCallActivity) => void;
    row: ConversationTimelineRow;
    workspaceRoot: string;
  },
  next: {
    actions: ConversationTimelineActions;
    agentRuns: AgentRun[];
    onOpenSession?: (sessionId: string) => void;
    onOpenToolActivity?: (activity: ToolCallActivity) => void;
    row: ConversationTimelineRow;
    workspaceRoot: string;
  },
) {
  return (
    sameTimelineRow(previous.row, next.row) &&
    previous.actions === next.actions &&
    previous.onOpenSession === next.onOpenSession &&
    previous.onOpenToolActivity === next.onOpenToolActivity &&
    previous.workspaceRoot === next.workspaceRoot &&
    (!timelineRowUsesAgentRuns(previous.row) ||
      sameAgentRuns(previous.agentRuns, next.agentRuns))
  );
}
