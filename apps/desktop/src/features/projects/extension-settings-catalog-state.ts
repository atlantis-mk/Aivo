import { useCallback, useEffect, useState } from "react";

import type { CatalogState } from "@/lib/provider-catalog";
import {
  getProviderCatalog,
  getProviderCatalogForProject,
  listAgentModes,
  listExtensionInstalls,
  listMCPServers,
  listSkills,
  listToolCatalog,
  type AgentModeDefinition,
  type ExtensionInstall,
  type MCPServerListItem,
  type SkillEntry,
  type SkillImportCandidate,
  type ToolCatalogEntry,
} from "@/services/aivo";

export function useExtensionSettingsCatalogState({
  active,
  workspaceRoot,
}: {
  active: boolean;
  workspaceRoot?: string;
}) {
  const [extensions, setExtensions] = useState<ExtensionInstall[]>([]);
  const [agentModes, setAgentModes] = useState<AgentModeDefinition[]>([]);
  const [servers, setServers] = useState<MCPServerListItem[]>([]);
  const [tools, setTools] = useState<ToolCatalogEntry[]>([]);
  const [skills, setSkills] = useState<SkillEntry[]>([]);
  const [skillCandidates, setSkillCandidates] = useState<
    SkillImportCandidate[]
  >([]);
  const [providerCatalog, setProviderCatalog] = useState<CatalogState | null>(
    null,
  );
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const reload = useCallback(async () => {
    setLoading(true);
    setError("");
    setProviderCatalog(null);
    const results = await Promise.allSettled([
      listExtensionInstalls(),
      listMCPServers(true, true),
      listToolCatalog(workspaceRoot ?? ""),
      listSkills({
        includeCandidates: true,
        includeDisabled: true,
        includeIgnored: true,
      }),
      listAgentModes(false),
      workspaceRoot
        ? getProviderCatalogForProject(workspaceRoot)
        : getProviderCatalog(),
    ] as const);

    if (results[0].status === "fulfilled") {
      setExtensions(results[0].value);
    }
    if (results[1].status === "fulfilled") {
      setServers(results[1].value);
    }
    if (results[2].status === "fulfilled") {
      setTools(results[2].value);
    }
    if (results[3].status === "fulfilled") {
      setSkills(results[3].value.entries ?? []);
      setSkillCandidates(results[3].value.candidates ?? []);
    }
    if (results[4].status === "fulfilled") {
      setAgentModes(results[4].value);
    }
    if (results[5].status === "fulfilled") {
      setProviderCatalog(results[5].value);
    }

    const failures = results
      .map((result, index) => {
        if (result.status === "fulfilled") return "";
        const label = [
          "Extensions",
          "MCP",
          "Tools",
          "Skills",
          "Agent modes",
          "Provider catalog",
        ][index];
        const reason = result.reason;
        return `${label}: ${reason instanceof Error ? reason.message : String(reason)}`;
      })
      .filter(Boolean);

    setError(failures.join("\n"));
    setLoading(false);
  }, [workspaceRoot]);

  useEffect(() => {
    if (!active) return;
    void reload();
  }, [active, reload]);

  return {
    agentModes,
    error,
    extensions,
    loading,
    providerCatalog,
    reload,
    servers,
    setError,
    setLoading,
    skills,
    skillCandidates,
    tools,
  };
}
