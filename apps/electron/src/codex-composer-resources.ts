export type CodexResourceInput = {
  type: "skill" | "mention";
  name: string;
  path: string;
};
export type CodexSkill = {
  name: string;
  displayName: string;
  description: string;
  path: string;
  enabled: boolean;
};
export type CodexSkillCatalog = { skills: CodexSkill[]; errors: string[] };
export type CodexMcpApp = {
  id: string;
  name: string;
  description: string;
  toolNames: string[];
};
export type CodexMcpServer = {
  name: string;
  displayName: string;
  toolNames: string[];
  apps?: CodexMcpApp[];
  runtimeStatus?: string;
  toolsError?: string;
};
type Request = (
  method: string,
  params: Record<string, unknown>,
) => Promise<unknown>;

function record(value: unknown): Record<string, unknown> | undefined {
  return value !== null && typeof value === "object" && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : undefined;
}
function string(value: unknown) {
  return typeof value === "string" ? value : "";
}

function appsFromTools(tools: Record<string, unknown>): CodexMcpApp[] {
  const apps = new Map<string, CodexMcpApp>();
  for (const [toolName, value] of Object.entries(tools)) {
    const meta = record(record(value)?._meta);
    const id = string(meta?.connector_id);
    // A namespace is not necessarily a connector ID. Only use native metadata.
    if (!id) continue;
    let app = apps.get(id);
    if (!app) {
      app = {
        id,
        name: string(meta?.connector_name) || id,
        description: string(meta?.connector_description),
        toolNames: [],
      };
      apps.set(id, app);
    }
    app.toolNames.push(toolName);
  }
  return [...apps.values()].sort(
    (a, b) => a.name.localeCompare(b.name) || a.id.localeCompare(b.id),
  );
}

export async function loadCodexSkills(
  request: Request,
  workspaceRoot?: string,
  forceReload = false,
): Promise<CodexSkillCatalog> {
  const result = record(
    await request("skills/list", {
      cwds: workspaceRoot ? [workspaceRoot] : [],
      forceReload,
    }),
  );
  if (!result || !Array.isArray(result.data))
    throw new Error("技能接口返回了无效的数据");
  const skills = new Map<string, CodexSkill>();
  const errors: string[] = [];
  for (const value of result.data) {
    const entry = record(value);
    if (!entry || !Array.isArray(entry.skills))
      throw new Error("技能列表格式不正确");
    for (const error of Array.isArray(entry.errors) ? entry.errors : []) {
      const issue = record(error);
      errors.push(
        [string(issue?.path), string(issue?.message)]
          .filter(Boolean)
          .join(": ") || "技能扫描失败",
      );
    }
    for (const value of entry.skills) {
      const skill = record(value);
      const path = string(skill?.path),
        name = string(skill?.name);
      if (!skill || !name || !path) {
        errors.push("发现缺少名称或路径的技能");
        continue;
      }
      const ui = record(skill.interface);
      skills.set(path, {
        name,
        path,
        enabled: skill.enabled !== false,
        displayName: string(ui?.displayName) || name,
        description:
          string(ui?.shortDescription) ||
          string(skill.shortDescription) ||
          string(skill.description),
      });
    }
  }
  return { skills: [...skills.values()], errors };
}

export async function loadCodexMcpServers(
  request: Request,
): Promise<CodexMcpServer[]> {
  const servers = new Map<string, CodexMcpServer>();
  const cursors = new Set<string>();
  let cursor: string | undefined;
  do {
    const result = record(
      await request("mcpServerStatus/list", {
        limit: 100,
        detail: "toolsAndAuthOnly",
        ...(cursor ? { cursor } : {}),
      }),
    );
    if (!result || !Array.isArray(result.data))
      throw new Error("MCP 接口返回了无效的数据");
    for (const value of result.data) {
      const server = record(value),
        name = string(server?.name);
      if (!server || !name) throw new Error("MCP 服务缺少名称");
      const info = record(server.serverInfo);
      const tools = record(server.tools) ?? {};
      servers.set(name, {
        name,
        displayName: string(info?.title) || name,
        toolNames: Object.keys(tools),
        ...(name === "codex_apps" ? { apps: appsFromTools(tools) } : {}),
        runtimeStatus: string(server.runtimeStatus) || undefined,
        toolsError: string(server.toolsError) || undefined,
      });
    }
    cursor = string(result.nextCursor) || undefined;
    if (cursor && cursors.has(cursor))
      throw new Error("MCP 分页游标重复，无法读取完整列表");
    if (cursor) cursors.add(cursor);
  } while (cursor);
  return [...servers.values()];
}

export function codexResourceInputs(
  references: Array<{ input?: CodexResourceInput }>,
): CodexResourceInput[] {
  const inputs = new Map<string, CodexResourceInput>();
  for (const { input } of references) {
    if (!input) continue;
    if (
      !input.name ||
      !input.path ||
      !["skill", "mention"].includes(input.type)
    )
      throw new Error("引用的名称或路径无效");
    inputs.set(`${input.type}:${input.path}`, {
      type: input.type,
      name: input.name,
      path: input.path,
    });
  }
  return [...inputs.values()];
}
