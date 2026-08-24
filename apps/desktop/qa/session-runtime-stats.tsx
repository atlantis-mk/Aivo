import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import { ProjectSessionRuntimeStatsLine } from "../src/features/projects/project-session-runtime-stats-line";
import "../src/index.css";

const stats =
  "1 轮 · 1 步 | LLM 1.1s | 首 token 平均 0.7s · 130 tok/s | 缓存命中 0% | 输入 9.4K tok · 输出 50 tok";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <main className="relative min-h-screen overflow-hidden bg-background text-foreground">
      <div className="mx-auto flex min-h-screen w-full max-w-5xl flex-col px-6 pb-10 pt-8">
        <div className="border-b border-border/60 pb-5 text-sm font-medium">
          当前会话
        </div>
        <div className="flex flex-1 flex-col justify-center gap-5 pb-44">
          <div className="ml-auto max-w-sm rounded-2xl bg-muted px-4 py-2 text-sm">
            hi
          </div>
          <div className="max-w-xl text-sm leading-7">
            你好！今天想一起完成什么？
          </div>
        </div>
      </div>
      <div className="absolute bottom-6 left-1/2 w-[calc(100%-2rem)] max-w-[680px] -translate-x-1/2 sm:w-[calc(100%-48px)]">
        <div className="min-w-0 rounded-3xl border border-border bg-card px-5 pb-3 pt-4 shadow-lg shadow-foreground/5">
          <div className="min-h-10 text-sm text-muted-foreground">
            给智能体发消息
          </div>
          <div className="flex items-center justify-between pt-1 text-xs text-muted-foreground">
            <span>工作区写入</span>
            <span>GPT-5.6</span>
          </div>
        </div>
        <ProjectSessionRuntimeStatsLine value={stats} />
      </div>
    </main>
  </StrictMode>,
);
