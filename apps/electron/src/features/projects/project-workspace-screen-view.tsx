import { useState, type ComponentProps } from "react";

import { ProjectWorkspaceDialogs } from "@/features/projects/project-workspace-dialogs";
import { ProjectWorkspaceAppSidebar } from "@/features/projects/project-workspace-app-sidebar";
import { ProjectSidebarToggleButton } from "@/features/projects/project-workspace-top-bars";
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
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);

  return (
    <>
      <ProjectWorkspaceDialogs {...dialogs} />
      <div className="relative flex h-dvh w-screen min-h-0 overflow-hidden bg-background text-foreground">
        <ProjectWorkspaceAppSidebar
          {...leftSidebar}
          isCollapsed={isSidebarCollapsed}
        />
        <ProjectColumnShell
          mainTopBar={
            <ProjectWorkspaceTopBar
              {...topBar}
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
              isSidebarCollapsed={isSidebarCollapsed}
              onToggleSidebar={() =>
                setIsSidebarCollapsed((collapsed) => !collapsed)
              }
              onNewConversation={leftSidebar.onNewConversation}
              repositoryPath={mainTopBar.repositoryPath}
              sessionId={mainTopBar.sessionId}
            />
          }
          main={<ProjectWorkspaceMainContent {...main} />}
        />
      </div>
    </>
  );
}
