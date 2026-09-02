import { useLayoutEffect } from "react";
import type { Dispatch, SetStateAction } from "react";

import { MARKDOWN_CONTENT_RESIZE_EVENT } from "@/features/projects/project-workspace-state-model";

type MutableRef<T> = {
  current: T;
};

export function useConversationScrollViewportBinding({
  rootRef,
  showConversationLayout,
  updateStickToBottom,
  viewportRef,
}: {
  rootRef: MutableRef<HTMLDivElement | null>;
  showConversationLayout: boolean;
  updateStickToBottom: (viewport: HTMLDivElement) => void;
  viewportRef: MutableRef<HTMLDivElement | null>;
}) {
  useLayoutEffect(() => {
    const root = rootRef.current;
    if (!root || !showConversationLayout) {
      viewportRef.current = null;
      return;
    }

    const viewport = root.querySelector<HTMLDivElement>(
      '[data-slot="scroll-area-viewport"]',
    );
    if (!viewport) {
      viewportRef.current = null;
      return;
    }

    viewportRef.current = viewport;

    const handleScroll = () => {
      updateStickToBottom(viewport);
    };

    viewport.addEventListener("scroll", handleScroll, { passive: true });
    updateStickToBottom(viewport);

    return () => {
      viewport.removeEventListener("scroll", handleScroll);
      if (viewportRef.current === viewport) {
        viewportRef.current = null;
      }
    };
  }, [rootRef, showConversationLayout, updateStickToBottom, viewportRef]);
}

export function useConversationScrollProgression({
  animateToBottom,
  composerHeight,
  hasTurns,
  lastTurnStateKey,
  previousTurnCountRef,
  scrollToBottom,
  setShowScrollToBottomButton,
  snapNextScrollRef,
  startForceScrollToBottom,
  stickToBottomRef,
  turnCount,
}: {
  animateToBottom: () => void;
  composerHeight: number;
  hasTurns: boolean;
  lastTurnStateKey: string;
  previousTurnCountRef: MutableRef<number>;
  scrollToBottom: (behavior?: ScrollBehavior) => void;
  setShowScrollToBottomButton: Dispatch<SetStateAction<boolean>>;
  snapNextScrollRef: MutableRef<boolean>;
  startForceScrollToBottom: () => void;
  stickToBottomRef: MutableRef<boolean>;
  turnCount: number;
}) {
  useLayoutEffect(() => {
    if (!hasTurns) {
      previousTurnCountRef.current = 0;
      stickToBottomRef.current = true;
      setShowScrollToBottomButton(false);
      return;
    }

    const hasNewTurn = turnCount > previousTurnCountRef.current;
    previousTurnCountRef.current = turnCount;

    if (snapNextScrollRef.current) {
      snapNextScrollRef.current = false;
      stickToBottomRef.current = true;
      animateToBottom();
      startForceScrollToBottom();
      return;
    }

    if (hasNewTurn) {
      stickToBottomRef.current = true;
      scrollToBottom("smooth");
      return;
    }

    if (stickToBottomRef.current) {
      scrollToBottom("auto");
    }
  }, [
    animateToBottom,
    composerHeight,
    hasTurns,
    lastTurnStateKey,
    previousTurnCountRef,
    scrollToBottom,
    setShowScrollToBottomButton,
    snapNextScrollRef,
    startForceScrollToBottom,
    stickToBottomRef,
    turnCount,
  ]);
}

export function useConversationContentResizeEffect({
  contentRef,
  hasTurns,
  scheduleResizeScrollToBottom,
}: {
  contentRef: MutableRef<HTMLDivElement | null>;
  hasTurns: boolean;
  scheduleResizeScrollToBottom: () => void;
}) {
  useLayoutEffect(() => {
    const contentElement = contentRef.current;
    if (!contentElement || !hasTurns) return;

    const resizeObserver = new ResizeObserver(scheduleResizeScrollToBottom);

    resizeObserver.observe(contentElement);

    return () => {
      resizeObserver.disconnect();
    };
  }, [contentRef, hasTurns, scheduleResizeScrollToBottom]);
}

export function useMarkdownContentResizeEffect({
  hasTurns,
  scheduleResizeScrollToBottom,
}: {
  hasTurns: boolean;
  scheduleResizeScrollToBottom: () => void;
}) {
  useLayoutEffect(() => {
    if (!hasTurns) return;

    function handleMarkdownContentResize() {
      scheduleResizeScrollToBottom();
    }

    window.addEventListener(
      MARKDOWN_CONTENT_RESIZE_EVENT,
      handleMarkdownContentResize,
    );

    return () => {
      window.removeEventListener(
        MARKDOWN_CONTENT_RESIZE_EVENT,
        handleMarkdownContentResize,
      );
    };
  }, [hasTurns, scheduleResizeScrollToBottom]);
}
