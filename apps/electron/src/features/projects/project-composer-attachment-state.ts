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
import type { ModelInfo } from "@/lib/provider-catalog";
import { inspectDroppedComposerResources } from "@/services/aivo/project-service";
import type { domain } from "@/types/codex-domain";

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
    if (isNativeComposerDirectory(input)) {
      setAttachments((current) => [
        ...current,
        directoryAttachment(input.path),
      ]);
      return;
    }
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

  async function addDroppedResources(files: File[]) {
    const selections = await inspectDroppedComposerResources(files);
    if (selections.length > 0) {
      const nativeFiles: Extract<
        NonNullable<ComposerAttachmentInput>,
        { kind: "file" }
      >[] = [];
      const { ignoredDirectoryCount } = routeComposerLocalSelections(selections, {
        onDirectory: (path) => {
          setAttachments((current) => [...current, directoryAttachment(path)]);
        },
        onFile: (file) => nativeFiles.push(file),
      });
      for (const file of nativeFiles) {
        await addFiles(file);
      }
      if (ignoredDirectoryCount > 0) {
        toast.info("一次只能添加一个文件夹，已忽略其余文件夹。");
      }
      return;
    }
    await addFiles(files);
  }

  function handleDrop(
    event: DragEvent<HTMLDivElement>,
  ) {
    if (!dragEventHasFiles(event)) return;
    event.preventDefault();
    event.stopPropagation();
    dropDepthRef.current = 0;
    setDropActive(false);
    void addDroppedResources(Array.from(event.dataTransfer.files));
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

function isNativeComposerDirectory(
  input: NonNullable<ComposerAttachmentInput>,
): input is Extract<NonNullable<ComposerAttachmentInput>, { kind: "directory" }> {
  return (
    !Array.isArray(input) &&
    !("length" in input) &&
    input.kind === "directory"
  );
}

function directoryAttachment(path: string): ComposerAttachment {
  return {
    id: crypto.randomUUID(),
    name: path.split(/[\\/]/).filter(Boolean).at(-1) || path,
    mimeType: "inode/directory",
    size: 0,
    kind: "directory",
    data: "",
    path,
  };
}
