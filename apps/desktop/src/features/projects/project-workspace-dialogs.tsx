import type { Dispatch, SetStateAction } from "react";

import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";
import {
  ToolActivationDialog,
} from "@/features/projects/project-tool-activation-dialog";
import { ExtensionSettingsDialog } from "@/features/projects/extension-settings-dialog";
import { usedToolNamesFromTurns } from "@/features/projects/project-tool-activation-model";

export function ProjectWorkspaceDialogs({
  activeSessionId,
  pendingActiveToolNames,
  onPendingActiveToolNamesChange,
  onExtensionSettingsDrawerOpenChange,
  onToolActivationOpenChange,
  toolActivationDialogOpen,
  extensionSettingsDrawerOpen,
  turns,
  workspaceRoot,
}: {
  activeSessionId: string;
  pendingActiveToolNames: string[];
  onPendingActiveToolNamesChange: Dispatch<SetStateAction<string[]>>;
  onExtensionSettingsDrawerOpenChange: (open: boolean) => void;
  onToolActivationOpenChange: (open: boolean) => void;
  toolActivationDialogOpen: boolean;
  extensionSettingsDrawerOpen: boolean;
  turns: ConversationTurn[];
  workspaceRoot: string;
}) {
  return (
    <>
      <ToolActivationDialog
        activeSessionId={activeSessionId}
        pendingActiveToolNames={pendingActiveToolNames}
        onPendingActiveToolNamesChange={onPendingActiveToolNamesChange}
        onOpenChange={onToolActivationOpenChange}
        open={toolActivationDialogOpen}
        usedToolNames={usedToolNamesFromTurns(turns)}
        workspaceRoot={workspaceRoot}
      />
      <ExtensionSettingsDialog
        onOpenChange={onExtensionSettingsDrawerOpenChange}
        open={extensionSettingsDrawerOpen}
        sessionId={activeSessionId}
        workspaceRoot={workspaceRoot}
      />
    </>
  );
}
