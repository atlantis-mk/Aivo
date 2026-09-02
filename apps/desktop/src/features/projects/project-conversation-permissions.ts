import {
  getTurnElapsedSeconds,
  type ConversationTurn,
} from "@/features/projects/conversation-timeline-model";
import type { PermissionRequest, QuestionRequest } from "@/services/aivo";
import { mergeToolCallLists } from "./project-conversation-tool-calls";
import type { domain } from "../../../bridge/go/models";

export function updatePermissionPauseState(
  turns: ConversationTurn[],
  permissions: PermissionRequest[],
  now: number,
) {
  let changed = false;
  const pendingPermissions = permissions.filter(
    (permission) => permission.status === "pending",
  );

  const nextTurns = turns.map((turn) => {
    if (turn.stopped || turn.responseCompletedAt) {
      if (!turn.pauseStartedAt) return turn;
      changed = true;
      return {
        ...turn,
        pauseStartedAt: null,
        pausedMilliseconds:
          (turn.pausedMilliseconds ?? 0) +
          Math.max(0, now - turn.pauseStartedAt),
      };
    }

    const shouldPause = pendingPermissions.some((permission) =>
      permissionMatchesTurn(permission, turn),
    );
    if (shouldPause) {
      if (turn.pauseStartedAt && turn.activityVisible) return turn;
      changed = true;
      return {
        ...turn,
        activityVisible: true,
        pauseStartedAt: turn.pauseStartedAt ?? now,
        thinkingSeconds: getTurnElapsedSeconds(turn, now),
      };
    }

    if (!turn.pauseStartedAt) return turn;
    changed = true;
    const pausedMilliseconds =
      (turn.pausedMilliseconds ?? 0) + Math.max(0, now - turn.pauseStartedAt);
    return {
      ...turn,
      pausedMilliseconds,
      pauseStartedAt: null,
      thinkingSeconds: getTurnElapsedSeconds(
        {
          pausedMilliseconds,
          pauseStartedAt: null,
          startedAt: turn.startedAt,
        },
        now,
      ),
    };
  });

  return changed ? nextTurns : turns;
}

export function mergePendingPermissionToolCalls(
  turns: ConversationTurn[],
  permissions: PermissionRequest[],
) {
  const pendingPermissions = permissions.filter(
    (permission) => permission.status === "pending",
  );
  if (pendingPermissions.length === 0) return turns;

  let changed = false;
  const nextTurns = turns.map((turn, index) => {
    if (turn.stopped || turn.responseCompletedAt) return turn;
    const matchingPermissions = pendingPermissions.filter((permission) =>
      permissionMatchesTurnOrLastRunning(permission, turn, index, turns),
    );
    if (matchingPermissions.length === 0) return turn;

    const missingPermissionToolCalls = matchingPermissions
      .filter((permission) => {
        const toolCallId =
          permission.toolCallId || `permission:${permission.id}`;
        return !turn.toolCalls.some((toolCall) => toolCall.id === toolCallId);
      })
      .map(permissionToolCall);

    if (missingPermissionToolCalls.length === 0 && turn.activityVisible) {
      return turn;
    }

    changed = true;
    return {
      ...turn,
      activityVisible: true,
      toolCalls:
        missingPermissionToolCalls.length === 0
          ? turn.toolCalls
          : mergeToolCallLists(turn.toolCalls, missingPermissionToolCalls),
    };
  });

  return changed ? nextTurns : turns;
}

export function permissionToolCall(
  permission: PermissionRequest,
): domain.ToolCall {
  const now =
    permission.timeUpdated ||
    permission.timeCreated ||
    new Date().toISOString();
  return {
    id: permission.toolCallId || `permission:${permission.id}`,
    sessionId: permission.sessionId || "",
    turnId: permission.turnId || "",
    name: permission.toolName,
    arguments: permission.arguments,
    status: "pending_approval",
    resultSummary: "等待权限审批",
    result: { pendingApprovalId: permission.id },
    timeCreated: permission.timeCreated || now,
    timeUpdated: now,
  } as domain.ToolCall;
}

export function samePermissionRequests(
  a: PermissionRequest[],
  b: PermissionRequest[],
) {
  if (a.length !== b.length) return false;
  return a.every((permission, index) => {
    const other = b[index];
    return (
      other &&
      permission.id === other.id &&
      permission.status === other.status &&
      permission.timeUpdated === other.timeUpdated &&
      permission.turnId === other.turnId &&
      permission.toolCallId === other.toolCallId &&
      permission.toolName === other.toolName &&
      permission.action === other.action
    );
  });
}

export function upsertPermissionRequest(
  requests: PermissionRequest[],
  request: PermissionRequest,
) {
  const existingIndex = requests.findIndex((item) => item.id === request.id);
  if (existingIndex === -1) return [request, ...requests];
  const next = requests.slice();
  next[existingIndex] = request;
  return next;
}

export function sameQuestionRequests(
  a: QuestionRequest[],
  b: QuestionRequest[],
) {
  if (a.length !== b.length) return false;
  return a.every((request, index) => {
    const other = b[index];
    return (
      other &&
      request.id === other.id &&
      request.status === other.status &&
      request.timeUpdated === other.timeUpdated &&
      request.turnId === other.turnId &&
      request.toolCallId === other.toolCallId &&
      request.questions.length === other.questions.length
    );
  });
}

export function upsertQuestionRequest(
  requests: QuestionRequest[],
  request: QuestionRequest,
) {
  const existingIndex = requests.findIndex((item) => item.id === request.id);
  if (existingIndex === -1) return [request, ...requests];
  const next = requests.slice();
  next[existingIndex] = request;
  return next;
}

export function upsertSession(
  sessions: domain.Session[],
  session: domain.Session,
) {
  const existingIndex = sessions.findIndex((item) => item.id === session.id);
  if (existingIndex === -1) {
    return [session, ...sessions];
  }
  const next = sessions.slice();
  next[existingIndex] = session;
  return next;
}

function permissionMatchesTurnOrLastRunning(
  permission: PermissionRequest,
  turn: ConversationTurn,
  index: number,
  turns: ConversationTurn[],
) {
  if (permissionMatchesTurn(permission, turn)) return true;
  if (permission.turnId) return false;
  return (
    index ===
    turns.findLastIndex(
      (candidate) => !candidate.stopped && !candidate.responseCompletedAt,
    )
  );
}

function permissionMatchesTurn(
  permission: PermissionRequest,
  turn: ConversationTurn,
) {
  if (permission.turnId) return permission.turnId === turn.turnId;
  return Boolean(turn.turnId);
}
