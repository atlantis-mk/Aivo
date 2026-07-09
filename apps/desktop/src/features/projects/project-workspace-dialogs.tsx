import type { Dispatch, SetStateAction } from "react";

import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";
import {
  ToolActivationDialog,
} from "@/features/projects/project-tool-activation-dialog";
import { usedToolNamesFromTurns } from "@/features/projects/project-tool-activation-model";

export function ProjectWorkspaceDialogs({
  activeSessionId,
  defaultActiveToolNames,
  onDefaultActiveToolNamesChange,
  onToolActivationOpenChange,
  toolActivationDialogOpen,
  turns,
  workspaceRoot,
}: {
  activeSessionId: string;
  defaultActiveToolNames: string[];
  onDefaultActiveToolNamesChange: Dispatch<SetStateAction<string[]>>;
  onToolActivationOpenChange: (open: boolean) => void;
  toolActivationDialogOpen: boolean;
  turns: ConversationTurn[];
  workspaceRoot: string;
}) {
  return (
    <ToolActivationDialog
      activeSessionId={activeSessionId}
      defaultActiveToolNames={defaultActiveToolNames}
      onDefaultActiveToolNamesChange={onDefaultActiveToolNamesChange}
      onOpenChange={onToolActivationOpenChange}
      open={toolActivationDialogOpen}
      usedToolNames={usedToolNamesFromTurns(turns)}
      workspaceRoot={workspaceRoot}
    />
  );
}
