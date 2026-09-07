import type {
  ToolActivityCommandEntry,
  ToolActivityStatus,
} from "./tool-activity-types";

export function completedToolActivity(item: { status: ToolActivityStatus }) {
  return item.status === "success" || item.status === "failed";
}

export function fileTabId(toolCallId: string, path: string, movePath = "") {
  return `file:${toolCallId}:${path}:${movePath}`;
}

export function fileStateKey(toolCallId: string, path: string, movePath = "") {
  return `${toolCallId}:${normalizeActivityPath(path)}:${normalizeActivityPath(movePath)}`;
}

export function normalizeActivityPath(path: string | undefined) {
  return (path || "").replaceAll("\\", "/").replace(/^\.\/+/, "");
}

export function commandEntryId(toolCallId: string, index: number) {
  return `command-entry:${toolCallId}:${index}`;
}

export function shellTabId(sessionId?: string) {
  return `command:shell:${sessionId || "current"}`;
}

export function shellSessionIdFromTabId(tabId: string) {
  return tabId.startsWith("command:shell:")
    ? tabId.slice("command:shell:".length)
    : "current";
}

export function aggregateCommandStatus(
  entries: ToolActivityCommandEntry[],
): ToolActivityStatus {
  return entries.at(-1)?.status ?? "success";
}

export function normalizeToolStatus(status: string): ToolActivityStatus {
  switch (status) {
    case "success":
      return "success";
    case "failed":
    case "interrupted":
    case "cancelled":
      return "failed";
    case "pending_approval":
      return "pending_approval";
    default:
      return "running";
  }
}

export function stringArg(args: Record<string, unknown>, key: string) {
  return stringValue(args[key]);
}

export function stringValue(value: unknown) {
  return typeof value === "string" ? value : "";
}

export function numberValue(value: unknown) {
  return typeof value === "number" && Number.isFinite(value)
    ? value
    : undefined;
}

export function recordValue(value: unknown) {
  if (!value || typeof value !== "object" || Array.isArray(value))
    return undefined;
  return value as Record<string, unknown>;
}

export function arrayValue(value: unknown) {
  return Array.isArray(value) ? value : [];
}

export function previewText(text: string, maxChars: number) {
  if (!text || text.length <= maxChars) return text;
  const headLength = Math.floor(maxChars * 0.65);
  const tailLength = maxChars - headLength;
  const omitted = text.length - maxChars;
  return `${text.slice(0, headLength)}\n\n... omitted ${omitted.toLocaleString()} chars ...\n\n${text.slice(-tailLength)}`;
}
