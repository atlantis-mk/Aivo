import { Plug } from "lucide-react";

import type { ProjectSidebarState } from "@/features/projects/project-workspace-layout-model";
import {
  ProjectMainTopBar,
  ProjectTopBar,
} from "@/features/projects/project-workspace-top-bars";
import type { ProjectWorkspacePage } from "@/features/projects/project-workspace-derived-state";

function getProjectWorkspacePageHeader(activeProjectPage: ProjectWorkspacePage) {
  if (activeProjectPage !== "plugins") {
    return { pageIcon: undefined, pageTitle: undefined };
  }

  return {
    pageIcon: (
      <Plug
        aria-hidden="true"
        className="size-4 shrink-0 text-muted-foreground"
      />
    ),
    pageTitle: "Plugins & MCP",
  };
}

export function ProjectWorkspaceTopBar({
  activeProjectPage,
  canShowTerminalPanel,
  conversationTitle,
  hasConversation,
  leftSidebarState,
  onNewPage,
  onToggleLeftSidebar,
}: {
  activeProjectPage: ProjectWorkspacePage;
  canShowTerminalPanel: boolean;
  conversationTitle: string;
  hasConversation: boolean;
  leftSidebarState: ProjectSidebarState;
  onNewPage: () => void;
  onToggleLeftSidebar: () => void;
}) {
  const { pageIcon, pageTitle } =
    getProjectWorkspacePageHeader(activeProjectPage);

  return (
    <ProjectTopBar
      canShowTerminalPanel={canShowTerminalPanel}
      conversationTitle={conversationTitle}
      hasConversation={hasConversation}
      leftSidebarState={leftSidebarState}
      onNewPage={onNewPage}
      onToggleLeftSidebar={onToggleLeftSidebar}
      pageIcon={pageIcon}
      pageTitle={pageTitle}
      showTerminalButton={activeProjectPage === "chat"}
    />
  );
}

export function ProjectWorkspaceMainTopBar({
  activeProjectPage,
  canToggleEnvironmentSummaryPanel,
  conversationTitle,
  hasConversation,
  isPinnedSummaryOpen,
  isRightSidebarOpen,
  onTogglePinnedSummary,
}: {
  activeProjectPage: ProjectWorkspacePage;
  canToggleEnvironmentSummaryPanel: boolean;
  conversationTitle: string;
  hasConversation: boolean;
  isPinnedSummaryOpen: boolean;
  isRightSidebarOpen: boolean;
  onTogglePinnedSummary: () => void;
}) {
  const { pageIcon, pageTitle } =
    getProjectWorkspacePageHeader(activeProjectPage);

  return (
    <ProjectMainTopBar
      conversationTitle={conversationTitle}
      hasConversation={hasConversation}
      isLayoutPanelOpen={isPinnedSummaryOpen}
      onToggleLayoutPanel={onTogglePinnedSummary}
      pageIcon={pageIcon}
      pageTitle={pageTitle}
      rightOpen={activeProjectPage === "chat" && isRightSidebarOpen}
      showLayoutButton={canToggleEnvironmentSummaryPanel}
      showTerminalButton={activeProjectPage === "chat"}
    />
  );
}
