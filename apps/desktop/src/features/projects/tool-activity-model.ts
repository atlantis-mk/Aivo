import type { domain } from "../../../bridge/go/models";

const OUTPUT_PREVIEW_CHARS = 60_000;
const FILE_PREVIEW_CHARS = 32_000;

export type ToolActivityStatus =
  | "running"
  | "success"
  | "failed"
  | "pending_approval";

export type ToolActivityFileTab = {
  id: string;
  kind: "file";
  draft?: boolean;
  toolCallId: string;
  turnId?: string;
  toolName: string;
  path: string;
  relativePath?: string;
  movePath?: string;
  relativeMovePath?: string;
  operation: string;
  status: ToolActivityStatus;
  contentPreview?: string;
  diff?: string;
  additions?: number;
  deletions?: number;
  baseHash?: string;
  currentHash?: string;
  currentFileHash?: string;
  revertible?: boolean;
  unrevertible?: boolean;
  revertReason?: string;
  error?: string;
  timeCreated: string;
  timeUpdated: string;
};

export type ToolActivityFileState = {
  toolCallId: string;
  path: string;
  movePath?: string;
  revertible?: boolean;
  unrevertible?: boolean;
  reason?: string;
  currentFileHash?: string;
  timeUpdated?: string;
};

export type ToolActivityCommandEntry = {
  id: string;
  toolCallId: string;
  turnId?: string;
  toolName: string;
  command: string;
  cwd?: string;
  status: ToolActivityStatus;
  stdout: string;
  stderr: string;
  exitCode?: number;
  durationMs?: number;
  replayOfToolCallId?: string;
  error?: string;
  timeCreated: string;
  timeUpdated: string;
};

export type ToolActivityCommandTab = ToolActivityCommandEntry & {
  kind: "command";
  entries: ToolActivityCommandEntry[];
};

export type ToolActivityTab = ToolActivityFileTab | ToolActivityCommandTab;

export type ShellOutputPayload = {
  sessionId?: string;
  turnId?: string;
  toolCallId?: string;
  processRef?: string;
  stream?: string;
  chunk?: string;
  sequence?: number;
  timeCreated?: string;
};

type ToolFileChange = {
  path: string;
  fullPath?: string;
  movePath?: string;
  moveFullPath?: string;
  type: string;
  additions: number;
  deletions: number;
  diff?: string;
  baseHash?: string;
  currentHash?: string;
};

export function toolActivityTabsFromToolCall(
  toolCall: domain.ToolCall,
): ToolActivityTab[] {
  switch (toolCall.name) {
    case "write_file":
    case "edit_file":
    case "apply_patch":
      return writeFileTabs(toolCall);
    case "bash":
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
    tabsById.set(nextTab.id, current ? mergeToolActivityTab(current, nextTab) : nextTab);
    if (!current) order.push(nextTab.id);
  }
  return order.flatMap((id) => {
    const tab = tabsById.get(id);
    return tab ? [tab] : [];
  });
}

