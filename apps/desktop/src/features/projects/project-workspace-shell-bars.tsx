import type { ReactNode } from "react";
import { Plug } from "lucide-react";

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
  historyContent,
  isConversationPinned,
  isPinnedSummaryOpen,
  onArchiveConversation,
  onNewPage,
  onTogglePinnedConversation,
  onTogglePinnedSummary,
  repositoryPath,
  sessionId,
}: {
  activeProjectPage: ProjectWorkspacePage;
  canShowTerminalPanel: boolean;
  conversationTitle: string;
  hasConversation: boolean;
  historyContent: ReactNode;
  isConversationPinned: boolean;
  isPinnedSummaryOpen: boolean;
  onArchiveConversation: () => void;
  onNewPage: () => void;
  onTogglePinnedConversation: () => void;
  onTogglePinnedSummary: () => void;
  repositoryPath?: string;
  sessionId?: string;
}) {
  const { pageIcon, pageTitle } =
    getProjectWorkspacePageHeader(activeProjectPage);

  return (
    <ProjectTopBar
      canShowTerminalPanel={canShowTerminalPanel}
      conversationTitle={conversationTitle}
      hasConversation={hasConversation}
      historyContent={historyContent}
      isConversationPinned={isConversationPinned}
      isLayoutPanelOpen={isPinnedSummaryOpen}
      onArchiveConversation={onArchiveConversation}
      onNewPage={onNewPage}
      onToggleLayoutPanel={onTogglePinnedSummary}
      onTogglePinnedConversation={onTogglePinnedConversation}
      pageIcon={pageIcon}
      pageTitle={pageTitle}
      repositoryPath={repositoryPath}
      showTerminalButton={activeProjectPage === "chat"}
      sessionId={sessionId}
    />
  );
}

export function ProjectWorkspaceMainTopBar({
  activeProjectPage,
  conversationTitle,
  hasConversation,
  isPinnedSummaryOpen,
  isRightSidebarOpen,
  onTogglePinnedSummary,
  repositoryPath,
  sessionId,
}: {
  activeProjectPage: ProjectWorkspacePage;
  conversationTitle: string;
  hasConversation: boolean;
  isPinnedSummaryOpen: boolean;
  isRightSidebarOpen: boolean;
  onTogglePinnedSummary: () => void;
  repositoryPath?: string;
  sessionId?: string;
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
      repositoryPath={repositoryPath}
      rightOpen={activeProjectPage === "chat" && isRightSidebarOpen}
      showTerminalButton={activeProjectPage === "chat"}
      sessionId={sessionId}
    />
  );
}
