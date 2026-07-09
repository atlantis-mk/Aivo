import { File, Image, X } from "lucide-react";

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
  formatAttachmentMeta,
} from "@/features/projects/project-composer-attachments";
import type { ComposerAttachmentListProps } from "@/features/projects/project-prompt-composer-types";
import { cn } from "@/lib/utils";

export function ComposerAttachmentList({
  attachments,
  onRemoveAttachment,
}: ComposerAttachmentListProps) {
  if (attachments.length === 0) return null;

  return (
    <AttachmentGroup>
      {attachments.map((attachment) => (
        <Attachment
          className={cn(
            "bg-background/70",
            attachment.kind === "image"
              ? "size-24 overflow-hidden p-0"
              : "w-28",
          )}
          key={attachment.id}
          orientation="vertical"
          size="sm"
        >
          {attachment.kind === "image" ? (
            attachment.previewUrl ? (
              <img
                alt=""
                className="absolute inset-0 size-full object-cover"
                src={attachment.previewUrl}
              />
            ) : (
              <div className="absolute inset-0 flex items-center justify-center bg-muted text-muted-foreground">
                <Image />
              </div>
            )
          ) : (
            <AttachmentMedia className="h-16" variant="icon">
              <File />
            </AttachmentMedia>
          )}
          {attachment.kind === "file" ? (
            <AttachmentContent>
              <AttachmentTitle>{attachment.name}</AttachmentTitle>
              <AttachmentDescription>
                {formatAttachmentMeta(attachment)}
              </AttachmentDescription>
            </AttachmentContent>
          ) : null}
          <AttachmentActions className="group-data-[orientation=vertical]/attachment:right-1 group-data-[orientation=vertical]/attachment:top-1">
            <AttachmentAction
              aria-label={`移除附件 ${attachment.name}`}
              className="rounded-full bg-background/90 shadow-sm"
              onClick={() => onRemoveAttachment(attachment.id)}
              type="button"
            >
              <X />
            </AttachmentAction>
          </AttachmentActions>
        </Attachment>
      ))}
    </AttachmentGroup>
  );
}
