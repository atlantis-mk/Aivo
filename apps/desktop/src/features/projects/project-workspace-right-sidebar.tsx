import {
  ToolActivitySidebar,
} from "@/features/projects/tool-activity-sidebar";
import type {
  ToolActivityFileTab,
  ToolActivityTab,
} from "@/features/projects/tool-activity-types";
import type { ProjectWorkspacePage } from "@/features/projects/project-workspace-derived-state";

export function ProjectWorkspaceRightSidebar({
  activeProjectPage,
  activeTabId,
  enabled,
  onActiveTabChange,
  onApplyFileState,
  onCloseTab,
  tabs,
  workspaceRoot,
}: {
  activeProjectPage: ProjectWorkspacePage;
  activeTabId: string;
  enabled: boolean;
  onActiveTabChange: (tabId: string) => void;
  onApplyFileState: (
    tab: ToolActivityFileTab,
    targetState: "before" | "after",
  ) => void;
  onCloseTab: (tabId: string) => void;
  tabs: ToolActivityTab[];
  workspaceRoot: string;
}) {
  if (activeProjectPage !== "chat" || !enabled) return null;

  return (
    <ToolActivitySidebar
      activeTabId={activeTabId}
      onActiveTabChange={onActiveTabChange}
      onApplyFileState={onApplyFileState}
      onCloseTab={onCloseTab}
      tabs={tabs}
      workspaceRoot={workspaceRoot}
    />
  );
}
