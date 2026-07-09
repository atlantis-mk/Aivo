import type { ComponentProps } from "react";

import { TerminalDockProvider } from "@/features/projects/terminal/terminal-dock";
import { ProjectWorkspaceBottomPanel } from "@/features/projects/project-workspace-bottom-panel";
import { ProjectWorkspaceDialogs } from "@/features/projects/project-workspace-dialogs";
import { ProjectWorkspaceLeftSidebar } from "@/features/projects/project-workspace-left-sidebar";
import { ProjectColumnShell } from "@/features/projects/project-workspace-layout";
import { ProjectWorkspaceMainContent } from "@/features/projects/project-workspace-main-content";
import {
  ProjectWorkspaceMainTopBar,
  ProjectWorkspaceTopBar,
} from "@/features/projects/project-workspace-shell-bars";
import { ProjectWorkspaceRightSidebar } from "@/features/projects/project-workspace-right-sidebar";
import { SHOULD_MOUNT_TOOL_ACTIVITY_SIDEBAR } from "@/features/projects/project-workspace-state-model";

export type ProjectWorkspaceScreenViewProps = {
  bottomPanel: Omit<
    ComponentProps<typeof ProjectWorkspaceBottomPanel>,
    "height"
  >;
  dialogs: ComponentProps<typeof ProjectWorkspaceDialogs>;
  leftSidebar: ComponentProps<typeof ProjectWorkspaceLeftSidebar>;
  main: ComponentProps<typeof ProjectWorkspaceMainContent>;
  mainTopBar: ComponentProps<typeof ProjectWorkspaceMainTopBar>;
  onRightOpenChange: (open: boolean) => void;
  rightOpen: boolean;
  rightSidebar: Omit<ComponentProps<typeof ProjectWorkspaceRightSidebar>, "enabled">;
  topBar: Omit<
    ComponentProps<typeof ProjectWorkspaceTopBar>,
    "leftSidebarState" | "onToggleLeftSidebar"
  >;
};

export function ProjectWorkspaceScreenView({
  bottomPanel,
  dialogs,
  leftSidebar,
  main,
  mainTopBar,
  onRightOpenChange,
  rightOpen,
  rightSidebar,
  topBar,
}: ProjectWorkspaceScreenViewProps) {
  return (
    <TerminalDockProvider>
      <ProjectWorkspaceDialogs {...dialogs} />
      <ProjectColumnShell
        bottomPanel={(bottomHeight) => (
          <ProjectWorkspaceBottomPanel
            {...bottomPanel}
            height={bottomHeight}
          />
        )}
        leftSidebar={<ProjectWorkspaceLeftSidebar {...leftSidebar} />}
        topBar={({ leftSidebarState, onToggleLeftSidebar }) => (
          <ProjectWorkspaceTopBar
            {...topBar}
            leftSidebarState={leftSidebarState}
            onToggleLeftSidebar={onToggleLeftSidebar}
          />
        )}
        mainTopBar={<ProjectWorkspaceMainTopBar {...mainTopBar} />}
        main={<ProjectWorkspaceMainContent {...main} />}
        rightOpen={rightOpen}
        onRightOpenChange={onRightOpenChange}
        rightSidebar={
          <ProjectWorkspaceRightSidebar
            {...rightSidebar}
            enabled={SHOULD_MOUNT_TOOL_ACTIVITY_SIDEBAR}
          />
        }
        showTerminalButton={rightSidebar.activeProjectPage === "chat"}
      />
    </TerminalDockProvider>
  );
}
