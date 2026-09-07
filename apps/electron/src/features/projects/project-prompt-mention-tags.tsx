import { Cancel01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";

import { Badge } from "@/components/ui/badge";
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
    <div
      aria-label="已选择的引用"
      className="flex min-w-0 flex-wrap items-center gap-1 pb-1"
    >
      {references.map((reference) => (
        <Badge asChild key={`${reference.kind}:${reference.id}`} variant="ghost">
          <button
            aria-label={`移除引用 ${reference.token}`}
            className="group/reference min-w-0 max-w-full text-primary"
            onClick={() => onRemove(reference)}
            onMouseDown={(event) => event.preventDefault()}
            title={`移除 ${reference.token}`}
            type="button"
          >
            <PromptMentionIcon kind={reference.kind} />
            <span className="min-w-0 truncate text-sm">{reference.token}</span>
            <HugeiconsIcon
              aria-hidden
              className="opacity-0 transition-opacity group-hover/reference:opacity-100 group-focus-visible/reference:opacity-100"
              data-icon="inline-end"
              icon={Cancel01Icon}
              strokeWidth={2}
            />
          </button>
        </Badge>
      ))}
    </div>
  );
}
