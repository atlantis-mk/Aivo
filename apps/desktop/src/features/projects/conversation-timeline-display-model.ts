import type { ConversationUserAttachment } from "@/features/projects/conversation-timeline-model";

export const COLLAPSED_USER_MESSAGE_HEIGHT = 420;

export function formatTimelineAttachmentMeta(
  attachment: ConversationUserAttachment,
) {
  const type =
    attachment.kind === "image"
      ? "图片"
      : readableTimelineAttachmentType(attachment.mimeType);
  return attachment.size === undefined
    ? type
    : `${type} · ${formatTimelineBytes(attachment.size)}`;
}

export function shouldShowUserMessageDisclosure(contentHeight: number | null) {
  return (
    contentHeight !== null && contentHeight > COLLAPSED_USER_MESSAGE_HEIGHT
  );
}

export function formatCompletionTime(date: Date) {
  return `${date.getHours()}:${date.getMinutes().toString().padStart(2, "0")}`;
}

export function formatThinkingTime(totalSeconds: number) {
  if (totalSeconds < 60) return `${totalSeconds}秒`;

  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;

  return seconds === 0 ? `${minutes}分钟` : `${minutes}分钟 ${seconds}秒`;
}

function readableTimelineAttachmentType(mimeType: string) {
  if (mimeType === "application/pdf") return "PDF";
  if (mimeType.startsWith("text/")) return "文本";
  if (mimeType.includes("json")) return "JSON";
  if (mimeType.includes("csv")) return "CSV";
  if (mimeType === "application/octet-stream") return "文件";
  return mimeType.split("/").at(-1)?.toUpperCase() || "文件";
}

function formatTimelineBytes(size: number) {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}
