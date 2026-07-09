import { useCallback, useEffect, useState } from "react";
import type { Dispatch, SetStateAction } from "react";

import { EventsOn } from "../../../bridge/runtime/runtime";
import {
  mergePendingPermissionToolCalls,
  permissionToolCall,
  samePermissionRequests,
  sameQuestionRequests,
  upsertPermissionRequest,
  upsertQuestionRequest,
} from "@/features/projects/project-conversation-events";
import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";
import {
  normalizePermissionEventPayload,
  normalizeQuestionEventPayload,
} from "@/features/projects/project-event-payloads";
import { hasAppBridge } from "@/lib/app-config";
import {
  listPermissionRequests,
  listQuestionRequests,
  type PermissionRequest,
  type QuestionRequest,
} from "@/services/aivo";
import type { domain } from "../../../bridge/go/models";

export function useProjectInteractionRequestState({
  activeSessionId,
  activeSessionIdRef,
  mergeToolActivityFromCall,
  setTurns,
}: {
  activeSessionId: string;
  activeSessionIdRef: { current: string };
  mergeToolActivityFromCall: (toolCall: domain.ToolCall) => void;
  setTurns: Dispatch<SetStateAction<ConversationTurn[]>>;
}) {
  const [pendingPermissionRequests, setPendingPermissionRequests] = useState<
    PermissionRequest[]
  >([]);
  const [pendingQuestionRequests, setPendingQuestionRequests] = useState<
    QuestionRequest[]
  >([]);
  const [
    pendingPermissionCountsBySessionId,
    setPendingPermissionCountsBySessionId,
  ] = useState<Record<string, number>>({});

  const setPendingPermissionCountForSession = useCallback(
    (sessionId: string, count: number) => {
      setPendingPermissionCountsBySessionId((currentCounts) => {
        if ((currentCounts[sessionId] ?? 0) === count) {
          return currentCounts;
        }
        if (count === 0) {
          const nextCounts = { ...currentCounts };
          delete nextCounts[sessionId];
          return nextCounts;
        }
        return {
          ...currentCounts,
          [sessionId]: count,
        };
      });
    },
    [],
  );

  const refreshPendingPermissionRequests = useCallback(
    async (sessionId = activeSessionIdRef.current) => {
      if (!hasAppBridge() || !sessionId) {
        setPendingPermissionRequests([]);
        return;
      }
      const requests =
        (await listPermissionRequests(sessionId, "pending").catch(
          () => [] as PermissionRequest[],
        )) ?? [];
      if (activeSessionIdRef.current !== sessionId) return;
      setPendingPermissionRequests((currentRequests) =>
        samePermissionRequests(currentRequests, requests)
          ? currentRequests
          : requests,
      );
      setPendingPermissionCountForSession(sessionId, requests.length);
      if (requests.length > 0) {
        setTurns((currentTurns) =>
          mergePendingPermissionToolCalls(currentTurns, requests),
        );
      }
    },
    [activeSessionIdRef, setPendingPermissionCountForSession, setTurns],
  );

  const refreshPendingQuestionRequests = useCallback(
    async (sessionId = activeSessionIdRef.current) => {
      if (!hasAppBridge() || !sessionId) {
        setPendingQuestionRequests([]);
        return;
      }
      const requests =
        (await listQuestionRequests(sessionId, "pending").catch(
          () => [] as QuestionRequest[],
        )) ?? [];
      if (activeSessionIdRef.current !== sessionId) return;
      setPendingQuestionRequests((currentRequests) =>
        sameQuestionRequests(currentRequests, requests)
          ? currentRequests
          : requests,
      );
    },
    [activeSessionIdRef],
  );

  const clearPendingPermissionCountForSession = useCallback(
    (sessionId: string) => {
      setPendingPermissionCountsBySessionId((currentCounts) => {
        if (!(sessionId in currentCounts)) return currentCounts;
        const nextCounts = { ...currentCounts };
        delete nextCounts[sessionId];
        return nextCounts;
      });
    },
    [],
  );

  useEffect(() => {
    if (!activeSessionId) {
      setPendingPermissionRequests([]);
      return;
    }
    void refreshPendingPermissionRequests(activeSessionId);
  }, [activeSessionId, refreshPendingPermissionRequests]);

  useEffect(() => {
    if (!activeSessionId) {
      setPendingQuestionRequests([]);
      return;
    }
    void refreshPendingQuestionRequests(activeSessionId);
  }, [activeSessionId, refreshPendingQuestionRequests]);

  useEffect(() => {
    if (!hasAppBridge()) return;
    return EventsOn("permission.requested", (...payloads: unknown[]) => {
      const permission = normalizePermissionEventPayload(payloads);
      if (!permission?.id || !permission.sessionId) return;
      if (permission.sessionId !== activeSessionIdRef.current) {
        setPendingPermissionCountsBySessionId((currentCounts) => ({
          ...currentCounts,
          [permission.sessionId!]: Math.max(
            currentCounts[permission.sessionId!] ?? 0,
            1,
          ),
        }));
        return;
      }
      setPendingPermissionRequests((currentRequests) => {
        const nextRequests = upsertPermissionRequest(
          currentRequests,
          permission,
        );
        return samePermissionRequests(currentRequests, nextRequests)
          ? currentRequests
          : nextRequests;
      });
      setPendingPermissionCountForSession(permission.sessionId, 1);
      setTurns((currentTurns) =>
        mergePendingPermissionToolCalls(currentTurns, [permission]),
      );
      mergeToolActivityFromCall(permissionToolCall(permission));
    });
  }, [
    activeSessionIdRef,
    mergeToolActivityFromCall,
    setPendingPermissionCountForSession,
    setTurns,
  ]);

  useEffect(() => {
    if (!hasAppBridge()) return;
    return EventsOn("permission.resolved", (...payloads: unknown[]) => {
      const permission = normalizePermissionEventPayload(payloads);
      if (!permission?.id || !permission.sessionId) return;
      setPendingPermissionCountsBySessionId((currentCounts) => {
        const currentCount = currentCounts[permission.sessionId!] ?? 0;
        const nextCount = Math.max(0, currentCount - 1);
        if (nextCount === currentCount) return currentCounts;
        if (nextCount === 0) {
          const nextCounts = { ...currentCounts };
          delete nextCounts[permission.sessionId!];
          return nextCounts;
        }
        return { ...currentCounts, [permission.sessionId!]: nextCount };
      });
      if (permission.sessionId !== activeSessionIdRef.current) return;
      setPendingPermissionRequests((currentRequests) => {
        const nextRequests = currentRequests.filter(
          (currentRequest) => currentRequest.id !== permission.id,
        );
        return nextRequests.length === currentRequests.length
          ? currentRequests
          : nextRequests;
      });
    });
  }, [activeSessionIdRef]);

  useEffect(() => {
    if (!hasAppBridge()) return;
    return EventsOn("question.requested", (...payloads: unknown[]) => {
      const question = normalizeQuestionEventPayload(payloads);
      if (!question?.id || !question.sessionId) return;
      if (question.sessionId !== activeSessionIdRef.current) return;
      setPendingQuestionRequests((currentRequests) => {
        const nextRequests = upsertQuestionRequest(currentRequests, question);
        return sameQuestionRequests(currentRequests, nextRequests)
          ? currentRequests
          : nextRequests;
      });
    });
  }, [activeSessionIdRef]);

  useEffect(() => {
    if (!hasAppBridge()) return;
    return EventsOn("question.resolved", (...payloads: unknown[]) => {
      const question = normalizeQuestionEventPayload(payloads);
      if (!question?.id || !question.sessionId) return;
      if (question.sessionId !== activeSessionIdRef.current) return;
      setPendingQuestionRequests((currentRequests) => {
        const nextRequests = currentRequests.filter(
          (currentRequest) => currentRequest.id !== question.id,
        );
        return nextRequests.length === currentRequests.length
          ? currentRequests
          : nextRequests;
      });
    });
  }, [activeSessionIdRef]);

  return {
    clearPendingPermissionCountForSession,
    pendingPermissionCountsBySessionId,
    pendingPermissionRequests,
    pendingQuestionRequests,
    refreshPendingPermissionRequests,
    refreshPendingQuestionRequests,
  };
}
