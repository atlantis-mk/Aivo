import { RotateCcw, Trash2 } from "lucide-react";

import { Markdown } from "@/components/markdown";
import { Button } from "@/components/ui/button";
import { Marker, MarkerContent } from "@/components/ui/marker";
import { formatCompletionTime, formatThinkingTime } from "@/features/projects/conversation-timeline-display-model";
import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";
import { CopyTextButton } from "./conversation-timeline-copy-button";
import type { ConversationTimelineActions } from "./conversation-timeline-types";

export function AssistantPreamble({ text }: { text: string }) {
  return (
    <div className="animate-in fade-in slide-in-from-bottom-2 text-sm duration-300">
      <Markdown content={text} isFinished />
    </div>
  );
}

export function StoppedResponse({ stoppedSeconds }: { stoppedSeconds: number }) {
  return (
    <Marker
      className="animate-in fade-in slide-in-from-bottom-2 text-sm duration-300"
      role="status"
      variant="separator"
    >
      <MarkerContent>
        你在 {formatThinkingTime(stoppedSeconds)} 后停止了
      </MarkerContent>
    </Marker>
  );
}

export function ThinkingResponse({
  actionHeading,
  showSkeleton,
}: {
  actionHeading?: string;
  showSkeleton: boolean;
}) {
  return (
    <div className="animate-in flex min-w-0 max-w-full flex-col items-stretch gap-3 fade-in slide-in-from-bottom-2 duration-300">
      <ThinkingStatus actionHeading={actionHeading} />
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
  responseSeconds,
}: {
  actionHeading?: string;
  completed: boolean;
  responseSeconds: number;
}) {
  if (!completed) {
    return <ThinkingStatus actionHeading={actionHeading} />;
  }

  return (
    <Marker
      className="animate-in fade-in slide-in-from-bottom-2 text-sm duration-300"
      role="status"
      variant="separator"
    >
      <MarkerContent>已完成 {formatThinkingTime(responseSeconds)}</MarkerContent>
    </Marker>
  );
}

function ThinkingStatus({ actionHeading }: { actionHeading?: string }) {
  return (
    <div
      className="animate-in flex min-w-0 flex-col gap-1.5 fade-in slide-in-from-bottom-2 text-sm duration-300"
      role="status"
    >
      <ShimmerText text="正在思考" />
      {actionHeading ? (
        <div className="min-w-0 truncate text-muted-foreground">
          {actionHeading}
        </div>
      ) : null}
    </div>
  );
}

function ShimmerText({ text }: { text: string }) {
  return (
    <span
      aria-label={text}
      className="aivo-text-shimmer font-semibold text-muted-foreground"
    >
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
}: {
  actions: ConversationTimelineActions;
  completedAt: Date | null;
  responseText: string;
  turn: ConversationTurn;
}) {
  return (
    <div className="group/assistant-response relative">
      <Markdown content={responseText} isFinished={Boolean(completedAt)} />
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
          {completedAt ? (
            <span className="text-sm">{formatCompletionTime(completedAt)}</span>
          ) : null}
        </div>
      </div>
    </div>
  );
}
