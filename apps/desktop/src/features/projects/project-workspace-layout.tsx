import { ProjectColumnWorkspace } from "./project-column-workspace";
import type { ProjectColumnShellProps } from "./project-workspace-layout-model";

export { ProjectTopBarIconButton } from "./project-workspace-topbar-actions";

export function ProjectColumnShell({
  main,
  mainTopBar,
}: ProjectColumnShellProps) {
  return (
    <div className="h-dvh !min-h-0 overflow-hidden bg-background text-foreground">
      <ProjectColumnWorkspace main={main} mainTopBar={mainTopBar} />
    </div>
  );
}
