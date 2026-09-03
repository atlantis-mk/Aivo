import { domain } from "../../bridge/go/models";

export async function listCodexSessions(limit: number): Promise<domain.Session[]> {
  const threads = await window.aivoDesktop.codex.listThreads(limit);
  return threads.map((thread) => new domain.Session({
    id: thread.id,
    type: "coding",
    status: thread.status,
    source: thread.source,
    title: thread.name || thread.preview || "新对话",
    parentSessionId: thread.parentThreadId ?? undefined,
    projectPath: thread.cwd,
    model: thread.model
      ? { providerId: thread.modelProvider, modelId: thread.model }
      : undefined,
    timeCreated: thread.timeCreated,
    timeUpdated: thread.timeUpdated,
  }));
}

export function archiveCodexSession(sessionId: string): Promise<void> {
  return window.aivoDesktop.codex.archiveThread(sessionId);
}

export function listCodexThreadTurns(threadId: string): Promise<CodexThreadTurn[]> {
  return window.aivoDesktop.codex.listThreadTurns(threadId);
}

export function resumeCodexSession(sessionId: string): Promise<void> {
  return window.aivoDesktop.codex.resumeThread(sessionId);
}
