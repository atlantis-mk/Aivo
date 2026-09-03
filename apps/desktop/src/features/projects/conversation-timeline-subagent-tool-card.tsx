import { ExternalLink } from "lucide-react";

import { cn } from "@/lib/utils";
import {
  agentModeDisplayName,
  delegateToolCallSessionId,
  subagentStatusClass,
  subagentStatusLabel,
} from "@/features/projects/conversation-timeline-subagent-model";
import {
  stringArg,
  truncateInline,
} from "@/features/projects/conversation-timeline-value-model";
import type { AgentRun } from "@/services/aivo";
import type { domain } from "../../../bridge/go/models";

export function SubagentToolCard({
  agentRun,
  onOpenSession,
  toolCall,
}: {
  agentRun?: AgentRun;
  onOpenSession?: (sessionId: string) => void;
  toolCall: domain.ToolCall;
}) {
  const sessionId = agentRun?.sessionId ?? delegateToolCallSessionId(toolCall);
  const status = agentRun?.status ?? toolCall.status;
  const prompt =
    agentRun?.prompt ||
    stringArg(toolCall.arguments ?? {}, "prompt") ||
    stringArg(toolCall.arguments ?? {}, "goal") ||
    "子代理任务";
  const title =
    stringArg(toolCall.arguments ?? {}, "title") || truncateInline(prompt, 72);
  const mode =
    agentRun?.mode ||
    stringArg(toolCall.arguments ?? {}, "mode") ||
    "assistant";
  const clickable = Boolean(sessionId && onOpenSession);
  const modeLabel = agentModeDisplayName(mode);
  const statusLabel = subagentStatusLabel(status);

  return (
    <button
      aria-label={`子代理 ${modeLabel}：${title}，${statusLabel}`}
      className={cn(
        "group/subagent inline-flex max-w-full items-center gap-2 rounded-md border border-border bg-background px-3 py-2 text-left text-sm shadow-sm transition-colors",
        clickable && "hover:bg-muted/50",
        !clickable && "cursor-default opacity-80",
      )}
      disabled={!clickable}
      onClick={() => {
        if (sessionId) onOpenSession?.(sessionId);
      }}
      title={`${modeLabel} · ${title} · ${statusLabel}`}
      type="button"
    >
      <span className="shrink-0 text-amber-600 dark:text-amber-400">
        {modeLabel}
      </span>
      <span className="min-w-0 truncate text-foreground">{title}</span>
      <span className={cn("shrink-0 text-xs", subagentStatusClass(status))}>
        {statusLabel}
      </span>
      {clickable ? (
        <span className="-mr-1 flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground opacity-0 transition-opacity group-hover/subagent:opacity-100 group-focus-visible/subagent:opacity-100">
          <ExternalLink />
        </span>
      ) : null}
    </button>
  );
}
