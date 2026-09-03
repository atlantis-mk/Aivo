import { useMemo, useState } from "react";

import { ScrollArea } from "@/components/ui/scroll-area";
import {
  consumePromptMentionQuery,
  promptComposerEnterAction,
  promptMentionRange,
  type PromptMentionAction,
  type PromptMentionRange,
  type PromptMentionReference,
} from "@/features/projects/project-prompt-mention-model";
import { PromptMentionPicker } from "@/features/projects/project-prompt-mention-picker";
import { PromptMentionTags } from "@/features/projects/project-prompt-mention-tags";
import type {
  AutoTextareaHeightRef,
  PromptComposerProps,
} from "@/features/projects/project-prompt-composer-types";

type PromptComposerTextareaProps = Pick<
  PromptComposerProps,
  | "onAddAttachments"
  | "onPromptChange"
  | "onPromptMentionRemove"
  | "onPromptMentionSelect"
  | "onSubmit"
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
  onSelectLocalResource,
  onSubmit,
  projectPath,
  projects,
  prompt,
  promptResourceReferences,
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
