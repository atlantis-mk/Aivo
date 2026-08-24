import type { DragEvent } from "react";

import type {
  ConversationUserAttachment,
} from "@/features/projects/conversation-timeline-model";
import type { ModelInfo } from "@/lib/provider-catalog";
import type { ComposerLocalSelection } from "@/services/aivo/project-service";
import type { domain } from "../../../bridge/go/models";

const MAX_COMPOSER_ATTACHMENT_BYTES = 50 * 1024 * 1024;

export type ComposerAttachment = {
  id: string;
  name: string;
  mimeType: string;
  size: number;
  kind: "image" | "file";
  data: string;
  text?: string;
  previewUrl?: string;
};

export type ComposerAttachmentInput = FileList | File[] | Extract<
  ComposerLocalSelection,
  { kind: "file" }
> | null;

export function routeComposerLocalSelections(
  selections: ComposerLocalSelection[],
  handlers: {
    onDirectory: (path: string) => void;
    onFile: (file: Extract<ComposerLocalSelection, { kind: "file" }>) => void;
  },
) {
  let selectedDirectory = false;
  let ignoredDirectoryCount = 0;
  for (const selection of selections) {
    if (selection.kind === "file") {
      handlers.onFile(selection);
      continue;
    }
    if (selectedDirectory) {
      ignoredDirectoryCount += 1;
      continue;
    }
    selectedDirectory = true;
    handlers.onDirectory(selection.path);
  }
  return { ignoredDirectoryCount };
}

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
    if (file.size > MAX_COMPOSER_ATTACHMENT_BYTES) {
      rejections.push(`${file.name} 超过 50 MB，不能作为模型附件发送。`);
      continue;
    }
    let mimeType = resolveComposerAttachmentMimeType(file.type, file.name);
    let kind: ComposerAttachment["kind"] = mimeType.startsWith("image/")
      ? "image"
      : "file";
    let isText = isTextComposerAttachment(mimeType, file.name);
    let text: string | undefined;
    if (isText || mimeType === "application/octet-stream") {
      try {
        const decoded = await readFileAsUTF8Text(file);
        if (isText || looksLikeTextContent(decoded)) {
          isText = true;
          kind = "file";
          text = decoded;
          if (mimeType === "application/octet-stream") {
            mimeType = "text/plain";
          }
        }
      } catch {
        if (isText) {
          rejections.push(`${file.name} 不是有效的 UTF-8 文本文件。`);
          continue;
        }
      }
    }
    if (!isText && !isSupportedBinaryComposerAttachment(mimeType)) {
      rejections.push(unsupportedAttachmentTypeMessage(file.name));
      continue;
    }
    if (
      !modelSupportsAttachment(modelRef, modelInfo, kind, mimeType, file.name)
    ) {
      rejections.push(
        `当前模型不支持${attachmentKindLabel(kind, mimeType)}：${file.name}`,
      );
      continue;
    }
    try {
      const data = isText ? "" : await readFileAsBase64(file);
      attachments.push({
        id: crypto.randomUUID(),
        name: file.name || defaultAttachmentName(kind, mimeType),
        mimeType,
        size: file.size,
        kind,
        data,
        text,
        previewUrl: kind === "image" ? `data:${mimeType};base64,${data}` : undefined,
      });
    } catch {
      rejections.push(`${file.name} 读取失败。`);
    }
  }
  return { attachments, rejections };
}

export function readNativeComposerAttachment(
  file: Extract<ComposerLocalSelection, { kind: "file" }>,
  modelRef: domain.ModelRef | null | undefined,
  modelInfo: ModelInfo | undefined,
) {
  let mimeType = resolveComposerAttachmentMimeType(file.mimeType, file.name);
  let kind: ComposerAttachment["kind"] = mimeType.startsWith("image/")
    ? "image"
    : "file";
  let isText = isTextComposerAttachment(mimeType, file.name);
  let text: string | undefined;
  if (isText || mimeType === "application/octet-stream") {
    try {
      const decoded = decodeBase64Text(file.data);
      if (isText || looksLikeTextContent(decoded)) {
        isText = true;
        kind = "file";
        text = decoded;
        if (mimeType === "application/octet-stream") {
          mimeType = "text/plain";
        }
      }
    } catch {
      if (isText) {
        return {
          attachments: [] as ComposerAttachment[],
          rejections: [`${file.name} 不是有效的 UTF-8 文本文件。`],
        };
      }
    }
  }
  if (!isText && !isSupportedBinaryComposerAttachment(mimeType)) {
    return {
      attachments: [] as ComposerAttachment[],
      rejections: [unsupportedAttachmentTypeMessage(file.name)],
    };
  }
  if (
    !modelSupportsAttachment(
      modelRef,
      modelInfo,
      kind,
      mimeType,
      file.name,
    )
  ) {
    return {
      attachments: [] as ComposerAttachment[],
      rejections: [
        `当前模型不支持${attachmentKindLabel(kind, mimeType)}：${file.name}`,
      ],
    };
  }
  const attachment: ComposerAttachment = {
    id: crypto.randomUUID(),
    name: file.name,
    mimeType,
    size: file.size,
    kind,
    data: isText ? "" : file.data,
    text,
    previewUrl: kind === "image"
      ? `data:${mimeType};base64,${file.data}`
      : undefined,
  };
  return { attachments: [attachment], rejections: [] as string[] };
}

