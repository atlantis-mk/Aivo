import type {
  ToolActivityCommandTab,
  ToolActivityFileTab,
  ToolActivityTab,
} from "@/features/projects/tool-activity-model";
import { commandEntries as commandTabEntries } from "@/features/projects/tool-activity-command-entry-model";

export type ToolActivitySidebarProps = {
  activeTabId?: string;
  tabs: ToolActivityTab[];
  workspaceRoot: string;
  onActiveTabChange: (tabId: string) => void;
  onApplyFileState?: (
    tab: ToolActivityFileTab,
    targetState: "before" | "after",
  ) => void;
  onCloseTab: (tabId: string) => void;
};

export function commandEntries(tab: ToolActivityCommandTab) {
  return commandTabEntries(tab, tab.toolCallId || tab.id);
}

export function shellPrompt(cwd?: string) {
  return `agent@aivo ${shellCwdLabel(cwd)} % `;
}

export function terminalOutputSegment(content: string) {
  return content.endsWith("\n") ? content : `${content}\n`;
}

export function shellCwdLabel(cwd?: string) {
  const value = cwd?.trim();
  if (!value) return "~";
  const parts = value.split("/").filter(Boolean);
  return parts.at(-1) || "/";
}

export function tabShortTitle(tab: ToolActivityTab) {
  if (tab.kind === "command") return "Shell";
  const path = tab.movePath || tab.path;
  return path.split("/").filter(Boolean).at(-1) || path;
}

export function tabTitle(tab: ToolActivityTab) {
  return tab.kind === "command" ? "Agent Shell" : fileDisplayPath(tab);
}

export function fileDisplayPath(tab: ToolActivityFileTab) {
  return tab.movePath ? `${tab.path} -> ${tab.movePath}` : tab.path;
}

export function splitFilePath(path: string) {
  const parts = path.split("/");
  const name = parts.pop() || path;
  return {
    directory: parts.join("/"),
    name,
  };
}

export function diffLineClass(line: string) {
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