export function appendShellOutputToTabs(
  currentTabs: ToolActivityTab[],
  payload: ShellOutputPayload,
) {
  const toolCallId = payload.toolCallId?.trim();
  const chunk = payload.chunk ?? "";
  const stream = payload.stream === "stderr" ? "stderr" : "stdout";
  if (!toolCallId || !chunk) return currentTabs;

  let changed = false;
  const updatedTabs = currentTabs.map((tab) => {
    if (tab.kind !== "command" || !tab.entries.some((entry) => entry.toolCallId === toolCallId)) {
      return tab;
    }
    changed = true;
    return appendCommandOutput(tab, toolCallId, stream, chunk, payload.timeCreated);
  });
  if (changed) return updatedTabs;

  const now = payload.timeCreated || new Date().toISOString();
  return [
    ...currentTabs,
    appendCommandOutput(
      {
        id: shellTabId(payload.sessionId),
        kind: "command",
        entries: [
          {
            id: commandEntryId(toolCallId, 0),
            toolCallId,
            turnId: payload.turnId,
            toolName: "bash",
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
        toolName: "bash",
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
    ),
  ];
}

export function completedToolActivity(item: { status: ToolActivityStatus }) {
  return item.status === "success" || item.status === "failed";
}

export function annotateToolActivityTabsWithFileStates(
  tabs: ToolActivityTab[],
  states: ToolActivityFileState[],
): ToolActivityTab[] {
  if (tabs.length === 0 || states.length === 0) return tabs;
  const statesByKey = new Map(
    states.map((state) => [fileStateKey(state.toolCallId, state.path, state.movePath), state]),
  );
  let changed = false;
  const nextTabs = tabs.map((tab) => {
    if (tab.kind !== "file") return tab;
    const state =
      statesByKey.get(fileStateKey(tab.toolCallId, tab.relativePath || tab.path, tab.relativeMovePath)) ??
      statesByKey.get(fileStateKey(tab.toolCallId, tab.path, tab.movePath));
    if (!state) return tab;
    changed = true;
    return {
      ...tab,
      currentFileHash: state.currentFileHash,
      revertible: state.revertible,
      unrevertible: state.unrevertible,
      revertReason: state.reason,
      timeUpdated: state.timeUpdated || tab.timeUpdated,
    };
  });
  return changed ? nextTabs : tabs;
}

function writeFileTabs(toolCall: domain.ToolCall): ToolActivityFileTab[] {
  if (toolCall.name === "apply_patch" && toolCall.result?.draft === true) {
    return applyPatchDraftTabs(toolCall);
  }
  const files = getToolCallFileChanges(toolCall);
  if (files.length > 0) {
    return files.map((file) => ({
      id: fileTabId(toolCall.id, file.fullPath || file.path, file.moveFullPath || file.movePath),
      kind: "file",
      toolCallId: toolCall.id,
      turnId: toolCall.turnId,
      toolName: toolCall.name,
      path: file.fullPath || file.path,
      relativePath: file.path,
      movePath: file.moveFullPath || file.movePath,
      relativeMovePath: file.movePath,
      operation: file.type,
      status: normalizeToolStatus(toolCall.status),
      diff: previewText(file.diff ?? "", FILE_PREVIEW_CHARS),
      additions: file.additions,
      deletions: file.deletions,
      baseHash: file.baseHash,
      currentHash: file.currentHash,
      error: toolCall.error || stringValue(toolCall.result?.error),
      timeCreated: toolCall.timeCreated,
      timeUpdated: toolCall.timeUpdated,
    }));
  }
  const path = stringArg(toolCall.arguments ?? {}, "path");
  if (!path) return [];
  return [
    {
      id: fileTabId(toolCall.id, path),
      kind: "file",
      toolCallId: toolCall.id,
      turnId: toolCall.turnId,
      toolName: toolCall.name,
      path,
      relativePath: path,
      operation: toolCall.name === "write_file" ? "write" : "edit",
      status: normalizeToolStatus(toolCall.status),
      contentPreview: previewText(
        stringArg(toolCall.arguments ?? {}, "content") ||
          stringArg(toolCall.arguments ?? {}, "newString"),
        FILE_PREVIEW_CHARS,
      ),
      error: toolCall.error || stringValue(toolCall.result?.error),
      timeCreated: toolCall.timeCreated,
      timeUpdated: toolCall.timeUpdated,
    },
  ];
}

function applyPatchDraftTabs(toolCall: domain.ToolCall): ToolActivityFileTab[] {
  const files = getToolCallFileChanges(toolCall);
  const patchTextPreview = previewText(
    stringValue(toolCall.result?.patchTextPreview),
    FILE_PREVIEW_CHARS,
  );
  const draftFiles =
    files.length > 0
      ? files
      : [
          {
            path: "apply_patch",
            type: "update",
            additions: 0,
            deletions: 0,
          },
        ];
  return draftFiles.map((file, index) => ({
    id: draftFileTabId(toolCall.id, index),
    kind: "file",
    draft: true,
    toolCallId: toolCall.id,
    turnId: toolCall.turnId,
    toolName: toolCall.name,
    path: file.fullPath || file.path || "apply_patch",
    relativePath: file.path,
    movePath: file.moveFullPath || file.movePath,
    relativeMovePath: file.movePath,
    operation: file.type,
    status: "running",
    contentPreview: patchTextPreview,
    diff: patchTextPreview,
    additions: file.additions,
    deletions: file.deletions,
    error: toolCall.error || stringValue(toolCall.result?.error),
    timeCreated: toolCall.timeCreated,
    timeUpdated: toolCall.timeUpdated,
  }));
}

function commandTabs(toolCall: domain.ToolCall): ToolActivityCommandTab[] {
  const entries: ToolActivityCommandEntry[] = [];
  if (toolCall.name === "run_tests") {
    const commands = arrayValue(recordValue(toolCall.result?.structured)?.commands);
    if (commands.length > 0) {
      entries.push(
        ...commands.map((command, index) =>
          commandEntryFromStructured(toolCall, recordValue(command), index),
        ),
      );
    }
  }
  if (entries.length === 0) {
    entries.push(commandEntryFromStructured(toolCall, recordValue(toolCall.result?.structured), 0));
  }
  return [commandTabFromEntries(toolCall.sessionId, entries)];
}

function commandEntryFromStructured(
  toolCall: domain.ToolCall,
  structured: Record<string, unknown> | undefined,
  index: number,
): ToolActivityCommandEntry {
  const args = toolCall.arguments ?? {};
  const fallbackCommand = commandFromToolArgs(toolCall);
  return {
	    id: commandEntryId(toolCall.id, index),
	    toolCallId: toolCall.id,
	    turnId: toolCall.turnId,
	    toolName: toolCall.name,
    command: stringValue(structured?.command) || fallbackCommand,
    cwd: stringValue(structured?.cwd) || stringArg(args, "cwd"),
    status: normalizeToolStatus(toolCall.status),
    stdout: previewText(stringValue(structured?.stdout), OUTPUT_PREVIEW_CHARS),
    stderr: previewText(stringValue(structured?.stderr), OUTPUT_PREVIEW_CHARS),
    exitCode: numberValue(structured?.exitCode),
    durationMs: numberValue(structured?.durationMs),
    replayOfToolCallId: stringValue(toolCall.result?.replayOfToolCallId) || undefined,
    error: toolCall.error || stringValue(toolCall.result?.error),
    timeCreated: toolCall.timeCreated,
    timeUpdated: toolCall.timeUpdated,
  };
}

function commandTabFromEntries(
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
    toolName: latest?.toolName ?? "bash",
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

function mergeCommandTab(
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
    shellSessionIdFromTabId(current.id),
    order.flatMap((id) => {
      const entry = entriesById.get(id);
      return entry ? [entry] : [];
    }),
  );
}

function mergeCommandEntry(
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

function appendCommandOutput(
  tab: ToolActivityCommandTab,
  toolCallId: string,
  stream: "stdout" | "stderr",
  chunk: string,
  timeUpdated?: string,
): ToolActivityCommandTab {
  const updatedAt = timeUpdated || new Date().toISOString();
  const entries = commandEntries(tab).map((entry) => {
    if (entry.toolCallId !== toolCallId) {
      return entry;
    }
    const status: ToolActivityStatus = completedToolActivity(entry)
      ? entry.status
      : entry.status === "pending_approval"
        ? entry.status
        : "running";
    return {
      ...entry,
      status,
      [stream]: previewText(`${entry[stream]}${chunk}`, OUTPUT_PREVIEW_CHARS),
      timeUpdated: updatedAt,
    };
  });
  return commandTabFromEntries(shellSessionIdFromTabId(tab.id), entries);
}

function commandEntries(tab: ToolActivityCommandTab): ToolActivityCommandEntry[] {
  if (Array.isArray(tab.entries) && tab.entries.length > 0) return tab.entries;
  return [
    {
      id: commandEntryId(tab.toolCallId || tab.id, 0),
      toolCallId: tab.toolCallId,
      toolName: tab.toolName,
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

function commandFromToolArgs(toolCall: domain.ToolCall) {
  const args = toolCall.arguments ?? {};
  switch (toolCall.name) {
    case "bash":
      return stringArg(args, "command") || "bash";
    case "run_tests":
      return [stringArg(args, "target") || "all", stringArg(args, "kind") || "auto"]
        .filter(Boolean)
        .join(":");
    case "git_status":
      return "git status --short --branch";
    case "git_diff":
      return ["git diff", stringArg(args, "path")].filter(Boolean).join(" ");
    default:
      return toolCall.name || "tool";
  }
}

function getToolCallFileChanges(toolCall: domain.ToolCall): ToolFileChange[] {
  const resultFiles = parseToolFileChanges(toolCall.result?.files);
  if (resultFiles.length > 0) return resultFiles;
  return parseToolFileChanges(toolCall.arguments?.files);
}

function parseToolFileChanges(value: unknown): ToolFileChange[] {
  if (!Array.isArray(value)) return [];
  return value.flatMap((file) => {
    const record = recordValue(file);
    const path = stringValue(record?.path);
    const fullPath = stringValue(record?.fullPath);
    if (!path && !fullPath) return [];
    return [
      {
        path,
        fullPath: fullPath || undefined,
        movePath: stringValue(record?.movePath) || undefined,
        moveFullPath: stringValue(record?.moveFullPath) || undefined,
        type: stringValue(record?.type) || "update",
        additions: numberValue(record?.additions) ?? 0,
        deletions: numberValue(record?.deletions) ?? 0,
        diff: stringValue(record?.diff) || undefined,
        baseHash: stringValue(record?.baseHash) || undefined,
        currentHash: stringValue(record?.currentHash) || undefined,
      },
    ];
  });
}

function fileTabId(toolCallId: string, path: string, movePath = "") {
  return `file:${toolCallId}:${path}:${movePath}`;
}

function fileStateKey(toolCallId: string, path: string, movePath = "") {
  return `${toolCallId}:${normalizeActivityPath(path)}:${normalizeActivityPath(movePath)}`;
}

function normalizeActivityPath(path: string | undefined) {
  return (path || "").replaceAll("\\", "/").replace(/^\.\/+/, "");
}

function draftFileTabId(toolCallId: string, index: number) {
  return `file:${toolCallId}:draft:${index}`;
}

function commandEntryId(toolCallId: string, index: number) {
  return `command-entry:${toolCallId}:${index}`;
}

function shellTabId(sessionId?: string) {
  return `command:shell:${sessionId || "current"}`;
}

function shellSessionIdFromTabId(tabId: string) {
  return tabId.startsWith("command:shell:") ? tabId.slice("command:shell:".length) : "current";
}

function aggregateCommandStatus(entries: ToolActivityCommandEntry[]): ToolActivityStatus {
  return entries.at(-1)?.status ?? "success";
}

function normalizeToolStatus(status: string): ToolActivityStatus {
  switch (status) {
    case "success":
      return "success";
    case "failed":
    case "interrupted":
    case "cancelled":
      return "failed";
    case "pending_approval":
      return "pending_approval";
    default:
      return "running";
  }
}

function stringArg(args: Record<string, unknown>, key: string) {
  return stringValue(args[key]);
}

function stringValue(value: unknown) {
  return typeof value === "string" ? value : "";
}

function numberValue(value: unknown) {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

function recordValue(value: unknown) {
  if (!value || typeof value !== "object" || Array.isArray(value)) return undefined;
  return value as Record<string, unknown>;
}

function arrayValue(value: unknown) {
  return Array.isArray(value) ? value : [];
}

function previewText(text: string, maxChars: number) {
  if (!text || text.length <= maxChars) return text;
  const headLength = Math.floor(maxChars * 0.65);
  const tailLength = maxChars - headLength;
  const omitted = text.length - maxChars;
  return `${text.slice(0, headLength)}\n\n... omitted ${omitted.toLocaleString()} chars ...\n\n${text.slice(-tailLength)}`;
}
