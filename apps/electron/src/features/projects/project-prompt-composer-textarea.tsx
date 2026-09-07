import { useMemo, useState } from "react";

import { ArrowRight01Icon, Cancel01Icon, File02Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { toast } from "sonner";
import {
  Attachment,
  AttachmentAction,
  AttachmentActions,
  AttachmentContent,
  AttachmentGroup,
  AttachmentMedia,
  AttachmentTitle,
} from "@/components/ui/attachment";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  consumePromptMentionQuery,
  promptComposerEnterAction,
  promptMentionRange,
  type PromptMentionAction,
  type PromptMentionRange,
  type PromptMentionReference,
} from "@/features/projects/project-prompt-mention-model";
import {
  inferPromptPasteName,
  isExternalPromptPasteTarget,
  promptPasteTitle,
  promptPasteTarget,
  type PendingPromptPaste,
} from "@/features/projects/project-prompt-paste";
import { PromptMentionPicker } from "@/features/projects/project-prompt-mention-picker";
import { PromptMentionTags } from "@/features/projects/project-prompt-mention-tags";
import type {
  AutoTextareaHeightRef,
  PromptComposerProps,
} from "@/features/projects/project-prompt-composer-types";
import { cn } from "@/lib/utils";

type PromptComposerTextareaProps = Pick<
  PromptComposerProps,
  | "onAddAttachments"
  | "onPromptChange"
  | "onPromptMentionRemove"
  | "onPromptMentionSelect"
  | "onPromptPaste"
  | "onPromptPasteRemove"
  | "onSubmit"
  | "pendingPromptPastes"
  | "prompt"
  | "promptResourceReferences"
  | "projectPath"
  | "projects"
> & {
  onSelectLocalResource: () => Promise<void>;
  textareaHeights: { content: number; viewport: number };
  textareaRef: AutoTextareaHeightRef;
};

