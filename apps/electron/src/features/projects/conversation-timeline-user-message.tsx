import { memo, useLayoutEffect, useRef, useState } from "react";
import { Folder01Icon, File02Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { ChevronDown, File, Image, Pencil, Trash2 } from "lucide-react";

import {
  Attachment,
  AttachmentContent,
  AttachmentDescription,
  AttachmentMedia,
  AttachmentTitle,
} from "@/components/ui/attachment";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import {
  type ConversationTurn,
  type ConversationUserAttachment,
  type ConversationUserPaste,
} from "@/features/projects/conversation-timeline-model";
import {
  COLLAPSED_USER_MESSAGE_HEIGHT,
  formatTimelineAttachmentMeta,
  shouldShowUserMessageDisclosure,
} from "@/features/projects/conversation-timeline-display-model";
import { CopyTextButton } from "./conversation-timeline-copy-button";
import { TimelineRowFrame } from "./conversation-timeline-frame";
import type { ConversationTimelineActions } from "./conversation-timeline-types";

export const UserMessageRow = memo(function UserMessageRow({
  actions,
  turn,
}: {
  actions: ConversationTimelineActions;
  turn: ConversationTurn;
}) {
  const [userMessageExpanded, setUserMessageExpanded] = useState(false);
  const [userMessageContentHeight, setUserMessageContentHeight] = useState<
    number | null
  >(null);
  const userMessageContentRef = useRef<HTMLDivElement>(null);
  const userMessageExpandable = shouldShowUserMessageDisclosure(
    userMessageContentHeight,
  );
  const userMessageAnimatedHeight =
    userMessageContentHeight === null
      ? undefined
      : userMessageExpanded
        ? userMessageContentHeight
        : Math.min(userMessageContentHeight, COLLAPSED_USER_MESSAGE_HEIGHT);

  useLayoutEffect(() => {
    const contentElement = userMessageContentRef.current;
    if (!contentElement) return;

    const updateContentHeight = () => {
      const nextHeight = contentElement.scrollHeight;
      setUserMessageContentHeight((current) =>
        current === nextHeight ? current : nextHeight,
      );
    };

    updateContentHeight();
    const frame = requestAnimationFrame(updateContentHeight);
    const observer = new ResizeObserver(updateContentHeight);
    observer.observe(contentElement);

    return () => {
      cancelAnimationFrame(frame);
      observer.disconnect();
    };
  }, [turn.prompt]);

  return (
    <TimelineRowFrame role="user" turnId={turn.id}>
      <div className="aivo-user-message group/user-message relative ml-auto flex flex-col items-end">
        {turn.attachments?.length ? (
          <UserMessageAttachments attachments={turn.attachments} />
        ) : null}
        {turn.pastes?.length ? <UserMessagePastes pastes={turn.pastes} /> : null}
        {turn.prompt ? (
          <div
            className={cn(
              "aivo-user-message-bubble max-w-[min(90%,42rem)] rounded bg-[#f4f4f4] px-2 py-1.5 text-sm text-foreground shadow-none dark:bg-[#2f2f2f]",
              !userMessageExpandable &&
                "flex items-center justify-center",
            )}
          >
            <div
              className={cn(
                "whitespace-pre-wrap break-words text-left",
                userMessageExpandable &&
                  "overflow-hidden transition-[height] duration-300 ease-out",
              )}
              style={
                userMessageExpandable && userMessageAnimatedHeight !== undefined
                  ? { height: `${userMessageAnimatedHeight}px` }
                  : undefined
              }
            >
              <div ref={userMessageContentRef}>{turn.prompt}</div>
            </div>
            {userMessageExpandable ? (
              <Button
                className="mt-2 h-auto gap-1 px-0 py-0 text-xs text-muted-foreground hover:bg-transparent hover:text-foreground"
                onClick={() => setUserMessageExpanded((expanded) => !expanded)}
                type="button"
                variant="ghost"
              >
                {userMessageExpanded ? "收起" : "显示更多"}
                <ChevronDown
                  className={cn(
                    "transition-transform",
                    userMessageExpanded && "rotate-180",
                  )}
                />
              </Button>
            ) : null}
          </div>
        ) : null}
        <div className="aivo-user-message-meta flex items-center gap-2">
          {turn.prompt ? <CopyTextButton ariaLabel="复制消息" text={turn.prompt} /> : null}
          {turn.userEventId && actions.onEditUserMessage ? (
            <Button
              aria-label="编辑消息"
              onClick={() => actions.onEditUserMessage?.(turn)}
              size="icon-sm"
              title="编辑消息"
              type="button"
              variant="ghost"
            >
              <Pencil />
            </Button>
          ) : null}
          {(turn.userEventId || turn.assistantEventId) &&
          actions.onDeleteTurn ? (
            <Button
              aria-label="删除本轮"
              onClick={() => actions.onDeleteTurn?.(turn)}
              size="icon-sm"
              title="删除本轮"
              type="button"
              variant="ghost"
            >
              <Trash2 />
            </Button>
          ) : null}
        </div>
      </div>
    </TimelineRowFrame>
  );
}, areConversationTurnPropsEqual);

function UserMessageAttachments({
  attachments,
}: {
  attachments: ConversationUserAttachment[];
}) {
  return (
    <div className="mb-2 flex max-w-[min(90%,42rem)] flex-wrap justify-end gap-2">
      {attachments.map((attachment) =>
        attachment.kind === "image" ? (
          <Attachment
            className="size-24 overflow-hidden bg-card p-0 shadow-lg shadow-foreground/5"
            key={attachment.id}
            orientation="vertical"
            size="sm"
          >
            {attachment.previewUrl ? (
              <img
                alt={attachment.name}
                className="absolute inset-0 size-full object-cover"
                src={attachment.previewUrl}
              />
            ) : (
              <div className="absolute inset-0 flex items-center justify-center bg-muted text-muted-foreground">
                <Image />
              </div>
            )}
          </Attachment>
        ) : (
          <Attachment
            className="h-10 w-40 flex-nowrap gap-2 rounded-lg bg-card p-1.5 shadow-lg shadow-foreground/5"
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
            <AttachmentContent className="min-w-0 overflow-hidden py-0.5">
              <AttachmentTitle className="text-[13px] font-medium leading-[18px]">
                {attachment.name}
              </AttachmentTitle>
              <AttachmentDescription className="mt-px text-[10px] font-medium uppercase tracking-wide">
                {formatTimelineAttachmentMeta(attachment)}
              </AttachmentDescription>
            </AttachmentContent>
          </Attachment>
        ),
      )}
    </div>
  );
}

function UserMessagePastes({ pastes }: { pastes: ConversationUserPaste[] }) {
  return (
    <div className="mb-2 flex max-w-[min(90%,42rem)] flex-col items-end gap-2">
      {pastes.map((paste) => (
        <Attachment
          className="h-10 max-w-[min(90vw,28rem)] flex-nowrap gap-1.5 rounded-lg bg-background px-2 py-1 shadow-lg shadow-foreground/5"
          key={paste.id}
          orientation="horizontal"
          size="xs"
          title={paste.name}
        >
          <AttachmentMedia className="size-5 bg-transparent p-0 text-muted-foreground" variant="icon">
            <File className="size-3.5" />
          </AttachmentMedia>
          <AttachmentContent className="min-w-0 py-0">
            <AttachmentTitle className="text-sm font-medium leading-5">
              {paste.preview || paste.name}
            </AttachmentTitle>
          </AttachmentContent>
        </Attachment>
      ))}
    </div>
  );
}

function areConversationTurnPropsEqual(
  previous: { actions: ConversationTimelineActions; turn: ConversationTurn },
  next: { actions: ConversationTimelineActions; turn: ConversationTurn },
) {
  return previous.turn === next.turn && previous.actions === next.actions;
}
