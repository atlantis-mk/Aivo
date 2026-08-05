import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import { ConversationToolInspector } from "../src/features/projects/conversation-tool-inspector";
import type { ToolCallActivity } from "../src/features/projects/conversation-timeline-tool-model";
import "../src/index.css";

const activity: ToolCallActivity = {
  id: "qa-project-conflict",
  groups: [
    {
      id: "qa-project-conflict-group",
      kind: "write",
      title: "项目关联",
      description: "关联 Aivo Agent Projects",
      calls: [
        {
          id: "qa-project-conflict-call",
          sessionId: "qa-session",
          turnId: "qa-turn",
          name: "aivo_projects_associate",
          arguments: {
            projectId: "53b92311-98d9-4115-b937-d715b4264000",
            rootPath: "/Users/example/Documents/Aivo-Agent-Projects",
          },
          status: "failed",
          resultSummary:
            "project_already_bound：当前会话已绑定到另一个项目，不能切换或解除。请新建会话后再关联其他项目。",
          result: {
            ok: false,
            toolError: {
              code: "project_already_bound",
              message:
                "当前会话已绑定到另一个项目，不能切换或解除。请新建会话后再关联其他项目。",
            },
          },
          error:
            "project_already_bound：当前会话已绑定到另一个项目，不能切换或解除。",
          timeCreated: "2026-08-04T00:00:00Z",
          timeUpdated: "2026-08-04T00:00:00Z",
        },
      ],
    },
  ],
};

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <main className="flex h-screen overflow-hidden bg-background text-foreground">
      <div className="min-w-0 flex-1 px-8 py-10">
        <div className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
          Aivo · Project association conflict QA
        </div>
        <h1 className="mt-3 text-2xl font-semibold">会话已绑定到其他项目</h1>
        <p className="mt-2 max-w-xl text-sm leading-relaxed text-muted-foreground">
          冲突不会创建权限确认请求，也不会修改会话状态；工具结果继续使用通用活动检查器展示。
        </p>
      </div>
      <ConversationToolInspector activity={activity} onClose={() => undefined} />
    </main>
  </StrictMode>,
);
