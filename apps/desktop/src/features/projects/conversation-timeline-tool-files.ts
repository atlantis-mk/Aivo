import type { ToolFileChange } from "./conversation-timeline-tool-types";
import type { domain } from "../../../bridge/go/models";

export function toolFileChangeKey(file: ToolFileChange) {
  return `${file.type}:${file.path}:${file.movePath ?? ""}`;
}

export function getToolCallFileChanges(
  toolCall: domain.ToolCall,
): ToolFileChange[] {
  const resultFiles = parseToolFileChanges(toolCall.result?.files);
  if (resultFiles.length > 0) return resultFiles;
  return parseToolFileChanges(toolCall.arguments?.files);
}

export function toolFileChangeLabel(file: ToolFileChange, live = false) {
  switch (file.type) {
    case "add":
      return live ? "创建中" : "已创建";
    case "delete":
      return live ? "删除中" : "已删除";
    case "move":
      return live ? "移动中" : "已移动";
    default:
      return live ? "编辑中" : "已编辑";
  }
}

export function toolFileChangePath(file: ToolFileChange) {
  return file.movePath ? `${file.path} -> ${file.movePath}` : file.path;
}

export function uniqueToolFileChanges(files: ToolFileChange[]) {
  const unique = new Map<string, ToolFileChange>();
  for (const file of files) {
    unique.set(`${file.path}\u0000${file.movePath ?? ""}`, file);
  }
  return [...unique.values()];
}

function parseToolFileChanges(value: unknown): ToolFileChange[] {
  if (!Array.isArray(value)) return [];
  return value.flatMap((file) => {
    if (!file || typeof file !== "object") return [];
    const record = file as Record<string, unknown>;
    const path = typeof record.path === "string" ? record.path : "";
    if (!path) return [];
    return [
      {
        path,
        movePath:
          typeof record.movePath === "string" ? record.movePath : undefined,
        type: typeof record.type === "string" ? record.type : "update",
        additions: numberValue(record.additions),
        deletions: numberValue(record.deletions),
        diff: typeof record.diff === "string" ? record.diff : undefined,
      },
    ];
  });
}

function numberValue(value: unknown) {
  return typeof value === "number" && Number.isFinite(value) ? value : 0;
}
