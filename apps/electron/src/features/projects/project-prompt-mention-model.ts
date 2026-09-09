import {
  isRequiredCoreToolName,
  isStandaloneToolResource,
  type ToolInjectionResourceKind,
} from "./tool-injection-resource-model.ts";
import type { CodexMcpServer, CodexResourceInput, CodexSkill } from "../../codex-composer-resources.ts";

export type PromptMentionRange = { query: string; start: number };

export type PromptMentionReference = {
  input?: CodexResourceInput;
  id: string;
  kind: "conversation" | "project" | ToolInjectionResourceKind;
  rootPath?: string;
  token: string;
  toolNames?: string[];
};

export type PromptMentionItem = {
  detail?: string;
  id: string;
  label: string;
  reference: PromptMentionReference;
  token: string;
  type: "对话" | "项目" | "技能" | "工具" | "扩展" | "MCP";
};

export type PromptMentionAction = {
  action: "select-local" | "unavailable";
  disabled?: boolean;
  detail: string;
  id: string;
  label: string;
};

export type PromptMentionProject = {
  id?: string;
  name?: string;
  rootPath: string;
};

const promptMentionGroupOrder: PromptMentionItem["type"][] = [
  "项目",
  "对话",
  "技能",
  "扩展",
  "MCP",
  "工具",
];

const promptMentionActions: PromptMentionAction[] = [
  {
    action: "select-local",
    detail: "添加到此消息",
    id: "action:select-local",
    label: "选择文件或文件夹",
  },
  {
    action: "unavailable",
    detail: "即将支持",
    disabled: true,
    id: "action:smart-screenshot",
    label: "附加智能快照",
  },
];

export function promptMentionRange(value: string, caret: number): PromptMentionRange | null {
  const beforeCaret = value.slice(0, caret);
  const start = beforeCaret.lastIndexOf("@");
  if (start < 0 || (start > 0 && !/\s/.test(value[start - 1]))) return null;
  const query = beforeCaret.slice(start + 1);
  return /\s/.test(query) ? null : { query, start };
}

function promptMentionMatchRank(names: string[], detail: string | undefined, query: string) {
  const normalizedNames = names.map(name => name.toLocaleLowerCase());
  if (normalizedNames.some(name => name === query)) return 0;
  if (normalizedNames.some(name => name.startsWith(query))) return 1;
  if (normalizedNames.some(name => name.includes(query))) return 2;
  return detail?.toLocaleLowerCase().includes(query) ? 3 : -1;
}

export function filterPromptMentionItems(items: PromptMentionItem[], query: string) {
  const needle = query.trim().toLocaleLowerCase();
  if (!needle) return [...items];
  // Match each field independently; never assemble a keyword from scattered letters.
  return items
    .map(item => ({ item, rank: promptMentionMatchRank([item.label, item.token], item.detail, needle) }))
    .filter(({ rank }) => rank >= 0)
    .sort((a, b) => a.rank - b.rank)
    .map(({ item }) => item);
}

export function filterPromptMentionActions(query: string) {
  const needle = query.trim().toLocaleLowerCase();
  if (!needle) return [...promptMentionActions];
  return promptMentionActions
    .map(item => ({ item, rank: promptMentionMatchRank([item.label], item.detail, needle) }))
    .filter(({ rank }) => rank >= 0)
    .sort((a, b) => a.rank - b.rank)
    .map(({ item }) => item);
}

export const PROMPT_MENTION_TYPE_LIMIT = 10;

export function promptMentionCodexSkillItems(skills: CodexSkill[]): PromptMentionItem[] {
  return skills.filter(skill => skill.enabled).map(skill => ({
    id: `skill:${skill.path}`, label: skill.displayName, token: skill.name,
    detail: skill.description || "已启用技能", type: "技能",
    reference: { id: skill.path, kind: "skill", token: skill.displayName,
      input: { type: "skill", name: skill.name, path: skill.path } },
  }));
}

