import { memo } from "react";

import { sameToolCalls } from "@/features/projects/conversation-timeline-model";
import {
  findAgentRunForToolCall,
  sameAgentRuns,
  uniqueDelegateToolCalls,
} from "@/features/projects/conversation-timeline-subagent-model";
import { CodexToolActivity } from "./conversation-timeline-codex-tool-activity";
import { SubagentToolCard } from "./conversation-timeline-subagent-tool-card";
import { type AgentRun } from "@/services/aivo";
import type { ToolCallGroup } from "@/features/projects/conversation-timeline-tool-model";

export const TimelineToolGroup = memo(function TimelineToolGroup({
  agentRuns,
  group,
  expanded,
  onOpenSession,
  onToggle,
}: {
  agentRuns: AgentRun[];
  group: ToolCallGroup;
  expanded: boolean;
  onOpenSession?: (sessionId: string) => void;
  onToggle: () => void;
}) {
  if (group.kind === "delegate") {
    const delegateCalls = uniqueDelegateToolCalls(group.calls, agentRuns);
    return (
      <div
        className="flex flex-col gap-2 py-1"
        data-assistant-hover-ignore="true"
      >
        {delegateCalls.map((toolCall) => (
          <SubagentToolCard
            agentRun={findAgentRunForToolCall(toolCall, agentRuns)}
            key={toolCall.id}
            onOpenSession={onOpenSession}
            toolCall={toolCall}
          />
        ))}
      </div>
    );
  }

  return (
    <CodexToolActivity
      expanded={expanded}
      groups={[group]}
      onToggle={onToggle}
    />
  );
}, areTimelineToolGroupPropsEqual);

export const TimelineToolCluster = memo(function TimelineToolCluster({
  groups,
  expanded,
  onToggle,
}: {
  groups: ToolCallGroup[];
  expanded: boolean;
  onToggle: () => void;
}) {
  return (
    <CodexToolActivity
      expanded={expanded}
      groups={groups}
      onToggle={onToggle}
    />
  );
}, areTimelineToolClusterPropsEqual);

function areTimelineToolClusterPropsEqual(
  previous: {
    groups: ToolCallGroup[];
    expanded: boolean;
    onToggle: () => void;
  },
  next: {
    groups: ToolCallGroup[];
    expanded: boolean;
    onToggle: () => void;
  },
) {
  return (
    previous.groups.length === next.groups.length &&
    previous.expanded === next.expanded &&
    previous.onToggle === next.onToggle &&
    previous.groups.every((group, index) => {
      const nextGroup = next.groups[index];
      return (
        nextGroup &&
        group.description === nextGroup.description &&
        group.id === nextGroup.id &&
        group.kind === nextGroup.kind &&
        sameToolCalls(group.calls, nextGroup.calls)
      );
    })
  );
}

function areTimelineToolGroupPropsEqual(
  previous: {
    agentRuns: AgentRun[];
    group: ToolCallGroup;
    expanded: boolean;
    onOpenSession?: (sessionId: string) => void;
    onToggle: () => void;
  },
  next: {
    agentRuns: AgentRun[];
    group: ToolCallGroup;
    expanded: boolean;
    onOpenSession?: (sessionId: string) => void;
    onToggle: () => void;
  },
) {
  return (
    previous.group.description === next.group.description &&
    previous.expanded === next.expanded &&
    previous.group.id === next.group.id &&
    previous.group.kind === next.group.kind &&
    previous.group.title === next.group.title &&
    previous.onOpenSession === next.onOpenSession &&
    previous.onToggle === next.onToggle &&
    (previous.group.kind !== "delegate" ||
      sameAgentRuns(previous.agentRuns, next.agentRuns)) &&
    sameToolCalls(previous.group.calls, next.group.calls)
  );
}
