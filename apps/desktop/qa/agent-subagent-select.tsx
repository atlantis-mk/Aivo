import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import { NewPromptDialog } from "../src/routes/prompts";
import type { AgentModeDefinition } from "../src/services/aivo";
import "../src/index.css";

const agentModes: AgentModeDefinition[] = [
  {
    id: "assistant",
    displayName: "Assistant",
    description: "默认主 Agent",
    prompt: "Complete the user's task.",
    mode: "primary",
  },
  {
    id: "research",
    displayName: "Research",
    description: "收集并整理可靠资料",
    prompt: "Research one bounded question.",
    mode: "subagent",
  },
  {
    id: "review",
    displayName: "Review",
    description: "审查实现并报告问题",
    prompt: "Review one bounded change.",
    mode: "subagent",
  },
  {
    id: "test",
    displayName: "Test",
    description: "运行验证并汇总失败",
    prompt: "Verify one bounded change.",
    mode: "all",
  },
];

window.aivo = {
  invoke: async () => undefined,
} as typeof window.aivo;

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <main className="min-h-screen bg-muted/30 p-6 text-foreground">
      <div className="mx-auto flex max-w-4xl items-center justify-between rounded-xl border bg-background p-4 shadow-sm">
        <div>
          <h1 className="text-lg font-semibold">提示词管理</h1>
          <p className="text-sm text-muted-foreground">Agent 创建与关联子 Agent 注入</p>
        </div>
        <NewPromptDialog agentModes={agentModes} onCreated={() => undefined} />
      </div>
    </main>
  </StrictMode>,
);
