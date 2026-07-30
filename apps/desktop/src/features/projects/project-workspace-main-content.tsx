import { ProjectWorkspaceChatContent } from "@/features/projects/project-workspace-chat-content";
import type { ProjectWorkspaceMainContentProps } from "@/features/projects/project-workspace-main-content-model";
import { ProjectEnvironmentSummaryAside } from "@/features/projects/project-workspace-chat-overlays";
import { PluginMcpSettingsContent } from "@/features/projects/plugin-mcp-settings-dialog";

export function ProjectWorkspaceMainContent(
  props: ProjectWorkspaceMainContentProps,
) {
  if (props.activeProjectPage === "plugins") {
    return (
      <div className="relative min-h-0 flex-1">
        <PluginMcpSettingsContent
          className="bg-background"
          sessionId={props.activeSessionId}
          workspaceRoot={props.workspaceRoot}
        />
        {props.shouldShowEnvironmentSummaryPanel ? (
          <ProjectEnvironmentSummaryAside
            canDockPinnedSummary={props.canDockPinnedSummary}
            onOpenTools={props.onOpenToolActivationDialog}
          />
        ) : null}
      </div>
    );
  }

  return <ProjectWorkspaceChatContent {...props} />;
}
