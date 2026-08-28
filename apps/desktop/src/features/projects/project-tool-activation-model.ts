import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";
import type { ExtensionSettingsSection } from "@/features/projects/extension-settings-model";
import type {
  ExtensionInstall,
  MCPServerListItem,
  ToolCatalogEntry,
} from "@/services/aivo";
import {
  isRequiredCoreToolName,
  isStandaloneToolResource,
  requiredCoreToolNames,
} from "./tool-injection-resource-model.ts";

type ToolActivationSourceMetadata = {
  description?: string;
  label: string;
  section: Exclude<ExtensionSettingsSection, "skills" | "tools">;
};

export type ToolCatalogGroup = {
  description?: string;
  grouped: boolean;
  id: string;
  label: string;
  section: Exclude<ExtensionSettingsSection, "skills">;
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
  return (
    tool.enabled &&
    !isRequiredCoreToolName(tool.name) &&
    tool.activationPolicy !== "provider_declaration" &&
    !isBridgeCatalogTool(tool)
  );
}

export function defaultActiveBuiltinToolNames(_tools: ToolCatalogEntry[]) {
  return [...requiredCoreToolNames];
}

export function groupToolCatalogEntries(
  tools: ToolCatalogEntry[],
  sourceMetadata: Record<string, ToolActivationSourceMetadata>,
) {
  const groups = new Map<string, ToolCatalogGroup>();
  for (const tool of tools) {
    const metadataSection =
      tool.source === "mcp" || tool.category === "mcp" ? "mcp" : "extensions";
    const metadata = tool.sourceId
      ? sourceMetadata[`${metadataSection}:${tool.sourceId}`]
      : undefined;
    const section = toolActivationSection(tool, metadata);
    const selectionGroup = tool.selectionGroup;
    const grouped = Boolean(selectionGroup);
    const id = grouped
      ? `${section}:group:${selectionGroup!.id}`
      : `${section}:tool:${tool.name}`;
    const label = grouped
      ? selectionGroup!.name || metadata?.label || selectionGroup!.id
      : tool.name;
    const group = groups.get(id) ?? {
      description: grouped
        ? selectionGroup?.description || metadata?.description
        : tool.description || tool.capability,
      grouped,
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
  extensions: ExtensionInstall[],
  servers: MCPServerListItem[],
) {
  const metadata: Record<string, ToolActivationSourceMetadata> = {};
  for (const item of extensions) {
    metadata[`extensions:${item.id}`] = {
      description: item.summary.description,
      label: item.summary.name || item.id,
      section: "extensions",
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
      `mcp_${sanitizeMcpSourceId(server.id)}`,
      server.name,
      `mcp.${sanitizeMcpSourceId(server.name)}`,
      `mcp_${sanitizeMcpSourceId(server.name)}`,
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

function toolActivationSection(
  tool: ToolCatalogEntry,
  metadata?: ToolActivationSourceMetadata,
): Exclude<ExtensionSettingsSection, "skills"> {
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
  if (isStandaloneToolResource(tool)) return "tools";
  if (metadata) return metadata.section;
  if (
    tool.source === "extension" ||
    tool.category === "extension" ||
    (tool.toolsets ?? []).some(
      (toolset) =>
        toolset === "extension" || toolset.startsWith("extension:"),
    )
  ) {
    return "extensions";
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
