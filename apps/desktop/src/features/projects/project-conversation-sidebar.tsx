import {
  SidebarContent,
  SidebarHeader,
} from "@/components/ui/sidebar";
import { ProjectConversationHistoryList } from "@/features/projects/project-conversation-history-list";
import type { ConversationSidebarProps } from "@/features/projects/project-conversation-sidebar-model";

export function ConversationSidebar({
  activeConversationId,
  conversations,
  onArchiveConversation,
  onSelectConversation,
  pendingPermissionCountsBySessionId,
  runningConversationIds,
  topBar,
}: ConversationSidebarProps) {
  return (
    <>
      {topBar ? (
        <SidebarHeader className="h-9 shrink-0 p-0 text-base">
          {topBar}
        </SidebarHeader>
      ) : null}

      <SidebarContent className="min-h-0 flex-none overflow-hidden p-0">
        <ProjectConversationHistoryList
          activeConversationId={activeConversationId}
          conversations={conversations}
          onArchiveConversation={onArchiveConversation}
          onSelectConversation={onSelectConversation}
          pendingPermissionCountsBySessionId={pendingPermissionCountsBySessionId}
          runningConversationIds={runningConversationIds}
        />
      </SidebarContent>
    </>
  );
}
