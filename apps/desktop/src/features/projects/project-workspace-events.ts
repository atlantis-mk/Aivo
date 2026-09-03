import { useEffect } from "react";
import type { Dispatch, SetStateAction } from "react";

import { EventsOn } from "../../../bridge/runtime/runtime";
import {
  getTurnElapsedSeconds,
  type ConversationTurn,
} from "@/features/projects/conversation-timeline-model";
import {
  isDelegateTaskToolName,
  mergeRuntimeTurn,
  mergeSingleToolCall,
  mergeSystemNoteEvent,
  moveOpenResponseTextToAssistantPreambleBeforeTool,
  appendToolCallOutput,
  updatePermissionPauseState,
  upsertSession,
} from "@/features/projects/project-conversation-events";
import { codexToolCallFromItem } from "@/features/projects/project-codex-tool-calls";
import type { LoadConversationTurnsOptions } from "@/features/projects/project-conversation-turn-loader";
import {
  normalizeAssistantDeltaPayload,
  normalizeSessionUpdatedPayload,
  normalizeSessionEventUpdatedPayload,
  normalizeTodoItemsUpdatedPayload,
  normalizeToolCallUpdatedPayload,
  normalizeTurnUpdatedPayload,
} from "@/features/projects/project-event-payloads";
import { hasAppBridge, hasCodexDesktopBridge } from "@/lib/app-config";
import {
  listSessions,
  type PermissionRequest,
  type TodoItem,
} from "@/services/aivo";
import type { domain } from "../../../bridge/go/models";

