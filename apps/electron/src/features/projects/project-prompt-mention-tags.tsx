import { Cancel01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";

import { PromptMentionIcon } from "@/features/projects/project-prompt-mention-icon";
import type { PromptMentionReference } from "@/features/projects/project-prompt-mention-model";

export function PromptMentionTags({
  onRemove,
  references,
}: {
  onRemove: (reference: PromptMentionReference) => void;
  references: PromptMentionReference[];
}) {
  if (!references.length) return null;

  return (
    <span
      aria-label="已选择的引用"
      className="inline-flex shrink-0 items-center gap-x-3"
      contentEditable={false}
    >
      {references.map((reference) => (
        <button
          aria-label={`移除引用 ${reference.token}`}
          className="group/reference inline-flex min-w-0 max-w-full items-center gap-1 text-sm font-medium text-primary outline-none transition-colors hover:text-primary/75 focus-visible:rounded focus-visible:ring-2 focus-visible:ring-ring/50"
          contentEditable={false}
          key={`${reference.kind}:${reference.id}`}
          onClick={() => onRemove(reference)}
          onMouseDown={(event) => event.preventDefault()}
          title={`移除 ${reference.token}`}
          type="button"
        >
          <PromptMentionIcon className="text-primary" kind={reference.kind} />
          <span className="min-w-0 truncate">{reference.token}</span>
          <HugeiconsIcon
            aria-hidden
            className="size-3 opacity-0 transition-opacity group-hover/reference:opacity-100 group-focus-visible/reference:opacity-100"
            data-icon="inline-end"
            icon={Cancel01Icon}
            strokeWidth={2}
          />
        </button>
      ))}
    </span>
  );
}
