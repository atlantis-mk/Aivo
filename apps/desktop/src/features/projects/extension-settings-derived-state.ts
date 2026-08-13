import { useMemo } from "react";

import {
  filterAgentModes,
  filterExtensions,
  filterServers,
  filterSkillCandidates,
  filterSkills,
  filterTools,
  isAivoBuiltinTool,
  mergeToolCatalogEntries,
} from "@/features/projects/extension-settings-model";
import type {
  AgentModeDefinition,
  ExtensionInstall,
  MCPServerListItem,
  SkillEntry,
  SkillImportCandidate,
  ToolCatalogEntry,
} from "@/services/aivo";

export function useExtensionSettingsDerivedState({
  agentModes,
  extensions,
  query,
  servers,
  skillCandidates,
  skills,
  tools,
  workspaceRoot,
}: {
  agentModes: AgentModeDefinition[];
  extensions: ExtensionInstall[];
  query: string;
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
    [servers, tools, workspaceRoot],
  );
  const visibleAgentModes = useMemo(
    () => filterAgentModes(agentModes, query),
    [agentModes, query],
  );
  const visibleExtensions = useMemo(
    () => filterExtensions(extensions, query),
    [extensions, query],
  );
  const visibleSkills = useMemo(
    () => filterSkills(skills, query),
    [query, skills],
  );
  const visibleSkillCandidates = useMemo(
    () => filterSkillCandidates(skillCandidates, query),
    [query, skillCandidates],
  );
  const visibleServers = useMemo(
    () => filterServers(servers, query),
    [query, servers],
  );
  const manageableTools = useMemo(
    () => visibleTools.filter(isAivoBuiltinTool),
    [visibleTools],
  );
  const visibleAllTools = useMemo(
    () => filterTools(manageableTools, query),
    [manageableTools, query],
  );

  return {
    visibleAgentModes,
    visibleExtensions,
    visibleAllTools,
    visibleServers,
    visibleSkillCandidates,
    visibleSkills,
    visibleTools: manageableTools,
  };
}
