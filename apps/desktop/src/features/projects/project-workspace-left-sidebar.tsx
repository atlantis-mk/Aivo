import { ConversationSidebar } from "@/features/projects/project-conversation-sidebar";
import type { ConversationSidebarProps } from "@/features/projects/project-conversation-sidebar-model";

type ProjectWorkspaceLeftSidebarProps = Omit<
  ConversationSidebarProps,
  "topBar"
>;

export function ProjectWorkspaceLeftSidebar(
  props: ProjectWorkspaceLeftSidebarProps,
) {
  return <ConversationSidebar {...props} topBar={null} />;
}
