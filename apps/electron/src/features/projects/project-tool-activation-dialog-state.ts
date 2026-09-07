import { useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";

import {
  defaultActiveBuiltinToolNames,
  groupSkillCatalogEntries,
  groupToolCatalogEntries,
  isToolCatalogGroupActive,
  isToolCatalogGroupUsed,
  isToggleableCatalogTool,
  normalizeToolNames,
  toolActivationSourceMetadata,
  toolNameListsEqual,
} from "@/features/projects/project-tool-activation-model";
import { scopeToolActivationSave } from "@/features/projects/project-tool-activation-scope";
import { skillCanActivate } from "@/features/projects/skill-action-model";
import {
  getSessionActiveSkills,
  getSessionActiveTools,
  ignoreSkillCandidatesByName,
  importSkill,
  listExtensionInstalls,
  listMCPServers,
  listSkills,
  listToolCatalog,
  loadSkillIntoSession,
  setSessionActiveSkills,
  setSessionActiveTools,
  type MCPServerListItem,
  type SkillEntry,
  type SkillImportCandidate,
  type ToolCatalogEntry,
} from "@/services/aivo";

export type ToolActivationDialogProps = {
  activeSessionId: string;
  pendingActiveToolNames: string[];
  onPendingActiveToolNamesChange: (
    updater: string[] | ((current: string[]) => string[]),
  ) => void;
  onOpenChange: (open: boolean) => void;
  open: boolean;
  usedToolNames: string[];
  workspaceRoot: string;
};

export function useToolActivationDialogState({
  activeSessionId,
  pendingActiveToolNames,
  onPendingActiveToolNamesChange,
  onOpenChange,
  open,
  usedToolNames,
  workspaceRoot,
}: ToolActivationDialogProps) {
  const [activeToolNames, setActiveToolNames] = useState<string[]>([]);
  const [savedToolNames, setSavedToolNames] = useState<string[]>([]);
  const [automaticToolNames, setAutomaticToolNames] = useState<string[]>([]);
  const [activeSkillIds, setActiveSkillIds] = useState<string[]>([]);
  const [savedSkillIds, setSavedSkillIds] = useState<string[]>([]);
  const [visibleSkillIds, setVisibleSkillIds] = useState<string[]>([]);
  const [tools, setTools] = useState<ToolCatalogEntry[]>([]);
  const [skills, setSkills] = useState<SkillEntry[]>([]);
  const [skillCandidates, setSkillCandidates] = useState<
    SkillImportCandidate[]
  >([]);
  const [sourceMetadata, setSourceMetadata] = useState(
    () => toolActivationSourceMetadata([], []),
  );
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const pendingActiveToolNamesRef = useRef(pendingActiveToolNames);
  const activeToolSet = useMemo(
    () => new Set([...activeToolNames, ...automaticToolNames]),
    [activeToolNames, automaticToolNames],
  );
  const usedToolSet = useMemo(() => new Set(usedToolNames), [usedToolNames]);
  const toggleableTools = useMemo(
    () => tools.filter(isToggleableCatalogTool),
    [tools],
  );
  const groupedTools = useMemo(
    () => groupToolCatalogEntries(toggleableTools, sourceMetadata),
    [sourceMetadata, toggleableTools],
  );
  const groupedSkills = useMemo(
    () => groupSkillCatalogEntries(skills),
    [skills],
  );
  const activeGroupCount = groupedTools.filter((group) =>
    isToolCatalogGroupActive(group, activeToolSet),
  ).length;
  const usedGroupCount = groupedTools.filter((group) =>
    isToolCatalogGroupUsed(group, usedToolSet),
  ).length;
  const inactiveGroupCount = Math.max(
    0,
    groupedTools.length - activeGroupCount,
  );
  const activeSkillSet = useMemo(
    () => new Set([...activeSkillIds, ...visibleSkillIds]),
    [activeSkillIds, visibleSkillIds],
  );
  const hasDraftChanges =
    !toolNameListsEqual(activeToolNames, savedToolNames) ||
    !toolNameListsEqual(activeSkillIds, savedSkillIds);

  useEffect(() => {
    pendingActiveToolNamesRef.current = pendingActiveToolNames;
  }, [pendingActiveToolNames]);

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    setLoading(true);
    void Promise.all([
      listToolCatalog(workspaceRoot),
      listSkills({
        includeCandidates: true,
        includeDisabled: true,
        includeIgnored: true,
      }),
      activeSessionId
        ? getSessionActiveTools(activeSessionId).catch(() => ({
            sessionId: activeSessionId,
            toolNames: [],
            coreToolNames: [],
            automaticToolNames: [],
          }))
        : Promise.resolve({
            sessionId: "",
            toolNames: pendingActiveToolNamesRef.current,
            coreToolNames: [],
            automaticToolNames: [],
          }),
      activeSessionId
        ? getSessionActiveSkills(activeSessionId).catch(() => ({
            sessionId: activeSessionId,
            skillIds: [],
            skills: [],
            visibleSkillIds: [],
          }))
        : Promise.resolve({
            sessionId: "",
            skillIds: [],
            skills: [],
            visibleSkillIds: [],
          }),
      listExtensionInstalls().catch(() => []),
      listMCPServers(true, false).catch(() => [] as MCPServerListItem[]),
    ])
      .then(
        ([
          catalogTools,
          skillList,
          activeTools,
          activeSkills,
          extensions,
          servers,
        ]) => {
          if (cancelled) return;
          const defaultCoreTools = defaultActiveBuiltinToolNames(catalogTools);
          const normalizedActiveTools = normalizeToolNames([
            ...activeTools.toolNames,
            ...(activeSessionId
              ? activeTools.coreToolNames
              : defaultCoreTools),
          ]);
          const normalizedActiveSkills = normalizeToolNames(
            activeSkills.skillIds,
          );
          const normalizedAutomaticTools = normalizeToolNames(
            activeTools.automaticToolNames ?? [],
          );
          const normalizedVisibleSkills = normalizeToolNames(
            activeSkills.visibleSkillIds ?? [],
          );
          setTools(catalogTools);
          setSkills((skillList.entries ?? []).filter(skillCanActivate));
          setSkillCandidates(skillList.candidates ?? []);
          setActiveToolNames(normalizedActiveTools);
          setSavedToolNames(normalizedActiveTools);
          setAutomaticToolNames(normalizedAutomaticTools);
          setActiveSkillIds(normalizedActiveSkills);
          setSavedSkillIds(normalizedActiveSkills);
          setVisibleSkillIds(normalizedVisibleSkills);
          setSourceMetadata(toolActivationSourceMetadata(extensions, servers));
        },
      )
      .catch((err) => {
        if (!cancelled) {
          toast.error(err instanceof Error ? err.message : "加载工具失败");
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [activeSessionId, open, workspaceRoot]);

  async function submitActiveToolNames() {
    const scope = scopeToolActivationSave(activeSessionId, activeToolNames);
    setSaving(true);
    try {
      if (scope.kind === "pending") {
        setSavedToolNames(scope.toolNames);
        setSavedSkillIds(normalizeToolNames(activeSkillIds));
        onPendingActiveToolNamesChange(scope.toolNames);
        onOpenChange(false);
        return;
      }
      const saved = await setSessionActiveTools(
        scope.sessionId,
        scope.toolNames,
      );
      const savedSkills = await setSessionActiveSkills(
        scope.sessionId,
        normalizeToolNames(activeSkillIds),
      );
      const savedNames = normalizeToolNames(saved.toolNames);
      const savedSkillNames = normalizeToolNames(savedSkills.skillIds);
      setActiveToolNames(savedNames);
      setSavedToolNames(savedNames);
      setAutomaticToolNames(normalizeToolNames(saved.automaticToolNames ?? []));
      setActiveSkillIds(savedSkillNames);
      setSavedSkillIds(savedSkillNames);
      setVisibleSkillIds(
        normalizeToolNames(savedSkills.visibleSkillIds ?? []),
      );
      onOpenChange(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "更新工具失败");
    } finally {
      setSaving(false);
    }
  }

  function toggleToolGroup(names: string[], enabled: boolean) {
    const next = new Set(activeToolNames);
    for (const name of names) {
      if (enabled) {
        next.add(name);
      } else {
        next.delete(name);
      }
    }
    setActiveToolNames(normalizeToolNames([...next]));
  }

  function toggleSkill(ids: string[], enabled: boolean) {
    const next = new Set(activeSkillIds);
    for (const id of ids) {
      if (enabled) {
        next.add(id);
      } else {
        next.delete(id);
      }
    }
    setActiveSkillIds(normalizeToolNames([...next]));
  }

  async function loadSkill(skill: SkillEntry, reload = false) {
    if (!activeSessionId) return;
    setSaving(true);
    try {
      await loadSkillIntoSession({
        sessionId: activeSessionId,
        skillId: skill.id,
        reload,
      });
      const active = await getSessionActiveSkills(activeSessionId);
      setActiveSkillIds(normalizeToolNames(active.skillIds));
      setSavedSkillIds(normalizeToolNames(active.skillIds));
      setVisibleSkillIds(normalizeToolNames(active.visibleSkillIds ?? []));
      toast.success(`已加载技能 ${skill.name}`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "加载技能失败");
    } finally {
      setSaving(false);
    }
  }

  async function importCandidate(candidate: SkillImportCandidate) {
    setSaving(true);
    try {
      await importSkill(candidate.id, candidate.scope || "project");
      const list = await listSkills({
        includeCandidates: true,
        includeDisabled: true,
        includeIgnored: true,
      });
      setSkills((list.entries ?? []).filter(skillCanActivate));
      setSkillCandidates(list.candidates ?? []);
      toast.success(`已导入技能 ${candidate.name}`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "导入技能失败");
    } finally {
      setSaving(false);
    }
  }

  async function ignoreCandidate(candidate: SkillImportCandidate) {
    setSaving(true);
    try {
      await ignoreSkillCandidatesByName(candidate.name);
      const list = await listSkills({
        includeCandidates: true,
        includeDisabled: true,
        includeIgnored: true,
      });
      setSkills((list.entries ?? []).filter(skillCanActivate));
      setSkillCandidates(list.candidates ?? []);
      toast.success(`已忽略技能 ${candidate.name}`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "忽略技能失败");
    } finally {
      setSaving(false);
    }
  }

  return {
    activeSkillSet,
    activeToolCount: activeGroupCount,
    activeToolSet,
    candidates: skillCandidates,
    groupedTools,
    hasDraftChanges,
    inactiveToolCount: inactiveGroupCount,
    loading,
    saving,
    groupedSkills,
    skillCount: groupedSkills.length,
    skills,
    submitActiveToolNames,
    toggleableToolCount: groupedTools.length,
    toggleSkill,
    toggleToolGroup,
    usedToolCount: usedGroupCount,
    usedToolSet,
    loadSkill,
    importCandidate,
    ignoreCandidate,
  };
}
