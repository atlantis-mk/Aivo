import { useState } from "react";
import { Check } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  permissionAgentMode,
  permissionApprovalTarget,
  permissionApprovalTitle,
  permissionCommand,
  permissionFiles,
  permissionMCPRegistration,
  permissionProject,
  permissionRememberLabel,
  permissionToolsets,
  type PermissionActionState,
} from "@/features/projects/project-permission-approval-model";
import { cn } from "@/lib/utils";
import {
  approvePermissionRequest,
  denyPermissionRequest,
  type PermissionRequest,
} from "@/services/aivo";

export function PermissionRequestCard({
  compact,
  index,
  permission,
}: {
  compact: boolean;
  index: number;
  permission: PermissionRequest;
}) {
  const [action, setAction] = useState<PermissionActionState>("idle");
  const [remember, setRemember] = useState(false);
  const files = permissionFiles(permission);
  const command = permissionCommand(permission);
  const project = permissionProject(permission);
  const registration = permissionMCPRegistration(permission);
  const title = permissionApprovalTitle(permission, command);
  const target = permissionApprovalTarget(permission, command);
  const permissionModeLabel = permissionAgentMode(permission);
  const permissionToolsetLabel = permissionToolsets(permission).join(", ");
  const isBusy = action === "approving" || action === "denying";

  async function approve() {
    if (action === "approving" || action === "approved") return;
    setAction("approving");
    try {
      await approvePermissionRequest(permission.id, remember);
      setAction("approved");
    } catch {
      setAction("idle");
    }
  }

  async function deny() {
    if (action === "denying" || action === "denied") return;
    setAction("denying");
    try {
      await denyPermissionRequest(permission.id, remember, "Denied by user");
      setAction("denied");
    } catch {
      setAction("idle");
    }
  }

  return (
    <section className="border-b border-border/70 p-4 text-popover-foreground last:border-b-0">
      <div className="flex min-w-0 items-start gap-3">
        <span className="mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary">
          {index + 1}
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
            <h3 className="min-w-0 truncate text-sm font-semibold">{title}</h3>
            <span className="rounded-full bg-muted px-2 py-0.5 text-[11px] text-muted-foreground">
              {permission.toolName}
            </span>
          </div>
          <div className="mt-1 min-w-0 truncate text-xs text-muted-foreground">
            {target}
          </div>
          <div className="mt-1 min-w-0 truncate text-[11px] text-muted-foreground">
            mode: {permissionModeLabel || "assistant"}
            {permissionToolsetLabel
              ? ` · toolsets: ${permissionToolsetLabel}`
              : ""}
          </div>
        </div>
      </div>
      {files.length > 0 ? (
        <div
          className={cn(
            "mt-3 grid gap-1.5 text-xs",
            compact && files.length > 2 && "max-h-20 overflow-auto pr-1",
          )}
        >
          {files.map((file) => (
            <div
              className="flex min-w-0 items-center justify-between gap-3 rounded-md bg-muted/70 px-2 py-1.5"
              key={`${file.type}:${file.path}:${file.movePath}`}
            >
              <span className="min-w-0 truncate font-mono text-[11px]">
                <span className="mr-2 inline-flex min-w-4 justify-center rounded bg-background px-1 font-sans font-semibold text-muted-foreground">
                  {file.typeLabel}
                </span>
                {file.movePath ? `${file.path} -> ${file.movePath}` : file.path}
              </span>
              <span className="shrink-0 font-mono text-[11px] text-muted-foreground">
                +{file.additions} -{file.deletions}
              </span>
            </div>
          ))}
        </div>
      ) : null}
      {project ? (
        <div className="mt-3 grid gap-2 rounded-md bg-muted/70 p-3 text-xs">
          <div className="min-w-0">
            <div className="mb-1 text-[11px] font-semibold text-muted-foreground">
              {project.operation === "add" ? "项目目录" : "绑定到项目"}
            </div>
            {project.name ? (
              <div className="mb-1 font-medium text-foreground">
                {project.name}
              </div>
            ) : null}
            <div className="break-all font-mono text-[11px] leading-relaxed text-foreground">
              {project.rootPath}
            </div>
          </div>
          <div className="rounded border border-border/70 bg-background/70 px-2 py-1.5 text-[11px] leading-relaxed text-muted-foreground">
            {project.immutableAssociation
              ? "绑定后不可切换或解除。如需使用其他项目，请新建会话。"
              : "仅登记已有目录，不会创建文件，也不会自动绑定当前会话。"}
          </div>
        </div>
      ) : null}
      {registration ? (
        <div className="mt-3 grid gap-2 rounded-md bg-muted/70 p-3 text-xs">
          <div className="grid gap-1.5">
            <div className="flex min-w-0 items-center justify-between gap-3">
              <span className="font-semibold text-foreground">
                {registration.name}
              </span>
              <span className="rounded bg-background px-2 py-0.5 font-mono text-[11px] text-muted-foreground">
                {registration.id}
              </span>
            </div>
            <div className="grid gap-1 text-[11px] text-muted-foreground sm:grid-cols-2">
              <span>transport: {registration.transport}</span>
              <span>
                auth: {registration.auth}
                {registration.bearerTokenEnv
                  ? ` (${registration.bearerTokenEnv})`
                  : ""}
              </span>
            </div>
            <div className="break-all rounded border border-border/70 bg-background/70 px-2 py-1.5 font-mono text-[11px] leading-relaxed text-foreground">
              {registration.target}
            </div>
            {registration.cwd ? (
              <div className="break-all text-[11px] text-muted-foreground">
                cwd: {registration.cwd}
              </div>
            ) : null}
            {registration.roots.length > 0 ? (
              <div className="grid gap-1 text-[11px] text-muted-foreground">
                <span className="font-semibold">MCP roots</span>
                {registration.roots.map((root) => (
                  <span className="break-all font-mono" key={root}>
                    {root}
                  </span>
                ))}
              </div>
            ) : null}
          </div>
          <div className="rounded border border-border/70 bg-background/70 px-2 py-1.5 text-[11px] leading-relaxed text-muted-foreground">
            批准后 Host 才会启动或连接该来源并探测能力。成功后它会全局保存，后续会话可按需使用；这不是进程沙箱。
          </div>
        </div>
      ) : null}
      {command ? (
        <div className="mt-3 grid gap-2 rounded-md bg-muted/70 p-3 text-xs">
          <div className="min-w-0">
            <div className="mb-1 text-[11px] font-semibold text-muted-foreground">
              命令
            </div>
            <pre className="max-h-24 overflow-auto whitespace-pre-wrap break-words font-mono text-[11px] leading-relaxed text-foreground">
              {command.command}
            </pre>
          </div>
          <div className="grid gap-1 text-[11px] text-muted-foreground sm:grid-cols-2">
            <span className="min-w-0 truncate">cwd: {command.cwd || "."}</span>
            <span className="min-w-0 truncate">
              risk: {command.riskLevel || "unknown"}
            </span>
            <span className="min-w-0 truncate">
              category: {command.category || "unknown"}
            </span>
            <span className="min-w-0 truncate">
              network: {command.networkHint || "deny"}
            </span>
          </div>
        </div>
      ) : null}
      <div className="mt-3 flex flex-col gap-2 border-t border-border/70 pt-3 sm:flex-row sm:items-center sm:justify-between">
        {registration ? (
          <span className="text-xs text-muted-foreground">
            此注册必须单独确认，不能记住授权
          </span>
        ) : (
          <label className="flex min-w-0 items-center gap-2 text-xs text-muted-foreground">
            <input
              checked={remember}
              className="size-3.5 accent-primary"
              onChange={(event) => setRemember(event.target.checked)}
              type="checkbox"
            />
            <span className="truncate">
              {permissionRememberLabel(permission, command)}
            </span>
          </label>
        )}
        <div className="flex shrink-0 items-center gap-2">
          <Button
            className="h-8 px-3 text-xs"
            disabled={action !== "idle"}
            onClick={deny}
            size="sm"
            type="button"
            variant="outline"
          >
            {action === "denying"
              ? "拒绝中"
              : action === "denied"
                ? "已拒绝"
                : "拒绝"}
          </Button>
          <Button
            className="h-8 gap-1.5 px-3 text-xs"
            disabled={action !== "idle"}
            onClick={approve}
            size="sm"
            type="button"
          >
            {isBusy ? null : <Check />}
            {action === "approving"
              ? "批准中"
              : action === "approved"
                ? "已批准"
                : "批准并继续"}
          </Button>
        </div>
      </div>
    </section>
  );
}
