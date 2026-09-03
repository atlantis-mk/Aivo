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
  onOpenSession,
}: {
  agentRuns: AgentRun[];
  group: ToolCallGroup;
  onOpenSession?: (sessionId: string) => void;
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

  return <CodexToolActivity groups={[group]} />;
}, areTimelineToolGroupPropsEqual);

export const TimelineToolCluster = memo(function TimelineToolCluster({
  groups,
}: {
  groups: ToolCallGroup[];
}) {
  return <CodexToolActivity groups={groups} />;
}, areTimelineToolClusterPropsEqual);

function areTimelineToolClusterPropsEqual(
  previous: {
    groups: ToolCallGroup[];
  },
  next: {
    groups: ToolCallGroup[];
  },
) {
  return (
    previous.groups.length === next.groups.length &&
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
    onOpenSession?: (sessionId: string) => void;
  },
  next: {
    agentRuns: AgentRun[];
    group: ToolCallGroup;
    onOpenSession?: (sessionId: string) => void;
  },
) {
  return (
    previous.group.description === next.group.description &&
    previous.group.id === next.group.id &&
    previous.group.kind === next.group.kind &&
    previous.group.title === next.group.title &&
    previous.onOpenSession === next.onOpenSession &&
    (previous.group.kind !== "delegate" ||
      sameAgentRuns(previous.agentRuns, next.agentRuns)) &&
    sameToolCalls(previous.group.calls, next.group.calls)
  );
}
