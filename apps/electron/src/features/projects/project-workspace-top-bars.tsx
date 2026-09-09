import {
  Archive,
  Ellipsis,
  FileText,
  PanelLeft,
  Pin,
  SquarePen,
} from "lucide-react";

import { AnimatedTitle } from "@/components/animated-title";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ProjectTopBarIconButton } from "@/features/projects/project-workspace-layout";
import { cn } from "@/lib/utils";
import { ProjectWorktreeDialog } from "@/features/projects/project-worktree-dialog";
import { projectNameFromPath } from "@/features/projects/project-sidebar-model";
import { appNameFromConfig } from "@/lib/app-identity";
import { useAppConfig } from "@/lib/app-config";

export { SubagentSessionActionBar } from "@/features/projects/project-subagent-session-action-bar";

export function ProjectSidebarToggleButton({
  hasConversation,
  onNewConversation,
  isCollapsed,
  onToggleSidebar,
}: {
  hasConversation?: boolean;
  isCollapsed: boolean;
  onToggleSidebar?: () => void;
  onNewConversation?: () => void;
}) {
  const isMac = window.aivoDesktop?.platform === "darwin";
  const buttonPosition = isMac ? "left-20" : "left-2";

  return (
    <div
      className={cn(
        "fixed top-1 z-30",
        buttonPosition,
      )}
      data-app-no-drag
    >
      <div className="flex items-center gap-1">
        <ProjectTopBarIconButton
          aria-label={isCollapsed ? "展开侧边栏" : "收起侧边栏"}
          data-app-no-drag
          onPointerDown={(event) => {
            event.stopPropagation();
            onToggleSidebar?.();
          }}
        >
          <PanelLeft />
        </ProjectTopBarIconButton>
        {isCollapsed && hasConversation ? (
          <ProjectTopBarIconButton
            aria-label="新对话"
            data-app-no-drag
            onClick={onNewConversation}
          >
            <SquarePen />
          </ProjectTopBarIconButton>
        ) : null}
      </div>
    </div>
  );
}

export function ProjectTopBar({
  conversationTitle,
  hasConversation,
  isConversationPinned,
  isSidebarCollapsed,
  onArchiveConversation,
  onTogglePinnedConversation,
  pageTitle,
  repositoryPath,
  sessionId,
  onToggleSidebar,
  onNewConversation,
}: {
  conversationTitle: string;
  hasConversation: boolean;
  isConversationPinned: boolean;
  isSidebarCollapsed?: boolean;
  onArchiveConversation: () => void;
  onTogglePinnedConversation: () => void;
  pageIcon?: React.ReactNode;
  pageTitle?: string;
  repositoryPath?: string;
  sessionId?: string;
  onToggleSidebar?: () => void;
  onNewConversation?: () => void;
}) {
  const appName = appNameFromConfig(useAppConfig((state) => state.config));
  const projectLabel = repositoryPath
    ? projectNameFromPath(repositoryPath)
    : appName;
  const conversationLabel =
    pageTitle ||
    (hasConversation ? conversationTitle.trim() || "未命名会话" : "新对话");
  const isMac = window.aivoDesktop?.platform === "darwin";
  const collapsedTitleMargin = isMac ? "ml-36" : "ml-[4.5rem]";
  return (
    <header
      className={cn(
        "pointer-events-auto relative flex h-full min-w-0 items-center text-foreground",
        hasConversation
          ? "justify-between border-b border-border/60 bg-background"
          : "border-transparent bg-transparent",
      )}
      data-app-drag
    >
      <ProjectSidebarToggleButton
        hasConversation={hasConversation}
        isCollapsed={Boolean(isSidebarCollapsed)}
        onToggleSidebar={onToggleSidebar}
        onNewConversation={onNewConversation}
      />
      {hasConversation ? (
        <div
          className={cn(
            "pointer-events-none relative z-10 flex min-w-0 max-w-[min(52vw,620px)] items-center gap-aivo-1",
            // The application sidebar already occupies its own column. Keep
            // the conversation title aligned to the content edge instead of
            // applying a second sidebar-width offset.
            isSidebarCollapsed ? collapsedTitleMargin : "ml-3",
          )}
          data-app-drag
        >
          <h1 className="aivo-type-body min-w-0 truncate font-medium">
            {projectLabel} · {conversationLabel}
          </h1>
          <span
            className="pointer-events-auto flex items-center"
            data-app-no-drag
          >
            <ProjectWorktreeDialog
              repositoryPath={repositoryPath || ""}
              sessionId={sessionId}
            />
            <ProjectConversationActionsMenu
              isPinned={isConversationPinned}
              onArchive={onArchiveConversation}
              onTogglePinned={onTogglePinnedConversation}
            />
          </span>
        </div>
      ) : null}

    </header>
  );
}

function ProjectConversationActionsMenu({
  isPinned,
  onArchive,
  onTogglePinned,
}: {
  isPinned: boolean;
  onArchive: () => void;
  onTogglePinned: () => void;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <ProjectTopBarIconButton aria-label="更多会话操作">
          <Ellipsis />
        </ProjectTopBarIconButton>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onSelect={onTogglePinned}>
          <Pin className={isPinned ? "fill-current" : undefined} />
          {isPinned ? "取消置顶" : "置顶对话"}
        </DropdownMenuItem>
        <DropdownMenuItem onSelect={onArchive}>
          <Archive />
          归档对话
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function ProjectMainTopBar({
  conversationTitle,
  hasConversation,
  pageIcon,
  pageTitle,
  repositoryPath,
  sessionId,
}: {
  conversationTitle: string;
  hasConversation: boolean;
  pageIcon?: React.ReactNode;
  pageTitle?: string;
  repositoryPath?: string;
  sessionId?: string;
}) {
  const title = pageTitle || (hasConversation ? conversationTitle : "");

  return (
    <div className="pointer-events-auto relative flex h-full min-w-0 flex-1 border-b border-border/60 bg-background/80 text-foreground shadow-sm shadow-background/30 backdrop-blur-xl supports-[backdrop-filter]:bg-background/65">
      <div className="flex min-w-0 flex-1 items-center gap-2 ps-3">
        {title ? (
          <>
            <div className="flex min-w-0 items-center gap-2" data-app-drag>
              {pageIcon ?? (
                <FileText
                  aria-hidden="true"
                  className="size-4 shrink-0 text-muted-foreground"
                />
              )}
              <AnimatedTitle
                className="min-w-0 text-sm font-semibold text-foreground"
                value={title.trim() || "未命名会话"}
              />
            </div>
            {hasConversation ? (
              <span className="flex items-center" data-app-no-drag>
                <ProjectWorktreeDialog
                  repositoryPath={repositoryPath || ""}
                  sessionId={sessionId}
                />
                <ProjectTopBarIconButton aria-label="更多会话操作">
                  <Ellipsis />
                </ProjectTopBarIconButton>
              </span>
            ) : null}
          </>
        ) : null}
        <div className="h-full min-w-0 flex-1" data-app-drag />
      </div>
    </div>
  );
}
