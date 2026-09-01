import type {
  ToolActivityCommandEntry,
  ToolActivityCommandTab,
} from "./tool-activity-types";
import {
  aggregateCommandStatus,
  commandEntryId,
  shellSessionIdFromTabId,
  shellTabId,
} from "./tool-activity-utils";

export function commandTabFromEntries(
  sessionId: string | undefined,
  entries: ToolActivityCommandEntry[],
): ToolActivityCommandTab {
  const fallbackTime = new Date().toISOString();
  const latest = entries.at(-1);
  const status = aggregateCommandStatus(entries);
  return {
    id: shellTabId(sessionId),
    kind: "command",
    entries,
    toolCallId: latest?.toolCallId ?? "",
    turnId: latest?.turnId,
    toolName: latest?.toolName ?? "exec_command",
    processRef: latest?.processRef,
    inputMode: latest?.inputMode,
    inputRequest: latest?.inputRequest,
    attention: latest?.attention,
    inputOwner: latest?.inputOwner,
    leaseMode: latest?.leaseMode,
    leaseVersion: latest?.leaseVersion,
    command: latest?.command ?? "Shell",
    cwd: latest?.cwd,
    status,
    stdout: entries.map((entry) => entry.stdout).join(""),
    stderr: entries.map((entry) => entry.stderr).join(""),
    exitCode: latest?.exitCode,
    durationMs: latest?.durationMs,
    error: latest?.error,
    timeCreated: entries[0]?.timeCreated ?? fallbackTime,
    timeUpdated: latest?.timeUpdated ?? fallbackTime,
  };
}

export function commandEntries(
  tab: ToolActivityCommandTab,
  fallbackEntryId = commandEntryId(tab.toolCallId || tab.id, 0),
): ToolActivityCommandEntry[] {
  if (Array.isArray(tab.entries) && tab.entries.length > 0) return tab.entries;
  return [
    {
      id: fallbackEntryId,
      toolCallId: tab.toolCallId,
      toolName: tab.toolName,
      processRef: tab.processRef,
      inputMode: tab.inputMode,
      inputRequest: tab.inputRequest,
      attention: tab.attention,
      inputOwner: tab.inputOwner,
      leaseMode: tab.leaseMode,
      leaseVersion: tab.leaseVersion,
      command: tab.command,
      cwd: tab.cwd,
      status: tab.status,
      stdout: tab.stdout,
      stderr: tab.stderr,
      exitCode: tab.exitCode,
      durationMs: tab.durationMs,
      replayOfToolCallId: tab.replayOfToolCallId,
      error: tab.error,
      timeCreated: tab.timeCreated,
      timeUpdated: tab.timeUpdated,
    },
  ];
}

export function mergeCommandEntry(
  current: ToolActivityCommandEntry,
  next: ToolActivityCommandEntry,
): ToolActivityCommandEntry {
  return {
    ...current,
    ...next,
    stdout: next.stdout || current.stdout,
    stderr: next.stderr || current.stderr,
  };
}

export function commandSessionId(tab: ToolActivityCommandTab) {
  return shellSessionIdFromTabId(tab.id);
}
