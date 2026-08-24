import type { domain } from "../../../bridge/go/models";
import { invoke } from "@/services/aivo/invoke";

export function createSession(input: domain.CreateSessionRequest) {
  return invoke<domain.Session>("CreateSession", input);
}

export function listSessions(limit: number) {
  return invoke<domain.Session[]>("ListSessions", {
    type: "coding",
    status: "active",
    limit,
  } as domain.ListSessionsRequest);
}

export function listSessionToolCalls(sessionId: string) {
  return invoke<domain.ToolCall[]>("ListSessionToolCalls", sessionId);
}

export type SessionRuntimeStats = {
  turns: number;
  steps: number;
  llmMs: number;
  ttftMs?: number;
  ttftSteps?: number;
  decodeMs?: number;
  decodeTokens?: number;
  inputTokens?: number;
  outputTokens?: number;
  cacheReadTokens?: number;
  cacheReadAvailable?: boolean;
};

export function getSessionRuntimeStats(sessionId: string) {
  return invoke<SessionRuntimeStats>("GetSessionRuntimeStats", sessionId);
}

export type ReplaySessionToolCallInput = {
  sessionId?: string;
  toolCallId: string;
  permissionScope?: string;
};

export function replaySessionToolCall(input: ReplaySessionToolCallInput) {
  return invoke<domain.ToolCall>("ReplaySessionToolCall", input);
}

export type RetainedOutputReadInput = {
  ref: string;
  offset?: number;
  limit?: number;
};

export type RetainedOutputReadResult = {
  ref: string;
  content: string;
  offset: number;
  nextOffset: number;
  size: number;
  truncated?: boolean;
};

export function readRetainedOutput(input: RetainedOutputReadInput) {
  return invoke<RetainedOutputReadResult>("ReadRetainedOutput", input);
}

export function listSessionTurns(sessionId: string, limit = 100) {
  return invoke<domain.Turn[]>("ListSessionTurns", sessionId, limit);
}

export function archiveSession(sessionId: string) {
  return invoke<domain.Session>("ArchiveSession", sessionId);
}

export function submitSessionMessage(
  input: domain.SubmitSessionMessageRequest,
) {
  return invoke<domain.PreparedSessionTurn>(
    "SubmitSessionMessageStreaming",
    input,
  );
}

export function cancelSessionTurn(input: domain.CancelTurnRequest) {
  return invoke<domain.Turn>("CancelSessionTurn", input);
}
