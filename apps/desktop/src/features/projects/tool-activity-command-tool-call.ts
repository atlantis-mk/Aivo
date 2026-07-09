import type { domain } from "../../../bridge/go/models";
import {
  OUTPUT_PREVIEW_CHARS,
  type ToolActivityCommandEntry,
  type ToolActivityCommandTab,
} from "./tool-activity-types";
import { commandTabFromEntries } from "./tool-activity-command-entry-model";
import {
  arrayValue,
  commandEntryId,
  normalizeToolStatus,
  numberValue,
  previewText,
  recordValue,
  stringArg,
  stringValue,
} from "./tool-activity-utils";

export function commandTabs(toolCall: domain.ToolCall): ToolActivityCommandTab[] {
  const entries: ToolActivityCommandEntry[] = [];
  if (toolCall.name === "run_tests") {
    const commands = arrayValue(
      recordValue(toolCall.result?.structured)?.commands,
    );
    if (commands.length > 0) {
      entries.push(
        ...commands.map((command, index) =>
          commandEntryFromStructured(toolCall, recordValue(command), index),
        ),
      );
    }
  }
  if (entries.length === 0) {
    entries.push(
      commandEntryFromStructured(
        toolCall,
        recordValue(toolCall.result?.structured),
        0,
      ),
    );
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
    replayOfToolCallId:
      stringValue(toolCall.result?.replayOfToolCallId) || undefined,
    error: toolCall.error || stringValue(toolCall.result?.error),
    timeCreated: toolCall.timeCreated,
    timeUpdated: toolCall.timeUpdated,
  };
}

function commandFromToolArgs(toolCall: domain.ToolCall) {
  const args = toolCall.arguments ?? {};
  switch (toolCall.name) {
    case "bash":
      return stringArg(args, "command") || "bash";
    case "run_tests":
      return [
        stringArg(args, "target") || "all",
        stringArg(args, "kind") || "auto",
      ]
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
