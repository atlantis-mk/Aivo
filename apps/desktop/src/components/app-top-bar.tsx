import {
  useState,
  type ComponentProps,
  type ComponentType,
  type KeyboardEvent,
  type MouseEvent,
} from "react";

import {
  Archive,
  ArrowLeft,
  ArrowRight,
  Bot,
  ChevronDown,
  Clock,
  Copy,
  Ellipsis,
  ExternalLink,
  FileDiff,
  FileText,
  GitBranch,
  GitCommitHorizontal,
  Globe,
  MessageSquarePlus,
  Pencil,
  LayoutGrid,
  Laptop,
  List,
  Pin,
  PanelLeft,
  PanelBottom,
  PanelRight,
  Plus,
  SquarePen,
  Terminal,
  Wrench,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { AnimatedTitle } from "@/components/animated-title";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

type AppTopBarProps = {
  hasMessage: boolean;
  sidebarExpanded?: boolean;
  title?: string;
  onToggleSidebar?: () => void;
  onBack?: () => void;
  onForward?: () => void;
  onNewPage?: () => void;
  onMore?: () => void;
  onModeSwitch?: () => void;
  onTogglePanel?: () => void;
  onToggleTerminal?: () => void;
  showTerminalButton?: boolean;
  onPinConversation?: () => void;
  onRenameConversation?: () => void;
  onArchiveConversation?: () => void;
  onOpenSideChat?: () => void;
  onCopy?: () => void;
  onBranch?: () => void;
  onAddScheduledTask?: () => void;
  onOpenInNewWindow?: () => void;
  isPinnedSummaryOpen?: boolean;
  onTogglePinnedSummary?: () => void;
  onModelSelect?: () => void;
  onAddContext?: () => void;
  onSelectLocalEnvironment?: () => void;
  onSelectBranch?: () => void;
  onCommitOrPush?: () => void;
  onOpenLayoutPanel?: () => void;
};

type MoreMenuItem = {
  id: string;
  label: string;
  icon: ComponentType<{ className?: string }>;
  shortcut?: string;
  hasSubmenu?: boolean;
  children?: MoreMenuItem[];
  onClick?: () => void;
  disabled?: boolean;
};

const menuItemClassName =
  "relative flex min-h-7 cursor-default items-center gap-2 rounded-md px-2 py-1 text-xs/relaxed outline-hidden select-none hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-3.5";

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
  const moreMenuGroups: MoreMenuItem[][] = [
    [
      {
        id: "pin-conversation",
        label: "置顶对话",
        icon: Pin,
        shortcut: "⌥⌘P",
        onClick: onPinConversation,
      },
      {
        id: "rename-conversation",
        label: "重命名对话",
        icon: Pencil,
        shortcut: "⌥⌘R",
        onClick: onRenameConversation,
      },
      {
        id: "archive-conversation",
        label: "归档对话",
        icon: Archive,
        shortcut: "⇧⌘A",
        onClick: onArchiveConversation,
      },
    ],
    [
      {
        id: "open-side-chat",
        label: "打开侧边聊天",
        icon: MessageSquarePlus,
        shortcut: "⌥⌘S",
        onClick: onOpenSideChat,
      },
      {
        id: "copy",
        label: "复制",
        icon: Copy,
        hasSubmenu: true,
        children: [
          {
            id: "copy-conversation",
            label: "复制对话",
            icon: Copy,
            onClick: onCopy,
          },
        ],
      },
      {
        id: "branch",
        label: "分支",
        icon: GitBranch,
        hasSubmenu: true,
        children: [
          {
            id: "create-branch",
            label: "创建分支",
            icon: GitBranch,
            onClick: onBranch,
          },
        ],
      },
      {
        id: "add-scheduled-task",
        label: "添加计划任务...",
        icon: Clock,
        onClick: onAddScheduledTask,
      },
    ],
    [
      {
        id: "open-in-new-window",
        label: "在新窗口中打开",
        icon: ExternalLink,
        onClick: onOpenInNewWindow,
      },
    ],
  ];

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
          "group-data-[resizing=true]/sidebar-wrapper:transition-none"
        )}
        style={{ left: sidebarExpanded ? "var(--sidebar-width, 260px)" : 0 }}
      />
      <div
        className="relative z-10 flex h-full min-w-0 flex-1 items-center px-3"
      >
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

