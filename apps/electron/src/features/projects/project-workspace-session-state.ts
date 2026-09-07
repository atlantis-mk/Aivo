import { useEffect, useMemo, useState } from "react";

import type { domain } from "@/types/codex-domain";

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
    if (!activeSessionId) {
      setCodingWorkspaceRoot("");
      return;
    }
    if (activeSession?.projectPath) {
      setCodingWorkspaceRoot("");
      return;
    }
    setCodingWorkspaceRoot("");
  }, [activeSession?.projectPath, activeSessionId]);

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
