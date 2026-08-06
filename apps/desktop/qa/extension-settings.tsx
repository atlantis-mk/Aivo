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
