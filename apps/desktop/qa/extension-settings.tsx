import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import { ExtensionSettingsContent } from "../src/features/projects/extension-settings-dialog";
import "../src/index.css";

const extension = {
  id: "com.example.workspace-tools",
  installMode: "managed",
  rootPath: "/Users/example/.aivo/extensions/com.example.workspace-tools/sha256-qa",
  manifestPath: "/Users/example/.aivo/extensions/com.example.workspace-tools/sha256-qa/aivo.extension.json",
  integrity: "sha256:qa",
  enabled: true,
  status: "ready",
  summary: {
    id: "com.example.workspace-tools",
    name: "Workspace Tools",
    version: "2.1.0",
    description: "为当前工作区提供受控工具、上下文和详情视图。",
    runtimeType: "service",
    permissions: ["runtime.messaging"],
    tools: ["workspace_search", "workspace_summary"],
    views: ["tool-detail"],
  },
};

window.aivo = {
  invoke: async (method: string) => {
    switch (method) {
      case "ListExtensionInstalls":
        return [extension];
      case "ListMCPServers":
        return [];
      case "ListToolCatalog":
        return [
          {
            name: "workspace_search",
            description: "Search the current workspace",
            source: "extension",
            sourceId: extension.id,
            category: "extension",
            toolsets: ["coding", "extension"],
            enabled: true,
          },
        ];
      case "ListSkills":
        return { entries: [], candidates: [] };
      case "ListAgentModes":
        return [
          {
            id: "ask",
            displayName: "Ask",
            description: "分析、解释并回答问题，不直接修改工作区。",
            prompt: "Provide clear, evidence-backed answers without modifying the workspace.",
            mode: "primary",
            permissionScope: "read_only",
            source: "builtin",
            builtIn: true,
            overridden: false,
          },
          {
            id: "agent",
            displayName: "Agent",
            description: "自主完成编码任务，并在必要时使用工具。",
            prompt: "Implement the requested change and verify it.",
            mode: "all",
            permissionScope: "workspace_approval",
            subagents: ["research", "specialist-1"],
            source: "builtin",
            builtIn: true,
            overridden: true,
          },
          {
            id: "research",
            displayName: "Research",
            description: "收集多来源证据并生成带引用的研究结论。",
            prompt: "Research the topic using reliable sources and cite the evidence.",
            mode: "all",
            permissionScope: "read_only",
            source: "user",
            builtIn: false,
            overridden: false,
          },
          ...Array.from({ length: 12 }, (_, index) => ({
            id: `specialist-${index + 1}`,
            displayName: `Specialist ${index + 1}`,
            description: `处理第 ${index + 1} 类独立任务，用于验证较长的子 Agent 关联列表。`,
            prompt: `Complete specialist task ${index + 1}.`,
            mode: "subagent" as const,
            permissionScope: "read_only",
            source: "user",
            builtIn: false,
            overridden: false,
          })),
        ];
      case "GetProviderCatalogForProject":
        return {
          connected: ["openai", "anthropic"],
          connectedProviders: [],
          defaultModel: { providerId: "openai", modelId: "gpt-5.5" },
          models: [
            { id: "gpt-5.5", name: "GPT-5.5", providerId: "openai" },
            { id: "gpt-5-mini", name: "GPT-5 mini", providerId: "openai" },
            {
              id: "claude-opus-4-6",
              name: "Claude Opus 4.6",
              providerId: "anthropic",
            },
          ],
          providers: [
            {
              authMethods: [],
              builtIn: true,
              connected: true,
              custom: false,
              id: "openai",
              models: [
                { id: "gpt-5.5", name: "GPT-5.5", providerId: "openai" },
                { id: "gpt-5-mini", name: "GPT-5 mini", providerId: "openai" },
              ],
              name: "OpenAI",
              type: "openai",
            },
            {
              authMethods: [],
              builtIn: true,
              connected: true,
              custom: false,
              id: "anthropic",
              models: [
                {
                  id: "claude-opus-4-6",
                  name: "Claude Opus 4.6",
                  providerId: "anthropic",
                },
              ],
              name: "Anthropic",
              type: "anthropic",
            },
          ],
        };
      default:
        throw new Error(`Unexpected QA method: ${method}`);
    }
  },
  openPath: async () => undefined,
} as unknown as typeof window.aivo;

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <main className="h-screen bg-background text-foreground">
      <ExtensionSettingsContent active surface="page" workspaceRoot="/Users/example/Aivo" />
    </main>
  </StrictMode>,
);
