import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";
import {
  isApplicationPlugin,
  type PluginSettingsSection,
} from "@/features/projects/plugin-mcp-settings-model";
import type {
  MCPServerListItem,
  PluginListItem,
  ToolCatalogEntry,
} from "@/services/aivo";

type ToolActivationSourceMetadata = {
  description?: string;
  label: string;
  section: Exclude<PluginSettingsSection, "skills" | "tools">;
};

export type ToolCatalogGroup = {
  description?: string;
  id: string;
  label: string;
  section: Exclude<PluginSettingsSection, "skills">;
  tools: ToolCatalogEntry[];
};

export function normalizeToolNames(names: string[]) {
  return [...new Set(names.map((name) => name.trim()).filter(Boolean))].toSorted();
}

export function usedToolNamesFromTurns(turns: ConversationTurn[]) {
  return normalizeToolNames(
    turns.flatMap((turn) => turn.toolCalls.map((toolCall) => toolCall.name || "")),
  );
}

export function isToggleableCatalogTool(tool: ToolCatalogEntry) {
  if (!tool.enabled || isBridgeCatalogTool(tool)) return false;
  if (tool.source === "plugin" || tool.source === "mcp") return true;
  if (
    ["mcp", "plugin", "agent", "automation", "admin"].includes(
      tool.category ?? "",
    )
  ) {
    return true;
  }
  return (tool.toolsets ?? []).some(
    (toolset) =>
      toolset === "mcp" ||
      toolset === "plugin" ||
      toolset === "admin" ||
      toolset.startsWith("mcp:") ||
      toolset.startsWith("plugin:"),
  );
}

export function groupToolCatalogEntries(
  tools: ToolCatalogEntry[],
  sourceMetadata: Record<string, ToolActivationSourceMetadata>,
) {
  const groups = new Map<string, ToolCatalogGroup>();
  for (const tool of tools) {
    const baseSection = toolActivationSection(tool);
    const metadata = tool.sourceId
      ? sourceMetadata[
          `${baseSection === "mcp" ? "mcp" : "plugin"}:${tool.sourceId}`
        ]
      : undefined;
    const section = toolActivationSection(tool, metadata);
    const id = `${section}:${tool.source}:${tool.sourceId || tool.namespace || tool.category || tool.name}`;
    const label =
      metadata?.label ||
      tool.namespace ||
      tool.sourceId ||
      toolCategoryLabel(tool);
    const group = groups.get(id) ?? {
      description: metadata?.description,
      id,
      label,
      section,
      tools: [],
    };
    group.tools.push(tool);
    groups.set(id, group);
  }
  return [...groups.values()];
}

export function toolActivationSourceMetadata(
  plugins: PluginListItem[],
  servers: MCPServerListItem[],
) {
  const metadata: Record<string, ToolActivationSourceMetadata> = {};
  for (const item of plugins) {
    metadata[`plugin:${item.plugin.id}`] = {
      description: item.plugin.manifest.description,
      label:
        item.plugin.manifest.displayName ||
        item.plugin.manifest.name ||
        item.plugin.id,
      section: isApplicationPlugin(item) ? "apps" : "plugins",
    };
  }
  for (const item of servers) {
    const server = item.server;
    const value = {
      description: server.description,
      label: server.displayName || server.name || server.id,
      section: "mcp" as const,
    };
    for (const key of [
      server.id,
      `mcp.${sanitizeMcpSourceId(server.id)}`,
      server.name,
      `mcp.${sanitizeMcpSourceId(server.name)}`,
    ]) {
      if (key) metadata[`mcp:${key}`] = value;
    }
  }
  return metadata;
}

export function isToolCatalogGroupActive(
  group: ToolCatalogGroup,
  activeToolSet: Set<string>,
) {
  return group.tools.every((tool) => activeToolSet.has(tool.name));
}

export function isToolCatalogGroupPartiallyActive(
  group: ToolCatalogGroup,
  activeToolSet: Set<string>,
) {
  return (
    !isToolCatalogGroupActive(group, activeToolSet) &&
    group.tools.some((tool) => activeToolSet.has(tool.name))
  );
}

export function isToolCatalogGroupUsed(
  group: ToolCatalogGroup,
  usedToolSet: Set<string>,
) {
  return group.tools.some((tool) => usedToolSet.has(tool.name));
}

export function toolActivationSwitchId(groupId: string) {
  return `tool-activation-${encodeURIComponent(groupId)}`;
}

export function toolNameListsEqual(left: string[], right: string[]) {
  const normalizedLeft = normalizeToolNames(left);
  const normalizedRight = normalizeToolNames(right);
  return (
    normalizedLeft.length === normalizedRight.length &&
    normalizedLeft.every((name, index) => name === normalizedRight[index])
  );
}

function isBridgeCatalogTool(tool: ToolCatalogEntry) {
  return [
    "tool_resolve",
    "tool_search",
    "tool_list",
    "tool_detail",
    "tool_call",
  ].includes(tool.name);
}

function toolCategoryLabel(tool: ToolCatalogEntry) {
  if (tool.source === "mcp") return "MCP";
  if (tool.source === "plugin") return "插件";
  if (tool.category) return tool.category;
  return "工具";
}

function toolActivationSection(
  tool: ToolCatalogEntry,
  metadata?: ToolActivationSourceMetadata,
): Exclude<PluginSettingsSection, "skills"> {
  if (
    tool.source === "mcp" ||
    tool.category === "mcp" ||
    tool.sourceId?.startsWith("mcp.") ||
    (tool.toolsets ?? []).some(
      (toolset) => toolset === "mcp" || toolset.startsWith("mcp:"),
    )
  ) {
    return "mcp";
  }
  if (metadata) return metadata.section;
  if (
    tool.source === "plugin" ||
    tool.category === "plugin" ||
    (tool.toolsets ?? []).some(
      (toolset) => toolset === "plugin" || toolset.startsWith("plugin:"),
    )
  ) {
    return "plugins";
  }
  return "tools";
}

function sanitizeMcpSourceId(value: string) {
  return (
    value
      .trim()
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, "_")
      .replace(/^_+|_+$/g, "") || "server"
  );
}
