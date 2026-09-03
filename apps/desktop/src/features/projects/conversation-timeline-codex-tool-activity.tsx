import {
  ChevronRight,
  CircleAlert,
  Clock3,
  Globe2,
  Loader2,
  SquareTerminal,
  Wrench,
} from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";

import { cn } from "@/lib/utils";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  getRetainedOutputRefs,
  getToolCallCommand,
  getToolCallFileChanges,
  getToolResultText,
  isCommandToolCall,
} from "@/features/projects/conversation-timeline-tool-model";
import { AnimatedDisclosure } from "./conversation-timeline-disclosure";
import { ToolCallCommandLine } from "./conversation-timeline-tool-command-line";
import type { ToolCallGroup } from "./conversation-timeline-tool-types";
import type { domain } from "../../../bridge/go/models";

export function CodexToolActivity({ groups }: { groups: ToolCallGroup[] }) {
  const calls = useMemo(() => groups.flatMap((group) => group.calls), [groups]);
  const isRunning = calls.some((call) => call.status === "running");
  const needsApproval = calls.some(
    (call) => call.status === "pending_approval",
  );
  const hasFailed = calls.some((call) => call.status === "failed");
  const [expanded, setExpanded] = useState(
    () => isRunning || needsApproval || hasFailed,
  );
  const toggledByUser = useRef(false);

  useEffect(() => {
    if (!toggledByUser.current && (isRunning || needsApproval || hasFailed)) {
      setExpanded(true);
    }
  }, [hasFailed, isRunning, needsApproval]);

  if (calls.length === 0) return null;

  const status = hasFailed
    ? "failed"
    : needsApproval
      ? "pending_approval"
      : isRunning
        ? "running"
        : "success";
  const summary = activitySummary({
    groups,
    hasFailed,
    isRunning,
    needsApproval,
  });

  if (
    calls.length === 1 &&
    (isCommandToolCall(calls[0]) || calls[0].name === "web_search")
  ) {
    return <CodexStandaloneToolActivity call={calls[0]} status={status} />;
  }

  return (
    <section
      className="aivo-codex-tool-activity min-w-0 py-1"
      data-assistant-hover-ignore="true"
      data-state={expanded ? "open" : "closed"}
      data-status={status}
    >
      <button
        aria-expanded={expanded}
        className="aivo-tool-activity-header group/activity-header flex min-w-0 max-w-full cursor-pointer items-center gap-1.5 rounded-md py-1 text-left text-sm text-muted-foreground outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
        onClick={() => {
          toggledByUser.current = true;
          setExpanded((current) => !current);
        }}
        type="button"
      >
        <ToolActivityIcon status={status} />
        <span className="min-w-0 flex-1 truncate">{summary}</span>
        {isRunning ? (
          <span className="shrink-0 text-xs text-muted-foreground">运行中</span>
        ) : null}
        {needsApproval ? (
          <span className="shrink-0 text-xs text-amber-600 dark:text-amber-300">
            等待批准
          </span>
        ) : null}
        <ChevronRight
          aria-hidden="true"
          className={cn(
            "size-3 shrink-0 text-muted-foreground/80 transition-transform duration-150",
            expanded && "rotate-90",
          )}
        />
      </button>

      <div
        className={cn(
          "grid transition-[grid-template-rows,opacity] duration-200 ease-out",
          expanded
            ? "grid-rows-[1fr] opacity-100"
            : "grid-rows-[0fr] opacity-0",
        )}
      >
        <div className="min-h-0 overflow-hidden">
          <ScrollArea className="aivo-tool-activity-items mt-1 max-h-64 w-full min-w-0 max-w-full overflow-hidden [&>[data-slot=scroll-area-viewport]]:h-auto [&>[data-slot=scroll-area-viewport]]:max-h-64 [&>[data-slot=scroll-area-viewport]]:overflow-x-hidden [&>[data-slot=scroll-area-viewport]>div]:!block [&>[data-slot=scroll-area-viewport]>div]:!w-full [&>[data-slot=scroll-area-viewport]>div]:!min-w-0">
            <div className="aivo-tool-activity-items-list flex w-full min-w-0 flex-col">
              {calls.map((call) =>
                isCommandToolCall(call) ? (
                <CodexStandaloneToolActivity
                    call={call}
                    key={call.id}
                    status={call.status === "failed" ? "failed" : call.status}
                  />
                ) : (
                  <CodexToolCallDetail
                    call={call}
                    collapseDetails
                    key={call.id}
                  />
                ),
              )}
            </div>
          </ScrollArea>
        </div>
      </div>
    </section>
  );
}

