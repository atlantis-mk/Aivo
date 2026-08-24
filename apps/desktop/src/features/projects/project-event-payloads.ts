import type { ShellOutputPayload } from "@/features/projects/tool-activity-model";
import type {
  PermissionRequest,
  QuestionRequest,
  TodoItem,
} from "@/services/aivo";
import type { domain } from "../../../bridge/go/models";

export function normalizeSessionUpdatedPayload(payloads: unknown[]) {
  const payload = normalizeRecordPayload(payloads);
  return {
    sessionId: typeof payload?.sessionId === "string" ? payload.sessionId : "",
    session: normalizeSessionObject(payload?.session),
  };
}

export function normalizeTurnUpdatedPayload(payloads: unknown[]) {
  const payload = normalizeRecordPayload(payloads);
  return normalizeTurnObject(payload?.turn);
}

export function normalizeToolCallUpdatedPayload(payloads: unknown[]) {
  const payload = normalizeRecordPayload(payloads);
  return normalizeToolCallObject(payload?.toolCall);
}

export function normalizeSessionEventUpdatedPayload(payloads: unknown[]) {
  const payload = normalizeRecordPayload(payloads);
  return normalizeSessionEventObject(payload?.event);
}

export function normalizeShellOutputPayload(
  payloads: unknown[],
): ShellOutputPayload {
  const payload = normalizeRecordPayload(payloads);
  return {
    sessionId: typeof payload?.sessionId === "string" ? payload.sessionId : "",
    turnId: typeof payload?.turnId === "string" ? payload.turnId : "",
    toolCallId:
      typeof payload?.toolCallId === "string" ? payload.toolCallId : "",
    processRef:
      typeof payload?.processRef === "string" ? payload.processRef : "",
    stream: typeof payload?.stream === "string" ? payload.stream : "",
    chunk: typeof payload?.chunk === "string" ? payload.chunk : "",
    sequence:
      typeof payload?.sequence === "number" ? payload.sequence : undefined,
    cursor: typeof payload?.cursor === "number" ? payload.cursor : undefined,
    status:
      payload?.status === "running" || payload?.status === "waiting_input" || payload?.status === "exited"
        ? payload.status
        : undefined,
    truncated:
      typeof payload?.truncated === "boolean"
        ? payload.truncated
        : undefined,
    timeCreated:
      typeof payload?.timeCreated === "string" ? payload.timeCreated : "",
  };
}

export function normalizePermissionEventPayload(
  payloads: unknown[],
): PermissionRequest | null {
  const payload = normalizeRecordPayload(payloads);
  return (
    normalizePermissionRequestObject(payload?.permission) ??
    normalizePermissionRequestObject(payload)
  );
}

export function normalizeQuestionEventPayload(
  payloads: unknown[],
): QuestionRequest | null {
  const payload = normalizeRecordPayload(payloads);
  return (
    normalizeQuestionRequestObject(payload?.question) ??
    normalizeQuestionRequestObject(payload)
  );
}

export function normalizeTodoItemsUpdatedPayload(
  payloads: unknown[],
): { sessionId: string; projectPath: string; items: TodoItem[] } | null {
  const payload = normalizeRecordPayload(payloads);
  if (!payload || typeof payload.sessionId !== "string") return null;
  if (!Array.isArray(payload.items)) return null;
  return {
    sessionId: payload.sessionId,
    projectPath:
      typeof payload.projectPath === "string" ? payload.projectPath : "",
    items: payload.items as TodoItem[],
  };
}

export function normalizeAssistantDeltaPayload(payloads: unknown[]) {
  const first = payloads[0];
  const payload =
    normalizeAssistantDeltaObject(first) ??
    normalizeAssistantDeltaObject(payloads);
  return {
    sessionId: payload?.sessionId ?? "",
    turnId: payload?.turnId ?? "",
    delta: payload?.delta ?? "",
  };
}

function normalizeSessionObject(value: unknown): domain.Session | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  const record = value as Record<string, unknown>;
  if (typeof record.id === "string") return record as unknown as domain.Session;
  return null;
}

function normalizeTurnObject(value: unknown): domain.Turn | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  const record = value as Record<string, unknown>;
  if (typeof record.id === "string" && typeof record.sessionId === "string") {
    return record as unknown as domain.Turn;
  }
  return null;
}

function normalizeToolCallObject(value: unknown): domain.ToolCall | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  const record = value as Record<string, unknown>;
  if (typeof record.id === "string" && typeof record.sessionId === "string") {
    return record as unknown as domain.ToolCall;
  }
  return null;
}

function normalizeSessionEventObject(value: unknown): domain.SessionEvent | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  const record = value as Record<string, unknown>;
  if (
    typeof record.id === "string" &&
    typeof record.sessionId === "string" &&
    typeof record.type === "string"
  ) {
    return record as unknown as domain.SessionEvent;
  }
  return null;
}

function normalizeRecordPayload(
  payloads: unknown[],
): Record<string, unknown> | null {
  const first = payloads[0];
  if (first && typeof first === "object" && !Array.isArray(first)) {
    return first as Record<string, unknown>;
  }
  if (Array.isArray(first) && first[0] && typeof first[0] === "object") {
    return first[0] as Record<string, unknown>;
  }
  return null;
}

function normalizePermissionRequestObject(
  value: unknown,
): PermissionRequest | null {
  if (!value || typeof value !== "object") return null;
  if (Array.isArray(value)) return normalizePermissionRequestObject(value[0]);
  const record = value as Record<string, unknown>;
  if (typeof record.id === "string" && typeof record.toolName === "string") {
    return record as PermissionRequest;
  }
  return (
    normalizePermissionRequestObject(record.data) ??
    normalizePermissionRequestObject(record.properties)
  );
}

function normalizeQuestionRequestObject(value: unknown): QuestionRequest | null {
  if (!value || typeof value !== "object") return null;
  if (Array.isArray(value)) return normalizeQuestionRequestObject(value[0]);
  const record = value as Record<string, unknown>;
  if (typeof record.id === "string" && Array.isArray(record.questions)) {
    return record as QuestionRequest;
  }
  return (
    normalizeQuestionRequestObject(record.data) ??
    normalizeQuestionRequestObject(record.properties)
  );
}

function normalizeAssistantDeltaObject(
  value: unknown,
): { sessionId?: string; turnId?: string; delta?: string } | null {
  if (!value || typeof value !== "object") return null;
  if (Array.isArray(value)) return normalizeAssistantDeltaObject(value[0]);
  const record = value as Record<string, unknown>;
  if (
    typeof record.sessionId === "string" ||
    typeof record.delta === "string"
  ) {
    return {
      sessionId:
        typeof record.sessionId === "string" ? record.sessionId : undefined,
      turnId: typeof record.turnId === "string" ? record.turnId : undefined,
      delta: typeof record.delta === "string" ? record.delta : undefined,
    };
  }
  return (
    normalizeAssistantDeltaObject(record.data) ??
    normalizeAssistantDeltaObject(record.properties)
  );
}
