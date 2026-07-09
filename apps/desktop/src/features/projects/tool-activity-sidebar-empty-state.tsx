import { Globe } from "lucide-react";

import {
  SUPPORTED_SIDEBAR_TABS,
  type SidebarSupportedTab,
} from "@/features/projects/tool-activity-sidebar-supported-tabs";
import { cn } from "@/lib/utils";

export function ToolActivityEmptyState({
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
      <span className="min-w-0 flex-1 truncate leading-none">{tab.label}</span>
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
