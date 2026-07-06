import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type ComponentType,
} from "react";

import {
  CheckCircle2,
  CircleAlert,
  Clock,
  ExternalLink,
  FileText,
  Globe,
  Loader2,
  Plus,
  RotateCcw,
  Terminal,
  Undo2,
  X,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import {
  type ToolActivityCommandTab,
  type ToolActivityFileTab,
  type ToolActivityTab,
} from "@/features/projects/tool-activity-model";
import { BuiltinBrowserSidebar } from "@/features/projects/builtin-browser-sidebar";

export const BUILTIN_BROWSER_TAB_ID = "builtin-browser";

type ToolActivitySidebarProps = {
  activeTabId?: string;
  browserInitialUrls?: Record<string, string>;
  browserReadyTokens?: Record<string, number>;
  browserTabIds?: string[];
  browserVisible?: boolean;
  tabs: ToolActivityTab[];
  onActiveTabChange: (tabId: string) => void;
  onApplyFileState?: (
    tab: ToolActivityFileTab,
    targetState: "before" | "after",
  ) => void;
  onBrowserReady?: (tabId: string) => void;
  onCloseBrowser?: (tabId?: string) => void;
  onCloseTab: (tabId: string) => void;
  onOpenBrowser?: (targetUrl?: string) => void;
};

export function ToolActivitySidebar({
  activeTabId,
  browserInitialUrls = {},
  browserReadyTokens = {},
  browserTabIds = [],
  browserVisible = true,
  tabs,
  onActiveTabChange,
  onApplyFileState,
  onBrowserReady,
  onCloseBrowser,
  onCloseTab,
  onOpenBrowser,
}: ToolActivitySidebarProps) {
  const activeBrowserTabId = browserTabIds.includes(activeTabId ?? "")
    ? activeTabId ?? ""
    : "";
  const activeTab =
    activeBrowserTabId
      ? null
      : tabs.find((tab) => tab.id === activeTabId) ?? tabs.at(-1) ?? null;
  const hasTabs = tabs.length > 0 || browserTabIds.length > 0;
  const [browserStates, setBrowserStates] = useState<Record<string, AivoBrowserState>>({});
  const handleBrowserStateChange = useCallback((
    browserTabId: string,
    browserState: AivoBrowserState,
  ) => {
    setBrowserStates((currentStates) => {
      if (currentStates[browserTabId] === browserState) {
        return currentStates;
      }
      return {
        ...currentStates,
        [browserTabId]: browserState,
      };
    });
  }, []);
  const handleActiveBrowserReady = useCallback(() => {
    if (!activeBrowserTabId) return;
    onBrowserReady?.(activeBrowserTabId);
  }, [activeBrowserTabId, onBrowserReady]);

  useEffect(() => {
    const browserTabIdSet = new Set(browserTabIds);
    setBrowserStates((currentStates) => {
      const nextStates = Object.fromEntries(
        Object.entries(currentStates).filter(([browserTabId]) =>
          browserTabIdSet.has(browserTabId),
        ),
      );
      return Object.keys(nextStates).length === Object.keys(currentStates).length
        ? currentStates
        : nextStates;
    });
  }, [browserTabIds]);

  return (
    <section
      className="relative z-[70] flex h-full min-h-0 flex-col bg-background"
      data-app-no-drag
    >
      {hasTabs ? (
        <div className="flex h-9 shrink-0 items-center pl-2 pr-(--project-right-topbar-actions-width)">
          <ToolActivitySidebarTabs
            activeTabId={activeTabId}
            browserStates={browserStates}
            browserTabIds={browserTabIds}
            onActiveTabChange={onActiveTabChange}
            onCloseBrowser={onCloseBrowser}
            onCloseTab={onCloseTab}
            onOpenBrowser={onOpenBrowser}
            tabs={tabs}
          />
        </div>
      ) : null}
      {hasTabs ? (
        <div className="min-h-0 flex-1 overflow-hidden">
          {activeBrowserTabId ? (
            <BuiltinBrowserSidebar
              browserTabId={activeBrowserTabId}
              initialUrl={browserInitialUrls[activeBrowserTabId]}
              onClose={() => onCloseBrowser?.(activeBrowserTabId)}
              onReady={handleActiveBrowserReady}
              onStateChange={(browserState) =>
                handleBrowserStateChange(activeBrowserTabId, browserState)
              }
              readyToken={browserReadyTokens[activeBrowserTabId]}
              visible={browserVisible}
            />
          ) : activeTab ? (
            <ToolActivityDetail
              onApplyFileState={onApplyFileState}
              tab={activeTab}
            />
          ) : (
            <ToolActivityEmptyState onOpenBrowser={onOpenBrowser} />
          )}
        </div>
      ) : (
        <ToolActivityEmptyState onOpenBrowser={onOpenBrowser} />
      )}
    </section>
  );
}

type ToolActivitySidebarTabsProps = Pick<
  ToolActivitySidebarProps,
  | "activeTabId"
  | "browserTabIds"
  | "onActiveTabChange"
  | "onCloseBrowser"
  | "onCloseTab"
  | "onOpenBrowser"
  | "tabs"
>;

export function ToolActivitySidebarTabs({
  activeTabId,
  browserStates,
  browserTabIds = [],
  tabs,
  onActiveTabChange,
  onCloseBrowser,
  onCloseTab,
  onOpenBrowser,
}: ToolActivitySidebarTabsProps & {
  browserStates?: Record<string, AivoBrowserState>;
}) {
  const activeBrowserTabId = browserTabIds.includes(activeTabId ?? "")
    ? activeTabId ?? ""
    : "";
  const activeTab =
    activeBrowserTabId
      ? null
      : tabs.find((tab) => tab.id === activeTabId) ?? tabs.at(-1) ?? null;
  const activeValue = activeBrowserTabId || activeTab?.id || "";
  const hasTabs = tabs.length > 0 || browserTabIds.length > 0;

  if (!hasTabs) return null;

  return (
    <div
      aria-label="右侧栏标签页"
      className="flex min-w-0 flex-1 items-center overflow-hidden"
      role="tablist"
    >
      {browserTabIds.map((browserTabId) => (
        <BrowserActivityTopTab
          active={activeValue === browserTabId}
          browserTabId={browserTabId}
          key={browserTabId}
          onClick={() => onActiveTabChange(browserTabId)}
          onClose={() => onCloseBrowser?.(browserTabId)}
          state={browserStates?.[browserTabId]}
        />
      ))}
      {tabs.map((tab) => (
        <ToolActivityTopTab
          active={activeValue === tab.id}
          key={tab.id}
          onClick={() => onActiveTabChange(tab.id)}
          onClose={() => onCloseTab(tab.id)}
          title={tabTitle(tab)}
        >
          <ToolActivityTabLabel tab={tab} />
        </ToolActivityTopTab>
      ))}
      <ToolActivitySidebarAddMenu onOpenBrowser={onOpenBrowser} />
    </div>
  );
}

function BrowserActivityTopTab({
  active,
  onClick,
  onClose,
  state,
}: {
  active: boolean;
  browserTabId: string;
  onClick: () => void;
  onClose?: () => void;
  state?: AivoBrowserState;
}) {
  const title = state?.title.trim() || "内置浏览器";
  const favicon = state?.favicon.trim() || "";
  return (
    <ToolActivityTopTab
      active={active}
      onClick={onClick}
      onClose={onClose}
      title={title}
    >
      <BrowserTabIcon favicon={favicon} />
      <span className="min-w-0 truncate">{title}</span>
    </ToolActivityTopTab>
  );
}

function BrowserTabIcon({ favicon }: { favicon: string }) {
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    setFailed(false);
  }, [favicon]);

  if (!favicon || failed) {
    return <Globe className="size-3 shrink-0" />;
  }

  return (
    <img
      alt=""
      className="size-3 shrink-0 rounded-[2px] object-contain"
      draggable={false}
      onError={() => setFailed(true)}
      src={favicon}
    />
  );
}

