import { useRef, useState, type DragEvent } from "react";
import { toast } from "sonner";

import {
  dragEventHasFiles,
  readComposerAttachmentFiles,
  type ComposerAttachment,
} from "@/features/projects/project-composer-attachments";
import type { ModelInfo } from "@/lib/provider-catalog";
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

  async function addFiles(files: FileList | null) {
    if (!files?.length) return;
    const result = await readComposerAttachmentFiles(
      Array.from(files),
      activeModelRef,
      modelOptions.find((model) => model.id === activeModelId),
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

  function handleDrop(event: DragEvent<HTMLDivElement>) {
    if (!dragEventHasFiles(event)) return;
    event.preventDefault();
    event.stopPropagation();
    dropDepthRef.current = 0;
    setDropActive(false);
    void addFiles(event.dataTransfer.files);
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
