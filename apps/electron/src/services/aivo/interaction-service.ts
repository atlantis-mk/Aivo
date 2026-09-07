import type { domain } from "@/types/codex-domain";
import { invoke } from "@/services/aivo/invoke";

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

export function getCodingContext(sessionId: string) {
  return invoke<domain.CodingContext>("GetCodingContext", sessionId);
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
