import { useRef, useState, type DragEvent } from "react";
import { toast } from "sonner";

import {
  dragEventHasFiles,
  readComposerAttachmentFiles,
  readNativeComposerAttachment,
  routeComposerLocalSelections,
  type ComposerAttachmentInput,
  type ComposerAttachment,
} from "@/features/projects/project-composer-attachments";
import { hasAppBridge } from "@/lib/app-config";
import type { ModelInfo } from "@/lib/provider-catalog";
import {
  inspectDroppedComposerResources,
  type ComposerLocalSelection,
} from "@/services/aivo/project-service";
import type { domain } from "../../../bridge/go/models";

export function useProjectComposerAttachmentState({
  activeModelId,
  activeModelRef,
  modelOptions,
}: {
  activeModelId: string;
  activeModelRef: domain.ModelRef | undefined;
  modelOptions: ModelInfo[];
}) {
  const [attachments, setAttachments] = useState<ComposerAttachment[]>([]);
  const [isDropActive, setDropActive] = useState(false);
  const dropDepthRef = useRef(0);

  async function addFiles(input: ComposerAttachmentInput) {
    if (!input) return;
    const activeModel = modelOptions.find((model) => model.id === activeModelId);
    const result = isNativeComposerFile(input)
      ? readNativeComposerAttachment(input, activeModelRef, activeModel)
      : await readComposerAttachmentFiles(
          Array.from(input),
          activeModelRef,
          activeModel,
        );
    for (const message of result.rejections) {
      toast.error(message);
    }
    if (result.attachments.length === 0) return;
    setAttachments((current) => [...current, ...result.attachments]);
  }

  function handleDragEnter(event: DragEvent<HTMLDivElement>) {
    if (!dragEventHasFiles(event)) return;
    event.preventDefault();
    event.stopPropagation();
    dropDepthRef.current += 1;
    setDropActive(true);
  }

  function handleDragOver(event: DragEvent<HTMLDivElement>) {
    if (!dragEventHasFiles(event)) return;
    event.preventDefault();
    event.stopPropagation();
    event.dataTransfer.dropEffect = "copy";
    setDropActive(true);
  }

  function handleDragLeave(event: DragEvent<HTMLDivElement>) {
    if (!dragEventHasFiles(event)) return;
    event.preventDefault();
    event.stopPropagation();
    dropDepthRef.current = Math.max(0, dropDepthRef.current - 1);
    if (dropDepthRef.current === 0) {
      setDropActive(false);
    }
  }

  async function addDroppedResources(
    files: File[],
    onProjectAdd: (rootPath?: string) => void,
  ) {
    if (
      hasAppBridge() &&
      typeof window.aivo.inspectDroppedComposerResources === "function"
    ) {
      try {
        const selections = await inspectDroppedComposerResources(files);
        if (selections.length === 0) {
          toast.error("未能读取拖放的文件或文件夹。");
          return;
        }
        const nativeFiles: Extract<
          ComposerLocalSelection,
          { kind: "file" }
        >[] = [];
        const { ignoredDirectoryCount } = routeComposerLocalSelections(
          selections,
          {
            onDirectory: onProjectAdd,
            onFile: (file) => nativeFiles.push(file),
          },
        );
        for (const file of nativeFiles) {
          await addFiles(file);
        }
        if (ignoredDirectoryCount > 0) {
          toast.error(
            `一次只能使用一个文件夹，已忽略其余 ${ignoredDirectoryCount} 个文件夹。`,
          );
        }
        return;
      } catch (error) {
        toast.error(
          error instanceof Error ? error.message : "读取拖放资源失败。",
        );
        return;
      }
    }
    await addFiles(files);
  }

  function handleDrop(
    event: DragEvent<HTMLDivElement>,
    onProjectAdd: (rootPath?: string) => void,
  ) {
    if (!dragEventHasFiles(event)) return;
    event.preventDefault();
    event.stopPropagation();
    dropDepthRef.current = 0;
    setDropActive(false);
    void addDroppedResources(Array.from(event.dataTransfer.files), onProjectAdd);
  }

  function removeAttachment(id: string) {
    setAttachments((current) =>
      current.filter((attachment) => attachment.id !== id),
    );
  }

  return {
    addFiles,
    attachments,
    handleDragEnter,
    handleDragLeave,
    handleDragOver,
    handleDrop,
    isDropActive,
    removeAttachment,
    setAttachments,
  };
}

function isNativeComposerFile(
  input: NonNullable<ComposerAttachmentInput>,
): input is Extract<NonNullable<ComposerAttachmentInput>, { kind: "file" }> {
  return !Array.isArray(input) && !("length" in input) && input.kind === "file";
}