export function PromptComposerTextarea({
  onAddAttachments,
  onPromptChange,
  onPromptMentionRemove,
  onPromptMentionSelect,
  onPromptPaste,
  onPromptPasteRemove,
  onSelectLocalResource,
  onSubmit,
  projectPath,
  projects,
  prompt,
  promptResourceReferences,
  pendingPromptPastes,
  textareaHeights,
  textareaRef,
}: PromptComposerTextareaProps) {
  const [caret, setCaret] = useState(0);
  const [activeIndex, setActiveIndex] = useState(0);
  const [mentionDismissed, setMentionDismissed] = useState(false);
  const mentionRange = useMemo(
    () => (mentionDismissed ? null : promptMentionRange(prompt, caret)),
    [caret, mentionDismissed, prompt],
  );

  function syncCaret(target: HTMLTextAreaElement) {
    setCaret(target.selectionStart ?? 0);
    setActiveIndex(0);
    setMentionDismissed(false);
  }

  function syncMovedCaret(target: HTMLTextAreaElement) {
    const nextCaret = target.selectionStart ?? 0;
    setCaret(nextCaret);
    if (nextCaret === caret) return;
    setActiveIndex(0);
    setMentionDismissed(false);
  }

  function selectMention(
    range: PromptMentionRange,
    item: PromptMentionReference,
  ) {
    const next = consumePromptMentionQuery(prompt, caret, range);
    onPromptMentionSelect(item);
    onPromptChange(next.value);
    setCaret(next.caret);
    setActiveIndex(0);
    setMentionDismissed(false);
    requestAnimationFrame(() => {
      textareaRef.current?.focus();
      textareaRef.current?.setSelectionRange(next.caret, next.caret);
    });
  }

  function selectMentionAction(
    range: PromptMentionRange,
    item: PromptMentionAction,
  ) {
    const next = consumePromptMentionQuery(prompt, caret, range);
    onPromptChange(next.value);
    setCaret(next.caret);
    setActiveIndex(0);
    setMentionDismissed(true);
    if (item.action === "select-local") {
      void selectLocalComposerResource(next.caret);
    }
  }

  function showPromptPaste(paste: PendingPromptPaste) {
    const nextPrompt = prompt ? `${prompt}\n\n${paste.text}` : paste.text;
    onPromptChange(nextPrompt);
    onPromptPasteRemove(paste.id);
    requestAnimationFrame(() => {
      textareaRef.current?.focus();
      textareaRef.current?.setSelectionRange(nextPrompt.length, nextPrompt.length);
      setCaret(nextPrompt.length);
      setActiveIndex(0);
      setMentionDismissed(true);
    });
  }

  function openPromptPaste(paste: PendingPromptPaste) {
    const target = paste.target ?? promptPasteTarget(paste.text);
    if (!target) {
      const name = paste.name
        ?? inferPromptPasteName(paste.text)
        ?? "粘贴内容.txt";
      void window.aivoDesktop.file
        .openText(name, paste.text)
        .catch((error: unknown) => {
          const detail = error instanceof Error ? error.message : String(error);
          toast.error("无法用系统默认方式打开", { description: detail });
        });
      return;
    }
    if (!window.aivoDesktop?.file) {
      toast.error("当前环境不支持系统默认打开");
      return;
    }
    const open = isExternalPromptPasteTarget(target)
      ? window.aivoDesktop.file.openExternal(target)
      : window.aivoDesktop.file.openPath(target);
    void open.catch((error: unknown) => {
      const detail = error instanceof Error ? error.message : String(error);
      toast.error("无法用系统默认方式打开", { description: detail });
    });
  }

  async function selectLocalComposerResource(nextCaret: number) {
    try {
      await onSelectLocalResource();
    } finally {
      requestAnimationFrame(() => {
        textareaRef.current?.focus();
        textareaRef.current?.setSelectionRange(nextCaret, nextCaret);
      });
    }
  }

  return (
    <div className="min-w-0">
      {mentionRange ? (
        <PromptMentionPicker
          activeIndex={activeIndex}
          onSelect={(item) => selectMention(mentionRange, item.reference)}
          onSelectAction={(item) => selectMentionAction(mentionRange, item)}
          projectPath={projectPath}
          projects={projects}
          query={mentionRange.query}
        />
      ) : null}
      <PromptMentionTags
        onRemove={onPromptMentionRemove}
        references={promptResourceReferences}
      />
      {pendingPromptPastes.length > 0 ? (
        <AttachmentGroup className="gap-2">
          {pendingPromptPastes.map((paste) => (
            <PromptPasteCard
              key={paste.id}
              onOpen={() => openPromptPaste(paste)}
              onRemove={() => onPromptPasteRemove(paste.id)}
              onShow={() => showPromptPaste(paste)}
              paste={paste}
            />
          ))}
        </AttachmentGroup>
      ) : null}
      <ScrollArea
        className="min-h-8 [&_[data-slot=scroll-area-scrollbar]]:mr-2 [&_[data-slot=scroll-area-scrollbar]]:mt-2"
        style={textareaHeights.viewport
          ? { height: textareaHeights.viewport }
          : undefined}
      >
        <textarea
          aria-label="任务描述"
          className="block min-h-8 w-full resize-none overflow-hidden bg-transparent text-sm leading-normal text-foreground outline-none placeholder:text-muted-foreground"
          onChange={(event) => {
            onPromptChange(event.target.value);
            syncCaret(event.target);
          }}
          onClick={(event) => syncCaret(event.currentTarget)}
          onKeyDown={(event) => {
            const enterAction = event.key === "Enter"
              ? promptComposerEnterAction(
                  Boolean(mentionRange),
                  event.shiftKey,
                  event.nativeEvent.isComposing,
                )
              : "none";
            if (mentionRange) {
              if (event.key === "Escape") {
                event.preventDefault();
                setMentionDismissed(true);
                return;
              }
              if (event.key === "ArrowDown" || event.key === "ArrowUp") {
                event.preventDefault();
                const itemCount = document.querySelectorAll(
                  '[aria-label="引用资源"] [role="option"]',
                ).length;
                setActiveIndex((current) => {
                  if (!itemCount) return 0;
                  return event.key === "ArrowDown"
                    ? Math.min(current + 1, itemCount - 1)
                    : Math.max(current - 1, 0);
                });
                return;
              }
              if (enterAction === "mention") {
                const selected = document.querySelectorAll<HTMLButtonElement>(
                  '[aria-label="引用资源"] [role="option"]',
                )[activeIndex];
                if (selected) {
                  event.preventDefault();
                  selected.click();
                  return;
                }
              }
            }
            if (
              event.key === "Backspace" &&
              !prompt &&
              promptResourceReferences.length > 0
            ) {
              event.preventDefault();
              onPromptMentionRemove(promptResourceReferences.at(-1)!);
              return;
            }
            if (enterAction === "submit") {
              event.preventDefault();
              onSubmit();
            }
          }}
          onPaste={(event) => {
            const files = event.clipboardData.files;
            const paths = Array.from(files).map((file) =>
              window.aivoDesktop?.file?.pathForFile?.(file) ?? "",
            ).filter(Boolean);
            if (paths.length > 0) {
              event.preventDefault();
              paths.forEach((path, index) =>
                onPromptPaste(index === 0 ? event.clipboardData?.getData("text/plain") ?? "" : "", path),
              );
              return;
            }
            const result = onPromptPaste(
              event.clipboardData?.getData("text/plain") ?? "",
            );
            if (result) {
              event.preventDefault();
              return;
            }
            if (!files.length) return;
            event.preventDefault();
            onAddAttachments(files);
          }}
          onSelect={(event) => syncMovedCaret(event.currentTarget)}
          placeholder={promptResourceReferences.length ? "" : "随心输入"}
          ref={textareaRef}
          rows={1}
          style={textareaHeights.content
            ? { height: textareaHeights.content }
            : undefined}
          value={prompt}
        />
      </ScrollArea>
    </div>
  );
}

