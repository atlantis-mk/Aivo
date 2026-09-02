import {
  SUPPORTED_SIDEBAR_TABS,
  type SidebarSupportedTab,
} from "@/features/projects/tool-activity-sidebar-supported-tabs";

export function ToolActivityEmptyState() {
  return (
    <div className="flex min-h-0 flex-1 items-center justify-center px-4 py-8">
      <div
        aria-label="当前支持的侧边栏标签页"
        className="grid w-full max-w-[42rem] gap-2"
        role="list"
      >
        {SUPPORTED_SIDEBAR_TABS.map((tab) => (
          <SupportedSidebarTabItem tab={tab} key={tab.id} />
        ))}
      </div>
    </div>
  );
}

function SupportedSidebarTabItem({ tab }: { tab: SidebarSupportedTab }) {
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

  return (
    <div className={className} role="listitem">
      {content}
    </div>
  );
}
