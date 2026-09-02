import type {
  AgentModeDefinition,
  ExtensionInstall,
  MCPServerListItem,
  SkillEntry,
  SkillImportCandidate,
  ToolCatalogEntry,
} from "@/services/aivo";
import type { ExtensionSettingsSection } from "@/features/projects/extension-settings-types";
import {
  isRequiredCoreToolName,
  isStandaloneToolResource,
} from "./tool-injection-resource-model.ts";

export function filterExtensions(items: ExtensionInstall[], query: string) {
  const normalized = normalizeSearch(query);
  if (!normalized) return items;
  return items.filter((item) =>
    matchesSearch(
      [
        item.id,
        item.summary.name,
        item.summary.description,
        item.summary.runtimeType,
        item.installMode,
        item.rootPath,
        item.status,
        ...(item.summary.permissions ?? []),
        ...(item.summary.tools ?? []),
        ...(item.summary.views ?? []),
      ],
      normalized,
    ),
  );
}

export function addButtonLabel(section: ExtensionSettingsSection) {
  if (section === "extensions") {
    return "安装本地扩展";
  }
  if (section === "skills") {
    return "扫描技能";
  }
  if (section === "mcp" || section === "tools") {
    return "添加 MCP 工具";
  }
  return "添加 MCP 工具";
}

export function filterAgentModes(items: AgentModeDefinition[], query: string) {
  const normalized = normalizeSearch(query);
  if (!normalized) return items;
  return items.filter((mode) =>
    matchesSearch(
      [
        mode.id,
        mode.displayName,
        mode.description,
        mode.prompt,
        mode.mode,
        mode.permissionScope,
        mode.source,
        mode.model?.providerId,
        mode.model?.modelId,
        ...(mode.subagents ?? []),
      ],
      normalized,
    ),
  );
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
  return items.filter(
    (tool) =>
      isAivoBuiltinTool(tool) &&
      !isRequiredCoreToolName(tool.name) &&
      (!normalized ||
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
        )),
  );
}

export function isAivoBuiltinTool(tool: ToolCatalogEntry) {
  return isStandaloneToolResource(tool);
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

export function selectVisibleSkillCandidates(
  candidates: SkillImportCandidate[],
  skills: SkillEntry[],
) {
  const installedNames = new Set(
    skills.map((skill) => normalizeSearch(skill.name)).filter(Boolean),
  );
  const selectedNames = new Set<string>();

  return candidates.filter((candidate) => {
    const name = normalizeSearch(candidate.name);
    if (!name || installedNames.has(name) || selectedNames.has(name)) {
      return false;
    }
    selectedNames.add(name);
    return true;
  });
}

export function normalizeSearch(value?: string) {
  return value?.trim().toLowerCase() ?? "";
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
