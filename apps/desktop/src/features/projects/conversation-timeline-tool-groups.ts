import type {
  ConversationAssistantTextPart,
} from "@/features/projects/conversation-timeline-model";
import { stringArg } from "@/features/projects/conversation-timeline-value-model";
import {
  getToolCallFileChanges,
  toolFileChangeLabel,
  uniqueToolFileChanges,
} from "./conversation-timeline-tool-files";
import type { ToolCallGroup } from "./conversation-timeline-tool-types";
import type { domain } from "../../../bridge/go/models";

const hiddenToolCallNames = new Set(["update_plan"]);

export function toolActionHeading(toolGroups: ToolCallGroup[]) {
  if (toolGroups.length === 0) return undefined;
  const activeGroup =
    toolGroups.find((group) =>
      group.calls.some(
        (call) =>
          call.status === "running" || call.status === "pending_approval",
      ),
    ) ?? toolGroups.at(-1);
  return activeGroup?.title;
}

export function groupToolCalls(
  toolCalls: domain.ToolCall[],
  separators: ConversationAssistantTextPart[] = [],
): ToolCallGroup[] {
  const groups: ToolCallGroup[] = [];
  const separatorTimes = separators
    .map((part) => Date.parse(part.timeCreated ?? ""))
    .filter((time) => !Number.isNaN(time))
    .toSorted((a, b) => a - b);
  for (const toolCall of toolCalls) {
    const kind = toolCallKind(toolCall);
    const last = groups.at(-1);
    if (
      last?.kind === kind &&
      !hasSeparatorBetweenToolCalls(separatorTimes, last.calls.at(-1), toolCall)
    ) {
      last.calls.push(toolCall);
      last.title = toolGroupTitle(kind, last.calls);
      continue;
    }
    groups.push({
      description: toolGroupDescription(separators, toolCall),
      id: `${kind}:${toolCall.id}`,
      kind,
      calls: [toolCall],
      timeCreated: toolCall.timeCreated,
      title: toolGroupTitle(kind, [toolCall]),
    });
  }
  return groups;
}

function toolGroupDescription(
  separators: ConversationAssistantTextPart[],
  toolCall: domain.ToolCall,
) {
  const toolTime = Date.parse(toolCall.timeCreated ?? "");
  if (Number.isNaN(toolTime)) {
    return separators.length === 1 ? separators[0].text.trim() : undefined;
  }
  return separators
    .map((separator, index) => ({
      index,
      separator,
      time: Date.parse(separator.timeCreated ?? ""),
    }))
    .filter(({ time }) => !Number.isNaN(time) && time <= toolTime)
    .toSorted((left, right) => {
      const timeDelta = right.time - left.time;
      return timeDelta !== 0 ? timeDelta : right.index - left.index;
    })[0]
    ?.separator.text.trim();
}

export function filterVisibleToolCalls(toolCalls: domain.ToolCall[]) {
  const visible: domain.ToolCall[] = [];
  const laterGlobPatterns: string[] = [];

  for (let index = toolCalls.length - 1; index >= 0; index -= 1) {
    const toolCall = toolCalls[index];
    if (hiddenToolCallNames.has(toolCall.name)) continue;

    if (toolCall.name === "find" || toolCall.name === "glob") {
      const pattern = stringArg(toolCall.arguments ?? {}, "pattern")
        .trim()
        .toLowerCase();
      if (pattern) laterGlobPatterns.push(pattern);
      visible.push(toolCall);
      continue;
    }

    if (toolCall.name === "grep" || toolCall.name === "search_files") {
      const query = stringArg(toolCall.arguments ?? {}, "query")
        .trim()
        .toLowerCase();
      if (query && laterGlobPatterns.some((pattern) => pattern.includes(query))) {
        continue;
      }
    }

    visible.push(toolCall);
  }

  return visible.reverse();
}

export function toolCallKind(toolCall: domain.ToolCall) {
  switch (toolCall.name) {
    case "read":
    case "read_file":
      return "read";
    case "find":
    case "grep":
    case "glob":
    case "search_files":
      return "search";
    case "ls":
    case "list_files":
      return "list";
    case "tool_resolve":
      return "tool-resolve";
    case "tool_search":
      return "tool-search";
    case "tool_list":
      return "tool-list";
    case "tool_detail":
      return "tool-detail";
    case "tool_call":
      return "tool-bridge";
    case "write":
    case "edit":
    case "write_file":
    case "edit_file":
      return "write";
    case "git_status":
    case "git_diff":
      return "git";
    case "bash":
    case "run_tests":
      return "shell";
    case "agent_delegate_task":
      return "delegate";
    default:
      return "tool";
  }
}

function hasSeparatorBetweenToolCalls(
  separatorTimes: number[],
  previous: domain.ToolCall | undefined,
  next: domain.ToolCall,
) {
  if (!previous || separatorTimes.length === 0) return false;
  const previousTime = Date.parse(previous.timeCreated ?? "");
  const nextTime = Date.parse(next.timeCreated ?? "");
  if (Number.isNaN(previousTime) || Number.isNaN(nextTime)) return false;
  return separatorTimes.some((time) => time > previousTime && time <= nextTime);
}

function toolGroupTitle(kind: string, calls: domain.ToolCall[]) {
  const count = calls.length;
  switch (kind) {
    case "read":
      return `已探索 ${count} 次读取`;
    case "search":
      return `已探索 ${count} 次搜索`;
    case "list":
      return `已探索 ${count} 次列出`;
    case "tool-resolve":
      return `已解析 ${count} 次工具`;
    case "tool-search":
      return `已搜索 ${count} 次工具`;
    case "tool-list":
      return `已列出 ${count} 次工具`;
    case "tool-detail":
      return `已查看 ${count} 次工具详情`;
    case "tool-bridge":
      return `已调用 ${count} 次工具`;
    case "write":
      return writeToolGroupTitle(calls);
    case "git":
      return `已检查 ${count} 次 Git`;
    case "shell":
      return shellToolGroupTitle(calls);
    case "delegate":
      return count === 1 ? "已启动 1 个子代理" : `已启动 ${count} 个子代理`;
    default:
      return `已探索 ${count} 次工具调用`;
  }
}

function shellToolGroupTitle(calls: domain.ToolCall[]) {
  const failed = calls.some((call) => call.status === "failed");
  const pending = calls.some((call) => call.status === "pending_approval");
  if (pending) return `等待批准 ${calls.length} 条命令`;
  if (failed) return `已运行 ${calls.length} 条命令，存在失败`;
  return `已运行 ${calls.length} 条命令`;
}

function writeToolGroupTitle(calls: domain.ToolCall[]) {
  const files = uniqueToolFileChanges(calls.flatMap(getToolCallFileChanges));
  if (files.length === 0) return `已请求 ${calls.length} 次写入`;
  const labels = new Set(files.map((file) => toolFileChangeLabel(file)));
  const label = labels.size === 1 ? [...labels][0] : "已更新";
  return `${label} ${files.length} 个文件`;
}
