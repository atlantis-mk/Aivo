import { type ReactNode } from "react";

import type { ConversationSystemNote } from "@/features/projects/conversation-timeline-model";

export function TimelineRowFrame({
  children,
  role,
  turnId,
}: {
  children: ReactNode;
  role: "assistant" | "user";
  turnId: string;
}) {
  return (
    <div
      className="aivo-timeline-row"
      data-timeline-role={role}
      data-turn-id={turnId}
    >
      {children}
    </div>
  );
}

export function SystemNoteRow({ note }: { note: ConversationSystemNote }) {
  return (
    <div className="my-2 flex justify-center">
      <div className="max-w-[min(90%,34rem)] rounded-md border border-border/70 bg-muted/35 px-3 py-2 text-center text-xs text-muted-foreground">
        {note.content}
      </div>
    </div>
  );
}
