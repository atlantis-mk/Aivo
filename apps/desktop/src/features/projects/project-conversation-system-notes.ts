import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";
import type { domain } from "../../../bridge/go/models";

export function attachSystemNotesToTurns(
  turns: ConversationTurn[],
  events: domain.SessionEvent[],
): ConversationTurn[] {
  const notesByTurnId = new Map<string, domain.SessionEvent[]>();
  for (const event of events) {
    if (!isVisibleTurnSystemNote(event)) continue;
    const turnId = event.turnId;
    if (!turnId) continue;
    const notes = notesByTurnId.get(turnId) ?? [];
    notes.push(event);
    notesByTurnId.set(turnId, notes);
  }
  if (notesByTurnId.size === 0) return turns;
  return turns.map((turn) => {
    if (!turn.turnId) return turn;
    const notes = notesByTurnId.get(turn.turnId);
    if (!notes?.length) return turn;
    return {
      ...turn,
      systemNotes: notes.map(conversationSystemNoteFromEvent),
    };
  });
}

export function mergeSystemNoteEvent(
  turns: ConversationTurn[],
  event: domain.SessionEvent,
): ConversationTurn[] {
  if (!isVisibleTurnSystemNote(event)) return turns;
  let changed = false;
  const nextTurns = turns.map((turn) => {
    if (turn.turnId !== event.turnId) return turn;
    const notes = [...(turn.systemNotes ?? [])];
    const nextNote = conversationSystemNoteFromEvent(event);
    const index = notes.findIndex((note) => note.id === event.id);
    if (index >= 0) {
      notes[index] = nextNote;
    } else {
      notes.push(nextNote);
    }
    changed = true;
    return { ...turn, systemNotes: notes };
  });
  return changed ? nextTurns : turns;
}

function isVisibleTurnSystemNote(event: domain.SessionEvent) {
  return !(
    (event.visibility && event.visibility !== "normal") ||
    event.type !== "system_note" ||
    !event.turnId ||
    !event.content?.trim() ||
    event.content.trim() === "User stopped generation"
  );
}

function conversationSystemNoteFromEvent(note: domain.SessionEvent) {
  return {
    content: note.content ?? "",
    id: note.id,
    payload: note.payload,
    timeCreated: note.timeCreated,
  };
}
