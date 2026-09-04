import { ChevronRight, RotateCcw, Trash2 } from "lucide-react";
import { useEffect, useState, type ReactNode } from "react";

import { Markdown } from "@/components/markdown";
import { Button } from "@/components/ui/button";
import { formatThinkingTime } from "@/features/projects/conversation-timeline-display-model";
import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";
import { cn } from "@/lib/utils";
import { CopyTextButton } from "./conversation-timeline-copy-button";
import { useToolTurnExpansion } from "./conversation-timeline-tool-turn-expansion";
import type { ConversationTimelineActions } from "./conversation-timeline-types";

export function AssistantPreamble({
  text,
  workspaceRoot,
}: {
  text: string;
  workspaceRoot: string;
}) {
  return (
    <div className="aivo-assistant-preamble animate-in fade-in slide-in-from-bottom-2 text-sm duration-300">
      <Markdown content={text} isFinished workspaceRoot={workspaceRoot} />
    </div>
  );
}

export function StoppedResponse({
  stoppedSeconds,
}: {
  stoppedSeconds: number;
}) {
  return (
    <AssistantCompletionStatus>
      你在 {formatThinkingTime(stoppedSeconds)} 后停止了
    </AssistantCompletionStatus>
  );
}

export function ThinkingResponse({
  actionHeading,
  isExecuting,
  responseSeconds,
  showSkeleton,
}: {
  actionHeading?: string;
  isExecuting: boolean;
  responseSeconds: number;
  showSkeleton: boolean;
}) {
  return (
    <div className="animate-in flex min-w-0 max-w-full flex-col items-stretch gap-3 fade-in slide-in-from-bottom-2 duration-300">
      <ThinkingStatus
        actionHeading={actionHeading}
        responseSeconds={responseSeconds}
      />
      {showSkeleton ? (
        <div className="flex min-w-0 max-w-full flex-col gap-3.5">
          <div className="h-4 w-full max-w-[540px] rounded-full bg-muted" />
          <div className="h-4 w-full max-w-[680px] rounded-full bg-muted" />
          <div className="h-4 w-full max-w-[460px] rounded-full bg-muted" />
        </div>
      ) : null}
    </div>
  );
}

export function AssistantStatus({
  actionHeading,
  completed,
  hasToolActivity,
  isExecuting,
  model,
  modelProvider,
  responseSeconds,
  turnId,
}: {
  actionHeading?: string;
  completed: boolean;
  hasToolActivity: boolean;
  isExecuting: boolean;
  model?: string;
  modelProvider?: string;
  responseSeconds: number;
  turnId: string;
}) {
  if (!completed) {
    return (
      <ThinkingStatus
        actionHeading={actionHeading}
        responseSeconds={responseSeconds}
      />
    );
  }

  return (
    <AssistantCompletionStatus
      hasToolActivity={hasToolActivity}
      turnId={turnId}
    >
      <span>
        用时 {formatThinkingTime(responseSeconds)}
        {modelProvider || model ? (
          <span className="ml-2 rounded-full bg-muted px-2 py-0.5 text-[11px] text-muted-foreground">
            {[modelProvider, model].filter(Boolean).join(" · ")}
          </span>
        ) : null}
      </span>
    </AssistantCompletionStatus>
  );
}

function AssistantCompletionStatus({
  children,
  hasToolActivity = false,
  turnId = "",
}: {
  children: ReactNode;
  hasToolActivity?: boolean;
  turnId?: string;
}) {
  const { expanded, toggle } = useToolTurnExpansion(turnId);
  if (hasToolActivity) {
    return (
      <div className="aivo-assistant-status animate-in fade-in slide-in-from-bottom-2 duration-300">
        <button
          aria-expanded={expanded}
          className="aivo-assistant-status-toggle"
          onClick={toggle}
          type="button"
        >
          {children}
          <ChevronRight
            aria-hidden="true"
            className={cn(
              "size-3 shrink-0 text-muted-foreground/80 transition-transform duration-150",
              expanded && "rotate-90",
            )}
          />
        </button>
      </div>
    );
  }

  return (
    <div
      className="aivo-assistant-status animate-in fade-in slide-in-from-bottom-2 duration-300"
      role="status"
    >
      {children}
    </div>
  );
}

function ThinkingStatus({
  actionHeading,
  responseSeconds,
}: {
  actionHeading?: string;
  responseSeconds: number;
}) {
  const elapsedSeconds = useElapsedSeconds(responseSeconds);
  const statusText = `已处理 ${formatThinkingTime(elapsedSeconds)}`;

  return (
    <div
      className="aivo-assistant-status animate-in fade-in slide-in-from-bottom-2 duration-300"
      role="status"
    >
      <ShimmerText text={statusText} />
      {actionHeading ? (
        <div className="min-w-0 truncate text-muted-foreground">
          {actionHeading}
        </div>
      ) : null}
    </div>
  );
}

function useElapsedSeconds(responseSeconds: number) {
  const [elapsedSeconds, setElapsedSeconds] = useState(responseSeconds);

  useEffect(() => {
    setElapsedSeconds(responseSeconds);
  }, [responseSeconds]);

  useEffect(() => {
    const timer = window.setInterval(() => {
      setElapsedSeconds((seconds) => seconds + 1);
    }, 1000);
    return () => window.clearInterval(timer);
  }, []);

  return elapsedSeconds;
}

function ShimmerText({ text }: { text: string }) {
  return (
    <span aria-label={text} className="aivo-text-shimmer text-muted-foreground">
      <span data-slot="text-shimmer-char">
        <span aria-hidden="true" data-slot="text-shimmer-char-base">
          {text}
        </span>
        <span aria-hidden="true" data-slot="text-shimmer-char-shimmer">
          {text}
        </span>
      </span>
    </span>
  );
}

export function AssistantResponse({
  actions,
  completedAt,
  responseText,
  turn,
  workspaceRoot,
}: {
  actions: ConversationTimelineActions;
  completedAt: Date | null;
  responseText: string;
  turn: ConversationTurn;
  workspaceRoot: string;
}) {
  return (
    <div className="aivo-assistant-response group/assistant-response relative">
      <Markdown
        content={responseText}
        isFinished={Boolean(completedAt)}
        workspaceRoot={workspaceRoot}
      />
      <div className="relative mb-3 h-6">
        <div className="absolute left-0 top-0 z-10 flex items-center gap-2">
          <CopyTextButton ariaLabel="复制回复" text={responseText} />
          {turn.turnId && completedAt && actions.onRetryTurn ? (
            <Button
              aria-label="重试"
              onClick={() => actions.onRetryTurn?.(turn)}
              size="icon-sm"
              title="重试"
              type="button"
              variant="ghost"
            >
              <RotateCcw />
            </Button>
          ) : null}
          {turn.assistantEventId && actions.onDeleteAssistantMessage ? (
            <Button
              aria-label="删除回复"
              onClick={() => actions.onDeleteAssistantMessage?.(turn)}
              size="icon-sm"
              title="删除回复"
              type="button"
              variant="ghost"
            >
              <Trash2 />
            </Button>
          ) : null}
        </div>
      </div>
    </div>
  );
}
