import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
} from "@/components/ui/sidebar";
import {
  ProjectSidebarItem,
  SidebarSectionHeader,
} from "@/features/projects/project-conversation-sidebar-items";
import { collapsedSidebarSectionClassName } from "@/features/projects/project-conversation-sidebar-section-layout";
import { normalizeProjectPathKey } from "@/features/projects/project-sidebar-model";
import type { domain } from "../../../bridge/go/models";
import type { ProjectConversationGroup } from "@/features/projects/project-sidebar-model";

export function ProjectConversationSidebarSection({
  activeConversationId,
  activeProjectPathKey,
  collapsedProjectPathSet,
  pendingPermissionCountsBySessionId,
  pinnedConversationIdSet,
  projectGroups,
  projectsCollapsed,
  runningConversationIdSet,
  onArchiveConversation,
  onHideProject,
  onNewProjectConversation,
  onSelectConversation,
  onTogglePinnedConversation,
  onToggleProjectCollapsed,
  onToggleProjectsCollapsed,
}: {
  activeConversationId: string;
  activeProjectPathKey: string;
  collapsedProjectPathSet: ReadonlySet<string>;
  pendingPermissionCountsBySessionId: Record<string, number>;
  pinnedConversationIdSet: ReadonlySet<string>;
  projectGroups: ProjectConversationGroup[];
  projectsCollapsed: boolean;
  runningConversationIdSet: ReadonlySet<string>;
  onArchiveConversation: (sessionId: string) => void;
  onHideProject: (projectPath: string) => void;
  onNewProjectConversation: (projectPath: string) => void;
  onSelectConversation: (session: domain.Session) => void;
  onTogglePinnedConversation: (sessionId: string) => void;
  onToggleProjectCollapsed: (projectPath: string) => void;
  onToggleProjectsCollapsed: () => void;
}) {
  if (projectGroups.length === 0) return null;

  return (
    <SidebarGroup className="mt-3 shrink-0 p-0">
      <SidebarSectionHeader
        collapsed={projectsCollapsed}
        label="项目"
        moreLabel="更多项目操作"
        onToggle={onToggleProjectsCollapsed}
      />
      <SidebarGroupContent
        aria-hidden={projectsCollapsed}
        className={collapsedSidebarSectionClassName(projectsCollapsed)}
      >
        <div className="min-h-0 overflow-hidden">
          <SidebarMenu className="min-w-0 gap-1 px-1.5">
            {projectGroups.map((group) => {
              const projectCollapsed = collapsedProjectPathSet.has(
                group.projectPath,
              );
              const projectActive =
                activeProjectPathKey ===
                normalizeProjectPathKey(group.projectPath);
              return (
                <ProjectSidebarItem
                  activeConversationId={activeConversationId}
                  collapsed={projectCollapsed}
                  group={group}
                  isActive={projectActive}
                  key={group.projectPath}
                  onArchiveConversation={onArchiveConversation}
                  onHideProject={onHideProject}
                  onNewProjectConversation={onNewProjectConversation}
                  onSelectConversation={onSelectConversation}
                  onTogglePinnedConversation={onTogglePinnedConversation}
                  onToggleProjectCollapsed={onToggleProjectCollapsed}
                  pendingPermissionCountsBySessionId={
                    pendingPermissionCountsBySessionId
                  }
                  pinnedConversationIdSet={pinnedConversationIdSet}
                  runningConversationIdSet={runningConversationIdSet}
                />
              );
            })}
          </SidebarMenu>
        </div>
      </SidebarGroupContent>
    </SidebarGroup>
  );
}
