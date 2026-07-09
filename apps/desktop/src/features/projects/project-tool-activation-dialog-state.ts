import { useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";

import {
  groupToolCatalogEntries,
  isToggleableCatalogTool,
  normalizeToolNames,
  pluginLabelsById,
  toolNameListsEqual,
} from "@/features/projects/project-tool-activation-model";
import {
  getSessionActiveSkills,
  getSessionActiveTools,
  importSkill,
  listPlugins,
  listSkills,
  listToolCatalog,
  loadSkillIntoSession,
  setSessionActiveSkills,
  setSessionActiveTools,
  type PluginListItem,
  type SkillEntry,
  type SkillImportCandidate,
  type ToolCatalogEntry,
} from "@/services/aivo";

export type ToolActivationDialogProps = {
  activeSessionId: string;
  defaultActiveToolNames: string[];
  onDefaultActiveToolNamesChange: (
    updater: string[] | ((current: string[]) => string[]),
  ) => void;
  onOpenChange: (open: boolean) => void;
  open: boolean;
  usedToolNames: string[];
  workspaceRoot: string;
};

export function useToolActivationDialogState({
  activeSessionId,
  defaultActiveToolNames,
  onDefaultActiveToolNamesChange,
  onOpenChange,
  open,
  usedToolNames,
  workspaceRoot,
}: ToolActivationDialogProps) {
  const [activeToolNames, setActiveToolNames] = useState<string[]>([]);
  const [savedToolNames, setSavedToolNames] = useState<string[]>([]);
  const [activeSkillIds, setActiveSkillIds] = useState<string[]>([]);
  const [savedSkillIds, setSavedSkillIds] = useState<string[]>([]);
  const [tools, setTools] = useState<ToolCatalogEntry[]>([]);
  const [skills, setSkills] = useState<SkillEntry[]>([]);
  const [skillCandidates, setSkillCandidates] = useState<
    SkillImportCandidate[]
  >([]);
  const [pluginLabels, setPluginLabels] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const defaultActiveToolNamesRef = useRef(defaultActiveToolNames);
  const activeToolSet = useMemo(
    () => new Set(activeToolNames),
    [activeToolNames],
  );
  const usedToolSet = useMemo(() => new Set(usedToolNames), [usedToolNames]);
  const toggleableTools = useMemo(
    () => tools.filter(isToggleableCatalogTool),
    [tools],
  );
  const groupedTools = useMemo(
    () => groupToolCatalogEntries(toggleableTools, pluginLabels),
    [pluginLabels, toggleableTools],
  );
  const inactiveCount = Math.max(
    0,
    toggleableTools.length - activeToolNames.length,
  );
  const activeSkillSet = useMemo(
    () => new Set(activeSkillIds),
    [activeSkillIds],
  );
  const hasDraftChanges =
    !toolNameListsEqual(activeToolNames, savedToolNames) ||
    !toolNameListsEqual(activeSkillIds, savedSkillIds);

  useEffect(() => {
    defaultActiveToolNamesRef.current = defaultActiveToolNames;
  }, [defaultActiveToolNames]);

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    setLoading(true);
    void Promise.all([
      listToolCatalog(workspaceRoot),
      listSkills({
        workspaceRoot,
        includeCandidates: true,
        includeDisabled: true,
      }),
      activeSessionId
        ? getSessionActiveTools(activeSessionId).catch(() => ({
            sessionId: activeSessionId,
            toolNames: defaultActiveToolNamesRef.current,
          }))
        : Promise.resolve({
            sessionId: "",
            toolNames: defaultActiveToolNamesRef.current,
          }),
      activeSessionId
        ? getSessionActiveSkills(activeSessionId).catch(() => ({
            sessionId: activeSessionId,
            skillIds: [],
            skills: [],
          }))
        : Promise.resolve({ sessionId: "", skillIds: [], skills: [] }),
      listPlugins(true).catch(() => [] as PluginListItem[]),
    ])
      .then(([catalogTools, skillList, activeTools, activeSkills, plugins]) => {
        if (cancelled) return;
        const normalizedActiveTools = normalizeToolNames(activeTools.toolNames);
        const normalizedActiveSkills = normalizeToolNames(activeSkills.skillIds);
        setTools(catalogTools);
        setSkills(skillList.entries ?? []);
        setSkillCandidates(skillList.candidates ?? []);
        setActiveToolNames(normalizedActiveTools);
        setSavedToolNames(normalizedActiveTools);
        setActiveSkillIds(normalizedActiveSkills);
        setSavedSkillIds(normalizedActiveSkills);
        setPluginLabels(pluginLabelsById(plugins));
      })
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
    const normalized = normalizeToolNames(activeToolNames);
    setSaving(true);
    try {
      if (!activeSessionId) {
        setSavedToolNames(normalized);
        setSavedSkillIds(normalizeToolNames(activeSkillIds));
        onDefaultActiveToolNamesChange(normalized);
        onOpenChange(false);
        return;
      }
      const saved = await setSessionActiveTools(activeSessionId, normalized);
      const savedSkills = await setSessionActiveSkills(
        activeSessionId,
        normalizeToolNames(activeSkillIds),
      );
      const savedNames = normalizeToolNames(saved.toolNames);
      const savedSkillNames = normalizeToolNames(savedSkills.skillIds);
      setActiveToolNames(savedNames);
      setSavedToolNames(savedNames);
      setActiveSkillIds(savedSkillNames);
      setSavedSkillIds(savedSkillNames);
      onDefaultActiveToolNamesChange(savedNames);
      onOpenChange(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "更新工具失败");
    } finally {
      setSaving(false);
    }
  }

  function toggleTool(name: string, enabled: boolean) {
    const next = new Set(activeToolNames);
    if (enabled) {
      next.add(name);
    } else {
      next.delete(name);
    }
    setActiveToolNames(normalizeToolNames([...next]));
  }

  function toggleSkill(id: string, enabled: boolean) {
    const next = new Set(activeSkillIds);
    if (enabled) {
      next.add(id);
    } else {
      next.delete(id);
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
        workspaceRoot,
        includeCandidates: true,
        includeDisabled: true,
      });
      setSkills(list.entries ?? []);
      setSkillCandidates(list.candidates ?? []);
      toast.success(`已导入技能 ${candidate.name}`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "导入技能失败");
    } finally {
      setSaving(false);
    }
  }

  return {
    activeSkillSet,
    activeToolCount: activeToolNames.length,
    activeToolSet,
    candidates: skillCandidates,
    groupedTools,
    hasDraftChanges,
    inactiveToolCount: inactiveCount,
    loading,
    saving,
    skillCount: skills.length,
    skills,
    submitActiveToolNames,
    toggleableToolCount: toggleableTools.length,
    toggleSkill,
    toggleTool,
    usedToolCount: usedToolNames.length,
    usedToolSet,
    loadSkill,
    importCandidate,
  };
}
