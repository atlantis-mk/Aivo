import {
  Folder01Icon,
  PackageOpenIcon,
  PuzzleIcon,
  ServerStack01Icon,
  ToolsIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";

import type { PromptMentionReference } from "@/features/projects/project-prompt-mention-model";
import { cn } from "@/lib/utils";

export function PromptMentionIcon({
  className,
  kind,
}: {
  className?: string;
  kind: PromptMentionReference["kind"];
}) {
  const icon = kind === "project"
    ? Folder01Icon
    : kind === "skill"
      ? PuzzleIcon
      : kind === "extension"
        ? PackageOpenIcon
        : kind === "mcp"
          ? ServerStack01Icon
          : ToolsIcon;

  return (
    <HugeiconsIcon
      aria-hidden
      className={cn("size-4 shrink-0", className)}
      data-icon="inline-start"
      icon={icon}
      strokeWidth={1.8}
    />
  );
}
