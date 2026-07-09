import { memo } from "react";
import { Archive, Pin } from "lucide-react";

import { AnimatedTitle } from "@/components/animated-title";
import { Button } from "@/components/ui/button";
import {
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { Spinner } from "@/components/ui/spinner";
import {
  relativeConversationTime,
  type ConversationSidebarItemProps,
} from "@/features/projects/project-conversation-sidebar-model";
import { cn } from "@/lib/utils";

export const ConversationSidebarItem = memo(function ConversationSidebarItem({
  conversation,
  isActive,
  isPinned,
  onArchiveConversation,
  onSelectConversation,
  onTogglePinnedConversation,
  pendingPermissionCount,
  isRunning,
}: ConversationSidebarItemProps) {
  const hasPendingPermission = pendingPermissionCount > 0;

  return (
    <SidebarMenuItem
      className="group/conversation-item relative min-w-0 "
      key={conversation.id}
    >
      <SidebarMenuButton
        aria-current={isActive ? "page" : undefined}
        className="min-w-0 justify-between gap-2 rounded-md px-1.5 text-sidebar-foreground transition-colors"
        isActive={isActive}
        onClick={() => onSelectConversation(conversation)}
        type="button"
      >
        <AnimatedTitle
          className="min-w-0 flex-1 text-xs leading-5"
          value={conversation.title}
        />
        {hasPendingPermission ? (
          <span className="shrink-0 rounded-full bg-primary px-2 py-0.5 text-[11px] font-semibold leading-none text-primary-foreground shadow-sm shadow-primary/20">
            {pendingPermissionCount > 1
              ? `待批准 ${pendingPermissionCount}`
              : "待批准"}
          </span>
        ) : isRunning ? (
          <span className="shrink-0 transition-opacity group-hover/conversation-item:opacity-0 group-focus-within/conversation-item:opacity-0">
            <Spinner
              className={cn(
                "text-muted-foreground",
                isActive && "text-sidebar-accent-foreground/70",
              )}
            />
          </span>
        ) : (
          <span
            className={cn(
              "shrink-0 text-xs  text-muted-foreground transition-[opacity,color] group-hover/conversation-item:opacity-0 group-focus-within/conversation-item:opacity-0",
              isActive && "text-sidebar-accent-foreground/70",
            )}
          >
            {relativeConversationTime(
              conversation.timeUpdated || conversation.timeCreated,
            )}
          </span>
        )}
      </SidebarMenuButton>
      {!hasPendingPermission && (
        <span className="pointer-events-none absolute right-2 top-1/2 flex -translate-y-1/2 items-center gap-0.5 opacity-0 transition-opacity group-hover/conversation-item:pointer-events-auto group-hover/conversation-item:opacity-100 group-focus-within/conversation-item:pointer-events-auto group-focus-within/conversation-item:opacity-100">
          <Button
            aria-label={isPinned ? "取消置顶" : "置顶对话"}
            onClick={(event) => {
              event.preventDefault();
              event.stopPropagation();
              onTogglePinnedConversation(conversation.id);
            }}
            size="icon-xs"
            title={isPinned ? "取消置顶" : "置顶对话"}
            type="button"
            variant="ghost"
          >
            <Pin className={cn(isPinned && "fill-current")} />
          </Button>
          <Button
            aria-label="归档对话"
            onClick={(event) => {
              event.preventDefault();
              event.stopPropagation();
              onArchiveConversation(conversation.id);
            }}
            size="icon-xs"
            title="归档对话"
            type="button"
            variant="ghost"
          >
            <Archive />
          </Button>
        </span>
      )}
    </SidebarMenuItem>
  );
});
