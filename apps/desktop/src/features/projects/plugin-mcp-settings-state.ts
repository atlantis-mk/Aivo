import { useCallback, useState } from "react";

import {
  addToolModeForSection,
  normalizeMcpDraft,
  type AddToolMode,
  type PluginSettingsSection,
} from "@/features/projects/plugin-mcp-settings-model";
import { usePluginMcpSettingsCatalogState } from "@/features/projects/plugin-mcp-settings-catalog-state";
import { usePluginMcpSettingsDerivedState } from "@/features/projects/plugin-mcp-settings-derived-state";
import {
  deleteManagedSkill,
  ignoreSkillCandidatesByName,
  importSkill,
  installPluginFromPath,
  probeMCPServer,
  reloadPlugins,
  saveMCPServer,
  setSkillEnabled,
  type MCPServerConfig,
  type SkillEntry,
  type SkillImportCandidate,
} from "@/services/aivo";

export function usePluginMcpSettingsState({
  active,
  workspaceRoot,
}: {
  active: boolean;
  workspaceRoot?: string;
}) {
  const catalog = usePluginMcpSettingsCatalogState({ active, workspaceRoot });
  const [pluginPath, setPluginPath] = useState("");
  const [query, setQuery] = useState("");
  const [section, setSection] = useState<PluginSettingsSection>("plugins");
  const [addOpen, setAddOpen] = useState(false);
  const [addMode, setAddMode] = useState<AddToolMode>("plugin");
  const {
    applicationPlugins,
    visibleAllTools,
    visiblePlugins,
    visibleServers,
    visibleSkillCandidates,
    visibleSkills,
    visibleTools,
  } = usePluginMcpSettingsDerivedState({
    plugins: catalog.plugins,
    query,
    section,
    servers: catalog.servers,
    skillCandidates: catalog.skillCandidates,
    skills: catalog.skills,
    tools: catalog.tools,
    workspaceRoot,
  });

  const reloadPluginCatalog = useCallback(() => {
    void reloadPlugins().then(catalog.reload);
  }, [catalog.reload]);

  async function installPluginPath(path: string) {
    if (!path.trim()) return;
    catalog.setLoading(true);
    catalog.setError("");
    try {
      await installPluginFromPath(path.trim(), true);
      await catalog.reload();
    } catch (err) {
      catalog.setError(err instanceof Error ? err.message : String(err));
      throw err;
    } finally {
      catalog.setLoading(false);
    }
  }

  async function installPlugin() {
    if (!pluginPath.trim()) return;
    await installPluginPath(pluginPath);
    setPluginPath("");
  }

  async function addMcpServer(server: MCPServerConfig) {
    await addMcpServers([server], true);
  }

  async function addMcpServers(
    serversToAdd: MCPServerConfig[],
    failOnProbeError = false,
  ) {
    catalog.setLoading(true);
    catalog.setError("");
    const failures: string[] = [];
    let savedCount = 0;
    try {
      for (const server of serversToAdd) {
        const normalized = normalizeMcpDraft(server);
        const label =
          normalized.displayName || normalized.name || normalized.id || "MCP server";
        try {
          const saved = await saveMCPServer(normalized);
          savedCount += 1;
          if (normalized.enabled) {
            try {
              await probeMCPServer(saved.id || normalized.id);
            } catch (err) {
              const message = err instanceof Error ? err.message : String(err);
              failures.push(`${label}: ${message}`);
              if (failOnProbeError) {
                throw err;
              }
            }
          }
        } catch (err) {
          const message = err instanceof Error ? err.message : String(err);
          failures.push(`${label}: ${message}`);
          if (failOnProbeError) {
            throw err;
          }
        }
      }
      await catalog.reload();
      if (failures.length > 0) {
        const message = `部分 MCP 已保存，但存在问题：\n${failures.join("\n")}`;
        catalog.setError(message);
        if (savedCount === 0) {
          throw new Error(message);
        }
      }
    } catch (err) {
      catalog.setError(err instanceof Error ? err.message : String(err));
      throw err;
    } finally {
      catalog.setLoading(false);
    }
  }

  async function importSkillCandidate(candidate: SkillImportCandidate) {
    catalog.setLoading(true);
    catalog.setError("");
    try {
      await importSkill(candidate.id, "global");
      await catalog.reload();
    } catch (err) {
      catalog.setError(err instanceof Error ? err.message : String(err));
    } finally {
      catalog.setLoading(false);
    }
  }

  async function ignoreSkillCandidate(candidate: SkillImportCandidate) {
    catalog.setLoading(true);
    catalog.setError("");
    try {
      await ignoreSkillCandidatesByName(candidate.name);
      await catalog.reload();
    } catch (err) {
      catalog.setError(err instanceof Error ? err.message : String(err));
    } finally {
      catalog.setLoading(false);
    }
  }

  async function toggleSkillEnabled(skill: SkillEntry, enabled: boolean) {
    catalog.setLoading(true);
    catalog.setError("");
    try {
      await setSkillEnabled(skill.id, enabled);
      await catalog.reload();
    } catch (err) {
      catalog.setError(err instanceof Error ? err.message : String(err));
    } finally {
      catalog.setLoading(false);
    }
  }

  async function deleteSkill(skill: SkillEntry) {
    catalog.setLoading(true);
    catalog.setError("");
    try {
      await deleteManagedSkill(skill.id);
      await catalog.reload();
    } catch (err) {
      catalog.setError(err instanceof Error ? err.message : String(err));
    } finally {
      catalog.setLoading(false);
    }
  }

  function openAddDialog() {
    if (section === "skills") {
      void catalog.reload();
      return;
    }
    setAddMode(addToolModeForSection(section));
    setAddOpen(true);
  }

  return {
    addMcpServer,
    addMcpServers,
    addMode,
    addOpen,
    applicationPlugins,
    deleteSkill,
    error: catalog.error,
    ignoreSkillCandidate,
    importSkillCandidate,
    installPlugin,
    installPluginPath,
    loading: catalog.loading,
    openAddDialog,
    pluginPath,
    plugins: catalog.plugins,
    query,
    reload: catalog.reload,
    reloadPluginCatalog,
    section,
    servers: catalog.servers,
    setAddMode,
    setAddOpen,
    setPluginPath,
    setQuery,
    setSection,
    skills: catalog.skills,
    toggleSkillEnabled,
    visibleAllTools,
    visiblePlugins,
    visibleServers,
    visibleSkillCandidates,
    visibleSkills,
    visibleTools,
  };
}