function CodexStandaloneToolActivity({
  call,
  status,
}: {
  call: domain.ToolCall;
  status: string;
}) {
  return (
    <section
      className="aivo-codex-tool-activity aivo-codex-tool-activity--single-command min-w-0"
      data-assistant-hover-ignore="true"
      data-state="open"
      data-status={status}
    >
      <CodexToolCallDetail call={call} collapseDetails />
    </section>
  );
}

function CodexToolCallDetail({
  call,
  collapseDetails,
}: {
  call: domain.ToolCall;
  collapseDetails: boolean;
}) {
  const command = getToolCallCommand(call);
  const status = call.status === "failed" ? "failed" : call.status;
  const hasDetails = toolCallHasDetails(call);
  const [detailsOpen, setDetailsOpen] = useState(
    () =>
      !collapseDetails ||
      status === "running" ||
      status === "pending_approval",
  );

  useEffect(() => {
    if (status === "running" || status === "pending_approval") {
      setDetailsOpen(true);
    }
  }, [status]);

  return (
    <div className="aivo-tool-call-item min-w-0">
      {hasDetails ? (
        <button
          aria-expanded={detailsOpen}
          className="aivo-tool-call-row flex w-full min-w-0 items-center gap-2 text-left text-muted-foreground outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
          onClick={() => setDetailsOpen((current) => !current)}
          type="button"
        >
          <ToolActivityIcon
            command={isCommandToolCall(call)}
            status={status}
            webSearch={call.name === "web_search"}
          />
          <span className="shrink-0">{command.label}</span>
          {command.detail ? (
            <span
              className="min-w-0 truncate font-mono text-muted-foreground"
              title={command.detail}
            >
              {command.detail}
            </span>
          ) : null}
          <ChevronRight
            aria-hidden="true"
            className={cn(
              "size-3 shrink-0 text-muted-foreground/80 transition-transform duration-150",
              detailsOpen && "rotate-90",
            )}
          />
        </button>
      ) : (
        <div className="aivo-tool-call-row flex min-w-0 items-center gap-2 text-muted-foreground">
          <ToolActivityIcon
            command={isCommandToolCall(call)}
            status={status}
            webSearch={call.name === "web_search"}
          />
          <span className="shrink-0">{command.label}</span>
          {command.detail ? (
            <span
              className="min-w-0 truncate font-mono text-muted-foreground"
              title={command.detail}
            >
              {command.detail}
            </span>
          ) : null}
        </div>
      )}
      {hasDetails ? (
        <AnimatedDisclosure open={detailsOpen}>
          <ToolCallCommandLine compact toolCall={call} />
        </AnimatedDisclosure>
      ) : null}
    </div>
  );
}

function toolCallHasDetails(call: domain.ToolCall) {
  if (call.name === "web_search") return false;
  return Boolean(
    getToolResultText(call) ||
      getRetainedOutputRefs(call).length > 0 ||
      getToolCallFileChanges(call).length > 0 ||
      Object.keys(call.arguments ?? {}).length > 0,
  );
}

function ToolActivityIcon({
  command = false,
  status,
  webSearch = false,
}: {
  command?: boolean;
  status: string;
  webSearch?: boolean;
}) {
  if (status === "running") {
    return (
      <Loader2 aria-hidden="true" className="size-3 shrink-0 animate-spin" />
    );
  }
  if (status === "pending_approval") {
    return (
      <Clock3
        aria-hidden="true"
        className="size-3 shrink-0 text-amber-600 dark:text-amber-300"
      />
    );
  }
  if (status === "failed") {
    return (
      <CircleAlert
        aria-hidden="true"
        className="size-3 shrink-0 text-destructive"
      />
    );
  }
  if (command) {
    return <SquareTerminal aria-hidden="true" className="size-3 shrink-0" />;
  }
  if (webSearch) {
    return <Globe2 aria-hidden="true" className="size-3 shrink-0" />;
  }
  return <Wrench aria-hidden="true" className="size-3 shrink-0" />;
}

function activitySummary({
  groups,
  hasFailed,
  isRunning,
  needsApproval,
}: {
  groups: ToolCallGroup[];
  hasFailed: boolean;
  isRunning: boolean;
  needsApproval: boolean;
}) {
  const callCount = groups.reduce(
    (count, group) => count + group.calls.length,
    0,
  );
  if (needsApproval) return `等待批准 ${callCount} 项操作`;
  if (hasFailed) return `完成 ${callCount} 项操作，其中存在失败`;
  if (isRunning)
    return groups.length === 1
      ? groups[0].title
      : `正在执行 ${callCount} 项操作`;
  return groups.length === 1 ? groups[0].title : `已完成 ${callCount} 项操作`;
}
