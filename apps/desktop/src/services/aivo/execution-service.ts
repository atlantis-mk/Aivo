import type { domain } from "../../../bridge/go/models";
import { invoke } from "@/services/aivo/invoke";

export type SessionExecutionState = {
  id: string;
  sessionId: string;
  turnId?: string;
  status: "idle" | "running" | "interrupted" | "failed" | "compacting";
  reason?: string;
  lastEventId?: string;
  pendingInputIds?: string[];
  metadata?: Record<string, unknown>;
  timeCreated: string;
  timeUpdated: string;
};

export type CompactSessionContextResult = {
  state: SessionExecutionState;
  summary: domain.SessionSummary;
  context: domain.BuildSessionContextResult;
  compactedEventId?: string;
};

export function getSessionExecutionState(sessionId: string) {
  return invoke<SessionExecutionState>("GetSessionExecutionState", sessionId);
}

export function interruptSessionExecution(sessionId: string, reason = "") {
  return invoke<SessionExecutionState>("InterruptSessionExecution", {
    sessionId,
    reason,
  });
}

export function resumeSessionExecution(sessionId: string) {
  return invoke<SessionExecutionState>("ResumeSessionExecution", {
    sessionId,
  });
}

export function compactSessionContext(sessionId: string, characterBudget = 6000) {
  return invoke<CompactSessionContextResult>("CompactSessionContext", {
    sessionId,
    characterBudget,
  });
}
