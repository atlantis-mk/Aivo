import type {
  CodexMcpServer,
  CodexSkillCatalog,
} from "@/codex-composer-resources";

export async function listCodexSkills(
  workspaceRoot?: string,
  forceReload = false,
): Promise<CodexSkillCatalog> {
  if (!window.aivoDesktop?.codex?.listSkills)
    throw new Error("技能接口未就绪，请重启 Aivo 后重试");
  return window.aivoDesktop.codex.listSkills(workspaceRoot, forceReload);
}
export async function listCodexMcpServers(): Promise<CodexMcpServer[]> {
  if (!window.aivoDesktop?.codex?.listMcpServers)
    throw new Error("MCP 接口未就绪，请重启 Aivo 后重试");
  return window.aivoDesktop.codex.listMcpServers();
}
