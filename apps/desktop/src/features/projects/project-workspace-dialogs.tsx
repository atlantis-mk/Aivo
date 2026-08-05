import type { Dispatch, SetStateAction } from "react";

import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";
import {
  ToolActivationDialog,
} from "@/features/projects/project-tool-activation-dialog";
import { usedToolNamesFromTurns } from "@/features/projects/project-tool-activation-model";

export function ProjectWorkspaceDialogs({
  activeSessionId,
  pendingActiveToolNames,
  onPendingActiveToolNamesChange,
  onToolActivationOpenChange,
  toolActivationDialogOpen,
  turns,
  workspaceRoot,
}: {
  activeSessionId: string;
  pendingActiveToolNames: string[];
  onPendingActiveToolNamesChange: Dispatch<SetStateAction<string[]>>;
  onToolActivationOpenChange: (open: boolean) => void;
  toolActivationDialogOpen: boolean;
  turns: ConversationTurn[];
  workspaceRoot: string;
}) {
  return (
    <ToolActivationDialog
      activeSessionId={activeSessionId}
      pendingActiveToolNames={pendingActiveToolNames}
      onPendingActiveToolNamesChange={onPendingActiveToolNamesChange}
      onOpenChange={onToolActivationOpenChange}
      open={toolActivationDialogOpen}
      usedToolNames={usedToolNamesFromTurns(turns)}
      workspaceRoot={workspaceRoot}
    />
  );
}
