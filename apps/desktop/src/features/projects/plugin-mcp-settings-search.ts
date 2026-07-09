import type {
  MCPServerListItem,
  PluginListItem,
  SkillEntry,
  SkillImportCandidate,
  ToolCatalogEntry,
} from "@/services/aivo";
import type {
  AddToolMode,
  PluginSettingsSection,
} from "@/features/projects/plugin-mcp-settings-types";

export function addToolModeForSection(
  section: PluginSettingsSection,
): AddToolMode {
  if (section === "mcp" || section === "tools") {
    return "mcp";
  }
  return "plugin";
}

export function addButtonLabel(section: PluginSettingsSection) {
  if (section === "skills") {
    return "扫描技能";
  }
  if (section === "mcp" || section === "tools") {
    return "添加 MCP 工具";
  }
  return "添加插件工具";
}

export function filterPlugins(items: PluginListItem[], query: string) {
  const normalized = normalizeSearch(query);
  if (!normalized) {
    return items;
  }
  return items.filter((item) => {
    const plugin = item.plugin;
    return matchesSearch(
      [
        plugin.id,
        plugin.manifest.name,
        plugin.manifest.displayName,
        plugin.manifest.description,
        plugin.rootPath,
        ...(plugin.manifest.keywords ?? []),
      ],
      normalized,
    );
  });
}

export function filterServers(items: MCPServerListItem[], query: string) {
  const normalized = normalizeSearch(query);
  if (!normalized) {
    return items;
  }
  return items.filter((item) => {
    const server = item.server;
    return matchesSearch(
      [
        server.id,
        server.name,
        server.displayName,
        server.description,
        server.command,
        server.url,
        server.transport,
      ],
      normalized,
    );
  });
}

export function filterTools(items: ToolCatalogEntry[], query: string) {
  const normalized = normalizeSearch(query);
  if (!normalized) {
    return items;
  }
  return items.filter((tool) =>
    matchesSearch(
      [
        tool.name,
        tool.description,
        tool.namespace,
        tool.capability,
        tool.category,
        tool.source,
        ...(tool.toolsets ?? []),
      ],
      normalized,
    ),
  );
}

export function filterSkills(items: SkillEntry[], query: string) {
  const normalized = normalizeSearch(query);
  if (!normalized) {
    return items;
  }
  return items.filter((skill) =>
    matchesSearch(
      [
        skill.name,
        skill.description,
        skill.scope,
        skill.source,
        skill.rootPath,
        skill.skillPath,
      ],
      normalized,
    ),
  );
}

export function filterSkillCandidates(
  items: SkillImportCandidate[],
  query: string,
) {
  const normalized = normalizeSearch(query);
  if (!normalized) {
    return items;
  }
  return items.filter((candidate) =>
    matchesSearch(
      [
        candidate.name,
        candidate.description,
        candidate.scope,
        candidate.source,
        candidate.status,
        candidate.rootPath,
        candidate.skillPath,
      ],
      normalized,
    ),
  );
}

export function normalizeSearch(value?: string) {
  return value?.trim().toLowerCase() ?? "";
}

export function pluginToolsForDisplay(
  item: PluginListItem,
): ToolCatalogEntry[] {
  if (item.tools?.length) {
    return item.tools;
  }
  return (item.plugin.manifest.tools ?? []).map((tool) => ({
    name: tool.name,
    description: tool.description,
    inputSchema: tool.inputSchema,
    namespace: item.plugin.manifest.name || item.plugin.id,
    capability: tool.capability,
    riskLevel: tool.riskLevel,
    category: tool.category,
    toolsets: tool.toolsets,
    source: "plugin",
    sourceId: item.plugin.id,
    enabled: item.plugin.enabled,
  }));
}

export function mergeToolCatalogEntries(groups: ToolCatalogEntry[][]) {
  const merged = new Map<string, ToolCatalogEntry>();
  for (const group of groups) {
    for (const tool of group) {
      const key = [
        tool.source,
        tool.sourceId ?? "",
        tool.registrationId ?? "",
        tool.name,
      ].join(":");
      if (!merged.has(key)) {
        merged.set(key, tool);
      }
    }
  }
  return Array.from(merged.values());
}

function matchesSearch(values: Array<string | undefined>, query: string) {
  return values.some((value) => normalizeSearch(value).includes(query));
}