export function promptMentionCodexMcpItems(servers: CodexMcpServer[]): PromptMentionItem[] {
  return servers.filter(server => server.runtimeStatus !== "disabled").flatMap((server): PromptMentionItem[] => {
    if (server.name === "codex_apps") {
      return (server.apps ?? []).map(app => ({
        id: `app:${app.id}`, label: app.name, token: app.name,
        detail: app.description || `${app.toolNames.length} 个工具`, type: "MCP",
        reference: { id: `app:${app.id}`, kind: "mcp", token: app.name,
          toolNames: app.toolNames,
          input: { type: "mention", name: app.name, path: `app://${app.id}` } },
      }));
    }
    return [{
    id: `mcp:${server.name}`, label: server.displayName, token: server.name,
    detail: server.toolsError ? `工具发现失败：${server.toolsError}` : `${server.toolNames.length} 个工具`, type: "MCP",
    reference: { id: server.name, kind: "mcp", token: server.displayName,
      input: { type: "mention", name: server.name, path: `mcp://${server.name}` } },
    }];
  });
}

export function groupPromptMentionItems(items: PromptMentionItem[]) {
  return promptMentionGroupOrder.flatMap((type) => {
    const groupItems = items.filter((item) => item.type === type).slice(0, PROMPT_MENTION_TYPE_LIMIT);
    return groupItems.length ? [{ items: groupItems, type }] : [];
  });
}

export function isPromptMentionBuiltinTool(tool: {
  activationPolicy?: string;
  category?: string;
  enabled: boolean;
  name?: string;
  source: string;
  sourceId?: string;
  toolsets?: string[];
}) {
  return (
    tool.enabled &&
    !isRequiredCoreToolName(tool.name) &&
    isStandaloneToolResource(tool)
  );
}

export function promptMentionCatalogItems({
  extensions,
  mcpServers,
  skills,
  tools,
}: {
  extensions: Array<{
    enabled: boolean;
    id: string;
    summary: { description?: string; name: string; tools?: string[] };
  }>;
  mcpServers: Array<{
    server: {
      description?: string;
      displayName?: string;
      enabled: boolean;
      id: string;
      name: string;
    };
    tools?: Array<{ name: string }>;
  }>;
  skills: Array<{
    description?: string;
    enabled: boolean;
    id: string;
    name: string;
  }>;
  tools: Array<{
    activationPolicy?: string;
    description?: string;
    enabled: boolean;
    name: string;
    source: string;
    sourceId?: string;
  }>;
}): PromptMentionItem[] {
  const catalogToolNames = new Set(tools.map((tool) => tool.name));
  const extensionItems = extensions
    .filter((extension) => extension.enabled)
    .map((extension) => {
      const toolNames = tools
        .filter((tool) => tool.enabled && tool.sourceId === extension.id)
        .map((tool) => tool.name);
      return {
        detail: extension.summary.description || "已安装扩展",
        id: `extension:${extension.id}`,
        label: extension.summary.name,
        reference: {
          id: extension.id,
          kind: "extension" as const,
          token: extension.summary.name,
          toolNames: toolNames.length ? toolNames : extension.summary.tools,
        },
        token: extension.summary.name,
        type: "扩展" as const,
      };
    });
  const mcpItems = mcpServers
    .filter((item) => item.server.enabled)
    .map((item) => {
      const label = item.server.displayName || item.server.name;
      const registeredToolNames = tools
        .filter((tool) => tool.enabled && tool.sourceId === item.server.id)
        .map((tool) => tool.name);
      const discoveredToolNames = (item.tools ?? [])
        .map((tool) => tool.name)
        .filter((name) => catalogToolNames.has(name));
      const toolNames = registeredToolNames.length
        ? registeredToolNames
        : discoveredToolNames;
      return {
        detail: item.server.description || (toolNames.length ? `${toolNames.length} 个工具` : "MCP 服务"),
        id: `mcp:${item.server.id}`,
        label,
        reference: {
          id: item.server.id,
          kind: "mcp" as const,
          token: label,
          toolNames,
        },
        token: label,
        type: "MCP" as const,
      };
    });
  const skillItems = skills
    .filter((skill) => skill.enabled)
    .map((skill) => ({
      detail: skill.description || "已启用技能",
      id: `skill:${skill.id}`,
      label: skill.name,
      reference: { id: skill.id, kind: "skill" as const, token: skill.name },
      token: skill.name,
      type: "技能" as const,
    }));
  const toolItems = tools
    .filter(isPromptMentionBuiltinTool)
    .map((tool) => ({
      detail: tool.description || "可用工具",
      id: `tool:${tool.name}`,
      label: tool.name,
      reference: {
        id: tool.name,
        kind: "tool" as const,
        token: tool.name,
        toolNames: [tool.name],
      },
      token: tool.name,
      type: "工具" as const,
    }));
  return [...extensionItems, ...skillItems, ...mcpItems, ...toolItems];
}

