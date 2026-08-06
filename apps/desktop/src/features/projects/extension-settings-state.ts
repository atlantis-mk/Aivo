import { useEffect, useMemo, useState } from "react";

import {
  normalizeMcpDraft,
  type ExtensionSettingsSection,
} from "@/features/projects/extension-settings-model";
import { useExtensionSettingsCatalogState } from "@/features/projects/extension-settings-catalog-state";
import { useExtensionSettingsDerivedState } from "@/features/projects/extension-settings-derived-state";
import {
  deleteManagedSkill,
  ignoreSkillCandidatesByName,
  importSkill,
  probeMCPServer,
  saveMCPServer,
  setSkillEnabled,
  getSessionActiveTools,
  setSessionActiveTools,
  type MCPServerConfig,
  type SkillEntry,
  type SkillImportCandidate,
} from "@/services/aivo";

export function useExtensionSettingsState({
  active,
  sessionId,
  workspaceRoot,
}: {
  active: boolean;
  sessionId?: string;
  workspaceRoot?: string;
}) {
  const catalog = useExtensionSettingsCatalogState({ active, workspaceRoot });
  const setCatalogError = catalog.setError;
  const [query, setQuery] = useState("");
  const [section, setSection] = useState<ExtensionSettingsSection>("extensions");
  const [addOpen, setAddOpen] = useState(false);
  const [extensionInstallOpen, setExtensionInstallOpen] = useState(false);
  const [activeToolNames, setActiveToolNames] = useState<string[]>([]);
  const activeToolSet = useMemo(
    () => new Set(activeToolNames),
    [activeToolNames],
  );
  const {
    visibleExtensions,
    visibleAllTools,
    visibleServers,
    visibleSkillCandidates,
    visibleSkills,
    visibleTools,
  } = useExtensionSettingsDerivedState({
    extensions: catalog.extensions,
    query,
    servers: catalog.servers,
    skillCandidates: catalog.skillCandidates,
    skills: catalog.skills,
    tools: catalog.tools,
    workspaceRoot,
  });

  useEffect(() => {
    if (!active || !sessionId) {
      setActiveToolNames([]);
      return;
    }
    let cancelled = false;
    void getSessionActiveTools(sessionId)
      .then((result) => {
        if (!cancelled) {
          setActiveToolNames([
            ...new Set([...result.toolNames, ...result.coreToolNames]),
          ]);
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setCatalogError(err instanceof Error ? err.message : String(err));
        }
      });
    return () => {
      cancelled = true;
    };
  }, [active, sessionId, setCatalogError]);

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

  async function toggleTool(toolName: string, enabled: boolean) {
    if (!sessionId) return;
    const next = new Set(activeToolNames);
    if (enabled) {
      next.add(toolName);
    } else {
      next.delete(toolName);
    }
    const toolNames = [...next].toSorted();
    setActiveToolNames(toolNames);
    try {
      const saved = await setSessionActiveTools(sessionId, toolNames);
      setActiveToolNames([
        ...new Set([...saved.toolNames, ...saved.coreToolNames]),
      ]);
    } catch (err) {
      catalog.setError(err instanceof Error ? err.message : String(err));
      const current = await getSessionActiveTools(sessionId).catch(() => null);
      if (current) {
        setActiveToolNames([
          ...new Set([...current.toolNames, ...current.coreToolNames]),
        ]);
      }
    }
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
    addMcpServer,
    addMcpServers,
    addOpen,
    activeToolSet,
    deleteSkill,
    error: catalog.error,
    extensionInstallOpen,
    extensions: catalog.extensions,
    ignoreSkillCandidate,
    importSkillCandidate,
    loading: catalog.loading,
    openAddDialog,
    query,
    reload: catalog.reload,
    section,
    servers: catalog.servers,
    setAddOpen,
    setExtensionInstallOpen,
    setQuery,
    setSection,
    skills: catalog.skills,
    toggleSkillEnabled,
    toggleTool,
    visibleAllTools,
    visibleExtensions,
    visibleServers,
    visibleSkillCandidates,
    visibleSkills,
    visibleTools,
  };
}
