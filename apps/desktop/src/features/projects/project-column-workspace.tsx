import {
  type CSSProperties,
  type PointerEvent as ReactPointerEvent,
  type ReactNode,
} from "react";

import { Sidebar, SidebarInset, useSidebar } from "@/components/ui/sidebar";
import { TerminalDockPanel } from "@/features/projects/terminal/terminal-dock";
import { useTerminalDock } from "@/features/projects/terminal/terminal-dock-store";
import type { ProjectPanelLayout } from "@/features/projects/project-preferences-store";
import { ProjectResizeHandle } from "./project-workspace-resize-handle";

export function ProjectColumnWorkspace({
  bottomPanel,
  leftSidebarState,
  main,
  mainTopBar,
  onResizeStart,
  panelLayout,
  rightSidebarMaximized,
  rightSidebar,
}: {
  bottomPanel: (height: number) => ReactNode;
  leftSidebarState: "expanded" | "collapsed";
  main: ReactNode;
  mainTopBar?: ReactNode;
  onResizeStart: (
    event: ReactPointerEvent<HTMLButtonElement>,
    key: "bottomHeight" | "rightWidth",
  ) => void;
  panelLayout: ProjectPanelLayout;
  rightSidebarMaximized: boolean;
  rightSidebar: ReactNode;
}) {
  const { state: bottomPanelState } = useTerminalDock();
  const { state: rightSidebarState } = useSidebar();
  const bottomOpen = bottomPanelState === "expanded";
  const rightOpen = rightSidebarState === "expanded";
  const isMac = window.aivo?.platform === "darwin";
  const leftCompactWidth = isMac ? 202 : 148;
  const mainTopBarLeftOffset =
    leftSidebarState === "collapsed"
      ? `calc(${leftCompactWidth}px - var(--sidebar-width-icon, 52px))`
      : undefined;

  return (
    <SidebarInset className="h-full min-h-0 min-w-0 overflow-hidden">
      <div className="flex h-full min-h-0 flex-col overflow-hidden">
        <div
          className="relative flex min-h-0 flex-1 overflow-hidden"
          data-project-workspace-content
        >
          <main className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
            {mainTopBar ? (
              <div
                className="z-50 h-9 shrink-0 transition-[margin-left] duration-[var(--project-panel-transition-duration,200ms)] ease-linear"
                style={{ marginLeft: mainTopBarLeftOffset }}
              >
                {mainTopBar}
              </div>
            ) : null}
            <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
              {main}
            </div>
          </main>
          {rightOpen && !rightSidebarMaximized ? (
            <ProjectResizeHandle
              ariaLabel="调整右侧栏宽度"
              className="absolute inset-y-0 z-40 -mr-px hidden md:flex"
              orientation="vertical"
              onResizeStart={(event) => onResizeStart(event, "rightWidth")}
              style={{ right: "var(--project-right-sidebar-width)" }}
            />
          ) : null}
          <Sidebar
            className="!absolute !inset-y-0 !z-[70] !h-auto bg-background"
            collapsible="offcanvas"
            side="right"
            style={
              {
                "--sidebar-width": "var(--project-right-sidebar-width)",
              } as CSSProperties
            }
          >
            {rightSidebar}
          </Sidebar>
        </div>
        {bottomOpen ? (
          <ProjectResizeHandle
            ariaLabel="调整底部栏高度"
            orientation="horizontal"
            onResizeStart={(event) => onResizeStart(event, "bottomHeight")}
          />
        ) : null}
        <TerminalDockPanel height="var(--project-bottom-panel-height)">
          {bottomPanel(panelLayout.bottomHeight)}
        </TerminalDockPanel>
      </div>
    </SidebarInset>
  );
}
