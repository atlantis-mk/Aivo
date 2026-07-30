import { useEffect, useMemo, useState } from "react";

import { hasAppBridge } from "@/lib/app-config";
import { getCodingContext, scanProjectSkills } from "@/services/aivo";
import type { domain } from "../../../bridge/go/models";

export function useProjectWorkspaceSessionState({
  activeSessionId,
  hiddenTodoPlanKeys,
  recentProjects,
  selectedProjectPath,
  sessions,
  turns,
}: {
  activeSessionId: string;
  hiddenTodoPlanKeys: Record<string, string>;
  recentProjects: domain.AssistantProject[];
  selectedProjectPath: string;
  sessions: domain.Session[];
  turns: Array<{ prompt: string }>;
}) {
  const [codingWorkspaceRoot, setCodingWorkspaceRoot] = useState("");
  const hiddenTodoPlanKey = activeSessionId
    ? hiddenTodoPlanKeys[activeSessionId] ?? ""
    : "";
  const activeSession = sessions.find(
    (session) => session.id === activeSessionId,
  );
  const activeWorkspaceRoot = activeSession?.projectPath || codingWorkspaceRoot;
  const conversationTitle =
    sessions.find((session) => session.id === activeSessionId)?.title ||
    turns[0]?.prompt ||
    "";
  const composerProjectPath = activeSessionId
    ? activeWorkspaceRoot
    : selectedProjectPath;
  const composerProject = useMemo(
    () =>
      recentProjects.find(
        (project) => project.rootPath === composerProjectPath,
      ) ?? null,
    [composerProjectPath, recentProjects],
  );

  useEffect(() => {
    if (!hasAppBridge() || !activeSessionId) {
      setCodingWorkspaceRoot("");
      return;
    }
    if (activeSession?.projectPath) {
      setCodingWorkspaceRoot("");
      return;
    }
    let cancelled = false;
    void getCodingContext(activeSessionId)
      .then((context) => {
        if (!cancelled) {
          setCodingWorkspaceRoot(context?.projectPath || "");
        }
      })
      .catch(() => {
        if (!cancelled) setCodingWorkspaceRoot("");
      });
    return () => {
      cancelled = true;
    };
  }, [activeSession?.projectPath, activeSessionId]);

  useEffect(() => {
    if (!hasAppBridge() || !activeSessionId || !activeWorkspaceRoot) return;
    void scanProjectSkills(activeWorkspaceRoot).catch(() => undefined);
  }, [activeSessionId, activeWorkspaceRoot]);

  return {
    activeSession,
    activeWorkspaceRoot,
    composerProject,
    composerProjectPath,
    conversationTitle,
    hiddenTodoPlanKey,
    setCodingWorkspaceRoot,
  };
}
