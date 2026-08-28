import { useEffect, useState } from "react";
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
  toolCall,
}: {
  toolCall: domain.ToolCall;
}) {
  const failed = toolCall.status === "failed";
  const running = toolCall.status === "running";
  const pendingApproval = toolCall.status === "pending_approval";
  const commandTool = isCommandToolCall(toolCall);
  const skillTool = toolCall.name === "skill";
  const command = getToolCallCommand(toolCall);
  const fileChanges = getToolCallFileChanges(toolCall);
  const retainedRefs = getRetainedOutputRefs(toolCall);
  const retainedRef = retainedRefs[0] ?? "";
  const retainedRefsKey = retainedRefs.join("\n");
  const showFileChanges =
    toolCallKind(toolCall) === "write" && fileChanges.length > 0;
  const resultText =
    toolCall.name === "ls" ||
    toolCall.name === "list_files" ||
    commandTool ||
    skillTool ||
    failed
      ? getToolResultText(toolCall)
      : "";
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
  const [resultDetailsOpen, setResultDetailsOpen] = useState(false);
  const [retainedOutput, setRetainedOutput] =
    useState<RetainedOutputViewState | null>(null);
  const hasRetainedOutputRef = retainedRefs.length > 0;
  const showInlineResult = Boolean(
    resultText &&
      (!resultDetailsExpandable || resultDetailsOpen) &&
      !hasRetainedOutputRef,
  );

  useEffect(() => {
    setResultDetailsOpen(false);
    setRetainedOutput(null);
  }, [toolCall.id, resultText, retainedRef, retainedRefsKey]);

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
    resultDetailsExpandable && "cursor-pointer rounded-sm hover:bg-muted/50",
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
    <div className="flex min-w-0 flex-col gap-1">
      {showCommandLine || showStatusLine ? (
        resultDetailsExpandable ? (
          <button
            aria-expanded={resultDetailsOpen}
            className={statusLineClassName}
            onClick={() => setResultDetailsOpen((current) => !current)}
            type="button"
          >
            {statusLineContent}
          </button>
        ) : (
          <div className={statusLineClassName}>{statusLineContent}</div>
        )
      ) : null}
      <AnimatedDisclosure open={showFileChanges}>
        <ToolFileChangeLines files={fileChanges} live={running} />
      </AnimatedDisclosure>
      <AnimatedDisclosure open={showInlineResult}>
        {resultText && commandTool ? (
          <InlineShellPreview resultText={resultText} toolCall={toolCall} />
        ) : resultText ? (
          <pre className="max-h-56 overflow-auto whitespace-pre-wrap break-words rounded-md bg-muted/50 px-3 py-2 font-mono text-xs leading-relaxed text-muted-foreground">
            {resultText}
          </pre>
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
              <pre className="max-h-80 overflow-auto whitespace-pre-wrap break-words rounded-md bg-muted/50 px-3 py-2 font-mono text-xs leading-relaxed text-muted-foreground">
                {retainedOutput.content}
              </pre>
            ) : null}
          </AnimatedDisclosure>
        </div>
      ) : null}
    </div>
  );
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
