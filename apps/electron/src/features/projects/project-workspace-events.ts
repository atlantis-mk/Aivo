import { useCallback, useEffect } from "react";
import { useRef, type Dispatch, type SetStateAction } from "react";

import { onDesktopEvent } from "@/lib/desktop-events";
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
import { hasCodexDesktopBridge } from "@/lib/app-config";
import { listCodexSessions } from "@/services/codex-thread-service";
import {
  useProjectCodexDeltaBuffer,
  type ProjectCodexDelta,
} from "@/features/projects/project-codex-delta-buffer";
import { listSessions, type TodoItem } from "@/services/aivo";
import type { domain } from "@/types/codex-domain";

export function useProjectWorkspaceEvents({
  activeSessionIdRef,
  activeWorkspaceRoot,
  enqueueAssistantDelta,
  flushPendingAssistantDelta,
  loadConversationTurns,
  mergeToolActivityFromCall,
  refreshAgentRuntimeState,
  refreshPendingQuestionRequests,
  setConversationRunning,
  setSessions,
  setTodoItems,
  setTurns,
  markConversationUnread,
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
  refreshAgentRuntimeState: (sessionId?: string) => Promise<void>;
  refreshPendingQuestionRequests: (sessionId?: string) => Promise<void>;
  setConversationRunning: (sessionId: string, running: boolean) => void;
  setSessions: Dispatch<SetStateAction<domain.Session[]>>;
  setTodoItems: Dispatch<SetStateAction<TodoItem[]>>;
  setTurns: Dispatch<SetStateAction<ConversationTurn[]>>;
  markConversationUnread: (sessionId: string) => void;
}) {
  const codexDeltaTurnIdsRef = useRef(new Set<string>());

  const renderCodexDeltas = useCallback(
    (deltas: ProjectCodexDelta[]) => {
      setTurns((currentTurns) => {
        let nextTurns = currentTurns;
        for (const { delta, threadId, turnId } of deltas) {
          nextTurns = updateCodexTurn(nextTurns, turnId, threadId, (turn) => ({
            ...turn,
            responseText: `${turn.responseText}${delta}`,
            responseVisible: true,
            thinkingSeconds: getTurnElapsedSeconds(turn),
          }));
        }
        return nextTurns;
      });
    },
    [setTurns],
  );

  const { enqueue: enqueueCodexDelta, flush: flushCodexDeltas } =
    useProjectCodexDeltaBuffer({
      activeThreadIdRef: activeSessionIdRef,
      render: renderCodexDeltas,
    });

  useEffect(() => {
    if (!hasCodexDesktopBridge()) return;
    return window.aivoDesktop.codex.onRuntimeEvent((event) => {
      const payload = recordValue(event.params);
      const threadId = stringValue(payload?.threadId);
      if (window.localStorage.getItem("aivo:debug-stream") === "1") {
        console.log("[aivo-stream] codex runtime event", {
          method: event.method,
          threadId,
          activeThreadId: activeSessionIdRef.current,
          turnId: stringValue(payload?.turnId),
          deltaLength:
            typeof payload?.delta === "string" ? payload.delta.length : 0,
        });
      }
      if (event.method === "turn/completed" && threadId) {
        setConversationRunning(threadId, false);
        if (threadId !== activeSessionIdRef.current) {
          markConversationUnread(threadId);
        }
        void listCodexSessions(50)
          .then((nextSessions) => setSessions(nextSessions ?? []))
          .catch(() => undefined);
      }
      const completedTurn = recordValue(payload?.turn);
      const turnId =
        stringValue(payload?.turnId) ?? stringValue(completedTurn?.id);
      if (!threadId || !turnId || threadId !== activeSessionIdRef.current)
        return;

      if (event.method === "item/agentMessage/delta") {
        const delta = stringValue(payload?.delta);
        if (!delta) return;
        codexDeltaTurnIdsRef.current.add(turnId);
        enqueueCodexDelta({ delta, threadId, turnId });
        return;
      }

      if (event.method === "item/commandExecution/outputDelta") {
        const itemId = stringValue(payload?.itemId);
        const delta = stringValue(payload?.delta);
        if (!itemId || !delta) return;
        const toolCallId = `codex:${threadId}:${itemId}`;
        setTurns((currentTurns) =>
          updateCodexTurn(currentTurns, turnId, threadId, (turn) => ({
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

      if (
        event.method === "item/started" ||
        event.method === "item/completed"
      ) {
        const agentMessageText = agentMessageTextFromItem(payload?.item);
        const item = recordValue(payload?.item);
        if (window.localStorage.getItem("aivo:debug-stream") === "1") {
          console.log("[aivo-stream] codex item event", {
            method: event.method,
            threadId,
            turnId,
            itemType: item?.type ?? null,
            itemId: typeof item?.id === "string" ? item.id : null,
            textLength: typeof item?.text === "string" ? item.text.length : 0,
            agentMessageTextLength: agentMessageText?.length ?? 0,
            deltaSeen: codexDeltaTurnIdsRef.current.has(turnId),
          });
        }
        if (agentMessageText && !codexDeltaTurnIdsRef.current.has(turnId)) {
          flushPendingAssistantDelta();
          setTurns((currentTurns) =>
            updateCodexTurn(currentTurns, turnId, threadId, (turn) => ({
              ...turn,
              responseText:
                turn.responseText &&
                !agentMessageText.startsWith(turn.responseText)
                  ? `${turn.responseText}\n\n${agentMessageText}`
                  : agentMessageText,
              responseVisible: true,
              thinkingSeconds: Math.max(
                0,
                Math.floor((Date.now() - turn.startedAt) / 1000),
              ),
            })),
          );
          if (event.method === "item/completed") {
            codexDeltaTurnIdsRef.current.delete(turnId);
          }
          return;
        }

        if (event.method === "item/completed") {
          codexDeltaTurnIdsRef.current.delete(turnId);
        }

        const toolCall = codexToolCallFromItem({
          item: payload?.item,
          threadId,
          turnId,
        });
        if (!toolCall) return;

        if (event.method === "item/started") {
          flushCodexDeltas(turnId);
          flushPendingAssistantDelta();
          setTurns((currentTurns) =>
            moveOpenResponseTextToAssistantPreambleBeforeTool(
              mergeSingleToolCall(currentTurns, toolCall),
              toolCall,
            ),
          );
        } else {
          setTurns((currentTurns) =>
            mergeSingleToolCall(currentTurns, toolCall),
          );
        }
        mergeToolActivityFromCall(toolCall);
        return;
      }

      if (event.method !== "turn/completed") return;

      const error = recordValue(completedTurn?.error);
      const errorMessage = stringValue(error?.message);
      const durationMs = numberValue(completedTurn?.durationMs);
      setTurns((currentTurns) =>
        finalizeCodexTurn(currentTurns, turnId, threadId, {
          completedTurn,
          durationMs,
          errorMessage,
        }),
      );
    });
  }, [
    activeSessionIdRef,
    enqueueCodexDelta,
    flushCodexDeltas,
    flushPendingAssistantDelta,
    markConversationUnread,
    mergeToolActivityFromCall,
    setConversationRunning,
    setSessions,
    setTurns,
  ]);
}

function updateCodexTurn(
  turns: ConversationTurn[],
  turnId: string,
  sessionId: string,
  update: (turn: ConversationTurn) => ConversationTurn,
) {
  const ownedTurns = turns.filter((turn) => turn.sessionId === sessionId);
  const targetId =
    [...ownedTurns]
      .reverse()
      .find(
        (turn) =>
          turn.steered &&
          (turn.turnId === turnId || turn.steerTargetTurnId === turnId),
      )?.id ??
    ownedTurns.find((turn) => turn.turnId === turnId)?.id ??
    [...ownedTurns]
      .reverse()
      .find((turn) => !turn.turnId || !turn.responseCompletedAt)?.id;
  if (window.localStorage.getItem("aivo:debug-stream") === "1") {
    console.log("[aivo-stream] codex turn update", {
      found: Boolean(targetId),
      targetId,
      turnId,
      turnIds: ownedTurns.map((turn) => turn.turnId ?? null),
      localIds: turns.map((turn) => turn.id),
    });
  }
  if (!targetId) return turns;
  return turns.map((turn) =>
    turn.id === targetId ? update({ ...turn, turnId }) : turn,
  );
}

function finalizeCodexTurn(
  turns: ConversationTurn[],
  turnId: string,
  sessionId: string,
  {
    completedTurn,
    durationMs,
    errorMessage,
  }: {
    completedTurn: Record<string, unknown> | null;
    durationMs: number | null;
    errorMessage: string | null;
  },
) {
  const updatedTurns = updateCodexTurn(turns, turnId, sessionId, (currentTurn) => ({
    ...currentTurn,
    model: stringValue(completedTurn?.model) || currentTurn.model,
    modelProvider:
      stringValue(completedTurn?.modelProvider) || currentTurn.modelProvider,
    responseCompletedAt: new Date(),
    responseText:
      currentTurn.responseText ||
      errorMessage ||
      currentTurn.responseText ||
      "",
    responseVisible: true,
    thinkingSeconds:
      durationMs === null
        ? getTurnElapsedSeconds(currentTurn)
        : Math.max(0, Math.floor(durationMs / 1000)),
  }));

  return updatedTurns.map((turn) =>
    turn.sessionId === sessionId &&
      (turn.turnId === turnId || turn.steerTargetTurnId === turnId)
      ? turn.responseCompletedAt
        ? turn
        : {
            ...turn,
            model: stringValue(completedTurn?.model) || turn.model,
            modelProvider:
              stringValue(completedTurn?.modelProvider) || turn.modelProvider,
            responseCompletedAt: new Date(),
            responseVisible: true,
            thinkingSeconds:
              durationMs === null
                ? getTurnElapsedSeconds(turn)
                : Math.max(0, Math.floor(durationMs / 1000)),
          }
      : turn,
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

function agentMessageTextFromItem(value: unknown): string | null {
  if (typeof value !== "object" || value === null) return null;
  const item = value as Record<string, unknown>;
  if (item.type !== "agentMessage") return null;
  return typeof item.text === "string" && item.text.trim() ? item.text : null;
}
