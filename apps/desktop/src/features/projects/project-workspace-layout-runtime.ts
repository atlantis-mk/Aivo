import {
  type PointerEvent as ReactPointerEvent,
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";

import type { ProjectPanelLayout } from "@/features/projects/project-preferences-store";
import {
  getRightSidebarMaximizedWidth,
  startProjectPanelResize,
} from "@/features/projects/project-workspace-resize";

export function useProjectPanelLayoutRuntime({
  panelLayout,
  rightOpen,
  setPanelLayout,
}: {
  panelLayout: ProjectPanelLayout;
  rightOpen: boolean;
  setPanelLayout: (layout: ProjectPanelLayout) => void;
}) {
  const rootRef = useRef<HTMLDivElement | null>(null);
  const panelLayoutRef = useRef(panelLayout);
  const [isRightSidebarMaximized, setRightSidebarMaximized] = useState(false);

  useEffect(() => {
    panelLayoutRef.current = panelLayout;
  }, [panelLayout]);

  const commitResizedPanelLayout = useCallback(
    (nextLayout: ProjectPanelLayout) => {
      panelLayoutRef.current = nextLayout;
      setPanelLayout(nextLayout);
    },
    [setPanelLayout],
  );

  const updatePanelLayoutVariable = useCallback(
    (key: keyof ProjectPanelLayout, value: number) => {
      const root = rootRef.current;
      if (!root) return;
      const cssName =
        key === "leftWidth"
          ? "--project-left-sidebar-width"
          : key === "rightWidth"
            ? "--project-right-sidebar-width"
            : "--project-bottom-panel-height";
      root.style.setProperty(cssName, `${Math.round(value)}px`);
    },
    [],
  );

  const applyRightSidebarRuntimeWidth = useCallback(
    (maximized: boolean) => {
      const root = rootRef.current;
      if (!root) return;
      const nextWidth = maximized
        ? getRightSidebarMaximizedWidth(root)
        : panelLayoutRef.current.rightWidth;
      updatePanelLayoutVariable("rightWidth", nextWidth);
    },
    [updatePanelLayoutVariable],
  );

  const toggleRightSidebarMaximized = useCallback(() => {
    setRightSidebarMaximized((current) => {
      const next = !current;
      applyRightSidebarRuntimeWidth(next);
      return next;
    });
  }, [applyRightSidebarRuntimeWidth]);

  const startPanelResize = useCallback(
    (
      event: ReactPointerEvent<HTMLButtonElement>,
      key: keyof ProjectPanelLayout,
      leftOpen: boolean,
    ) => {
      const root = rootRef.current;
      if (!root) return;
      startProjectPanelResize({
        commitResizedPanelLayout,
        event,
        key,
        leftOpen,
        panelLayout: panelLayoutRef.current,
        rightOpen,
        root,
        updatePanelLayoutVariable,
      });
    },
    [commitResizedPanelLayout, rightOpen, updatePanelLayoutVariable],
  );

  useLayoutEffect(() => {
    if (!rightOpen && isRightSidebarMaximized) {
      setRightSidebarMaximized(false);
      updatePanelLayoutVariable("rightWidth", panelLayout.rightWidth);
      return;
    }

    applyRightSidebarRuntimeWidth(rightOpen && isRightSidebarMaximized);
  }, [
    applyRightSidebarRuntimeWidth,
    isRightSidebarMaximized,
    panelLayout.leftWidth,
    panelLayout.rightWidth,
    rightOpen,
    updatePanelLayoutVariable,
  ]);

  useEffect(() => {
    if (!rightOpen || !isRightSidebarMaximized) return;

    function handleResize() {
      applyRightSidebarRuntimeWidth(true);
    }

    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, [applyRightSidebarRuntimeWidth, isRightSidebarMaximized, rightOpen]);

  return {
    isRightSidebarMaximized,
    rootRef,
    startPanelResize,
    toggleRightSidebarMaximized,
  };
}
