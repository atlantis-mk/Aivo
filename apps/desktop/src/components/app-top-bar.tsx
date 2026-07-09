import { useState } from "react";

import {
  ArrowLeft,
  ArrowRight,
  Bot,
  ChevronDown,
  FileText,
  LayoutGrid,
  List,
  PanelLeft,
  PanelRight,
  SquarePen,
} from "lucide-react";

import { AnimatedTitle } from "@/components/animated-title";
import {
  TerminalTopBarButton,
  TopBarIconButton,
  WindowControls,
} from "@/components/app-top-bar-controls";
import { MoreConversationMenu } from "@/components/app-top-bar-menu";
import { createMoreMenuGroups } from "@/components/app-top-bar-menu-model";
import type { AppTopBarProps } from "@/components/app-top-bar-types";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

export { EnvironmentSummaryPanel } from "@/components/app-top-bar-environment-summary";

const narrowTopBarGlassClass =
  "pointer-events-none absolute inset-y-0 right-0 max-[1800px]:border-b max-[1800px]:border-border/60 max-[1800px]:bg-background/75 max-[1800px]:shadow-sm max-[1800px]:shadow-background/30 max-[1800px]:backdrop-blur-xl max-[1800px]:supports-[backdrop-filter]:bg-background/60";

export function AppTopBar({
  hasMessage,
  sidebarExpanded = true,
  title,
  onToggleSidebar,
  onBack,
  onForward,
  onNewPage,
  onMore,
  onModeSwitch,
  onTogglePanel,
  onPinConversation,
  onRenameConversation,
  onArchiveConversation,
  onOpenSideChat,
  onCopy,
  onBranch,
  onAddScheduledTask,
  onOpenInNewWindow,
  isPinnedSummaryOpen,
  onTogglePinnedSummary,
  onModelSelect,
  onOpenLayoutPanel,
  onToggleTerminal,
  showTerminalButton = false,
}: AppTopBarProps) {
  const [internalPinnedSummaryOpen, setInternalPinnedSummaryOpen] = useState(false);
  const resolvedTitle = title?.trim() || "未命名会话";
  const pinnedSummaryOpen = isPinnedSummaryOpen ?? internalPinnedSummaryOpen;
  const moreMenuGroups = createMoreMenuGroups({
    onAddScheduledTask,
    onArchiveConversation,
    onBranch,
    onCopy,
    onOpenInNewWindow,
    onOpenSideChat,
    onPinConversation,
    onRenameConversation,
  });

  function handleTogglePinnedSummary() {
    onTogglePinnedSummary?.();
    if (isPinnedSummaryOpen === undefined) {
      setInternalPinnedSummaryOpen((current) => !current);
    }
  }

  return (
    <header
      className="fixed inset-x-0 top-0 z-50 flex h-9 items-center bg-transparent text-foreground"
      data-app-drag
    >
      <div
        className={cn(
          narrowTopBarGlassClass,
          "group-data-[resizing=true]/sidebar-wrapper:transition-none",
        )}
        style={{ left: sidebarExpanded ? "var(--sidebar-width, 260px)" : 0 }}
      />
      <div className="relative z-10 flex h-full min-w-0 flex-1 items-center px-3">
        <div
          className={cn(
            "flex shrink-0 items-center gap-1 transition-[width] duration-300 ease-out group-data-[resizing=true]/sidebar-wrapper:transition-none",
          )}
          style={{ width: sidebarExpanded ? "var(--sidebar-width, 260px)" : "12rem" }}
        >
          <WindowControls />

          <div className="flex items-center gap-1" data-app-no-drag>
            <TopBarIconButton aria-label="展开或收起侧边栏" onClick={onToggleSidebar}>
              <PanelLeft />
            </TopBarIconButton>
            <TopBarIconButton aria-label="返回" onClick={onBack}>
              <ArrowLeft />
            </TopBarIconButton>
            <TopBarIconButton aria-label="前进" onClick={onForward}>
              <ArrowRight />
            </TopBarIconButton>
            <TopBarIconButton aria-label="新建页面" onClick={onNewPage}>
              <SquarePen />
            </TopBarIconButton>
          </div>
        </div>

        <div
          className={cn(
            "flex h-full min-w-0 flex-1 items-center transition-[padding] duration-300 ease-out",
            sidebarExpanded ? "pl-0" : "pl-1",
          )}
        >
          <div
            className={cn(
              "flex min-w-0 cursor-default select-none items-center gap-2 transition-[opacity,transform] duration-300 ease-out",
              hasMessage
                ? "pointer-events-auto translate-y-0 opacity-100"
                : "pointer-events-none -translate-y-1 opacity-0",
            )}
          >
            <FileText className="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
            <AnimatedTitle
              className="min-w-0 text-sm font-semibold text-foreground"
              value={resolvedTitle}
            />
            <span data-app-no-drag>
              <MoreConversationMenu groups={moreMenuGroups} onOpen={onMore} />
            </span>
          </div>
        </div>

        <div
          className="flex h-full min-w-0 flex-[0_1_16rem] items-center justify-end"
          data-app-no-drag
        >
          {hasMessage ? (
            <TopRightControls
              isPinnedSummaryOpen={pinnedSummaryOpen}
              onModelSelect={onModelSelect ?? onModeSwitch}
              onOpenLayoutPanel={onOpenLayoutPanel}
              onTogglePanel={onTogglePanel}
              onToggleTerminal={onToggleTerminal}
              onTogglePinnedSummary={handleTogglePinnedSummary}
              showTerminalButton={showTerminalButton}
            />
          ) : (
            <div className="flex items-center gap-1">
              <TopBarIconButton aria-label="切换布局">
                <LayoutGrid />
              </TopBarIconButton>
              {showTerminalButton ? (
                <TerminalTopBarButton onClick={onToggleTerminal} />
              ) : null}
              <TopBarIconButton aria-label="打开或关闭侧边面板" onClick={onTogglePanel}>
                <PanelRight />
              </TopBarIconButton>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}

export function AppTopBarSidebarSegment({
  onBack,
  onForward,
  onNewPage,
  onToggleSidebar,
}: Pick<
  AppTopBarProps,
  "onBack" | "onForward" | "onNewPage" | "onToggleSidebar"
>) {
  return (
    <div
      className="relative z-50 flex h-full min-w-0 items-center bg-sidebar px-3 text-sidebar-foreground"
      data-app-drag
    >
      <WindowControls />

      <div className="flex items-center gap-1" data-app-no-drag>
        <TopBarIconButton aria-label="展开或收起侧边栏" onClick={onToggleSidebar}>
          <PanelLeft />
        </TopBarIconButton>
        <TopBarIconButton aria-label="返回" onClick={onBack}>
          <ArrowLeft />
        </TopBarIconButton>
        <TopBarIconButton aria-label="前进" onClick={onForward}>
          <ArrowRight />
        </TopBarIconButton>
        <TopBarIconButton aria-label="新建页面" onClick={onNewPage}>
          <SquarePen />
        </TopBarIconButton>
      </div>
    </div>
  );
}

export function AppTopBarMainSegment({
  hasMessage,
  title,
  onToggleSidebar,
  onBack,
  onForward,
  onNewPage,
  onMore,
  onModeSwitch,
  onTogglePanel,
  onPinConversation,
  onRenameConversation,
  onArchiveConversation,
  onOpenSideChat,
  onCopy,
  onBranch,
  onAddScheduledTask,
  onOpenInNewWindow,
  isPinnedSummaryOpen,
  onTogglePinnedSummary,
  onModelSelect,
  onOpenLayoutPanel,
  onToggleTerminal,
  showSidebarControls = false,
  showTerminalButton = false,
}: AppTopBarProps & {
  showSidebarControls?: boolean;
}) {
  const [internalPinnedSummaryOpen, setInternalPinnedSummaryOpen] = useState(false);
  const resolvedTitle = title?.trim() || "未命名会话";
  const pinnedSummaryOpen = isPinnedSummaryOpen ?? internalPinnedSummaryOpen;
  const moreMenuGroups = createMoreMenuGroups({
    onAddScheduledTask,
    onArchiveConversation,
    onBranch,
    onCopy,
    onOpenInNewWindow,
    onOpenSideChat,
    onPinConversation,
    onRenameConversation,
  });

  function handleTogglePinnedSummary() {
    onTogglePinnedSummary?.();
    if (isPinnedSummaryOpen === undefined) {
      setInternalPinnedSummaryOpen((current) => !current);
    }
  }

  return (
    <header
      className="relative z-50 flex h-9 shrink-0 items-center bg-transparent text-foreground"
      data-app-drag
    >
      <div className={cn(narrowTopBarGlassClass, "left-0")} />
      <div className="relative z-10 flex h-full min-w-0 flex-1 items-center px-3">
        {showSidebarControls ? (
          <div className="flex w-48 shrink-0 items-center gap-1" data-app-no-drag>
            <WindowControls />
            <TopBarIconButton aria-label="展开或收起侧边栏" onClick={onToggleSidebar}>
              <PanelLeft />
            </TopBarIconButton>
            <TopBarIconButton aria-label="返回" onClick={onBack}>
              <ArrowLeft />
            </TopBarIconButton>
            <TopBarIconButton aria-label="前进" onClick={onForward}>
              <ArrowRight />
            </TopBarIconButton>
            <TopBarIconButton aria-label="新建页面" onClick={onNewPage}>
              <SquarePen />
            </TopBarIconButton>
          </div>
        ) : null}

        <div className="flex h-full min-w-0 flex-1 items-center">
          <div
            className={cn(
              "flex min-w-0 cursor-default select-none items-center gap-2 transition-[opacity,transform] duration-300 ease-out",
              hasMessage
                ? "pointer-events-auto translate-y-0 opacity-100"
                : "pointer-events-none -translate-y-1 opacity-0",
            )}
          >
            {onBack && !showSidebarControls ? (
              <span data-app-no-drag>
                <TopBarIconButton aria-label="返回父会话" onClick={onBack}>
                  <ArrowLeft />
                </TopBarIconButton>
              </span>
            ) : null}
            <FileText className="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
            <AnimatedTitle
              className="min-w-0 text-sm font-semibold text-foreground"
              value={resolvedTitle}
            />
            <span data-app-no-drag>
              <MoreConversationMenu groups={moreMenuGroups} onOpen={onMore} />
            </span>
          </div>
        </div>

        <div
          className="flex h-full min-w-0 flex-[0_1_16rem] items-center justify-end"
          data-app-no-drag
        >
          {hasMessage ? (
            <TopRightControls
              isPinnedSummaryOpen={pinnedSummaryOpen}
              onModelSelect={onModelSelect ?? onModeSwitch}
              onOpenLayoutPanel={onOpenLayoutPanel}
              onTogglePanel={onTogglePanel}
              onToggleTerminal={onToggleTerminal}
              onTogglePinnedSummary={handleTogglePinnedSummary}
              showTerminalButton={showTerminalButton}
            />
          ) : (
            <div className="flex items-center gap-1">
              <TopBarIconButton aria-label="切换布局">
                <LayoutGrid />
              </TopBarIconButton>
              {showTerminalButton ? (
                <TerminalTopBarButton onClick={onToggleTerminal} />
              ) : null}
              <TopBarIconButton aria-label="打开或关闭侧边面板" onClick={onTogglePanel}>
                <PanelRight />
              </TopBarIconButton>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}

function TopRightControls({
  isPinnedSummaryOpen,
  onModelSelect,
  onOpenLayoutPanel,
  onTogglePanel,
  onToggleTerminal,
  onTogglePinnedSummary,
  showTerminalButton,
}: {
  isPinnedSummaryOpen: boolean;
  onModelSelect?: () => void;
  onOpenLayoutPanel?: () => void;
  onTogglePanel?: () => void;
  onToggleTerminal?: () => void;
  onTogglePinnedSummary?: () => void;
  showTerminalButton?: boolean;
}) {
  return (
    <div className="relative flex items-center gap-1">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            aria-haspopup="menu"
            aria-label="切换模型或助手"
            className="rounded-full"
            onClick={onModelSelect}
            size="sm"
            type="button"
            variant="outline"
          >
            <Bot data-icon="inline-start" />
            <ChevronDown data-icon="inline-end" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="border border-border bg-popover text-popover-foreground">
          <DropdownMenuItem
            className="hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground"
            onClick={onModelSelect}
          >
            模型 / 助手选择
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              aria-label="切换置顶摘要"
              className={cn(
                "text-muted-foreground",
                isPinnedSummaryOpen && "bg-accent text-accent-foreground",
              )}
              onClick={onTogglePinnedSummary}
              size="icon"
              type="button"
              variant="ghost"
            >
              <List />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="bottom" sideOffset={8}>
            切换置顶摘要
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>

      <TopBarIconButton aria-label="打开布局控制" onClick={onOpenLayoutPanel}>
        <LayoutGrid />
      </TopBarIconButton>
      {showTerminalButton ? (
        <TerminalTopBarButton onClick={onToggleTerminal} />
      ) : null}
      <TopBarIconButton aria-label="打开或关闭侧边面板" onClick={onTogglePanel}>
        <PanelRight />
      </TopBarIconButton>
    </div>
  );
}
