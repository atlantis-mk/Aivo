import { AnimatedTitle } from "@/components/animated-title";
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
  onSelectConversation,
  pendingPermissionCountsBySessionId,
  runningConversationIds,
}: {
  activeConversationId: string;
  conversations: domain.Session[];
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

  return (
    <ScrollArea className="max-h-[min(72vh,560px)] [&>[data-slot=scroll-area-viewport]]:h-auto [&>[data-slot=scroll-area-viewport]]:max-h-[min(72vh,560px)]">
      <ItemGroup className="gap-1 p-2">
        {orderedConversations.map((conversation) => {
          const projectName = sessionProjectName(conversation);
          const pendingPermissionCount =
            pendingPermissionCountsBySessionId[conversation.id] ?? 0;
          const status = pendingPermissionCount
            ? pendingPermissionCount > 1
              ? `待批准 ${pendingPermissionCount}`
              : "待批准"
            : runningConversationIdSet.has(conversation.id)
              ? "运行中"
              : relativeConversationTime(
                  conversation.timeUpdated || conversation.timeCreated,
                );

          return (
            <Item
              asChild
              className="aria-[current=page]:bg-muted"
              key={conversation.id}
              size="sm"
              variant="outline"
            >
              <button
                aria-current={
                  conversation.id === activeConversationId ? "page" : undefined
                }
                onClick={() => onSelectConversation(conversation)}
                role="listitem"
                type="button"
              >
                <ItemContent className="min-w-0">
                  <ItemTitle className="min-w-0">
                    <AnimatedTitle
                      className="min-w-0"
                      value={conversation.title}
                    />
                  </ItemTitle>
                  <ItemDescription className="flex w-full items-center gap-3">
                    {projectName ? (
                      <span className="truncate">{projectName}</span>
                    ) : null}
                    <span className="ml-auto shrink-0">{status}</span>
                  </ItemDescription>
                </ItemContent>
              </button>
            </Item>
          );
        })}
      </ItemGroup>
    </ScrollArea>
  );
}
