import type { domain } from "../../../bridge/go/models";

export function codexToolCallFromItem({
  item,
  threadId,
  timeCreated,
  timeUpdated,
  turnId,
}: {
  item: unknown;
  threadId: string;
  timeCreated?: string;
  timeUpdated?: string;
  turnId: string;
}): domain.ToolCall | null {
  const itemRecord = recordFromUnknown(item);
  const itemId = stringValue(itemRecord?.id);
  const itemType = stringValue(itemRecord?.type);
  if (!itemRecord || !itemId || !itemType) return null;

  const now = new Date().toISOString();
  const base = {
    id: `codex:${threadId}:${itemId}`,
    sessionId: threadId,
    status: codexToolStatus(stringValue(itemRecord.status)),
    timeCreated: timeCreated ?? now,
    timeUpdated: timeUpdated ?? now,
    turnId,
  };

  if (itemType === "commandExecution") {
    const command =
      stringValue(itemRecord.command) ?? stringValue(itemRecord.cmd) ?? "";
    const cwd = stringValue(itemRecord.cwd) ?? "";
    const output =
      stringValue(itemRecord.aggregatedOutput) ??
      stringValue(itemRecord.aggregated_output);
    const error = stringValue(itemRecord.error);
    const exitCode = numberValue(itemRecord.exitCode ?? itemRecord.exit_code);
    const state = stringValue(itemRecord.status);
    return {
      ...base,
      arguments: {
        cmd: command,
        workdir: cwd,
      },
      error: error ?? undefined,
      name: "exec_command",
      result: {
        content: output ?? "",
        structured: {
          command,
          cwd,
          exitCode,
          isPty: booleanValue(itemRecord.isPty ?? itemRecord.pty),
          state,
          stdout: output ?? "",
        },
      },
      resultSummary: output ?? undefined,
      status: error ? "failed" : base.status,
    } as domain.ToolCall;
  }

  if (itemType === "mcpToolCall") {
    const result = recordFromUnknown(itemRecord.result);
    const error = stringValue(itemRecord.error);
    return {
      ...base,
      arguments: recordFromUnknown(itemRecord.arguments) ?? {},
      error: error ?? undefined,
      name: [stringValue(itemRecord.server), stringValue(itemRecord.tool)]
        .filter(Boolean)
        .join("/") || "mcp_tool",
      result: result ?? (error ? { error } : undefined),
      status: error ? "failed" : base.status,
    } as domain.ToolCall;
  }

  if (itemType === "webSearch") {
    return {
      ...base,
      arguments: {
        action: itemRecord.action,
        query: stringValue(itemRecord.query) ?? "",
      },
      name: "web_search",
      result: itemRecord.results ? { results: itemRecord.results } : undefined,
    } as domain.ToolCall;
  }

  return null;
}

function codexToolStatus(status: string | null) {
  if (status === "inProgress" || status === "in_progress") return "running";
  if (status === "failed" || status === "declined") return "failed";
  return "completed";
}

function recordFromUnknown(
  value: unknown,
): Record<string, unknown> | undefined {
  return typeof value === "object" && value !== null && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : undefined;
}

function stringValue(value: unknown) {
  return typeof value === "string" ? value : null;
}

function numberValue(value: unknown) {
  return typeof value === "number" && Number.isFinite(value)
    ? value
    : undefined;
}

function booleanValue(value: unknown) {
  return typeof value === "boolean" ? value : false;
}
