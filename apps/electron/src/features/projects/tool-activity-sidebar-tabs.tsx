import type { ReactNode } from "react";
import { FileText, Terminal, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import type { ToolActivityTab } from "@/features/projects/tool-activity-model";
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
  | "onActiveTabChange"
  | "onCloseTab"
  | "tabs"
>;

export function ToolActivitySidebarTabs({
  activeTabId,
  tabs,
  onActiveTabChange,
  onCloseTab,
}: ToolActivitySidebarTabsProps) {
  const activeTab =
    tabs.find((tab) => tab.id === activeTabId) ?? tabs.at(-1) ?? null;
  const activeValue = activeTab?.id || "";

  if (tabs.length === 0) return null;

  return (
    <div
      aria-label="右侧栏标签页"
      className="flex min-w-0 flex-1 items-center overflow-hidden"
      role="tablist"
    >
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
    </div>
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
