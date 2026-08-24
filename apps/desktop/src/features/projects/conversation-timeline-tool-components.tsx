import { memo } from "react";

import { Button } from "@/components/ui/button";
import { sameToolCalls } from "@/features/projects/conversation-timeline-model";
import {
  findAgentRunForToolCall,
  sameAgentRuns,
  uniqueDelegateToolCalls,
} from "@/features/projects/conversation-timeline-subagent-model";
import { ConversationToolCallBadge } from "@/features/projects/conversation-tool-inspector";
import { SubagentToolCard } from "./conversation-timeline-subagent-tool-card";
import { type AgentRun } from "@/services/aivo";
import type {
  ToolCallActivity,
  ToolCallGroup,
} from "@/features/projects/conversation-timeline-tool-model";

export const TimelineToolGroup = memo(function TimelineToolGroup({
  agentRuns,
  group,
  onOpenSession,
  onOpenToolActivity,
}: {
  agentRuns: AgentRun[];
  group: ToolCallGroup;
  onOpenSession?: (sessionId: string) => void;
  onOpenToolActivity?: (activity: ToolCallActivity) => void;
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
    <ToolActivityTrigger
      activity={{ id: `tool-group:${group.id}`, groups: [group] }}
      onOpen={onOpenToolActivity}
    />
  );
}, areTimelineToolGroupPropsEqual);

export const TimelineToolCluster = memo(function TimelineToolCluster({
  activityId,
  groups,
  onOpenToolActivity,
}: {
  activityId: string;
  groups: ToolCallGroup[];
  onOpenToolActivity?: (activity: ToolCallActivity) => void;
}) {
  return (
    <ToolActivityTrigger
      activity={{ id: activityId, groups }}
      onOpen={onOpenToolActivity}
    />
  );
}, areTimelineToolClusterPropsEqual);

function ToolActivityTrigger({
  activity,
  onOpen,
}: {
  activity: ToolCallActivity;
  onOpen?: (activity: ToolCallActivity) => void;
}) {
  const toolCallCount = activity.groups.reduce(
    (count, group) => count + group.calls.length,
    0,
  );

  return (
    <Button
      aria-label={`查看全部 ${toolCallCount} 个工具调用`}
      className="h-auto w-full flex-wrap justify-start gap-1.5 whitespace-normal px-1 py-1.5"
      data-assistant-hover-ignore="true"
      onClick={() => onOpen?.(activity)}
      type="button"
      variant="ghost"
    >
      {activity.groups.flatMap((group) =>
        group.calls.map((toolCall) => (
          <ConversationToolCallBadge
            group={group}
            key={toolCall.id}
            toolCall={toolCall}
          />
        )),
      )}
    </Button>
  );
}

function areTimelineToolClusterPropsEqual(
  previous: {
    activityId: string;
    groups: ToolCallGroup[];
    onOpenToolActivity?: (activity: ToolCallActivity) => void;
  },
  next: {
    activityId: string;
    groups: ToolCallGroup[];
    onOpenToolActivity?: (activity: ToolCallActivity) => void;
  },
) {
  return (
    previous.activityId === next.activityId &&
    previous.onOpenToolActivity === next.onOpenToolActivity &&
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
    onOpenToolActivity?: (activity: ToolCallActivity) => void;
  },
  next: {
    agentRuns: AgentRun[];
    group: ToolCallGroup;
    onOpenSession?: (sessionId: string) => void;
    onOpenToolActivity?: (activity: ToolCallActivity) => void;
  },
) {
  return (
    previous.group.description === next.group.description &&
    previous.group.id === next.group.id &&
    previous.group.kind === next.group.kind &&
    previous.group.title === next.group.title &&
    previous.onOpenSession === next.onOpenSession &&
    previous.onOpenToolActivity === next.onOpenToolActivity &&
    (previous.group.kind !== "delegate" ||
      sameAgentRuns(previous.agentRuns, next.agentRuns)) &&
    sameToolCalls(previous.group.calls, next.group.calls)
  );
}
