import { ShieldCheck } from "lucide-react";

import { PermissionRequestCard } from "@/features/projects/project-permission-request-card";
import type { PermissionRequest } from "@/services/aivo";

export function PermissionApprovalDock({
  permissions,
}: {
  permissions: PermissionRequest[];
}) {
  if (permissions.length === 0) return null;

  return (
    <div
      className="absolute bottom-4 left-1/2 z-20 w-[calc(100%-2rem)] max-w-[760px] -translate-x-1/2 transition-[margin,transform] duration-500 ease-[cubic-bezier(0.22,1,0.36,1)] sm:bottom-6 sm:w-[calc(100%-48px)]"
      data-assistant-hover-ignore="true"
    >
      <div className="overflow-hidden rounded-2xl border border-border/80 bg-popover text-popover-foreground shadow-2xl shadow-foreground/15 ring-1 ring-foreground/5">
        <div className="flex items-center gap-3 border-b border-border/70 px-4 py-3">
          <span className="flex size-8 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground">
            <ShieldCheck className="size-4" />
          </span>
          <div className="min-w-0 flex-1">
            <div className="truncate text-sm font-semibold">等待权限审批</div>
            <div className="truncate text-xs text-muted-foreground">
              审批后任务会继续执行；拒绝会停止这次工具调用。
            </div>
          </div>
          {permissions.length > 1 ? (
            <span className="rounded-full bg-primary/10 px-2 py-1 text-xs text-primary">
              {permissions.length} 项
            </span>
          ) : null}
        </div>
        <div className="max-h-[min(54vh,420px)] overflow-auto">
          <div className="flex flex-col">
            {permissions.map((permission, index) => (
              <PermissionRequestCard
                compact={permissions.length > 1}
                index={index}
                key={permission.id}
                permission={permission}
              />
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
