import type { ConversationUserAttachment } from "@/features/projects/conversation-timeline-model";

export const COLLAPSED_USER_MESSAGE_HEIGHT = 420;

export function formatTimelineAttachmentMeta(
  attachment: ConversationUserAttachment,
) {
  if (attachment.kind === "image") return "图片";
  if (attachment.kind === "directory") return "文件夹";
  return readableTimelineAttachmentType(attachment);
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

function readableTimelineAttachmentType(
  attachment: ConversationUserAttachment,
) {
  const extension = attachment.name.match(/\.([^.]+)$/)?.[1]?.trim();
  if (extension && extension.length <= 10) return extension.toUpperCase();
  if (attachment.mimeType === "application/pdf") return "PDF";
  if (attachment.mimeType.startsWith("text/")) return "文本";
  if (attachment.mimeType.includes("json")) return "JSON";
  if (attachment.mimeType.includes("csv")) return "CSV";
  if (attachment.mimeType === "application/octet-stream") return "文件";
  return attachment.mimeType.split("/").at(-1)?.toUpperCase() || "文件";
}
