import { useLayoutEffect } from "react";
import { ArrowLeft, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { agentModeLabel } from "@/features/projects/project-agent-mode-model";
import { cn } from "@/lib/utils";
import type { AgentRun } from "@/services/aivo";

export function SubagentSessionActionBar({
  agentRun,
  onBack,
  onCancel,
  onHeightChange,
}: {
  agentRun?: AgentRun;
  onBack: () => void;
  onCancel?: () => void;
  onHeightChange: (height: number) => void;
}) {
  const barHeight = 72;

  useLayoutEffect(() => {
    onHeightChange(barHeight);
  }, [onHeightChange]);

  const status = agentRun?.status || "";
  const modeLabel = agentModeLabel(agentRun?.mode || "assistant");
  const statusLabel = agentRunStatusLabel(status);

  return (
    <div
      className="flex min-w-0 items-center justify-between gap-3 rounded-2xl border border-border bg-card px-4 shadow-lg shadow-foreground/5"
      style={{ height: barHeight }}
    >
      <div className="flex min-w-0 items-center gap-3">
        <Button onClick={onBack} type="button" variant="outline">
          <ArrowLeft />
          返回父会话
        </Button>
        <div className="hidden min-w-0 flex-col sm:flex">
          <div className="truncate text-sm font-medium text-foreground">
            子代理 · {modeLabel}
          </div>
          <div className="truncate text-xs text-muted-foreground">
            {agentRun?.prompt || "只读子代理会话"}
          </div>
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-2">
        {statusLabel ? (
          <span className={cn("text-xs", agentRunStatusClass(status))}>
            {statusLabel}
          </span>
        ) : null}
        {onCancel ? (
          <Button onClick={onCancel} type="button" variant="destructive">
            <X />
            取消运行
          </Button>
        ) : null}
      </div>
    </div>
  );
}

function agentRunStatusLabel(status: string) {
  switch (status) {
    case "running":
      return "运行中";
    case "completed":
    case "success":
      return "已完成";
    case "failed":
      return "失败";
    case "cancelled":
      return "已取消";
    case "pending_approval":
      return "等待批准";
    default:
      return status;
  }
}

function agentRunStatusClass(status: string) {
  if (status === "completed" || status === "success") {
    return "text-emerald-600 dark:text-emerald-400";
  }
  if (status === "failed" || status === "cancelled") {
    return "text-destructive";
  }
  return "text-muted-foreground";
}
