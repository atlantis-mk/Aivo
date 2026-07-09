import type { DragEvent } from "react";

import type {
  ConversationUserAttachment,
} from "@/features/projects/conversation-timeline-model";
import type { ModelInfo } from "@/lib/provider-catalog";
import type { domain } from "../../../bridge/go/models";

const MAX_COMPOSER_ATTACHMENT_BYTES = 50 * 1024 * 1024;

export type ComposerAttachment = {
  id: string;
  name: string;
  mimeType: string;
  size: number;
  kind: "image" | "file";
  data: string;
  previewUrl?: string;
};

export function dragEventHasFiles(event: DragEvent<HTMLElement>) {
  return Array.from(event.dataTransfer.types).includes("Files");
}

export async function readComposerAttachmentFiles(
  files: File[],
  modelRef: domain.ModelRef | null | undefined,
  modelInfo: ModelInfo | undefined,
) {
  const attachments: ComposerAttachment[] = [];
  const rejections: string[] = [];
  for (const file of files) {
    const mimeType = file.type || mimeTypeFromName(file.name);
    const kind = mimeType.startsWith("image/") ? "image" : "file";
    if (file.size > MAX_COMPOSER_ATTACHMENT_BYTES) {
      rejections.push(`${file.name} 超过 50 MB，不能作为模型附件发送。`);
      continue;
    }
    if (!modelSupportsAttachment(modelRef, modelInfo, kind, mimeType)) {
      rejections.push(
        `当前模型不支持${attachmentKindLabel(kind, mimeType)}：${file.name}`,
      );
      continue;
    }
    try {
      const data = await readFileAsBase64(file);
      attachments.push({
        id: crypto.randomUUID(),
        name: file.name || defaultAttachmentName(kind, mimeType),
        mimeType,
        size: file.size,
        kind,
        data,
        previewUrl: kind === "image" ? `data:${mimeType};base64,${data}` : undefined,
      });
    } catch {
      rejections.push(`${file.name} 读取失败。`);
    }
  }
  return { attachments, rejections };
}

export function modelSupportsAttachment(
  modelRef: domain.ModelRef | null | undefined,
  modelInfo: ModelInfo | undefined,
  kind: ComposerAttachment["kind"],
  mimeType: string,
) {
  const capabilities = new Set(
    [...(modelInfo?.capabilities ?? []), ...(modelInfo?.modalities ?? [])].map(
      (item) => item.toLowerCase(),
    ),
  );
  if (kind === "image" && hasAnyCapability(capabilities, ["vision", "image", "image-input", "multimodal"])) {
    return true;
  }
  if (kind === "file" && hasAnyCapability(capabilities, ["file", "file-input", "document", "pdf", "multimodal"])) {
    return true;
  }
  const providerId = normalizeProviderIdForCapability(modelRef?.providerId);
  const modelId = modelRef?.modelId.toLowerCase() ?? "";
  if (!providerId || !modelId) return false;
  if (providerId === "openai" || providerId === "azure-openai") {
    return isLatestOpenAIMultimodalModel(modelId);
  }
  if (providerId === "anthropic" || providerId === "claude-code") {
    if (!isCurrentClaudeModel(modelId)) return false;
    return kind === "image" || mimeType === "application/pdf";
  }
  if (providerId === "google" || providerId === "gemini" || providerId === "google-vertex") {
    return modelId.includes("gemini-");
  }
  return false;
}

export function attachmentKindLabel(
  kind: ComposerAttachment["kind"],
  mimeType: string,
) {
  if (kind === "image") return "图片输入";
  if (mimeType === "application/pdf") return "PDF 文件输入";
  return "文件输入";
}

export function composerAttachmentToConversationAttachment(
  attachment: ComposerAttachment,
): ConversationUserAttachment {
  return {
    id: attachment.id,
    name: attachment.name,
    mimeType: attachment.mimeType,
    kind: attachment.kind,
    previewUrl: attachment.previewUrl,
    size: attachment.size,
  };
}

export function formatAttachmentMeta(attachment: ComposerAttachment) {
  const type = attachment.kind === "image" ? "图片" : readableAttachmentType(attachment.mimeType);
  return `${type} · ${formatBytes(attachment.size)}`;
}

export function formatAttachmentOnlyPrompt(attachments: ComposerAttachment[]) {
  if (attachments.length === 0) return "附件";
  if (attachments.length === 1) return `附件：${attachments[0].name}`;
  return `附件：${attachments.length} 个文件`;
}

function readFileAsBase64(file: File) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onerror = () => reject(reader.error);
    reader.onload = () => {
      const result = typeof reader.result === "string" ? reader.result : "";
      resolve(result.includes(",") ? result.split(",").at(-1) || "" : result);
    };
    reader.readAsDataURL(file);
  });
}

function hasAnyCapability(capabilities: Set<string>, values: string[]) {
  return values.some((value) => capabilities.has(value));
}

function normalizeProviderIdForCapability(providerId: string | undefined) {
  const normalized = providerId?.toLowerCase().trim() ?? "";
  if (normalized === "claude") return "anthropic";
  if (normalized === "vertex") return "google-vertex";
  return normalized;
}

function isLatestOpenAIMultimodalModel(modelId: string) {
  return /(^|[/:-])(gpt-5|gpt-4o|gpt-4\.1|o3|o4)/.test(modelId);
}

function isCurrentClaudeModel(modelId: string) {
  return modelId.includes("claude-3") || modelId.includes("claude-4") || modelId.includes("claude-5") || modelId.includes("sonnet") || modelId.includes("opus") || modelId.includes("haiku") || modelId.includes("fable") || modelId.includes("mythos");
}

function mimeTypeFromName(name: string) {
  const extension = name.toLowerCase().split(".").pop() ?? "";
  switch (extension) {
    case "png":
      return "image/png";
    case "jpg":
    case "jpeg":
      return "image/jpeg";
    case "gif":
      return "image/gif";
    case "webp":
      return "image/webp";
    case "pdf":
      return "application/pdf";
    case "txt":
    case "md":
      return "text/plain";
    case "json":
      return "application/json";
    case "csv":
      return "text/csv";
    default:
      return "application/octet-stream";
  }
}

function defaultAttachmentName(kind: ComposerAttachment["kind"], mimeType: string) {
  if (kind === "image") return `pasted-image.${mimeType.split("/").at(-1) || "png"}`;
  return "attachment";
}

function readableAttachmentType(mimeType: string) {
  if (mimeType === "application/pdf") return "PDF";
  if (mimeType.includes("spreadsheet") || mimeType.includes("csv")) return "表格";
  if (mimeType.includes("presentation")) return "演示文稿";
  if (mimeType.includes("wordprocessing") || mimeType.includes("document")) return "文档";
  if (mimeType.startsWith("text/") || mimeType.includes("json")) return "文本";
  return "文件";
}

function formatBytes(size: number) {
  if (!Number.isFinite(size) || size <= 0) return "0 B";
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}