function PromptPasteCard({
  onOpen,
  onRemove,
  onShow,
  paste,
}: {
  onOpen: () => void;
  onRemove: () => void;
  onShow: () => void;
  paste: PendingPromptPaste;
}) {
  const title = promptPasteTitle(paste);
  const openableTarget = paste.target ?? promptPasteTarget(paste.text);

  return (
    <Attachment
      className={cn(
        "h-14 w-56 flex-nowrap gap-3 rounded-xl bg-background/70 p-2",
        openableTarget && "cursor-pointer",
      )}
      onClick={onOpen}
    >
      <AttachmentMedia
        className="size-10 rounded-lg bg-muted/80 text-muted-foreground"
        variant="icon"
      >
        <HugeiconsIcon icon={File02Icon} strokeWidth={2} />
      </AttachmentMedia>
      <AttachmentContent className="min-w-0 overflow-hidden py-0.5">
        <AttachmentTitle
          className="w-full text-sm leading-5"
          title={openableTarget}
        >
          {title}
        </AttachmentTitle>
        <button
          className="mt-0.5 flex min-w-0 items-center gap-1 text-xs font-medium text-muted-foreground underline underline-offset-2 transition-colors hover:text-foreground"
          onClick={(event) => {
            event.stopPropagation();
            onShow();
          }}
          type="button"
        >
          <span className="truncate">在文本框中显示</span>
          <HugeiconsIcon className="size-3 shrink-0" icon={ArrowRight01Icon} strokeWidth={2} />
        </button>
      </AttachmentContent>
      <AttachmentActions className="relative -mr-0.5 -mt-0.5 self-start">
        <AttachmentAction
          aria-label={`移除${title}`}
          className="size-5 rounded-full bg-background/95 p-0 shadow-sm ring-1 ring-border/60 hover:bg-muted"
          onClick={(event) => {
            event.stopPropagation();
            onRemove();
          }}
          type="button"
        >
          <HugeiconsIcon className="size-3" icon={Cancel01Icon} strokeWidth={2} />
        </AttachmentAction>
      </AttachmentActions>
    </Attachment>
  );
}