export function useProjectWorkspaceEvents({
  activeSessionIdRef,
  activeWorkspaceRoot,
  enqueueAssistantDelta,
  flushPendingAssistantDelta,
  loadConversationTurns,
  mergeToolActivityFromCall,
  pendingPermissionRequests,
  refreshAgentRuntimeState,
  refreshPendingPermissionRequests,
  refreshPendingQuestionRequests,
  setConversationRunning,
  setSessions,
  setTodoItems,
  setTurns,
}: {
  activeSessionIdRef: { current: string };
  activeWorkspaceRoot: string;
  enqueueAssistantDelta: (payload: {
    delta: string;
    sessionId?: string;
    turnId?: string;
  }) => void;
  flushPendingAssistantDelta: () => void;
  loadConversationTurns: (
    sessionId: string,
    options?: LoadConversationTurnsOptions,
  ) => Promise<void>;
  mergeToolActivityFromCall: (toolCall: domain.ToolCall) => void;
  pendingPermissionRequests: PermissionRequest[];
  refreshAgentRuntimeState: (sessionId?: string) => Promise<void>;
  refreshPendingPermissionRequests: (sessionId?: string) => Promise<void>;
  refreshPendingQuestionRequests: (sessionId?: string) => Promise<void>;
  setConversationRunning: (sessionId: string, running: boolean) => void;
  setSessions: Dispatch<SetStateAction<domain.Session[]>>;
  setTodoItems: Dispatch<SetStateAction<TodoItem[]>>;
  setTurns: Dispatch<SetStateAction<ConversationTurn[]>>;
}) {
  useEffect(() => {
    if (!hasAppBridge()) return;
    return EventsOn("session.updated", (...payloads: unknown[]) => {
      const payload = normalizeSessionUpdatedPayload(payloads);
      if (payload.session) {
        setSessions((currentSessions) =>
          upsertSession(currentSessions, payload.session!),
        );
      }
    });
  }, [setSessions]);

  useEffect(() => {
    if (!hasAppBridge()) return;
    const handleSessionEvent = (...payloads: unknown[]) => {
      const event = normalizeSessionEventUpdatedPayload(payloads);
      if (!event?.id || event.sessionId !== activeSessionIdRef.current) return;
      setTurns((currentTurns) => mergeSystemNoteEvent(currentTurns, event));
    };
    const offCreated = EventsOn("session_event.created", handleSessionEvent);
    const offUpdated = EventsOn("session_event.updated", handleSessionEvent);
    return () => {
      offCreated();
      offUpdated();
    };
  }, [activeSessionIdRef, setTurns]);

  useEffect(() => {
    if (!hasAppBridge()) return;
    const handleTurnEvent = (...payloads: unknown[]) => {
      const turn = normalizeTurnUpdatedPayload(payloads);
      if (!turn?.id || !turn.sessionId) return;
      setConversationRunning(turn.sessionId, turn.status === "running");
      if (turn.sessionId !== activeSessionIdRef.current) return;
      if (turn.status !== "running") {
        // Assistant deltas are batched until the next animation frame. A
        // terminal event can arrive first and would otherwise close the turn
        // before the buffered final text has a chance to attach to it.
        flushPendingAssistantDelta();
      }
      setTurns((currentTurns) => mergeRuntimeTurn(currentTurns, turn));
    };
    const offStarted = EventsOn("turn.started", handleTurnEvent);
    const offCompleted = EventsOn("turn.completed", handleTurnEvent);
    const offFailed = EventsOn("turn.failed", handleTurnEvent);
    const offCancelled = EventsOn("turn.cancelled", handleTurnEvent);
    return () => {
      offStarted();
      offCompleted();
      offFailed();
      offCancelled();
    };
  }, [
    activeSessionIdRef,
    flushPendingAssistantDelta,
    setConversationRunning,
    setTurns,
  ]);

  useEffect(() => {
    if (!hasAppBridge()) return;
    const handleToolCallCreatedEvent = (...payloads: unknown[]) => {
      const toolCall = normalizeToolCallUpdatedPayload(payloads);
      if (!toolCall?.id || !toolCall.sessionId) return;
      if (toolCall.sessionId !== activeSessionIdRef.current) return;
      flushPendingAssistantDelta();
      setTurns((currentTurns) =>
        moveOpenResponseTextToAssistantPreambleBeforeTool(
          mergeSingleToolCall(currentTurns, toolCall),
          toolCall,
        ),
      );
      mergeToolActivityFromCall(toolCall);
      if (isDelegateTaskToolName(toolCall.name)) {
        void refreshAgentRuntimeState(toolCall.sessionId);
      }
    };
    const handleToolCallUpdatedEvent = (...payloads: unknown[]) => {
      const toolCall = normalizeToolCallUpdatedPayload(payloads);
      if (!toolCall?.id || !toolCall.sessionId) return;
      if (toolCall.sessionId !== activeSessionIdRef.current) return;
      setTurns((currentTurns) => mergeSingleToolCall(currentTurns, toolCall));
      mergeToolActivityFromCall(toolCall);
      if (isDelegateTaskToolName(toolCall.name)) {
        void refreshAgentRuntimeState(toolCall.sessionId);
      }
    };
    const offCreated = EventsOn(
      "tool_call.created",
      handleToolCallCreatedEvent,
    );
    const offUpdated = EventsOn(
      "tool_call.updated",
      handleToolCallUpdatedEvent,
    );
    return () => {
      offCreated();
      offUpdated();
    };
  }, [
    activeSessionIdRef,
    flushPendingAssistantDelta,
    mergeToolActivityFromCall,
    refreshAgentRuntimeState,
    setTurns,
  ]);

  useEffect(() => {
    if (!hasAppBridge()) return;
    return EventsOn("todo_items.updated", (...payloads: unknown[]) => {
      const payload = normalizeTodoItemsUpdatedPayload(payloads);
      if (!payload || payload.sessionId !== activeSessionIdRef.current) return;
      if (
        payload.projectPath &&
        activeWorkspaceRoot &&
        payload.projectPath !== activeWorkspaceRoot
      ) {
        return;
      }
      setTodoItems(payload.items);
    });
  }, [activeSessionIdRef, activeWorkspaceRoot, setTodoItems]);

  useEffect(() => {
    if (!hasAppBridge()) return;
    return EventsOn("assistant.delta", (...payloads: unknown[]) => {
      const payload = normalizeAssistantDeltaPayload(payloads);
      const delta = payload.delta;
      if (!delta || payload?.sessionId !== activeSessionIdRef.current) return;
      if (window.localStorage.getItem("aivo:debug-stream") === "1") {
        console.debug("[aivo-stream] assistant.delta", {
          length: delta.length,
          preview: delta.slice(0, 120),
          sessionId: payload.sessionId,
          turnId: payload.turnId,
        });
      }
      enqueueAssistantDelta({
        delta,
        sessionId: payload.sessionId,
        turnId: payload.turnId,
      });
    });
  }, [activeSessionIdRef, enqueueAssistantDelta]);

  useEffect(() => {
    if (!hasCodexDesktopBridge()) return;
    return window.aivoDesktop.codex.onRuntimeEvent((event) => {
      const payload = recordValue(event.params);
      const threadId = stringValue(payload?.threadId);
      const completedTurn = recordValue(payload?.turn);
      const turnId = stringValue(payload?.turnId) ?? stringValue(completedTurn?.id);
      if (!threadId || !turnId || threadId !== activeSessionIdRef.current) return;

      if (event.method === "item/agentMessage/delta") {
        const delta = stringValue(payload?.delta);
        if (!delta) return;
        setTurns((currentTurns) =>
          updateCodexTurn(currentTurns, turnId, (turn) => ({
            ...turn,
            responseText: `${turn.responseText}${delta}`,
            responseVisible: true,
            thinkingSeconds: Math.max(0, Math.floor((Date.now() - turn.startedAt) / 1000)),
          })),
        );
        return;
      }

      if (event.method === "item/commandExecution/outputDelta") {
        const itemId = stringValue(payload?.itemId);
        const delta = stringValue(payload?.delta);
        if (!itemId || !delta) return;
        const toolCallId = `codex:${threadId}:${itemId}`;
        setTurns((currentTurns) =>
          updateCodexTurn(currentTurns, turnId, (turn) => ({
            ...turn,
            toolCalls: turn.toolCalls.map((toolCall) =>
              toolCall.id === toolCallId
                ? appendToolCallOutput(toolCall, delta)
                : toolCall,
            ),
          })),
        );
        return;
      }

      if (event.method === "item/started" || event.method === "item/completed") {
        const toolCall = codexToolCallFromItem({
          item: payload?.item,
          threadId,
          turnId,
        });
        if (!toolCall) return;

        if (event.method === "item/started") {
          flushPendingAssistantDelta();
          setTurns((currentTurns) =>
            moveOpenResponseTextToAssistantPreambleBeforeTool(
              mergeSingleToolCall(currentTurns, toolCall),
              toolCall,
            ),
          );
        } else {
          setTurns((currentTurns) => mergeSingleToolCall(currentTurns, toolCall));
        }
        mergeToolActivityFromCall(toolCall);
        return;
      }

      if (event.method !== "turn/completed") return;

      const error = recordValue(completedTurn?.error);
      const errorMessage = stringValue(error?.message);
      const durationMs = numberValue(completedTurn?.durationMs);
      setConversationRunning(threadId, false);
      setTurns((currentTurns) =>
        updateCodexTurn(currentTurns, turnId, (currentTurn) => ({
          ...currentTurn,
          responseCompletedAt: new Date(),
          responseText: currentTurn.responseText || errorMessage || currentTurn.responseText,
          responseVisible: true,
          thinkingSeconds:
            durationMs === null
              ? getTurnElapsedSeconds(currentTurn)
              : Math.max(0, Math.floor(durationMs / 1000)),
        })),
      );
    });
  }, [
    activeSessionIdRef,
    flushPendingAssistantDelta,
    mergeToolActivityFromCall,
    setConversationRunning,
    setTurns,
  ]);

  useEffect(() => {
    if (!hasAppBridge()) return;
    return EventsOn("events.reconnected", () => {
      void listSessions(50)
        .then((nextSessions) => setSessions(nextSessions ?? []))
        .catch(() => undefined);
      const sessionId = activeSessionIdRef.current;
      if (sessionId) {
        void loadConversationTurns(sessionId, { snapToBottomAfterLoad: true });
        void refreshPendingPermissionRequests(sessionId);
        void refreshPendingQuestionRequests(sessionId);
      }
    });
  }, [
    activeSessionIdRef,
    loadConversationTurns,
    refreshPendingPermissionRequests,
    refreshPendingQuestionRequests,
    setSessions,
  ]);

  useEffect(() => {
    const now = Date.now();
    setTurns((currentTurns) =>
      updatePermissionPauseState(currentTurns, pendingPermissionRequests, now),
    );
  }, [pendingPermissionRequests, setTurns]);
}

function updateCodexTurn(
  turns: ConversationTurn[],
  turnId: string,
  update: (turn: ConversationTurn) => ConversationTurn,
) {
  const targetId =
    turns.find((turn) => turn.turnId === turnId)?.id ??
    [...turns].reverse().find((turn) => !turn.turnId && !turn.responseCompletedAt)?.id;
  if (!targetId) return turns;
  return turns.map((turn) =>
    turn.id === targetId ? update({ ...turn, turnId }) : turn,
  );
}

function recordValue(value: unknown): Record<string, unknown> | null {
  return typeof value === "object" && value !== null
    ? (value as Record<string, unknown>)
    : null;
}

function stringValue(value: unknown): string | null {
  return typeof value === "string" ? value : null;
}

function numberValue(value: unknown): number | null {
  return typeof value === "number" && Number.isFinite(value) ? value : null;
}
