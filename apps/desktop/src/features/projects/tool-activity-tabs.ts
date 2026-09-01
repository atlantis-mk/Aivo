import { commandTabs, mergeCommandTab } from "./tool-activity-command-tabs";
import { writeFileTabs } from "./tool-activity-file-tabs";
import type { ToolActivityTab } from "./tool-activity-types";
import type { domain } from "../../../bridge/go/models";

export function toolActivityTabsFromToolCall(
  toolCall: domain.ToolCall,
): ToolActivityTab[] {
  switch (toolCall.name) {
    case "write_file":
    case "edit_file":
      return writeFileTabs(toolCall);
    case "exec_command":
    case "write_stdin":
    case "run_tests":
    case "git_status":
    case "git_diff":
      return commandTabs(toolCall);
    default:
      return [];
  }
}

export function toolActivityTabsFromToolCalls(
  toolCalls: domain.ToolCall[],
): ToolActivityTab[] {
  return toolCalls.reduce<ToolActivityTab[]>(
    (tabs, toolCall) =>
      upsertToolActivityTabs(tabs, toolActivityTabsFromToolCall(toolCall)),
    [],
  );
}

export function upsertToolActivityTabs(
  currentTabs: ToolActivityTab[],
  nextTabs: ToolActivityTab[],
) {
  if (nextTabs.length === 0) return currentTabs;
  const nextIds = new Set(nextTabs.map((tab) => tab.id));
  const finalToolCallIds = new Set(
    nextTabs
      .filter((tab) => tab.kind !== "file" || !tab.draft)
      .map((tab) => tab.toolCallId),
  );
  const retainedTabs = currentTabs.filter(
    (tab) =>
      !finalToolCallIds.has(tab.toolCallId) ||
      nextIds.has(tab.id) ||
      tab.kind !== "file",
  );
  const tabsById = new Map<string, ToolActivityTab>();
  const order: string[] = [];
  for (const retainedTab of retainedTabs) {
    const current = tabsById.get(retainedTab.id);
    tabsById.set(
      retainedTab.id,
      current ? mergeToolActivityTab(current, retainedTab) : retainedTab,
    );
    if (!current) order.push(retainedTab.id);
  }
  for (const nextTab of nextTabs) {
    const current = tabsById.get(nextTab.id);
    tabsById.set(
      nextTab.id,
      current ? mergeToolActivityTab(current, nextTab) : nextTab,
    );
    if (!current) order.push(nextTab.id);
  }
  return order.flatMap((id) => {
    const tab = tabsById.get(id);
    return tab ? [tab] : [];
  });
}

function mergeToolActivityTab(
  current: ToolActivityTab,
  next: ToolActivityTab,
): ToolActivityTab {
  if (current.kind !== next.kind) return next;
  if (current.kind === "command" && next.kind === "command") {
    return mergeCommandTab(current, next);
  }
  return { ...current, ...next };
}
