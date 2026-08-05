import { useMemo } from "react";

import {
  filterPlugins,
  filterServers,
  filterSkillCandidates,
  filterSkills,
  filterTools,
  isApplicationPlugin,
  mergeToolCatalogEntries,
  pluginToolsForDisplay,
  type PluginSettingsSection,
} from "@/features/projects/plugin-mcp-settings-model";
import type {
  MCPServerListItem,
  PluginListItem,
  SkillEntry,
  SkillImportCandidate,
  ToolCatalogEntry,
} from "@/services/aivo";

export function usePluginMcpSettingsDerivedState({
  plugins,
  query,
  section,
  servers,
  skillCandidates,
  skills,
  tools,
  workspaceRoot,
}: {
  plugins: PluginListItem[];
  query: string;
  section: PluginSettingsSection;
  servers: MCPServerListItem[];
  skillCandidates: SkillImportCandidate[];
  skills: SkillEntry[];
  tools: ToolCatalogEntry[];
  workspaceRoot?: string;
}) {
  const visibleTools = useMemo(
    () =>
      workspaceRoot || tools.length > 0
        ? tools
        : mergeToolCatalogEntries([
            plugins.flatMap(pluginToolsForDisplay),
            servers.flatMap((item) =>
              (item.tools ?? []).map((tool) => ({
                name: tool.name,
                description: tool.description,
                inputSchema: tool.inputSchema,
                namespace: item.server.name || item.server.id,
                capability: tool.capability,
                riskLevel: tool.riskLevel,
                source: "mcp",
                sourceId: item.server.id,
                registrationId: tool.id,
                enabled: item.server.enabled,
              })),
            ),
          ]),
    [plugins, servers, tools, workspaceRoot],
  );
  const applicationPlugins = useMemo(
    () => plugins.filter(isApplicationPlugin),
    [plugins],
  );
  const visibleSkills = useMemo(
    () => filterSkills(skills, query),
    [query, skills],
  );
  const visibleSkillCandidates = useMemo(
    () => filterSkillCandidates(skillCandidates, query),
    [query, skillCandidates],
  );
  const visiblePlugins = useMemo(
    () =>
      filterPlugins(section === "apps" ? applicationPlugins : plugins, query),
    [applicationPlugins, plugins, query, section],
  );
  const visibleServers = useMemo(
    () => filterServers(servers, query),
    [query, servers],
  );
  const visibleAllTools = useMemo(
    () => filterTools(visibleTools, query),
    [query, visibleTools],
  );

  return {
    applicationPlugins,
    visibleAllTools,
    visiblePlugins,
    visibleServers,
    visibleSkillCandidates,
    visibleSkills,
    visibleTools,
  };
}
