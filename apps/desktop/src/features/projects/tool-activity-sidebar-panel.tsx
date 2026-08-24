import { ToolActivityDetail } from "@/features/projects/tool-activity-detail";
import { ToolActivityEmptyState } from "@/features/projects/tool-activity-sidebar-empty-state";
import { ToolActivitySidebarTabs } from "@/features/projects/tool-activity-sidebar-tabs";
import { upsertToolActivityTabs } from "@/features/projects/tool-activity-model";
import type { ToolActivitySidebarProps } from "@/features/projects/tool-activity-sidebar-model";

export function ToolActivitySidebar({
  activeTabId,
  tabs,
  workspaceRoot,
  onActiveTabChange,
  onApplyFileState,
  onCloseTab,
}: ToolActivitySidebarProps) {
  const normalizedTabs = upsertToolActivityTabs([], tabs);
  const activeTab =
    normalizedTabs.find((tab) => tab.id === activeTabId) ??
    normalizedTabs.at(-1) ??
    null;
  const hasTabs = normalizedTabs.length > 0;

  return (
    <section
      className="relative z-[70] flex h-full min-h-0 flex-col bg-background"
      data-app-no-drag
    >
      {hasTabs ? (
        <div className="flex h-9 shrink-0 items-center pl-2 pr-(--project-right-topbar-actions-width)">
          <ToolActivitySidebarTabs
            activeTabId={activeTabId}
            onActiveTabChange={onActiveTabChange}
            onCloseTab={onCloseTab}
            tabs={normalizedTabs}
          />
        </div>
      ) : null}
      {hasTabs ? (
        <div className="min-h-0 flex-1 overflow-hidden">
          {activeTab ? (
            <ToolActivityDetail
              onApplyFileState={onApplyFileState}
              tab={activeTab}
              workspaceRoot={workspaceRoot}
            />
          ) : (
            <ToolActivityEmptyState />
          )}
        </div>
      ) : (
        <ToolActivityEmptyState />
      )}
    </section>
  );
}
