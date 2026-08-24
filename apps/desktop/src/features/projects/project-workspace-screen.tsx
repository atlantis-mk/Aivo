import { useProjectWorkspaceScreenController } from "@/features/projects/project-workspace-screen-controller";
import { ProjectWorkspaceScreenView } from "@/features/projects/project-workspace-screen-view";

export function ProjectWorkspaceScreen() {
  const workspaceViewProps = useProjectWorkspaceScreenController();

  return <ProjectWorkspaceScreenView {...workspaceViewProps} />;
}
