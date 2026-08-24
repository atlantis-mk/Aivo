import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import { PermissionApprovalDock } from "../src/features/projects/project-permission-approval-dock";
import type { PermissionRequest } from "../src/services/aivo";
import "../src/index.css";

window.aivo = {
  invoke: async () => ({ status: "approved" }),
} as typeof window.aivo;

const permissions: PermissionRequest[] = [
  {
    id: "qa-add",
    sessionId: "qa-session",
    toolName: "aivo_projects_add",
    action: "write",
    paths: ["project:add:opaque"],
    arguments: {
      agentMode: "code",
      toolsets: ["coding", "extension"],
      projectOperation: "add",
      projectRoot:
        "/Users/example/Documents/客户项目/2026/一个用于验证窄屏长路径折行行为的项目目录/Aivo-Agent-Projects",
      rememberScope: "exact_project",
    },
    status: "pending",
    timeCreated: "2026-08-04T00:00:00Z",
    timeUpdated: "2026-08-04T00:00:00Z",
  },
  {
    id: "qa-associate",
    sessionId: "qa-session",
    toolName: "aivo_projects_associate",
    action: "write",
    paths: ["project:associate:opaque"],
    arguments: {
      agentMode: "code",
      toolsets: ["coding", "extension"],
      projectOperation: "associate",
      projectName: "Aivo Agent Projects",
      projectRoot: "/Users/example/Documents/Aivo-Agent-Projects",
      immutableAssociation: true,
      rememberScope: "exact_project",
    },
    status: "pending",
    timeCreated: "2026-08-04T00:00:00Z",
    timeUpdated: "2026-08-04T00:00:00Z",
  },
];

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <main className="relative min-h-screen overflow-hidden bg-background text-foreground">
      <div className="mx-auto max-w-5xl px-6 pt-12">
        <div className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
          Aivo · Project permission QA
        </div>
        <h1 className="mt-3 text-2xl font-semibold">项目工具权限确认</h1>
        <p className="mt-2 max-w-xl text-sm leading-relaxed text-muted-foreground">
          验证添加与不可变会话绑定的路径展示、长文本折行、精确记忆范围和批准/拒绝反馈。
        </p>
      </div>
      <PermissionApprovalDock dockPinnedSummary={false} permissions={permissions} />
    </main>
  </StrictMode>,
);
