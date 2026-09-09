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
  const [unreadConversationIds, setUnreadConversationIds] = useState<
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
  const markConversationUnread = useCallback((sessionId: string) => {
    if (!sessionId) return;
    setUnreadConversationIds((currentIds) =>
      currentIds.includes(sessionId)
        ? currentIds
        : [sessionId, ...currentIds],
    );
  }, []);
  const clearConversationUnread = useCallback((sessionId: string) => {
    if (!sessionId) return;
    setUnreadConversationIds((currentIds) =>
      currentIds.filter((currentId) => currentId !== sessionId),
    );
  }, []);

  useEffect(() => {
    flushPendingAssistantDelta();
    activeSessionIdRef.current = activeSessionId;
    setUnreadConversationIds((currentIds) =>
      activeSessionId
        ? currentIds.filter((currentId) => currentId !== activeSessionId)
        : currentIds,
    );
  }, [activeSessionId, flushPendingAssistantDelta]);

  return {
    clearConversationUnread,
    activeSessionIdRef,
    pendingStopRequestedRef,
    runningConversationIds,
    setConversationRunning,
    sidebarConversationSelectionRef,
    markConversationUnread,
    unreadConversationIds,
  };
}
