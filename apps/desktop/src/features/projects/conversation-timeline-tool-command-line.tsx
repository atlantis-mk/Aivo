import { useEffect, useState, type ReactNode } from "react";
import { ChevronDown } from "lucide-react";

import { cn } from "@/lib/utils";
import { AnimatedDisclosure } from "@/features/projects/conversation-timeline-disclosure";
import {
  failedToolRunDetail,
  getRetainedOutputRefs,
  getToolCallCommand,
  getToolCallFileChanges,
  getToolResultText,
  isCommandToolCall,
  toolCallKind,
} from "@/features/projects/conversation-timeline-tool-model";
import { readRetainedOutput } from "@/services/aivo";
import { ToolFileChangeLines } from "./conversation-timeline-tool-file-changes";
import { InlineShellPreview } from "./conversation-timeline-tool-shell-preview";
import type { domain } from "../../../bridge/go/models";

export function ToolCallCommandLine({
  compact = false,
  toolCall,
}: {
  compact?: boolean;
  toolCall: domain.ToolCall;
}) {
  const failed = toolCall.status === "failed";
  const running = toolCall.status === "running";
  const pendingApproval = toolCall.status === "pending_approval";
  const commandTool = isCommandToolCall(toolCall);
  const skillTool = toolCall.name === "skill";
  const command = getToolCallCommand(toolCall);
  const argumentsText = formatToolArguments(toolCall.arguments);
  const fileChanges = getToolCallFileChanges(toolCall);
  const retainedRefs = getRetainedOutputRefs(toolCall);
  const retainedRef = retainedRefs[0] ?? "";
  const retainedRefsKey = retainedRefs.join("\n");
  const showFileChanges =
    toolCallKind(toolCall) === "write" && fileChanges.length > 0;
  const resultText = getToolResultText(toolCall);
  const showCommandLine =
    skillTool || commandTool || (!resultText && !showFileChanges);
  const showRunningStatus = running && !showFileChanges;
  const showStatusLine = showRunningStatus || pendingApproval || failed;
  const failedRunDetail =
    failed && !showCommandLine ? failedToolRunDetail(toolCall, resultText) : "";
  const resultDetailsExpandable = Boolean(
    (skillTool && (resultText || retainedRefs.length > 0)) ||
      (resultText && (failed || commandTool) && retainedRefs.length === 0),
  );
  const [resultDetailsOpen, setResultDetailsOpen] = useState(compact);
  const [retainedOutput, setRetainedOutput] =
    useState<RetainedOutputViewState | null>(null);
  const hasRetainedOutputRef = retainedRefs.length > 0;
  const showInlineResult = Boolean(
    resultText &&
      (!resultDetailsExpandable || resultDetailsOpen) &&
      !hasRetainedOutputRef,
  );
  const showArguments = Boolean(argumentsText) && !commandTool;
  const hasDetailContent = Boolean(
    showArguments ||
      showFileChanges ||
      showInlineResult ||
      retainedRefs.length > 0,
  );

  useEffect(() => {
    setResultDetailsOpen(compact);
    setRetainedOutput(null);
  }, [compact, retainedRef, retainedRefsKey, resultText, toolCall.id]);

  useEffect(() => {
    let cancelled = false;
    const ref = retainedRef;
    if (ref && (!skillTool || resultDetailsOpen)) {
      void loadRetainedOutput(ref, () => cancelled);
    }
    return () => {
      cancelled = true;
    };
  }, [retainedRef, resultDetailsOpen, skillTool]);

  async function loadRetainedOutput(ref: string, isCancelled: () => boolean) {
    let content = "";
    let nextOffset = 0;
    let size = 0;
    let truncated = true;
    setRetainedOutput({
      ref,
      content,
      nextOffset,
      size,
      truncated: false,
      loading: true,
    });
    try {
      while (!isCancelled() && truncated) {
        const offset = nextOffset;
        const chunk = await readRetainedOutput({
          ref,
          offset,
          limit: 100_000,
        });
        content += chunk.content;
        nextOffset = chunk.nextOffset;
        size = chunk.size;
        truncated = Boolean(chunk.truncated) && nextOffset > offset;
        if (isCancelled()) return;
        setRetainedOutput({
          ref: chunk.ref,
          content,
          nextOffset,
          size,
          truncated,
          loading: truncated,
        });
      }
      if (isCancelled()) return;
      setRetainedOutput({
        ref,
        content,
        nextOffset,
        size,
        truncated: false,
        loading: false,
      });
    } catch (error) {
      if (isCancelled()) return;
      setRetainedOutput({
        ref,
        content,
        nextOffset,
        size,
        truncated: false,
        loading: false,
        error: error instanceof Error ? error.message : String(error),
      });
    }
  }

  const statusLineClassName = cn(
    "flex min-w-0 items-baseline gap-2 text-sm",
    failed && "text-destructive",
    running && "text-muted-foreground",
    pendingApproval && "text-amber-700 dark:text-amber-300",
    resultDetailsExpandable && "cursor-pointer rounded-md hover:bg-muted/50",
  );
  const statusLineContent = (
    <>
      {showCommandLine ? (
        <>
          <span className="shrink-0 text-foreground">{command.label}</span>
          <span className="min-w-0 truncate text-muted-foreground">
            {command.detail}
          </span>
        </>
      ) : null}
      {showRunningStatus ? (
        <span className="shrink-0 text-muted-foreground">运行中</span>
      ) : null}
      {pendingApproval ? (
        <span className="shrink-0 text-amber-700 dark:text-amber-300">
          等待批准
        </span>
      ) : null}
      {failedRunDetail ? (
        <>
          <span className="shrink-0 text-muted-foreground">已运行</span>
          <span className="min-w-0 truncate text-muted-foreground">
            {failedRunDetail}
          </span>
        </>
      ) : null}
      {failed ? <span className="shrink-0 text-destructive">失败</span> : null}
      {resultDetailsExpandable ? (
        <ChevronDown
          className={cn(
            "size-3 shrink-0 self-center text-muted-foreground transition-transform",
            resultDetailsOpen && "rotate-180",
          )}
        />
      ) : null}
    </>
  );

  return (
    <div
      className={cn(
        "aivo-tool-call-details my-0 flex min-w-0 flex-col overflow-hidden rounded-md border border-border/70 bg-muted/35 text-card-foreground",
        compact && "aivo-tool-call-details--compact",
      )}
      data-assistant-hover-ignore="true"
    >
      {!compact && (showCommandLine || showStatusLine) ? (
        <div className="min-h-11 px-4 pt-3 pb-2">
          {resultDetailsExpandable ? (
            <button
              aria-expanded={resultDetailsOpen}
              className={cn(statusLineClassName, "w-full")}
              onClick={() => setResultDetailsOpen((current) => !current)}
              type="button"
            >
              {statusLineContent}
            </button>
          ) : (
            <div className={statusLineClassName}>{statusLineContent}</div>
          )}
        </div>
      ) : null}
      {hasDetailContent ? (
        <div className="min-w-0">
          <AnimatedDisclosure open={showArguments}>
            {showArguments ? (
              <ToolCallDetailBlock label="参数">
                <pre className="max-h-56 overflow-auto whitespace-pre-wrap break-words font-mono text-xs leading-relaxed text-muted-foreground">
                  {argumentsText}
                </pre>
              </ToolCallDetailBlock>
            ) : null}
          </AnimatedDisclosure>
          <AnimatedDisclosure open={showFileChanges}>
            <ToolFileChangeLines files={fileChanges} live={running} />
          </AnimatedDisclosure>
          <AnimatedDisclosure open={showInlineResult}>
            {resultText && commandTool ? (
              <InlineShellPreview resultText={resultText} toolCall={toolCall} />
            ) : resultText ? (
              <ToolCallDetailBlock label={failed ? "错误" : "结果"}>
                <pre
                  className={cn(
                    "max-h-56 overflow-auto whitespace-pre-wrap break-words font-mono text-xs leading-relaxed text-muted-foreground",
                    failed && "text-destructive",
                  )}
                >
                  {resultText}
                </pre>
              </ToolCallDetailBlock>
            ) : null}
          </AnimatedDisclosure>
          {retainedRefs.length > 0 && (!skillTool || resultDetailsOpen) ? (
            <div className="flex min-w-0 flex-col gap-1">
              {retainedOutput?.loading && !retainedOutput.content ? (
                <div className="text-xs text-muted-foreground">
                  加载完整输出...
                </div>
              ) : null}
              {retainedOutput?.error ? (
                <div className="text-xs text-destructive">
                  {retainedOutput.error}
                </div>
              ) : null}
              <AnimatedDisclosure open={Boolean(retainedOutput?.content)}>
                {retainedOutput?.content ? (
                  <ToolCallDetailBlock label="完整输出">
                    <pre className="max-h-80 overflow-auto whitespace-pre-wrap break-words font-mono text-xs leading-relaxed text-muted-foreground">
                      {retainedOutput.content}
                    </pre>
                  </ToolCallDetailBlock>
                ) : null}
              </AnimatedDisclosure>
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

function ToolCallDetailBlock({
  children,
  label,
}: {
  children: ReactNode;
  label: string;
}) {
  return (
    <section className="aivo-tool-detail-block mb-2 min-w-0 rounded-sm border border-border/50 bg-background/45 px-2.5 py-2">
      <div className="mb-2 text-[11px] text-muted-foreground">
        {label}
      </div>
      {children}
    </section>
  );
}

function formatToolArguments(value: Record<string, unknown> | undefined) {
  if (!value || Object.keys(value).length === 0) return "";
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

type RetainedOutputViewState = {
  ref: string;
  content: string;
  nextOffset: number;
  size: number;
  truncated: boolean;
  loading?: boolean;
  error?: string;
};
