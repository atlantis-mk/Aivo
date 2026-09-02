import { useMemo, useState } from "react";

import {
  normalizeMcpDraft,
  type ExtensionSettingsSection,
} from "@/features/projects/extension-settings-model";
import { useExtensionSettingsCatalogState } from "@/features/projects/extension-settings-catalog-state";
import { useExtensionSettingsDerivedState } from "@/features/projects/extension-settings-derived-state";
import {
  SKILL_ACTION_DELETE,
  SKILL_ACTION_SET_ENABLED,
  skillSupportsAction,
} from "@/features/projects/skill-action-model";
import {
  deleteAgentMode,
  deleteManagedSkill,
  ignoreSkillCandidatesByName,
  importSkill,
  probeMCPServer,
  saveAgentMode,
  saveMCPServer,
  setSkillEnabled,
  setGlobalToolEnabled,
  type AgentModeDefinition,
  type MCPServerConfig,
  type SkillEntry,
  type SkillImportCandidate,
} from "@/services/aivo";

export function useExtensionSettingsState({
  active,
  workspaceRoot,
}: {
  active: boolean;
  workspaceRoot?: string;
}) {
  const catalog = useExtensionSettingsCatalogState({ active, workspaceRoot });
  const [query, setQuery] = useState("");
  const [section, setSection] = useState<ExtensionSettingsSection>("extensions");
  const [agentModeEditorOpen, setAgentModeEditorOpen] = useState(false);
  const [editingAgentMode, setEditingAgentMode] = useState<
    AgentModeDefinition | undefined
  >();
  const [addOpen, setAddOpen] = useState(false);
  const [extensionInstallOpen, setExtensionInstallOpen] = useState(false);
  const {
    visibleAgentModes,
    visibleExtensions,
    visibleAllTools,
    visibleServers,
    visibleSkillCandidates,
    visibleSkills,
    visibleTools,
  } = useExtensionSettingsDerivedState({
    agentModes: catalog.agentModes,
    extensions: catalog.extensions,
    query,
    servers: catalog.servers,
    skillCandidates: catalog.skillCandidates,
    skills: catalog.skills,
    tools: catalog.tools,
    workspaceRoot,
  });
  const activeToolSet = useMemo(
    () =>
      new Set(
        visibleTools.filter((tool) => tool.enabled).map((tool) => tool.name),
      ),
    [visibleTools],
  );

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
    if (!skillSupportsAction(skill, SKILL_ACTION_SET_ENABLED)) return;
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
    if (!skillSupportsAction(skill, SKILL_ACTION_DELETE)) return;
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

  async function toggleTools(toolNames: string[], enabled: boolean) {
    catalog.setLoading(true);
    catalog.setError("");
    try {
      for (const toolName of toolNames) {
        await setGlobalToolEnabled(toolName, enabled, workspaceRoot ?? "");
      }
      await catalog.reload();
    } catch (err) {
      catalog.setError(err instanceof Error ? err.message : String(err));
      await catalog.reload();
    } finally {
      catalog.setLoading(false);
    }
  }

  async function saveManagedAgentMode(definition: AgentModeDefinition) {
    catalog.setLoading(true);
    catalog.setError("");
    try {
      await saveAgentMode(definition);
      await catalog.reload();
      setAgentModeEditorOpen(false);
      setEditingAgentMode(undefined);
    } catch (err) {
      catalog.setError(err instanceof Error ? err.message : String(err));
      throw err;
    } finally {
      catalog.setLoading(false);
    }
  }

  async function deleteManagedAgentMode(mode: AgentModeDefinition) {
    catalog.setLoading(true);
    catalog.setError("");
    try {
      await deleteAgentMode(mode.id);
      await catalog.reload();
      setAgentModeEditorOpen(false);
      setEditingAgentMode(undefined);
    } catch (err) {
      catalog.setError(err instanceof Error ? err.message : String(err));
      throw err;
    } finally {
      catalog.setLoading(false);
    }
  }

  function editAgentMode(mode?: AgentModeDefinition) {
    setEditingAgentMode(mode);
    setAgentModeEditorOpen(true);
  }

  function openAddDialog() {
    if (section === "extensions") {
      setExtensionInstallOpen(true);
      return;
    }
    if (section === "skills") {
      void catalog.reload();
      return;
    }
    setAddOpen(true);
  }

  return {
    agentModeEditorOpen,
    agentModes: catalog.agentModes,
    addMcpServer,
    addMcpServers,
    addOpen,
    activeToolSet,
    deleteManagedAgentMode,
    deleteSkill,
    editAgentMode,
    editingAgentMode,
    error: catalog.error,
    extensionInstallOpen,
    extensions: catalog.extensions,
    ignoreSkillCandidate,
    importSkillCandidate,
    loading: catalog.loading,
    openAddDialog,
    providerCatalog: catalog.providerCatalog,
    query,
    reload: catalog.reload,
    saveManagedAgentMode,
    section,
    servers: catalog.servers,
    setAddOpen,
    setAgentModeEditorOpen,
    setExtensionInstallOpen,
    setQuery,
    setSection,
    skills: catalog.skills,
    toggleSkillEnabled,
    toggleTools,
    visibleAgentModes,
    visibleAllTools,
    visibleExtensions,
    visibleServers,
    visibleSkillCandidates,
    visibleSkills,
    visibleTools,
  };
}
