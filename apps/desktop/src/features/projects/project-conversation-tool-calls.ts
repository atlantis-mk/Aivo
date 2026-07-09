import { parseTime } from "@/features/projects/project-time-model";
import { recordFromUnknown, stringFromUnknown } from "./project-conversation-payload";
import type { domain } from "../../../bridge/go/models";

export function toolCallsForTurn(toolCalls: domain.ToolCall[], turnId?: string) {
  if (!turnId) return [];
  return toolCalls
    .filter((toolCall) => toolCall.turnId === turnId)
    .toSorted((a, b) => {
      const timeDelta =
        parseTime(a.timeCreated).getTime() - parseTime(b.timeCreated).getTime();
      if (timeDelta !== 0) return timeDelta;
      return a.id.localeCompare(b.id);
    });
}

export function groupToolCallsByTurnId(toolCalls: domain.ToolCall[]) {
  const callsByTurnId = new Map<string, domain.ToolCall[]>();
  for (const toolCall of toolCalls) {
    if (!toolCall.turnId) continue;
    const calls = callsByTurnId.get(toolCall.turnId) ?? [];
    calls.push(toolCall);
    callsByTurnId.set(toolCall.turnId, calls);
  }
  for (const [turnId, calls] of callsByTurnId) {
    callsByTurnId.set(
      turnId,
      calls.toSorted((a, b) => {
        const timeDelta =
          parseTime(a.timeCreated).getTime() -
          parseTime(b.timeCreated).getTime();
        if (timeDelta !== 0) return timeDelta;
        return a.id.localeCompare(b.id);
      }),
    );
  }
  return callsByTurnId;
}

export function mergeToolCallLists(
  currentToolCalls: domain.ToolCall[],
  nextToolCalls: domain.ToolCall[],
) {
  if (currentToolCalls.length === 0) return dedupeDelegateToolCalls(nextToolCalls);
  if (nextToolCalls.length === 0) return currentToolCalls;

  const callsById = new Map<string, domain.ToolCall>();
  for (const toolCall of currentToolCalls) {
    callsById.set(toolCall.id, toolCall);
  }
  for (const toolCall of nextToolCalls) {
    callsById.set(toolCall.id, toolCall);
  }

  return dedupeDelegateToolCalls([...callsById.values()]).toSorted((a, b) => {
    const timeDelta =
      parseTime(a.timeCreated).getTime() - parseTime(b.timeCreated).getTime();
    if (timeDelta !== 0) return timeDelta;
    return a.id.localeCompare(b.id);
  });
}

export function isDelegateTaskToolName(name: string) {
  return name === "agent_delegate_task";
}

function dedupeDelegateToolCalls(toolCalls: domain.ToolCall[]) {
  const output: domain.ToolCall[] = [];
  const indexByKey = new Map<string, number>();

  for (const toolCall of toolCalls) {
    const key = delegateToolCallIdentityKey(toolCall);
    if (!key) {
      output.push(toolCall);
      continue;
    }

    const existingIndex = indexByKey.get(key);
    if (existingIndex === undefined) {
      indexByKey.set(key, output.length);
      output.push(toolCall);
      continue;
    }

    output[existingIndex] = preferredDelegateToolCall(
      output[existingIndex],
      toolCall,
    );
  }

  return output;
}

function delegateToolCallIdentityKey(toolCall: domain.ToolCall) {
  if (!isDelegateTaskToolName(toolCall.name)) return "";
  const run = delegateToolCallStructuredRun(toolCall);
  const runId = stringFromUnknown(run?.id);
  const sessionId = stringFromUnknown(run?.sessionId);
  const metadata = recordFromUnknown(run?.metadata);
  const toolCallId = stringFromUnknown(metadata?.toolCallId);
  if (sessionId) return `delegate:session:${sessionId}`;
  if (runId) return `delegate:run:${runId}`;
  if (toolCallId) return `delegate:tool-call:${toolCallId}`;
  return "";
}

function delegateToolCallStructuredRun(toolCall: domain.ToolCall) {
  const structured = recordFromUnknown(toolCall.result?.structured);
  return recordFromUnknown(structured?.result);
}

function preferredDelegateToolCall(
  current: domain.ToolCall,
  next: domain.ToolCall,
) {
  if (
    !delegateToolCallStructuredRun(current) &&
    delegateToolCallStructuredRun(next)
  ) {
    return next;
  }
  if (current.status === "running" && next.status !== "running") return next;
  if (!current.result && next.result) return next;
  if (
    parseTime(next.timeUpdated).getTime() >
    parseTime(current.timeUpdated).getTime()
  ) {
    return next;
  }
  return current;
}
