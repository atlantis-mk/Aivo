import type { ReactNode } from "react";

export function ProjectColumnWorkspace({
  main,
  mainTopBar,
}: {
  main: ReactNode;
  mainTopBar?: ReactNode;
}) {
  return (
    <main
      className="flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden"
      data-project-workspace-content
    >
      {mainTopBar ? (
        <div className="z-50 h-9 shrink-0">{mainTopBar}</div>
      ) : null}
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
        {main}
      </div>
    </main>
  );
}