export function modelSupportsAttachment(
  modelRef: domain.ModelRef | null | undefined,
  modelInfo: ModelInfo | undefined,
  kind: ComposerAttachment["kind"],
  mimeType: string,
  name = "",
) {
  if (kind === "file" && isTextComposerAttachment(mimeType, name)) {
    return true;
  }
  const capabilities = new Set(
    [...(modelInfo?.capabilities ?? []), ...(modelInfo?.modalities ?? [])].map(
      (item) => item.toLowerCase(),
    ),
  );
  if (kind === "image" && hasAnyCapability(capabilities, ["vision", "image", "image-input", "multimodal"])) {
    return true;
  }
  if (kind === "file" && hasAnyCapability(capabilities, ["attachment", "attachments", "file", "file-input", "document", "pdf", "multimodal"])) {
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

export function attachmentFileTypeLabel(
  attachment: Pick<ComposerAttachment, "mimeType" | "name">,
) {
  const extension = attachment.name.match(/\.([^.]+)$/)?.[1]?.trim();
  if (extension && extension.length <= 10) return extension.toUpperCase();
  if (attachment.mimeType === "application/pdf") return "PDF";
  if (attachment.mimeType === "application/json") return "JSON";
  if (attachment.mimeType.startsWith("text/")) return "TXT";
  return "FILE";
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

async function readFileAsUTF8Text(file: File) {
  return new TextDecoder("utf-8", { fatal: true }).decode(
    await file.arrayBuffer(),
  );
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
    case "docx":
      return "application/vnd.openxmlformats-officedocument.wordprocessingml.document";
    case "xlsx":
      return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet";
    case "pptx":
      return "application/vnd.openxmlformats-officedocument.presentationml.presentation";
    case "txt":
    case "md":
      return "text/plain";
    case "css":
      return "text/css";
    case "html":
    case "htm":
      return "text/html";
    case "js":
    case "jsx":
    case "mjs":
    case "cjs":
      return "text/javascript";
    case "json":
      return "application/json";
    case "csv":
      return "text/csv";
    case "ts":
    case "tsx":
      return "text/typescript";
    case "xml":
      return "application/xml";
    case "yaml":
    case "yml":
      return "application/yaml";
    case "toml":
      return "application/toml";
    case "c":
    case "cc":
    case "conf":
    case "cpp":
    case "cs":
    case "env":
    case "go":
    case "h":
    case "hpp":
    case "ini":
    case "java":
    case "jsonl":
    case "kt":
    case "kts":
    case "log":
    case "lua":
    case "php":
    case "pl":
    case "properties":
    case "py":
    case "rb":
    case "rs":
    case "sh":
    case "sql":
    case "swift":
    case "vue":
      return "text/plain";
    default:
      return "application/octet-stream";
  }
}

function resolveComposerAttachmentMimeType(mimeType: string, name: string) {
  const normalized = mimeType.toLowerCase().split(";", 1)[0].trim();
  if (normalized === "image/jpg") return "image/jpeg";
  if (normalized && normalized !== "application/octet-stream") {
    return normalized;
  }
  return mimeTypeFromName(name);
}

export function isSupportedBinaryComposerAttachment(mimeType: string) {
  return [
    "image/png",
    "image/jpeg",
    "image/gif",
    "image/webp",
    "application/pdf",
    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    "application/vnd.openxmlformats-officedocument.presentationml.presentation",
  ].includes(mimeType.toLowerCase().split(";", 1)[0].trim());
}

export function isTextComposerAttachment(mimeType: string, name: string) {
  const normalizedMimeType = mimeType.toLowerCase().split(";", 1)[0].trim();
  if (normalizedMimeType.startsWith("text/")) return true;
  if (
    [
      "application/json",
      "application/ld+json",
      "application/toml",
      "application/xml",
      "application/x-httpd-php",
      "application/x-sh",
      "application/x-yaml",
      "application/yaml",
    ].includes(normalizedMimeType)
  ) {
    return true;
  }
  return /\.(?:c|cc|conf|cpp|cs|css|env|go|h|hpp|htm|html|ini|java|js|jsx|json|jsonl|kt|kts|log|lua|md|mjs|php|pl|properties|py|rb|rs|sh|sql|swift|toml|ts|tsx|txt|vue|xml|ya?ml)$/i.test(
    name,
  );
}

function decodeBase64Text(data: string) {
  const decoded = atob(data);
  const bytes = Uint8Array.from(decoded, (character) => character.charCodeAt(0));
  return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
}

function looksLikeTextContent(text: string) {
  if (text.length === 0) return false;
  for (const character of text) {
    const codePoint = character.codePointAt(0) ?? 0;
    if (
      codePoint <= 0x1f
      && codePoint !== 0x09
      && codePoint !== 0x0a
      && codePoint !== 0x0d
    ) {
      return false;
    }
  }
  return true;
}

function unsupportedAttachmentTypeMessage(name: string) {
  return `${name} 的文件类型不受支持，不能作为模型附件发送。`;
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
