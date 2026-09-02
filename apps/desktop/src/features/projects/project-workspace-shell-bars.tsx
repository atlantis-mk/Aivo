import type { ReactNode } from "react";
import { Plug } from "lucide-react";

import {
  ProjectMainTopBar,
  ProjectTopBar,
} from "@/features/projects/project-workspace-top-bars";
import type { ProjectWorkspacePage } from "@/features/projects/project-workspace-derived-state";

function getProjectWorkspacePageHeader(activeProjectPage: ProjectWorkspacePage) {
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
  historyContent,
  isConversationPinned,
  onArchiveConversation,
  onNewConversationWindow,
  onNewPage,
  onTogglePinnedConversation,
  repositoryPath,
  sessionId,
}: {
  activeProjectPage: ProjectWorkspacePage;
  conversationTitle: string;
  hasConversation: boolean;
  historyContent: ReactNode;
  isConversationPinned: boolean;
  onArchiveConversation: () => void;
  onNewConversationWindow: () => void;
  onNewPage: () => void;
  onTogglePinnedConversation: () => void;
  repositoryPath?: string;
  sessionId?: string;
}) {
  const { pageIcon, pageTitle } =
    getProjectWorkspacePageHeader(activeProjectPage);

  return (
    <ProjectTopBar
      conversationTitle={conversationTitle}
      hasConversation={hasConversation}
      historyContent={historyContent}
      isConversationPinned={isConversationPinned}
      onArchiveConversation={onArchiveConversation}
      onNewConversationWindow={onNewConversationWindow}
      onNewPage={onNewPage}
      onTogglePinnedConversation={onTogglePinnedConversation}
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
