import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
} from "@/components/ui/sidebar";
import {
  ConversationSidebarItem,
  SidebarSectionHeader,
} from "@/features/projects/project-conversation-sidebar-items";
import { collapsedSidebarSectionClassName } from "@/features/projects/project-conversation-sidebar-section-layout";
import type { domain } from "@/types/codex-domain";

export function RegularConversationSidebarSection({
  activeConversationId,
  conversationsCollapsed,
  regularConversations,
  runningConversationIdSet,
  onArchiveConversation,
  onSelectConversation,
  onToggleConversationsCollapsed,
  onTogglePinnedConversation,
}: {
  activeConversationId: string;
  conversationsCollapsed: boolean;
  regularConversations: domain.Session[];
  runningConversationIdSet: ReadonlySet<string>;
  onArchiveConversation: (sessionId: string) => void;
  onSelectConversation: (session: domain.Session) => void;
  onToggleConversationsCollapsed: () => void;
  onTogglePinnedConversation: (sessionId: string) => void;
}) {
  return (
    <SidebarGroup className="mt-3 shrink-0 p-0">
      <SidebarSectionHeader
        collapsed={conversationsCollapsed}
        label="对话"
        moreLabel="更多对话操作"
        onToggle={onToggleConversationsCollapsed}
      />
      <SidebarGroupContent
        aria-hidden={conversationsCollapsed}
        className={collapsedSidebarSectionClassName(conversationsCollapsed)}
      >
        <div className="min-h-0 overflow-hidden">
          <SidebarMenu className="min-w-0 gap-0.5 px-3">
            {regularConversations.map((conversation) => (
              <ConversationSidebarItem
                conversation={conversation}
                isActive={conversation.id === activeConversationId}
                isPinned={false}
                isRunning={runningConversationIdSet.has(conversation.id)}
                key={conversation.id}
                onArchiveConversation={onArchiveConversation}
                onSelectConversation={onSelectConversation}
                onTogglePinnedConversation={onTogglePinnedConversation}
              />
            ))}
          </SidebarMenu>
        </div>
      </SidebarGroupContent>
    </SidebarGroup>
  );
}
