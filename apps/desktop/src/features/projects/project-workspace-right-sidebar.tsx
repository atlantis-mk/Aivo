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
  browserInitialUrls,
  browserReadyTokens,
  browserTabIds,
  browserVisible,
  enabled,
  onActiveTabChange,
  onApplyFileState,
  onBrowserReady,
  onCloseBrowser,
  onCloseTab,
  onOpenBrowser,
  tabs,
}: {
  activeProjectPage: ProjectWorkspacePage;
  activeTabId: string;
  browserInitialUrls: Record<string, string>;
  browserReadyTokens: Record<string, number>;
  browserTabIds: string[];
  browserVisible: boolean;
  enabled: boolean;
  onActiveTabChange: (tabId: string) => void;
  onApplyFileState: (
    tab: ToolActivityFileTab,
    targetState: "before" | "after",
  ) => void;
  onBrowserReady: (tabId: string) => void;
  onCloseBrowser: (tabId?: string) => void;
  onCloseTab: (tabId: string) => void;
  onOpenBrowser: (targetUrl?: string, requestedTabId?: string) => Promise<void>;
  tabs: ToolActivityTab[];
}) {
  if (activeProjectPage !== "chat" || !enabled) return null;

  return (
    <ToolActivitySidebar
      activeTabId={activeTabId}
      browserInitialUrls={browserInitialUrls}
      browserReadyTokens={browserReadyTokens}
      browserTabIds={browserTabIds}
      browserVisible={browserVisible}
      onActiveTabChange={onActiveTabChange}
      onApplyFileState={onApplyFileState}
      onBrowserReady={onBrowserReady}
      onCloseBrowser={onCloseBrowser}
      onCloseTab={onCloseTab}
      onOpenBrowser={onOpenBrowser}
      tabs={tabs}
    />
  );
}
