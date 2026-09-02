import { memo, useLayoutEffect, useRef, useState } from "react";
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
} from "@/features/projects/conversation-timeline-model";
import {
  COLLAPSED_USER_MESSAGE_HEIGHT,
  formatCompletionTime,
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
      <div className="group/user-message ml-auto flex flex-col items-end">
        {turn.attachments?.length ? (
          <UserMessageAttachments attachments={turn.attachments} />
        ) : null}
        <div
          className={cn(
            "max-w-[min(90%,42rem)] rounded-xl bg-card px-4 py-3 text-sm text-card-foreground shadow-lg shadow-foreground/5 sm:px-5",
            !userMessageExpandable &&
              "flex min-h-[52px] items-center justify-center",
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
        <div className="mb-3 mt-1.5 flex items-center gap-2">
          <span className="text-sm">{formatCompletionTime(turn.submittedAt)}</span>
          <CopyTextButton ariaLabel="复制消息" text={turn.prompt} />
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
            className="max-w-52 bg-card shadow-lg shadow-foreground/5"
            key={attachment.id}
            orientation="horizontal"
            size="default"
          >
            <AttachmentMedia variant="icon">
              <File />
            </AttachmentMedia>
            <AttachmentContent>
              <AttachmentTitle>{attachment.name}</AttachmentTitle>
              <AttachmentDescription>
                {formatTimelineAttachmentMeta(attachment)}
              </AttachmentDescription>
            </AttachmentContent>
          </Attachment>
        ),
      )}
    </div>
  );
}

function areConversationTurnPropsEqual(
  previous: { actions: ConversationTimelineActions; turn: ConversationTurn },
  next: { actions: ConversationTimelineActions; turn: ConversationTurn },
) {
  return previous.turn === next.turn && previous.actions === next.actions;
}
