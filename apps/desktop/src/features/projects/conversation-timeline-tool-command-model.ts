import {
  joinCommandParts,
  scalarArg,
  stringArg,
  toolCallArgumentLabels,
  visibleToolArgs,
} from "@/features/projects/conversation-timeline-value-model";
import type { domain } from "../../../bridge/go/models";

export function getToolCallCommand(toolCall: domain.ToolCall) {
  const args = toolCall.arguments ?? {};
  switch (toolCall.name) {
    case "read_file":
      return {
        label: "读取",
        detail: joinCommandParts([
          stringArg(args, "path"),
          ...visibleToolArgs(args, ["path"]),
        ]),
      };
    case "ls":
    case "list_files":
      return {
        label: "列出",
        detail: joinCommandParts([
          stringArg(args, "path"),
          ...visibleToolArgs(args, ["path"]),
        ]),
      };
    case "find":
    case "glob":
      return {
        label: "查找",
        detail: joinCommandParts([
          stringArg(args, "path"),
          stringArg(args, "pattern")
            ? `pattern=${stringArg(args, "pattern")}`
            : "",
          ...visibleToolArgs(args, ["path", "pattern"]),
        ]),
      };
    case "grep":
    case "search_files":
      return {
        label: "搜索",
        detail: joinCommandParts([
          stringArg(args, "query"),
          ...visibleToolArgs(args, ["query"]),
        ]),
      };
    case "tool_resolve":
      return {
        label: "解析工具",
        detail: joinCommandParts([
          stringArg(args, "intent"),
          scalarArg(args, "maxTools") ? `maxTools=${scalarArg(args, "maxTools")}` : "",
          stringArg(args, "category") ? `category=${stringArg(args, "category")}` : "",
          ...visibleToolArgs(args, [
            "intent",
            "maxTools",
            "category",
            "source",
            "riskLevel",
            "required",
          ]),
        ]),
      };
    case "tool_search":
      return {
        label: "搜索工具",
        detail: joinCommandParts([
          stringArg(args, "query"),
          scalarArg(args, "limit") ? `limit=${scalarArg(args, "limit")}` : "",
          ...visibleToolArgs(args, ["query", "limit"]),
        ]),
      };
    case "tool_list":
      return {
        label: "列出工具",
        detail: joinCommandParts([
          stringArg(args, "source") ? `source=${stringArg(args, "source")}` : "",
          stringArg(args, "category") ? `category=${stringArg(args, "category")}` : "",
          stringArg(args, "query"),
          scalarArg(args, "limit") ? `limit=${scalarArg(args, "limit")}` : "",
          ...visibleToolArgs(args, ["source", "category", "query", "limit", "offset"]),
        ]),
      };
    case "tool_detail":
      return {
        label: "工具详情",
        detail: joinCommandParts([
          stringArg(args, "name"),
          ...visibleToolArgs(args, ["name"]),
        ]),
      };
    case "tool_call":
      return {
        label: "调用工具",
        detail: joinCommandParts([
          stringArg(args, "name"),
          ...visibleToolArgs(args, ["name", "arguments"]),
        ]),
      };
    case "skill":
      return {
        label: "技能",
        detail: joinCommandParts([
          stringArg(args, "intent"),
          stringArg(args, "mode") ? `mode=${stringArg(args, "mode")}` : "",
          scalarArg(args, "maxSkills")
            ? `maxSkills=${scalarArg(args, "maxSkills")}`
            : "",
          ...visibleToolArgs(args, ["intent", "mode", "maxSkills", "names"]),
        ]),
      };
    case "write_file":
      return {
        label: "写入",
        detail: joinCommandParts([
          stringArg(args, "path"),
          ...visibleToolArgs(args, ["path", "content"]),
        ]),
      };
    case "edit_file":
      return {
        label: "编辑",
        detail: joinCommandParts([
          stringArg(args, "path"),
          ...visibleToolArgs(args, [
            "path",
            "oldString",
            "newString",
            "replaceAll",
          ]),
        ]),
      };
    case "git_status":
      return {
        label: "Git status",
        detail: "",
      };
    case "git_diff":
      return {
        label: "Git diff",
        detail: joinCommandParts([
          stringArg(args, "path"),
          ...visibleToolArgs(args, ["path"]),
        ]),
      };
    case "bash":
      return {
        label: "Bash",
        detail: joinCommandParts([
          stringArg(args, "command") || stringArg(args, "normalizedCommand"),
          stringArg(args, "cwd") ? `cwd=${stringArg(args, "cwd")}` : "",
        ]),
      };
    case "run_tests":
      return {
        label: "Run tests",
        detail: joinCommandParts([
          stringArg(args, "command") ||
            [stringArg(args, "target"), stringArg(args, "kind")]
              .filter(Boolean)
              .join(":"),
        ]),
      };
    default:
      return {
        label: toolCall.name || "工具",
        detail: joinCommandParts(toolCallArgumentLabels(args)),
      };
  }
}

export function isCommandToolCall(toolCall: domain.ToolCall) {
  return toolCall.name === "bash" || toolCall.name === "run_tests";
}

export function getToolResultText(toolCall: domain.ToolCall) {
  const result = toolCall.result ?? {};
  const content = result.content;
  if (typeof content === "string" && content.trim()) return content;
  if (toolCall.resultSummary?.trim()) return toolCall.resultSummary;
  const error = result.error;
  if (typeof error === "string" && error.trim()) return error;
  return "";
}

export function getRetainedOutputRefs(toolCall: domain.ToolCall) {
  const refs = toolCall.result?.retainedOutputRefs;
  if (!Array.isArray(refs)) return [];
  return refs.filter(
    (ref): ref is string => typeof ref === "string" && Boolean(ref.trim()),
  );
}

export function failedToolRunDetail(
  toolCall: domain.ToolCall,
  resultText: string,
) {
  if (toolCall.name === "format_code") return "format code";
  return commandFromToolResultText(resultText);
}

function commandFromToolResultText(text: string) {
  const commandLine = text
    .split("\n")
    .find((line) => line.trimStart().startsWith("Command:"));
  return commandLine?.replace(/^\s*Command:\s*/, "").trim() ?? "";
}
