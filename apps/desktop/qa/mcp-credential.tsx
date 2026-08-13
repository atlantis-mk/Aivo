import { StrictMode, useState } from "react";
import { createRoot } from "react-dom/client";

import { McpServerDraftForm } from "../src/features/projects/extension-settings-mcp-form";
import type { KeyValueRow } from "../src/features/projects/extension-settings-types";
import type { MCPServerConfig } from "../src/services/aivo";
import "../src/index.css";

function McpCredentialQA() {
  const [draft, setDraft] = useState<MCPServerConfig>({
    id: "aiblog",
    name: "aiblog",
    displayName: "AIBlog",
    description: "查询并管理 AIBlog 中的文章、草稿与发布状态",
    transport: "streamable_http",
    url: "https://example.com/mcp",
    authType: "bearer",
    bearerAuthMode: "direct",
    enabled: true,
  });
  const [args, setArgs] = useState<string[]>([""]);
  const [env, setEnv] = useState<KeyValueRow[]>([{ key: "", value: "" }]);
  const [headers, setHeaders] = useState<KeyValueRow[]>([{ key: "", value: "" }]);
  const [roots, setRoots] = useState<string[]>([""]);

  return (
    <main className="min-h-screen bg-muted/30 p-4 text-foreground sm:p-8">
      <section className="mx-auto grid max-w-2xl gap-4 rounded-xl border bg-background p-4 shadow-sm sm:p-6">
        <header className="grid gap-1">
          <h1 className="text-lg font-semibold">添加 MCP server</h1>
          <p className="text-sm text-muted-foreground">
            直接配置 Bearer Token，并由本地 Core 安全保存。
          </p>
        </header>
        <McpServerDraftForm
          argRows={args}
          draft={draft}
          envRows={env}
          headerRows={headers}
          onArgRowsChange={setArgs}
          onDraftChange={setDraft}
          onEnvRowsChange={setEnv}
          onHeaderRowsChange={setHeaders}
          onRootRowsChange={setRoots}
          rootRows={roots}
          showEnabledToggle
          transportEditable
        />
      </section>
    </main>
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <McpCredentialQA />
  </StrictMode>,
);
