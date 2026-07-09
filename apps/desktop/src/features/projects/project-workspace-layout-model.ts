import type { ReactNode } from "react";

import type { ProjectPanelLayout } from "@/features/projects/project-preferences-store";

export type ProjectSidebarState = "expanded" | "collapsed";

export type ProjectColumnShellProps = {
  bottomPanel: (height: number) => ReactNode;
  leftSidebar: ReactNode;
  main: ReactNode;
  mainTopBar?: ReactNode;
  onRightOpenChange: (open: boolean) => void;
  rightOpen: boolean;
  rightSidebar: ReactNode;
  showTerminalButton?: boolean;
  topBar: (controls: {
    leftSidebarState: ProjectSidebarState;
    onToggleLeftSidebar: () => void;
  }) => ReactNode;
};

export type ProjectColumnShellContentProps = ProjectColumnShellProps & {
  panelLayout: ProjectPanelLayout;
  setPanelLayout: (layout: ProjectPanelLayout) => void;
};

export function getProjectRightTopbarActionsWidth({
  rightOpen,
  showTerminalButton,
}: {
  rightOpen: boolean;
  showTerminalButton?: boolean;
}) {
  if (rightOpen && showTerminalButton) return "120px";
  if (showTerminalButton || rightOpen) return "84px";
  return "40px";
}