function ToolActivityTopTab({
  active,
  children,
  onClick,
  onClose,
  title,
}: {
  active: boolean;
  children: React.ReactNode;
  onClick: () => void;
  onClose?: () => void;
  title: string;
}) {
  return (
    <div className="group/tab relative flex-none" role="presentation">
      <button
        aria-selected={active}
        className={cn(
          "flex h-7 max-w-40 min-w-0 items-center justify-start gap-1.5 rounded-md px-2 pr-5 text-xs font-medium text-muted-foreground transition-none hover:bg-muted/70 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/35",
          active && "bg-muted text-foreground",
        )}
        onClick={onClick}
        role="tab"
        title={title}
        type="button"
      >
        {children}
      </button>
      {onClose ? (
        <Button
          aria-label="关闭右侧栏标签"
          className="invisible absolute right-1 top-1/2 size-4 shrink-0 !-translate-y-1/2 rounded-full transition-none active:!-translate-y-1/2 group-hover/tab:visible focus-visible:visible [&_svg:not([class*='size-'])]:size-2"
          onClick={(event) => {
            event.preventDefault();
            event.stopPropagation();
            onClose();
          }}
          size="icon-xs"
          title="关闭"
          type="button"
          variant="ghost"
        >
          <X />
        </Button>
      ) : null}
    </div>
  );
}

