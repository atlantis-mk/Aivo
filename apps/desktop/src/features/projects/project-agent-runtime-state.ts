import { useCallback, useEffect, useMemo, useState } from "react";

import { useProjectTodoFloatingState } from "@/features/projects/project-workspace-derived-state";
import { hasAppBridge } from "@/lib/app-config";
import {
  listAgentModesForProject,
  listAgentRuns,
  listTodoItems,
  type AgentModeDefinition,
  type AgentModeId,
  type AgentRun,
  type TodoItem,
} from "@/services/aivo";
import type { domain } from "../../../bridge/go/models";

export function useProjectAgentRuntimeState({
  activeSession,
  activeSessionId,
  activeSessionIdRef,
  activeWorkspaceRoot,
  hiddenTodoPlanKey,
  setHiddenTodoPlanKeyForSession,
}: {
  activeSession: domain.Session | undefined;
  activeSessionId: string;
  activeSessionIdRef: { current: string };
  activeWorkspaceRoot: string;
  hiddenTodoPlanKey: string;
  setHiddenTodoPlanKeyForSession: (
    sessionId: string,
    planKey: string,
  ) => void;
}) {
  const [agentModes, setAgentModes] = useState<AgentModeDefinition[]>([]);
  const [agentMode, setAgentMode] = useState<AgentModeId>("assistant");
  const [agentRuns, setAgentRuns] = useState<AgentRun[]>([]);
  const [todoItems, setTodoItems] = useState<TodoItem[]>([]);
  const [visibleTodoPlanItems, setVisibleTodoPlanItems] = useState<TodoItem[]>(
    [],
  );
  const activeParentSessionId = activeSession?.parentSessionId || "";
  const isSubagentSession = Boolean(activeParentSessionId);
  const activeSubagentRun = useMemo(
    () =>
      isSubagentSession
        ? agentRuns.find((run) => run.sessionId === activeSessionId)
        : undefined,
    [activeSessionId, agentRuns, isSubagentSession],
  );
  const activeRunningSubagentRun =
    activeSubagentRun?.status === "running" ? activeSubagentRun : undefined;

  const refreshAgentModes = useCallback(() => {
    if (!hasAppBridge()) return;
    void listAgentModesForProject(activeWorkspaceRoot || "", false)
      .then((modes) => {
        setAgentModes(
          modes.filter((mode) => !mode.hidden && mode.mode !== "subagent"),
        );
      })
      .catch(() => undefined);
  }, [activeWorkspaceRoot]);

  useEffect(() => {
    refreshAgentModes();
  }, [refreshAgentModes]);

  useEffect(() => {
    const handleAgentModesChanged = () => refreshAgentModes();
    window.addEventListener("aivo:agent-modes-changed", handleAgentModesChanged);
    return () => {
      window.removeEventListener(
        "aivo:agent-modes-changed",
        handleAgentModesChanged,
      );
    };
  }, [refreshAgentModes]);

  useEffect(() => {
    const sessionMode =
      ((
        activeSession as
          (domain.Session & { agentMode?: AgentModeId }) | undefined
      )?.agentMode as AgentModeId | undefined) || "assistant";
    setAgentMode(sessionMode);
  }, [activeSession]);

  useEffect(() => {
    setVisibleTodoPlanItems([]);
  }, [activeSessionId]);

  const refreshAgentRuntimeState = useCallback(
    async (sessionId = activeSessionIdRef.current) => {
      if (!hasAppBridge() || !sessionId) {
        setAgentRuns([]);
        setTodoItems([]);
        setVisibleTodoPlanItems([]);
        return;
      }
      const projectPath = activeWorkspaceRoot || "";
      const [runs, todos] = await Promise.all([
        listAgentRuns(sessionId, 12).catch(() => [] as AgentRun[]),
        listTodoItems(sessionId, projectPath, 12).catch(() => [] as TodoItem[]),
      ]);
      setAgentRuns(runs);
      setTodoItems(todos);
    },
    [activeSessionIdRef, activeWorkspaceRoot],
  );

  useEffect(() => {
    if (!hasAppBridge() || !activeSessionId) {
      setAgentRuns([]);
      setTodoItems([]);
      setVisibleTodoPlanItems([]);
      return;
    }
    let cancelled = false;
    void refreshAgentRuntimeState(activeSessionId).then(() => {
      if (cancelled) return;
    });
    return () => {
      cancelled = true;
    };
  }, [activeSessionId, refreshAgentRuntimeState]);

  useEffect(() => {
    setVisibleTodoPlanItems((current) =>
      todoItems.length > 0 ? todoItems : current,
    );
  }, [todoItems]);

  const {
    isVisibleTodoPlanComplete,
    shouldShowTodoFloatingStatus,
    visibleTodoPlanKey,
  } = useProjectTodoFloatingState({
    hiddenTodoPlanKey,
    visibleTodoPlanItems,
  });

  const hideCompletedTodoPlan = useCallback(() => {
    if (!isVisibleTodoPlanComplete || !visibleTodoPlanKey) return;
    setHiddenTodoPlanKeyForSession(activeSessionId, visibleTodoPlanKey);
  }, [
    activeSessionId,
    isVisibleTodoPlanComplete,
    setHiddenTodoPlanKeyForSession,
    visibleTodoPlanKey,
  ]);

  return {
    activeParentSessionId,
    activeRunningSubagentRun,
    activeSubagentRun,
    agentMode,
    agentModes,
    agentRuns,
    hideCompletedTodoPlan,
    isSubagentSession,
    isVisibleTodoPlanComplete,
    refreshAgentRuntimeState,
    setAgentMode,
    setTodoItems,
    shouldShowTodoFloatingStatus,
    visibleTodoPlanItems,
  };
}
