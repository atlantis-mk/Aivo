import {
  type PointerEvent as ReactPointerEvent,
} from "react";

import type { ProjectPanelLayout } from "@/features/projects/project-preferences-store";

const PROJECT_LEFT_SIDEBAR_MIN_WIDTH = 210;
const PROJECT_RIGHT_SIDEBAR_MIN_WIDTH = 240;
const PROJECT_BOTTOM_PANEL_MIN_HEIGHT = 180;
const PROJECT_MAIN_MIN_WIDTH = 360;
const PROJECT_UPPER_MIN_HEIGHT = 240;

function clampNumber(value: number, min: number, max: number) {
  if (!Number.isFinite(value)) return min;
  return Math.min(Math.max(value, min), Math.max(min, max));
}

export function getRightSidebarMaximizedWidth(root: HTMLDivElement) {
  const workspace = root.querySelector<HTMLElement>(
    "[data-project-workspace-content]",
  );
  const rect = (workspace ?? root).getBoundingClientRect();
  return Math.max(PROJECT_RIGHT_SIDEBAR_MIN_WIDTH, Math.round(rect.width));
}

export function startProjectPanelResize({
  commitResizedPanelLayout,
  event,
  key,
  leftOpen,
  panelLayout,
  rightOpen,
  root,
  updatePanelLayoutVariable,
}: {
  commitResizedPanelLayout: (layout: ProjectPanelLayout) => void;
  event: ReactPointerEvent<HTMLButtonElement>;
  key: keyof ProjectPanelLayout;
  leftOpen: boolean;
  panelLayout: ProjectPanelLayout;
  rightOpen: boolean;
  root: HTMLDivElement;
  updatePanelLayoutVariable: (
    key: keyof ProjectPanelLayout,
    value: number,
  ) => void;
}) {
  event.preventDefault();
  event.currentTarget.setPointerCapture(event.pointerId);

  const rect = root.getBoundingClientRect();
  const startX = event.clientX;
  const startY = event.clientY;
  const startLayout = panelLayout;
  const leftVisible = leftOpen ? startLayout.leftWidth : 0;
  const rightVisible = rightOpen ? startLayout.rightWidth : 0;
  let latestValue = startLayout[key];
  let latestClientX = startX;
  let latestClientY = startY;
  let hasMoved = false;
  let frame = 0;
  const previousCursor = document.body.style.cursor;
  const previousUserSelect = document.body.style.userSelect;

  function clampLayoutValue(clientX: number, clientY: number) {
    if (key === "leftWidth") {
      const maxWidth = rect.width - rightVisible - PROJECT_MAIN_MIN_WIDTH;
      return clampNumber(
        clientX - rect.left,
        PROJECT_LEFT_SIDEBAR_MIN_WIDTH,
        maxWidth,
      );
    }

    if (key === "rightWidth") {
      const maxWidth = rect.width - leftVisible - PROJECT_MAIN_MIN_WIDTH;
      return clampNumber(
        rect.right - clientX,
        PROJECT_RIGHT_SIDEBAR_MIN_WIDTH,
        maxWidth,
      );
    }

    return clampNumber(
      rect.bottom - clientY,
      PROJECT_BOTTOM_PANEL_MIN_HEIGHT,
      rect.height - PROJECT_UPPER_MIN_HEIGHT,
    );
  }

  function scheduleUpdate() {
    if (frame) return;
    frame = window.requestAnimationFrame(() => {
      frame = 0;
      latestValue = clampLayoutValue(latestClientX, latestClientY);
      updatePanelLayoutVariable(key, latestValue);
    });
  }

  function handlePointerMove(moveEvent: PointerEvent) {
    latestClientX = moveEvent.clientX;
    latestClientY = moveEvent.clientY;
    hasMoved =
      hasMoved ||
      Math.abs(latestClientX - startX) > 1 ||
      Math.abs(latestClientY - startY) > 1;
    scheduleUpdate();
  }

  function handlePointerUp() {
    if (frame) {
      window.cancelAnimationFrame(frame);
      frame = 0;
    }
    latestValue = clampLayoutValue(latestClientX, latestClientY);
    updatePanelLayoutVariable(key, latestValue);
    if (hasMoved) {
      commitResizedPanelLayout({
        ...startLayout,
        [key]: Math.round(latestValue),
      });
    }
    document.body.style.cursor = previousCursor;
    document.body.style.userSelect = previousUserSelect;
    root.style.removeProperty("--project-panel-transition-duration");
    window.removeEventListener("pointermove", handlePointerMove);
    window.removeEventListener("pointerup", handlePointerUp);
    window.removeEventListener("pointercancel", handlePointerUp);
  }

  document.body.style.cursor =
    key === "bottomHeight" ? "row-resize" : "col-resize";
  document.body.style.userSelect = "none";
  root.style.setProperty("--project-panel-transition-duration", "0ms");
  window.addEventListener("pointermove", handlePointerMove);
  window.addEventListener("pointerup", handlePointerUp);
  window.addEventListener("pointercancel", handlePointerUp);
}
