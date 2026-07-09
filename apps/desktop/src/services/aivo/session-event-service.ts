import type { domain } from "../../../bridge/go/models";
import { invoke } from "@/services/aivo/invoke";

export type UpdateSessionEventInput = {
  eventId: string;
  content: string;
};

export type DeleteSessionEventInput = {
  eventId: string;
};

export type RetrySessionTurnInput = {
  sessionId?: string;
  turnId: string;
  model?: domain.ModelRef;
  agentMode?: string;
  toolsets?: string[];
  permissionScope?: string;
  reasoningEffort?: string;
  serviceTier?: string;
};

export type GetSessionTurnDiffInput = {
  sessionId?: string;
  turnId: string;
};

export type ApplySessionTurnFileStateInput = {
  sessionId?: string;
  turnId: string;
  toolCallId?: string;
  path?: string;
  targetState: "before" | "after";
};

export type SessionTurnDiffFile = {
  toolCallId: string;
  toolName: string;
  path: string;
  movePath?: string;
  type: string;
  additions?: number;
  deletions?: number;
  diff?: string;
  baseHash?: string;
  currentHash?: string;
  currentFileHash?: string;
  revertible: boolean;
  unrevertible: boolean;
  reason?: string;
  timeUpdated?: string;
};

export type SessionTurnDiff = {
  sessionId: string;
  turnId: string;
  files: SessionTurnDiffFile[];
  diff?: string;
};

export interface SessionEvent {
  id: string;
  sessionId: string;
  turnId?: string;
  type: string;
  role?: string;
  visibility: string;
  content?: string;
  payload?: Record<string, unknown>;
  timeCreated: string;
}

export function listSessionEvents(
  sessionId: string,
  includeNonNormal = false,
  limit = 100,
) {
  return invoke<SessionEvent[]>(
    "ListSessionEvents",
    sessionId,
    includeNonNormal,
    limit,
  );
}

export type SessionEventsCursorResult = {
  events: SessionEvent[];
  nextCursor: string;
};

export function listSessionEventsAfterCursor(input: {
  sessionId: string;
  cursor?: string;
  includeNonNormal?: boolean;
  limit?: number;
}) {
  return invoke<SessionEventsCursorResult>("ListSessionEventsAfterCursor", input);
}

export function updateSessionEvent(input: UpdateSessionEventInput) {
  return invoke<SessionEvent>("UpdateSessionEvent", input);
}

export function deleteSessionEvent(input: DeleteSessionEventInput) {
  return invoke<SessionEvent>("DeleteSessionEvent", input);
}

export function retrySessionTurn(input: RetrySessionTurnInput) {
  return invoke<domain.PreparedSessionTurn>("RetrySessionTurn", input);
}

export function getSessionTurnDiff(input: GetSessionTurnDiffInput) {
  return invoke<SessionTurnDiff>("GetSessionTurnDiff", input);
}

export function applySessionTurnFileState(
  input: ApplySessionTurnFileStateInput,
) {
  return invoke<SessionTurnDiff>("ApplySessionTurnFileState", input);
}
