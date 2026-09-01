import { ExternalLink, RotateCcw, Undo2 } from "lucide-react";

import { ScrollArea } from "@/components/ui/scroll-area";
import type { ToolActivityFileTab } from "@/features/projects/tool-activity-model";
import { useFollowScrollToEnd } from "@/features/projects/tool-activity-follow-scroll";
import {
  diffLineClass,
  fileDisplayPath,
  splitFilePath,
} from "@/features/projects/tool-activity-sidebar-model";
import { ToolActivityStatusIcon } from "@/features/projects/tool-activity-status-icon";
import { cn } from "@/lib/utils";

export function FileActivityDetail({
  onApplyFileState,
  tab,
}: {
  onApplyFileState?: (
    tab: ToolActivityFileTab,
    targetState: "before" | "after",
  ) => void;
  tab: ToolActivityFileTab;
}) {
  const body = tab.diff || tab.contentPreview || "";
  const followScrollKey = `${body}\u0000${tab.error}\u0000${tab.status}`;
  const { endRef, scrollAreaRef } = useFollowScrollToEnd(followScrollKey);

  return (
    <div className="flex h-full min-h-0 flex-col bg-background p-2">
      <section className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-2xl border border-border/80 bg-card text-card-foreground shadow-sm shadow-foreground/[0.03]">
        <div className="shrink-0 px-4 pt-3 pb-2">
          <FileActivityHeader onApplyFileState={onApplyFileState} tab={tab} />
          {tab.revertReason ? (
            <div className="mt-1 px-1 text-[11px] text-muted-foreground">
              {tab.revertReason}
            </div>
          ) : null}
        </div>
        <ScrollArea className="min-h-0 flex-1" ref={scrollAreaRef}>
          <div className="flex min-h-full flex-col gap-3 px-4 pb-4 pt-1">
            {tab.error ? <ErrorBlock message={tab.error} /> : null}
            {body ? (
              <pre className="min-h-0 whitespace-pre-wrap break-words font-mono text-[11px] leading-relaxed">
                {body.split("\n").map((line, index) => (
                  <span
                    className={cn("block min-h-[1.35em]", diffLineClass(line))}
                    key={`${index}:${line}`}
                  >
                    {line || " "}
                  </span>
                ))}
              </pre>
            ) : (
              <div className="rounded-lg border border-border/70 bg-muted/20 p-3 text-xs text-muted-foreground">
                暂无可显示内容
              </div>
            )}
            <div ref={endRef} />
          </div>
        </ScrollArea>
      </section>
    </div>
  );
}

function FileActivityHeader({
  onApplyFileState,
  tab,
}: {
  onApplyFileState?: (
    tab: ToolActivityFileTab,
    targetState: "before" | "after",
  ) => void;
  tab: ToolActivityFileTab;
}) {
  const { directory, name } = splitFilePath(fileDisplayPath(tab));
  const openPath = tab.movePath || tab.path;
  const canRevert = tab.revertible !== false;
  const canUnrevert = tab.unrevertible !== false;

  return (
    <div className="flex min-w-0 items-center gap-2">
      <div
        className="relative min-w-0 flex-1 overflow-hidden text-sm leading-none"
        title={fileDisplayPath(tab)}
      >
        <div className="relative left-full flex w-max max-w-none -translate-x-full items-baseline whitespace-nowrap">
          {directory ? (
            <span className="text-muted-foreground">{directory}/</span>
          ) : null}
          <span className="font-semibold text-foreground">{name}</span>
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-1.5 font-mono text-xs">
        {typeof tab.additions === "number" ? (
          <span className="text-emerald-600 dark:text-emerald-400">
            +{tab.additions}
          </span>
        ) : null}
        {typeof tab.deletions === "number" ? (
          <span className="text-rose-600 dark:text-rose-400">
            -{tab.deletions}
          </span>
        ) : null}
        <ToolActivityStatusIcon className="size-3.5 shrink-0" status={tab.status} />
        {tab.status === "success" && !tab.draft && tab.turnId && onApplyFileState ? (
          <>
            <button
              aria-label="回滚此文件改动"
              className="flex size-5 shrink-0 items-center justify-center rounded-sm text-muted-foreground hover:bg-muted hover:text-foreground disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent disabled:hover:text-muted-foreground"
              disabled={!canRevert}
              onClick={() => onApplyFileState(tab, "before")}
              title={
                canRevert
                  ? "回滚此文件改动"
                  : tab.revertReason || "当前文件状态不可回滚"
              }
              type="button"
            >
              <Undo2 aria-hidden="true" />
            </button>
            <button
              aria-label="恢复此文件改动"
              className="flex size-5 shrink-0 items-center justify-center rounded-sm text-muted-foreground hover:bg-muted hover:text-foreground disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent disabled:hover:text-muted-foreground"
              disabled={!canUnrevert}
              onClick={() => onApplyFileState(tab, "after")}
              title={
                canUnrevert
                  ? "恢复此文件改动"
                  : tab.revertReason || "当前文件状态不可恢复"
              }
              type="button"
            >
              <RotateCcw aria-hidden="true" />
            </button>
          </>
        ) : null}
        <button
          aria-label="用系统默认应用打开文件"
          className="flex size-5 shrink-0 items-center justify-center rounded-sm text-muted-foreground hover:bg-muted hover:text-foreground"
          onClick={() => {
            void window.aivo?.openPath(openPath).catch((error: unknown) => {
              console.error("Failed to open file", error);
            });
          }}
          title={openPath}
          type="button"
        >
          <ExternalLink aria-hidden="true" />
        </button>
      </div>
    </div>
  );
}

function ErrorBlock({ message }: { message: string }) {
  return (
    <div className="rounded-md border border-destructive/30 bg-destructive/10 p-2 text-xs text-destructive">
      {message}
    </div>
  );
}
