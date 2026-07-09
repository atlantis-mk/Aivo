import type { RefObject } from "react";

import { ScrollArea } from "@/components/ui/scroll-area";
import { SubmittedPromptContent } from "@/features/projects/conversation-timeline";
import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";
import type { AgentRun } from "@/services/aivo";

type ProjectConversationViewportHandlers = {
  onDeleteAssistantMessage: (turn: ConversationTurn) => void;
  onDeleteTurn: (turn: ConversationTurn) => void;
  onEditUserMessage: (turn: ConversationTurn) => void;
  onOpenSession: (sessionId: string) => void;
  onRetryTurn: (turn: ConversationTurn) => void;
};

export function ProjectConversationViewport({
  agentRuns,
  contentRef,
  dockPinnedSummary,
  handlers,
  hasTurns,
  reservePermissionDock,
  revealFromHistory,
  rootRef,
  showConversationLayout,
  turns,
}: {
  agentRuns: AgentRun[];
  contentRef: RefObject<HTMLDivElement | null>;
  dockPinnedSummary: boolean;
  handlers: ProjectConversationViewportHandlers;
  hasTurns: boolean;
  reservePermissionDock: boolean;
  revealFromHistory: boolean;
  rootRef: RefObject<HTMLDivElement | null>;
  showConversationLayout: boolean;
  turns: ConversationTurn[];
}) {
  if (!showConversationLayout) return null;

  return (
    <div className="absolute inset-0 z-0" ref={rootRef}>
      <ScrollArea className="h-full [&_[data-slot=scroll-area-scrollbar]]:mt-2">
        {hasTurns ? (
          <SubmittedPromptContent
            agentRuns={agentRuns}
            contentRef={contentRef}
            dockPinnedSummary={dockPinnedSummary}
            onOpenSession={handlers.onOpenSession}
            onDeleteAssistantMessage={handlers.onDeleteAssistantMessage}
            onDeleteTurn={handlers.onDeleteTurn}
            onEditUserMessage={handlers.onEditUserMessage}
            onRetryTurn={handlers.onRetryTurn}
            revealFromHistory={revealFromHistory}
            reservePermissionDock={reservePermissionDock}
            turns={turns}
          />
        ) : null}
      </ScrollArea>
    </div>
  );
}
