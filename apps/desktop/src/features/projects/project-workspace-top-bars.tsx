import type React from "react";
import {
  ArrowLeft,
  ArrowRight,
  Ellipsis,
  FileText,
  LayoutGrid,
  PanelLeft,
  SquarePen,
} from "lucide-react";

import { AnimatedTitle } from "@/components/animated-title";
import { ProjectTopBarIconButton } from "@/features/projects/project-workspace-layout";
import { cn } from "@/lib/utils";
import { ProjectWorktreeDialog } from "@/features/projects/project-worktree-dialog";

export { SubagentSessionActionBar } from "@/features/projects/project-subagent-session-action-bar";
export function ProjectTopBar({
  leftSidebarState,
  onNewPage,
  onToggleLeftSidebar,
}: {
  canShowTerminalPanel: boolean;
  conversationTitle: string;
  hasConversation: boolean;
  leftSidebarState: "expanded" | "collapsed";
  onNewPage: () => void;
  onToggleLeftSidebar: () => void;
  pageIcon?: React.ReactNode;
  pageTitle?: string;
  showTerminalButton?: boolean;
}) {
  const isMac = window.aivo?.platform === "darwin";
  const leftCompactWidth = isMac ? 202 : 148;

  return (
    <header className="pointer-events-none relative flex h-full min-w-0 text-foreground">
      <div
        className={cn(
          "pointer-events-auto relative flex h-full shrink-0 items-center overflow-hidden text-sidebar-foreground transition-[width] duration-[var(--project-panel-transition-duration,200ms)] ease-linear",
          leftSidebarState === "collapsed" && "border-b border-border/60",
        )}
        style={{
          width:
            leftSidebarState === "expanded"
              ? "var(--project-left-sidebar-width)"
              : `${leftCompactWidth}px`,
        }}
      >
        <ProjectWindowControls isMac={isMac} />
        <div
          className="pointer-events-auto flex h-full shrink-0 items-center gap-1 px-3"
          data-app-no-drag
        >
          <ProjectTopBarIconButton
            aria-label="展开或收起侧边栏"
            onClick={onToggleLeftSidebar}
          >
            <PanelLeft />
          </ProjectTopBarIconButton>
          <ProjectTopBarIconButton aria-label="返回" onClick={() => undefined}>
            <ArrowLeft />
          </ProjectTopBarIconButton>
          <ProjectTopBarIconButton aria-label="前进" onClick={() => undefined}>
            <ArrowRight />
          </ProjectTopBarIconButton>
          <ProjectTopBarIconButton aria-label="新建页面" onClick={onNewPage}>
            <SquarePen />
          </ProjectTopBarIconButton>
        </div>
        <div className="h-full min-w-0 flex-1" data-app-drag />
      </div>
    </header>
  );
}

export function ProjectMainTopBar({
  conversationTitle,
  hasConversation,
  isLayoutPanelOpen,
  onToggleLayoutPanel,
  pageIcon,
  pageTitle,
  repositoryPath,
  rightOpen,
  sessionId,
  showTerminalButton,
}: {
  conversationTitle: string;
  hasConversation: boolean;
  isLayoutPanelOpen?: boolean;
  onToggleLayoutPanel?: () => void;
  pageIcon?: React.ReactNode;
  pageTitle?: string;
  repositoryPath?: string;
  rightOpen?: boolean;
  sessionId?: string;
  showTerminalButton?: boolean;
}) {
  const title = pageTitle || (hasConversation ? conversationTitle : "");
  const floatingActionsInset = showTerminalButton ? 76 : 40;
  const layoutActionRight = rightOpen ? 0 : floatingActionsInset;
  const actionsInset = `${layoutActionRight + 44}px`;

  return (
    <div className="pointer-events-auto relative flex h-full min-w-0 flex-1 border-b border-border/60 bg-background/80 text-foreground shadow-sm shadow-background/30 backdrop-blur-xl supports-[backdrop-filter]:bg-background/65">
      <div
        className="flex min-w-0 flex-1 items-center gap-2 ps-3"
        style={{ paddingRight: actionsInset }}
      >
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
      <ProjectMainTopBarActions
        isLayoutPanelOpen={isLayoutPanelOpen}
        onToggleLayoutPanel={onToggleLayoutPanel}
        right={layoutActionRight}
      />
    </div>
  );
}

function ProjectMainTopBarActions({
  isLayoutPanelOpen,
  onToggleLayoutPanel,
  right,
}: {
  isLayoutPanelOpen?: boolean;
  onToggleLayoutPanel?: () => void;
  right: number;
}) {
  return (
    <div
      className="pointer-events-auto absolute right-0 top-0 z-[60] flex h-9 shrink-0 items-center justify-end gap-2 pe-3 text-foreground"
      data-app-no-drag
      style={{ right }}
    >
      <ProjectTopBarIconButton
        aria-label="切换系统环境"
        aria-pressed={isLayoutPanelOpen}
        className={cn(isLayoutPanelOpen && "bg-muted text-foreground")}
        onClick={onToggleLayoutPanel}
      >
        <LayoutGrid />
      </ProjectTopBarIconButton>
    </div>
  );
}

function ProjectWindowControls({ isMac }: { isMac: boolean }) {
  return (
    <div
      aria-hidden="true"
      className={cn("shrink-0", isMac ? "w-[54px]" : "w-0")}
      data-app-no-drag
    />
  );
}