export function ToolActivitySidebarAddMenu({
  onOpenBrowser,
}: {
  onOpenBrowser?: (targetUrl?: string) => void;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          aria-label="添加右侧栏标签"
          className="size-7 text-muted-foreground transition-none"
          size="icon-sm"
          title="添加右侧栏标签"
          type="button"
          variant="ghost"
        >
          <Plus />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="z-[90] w-44">
        <DropdownMenuItem onClick={() => onOpenBrowser?.()}>
          <Globe />
          <span>内置浏览器</span>
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        {SUPPORTED_SIDEBAR_TABS.map((tab) => {
          const Icon = tab.icon;
          return (
            <DropdownMenuItem disabled key={tab.id}>
              <Icon />
              <span>{tab.label}</span>
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

type SidebarSupportedTab = {
  id: string;
  icon: ComponentType<{ className?: string }>;
  label: string;
  shortcut?: string;
};

const SUPPORTED_SIDEBAR_TABS: SidebarSupportedTab[] = [
  { id: "command", icon: Terminal, label: "命令输出" },
  { id: "file", icon: FileText, label: "文件改动" },
];

function ToolActivityEmptyState({
  onOpenBrowser,
}: {
  onOpenBrowser?: (targetUrl?: string) => void;
}) {
  return (
    <div className="flex min-h-0 flex-1 items-center justify-center px-4 py-8">
      <div
        aria-label="当前支持的侧边栏标签页"
        className="grid w-full max-w-[42rem] gap-2"
        role="list"
      >
        <SupportedSidebarTabItem
          onClick={() => onOpenBrowser?.()}
          tab={{ id: "browser", icon: Globe, label: "内置浏览器" }}
        />
        {SUPPORTED_SIDEBAR_TABS.map((tab) => (
          <SupportedSidebarTabItem tab={tab} key={tab.id} />
        ))}
      </div>
    </div>
  );
}

function SupportedSidebarTabItem({
  onClick,
  tab,
}: {
  onClick?: () => void;
  tab: SidebarSupportedTab;
}) {
  const Icon = tab.icon;

  const className =
    "flex min-w-0 items-center gap-4 rounded-lg bg-muted/25 px-5 py-4 text-foreground/90";
  const content = (
    <>
      <Icon
        className="size-5 shrink-0 text-muted-foreground"
        aria-hidden="true"
      />
      <span className="min-w-0 flex-1 truncate leading-none">
        {tab.label}
      </span>
      {tab.shortcut ? (
        <span className="shrink-0 rounded-full bg-muted px-2.5 py-1 font-mono leading-none text-muted-foreground">
          {tab.shortcut}
        </span>
      ) : null}
    </>
  );

  if (onClick) {
    return (
      <button
        className={cn(
          className,
          "text-left transition-colors hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/30",
        )}
        onClick={onClick}
        role="listitem"
        type="button"
      >
        {content}
      </button>
    );
  }

  return (
    <div className={className} role="listitem">
      {content}
    </div>
  );
}

function ToolActivityTabLabel({ tab }: { tab: ToolActivityTab }) {
  return (
    <>
      {tab.kind === "file" ? (
        <FileText className="size-2.5 shrink-0" />
      ) : (
        <Terminal className="size-3 shrink-0" />
      )}
      <span className="min-w-0 truncate">{tabShortTitle(tab)}</span>
      {statusIcon(tab.status, "size-3 shrink-0")}
    </>
  );
}

function ToolActivityDetail({
  onApplyFileState,
  tab,
}: {
  onApplyFileState?: (
    tab: ToolActivityFileTab,
    targetState: "before" | "after",
  ) => void;
  tab: ToolActivityTab;
}) {
  if (tab.kind === "file") {
    return <FileActivityDetail onApplyFileState={onApplyFileState} tab={tab} />;
  }
  return <CommandActivityDetail tab={tab} />;
}

function FileActivityDetail({
  onApplyFileState,
  tab,
}: {
  onApplyFileState?: (
    tab: ToolActivityFileTab,
    targetState: "before" | "after",
  ) => void;
  tab: ToolActivityFileTab;
}) {
  const body = tab.diff || tab.contentPreview || "";
  const endRef = useRef<HTMLDivElement>(null);
  const scrollAreaRef = useRef<HTMLDivElement>(null);
  const followContentRef = useRef(true);

  useEffect(() => {
    const viewport = scrollAreaRef.current?.querySelector<HTMLElement>(
      "[data-slot=scroll-area-viewport]",
    );
    if (!viewport) return;

    const updateFollowState = () => {
      const distanceToBottom =
        viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight;
      followContentRef.current = distanceToBottom < 24;
    };

    viewport.addEventListener("scroll", updateFollowState);
    updateFollowState();

    return () => {
      viewport.removeEventListener("scroll", updateFollowState);
    };
  }, []);

  useEffect(() => {
    if (!followContentRef.current) return;
    const frame = requestAnimationFrame(() => {
      endRef.current?.scrollIntoView({ block: "end" });
    });
    return () => cancelAnimationFrame(frame);
  }, [body, tab.error, tab.status]);

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="shrink-0 border-b border-border/70 px-1 py-1.5">
        <FileActivityHeader
          onApplyFileState={onApplyFileState}
          tab={tab}
        />
        {tab.revertReason ? (
          <div className="mt-1 px-1 text-[11px] text-muted-foreground">
            {tab.revertReason}
          </div>
        ) : null}
      </div>
      <ScrollArea className="min-h-0 flex-1" ref={scrollAreaRef}>
        <div className="flex min-h-full flex-col gap-3">
          {tab.error ? <ErrorBlock message={tab.error} /> : null}
          {body ? (
            <pre className="min-h-0 whitespace-pre-wrap break-words bg-muted/25 p-2 font-mono text-[11px] leading-relaxed">
              {body.split("\n").map((line, index) => (
                <span className={cn("block min-h-[1.35em]", diffLineClass(line))} key={`${index}:${line}`}>
                  {line || " "}
                </span>
              ))}
            </pre>
          ) : (
            <div className="rounded-md border border-border/70 bg-muted/20 p-3 text-xs text-muted-foreground">
              暂无可显示内容
            </div>
          )}
          <div ref={endRef} />
        </div>
      </ScrollArea>
    </div>
  );
}

function FileActivityHeader({
  onApplyFileState,
  tab,
}: {
  onApplyFileState?: (
    tab: ToolActivityFileTab,
    targetState: "before" | "after",
  ) => void;
  tab: ToolActivityFileTab;
}) {
  const { directory, name } = splitFilePath(fileDisplayPath(tab));
  const openPath = tab.movePath || tab.path;
  const canRevert = tab.revertible !== false;
  const canUnrevert = tab.unrevertible !== false;

  return (
    <div className="flex min-w-0 items-center gap-2">
      <div
        className="relative min-w-0 flex-1 overflow-hidden text-sm leading-none"
        title={fileDisplayPath(tab)}
      >
        <div className="relative left-full flex w-max max-w-none -translate-x-full items-baseline whitespace-nowrap">
          {directory ? <span className="text-muted-foreground">{directory}/</span> : null}
          <span className="font-semibold text-foreground">{name}</span>
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-1.5 font-mono text-xs">
        {typeof tab.additions === "number" ? (
          <span className="text-emerald-600 dark:text-emerald-400">+{tab.additions}</span>
        ) : null}
        {typeof tab.deletions === "number" ? (
          <span className="text-rose-600 dark:text-rose-400">-{tab.deletions}</span>
        ) : null}
        {statusIcon(tab.status, "size-3.5 shrink-0")}
        {tab.status === "success" && !tab.draft && tab.turnId && onApplyFileState ? (
          <>
            <button
              aria-label="回滚此文件改动"
              className="flex size-5 shrink-0 items-center justify-center rounded-sm text-muted-foreground hover:bg-muted hover:text-foreground disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent disabled:hover:text-muted-foreground"
              disabled={!canRevert}
              onClick={() => onApplyFileState(tab, "before")}
              title={canRevert ? "回滚此文件改动" : tab.revertReason || "当前文件状态不可回滚"}
              type="button"
            >
              <Undo2 className="size-3.5" aria-hidden="true" />
            </button>
            <button
              aria-label="恢复此文件改动"
              className="flex size-5 shrink-0 items-center justify-center rounded-sm text-muted-foreground hover:bg-muted hover:text-foreground disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent disabled:hover:text-muted-foreground"
              disabled={!canUnrevert}
              onClick={() => onApplyFileState(tab, "after")}
              title={canUnrevert ? "恢复此文件改动" : tab.revertReason || "当前文件状态不可恢复"}
              type="button"
            >
              <RotateCcw className="size-3.5" aria-hidden="true" />
            </button>
          </>
        ) : null}
        <button
          aria-label="用系统默认应用打开文件"
          className="flex size-5 shrink-0 items-center justify-center rounded-sm text-muted-foreground hover:bg-muted hover:text-foreground"
          onClick={() => {
            void window.aivo?.openPath(openPath).catch((error: unknown) => {
              console.error("Failed to open file", error);
            });
          }}
          title={openPath}
          type="button"
        >
          <ExternalLink className="size-3.5" aria-hidden="true" />
        </button>
      </div>
    </div>
  );
}

function CommandActivityDetail({ tab }: { tab: ToolActivityCommandTab }) {
  const entries = commandEntries(tab);
  const endRef = useRef<HTMLDivElement>(null);
  const scrollAreaRef = useRef<HTMLDivElement>(null);
  const followOutputRef = useRef(true);
  const outputKey = entries
    .map((entry) => `${entry.id}:${entry.stdout.length}:${entry.stderr.length}:${entry.status}`)
    .join("|");

  useEffect(() => {
    const viewport = scrollAreaRef.current?.querySelector<HTMLElement>(
      "[data-slot=scroll-area-viewport]",
    );
    if (!viewport) return;

    const updateFollowState = () => {
      const distanceToBottom =
        viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight;
      followOutputRef.current = distanceToBottom < 24;
    };

    viewport.addEventListener("scroll", updateFollowState);
    updateFollowState();

    return () => {
      viewport.removeEventListener("scroll", updateFollowState);
    };
  }, []);

  useEffect(() => {
    if (!followOutputRef.current) return;
    const frame = requestAnimationFrame(() => {
      endRef.current?.scrollIntoView({ block: "end" });
    });
    return () => cancelAnimationFrame(frame);
  }, [outputKey]);

  return (
    <ScrollArea
      className="min-h-0 flex-1 bg-background [&_[data-slot=scroll-area-viewport]>div]:!block [&_[data-slot=scroll-area-viewport]>div]:!min-w-0 [&_[data-slot=scroll-area-viewport]>div]:!w-full"
      ref={scrollAreaRef}
    >
      <div className="flex min-h-full w-full max-w-full flex-col p-3">
        {entries.map((entry) => (
          <div
            className="min-w-0"
            key={entry.id}
          >
            <pre className="m-0 w-full max-w-full whitespace-pre-wrap break-all font-mono text-[12px] leading-[1.45] text-foreground [overflow-wrap:anywhere]">
              <span>{shellPrompt(entry.cwd)}</span>
              <span>{entry.command}</span>
              {"\n"}
              {entry.stdout ? <span>{terminalOutputSegment(entry.stdout)}</span> : null}
              {entry.stderr ? (
                <span className="text-destructive">{terminalOutputSegment(entry.stderr)}</span>
              ) : null}
              {entry.error && !entry.stderr ? (
                <span className="text-destructive">{terminalOutputSegment(entry.error)}</span>
              ) : null}
            </pre>
          </div>
        ))}
        <span ref={endRef} />
      </div>
    </ScrollArea>
  );
}

function commandEntries(tab: ToolActivityCommandTab) {
  if (Array.isArray(tab.entries) && tab.entries.length > 0) return tab.entries;
  return [
    {
      id: tab.toolCallId || tab.id,
      toolCallId: tab.toolCallId,
      toolName: tab.toolName,
      command: tab.command,
      cwd: tab.cwd,
      status: tab.status,
      stdout: tab.stdout,
      stderr: tab.stderr,
      exitCode: tab.exitCode,
      durationMs: tab.durationMs,
      error: tab.error,
      timeCreated: tab.timeCreated,
      timeUpdated: tab.timeUpdated,
    },
  ];
}

function shellPrompt(cwd?: string) {
  return `agent@aivo ${shellCwdLabel(cwd)} % `;
}

function terminalOutputSegment(content: string) {
  return content.endsWith("\n") ? content : `${content}\n`;
}

function shellCwdLabel(cwd?: string) {
  const value = cwd?.trim();
  if (!value) return "~";
  const parts = value.split("/").filter(Boolean);
  return parts.at(-1) || "/";
}

function ErrorBlock({ message }: { message: string }) {
  return (
    <div className="rounded-md border border-destructive/30 bg-destructive/10 p-2 text-xs text-destructive">
      {message}
    </div>
  );
}

function statusIcon(status: ToolActivityTab["status"], className?: string) {
  switch (status) {
    case "success":
      return <CheckCircle2 className={cn("text-emerald-500", className)} />;
    case "failed":
      return <CircleAlert className={cn("text-destructive", className)} />;
    case "pending_approval":
      return <Clock className={cn("text-amber-500", className)} />;
    default:
      return <Loader2 className={cn("animate-spin text-muted-foreground", className)} />;
  }
}

function tabShortTitle(tab: ToolActivityTab) {
  if (tab.kind === "command") return "Shell";
  const path = tab.movePath || tab.path;
  return path.split("/").filter(Boolean).at(-1) || path;
}

function tabTitle(tab: ToolActivityTab) {
  return tab.kind === "command" ? "Agent Shell" : fileDisplayPath(tab);
}

function fileDisplayPath(tab: ToolActivityFileTab) {
  return tab.movePath ? `${tab.path} -> ${tab.movePath}` : tab.path;
}

function splitFilePath(path: string) {
  const parts = path.split("/");
  const name = parts.pop() || path;
  return {
    directory: parts.join("/"),
    name,
  };
}

function diffLineClass(line: string) {
  if (line.startsWith("+") && !line.startsWith("+++")) {
    return "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300";
  }
  if (line.startsWith("-") && !line.startsWith("---")) {
    return "bg-rose-500/10 text-rose-700 dark:text-rose-300";
  }
  if (line.startsWith("@@")) {
    return "bg-sky-500/10 text-sky-700 dark:text-sky-300";
  }
  return "text-muted-foreground";
}
