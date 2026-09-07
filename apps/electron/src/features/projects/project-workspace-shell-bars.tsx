import { Plug } from "lucide-react";

import {
  ProjectMainTopBar,
  ProjectTopBar,
} from "@/features/projects/project-workspace-top-bars";
import type { ProjectWorkspacePage } from "@/features/projects/project-workspace-derived-state";

function getProjectWorkspacePageHeader(
  activeProjectPage: ProjectWorkspacePage,
) {
  if (activeProjectPage !== "extensions") {
    return { pageIcon: undefined, pageTitle: undefined };
  }

  return {
    pageIcon: (
      <Plug
        aria-hidden="true"
        className="size-4 shrink-0 text-muted-foreground"
      />
    ),
    pageTitle: "扩展与 MCP",
  };
}

export function ProjectWorkspaceTopBar({
  activeProjectPage,
  conversationTitle,
  hasConversation,
  isConversationPinned,
  isSidebarCollapsed,
  onArchiveConversation,
  onTogglePinnedConversation,
  onToggleSidebar,
  onNewConversation,
  repositoryPath,
  sessionId,
}: {
  activeProjectPage: ProjectWorkspacePage;
  conversationTitle: string;
  hasConversation: boolean;
  isConversationPinned: boolean;
  isSidebarCollapsed?: boolean;
  onArchiveConversation: () => void;
  onTogglePinnedConversation: () => void;
  onToggleSidebar?: () => void;
  onNewConversation?: () => void;
  repositoryPath?: string;
  sessionId?: string;
}) {
  const { pageIcon, pageTitle } =
    getProjectWorkspacePageHeader(activeProjectPage);

  return (
    <ProjectTopBar
      conversationTitle={conversationTitle}
      hasConversation={hasConversation}
      isConversationPinned={isConversationPinned}
      isSidebarCollapsed={isSidebarCollapsed}
      onArchiveConversation={onArchiveConversation}
      onTogglePinnedConversation={onTogglePinnedConversation}
      onToggleSidebar={onToggleSidebar}
      onNewConversation={onNewConversation}
      pageIcon={pageIcon}
      pageTitle={pageTitle}
      repositoryPath={repositoryPath}
      sessionId={sessionId}
    />
  );
}

export function ProjectWorkspaceMainTopBar({
  activeProjectPage,
  conversationTitle,
  hasConversation,
  repositoryPath,
  sessionId,
}: {
  activeProjectPage: ProjectWorkspacePage;
  conversationTitle: string;
  hasConversation: boolean;
  repositoryPath?: string;
  sessionId?: string;
}) {
  const { pageIcon, pageTitle } =
    getProjectWorkspacePageHeader(activeProjectPage);

  return (
    <ProjectMainTopBar
      conversationTitle={conversationTitle}
      hasConversation={hasConversation}
      pageIcon={pageIcon}
      pageTitle={pageTitle}
      repositoryPath={repositoryPath}
      sessionId={sessionId}
    />
  );
}
