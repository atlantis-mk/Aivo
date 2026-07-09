import type { ConversationAssistantTextPart } from "@/features/projects/conversation-timeline-model";

export function appendAssistantText(current: string, next: string) {
  const trimmedNext = next.trim();
  if (!trimmedNext) return current;
  if (!current.trim()) return next;
  return `${current.trimEnd()}\n\n${trimmedNext}`;
}

export function appendAssistantPreamblePart(
  current: ConversationAssistantTextPart[] | undefined,
  next: ConversationAssistantTextPart,
) {
  const trimmedText = next.text.trim();
  if (!trimmedText) return current ?? [];
  const parts = current ?? [];
  const existingIndex = parts.findIndex((part) => part.id === next.id);
  if (existingIndex >= 0) {
    const existing = parts[existingIndex];
    const updated = {
      ...existing,
      text: next.text,
      timeCreated: next.timeCreated ?? existing.timeCreated,
    };
    if (
      existing.text === updated.text &&
      existing.timeCreated === updated.timeCreated
    ) {
      return parts;
    }
    return parts.map((part, index) =>
      index === existingIndex ? updated : part,
    );
  }
  return [...parts, next];
}

export function stripSessionAttachmentSummary(text: string) {
  return text.replace(/\n{2,}附件:\n(?:- .+(?:\n|$))+$/u, "").trimEnd();
}
