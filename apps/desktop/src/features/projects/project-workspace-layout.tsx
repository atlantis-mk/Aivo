import { type CSSProperties } from "react";

import { Sidebar, SidebarProvider, useSidebar } from "@/components/ui/sidebar";
import { useProjectPreferencesStore } from "@/features/projects/project-preferences-store";
import { ProjectColumnWorkspace } from "./project-column-workspace";
import {
  getProjectRightTopbarActionsWidth,
  type ProjectColumnShellContentProps,
  type ProjectColumnShellProps,
} from "./project-workspace-layout-model";
import { useProjectPanelLayoutRuntime } from "./project-workspace-layout-runtime";
import { ProjectResizeHandle } from "./project-workspace-resize-handle";
import { ProjectFloatingRightTopBarActions } from "./project-workspace-topbar-actions";

export { ProjectTopBarIconButton } from "./project-workspace-topbar-actions";

export function ProjectColumnShell({
  bottomPanel,
  leftSidebar,
  main,
  mainTopBar,
  onRightOpenChange,
  rightOpen,
  rightSidebar,
  showTerminalButton,
  topBar,
}: ProjectColumnShellProps) {
  const panelLayout = useProjectPreferencesStore((state) => state.panelLayout);
  const setPanelLayout = useProjectPreferencesStore(
    (state) => state.setPanelLayout,
  );
  const rightTopbarActionsWidth = getProjectRightTopbarActionsWidth({
    rightOpen,
    showTerminalButton,
  });

  return (
    <SidebarProvider
      className="h-dvh !min-h-0 overflow-hidden bg-background text-foreground"
      style={
        {
          "--project-bottom-panel-height": `${panelLayout.bottomHeight}px`,
          "--project-left-sidebar-width": `${panelLayout.leftWidth}px`,
          "--project-right-sidebar-width": `${panelLayout.rightWidth}px`,
          "--project-right-topbar-actions-width": rightTopbarActionsWidth,
          "--sidebar-width": "var(--project-left-sidebar-width)",
          "--sidebar-width-icon": "52px",
        } as CSSProperties
      }
    >
      <ProjectColumnShellContent
        bottomPanel={bottomPanel}
        leftSidebar={leftSidebar}
        main={main}
        mainTopBar={mainTopBar}
        onRightOpenChange={onRightOpenChange}
        panelLayout={panelLayout}
        rightOpen={rightOpen}
        rightSidebar={rightSidebar}
        setPanelLayout={setPanelLayout}
        showTerminalButton={showTerminalButton}
        topBar={topBar}
      />
    </SidebarProvider>
  );
}

function ProjectColumnShellContent({
  bottomPanel,
  leftSidebar,
  main,
  mainTopBar,
  onRightOpenChange,
  panelLayout,
  rightOpen,
  rightSidebar,
  setPanelLayout,
  showTerminalButton,
  topBar,
}: ProjectColumnShellContentProps) {
  const { state: leftSidebarState, toggleSidebar: toggleLeftSidebar } =
    useSidebar();
  const {
    isRightSidebarMaximized,
    rootRef,
    startPanelResize,
    toggleRightSidebarMaximized,
  } = useProjectPanelLayoutRuntime({
    panelLayout,
    rightOpen,
    setPanelLayout,
  });
  const rightTopbarActionsWidth = getProjectRightTopbarActionsWidth({
    rightOpen,
    showTerminalButton,
  });

  return (
    <div
      className="relative flex h-full min-h-0 min-w-0 flex-1 overflow-hidden [&_[data-slot=sidebar-container]]:[transition-duration:var(--project-panel-transition-duration,200ms)] [&_[data-slot=sidebar-gap]]:[transition-duration:var(--project-panel-transition-duration,200ms)]"
      ref={rootRef}
      style={
        {
          "--project-bottom-panel-height": `${panelLayout.bottomHeight}px`,
          "--project-left-sidebar-width": `${panelLayout.leftWidth}px`,
          "--project-right-sidebar-width": `${panelLayout.rightWidth}px`,
          "--project-right-topbar-actions-width": rightTopbarActionsWidth,
          "--sidebar-width": "var(--project-left-sidebar-width)",
        } as CSSProperties
      }
    >
      <Sidebar className="z-40" collapsible="offcanvas">
        {leftSidebar}
      </Sidebar>
      {leftSidebarState === "expanded" ? (
        <ProjectResizeHandle
          ariaLabel="调整左侧栏宽度"
          className="absolute inset-y-0 z-40 -ml-px hidden md:flex"
          orientation="vertical"
          onResizeStart={(event) => startPanelResize(event, "leftWidth", true)}
          style={{ left: "var(--project-left-sidebar-width)" }}
        />
      ) : null}
      <SidebarProvider
        className="!contents"
        onOpenChange={onRightOpenChange}
        open={rightOpen}
        style={
          {
            "--sidebar-width": "var(--project-right-sidebar-width)",
          } as CSSProperties
        }
      >
        <div className="pointer-events-none absolute inset-x-0 top-0 z-50 h-9">
          <div className="pointer-events-none h-full">
            {topBar({
              leftSidebarState,
              onToggleLeftSidebar: toggleLeftSidebar,
            })}
          </div>
        </div>
        <ProjectFloatingRightTopBarActions
          isRightSidebarMaximized={isRightSidebarMaximized}
          onToggleRightSidebarMaximized={toggleRightSidebarMaximized}
          rightOpen={rightOpen}
          showTerminalButton={showTerminalButton}
        />
        <ProjectColumnWorkspace
          bottomPanel={bottomPanel}
          leftSidebarState={leftSidebarState}
          main={main}
          mainTopBar={mainTopBar}
          onResizeStart={(event, key) => {
            startPanelResize(event, key, leftSidebarState === "expanded");
          }}
          panelLayout={panelLayout}
          rightSidebarMaximized={isRightSidebarMaximized}
          rightSidebar={rightSidebar}
        />
      </SidebarProvider>
    </div>
  );
}
