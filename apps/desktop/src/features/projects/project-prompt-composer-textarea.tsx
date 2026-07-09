import { ScrollArea } from "@/components/ui/scroll-area";
import type {
  AutoTextareaHeightRef,
  PromptComposerProps,
} from "@/features/projects/project-prompt-composer-types";

type PromptComposerTextareaProps = Pick<
  PromptComposerProps,
  "onAddAttachments" | "onPromptChange" | "onSubmit" | "prompt"
> & {
  textareaHeights: {
    content: number;
    viewport: number;
  };
  textareaRef: AutoTextareaHeightRef;
};

export function PromptComposerTextarea({
  onAddAttachments,
  onPromptChange,
  onSubmit,
  prompt,
  textareaHeights,
  textareaRef,
}: PromptComposerTextareaProps) {
  return (
    <ScrollArea
      className=" [&_[data-slot=scroll-area-scrollbar]]:mr-2 [&_[data-slot=scroll-area-scrollbar]]:mt-2"
      style={
        textareaHeights.viewport
          ? { height: textareaHeights.viewport }
          : undefined
      }
    >
      <textarea
        aria-label="任务描述"
        className="block w-full resize-none overflow-hidden bg-transparent text-sm  leading-normal text-foreground outline-none placeholder:text-muted-foreground"
        onChange={(event) => onPromptChange(event.target.value)}
        onPaste={(event) => {
          const files = event.clipboardData.files;
          if (!files.length) return;
          event.preventDefault();
          onAddAttachments(files);
        }}
        onKeyDown={(event) => {
          if (
            event.key === "Enter" &&
            !event.shiftKey &&
            !event.nativeEvent.isComposing
          ) {
            event.preventDefault();
            onSubmit();
          }
        }}
        placeholder="随心输入"
        ref={textareaRef}
        rows={2}
        style={
          textareaHeights.content
            ? { height: textareaHeights.content }
            : undefined
        }
        value={prompt}
      />
    </ScrollArea>
  );
}
