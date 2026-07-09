import { useEffect, useState, type ReactNode } from "react";
import { FileText, Globe, Terminal, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import type { ToolActivityTab } from "@/features/projects/tool-activity-model";
import { ToolActivitySidebarAddMenu } from "@/features/projects/tool-activity-sidebar-add-menu";
import {
  tabShortTitle,
  tabTitle,
  type ToolActivitySidebarProps,
} from "@/features/projects/tool-activity-sidebar-model";
import { ToolActivityStatusIcon } from "@/features/projects/tool-activity-status-icon";
import { cn } from "@/lib/utils";

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
  children: ReactNode;
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
          className="invisible absolute right-1 top-1/2 size-4 shrink-0 !-translate-y-1/2 rounded-full transition-none active:!-translate-y-1/2 group-hover/tab:visible focus-visible:visible"
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

function ToolActivityTabLabel({ tab }: { tab: ToolActivityTab }) {
  return (
    <>
      {tab.kind === "file" ? (
        <FileText className="size-2.5 shrink-0" />
      ) : (
        <Terminal className="size-3 shrink-0" />
      )}
      <span className="min-w-0 truncate">{tabShortTitle(tab)}</span>
      <ToolActivityStatusIcon className="size-3 shrink-0" status={tab.status} />
    </>
  );
}
