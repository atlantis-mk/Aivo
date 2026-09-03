import { useState, type ReactNode } from "react";
import { Link } from "@tanstack/react-router";
import {
  Add01Icon,
  HistoryIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import {
  Archive,
  AppWindow,
  Ellipsis,
  FilePenLine,
  FileText,
  Pin,
  Plug,
  Settings,
} from "lucide-react";

import { AnimatedTitle } from "@/components/animated-title";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { ProjectTopBarIconButton } from "@/features/projects/project-workspace-layout";
import { cn } from "@/lib/utils";
import { ProjectWorktreeDialog } from "@/features/projects/project-worktree-dialog";
import { projectNameFromPath } from "@/features/projects/project-sidebar-model";
import { appNameFromConfig } from "@/lib/app-identity";
import { useAppConfig } from "@/lib/app-config";

export { SubagentSessionActionBar } from "@/features/projects/project-subagent-session-action-bar";
export function ProjectTopBar({
  conversationTitle,
  hasConversation,
  historyContent,
  isConversationPinned,
  onNewConversationWindow,
  onNewPage,
  onArchiveConversation,
  onTogglePinnedConversation,
  pageTitle,
  repositoryPath,
  sessionId,
}: {
  conversationTitle: string;
  hasConversation: boolean;
  historyContent: ReactNode;
  isConversationPinned: boolean;
  onNewConversationWindow: () => void;
  onNewPage: () => void;
  onArchiveConversation: () => void;
  onTogglePinnedConversation: () => void;
  pageIcon?: React.ReactNode;
  pageTitle?: string;
  repositoryPath?: string;
  sessionId?: string;
}) {
  const appName = appNameFromConfig(useAppConfig((state) => state.config));
  const isMac = window.aivoDesktop?.platform === "darwin";
  const projectLabel = repositoryPath
    ? projectNameFromPath(repositoryPath)
    : appName;
  const conversationLabel =
    pageTitle || (hasConversation ? conversationTitle.trim() || "未命名会话" : "新对话");

  return (
    <header
      className="pointer-events-auto relative flex h-full min-w-0 items-center border-b border-border/60 bg-background text-foreground"
      data-app-drag
    >
      <div className="flex min-w-0 items-center gap-aivo-2" data-app-no-drag>
        <ProjectWindowControls isMac={isMac} />
        <Button
          onClick={onNewPage}
          size="sm"
          type="button"
          variant="outline"
        >
          <HugeiconsIcon icon={Add01Icon} strokeWidth={1.8} />
          <span className="hidden sm:inline">新对话</span>
        </Button>
        <Button
          onClick={onNewConversationWindow}
          size="sm"
          type="button"
          variant="outline"
        >
          <AppWindow />
          <span className="hidden sm:inline">新窗口对话</span>
        </Button>
        <ProjectHistoryPopover>{historyContent}</ProjectHistoryPopover>
      </div>

      <div
        className="pointer-events-none absolute left-1/2 z-10 flex max-w-[min(38vw,520px)] -translate-x-1/2 items-center gap-aivo-1"
        data-app-drag
      >
        <h1 className="aivo-type-body min-w-0 truncate font-medium">
          {projectLabel} · {conversationLabel}
        </h1>
        {hasConversation ? (
          <span className="pointer-events-auto flex items-center" data-app-no-drag>
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
        ) : null}
      </div>

      <div
        className="relative z-20 ml-auto flex items-center gap-aivo-1 pr-aivo-3"
        data-app-no-drag
      >
        <ProjectTopBarIconButton asChild aria-label="打开扩展">
          <Link to="/extensions">
            <Plug />
          </Link>
        </ProjectTopBarIconButton>
        <ProjectTopBarIconButton asChild aria-label="管理提示词">
          <Link to="/prompts">
            <FilePenLine />
          </Link>
        </ProjectTopBarIconButton>
        <ProjectTopBarIconButton asChild aria-label="打开设置">
          <Link to="/settings">
            <Settings />
          </Link>
        </ProjectTopBarIconButton>
      </div>
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

function ProjectHistoryPopover({ children }: { children: ReactNode }) {
  const [open, setOpen] = useState(false);

  return (
    <Popover onOpenChange={setOpen} open={open}>
      <PopoverTrigger asChild>
        <Button
          size="sm"
          type="button"
          variant="outline"
        >
          <HugeiconsIcon icon={HistoryIcon} strokeWidth={1.8} />
          <span className="hidden sm:inline">历史记录</span>
        </Button>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        className="w-[min(340px,calc(100vw-24px))] overflow-hidden p-0"
        onClickCapture={(event) => {
          const target = event.target as HTMLElement;
          if (target.closest("a") || target.closest("button")) setOpen(false);
        }}
        sideOffset={6}
      >
        {children}
      </PopoverContent>
    </Popover>
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
                <ProjectWorktreeDialog repositoryPath={repositoryPath || ""} sessionId={sessionId} />
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

function ProjectWindowControls({ isMac }: { isMac: boolean }) {
  return (
    <div
      aria-hidden="true"
      className={cn("shrink-0", isMac ? "w-[72px]" : "w-0")}
      data-app-no-drag
    />
  );
}