function createMoreMenuGroups({
  onAddScheduledTask,
  onArchiveConversation,
  onBranch,
  onCopy,
  onOpenInNewWindow,
  onOpenSideChat,
  onPinConversation,
  onRenameConversation,
}: Pick<
  AppTopBarProps,
  | "onAddScheduledTask"
  | "onArchiveConversation"
  | "onBranch"
  | "onCopy"
  | "onOpenInNewWindow"
  | "onOpenSideChat"
  | "onPinConversation"
  | "onRenameConversation"
>) {
  return [
    [
      {
        id: "pin-conversation",
        label: "置顶对话",
        icon: Pin,
        shortcut: "⌥⌘P",
        onClick: onPinConversation,
      },
      {
        id: "rename-conversation",
        label: "重命名对话",
        icon: Pencil,
        shortcut: "⌥⌘R",
        onClick: onRenameConversation,
      },
      {
        id: "archive-conversation",
        label: "归档对话",
        icon: Archive,
        shortcut: "⇧⌘A",
        onClick: onArchiveConversation,
      },
    ],
    [
      {
        id: "open-side-chat",
        label: "打开侧边聊天",
        icon: MessageSquarePlus,
        shortcut: "⌥⌘S",
        onClick: onOpenSideChat,
      },
      {
        id: "copy",
        label: "复制",
        icon: Copy,
        hasSubmenu: true,
        children: [
          {
            id: "copy-conversation",
            label: "复制对话",
            icon: Copy,
            onClick: onCopy,
          },
        ],
      },
      {
        id: "branch",
        label: "分支",
        icon: GitBranch,
        hasSubmenu: true,
        children: [
          {
            id: "create-branch",
            label: "创建分支",
            icon: GitBranch,
            onClick: onBranch,
          },
        ],
      },
      {
        id: "add-scheduled-task",
        label: "添加计划任务...",
        icon: Clock,
        onClick: onAddScheduledTask,
      },
    ],
    [
      {
        id: "open-in-new-window",
        label: "在新窗口中打开",
        icon: ExternalLink,
        onClick: onOpenInNewWindow,
      },
    ],
  ] satisfies MoreMenuItem[][];
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

function TerminalTopBarButton({ onClick }: { onClick?: () => void }) {
  return (
    <TopBarIconButton aria-label="打开终端面板" onClick={onClick}>
      <PanelBottom />
    </TopBarIconButton>
  );
}

export function EnvironmentSummaryPanel({
  className,
  onAddContext,
  onCommitOrPush,
  onOpenTools,
  onSelectBranch,
  onSelectLocalEnvironment,
}: {
  className?: string;
  onAddContext?: () => void;
  onCommitOrPush?: () => void;
  onOpenTools?: () => void;
  onSelectBranch?: () => void;
  onSelectLocalEnvironment?: () => void;
}) {
  const [expanded, setExpanded] = useState(true);
  const [sourceExpanded, setSourceExpanded] = useState(true);

  function toggleExpanded() {
    setExpanded((current) => !current);
  }

  function toggleSourceExpanded() {
    setSourceExpanded((current) => !current);
  }

  function handleHeaderKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    if (event.key !== "Enter" && event.key !== " ") return;
    event.preventDefault();
    toggleExpanded();
  }

  function handleSourceHeaderKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    if (event.key !== "Enter" && event.key !== " ") return;
    event.preventDefault();
    toggleSourceExpanded();
  }

  function handleAddContextClick(event: MouseEvent<HTMLButtonElement>) {
    event.stopPropagation();
    onAddContext?.();
  }

  function handleToggleButtonClick(event: MouseEvent<HTMLButtonElement>) {
    event.stopPropagation();
    toggleExpanded();
  }

  function handleSourceToggleButtonClick(event: MouseEvent<HTMLButtonElement>) {
    event.stopPropagation();
    toggleSourceExpanded();
  }

  return (
    <div className={cn("w-72 overflow-hidden rounded-lg bg-popover p-1 text-popover-foreground shadow-md ring-1 ring-foreground/10", className)}>
      <div
        aria-expanded={expanded}
        className={cn(
          "group/env-header flex cursor-pointer items-center rounded-md px-2 py-1.5 outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:ring-2 focus-visible:ring-ring/30",
        )}
        onClick={toggleExpanded}
        onKeyDown={handleHeaderKeyDown}
        role="button"
        tabIndex={0}
      >
        <div className="flex min-w-0 flex-1 items-center">
          <span className="min-w-0 truncate text-xs text-muted-foreground">环境信息</span>
          <Button
            aria-label={expanded ? "收起环境信息" : "展开环境信息"}
            className="ml-1 opacity-0 transition-opacity group-hover/env-header:opacity-100 group-focus-within/env-header:opacity-100"
            onClick={handleToggleButtonClick}
            size="icon-sm"
            type="button"
            variant="ghost"
          >
            <ChevronDown
              className={cn(
                "transition-transform duration-150",
                !expanded && "-rotate-90",
              )}
            />
          </Button>
        </div>
        <Button aria-label="添加上下文" onClick={handleAddContextClick} size="icon-sm" type="button" variant="ghost">
          <Plus />
        </Button>
      </div>
      <div
        className={cn(
          "grid transition-[grid-template-rows,opacity] duration-200 ease-out",
          expanded ? "grid-rows-[1fr] opacity-100" : "grid-rows-[0fr] opacity-0",
        )}
      >
        <div className="min-h-0 overflow-hidden">
          <div className="flex flex-col">
            <EnvironmentSummaryRow icon={FileDiff} label="变更" />
            <EnvironmentSummaryRow
              action="expand"
              icon={Laptop}
              label="本地"
              onClick={onSelectLocalEnvironment}
            />
            <EnvironmentSummaryRow
              action="expand"
              icon={GitBranch}
              label="main"
              onClick={onSelectBranch}
            />
            <EnvironmentSummaryRow icon={GitCommitHorizontal} label="提交或推送" onClick={onCommitOrPush} />
            <EnvironmentSummaryRow icon={Wrench} label="工具" onClick={onOpenTools} />
            <EnvironmentSummaryRow disabled icon={Terminal} label="GitHub CLI 未通过身份验证" />
          </div>
        </div>
      </div>
      <div className="-mx-1 my-1 h-px bg-border/50" />
      <div
        aria-expanded={sourceExpanded}
        className="group/source-header flex cursor-pointer items-center rounded-md px-2 py-1.5 outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:ring-2 focus-visible:ring-ring/30"
        onClick={toggleSourceExpanded}
        onKeyDown={handleSourceHeaderKeyDown}
        role="button"
        tabIndex={0}
      >
        <span className="min-w-0 truncate text-xs text-muted-foreground">来源</span>
        <Button
          aria-label={sourceExpanded ? "收起来源" : "展开来源"}
          className="ml-1 opacity-0 transition-opacity group-hover/source-header:opacity-100 group-focus-within/source-header:opacity-100"
          onClick={handleSourceToggleButtonClick}
          size="icon-sm"
          type="button"
          variant="ghost"
        >
          <ChevronDown
            className={cn(
              "transition-transform duration-150",
              !sourceExpanded && "-rotate-90",
            )}
          />
        </Button>
      </div>
      <div
        className={cn(
          "grid transition-[grid-template-rows,opacity] duration-200 ease-out",
          sourceExpanded ? "grid-rows-[1fr] opacity-100" : "grid-rows-[0fr] opacity-0",
        )}
      >
        <div className="min-h-0 overflow-hidden">
          <EnvironmentSummaryRow icon={Globe} label="" />
        </div>
      </div>
    </div>
  );
}

