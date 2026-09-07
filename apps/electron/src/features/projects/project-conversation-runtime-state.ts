import { useCallback, useEffect, useRef, useState } from "react";

export function useProjectConversationRuntimeState({
  activeSessionId,
  flushPendingAssistantDelta,
}: {
  activeSessionId: string;
  flushPendingAssistantDelta: () => void;
}) {
  const [runningConversationIds, setRunningConversationIds] = useState<
    string[]
  >([]);
  const activeSessionIdRef = useRef("");
  const pendingStopRequestedRef = useRef(false);
  const sidebarConversationSelectionRef = useRef(0);
  const setConversationRunning = useCallback(
    (sessionId: string, running: boolean) => {
      if (!sessionId) return;
      setRunningConversationIds((currentIds) => {
        const alreadyRunning = currentIds.includes(sessionId);
        if (running) {
          return alreadyRunning ? currentIds : [sessionId, ...currentIds];
        }
        return alreadyRunning
          ? currentIds.filter((currentId) => currentId !== sessionId)
          : currentIds;
      });
    },
    [],
  );

  useEffect(() => {
    flushPendingAssistantDelta();
    activeSessionIdRef.current = activeSessionId;
  }, [activeSessionId, flushPendingAssistantDelta]);

  return {
    activeSessionIdRef,
    pendingStopRequestedRef,
    runningConversationIds,
    setConversationRunning,
    sidebarConversationSelectionRef,
  };
}
