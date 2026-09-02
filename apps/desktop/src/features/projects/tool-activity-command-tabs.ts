import {
  OUTPUT_PREVIEW_CHARS,
  type ShellOutputPayload,
  type ToolActivityCommandEntry,
  type ToolActivityCommandTab,
  type ToolActivityTab,
} from "./tool-activity-types";
import {
  commandEntries,
  commandSessionId,
  commandTabFromEntries,
  mergeCommandEntry,
} from "./tool-activity-command-entry-model";
import {
  commandEntryId,
  completedToolActivity,
  previewText,
  shellTabId,
} from "./tool-activity-utils";

export { commandTabFromEntries } from "./tool-activity-command-entry-model";
export { commandTabs } from "./tool-activity-command-tool-call";

export function appendShellOutputToTabs(
  currentTabs: ToolActivityTab[],
  payload: ShellOutputPayload,
) {
  currentTabs = collapseDuplicateCommandTabs(currentTabs);
  const toolCallId = payload.toolCallId?.trim();
  const chunk = payload.chunk ?? "";
  const stream = payload.stream === "stderr" ? "stderr" : "stdout";
  if (!toolCallId || (!chunk && payload.status !== "exited" && payload.status !== "waiting_input")) return currentTabs;

  let changed = false;
  const updatedTabs = currentTabs.map((tab) => {
    if (
      tab.kind !== "command" ||
      !tab.entries.some((entry) => entry.toolCallId === toolCallId)
    ) {
      return tab;
    }
    changed = true;
    return appendCommandOutput(
      tab,
      toolCallId,
      stream,
      chunk,
      payload.timeCreated,
      payload.status,
      payload.processRef,
    );
  });
  if (changed) return updatedTabs;

  const now = payload.timeCreated || new Date().toISOString();
  const nextTab = appendCommandOutput(
    {
      id: shellTabId(payload.sessionId),
      kind: "command",
      entries: [
        {
          id: commandEntryId(toolCallId, 0),
          toolCallId,
          turnId: payload.turnId,
          toolName: "exec_command",
          processRef: payload.processRef,
          command: "Shell command",
          status: "running",
          stdout: "",
          stderr: "",
          timeCreated: now,
          timeUpdated: now,
        },
      ],
      toolCallId,
      turnId: payload.turnId,
      toolName: "exec_command",
      command: "Shell command",
      status: "running",
      stdout: "",
      stderr: "",
      timeCreated: now,
      timeUpdated: now,
    },
    toolCallId,
    stream,
    chunk,
    payload.timeCreated,
    payload.status,
    payload.processRef,
  );
  const existingIndex = currentTabs.findIndex(
    (tab) => tab.kind === "command" && tab.id === nextTab.id,
  );
  if (existingIndex < 0) return [...currentTabs, nextTab];
  return currentTabs.map((tab, index) =>
    index === existingIndex && tab.kind === "command"
      ? mergeCommandTab(tab, nextTab)
      : tab,
  );
}

function collapseDuplicateCommandTabs(tabs: ToolActivityTab[]) {
  const result: ToolActivityTab[] = [];
  const commandIndexes = new Map<string, number>();
  for (const tab of tabs) {
    if (tab.kind !== "command") {
      result.push(tab);
      continue;
    }
    const existingIndex = commandIndexes.get(tab.id);
    if (existingIndex === undefined) {
      commandIndexes.set(tab.id, result.length);
      result.push(tab);
      continue;
    }
    const existing = result[existingIndex];
    if (existing.kind === "command") result[existingIndex] = mergeCommandTab(existing, tab);
  }
  return result;
}

export function mergeCommandTab(
  current: ToolActivityCommandTab,
  next: ToolActivityCommandTab,
): ToolActivityCommandTab {
  const currentEntries = commandEntries(current);
  const nextEntries = commandEntries(next);
  const entriesById = new Map(currentEntries.map((entry) => [entry.id, entry]));
  const order = currentEntries.map((entry) => entry.id);
  for (const nextEntry of nextEntries) {
    const currentEntry = entriesById.get(nextEntry.id);
    entriesById.set(
      nextEntry.id,
      currentEntry ? mergeCommandEntry(currentEntry, nextEntry) : nextEntry,
    );
    if (!currentEntry) order.push(nextEntry.id);
  }
  return commandTabFromEntries(
    commandSessionId(current),
    order.flatMap((id) => {
      const entry = entriesById.get(id);
      return entry ? [entry] : [];
    }),
  );
}

function appendCommandOutput(
  tab: ToolActivityCommandTab,
  toolCallId: string,
  stream: "stdout" | "stderr",
  chunk: string,
  timeUpdated?: string,
  processStatus?: "running" | "waiting_input" | "exited",
  processRef?: string,
): ToolActivityCommandTab {
  const updatedAt = timeUpdated || new Date().toISOString();
  const entries = commandEntries(tab).map((entry) => {
    if (entry.toolCallId !== toolCallId) return entry;
    const status: ToolActivityCommandEntry["status"] =
      processStatus === "exited"
        ? "success"
        : completedToolActivity(entry)
          ? entry.status
          : entry.status === "pending_approval"
            ? entry.status
            : "running";
    return {
      ...entry,
      processRef: processRef || entry.processRef,
      status,
      [stream]: previewText(`${entry[stream]}${chunk}`, OUTPUT_PREVIEW_CHARS),
      timeUpdated: updatedAt,
    };
  });
  return commandTabFromEntries(commandSessionId(tab), entries);
}
