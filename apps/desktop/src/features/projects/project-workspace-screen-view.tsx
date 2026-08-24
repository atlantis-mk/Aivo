import type { ComponentProps } from "react";

import { ProjectWorkspaceDialogs } from "@/features/projects/project-workspace-dialogs";
import { ProjectWorkspaceLeftSidebar } from "@/features/projects/project-workspace-left-sidebar";
import { ProjectColumnShell } from "@/features/projects/project-workspace-layout";
import { ProjectWorkspaceMainContent } from "@/features/projects/project-workspace-main-content";
import {
  ProjectWorkspaceMainTopBar,
  ProjectWorkspaceTopBar,
} from "@/features/projects/project-workspace-shell-bars";

export type ProjectWorkspaceScreenViewProps = {
  dialogs: ComponentProps<typeof ProjectWorkspaceDialogs>;
  leftSidebar: ComponentProps<typeof ProjectWorkspaceLeftSidebar>;
  main: ComponentProps<typeof ProjectWorkspaceMainContent>;
  mainTopBar: ComponentProps<typeof ProjectWorkspaceMainTopBar>;
  topBar: Omit<
    ComponentProps<typeof ProjectWorkspaceTopBar>,
    | "historyContent"
    | "isConversationPinned"
    | "onArchiveConversation"
    | "onTogglePinnedConversation"
    | "repositoryPath"
    | "sessionId"
  >;
};

export function ProjectWorkspaceScreenView({
  dialogs,
  leftSidebar,
  main,
  mainTopBar,
  topBar,
}: ProjectWorkspaceScreenViewProps) {
  return (
    <>
      <ProjectWorkspaceDialogs {...dialogs} />
      <ProjectColumnShell
        mainTopBar={(
          <ProjectWorkspaceTopBar
            {...topBar}
            historyContent={
              <div className="flex max-h-[min(72vh,560px)] flex-col">
                <ProjectWorkspaceLeftSidebar {...leftSidebar} />
              </div>
            }
            isConversationPinned={leftSidebar.pinnedConversationIds.includes(
              leftSidebar.activeConversationId,
            )}
            onArchiveConversation={() =>
              leftSidebar.onArchiveConversation(
                leftSidebar.activeConversationId,
              )
            }
            onTogglePinnedConversation={() =>
              leftSidebar.onTogglePinnedConversation(
                leftSidebar.activeConversationId,
              )
            }
            repositoryPath={mainTopBar.repositoryPath}
            sessionId={mainTopBar.sessionId}
          />
        )}
        main={<ProjectWorkspaceMainContent {...main} />}
      />
    </>
  );
}
