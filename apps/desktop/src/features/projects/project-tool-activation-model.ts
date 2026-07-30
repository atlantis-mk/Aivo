import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";
import type { PluginListItem, ToolCatalogEntry } from "@/services/aivo";

export type ToolCatalogGroup = {
  id: string;
  label: string;
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
  pluginLabels: Record<string, string>,
) {
  const groups = new Map<string, ToolCatalogGroup>();
  for (const tool of tools) {
    const id = `${tool.source}:${tool.sourceId || tool.namespace || tool.category || "other"}`;
    const label =
      (tool.sourceId && pluginLabels[tool.sourceId]) ||
      tool.namespace ||
      tool.sourceId ||
      toolCategoryLabel(tool);
    const group = groups.get(id) ?? { id, label, tools: [] };
    group.tools.push(tool);
    groups.set(id, group);
  }
  return [...groups.values()];
}

export function pluginLabelsById(items: PluginListItem[]) {
  return Object.fromEntries(
    items.map((item) => [
      item.plugin.id,
      item.plugin.manifest.displayName ||
        item.plugin.manifest.name ||
        item.plugin.id,
    ]),
  );
}

export function toolActivationSwitchId(toolName: string) {
  return `tool-activation-${encodeURIComponent(toolName)}`;
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