export function promptMentionConversationItems(
  conversations: Array<{ id: string; projectPath?: string; title?: string }>,
): PromptMentionItem[] {
  return conversations.map((conversation) => {
    const label = conversation.title?.trim() || "新对话";
    return {
      detail: conversation.projectPath || "本地对话",
      id: `conversation:${conversation.id}`,
      label,
      reference: {
        id: conversation.id,
        kind: "conversation",
        rootPath: conversation.projectPath,
        token: label,
      },
      token: label,
      type: "对话",
    };
  });
}

export function promptMentionProjectItems(
  projects: PromptMentionProject[],
  currentProjectPath: string,
): PromptMentionItem[] {
  const currentPath = currentProjectPath.trim();
  const byPath = new Map<string, PromptMentionProject>();
  for (const project of projects) {
    const rootPath = project.rootPath.trim();
    if (rootPath && !byPath.has(rootPath)) byPath.set(rootPath, project);
  }
  return [...byPath.values()]
    .filter((project) => Boolean(project.id?.trim()))
    .toSorted((left, right) =>
      Number(right.rootPath === currentPath) - Number(left.rootPath === currentPath),
    )
    .map((project) => {
      const label = project.name?.trim() || projectNameFromPath(project.rootPath);
      const current = project.rootPath === currentPath;
      return {
        detail: current ? `${project.rootPath} · 当前项目` : project.rootPath,
        id: `project:${project.id || project.rootPath}`,
        label,
        reference: {
          id: project.id!,
          kind: "project",
          rootPath: project.rootPath,
          token: label,
        },
        token: label,
        type: "项目",
      };
    });
}

export function activePromptMentionReferences(
  references: PromptMentionReference[],
) {
  const seen = new Set<string>();
  return references.filter((reference) => {
    const key = `${reference.kind}\u0000${reference.id}`;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

export function addPromptMentionReference(
  references: PromptMentionReference[],
  reference: PromptMentionReference,
) {
  const withoutDuplicate = references.filter((item) =>
    item.kind !== reference.kind || item.id !== reference.id
  );
  if (reference.kind !== "project") {
    return [...withoutDuplicate, reference];
  }
  return [
    ...withoutDuplicate.filter((item) => item.kind !== "project"),
    reference,
  ];
}

export function removePromptMentionReference(
  references: PromptMentionReference[],
  reference: PromptMentionReference,
) {
  return references.filter((item) =>
    item.kind !== reference.kind || item.id !== reference.id
  );
}

function projectNameFromPath(rootPath: string) {
  const parts = rootPath.trim().replace(/[\\/]+$/, "").split(/[\\/]/).filter(Boolean);
  return parts.at(-1) || "Project";
}

export function consumePromptMentionQuery(
  value: string,
  caret: number,
  range: PromptMentionRange,
) {
  const prefix = value.slice(0, range.start);
  const suffix = value.slice(caret);
  if (!prefix) {
    const nextSuffix = suffix.replace(/^\s/, "");
    return { caret: 0, value: nextSuffix };
  }
  if (/\s$/.test(prefix) && /^\s/.test(suffix)) {
    return { caret: prefix.length, value: `${prefix}${suffix.slice(1)}` };
  }
  return { caret: prefix.length, value: `${prefix}${suffix}` };
}

export function promptComposerEnterAction(
  mentionOpen: boolean,
  shiftKey: boolean,
  isComposing: boolean,
) {
  if (isComposing) return "none" as const;
  if (shiftKey) return "newline" as const;
  return mentionOpen ? "mention" as const : "submit" as const;
}