function EnvironmentSummaryRow({
  action,
  disabled,
  icon: Icon,
  label,
  onClick,
}: {
  action?: "expand";
  disabled?: boolean;
  icon: ComponentType<{ className?: string }>;
  label: string;
  onClick?: () => void;
}) {
  const Comp = onClick ? "button" : "div";

  return (
    <Comp
      aria-label={onClick ? label : undefined}
      className={cn(
        menuItemClassName,
        "",
        disabled ? "text-muted-foreground opacity-70" : "text-card-foreground",
      )}
      onClick={onClick}
      type={onClick ? "button" : undefined}
    >
      <Icon className="text-muted-foreground" />
      <span className="min-w-0 flex-1 truncate text-left">{label}</span>
      {action === "expand" && <ChevronDown className="text-muted-foreground" />}
    </Comp>
  );
}

function MoreConversationMenu({
  groups,
  onOpen,
}: {
  groups: MoreMenuItem[][];
  onOpen?: () => void;
}) {
  const [open, setOpen] = useState(false);

  function handleOpenChange(nextOpen: boolean) {
    setOpen(nextOpen);
    if (nextOpen) {
      onOpen?.();
    }
  }

  return (
    <DropdownMenu open={open} onOpenChange={handleOpenChange}>
      <DropdownMenuTrigger asChild>
        <Button
          aria-expanded={open}
          aria-haspopup="menu"
          aria-label="更多会话操作"
          className="text-muted-foreground data-[state=open]:bg-muted data-[state=open]:text-foreground"
          size="icon"
          type="button"
          variant="ghost"
        >
          <Ellipsis />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="start"
        className="w-80 border border-border bg-popover text-popover-foreground shadow-xl shadow-foreground/10"
        side="bottom"
        sideOffset={8}
      >
        {groups.map((group, groupIndex) => (
          <DropdownMenuGroup key={group.map((item) => item.id).join("-")}>
            {group.map((item) => (
              <MoreConversationMenuItem item={item} key={item.id} />
            ))}
            {groupIndex < groups.length - 1 && (
              <DropdownMenuSeparator className="mx-2 my-2 bg-border" />
            )}
          </DropdownMenuGroup>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function MoreConversationMenuItem({ item }: { item: MoreMenuItem }) {
  const Icon = item.icon;

  if (item.hasSubmenu) {
    return (
        <DropdownMenuSub>
          <DropdownMenuSubTrigger
          className="hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground"
          disabled={item.disabled}
        >
          <Icon className="text-muted-foreground" />
          <span className="min-w-0 flex-1 truncate">{item.label}</span>
        </DropdownMenuSubTrigger>
        <DropdownMenuSubContent className="border border-border bg-popover text-popover-foreground shadow-xl shadow-foreground/10">
          {(item.children ?? []).map((child) => (
            <MoreConversationMenuItem item={child} key={child.id} />
          ))}
        </DropdownMenuSubContent>
      </DropdownMenuSub>
    );
  }

  return (
    <DropdownMenuItem
      className="hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground"
      disabled={item.disabled}
      onClick={item.onClick}
    >
      <Icon className="text-muted-foreground" />
      <span className="min-w-0 flex-1 truncate">{item.label}</span>
      {item.shortcut && (
        <DropdownMenuShortcut className="tracking-normal text-muted-foreground">
          {item.shortcut}
        </DropdownMenuShortcut>
      )}
    </DropdownMenuItem>
  );
}

function WindowControls() {
  const isMac = window.aivo?.platform === "darwin";

  return (
    <div className={cn("shrink-0", isMac ? "w-[54px]" : "w-0")} data-app-no-drag aria-hidden="true" />
  );
}

function TopBarIconButton({
  className,
  ...props
}: ComponentProps<typeof Button>) {
  return (
    <Button
      className={cn("text-muted-foreground", className)}
      size="icon"
      type="button"
      variant="ghost"
      {...props}
    />
  );
}
