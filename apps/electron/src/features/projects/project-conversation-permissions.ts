import type { QuestionRequest } from "@/services/aivo";
import type { domain } from "@/types/codex-domain";

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
