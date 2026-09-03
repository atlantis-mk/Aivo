import { Archive, History, LoaderCircle } from "lucide-react";

import { AnimatedTitle } from "@/components/animated-title";
import { Button } from "@/components/ui/button";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import {
  Item,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemTitle,
} from "@/components/ui/item";
import { ScrollArea } from "@/components/ui/scroll-area";
import { relativeConversationTime } from "@/features/projects/project-conversation-sidebar-model";
import { sessionProjectName } from "@/features/projects/project-sidebar-model";
import { parseTime } from "@/features/projects/project-time-model";
import type { domain } from "../../../bridge/go/models";

export function ProjectConversationHistoryList({
  activeConversationId,
  conversations,
  onArchiveConversation,
  onSelectConversation,
  pendingPermissionCountsBySessionId,
  runningConversationIds,
}: {
  activeConversationId: string;
  conversations: domain.Session[];
  onArchiveConversation: (sessionId: string) => void;
  onSelectConversation: (session: domain.Session) => void;
  pendingPermissionCountsBySessionId: Record<string, number>;
  runningConversationIds: string[];
}) {
  const runningConversationIdSet = new Set(runningConversationIds);
  const orderedConversations = Array.from(
    new Map(conversations.map((conversation) => [conversation.id, conversation])).values(),
  ).sort(
    (left, right) =>
      parseTime(right.timeUpdated || right.timeCreated).getTime() -
      parseTime(left.timeUpdated || left.timeCreated).getTime(),
  );

  if (!orderedConversations.length) {
    return (
      <Empty className="min-h-40 rounded-none px-6 py-10">
        <EmptyMedia variant="icon">
          <History />
        </EmptyMedia>
        <EmptyHeader>
          <EmptyTitle>暂无历史记录</EmptyTitle>
          <EmptyDescription>
            开始新对话后，对话会显示在这里。
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    );
  }

  return (
    <ScrollArea className="max-h-[min(72vh,560px)] w-full min-w-0 max-w-full overflow-hidden [&>[data-slot=scroll-area-viewport]]:h-auto [&>[data-slot=scroll-area-viewport]]:max-h-[min(72vh,560px)] [&>[data-slot=scroll-area-viewport]]:overflow-x-hidden [&>[data-slot=scroll-area-viewport]>div]:!block [&>[data-slot=scroll-area-viewport]>div]:!w-full [&>[data-slot=scroll-area-viewport]>div]:!min-w-0">
      <ItemGroup className="min-w-0 max-w-full gap-1 p-2">
        {orderedConversations.map((conversation) => {
          const projectName = sessionProjectName(conversation);
          const pendingPermissionCount =
            pendingPermissionCountsBySessionId[conversation.id] ?? 0;
          const isRunning =
            runningConversationIdSet.has(conversation.id) ||
            conversation.status === "inProgress";
          const status = pendingPermissionCount
            ? pendingPermissionCount > 1
              ? `待批准 ${pendingPermissionCount}`
              : "待批准"
            : isRunning
              ? "运行中"
              : relativeConversationTime(
                  conversation.timeUpdated || conversation.timeCreated,
                );

          return (
            <Item
              aria-current={
                conversation.id === activeConversationId ? "page" : undefined
              }
              className="group/history-item relative min-w-0 max-w-full flex-nowrap overflow-hidden p-0 aria-[current=page]:bg-muted"
              key={conversation.id}
              role="listitem"
              size="sm"
              variant="outline"
            >
              <button
                className="flex min-w-0 flex-1 rounded-md px-3 py-2.5 text-left outline-none focus-visible:ring-2 focus-visible:ring-ring/30"
                onClick={() => onSelectConversation(conversation)}
                type="button"
              >
                <ItemContent className="min-w-0 max-w-full">
                  <ItemTitle className="w-full min-w-0 max-w-full">
                    <AnimatedTitle
                      className="min-w-0"
                      value={conversation.title}
                    />
                  </ItemTitle>
                  <ItemDescription className="flex w-full items-center gap-3">
                    {projectName ? (
                      <span className="truncate">{projectName}</span>
                    ) : null}
                    <span className="ml-auto shrink-0 transition-opacity group-hover/history-item:opacity-0">
                      {isRunning ? (
                        <span className="flex items-center gap-1">
                          <LoaderCircle className="size-3 animate-spin" />
                          {status}
                        </span>
                      ) : (
                        status
                      )}
                    </span>
                  </ItemDescription>
                </ItemContent>
              </button>
              {!pendingPermissionCount ? (
                <Button
                  aria-label="归档对话"
                  className="pointer-events-none absolute right-2 bottom-2 opacity-0 transition-opacity group-hover/history-item:pointer-events-auto group-hover/history-item:opacity-100 focus-visible:pointer-events-auto focus-visible:opacity-100"
                  onClick={() => onArchiveConversation(conversation.id)}
                  size="icon-xs"
                  title="归档对话"
                  type="button"
                  variant="ghost"
                >
                  <Archive />
                </Button>
              ) : null}
            </Item>
          );
        })}
      </ItemGroup>
    </ScrollArea>
  );
}
