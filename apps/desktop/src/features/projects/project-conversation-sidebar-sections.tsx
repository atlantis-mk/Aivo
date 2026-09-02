import { Link } from "@tanstack/react-router";
import { Plug, Settings, Smartphone, SquarePen } from "lucide-react";

import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { ConversationSidebarItem } from "@/features/projects/project-conversation-sidebar-items";
import { ProjectConversationSidebarSection } from "@/features/projects/project-conversation-sidebar-project-section";
import { RegularConversationSidebarSection } from "@/features/projects/project-conversation-sidebar-regular-section";
import type { domain } from "../../../bridge/go/models";
import type { ProjectConversationGroup } from "@/features/projects/project-sidebar-model";

type ConversationSidebarCallbacks = {
  onArchiveConversation: (sessionId: string) => void;
  onHideProject: (projectPath: string) => void;
  onNewProjectConversation: (projectPath: string) => void;
  onSelectConversation: (session: domain.Session) => void;
  onTogglePinnedConversation: (sessionId: string) => void;
};

export function ConversationSidebarPrimaryNav({
  activeProjectPage,
  onNewConversation,
}: {
  activeProjectPage: "chat" | "extensions";
  onNewConversation: () => void;
}) {
  return (
    <SidebarGroup className="shrink-0 p-0 px-1.5 group-data-[collapsible=icon]:px-2">
      <SidebarMenu className="gap-2">
        <SidebarMenuItem>
          <SidebarMenuButton
            className="gap-2.5 px-1.5 py-2 text-sm text-sidebar-foreground group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:px-2"
            onClick={onNewConversation}
            tooltip="新对话"
            type="button"
          >
            <SquarePen />
            <span>新对话</span>
          </SidebarMenuButton>
        </SidebarMenuItem>
        <SidebarMenuItem>
          <SidebarMenuButton
            asChild
            className="gap-2.5 px-1.5 py-2 text-sm text-sidebar-foreground group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:px-2"
            isActive={activeProjectPage === "extensions"}
            tooltip="扩展"
          >
            <Link to="/extensions">
              <Plug />
              <span>扩展</span>
            </Link>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarGroup>
  );
}

export function PinnedConversationSection({
  activeConversationId,
  conversations,
  onArchiveConversation,
  onSelectConversation,
  onTogglePinnedConversation,
  pendingPermissionCountsBySessionId,
  runningConversationIdSet,
}: Pick<
  ConversationSidebarCallbacks,
  "onArchiveConversation" | "onSelectConversation" | "onTogglePinnedConversation"
> & {
  activeConversationId: string;
  conversations: domain.Session[];
  pendingPermissionCountsBySessionId: Record<string, number>;
  runningConversationIdSet: ReadonlySet<string>;
}) {
  if (conversations.length === 0) return null;

  return (
    <SidebarGroup className="mt-3 shrink-0 p-0">
      <SidebarGroupLabel className="mx-1.5 flex h-6 items-center px-3 text-sm font-semibold text-muted-foreground group-data-[collapsible=icon]:mx-2">
        <span className="min-w-0 truncate">置顶</span>
      </SidebarGroupLabel>
      <SidebarGroupContent className="group-data-[collapsible=icon]:hidden">
        <SidebarMenu className="min-w-0 gap-0.5 px-3">
          {conversations.map((conversation) => (
            <ConversationSidebarItem
              conversation={conversation}
              isActive={conversation.id === activeConversationId}
              isPinned
              isRunning={runningConversationIdSet.has(conversation.id)}
              key={conversation.id}
              onArchiveConversation={onArchiveConversation}
              onSelectConversation={onSelectConversation}
              onTogglePinnedConversation={onTogglePinnedConversation}
              pendingPermissionCount={
                pendingPermissionCountsBySessionId[conversation.id] ?? 0
              }
            />
          ))}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  );
}

export function ConversationSidebarScrollableSections({
  activeConversationId,
  activeProjectPathKey,
  collapsedProjectPathSet,
  conversationsCollapsed,
  pendingPermissionCountsBySessionId,
  pinnedConversationIdSet,
  projectGroups,
  projectsCollapsed,
  regularConversations,
  runningConversationIdSet,
  onArchiveConversation,
  onHideProject,
  onNewProjectConversation,
  onSelectConversation,
  onToggleConversationsCollapsed,
  onTogglePinnedConversation,
  onToggleProjectCollapsed,
  onToggleProjectsCollapsed,
}: ConversationSidebarCallbacks & {
  activeConversationId: string;
  activeProjectPathKey: string;
  collapsedProjectPathSet: ReadonlySet<string>;
  conversationsCollapsed: boolean;
  pendingPermissionCountsBySessionId: Record<string, number>;
  pinnedConversationIdSet: ReadonlySet<string>;
  projectGroups: ProjectConversationGroup[];
  projectsCollapsed: boolean;
  regularConversations: domain.Session[];
  runningConversationIdSet: ReadonlySet<string>;
  onToggleConversationsCollapsed: () => void;
  onToggleProjectCollapsed: (projectPath: string) => void;
  onToggleProjectsCollapsed: () => void;
}) {
  return (
    <ScrollArea className="min-h-0 flex-1 [&_[data-slot=scroll-area-viewport]]:overflow-x-hidden [&_[data-slot=scroll-area-viewport]>div]:!block [&_[data-slot=scroll-area-viewport]>div]:!min-w-0">
      <ProjectConversationSidebarSection
        activeConversationId={activeConversationId}
        activeProjectPathKey={activeProjectPathKey}
        collapsedProjectPathSet={collapsedProjectPathSet}
        onArchiveConversation={onArchiveConversation}
        onHideProject={onHideProject}
        onNewProjectConversation={onNewProjectConversation}
        onSelectConversation={onSelectConversation}
        onTogglePinnedConversation={onTogglePinnedConversation}
        onToggleProjectCollapsed={onToggleProjectCollapsed}
        onToggleProjectsCollapsed={onToggleProjectsCollapsed}
        pendingPermissionCountsBySessionId={pendingPermissionCountsBySessionId}
        pinnedConversationIdSet={pinnedConversationIdSet}
        projectGroups={projectGroups}
        projectsCollapsed={projectsCollapsed}
        runningConversationIdSet={runningConversationIdSet}
      />
      <RegularConversationSidebarSection
        activeConversationId={activeConversationId}
        conversationsCollapsed={conversationsCollapsed}
        onArchiveConversation={onArchiveConversation}
        onSelectConversation={onSelectConversation}
        onToggleConversationsCollapsed={onToggleConversationsCollapsed}
        onTogglePinnedConversation={onTogglePinnedConversation}
        pendingPermissionCountsBySessionId={pendingPermissionCountsBySessionId}
        regularConversations={regularConversations}
        runningConversationIdSet={runningConversationIdSet}
      />
    </ScrollArea>
  );
}

export function ConversationSidebarFooter() {
  return (
    <SidebarFooter className="flex-row items-center justify-between px-1.5 pb-5 pt-3 group-data-[collapsible=icon]:px-2">
      <SidebarMenu className="min-w-0 flex-1">
        <SidebarMenuItem>
          <Button asChild size="lg" variant="ghost">
            <Link to="/settings">
              <Settings />
              <span>设置</span>
            </Link>
          </Button>
        </SidebarMenuItem>
      </SidebarMenu>
      <Button
        aria-label="移动设备"
        className="text-muted-foreground group-data-[collapsible=icon]:hidden"
        size="icon"
        type="button"
        variant="ghost"
      >
        <Smartphone />
      </Button>
    </SidebarFooter>
  );
}
