import { useCallback, useMemo, useState } from "react";
import {
  SidebarContent,
  SidebarHeader,
} from "@/components/ui/sidebar";
import {
  ConversationSidebarFooter,
  ConversationSidebarPrimaryNav,
  ConversationSidebarScrollableSections,
  PinnedConversationSection,
} from "@/features/projects/project-conversation-sidebar-sections";
import {
  getActiveProjectPathKey,
  partitionConversationsByPinnedState,
  type ConversationSidebarProps,
} from "@/features/projects/project-conversation-sidebar-model";

export function ConversationSidebar({
  activeConversationId,
  activeProjectPage,
  conversations,
  onNewConversation,
  onNewProjectConversation,
  onArchiveConversation,
  onHideProject,
  onSelectConversation,
  onTogglePinnedConversation,
  pendingPermissionCountsBySessionId,
  pinnedConversationIds,
  projectGroups,
  runningConversationIds,
  selectedProjectPath,
  topBar,
}: ConversationSidebarProps) {
  const [conversationsCollapsed, setConversationsCollapsed] = useState(false);
  const [projectsCollapsed, setProjectsCollapsed] = useState(false);
  const [collapsedProjectPaths, setCollapsedProjectPaths] = useState<string[]>(
    [],
  );
  const pinnedConversationIdSet = useMemo(
    () => new Set(pinnedConversationIds),
    [pinnedConversationIds],
  );
  const runningConversationIdSet = useMemo(
    () => new Set(runningConversationIds),
    [runningConversationIds],
  );
  const { pinnedConversations, regularConversations } = useMemo(
    () => partitionConversationsByPinnedState(
      conversations,
      pinnedConversationIdSet,
    ),
    [conversations, pinnedConversationIdSet],
  );
  const collapsedProjectPathSet = useMemo(
    () => new Set(collapsedProjectPaths),
    [collapsedProjectPaths],
  );
  const activeProjectPathKey = useMemo(
    () =>
      getActiveProjectPathKey({
        activeConversationId,
        projectGroups,
        selectedProjectPath,
      }),
    [activeConversationId, projectGroups, selectedProjectPath],
  );

  const toggleConversationsCollapsed = useCallback(() => {
    setConversationsCollapsed((collapsed) => !collapsed);
  }, []);
  const toggleProjectsCollapsed = useCallback(() => {
    setProjectsCollapsed((collapsed) => !collapsed);
  }, []);
  const toggleProjectCollapsed = useCallback((projectPath: string) => {
    setCollapsedProjectPaths((currentPaths) =>
      currentPaths.includes(projectPath)
        ? currentPaths.filter((path) => path !== projectPath)
        : [projectPath, ...currentPaths],
    );
  }, []);

  return (
    <>
      <SidebarHeader className="h-9 shrink-0 p-0 text-base">
        {topBar}
      </SidebarHeader>

      <SidebarContent className="h-0 overflow-hidden px-0 pb-0">
        <ConversationSidebarPrimaryNav
          activeProjectPage={activeProjectPage}
          onNewConversation={onNewConversation}
        />
        <PinnedConversationSection
          activeConversationId={activeConversationId}
          conversations={pinnedConversations}
          onArchiveConversation={onArchiveConversation}
          onSelectConversation={onSelectConversation}
          onTogglePinnedConversation={onTogglePinnedConversation}
          pendingPermissionCountsBySessionId={pendingPermissionCountsBySessionId}
          runningConversationIdSet={runningConversationIdSet}
        />
        <ConversationSidebarScrollableSections
          activeConversationId={activeConversationId}
          activeProjectPathKey={activeProjectPathKey}
          collapsedProjectPathSet={collapsedProjectPathSet}
          conversationsCollapsed={conversationsCollapsed}
          onArchiveConversation={onArchiveConversation}
          onHideProject={onHideProject}
          onNewProjectConversation={onNewProjectConversation}
          onSelectConversation={onSelectConversation}
          onToggleConversationsCollapsed={toggleConversationsCollapsed}
          onTogglePinnedConversation={onTogglePinnedConversation}
          onToggleProjectCollapsed={toggleProjectCollapsed}
          onToggleProjectsCollapsed={toggleProjectsCollapsed}
          pendingPermissionCountsBySessionId={pendingPermissionCountsBySessionId}
          pinnedConversationIdSet={pinnedConversationIdSet}
          projectGroups={projectGroups}
          projectsCollapsed={projectsCollapsed}
          regularConversations={regularConversations}
          runningConversationIdSet={runningConversationIdSet}
        />
      </SidebarContent>

      <ConversationSidebarFooter />
    </>
  );
}
