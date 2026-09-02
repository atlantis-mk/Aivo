import {
  objectRecord,
  objectStringRecord,
  stringArg,
  stringValue,
} from "@/features/projects/conversation-timeline-value-model";
import { parseTime } from "@/features/projects/project-time-model";
import type { AgentRun } from "@/services/aivo";
import type { domain } from "../../../bridge/go/models";

export function findAgentRunForToolCall(
  toolCall: domain.ToolCall,
  agentRuns: AgentRun[],
) {
  const runByToolCallId = agentRuns.find(
    (run) => run.metadata?.toolCallId === toolCall.id,
  );
  if (runByToolCallId) return runByToolCallId;

  const embeddedRun = delegateToolCallAgentRun(toolCall);
  if (embeddedRun?.id) {
    return agentRuns.find((run) => run.id === embeddedRun.id) ?? embeddedRun;
  }
  const sessionId = delegateToolCallSessionId(toolCall);
  if (sessionId) {
    return agentRuns.find((run) => run.sessionId === sessionId);
  }
  const prompt = stringArg(toolCall.arguments ?? {}, "prompt");
  const mode = stringArg(toolCall.arguments ?? {}, "mode");
  const matchingRuns = agentRuns.filter(
    (run) =>
      (!prompt || run.prompt === prompt) && (!mode || run.mode === mode),
  );
  return matchingRuns.length === 1 ? matchingRuns[0] : undefined;
}

export function uniqueDelegateToolCalls(
  toolCalls: domain.ToolCall[],
  agentRuns: AgentRun[],
) {
  const callsByKey = new Map<string, domain.ToolCall>();
  for (const toolCall of toolCalls) {
    const key = delegateToolCallDisplayKey(toolCall, agentRuns);
    const existing = callsByKey.get(key);
    callsByKey.set(
      key,
      existing ? preferredDelegateToolCall(existing, toolCall) : toolCall,
    );
  }
  return [...callsByKey.values()];
}

function delegateToolCallDisplayKey(
  toolCall: domain.ToolCall,
  agentRuns: AgentRun[],
) {
  const agentRun = findAgentRunForToolCall(toolCall, agentRuns);
  if (agentRun?.id) return `run:${agentRun.id}`;
  if (agentRun?.sessionId) return `session:${agentRun.sessionId}`;
  const embeddedRun = delegateToolCallAgentRun(toolCall);
  if (embeddedRun?.id) return `run:${embeddedRun.id}`;
  if (embeddedRun?.sessionId) return `session:${embeddedRun.sessionId}`;
  return `call:${toolCall.id}`;
}

function preferredDelegateToolCall(
  current: domain.ToolCall,
  next: domain.ToolCall,
) {
  const currentRun = delegateToolCallAgentRun(current);
  const nextRun = delegateToolCallAgentRun(next);
  if (!currentRun && nextRun) return next;
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

export function delegateToolCallAgentRun(
  toolCall: domain.ToolCall,
): AgentRun | undefined {
  const structured = objectRecord(toolCall.result?.structured);
  const result = objectRecord(structured?.result);
  if (!result) return undefined;
  const id = stringValue(result.id);
  const sessionId = stringValue(result.sessionId);
  if (!id && !sessionId) return undefined;
  return {
    id: id || sessionId || toolCall.id,
    parentSessionId: stringValue(result.parentSessionId),
    sessionId,
    mode: (stringValue(result.mode) || "assistant") as AgentRun["mode"],
    status: stringValue(result.status) || toolCall.status,
    prompt: stringValue(result.prompt),
    result: stringValue(result.result),
    error: stringValue(result.error),
    metadata: objectStringRecord(result.metadata),
    timeCreated: stringValue(result.timeCreated) || toolCall.timeCreated,
    timeUpdated: stringValue(result.timeUpdated) || toolCall.timeUpdated,
    timeCompleted: stringValue(result.timeCompleted),
  };
}

export function delegateToolCallSessionId(toolCall: domain.ToolCall) {
  return delegateToolCallAgentRun(toolCall)?.sessionId || "";
}

export function subagentStatusLabel(status: string) {
  switch (status) {
    case "running":
      return "运行中";
    case "completed":
    case "success":
      return "已完成";
    case "failed":
      return "失败";
    case "cancelled":
      return "已取消";
    case "pending_approval":
      return "等待批准";
    default:
      return status || "未知";
  }
}

export function subagentStatusClass(status: string) {
  if (status === "completed" || status === "success") {
    return "text-emerald-600 dark:text-emerald-400";
  }
  if (status === "failed" || status === "cancelled") {
    return "text-destructive";
  }
  return "text-muted-foreground";
}

export function agentModeDisplayName(mode: string) {
  switch (mode) {
    case "planner":
      return "规划";
    case "assistant":
      return "助手";
    case "scheduler_worker":
      return "计划任务";
    default:
      return mode || "助手";
  }
}

export function sameAgentRuns(a: AgentRun[], b: AgentRun[]) {
  if (a.length !== b.length) return false;
  return a.every((item, index) => {
    const other = b[index];
    return (
      other &&
      item.id === other.id &&
      item.sessionId === other.sessionId &&
      item.status === other.status &&
      item.result === other.result &&
      item.error === other.error &&
      item.timeUpdated === other.timeUpdated
    );
  });
}
