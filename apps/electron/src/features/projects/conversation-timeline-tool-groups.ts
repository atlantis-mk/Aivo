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
import type { domain } from "@/types/codex-domain";

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
  if (!activeGroup) return undefined;
  if (activeGroup.calls.some((call) => call.status === "pending_approval")) {
    return activeGroup.title;
  }
  if (activeGroup.calls.some((call) => call.status === "running")) {
    return activeGroup.title.replace(/^已/, "正在");
  }
  return activeGroup.title;
}

export function groupToolCalls(
  toolCalls: domain.ToolCall[],
  separators: ConversationAssistantTextPart[] = [],
): ToolCallGroup[] {
  const groups: ToolCallGroup[] = [];
  const separatorsWithTimes = separators
    .map((part) => Date.parse(part.timeCreated ?? ""))
    .map((time, index) => ({
      id: separators[index].id,
      index,
      text: separators[index].text,
      time,
    }));
  const usedSeparatorIndexes = new Set<number>();
  for (const toolCall of toolCalls) {
    const separator = toolGroupDescription(
      separatorsWithTimes,
      toolCall,
      usedSeparatorIndexes,
    );
    const last = groups.at(-1);
    if (!separator && last) {
      last.calls.push(toolCall);
      last.kind =
        new Set(last.calls.map(toolCallKind)).size === 1
          ? last.kind
          : "mixed";
      last.title = toolGroupTitle(last.kind, last.calls);
      continue;
    }

    const kind = toolCallKind(toolCall);
    groups.push({
      description: separator,
      id: `segment:${toolCall.id}`,
      kind,
      calls: [toolCall],
      timeCreated: toolCall.timeCreated,
      title: toolGroupTitle(kind, [toolCall]),
    });
  }

  const fallbackSeparatorIndexes = separatorsWithTimes
    .filter(({ index }) => !usedSeparatorIndexes.has(index))
    .map(({ index }) => index)
    .toSorted((left, right) => left - right);
  for (const group of groups) {
    if (group.description || fallbackSeparatorIndexes.length === 0) continue;
    const separatorIndex = fallbackSeparatorIndexes.shift();
    if (separatorIndex === undefined) break;
    group.description =
      separatorsWithTimes[separatorIndex]?.text.trim() || undefined;
  }
  return groups;
}

function toolGroupDescription(
  separatorsWithTimes: {
    id: string;
    index: number;
    text: string;
    time: number;
  }[],
  toolCall: domain.ToolCall,
  usedSeparatorIndexes: Set<number>,
) {
  const separatorItemId = (id: string) => id.replace(/^[^:]+:/, "");
  const linked = separatorsWithTimes.find(
    ({ id, index }) =>
      !usedSeparatorIndexes.has(index) &&
      (id.endsWith(`:${toolCall.id}`) ||
        toolCall.id.endsWith(`:${separatorItemId(id)}`)),
  );
  if (linked) {
    usedSeparatorIndexes.add(linked.index);
    return linked.text.trim() || undefined;
  }

  const toolTime = Date.parse(toolCall.timeCreated ?? "");
  if (Number.isNaN(toolTime)) return undefined;
  const separator = separatorsWithTimes
    .filter(
      ({ index, time }) =>
        !usedSeparatorIndexes.has(index) &&
        !Number.isNaN(time) &&
        time <= toolTime,
    )
    .toSorted((left, right) => {
      const timeDelta = right.time - left.time;
      return timeDelta !== 0 ? timeDelta : right.index - left.index;
    })[0];
  if (!separator) return undefined;
  usedSeparatorIndexes.add(separator.index);
  return separator.text.trim() || undefined;
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
    case "resource_resolve":
      return "resource-resolve";
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
    case "exec_command":
    case "write_stdin":
    case "run_tests":
      return "shell";
    case "agent_delegate_task":
      return "delegate";
    default:
      return "tool";
  }
}

function toolGroupTitle(kind: string, calls: domain.ToolCall[]) {
  const count = calls.length;
  switch (kind) {
    case "mixed":
      return `已执行 ${count} 项操作`;
    case "read":
      return `已探索 ${count} 次读取`;
    case "search":
      return `已探索 ${count} 次搜索`;
    case "list":
      return `已探索 ${count} 次列出`;
    case "resource-resolve":
      return `已解析 ${count} 次资源`;
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
