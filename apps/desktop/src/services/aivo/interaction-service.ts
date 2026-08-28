import type { domain } from "../../../bridge/go/models";
import { invoke } from "@/services/aivo/invoke";

export type PermissionRequest = {
  id: string;
  sessionId?: string;
  turnId?: string;
  toolCallId?: string;
  toolName: string;
  action: string;
  paths?: string[];
  arguments?: Record<string, unknown>;
  status: string;
  remember?: boolean;
  reason?: string;
  timeCreated: string;
  timeUpdated: string;
};

export type QuestionOption = {
  label: string;
  description?: string;
};

export type QuestionPrompt = {
  id?: string;
  header?: string;
  question: string;
  options?: QuestionOption[];
  multiple?: boolean;
};

export type QuestionRequest = {
  id: string;
  sessionId?: string;
  turnId?: string;
  toolCallId?: string;
  toolName: string;
  questions: QuestionPrompt[];
  answers?: string[][];
  status: string;
  reason?: string;
  arguments?: Record<string, unknown>;
  timeCreated: string;
  timeUpdated: string;
};

export type PermissionMode = "request_approval" | "full_access";

export type PermissionModeState = {
  sessionId?: string;
  workspaceRoot?: string;
  mode: PermissionMode;
  timeUpdated?: string;
};

export function listPermissionRequests(sessionId: string, status = "pending") {
  return invoke<PermissionRequest[]>(
    "ListPermissionRequests",
    sessionId,
    status,
  );
}

export function getPermissionMode(sessionId: string) {
  return invoke<PermissionModeState>("GetPermissionMode", sessionId);
}

export function getCodingContext(sessionId: string) {
  return invoke<domain.CodingContext>("GetCodingContext", sessionId);
}

export function setPermissionMode(sessionId: string, mode: PermissionMode) {
  return invoke<PermissionModeState>("SetPermissionMode", { sessionId, mode });
}

export function approvePermissionRequest(requestId: string, remember = false) {
  return invoke<PermissionRequest>("ApprovePermissionRequest", {
    requestId,
    remember,
  });
}

export function denyPermissionRequest(
  requestId: string,
  remember = false,
  reason = "",
) {
  return invoke<PermissionRequest>("DenyPermissionRequest", {
    requestId,
    remember,
    reason,
  });
}

export function listQuestionRequests(sessionId: string, status = "pending") {
  return invoke<QuestionRequest[]>("ListQuestionRequests", sessionId, status);
}

export function replyQuestionRequest(requestId: string, answers: string[][]) {
  return invoke<QuestionRequest>("ReplyQuestionRequest", {
    requestId,
    answers,
  });
}

export function rejectQuestionRequest(requestId: string, reason = "") {
  return invoke<QuestionRequest>("RejectQuestionRequest", {
    requestId,
    reason,
  });
}
