import {
  FILE_PREVIEW_CHARS,
  type ToolActivityFileState,
  type ToolActivityFileTab,
  type ToolActivityTab,
  type ToolFileChange,
} from "./tool-activity-types";
import {
  fileStateKey,
  fileTabId,
  normalizeToolStatus,
  numberValue,
  previewText,
  recordValue,
  stringArg,
  stringValue,
} from "./tool-activity-utils";
import type { domain } from "../../../bridge/go/models";

export function annotateToolActivityTabsWithFileStates(
  tabs: ToolActivityTab[],
  states: ToolActivityFileState[],
): ToolActivityTab[] {
  if (tabs.length === 0 || states.length === 0) return tabs;
  const statesByKey = new Map(
    states.map((state) => [
      fileStateKey(state.toolCallId, state.path, state.movePath),
      state,
    ]),
  );
  let changed = false;
  const nextTabs = tabs.map((tab) => {
    if (tab.kind !== "file") return tab;
    const state =
      statesByKey.get(
        fileStateKey(
          tab.toolCallId,
          tab.relativePath || tab.path,
          tab.relativeMovePath,
        ),
      ) ?? statesByKey.get(fileStateKey(tab.toolCallId, tab.path, tab.movePath));
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

export function writeFileTabs(toolCall: domain.ToolCall): ToolActivityFileTab[] {
  const files = getToolCallFileChanges(toolCall);
  if (files.length > 0) {
    return files.map((file) => ({
      id: fileTabId(
        toolCall.id,
        file.fullPath || file.path,
        file.moveFullPath || file.movePath,
      ),
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
