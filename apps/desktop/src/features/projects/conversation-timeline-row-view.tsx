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
import {
  AssistantPreamble,
  AssistantResponse,
  AssistantStatus,
  StoppedResponse,
  ThinkingResponse,
} from "./conversation-timeline-assistant-message";
import { AnimatedDisclosure } from "./conversation-timeline-disclosure";
import { SystemNoteRow, TimelineRowFrame } from "./conversation-timeline-frame";
import { UserMessageRow } from "./conversation-timeline-user-message";
import { useToolTurnExpansion } from "./conversation-timeline-tool-turn-expansion";
import type { ConversationTimelineActions } from "./conversation-timeline-types";
import type { AgentRun } from "@/services/aivo";

export const ConversationTimelineRowView = memo(
  function ConversationTimelineRowView({
    actions,
    agentRuns,
    onOpenSession,
    row,
    workspaceRoot,
  }: {
    actions: ConversationTimelineActions;
    agentRuns: AgentRun[];
    onOpenSession?: (sessionId: string) => void;
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
          <TimelineAssistantPreambleRow
            hideWhenToolsCollapsed={row.hideWhenToolsCollapsed}
            text={row.text}
            turnId={row.turnId}
            workspaceRoot={workspaceRoot}
          />
        );
      case "tool-group":
        return (
          <TimelineToolGroupRow
            agentRuns={agentRuns}
            group={row.group}
            isCompleted={row.isCompleted}
            onOpenSession={onOpenSession}
            turnId={row.turnId}
          />
        );
      case "tool-cluster":
        return (
          <TimelineToolClusterRow
            groups={row.groups}
            isCompleted={row.isCompleted}
            turnId={row.turnId}
          />
        );
      case "assistant-status":
        return (
          <TimelineRowFrame role="assistant" turnId={row.turn.id}>
            <AssistantStatus
              actionHeading={undefined}
              completed={Boolean(row.turn.responseCompletedAt)}
              hasToolActivity={row.hasToolActivity}
              isExecuting={row.isExecuting}
              model={row.turn.model}
              modelProvider={row.turn.modelProvider}
              responseSeconds={row.turn.thinkingSeconds}
              turnId={row.turn.id}
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
              responseSeconds={row.responseSeconds}
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
  },
  areTimelineRowPropsEqual,
);

function TimelineAssistantPreambleRow({
  hideWhenToolsCollapsed = false,
  text,
  turnId,
  workspaceRoot,
}: {
  hideWhenToolsCollapsed?: boolean;
  text: string;
  turnId: string;
  workspaceRoot: string;
}) {
  const { expanded, toggle } = useToolTurnExpansion(turnId);
  if (hideWhenToolsCollapsed) {
    return (
      <AnimatedDisclosure open={expanded}>
        <TimelineRowFrame role="assistant" turnId={turnId}>
          <AssistantPreamble text={text} workspaceRoot={workspaceRoot} />
        </TimelineRowFrame>
      </AnimatedDisclosure>
    );
  }
  return (
    <TimelineRowFrame role="assistant" turnId={turnId}>
      <AssistantPreamble text={text} workspaceRoot={workspaceRoot} />
    </TimelineRowFrame>
  );
}

function TimelineToolGroupRow({
  agentRuns,
  group,
  isCompleted,
  onOpenSession,
  turnId,
}: {
  agentRuns: AgentRun[];
  group: Extract<ConversationTimelineRow, { type: "tool-group" }>["group"];
  isCompleted: boolean;
  onOpenSession?: (sessionId: string) => void;
  turnId: string;
}) {
  const { expanded, toggle } = useToolTurnExpansion(turnId);
  const open = !isCompleted || expanded;
  return (
    <AnimatedDisclosure open={open}>
      <TimelineRowFrame role="assistant" turnId={turnId}>
        <TimelineToolGroup
          agentRuns={agentRuns}
          expanded={open}
          group={group}
          onOpenSession={onOpenSession}
          onToggle={toggle}
        />
      </TimelineRowFrame>
    </AnimatedDisclosure>
  );
}

function TimelineToolClusterRow({
  groups,
  isCompleted,
  turnId,
}: {
  groups: Extract<ConversationTimelineRow, { type: "tool-cluster" }>["groups"];
  isCompleted: boolean;
  turnId: string;
}) {
  const { expanded, toggle } = useToolTurnExpansion(turnId);
  const open = !isCompleted || expanded;
  return (
    <AnimatedDisclosure open={open}>
      <TimelineRowFrame role="assistant" turnId={turnId}>
        <TimelineToolCluster
          expanded={open}
          groups={groups}
          onToggle={toggle}
        />
      </TimelineRowFrame>
    </AnimatedDisclosure>
  );
}

function areTimelineRowPropsEqual(
  previous: {
    actions: ConversationTimelineActions;
    agentRuns: AgentRun[];
    onOpenSession?: (sessionId: string) => void;
    row: ConversationTimelineRow;
    workspaceRoot: string;
  },
  next: {
    actions: ConversationTimelineActions;
    agentRuns: AgentRun[];
    onOpenSession?: (sessionId: string) => void;
    row: ConversationTimelineRow;
    workspaceRoot: string;
  },
) {
  return (
    sameTimelineRow(previous.row, next.row) &&
    previous.actions === next.actions &&
    previous.onOpenSession === next.onOpenSession &&
    previous.workspaceRoot === next.workspaceRoot &&
    (!timelineRowUsesAgentRuns(previous.row) ||
      sameAgentRuns(previous.agentRuns, next.agentRuns))
  );
}
