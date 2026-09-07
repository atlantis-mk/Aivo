import { useCallback, useEffect, useMemo, useState } from "react";

import { useProjectTodoFloatingState } from "@/features/projects/project-workspace-derived-state";
import type {
  AgentModeDefinition,
  AgentModeId,
  AgentRun,
  TodoItem,
} from "@/services/aivo";
import type { domain } from "@/types/codex-domain";

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
    setAgentModes([]);
  }, []);

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
      setAgentRuns([]);
      setTodoItems([]);
      setVisibleTodoPlanItems([]);
    },
    [activeSessionIdRef],
  );

  useEffect(() => {
    refreshAgentRuntimeState(activeSessionId);
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
