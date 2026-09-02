import { ProjectWorkspaceChatContent } from "@/features/projects/project-workspace-chat-content";
import type { ProjectWorkspaceMainContentProps } from "@/features/projects/project-workspace-main-content-model";

export function ProjectWorkspaceMainContent(
  props: ProjectWorkspaceMainContentProps,
) {
  return <ProjectWorkspaceChatContent {...props} />;
}
