import { Check, CircleAlert, Loader2 } from "lucide-react";
import { useMemo } from "react";

import { ScrollArea } from "@/components/ui/scroll-area";
import {
  compactShellCommand,
  shellPreviewEntries,
  type ShellPreviewEntry,
} from "@/features/projects/conversation-timeline-shell-model";
import type { domain } from "../../../bridge/go/models";

export function InlineShellPreview({
  resultText,
  toolCall,
}: {
  resultText: string;
  toolCall: domain.ToolCall;
}) {
  const entries = shellPreviewEntries(toolCall, resultText);
  if (entries.length === 0) return null;

  return (
    <div className="aivo-shell-preview-list flex min-w-0 flex-col gap-2">
      {entries.map((entry) => (
        <ShellPreviewEntryView entry={entry} key={entry.id} />
      ))}
    </div>
  );
}

function ShellPreviewEntryView({ entry }: { entry: ShellPreviewEntry }) {
  const status = useMemo(() => shellPreviewStatus(entry), [entry]);
  const hasOutput = Boolean(entry.stdout || entry.stderr || entry.error);
  return (
    <div
      className="aivo-shell-preview min-w-0"
      data-has-output={hasOutput}
    >
      <div className="aivo-shell-preview-label">Shell</div>
      <ScrollArea className="aivo-shell-preview-command max-h-12 min-w-0 max-w-full overflow-hidden [&>[data-slot=scroll-area-viewport]]:h-auto [&>[data-slot=scroll-area-viewport]]:max-h-12 [&>[data-slot=scroll-area-viewport]]:overflow-x-hidden [&>[data-slot=scroll-area-viewport]>div]:!block [&>[data-slot=scroll-area-viewport]>div]:!w-full [&>[data-slot=scroll-area-viewport]>div]:!min-w-0">
        <pre className="aivo-shell-preview-content">
          <span className="text-muted-foreground">$ </span>
          <span>{compactShellCommand(entry.command)}</span>
        </pre>
      </ScrollArea>
      <ScrollArea className="aivo-shell-preview-result h-full min-h-0 min-w-0 max-w-full overflow-hidden [&>[data-slot=scroll-area-viewport]]:min-h-0 [&>[data-slot=scroll-area-viewport]]:overflow-x-hidden [&>[data-slot=scroll-area-viewport]>div]:!block [&>[data-slot=scroll-area-viewport]>div]:!w-full [&>[data-slot=scroll-area-viewport]>div]:!min-w-0">
        <pre className="aivo-shell-preview-content">
          {entry.stdout ? <span>{entry.stdout.trim()}</span> : null}
          {entry.stderr ? (
            <span className="text-destructive">{entry.stderr.trim()}</span>
          ) : null}
          {entry.error && !entry.stderr ? (
            <span className="text-destructive">{entry.error}</span>
          ) : null}
          {!entry.stdout && !entry.stderr && !entry.error ? (
            <span className="text-muted-foreground">
              {status === "running" ? "等待输出…" : "暂无输出"}
            </span>
          ) : null}
        </pre>
      </ScrollArea>
      <div className="aivo-shell-preview-status" data-status={status}>
        {status === "success" ? (
          <Check aria-hidden="true" className="size-4" strokeWidth={2} />
        ) : status === "running" ? (
          <Loader2 aria-hidden="true" className="size-4 animate-spin" />
        ) : status !== "exit" ? (
          <CircleAlert aria-hidden="true" className="size-4" strokeWidth={2} />
        ) : null}
        {shellPreviewStatusLabel(entry)}
      </div>
    </div>
  );
}

type ShellPreviewStatus =
  | "exit"
  | "failed"
  | "running"
  | "stopped"
  | "success"
  | "waiting";

function shellPreviewStatus(entry: {
  error?: string;
  exitCode?: number;
  isPty: boolean;
  state?: string;
  stderr: string;
}): ShellPreviewStatus {
  if (entry.exitCode !== undefined && entry.exitCode !== 0) return "exit";
  const state = entry.state?.toLowerCase().replaceAll("_", "");
  if (entry.error) return "failed";
  if (state === "waitinginput") return "waiting";
  if (state === "inprogress" || state === "running" || state === "starting") {
    return "running";
  }
  if (state === "failed" || state === "cancelled" || state === "interrupted") {
    return "failed";
  }
  if (entry.exitCode !== undefined) {
    return entry.exitCode === 0 ? "success" : "exit";
  }
  if (entry.isPty) return "stopped";
  return entry.stderr ? "failed" : "success";
}

function shellPreviewStatusLabel(entry: {
  exitCode?: number;
  isPty: boolean;
  state?: string;
  error?: string;
  stderr: string;
}) {
  switch (shellPreviewStatus(entry)) {
    case "success":
      return "成功";
    case "exit":
      return `退出码 ${entry.exitCode}`;
    case "running":
      return "运行中";
    case "waiting":
      return "等待输入";
    case "stopped":
      return "已停止";
    case "failed":
      return "失败";
  }
}
