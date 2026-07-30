import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";
import type { domain } from "../../../bridge/go/models";

export function attachSystemNotesToTurns(
  turns: ConversationTurn[],
  events: domain.SessionEvent[],
): ConversationTurn[] {
  const notesByTurnId = new Map<string, domain.SessionEvent[]>();
  for (const event of events) {
    if (
      (event.visibility && event.visibility !== "normal") ||
      event.type !== "system_note" ||
      !event.turnId ||
      !event.content?.trim() ||
      event.content.trim() === "User stopped generation"
    ) {
      continue;
    }
    const notes = notesByTurnId.get(event.turnId) ?? [];
    notes.push(event);
    notesByTurnId.set(event.turnId, notes);
  }
  if (notesByTurnId.size === 0) return turns;
  return turns.map((turn) => {
    if (!turn.turnId) return turn;
    const notes = notesByTurnId.get(turn.turnId);
    if (!notes?.length) return turn;
    return {
      ...turn,
      systemNotes: notes.map((note) => ({
        content: note.content ?? "",
        id: note.id,
        timeCreated: note.timeCreated,
      })),
    };
  });
}
