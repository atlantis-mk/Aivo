import { useCallback, useEffect, useState } from "react";

import {
  listMCPServers,
  listPlugins,
  listSkills,
  listToolCatalog,
  type MCPServerListItem,
  type PluginListItem,
  type SkillEntry,
  type SkillImportCandidate,
  type ToolCatalogEntry,
} from "@/services/aivo";

export function usePluginMcpSettingsCatalogState({
  active,
  workspaceRoot,
}: {
  active: boolean;
  workspaceRoot?: string;
}) {
  const [plugins, setPlugins] = useState<PluginListItem[]>([]);
  const [servers, setServers] = useState<MCPServerListItem[]>([]);
  const [tools, setTools] = useState<ToolCatalogEntry[]>([]);
  const [skills, setSkills] = useState<SkillEntry[]>([]);
  const [skillCandidates, setSkillCandidates] = useState<
    SkillImportCandidate[]
  >([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const reload = useCallback(async () => {
    setLoading(true);
    setError("");
    const results = await Promise.allSettled([
      listPlugins(true),
      listMCPServers(true, true),
      listToolCatalog(workspaceRoot ?? ""),
      listSkills({
        includeCandidates: true,
        includeDisabled: true,
        includeIgnored: true,
      }),
    ] as const);

    if (results[0].status === "fulfilled") {
      setPlugins(results[0].value);
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

    const failures = results
      .map((result, index) => {
        if (result.status === "fulfilled") return "";
        const label =
          index === 0
            ? "Plugins"
            : index === 1
              ? "MCP"
              : index === 2
                ? "Tools"
                : "Skills";
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
    error,
    loading,
    plugins,
    reload,
    servers,
    setError,
    setLoading,
    skills,
    skillCandidates,
    tools,
  };
}
