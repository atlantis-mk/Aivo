import {
  Cancel01Icon,
  File02Icon,
  Folder01Icon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { Image } from "lucide-react";

import {
  Attachment,
  AttachmentAction,
  AttachmentActions,
  AttachmentContent,
  AttachmentDescription,
  AttachmentGroup,
  AttachmentMedia,
  AttachmentTitle,
} from "@/components/ui/attachment";
import {
  attachmentFileTypeLabel,
} from "@/features/projects/project-composer-attachments";
import type { ComposerAttachmentListProps } from "@/features/projects/project-prompt-composer-types";
import { cn } from "@/lib/utils";

export function ComposerAttachmentList({
  attachments,
  onRemoveAttachment,
}: ComposerAttachmentListProps) {
  if (attachments.length === 0) return null;

  return (
    <AttachmentGroup className="gap-2 py-0 pb-2">
      {attachments.map((attachment) =>
        attachment.kind === "image" ? (
          <Attachment
            className="size-24 overflow-hidden bg-background/70 p-0"
            key={attachment.id}
            orientation="vertical"
            size="sm"
          >
            {attachment.previewUrl ? (
              <img
                alt=""
                className="absolute inset-0 size-full object-cover"
                src={attachment.previewUrl}
              />
            ) : (
              <div className="absolute inset-0 flex items-center justify-center bg-muted text-muted-foreground">
                <Image />
              </div>
            )}
            <AttachmentRemoveAction
              name={attachment.name}
              onClick={() => onRemoveAttachment(attachment.id)}
              overlay
            />
          </Attachment>
        ) : (
          <Attachment
            className="h-10 w-40 max-w-[calc(100vw-3.5rem)] flex-nowrap gap-2 rounded-lg bg-background p-1.5"
            key={attachment.id}
            orientation="horizontal"
            size="sm"
          >
            <AttachmentMedia
              className="size-7 rounded-lg bg-muted/60 text-foreground"
              variant="icon"
            >
              <HugeiconsIcon
                className="size-4"
                icon={attachment.kind === "directory" ? Folder01Icon : File02Icon}
                strokeWidth={2}
              />
            </AttachmentMedia>
            <AttachmentContent className="min-w-0 overflow-hidden py-0.5 pr-4">
              <AttachmentTitle className="w-full text-[13px] font-medium leading-[18px]">
                {attachment.name}
              </AttachmentTitle>
              <AttachmentDescription className="mt-px text-[10px] font-medium uppercase tracking-wide">
                {attachmentFileTypeLabel(attachment)}
              </AttachmentDescription>
            </AttachmentContent>
            <AttachmentRemoveAction
              name={attachment.name}
              onClick={() => onRemoveAttachment(attachment.id)}
              overlay
            />
          </Attachment>
        ),
      )}
    </AttachmentGroup>
  );
}

function AttachmentRemoveAction({
  name,
  onClick,
  overlay = false,
}: {
  name: string;
  onClick: () => void;
  overlay?: boolean;
}) {
  return (
    <AttachmentActions
      className={cn(
        overlay
          ? "absolute right-0.5 top-0.5"
          : "relative -mr-0.5 -mt-0.5 self-start",
      )}
    >
      <AttachmentAction
        aria-label={`移除附件 ${name}`}
        className="size-5 rounded-full bg-background/95 p-0 shadow-sm ring-1 ring-border/60 hover:bg-muted"
        onClick={onClick}
        type="button"
      >
        <HugeiconsIcon className="size-3" icon={Cancel01Icon} strokeWidth={2} />
      </AttachmentAction>
    </AttachmentActions>
  );
}
