import { ProjectWorkspaceChatContent } from "@/features/projects/project-workspace-chat-content";
import type { ProjectWorkspaceMainContentProps } from "@/features/projects/project-workspace-main-content-model";
import { PluginMcpSettingsContent } from "@/features/projects/plugin-mcp-settings-dialog";

export function ProjectWorkspaceMainContent(
  props: ProjectWorkspaceMainContentProps,
) {
  if (props.activeProjectPage === "plugins") {
    return (
      <PluginMcpSettingsContent
        className="bg-background"
        sessionId={props.activeSessionId}
        workspaceRoot={props.workspaceRoot}
      />
    );
  }

  return <ProjectWorkspaceChatContent {...props} />;
}
