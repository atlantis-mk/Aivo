import { useCallback, useEffect, useState } from "react";

import type { QuestionRequest } from "@/services/aivo";

export function useProjectInteractionRequestState({
  activeSessionId,
  activeSessionIdRef,
}: {
  activeSessionId: string;
  activeSessionIdRef: { current: string };
}) {
  const [pendingQuestionRequests, setPendingQuestionRequests] = useState<
    QuestionRequest[]
  >([]);

  const refreshPendingQuestionRequests = useCallback(
    async (sessionId = activeSessionIdRef.current) => {
      void sessionId;
      setPendingQuestionRequests([]);
    },
    [activeSessionIdRef],
  );

  useEffect(() => {
    if (!activeSessionId) {
      setPendingQuestionRequests([]);
      return;
    }
    void refreshPendingQuestionRequests(activeSessionId);
  }, [activeSessionId, refreshPendingQuestionRequests]);

  return {
    pendingQuestionRequests,
    refreshPendingQuestionRequests,
  };
}
