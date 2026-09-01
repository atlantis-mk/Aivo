import { memo, useMemo, type RefObject } from "react";

import { cn } from "@/lib/utils";
import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";
import { constructConversationTimelineRows } from "@/features/projects/conversation-timeline-row-model";
import type { ToolCallActivity } from "@/features/projects/conversation-timeline-tool-model";
import { ConversationTimelineRowView } from "./conversation-timeline-row-view";
import type { ConversationTimelineActions } from "./conversation-timeline-types";
import type { AgentRun } from "@/services/aivo";

export const SubmittedPromptContent = memo(function SubmittedPromptContent({
  agentRuns = [],
  contentRef,
  onDeleteAssistantMessage,
  onDeleteTurn,
  onEditUserMessage,
  onOpenSession,
  onOpenToolActivity,
  onRetryTurn,
  revealFromHistory,
  reserveFloatingControls,
  reservePermissionDock,
  turns,
  workspaceRoot,
}: {
  agentRuns?: AgentRun[];
  contentRef: RefObject<HTMLDivElement | null>;
  onDeleteAssistantMessage?: (turn: ConversationTurn) => void;
  onDeleteTurn?: (turn: ConversationTurn) => void;
  onEditUserMessage?: (turn: ConversationTurn) => void;
  onOpenSession?: (sessionId: string) => void;
  onOpenToolActivity?: (activity: ToolCallActivity) => void;
  onRetryTurn?: (turn: ConversationTurn) => void;
  revealFromHistory: boolean;
  reserveFloatingControls: boolean;
  reservePermissionDock: boolean;
  turns: ConversationTurn[];
  workspaceRoot: string;
}) {
  const rows = useMemo(() => constructConversationTimelineRows(turns), [turns]);
  const actions: ConversationTimelineActions = useMemo(
    () => ({
      onDeleteAssistantMessage,
      onDeleteTurn,
      onEditUserMessage,
      onRetryTurn,
    }),
    [
      onDeleteAssistantMessage,
      onDeleteTurn,
      onEditUserMessage,
      onRetryTurn,
    ],
  );

  return (
    <div
      className={cn(
        "mx-auto flex w-[calc(100%-2rem)] max-w-[680px] flex-col px-0 pt-12 transition-transform ease-out sm:w-[calc(100%-48px)]",
        reservePermissionDock
          ? "pb-[19rem]"
          : reserveFloatingControls
            ? "pb-[calc(var(--conversation-bottom-height)+6rem)]"
            : "pb-[calc(var(--conversation-bottom-height)+3rem)]",
        revealFromHistory
          ? "animate-in fade-in duration-200 [&_.animate-in]:animate-none"
          : "animate-in fade-in slide-in-from-bottom-3 duration-500",
      )}
      ref={contentRef}
    >
      {rows.map((row) => (
        <ConversationTimelineRowView
          actions={actions}
          agentRuns={agentRuns}
          key={row.key}
          onOpenSession={onOpenSession}
          onOpenToolActivity={onOpenToolActivity}
          row={row}
          workspaceRoot={workspaceRoot}
        />
      ))}
    </div>
  );
});
