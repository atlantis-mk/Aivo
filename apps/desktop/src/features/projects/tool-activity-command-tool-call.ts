import type { domain } from "../../../bridge/go/models";
import {
  OUTPUT_PREVIEW_CHARS,
  type AgentTerminalInputRequest,
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
    processRef: stringValue(structured?.processRef) || undefined,
    inputMode:
      structured?.inputMode === "ask" ||
      structured?.inputMode === "agent_once" ||
      structured?.inputMode === "user_once" ||
      structured?.inputMode === "agent_always"
        ? structured.inputMode
        : undefined,
    inputRequest: agentTerminalInputRequest(structured?.inputRequest),
    attention:
      structured?.attention === "possibly_waiting" || structured?.attention === "interactive"
        ? structured.attention
        : "none",
    inputOwner:
      structured?.inputOwner === "user" || structured?.inputOwner === "agent"
        ? structured.inputOwner
        : "none",
    leaseMode:
      structured?.leaseMode === "once" || structured?.leaseMode === "always"
        ? structured.leaseMode
        : "none",
    leaseVersion: numberValue(structured?.leaseVersion),
    command: stringValue(structured?.command) || fallbackCommand,
    cwd: stringValue(structured?.cwd) || stringArg(args, "workdir"),
    status:
      structured?.status === "running" || structured?.status === "waiting_input"
        ? "running"
        : structured?.status === "exited"
          ? "success"
          : normalizeToolStatus(toolCall.status),
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

function agentTerminalInputRequest(value: unknown): AgentTerminalInputRequest | undefined {
  const request = recordValue(value);
  const mode = request?.mode;
  if (
    typeof request?.id !== "string" ||
    typeof request.cursor !== "number" ||
    (mode !== "ask" && mode !== "agent_once" && mode !== "user_once" && mode !== "agent_always")
  ) return undefined;
  return {
    id: request.id,
    cursor: request.cursor,
    mode: mode as AgentTerminalInputRequest["mode"],
    resolved: request.resolved === true,
    createdAt: typeof request.createdAt === "string" ? request.createdAt : "",
    prompt: typeof request.prompt === "string" ? request.prompt : undefined,
    secret: request.secret === true,
  };
}

function commandFromToolArgs(toolCall: domain.ToolCall) {
  const args = toolCall.arguments ?? {};
  switch (toolCall.name) {
    case "exec_command":
      return stringArg(args, "cmd") || "exec_command";
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
