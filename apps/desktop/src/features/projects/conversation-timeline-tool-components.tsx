import { memo, useState } from "react";
import { ChevronDown } from "lucide-react";

import { cn } from "@/lib/utils";
import { sameToolCalls } from "@/features/projects/conversation-timeline-model";
import { AnimatedDisclosure } from "@/features/projects/conversation-timeline-disclosure";
import {
  findAgentRunForToolCall,
  sameAgentRuns,
  uniqueDelegateToolCalls,
} from "@/features/projects/conversation-timeline-subagent-model";
import { ToolCallCommandLine } from "./conversation-timeline-tool-command-line";
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
  const [open, setOpen] = useState(false);

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
    <div
      className="flex flex-col overflow-hidden rounded-md border-0 bg-transparent"
      data-assistant-hover-ignore="true"
    >
      <ToolCallGroupView
        group={group}
        onToggle={() => setOpen((current) => !current)}
        open={open}
      />
    </div>
  );
}, areTimelineToolGroupPropsEqual);

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
    previous.group.id === next.group.id &&
    previous.group.kind === next.group.kind &&
    previous.group.title === next.group.title &&
    previous.onOpenSession === next.onOpenSession &&
    (previous.group.kind !== "delegate" ||
      sameAgentRuns(previous.agentRuns, next.agentRuns)) &&
    sameToolCalls(previous.group.calls, next.group.calls)
  );
}

const ToolCallGroupView = memo(function ToolCallGroupView({
  group,
  onToggle,
  open,
}: {
  group: ToolCallGroup;
  onToggle: (groupId: string) => void;
  open: boolean;
}) {
  return (
    <div className="border-0 bg-transparent data-open:bg-transparent">
      <button
        aria-expanded={open}
        className="group/accordion-trigger relative flex flex-none items-center justify-between gap-0.5 border border-transparent px-0 py-1 text-left text-sm text-muted-foreground transition-all outline-none hover:no-underline disabled:pointer-events-none disabled:opacity-50"
        onClick={() => onToggle(group.id)}
        type="button"
      >
        <span>{group.title}</span>
        <ChevronDown
          className={cn(
            "size-5 shrink-0 text-muted-foreground transition-transform",
            open && "rotate-180",
          )}
        />
      </button>
      <AnimatedDisclosure open={open}>
        <div className="pt-0 pb-1.5 pl-3 [&_a]:underline [&_a]:underline-offset-3 [&_a]:hover:text-foreground [&_p:not(:last-child)]:mb-4">
          {group.calls.map((toolCall) => (
            <ToolCallCommandLine key={toolCall.id} toolCall={toolCall} />
          ))}
        </div>
      </AnimatedDisclosure>
    </div>
  );
}, areToolCallGroupPropsEqual);

function areToolCallGroupPropsEqual(
  previous: {
    group: ToolCallGroup;
    open: boolean;
    onToggle: (groupId: string) => void;
  },
  next: {
    group: ToolCallGroup;
    open: boolean;
    onToggle: (groupId: string) => void;
  },
) {
  return (
    previous.open === next.open &&
    previous.group.id === next.group.id &&
    previous.group.title === next.group.title &&
    sameToolCalls(previous.group.calls, next.group.calls)
  );
}
