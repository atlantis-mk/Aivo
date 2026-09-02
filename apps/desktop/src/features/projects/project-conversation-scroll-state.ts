import { useCallback, useRef, useState } from "react";

import {
  useConversationContentResizeEffect,
  useConversationScrollProgression,
  useConversationScrollViewportBinding,
  useMarkdownContentResizeEffect,
} from "@/features/projects/project-conversation-scroll-effects";
import {
  FORCE_BOTTOM_FRAME_COUNT,
  SCROLL_BOTTOM_ANIMATION_MS,
  SCROLL_BOTTOM_SENTINEL,
  SHOW_SCROLL_TO_BOTTOM_DISTANCE,
} from "@/features/projects/project-workspace-state-model";

export function useProjectConversationScroll({
  composerHeight,
  hasTurns,
  lastTurnStateKey,
  showConversationLayout,
  turnCount,
}: {
  composerHeight: number;
  hasTurns: boolean;
  lastTurnStateKey: string;
  showConversationLayout: boolean;
  turnCount: number;
}) {
  const rootRef = useRef<HTMLDivElement>(null);
  const viewportRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const stickToBottomRef = useRef(true);
  const previousTurnCountRef = useRef(0);
  const snapNextScrollRef = useRef(false);
  const forceStickToBottomRef = useRef(false);
  const forceBottomFrameRef = useRef(0);
  const forceBottomRemainingFramesRef = useRef(0);
  const scrollAnimationFrameRef = useRef(0);
  const resizeScrollFrameRef = useRef(0);
  const [showScrollToBottomButton, setShowScrollToBottomButton] =
    useState(false);

  const updateStickToBottom = useCallback((viewport: HTMLDivElement) => {
    if (forceStickToBottomRef.current) {
      stickToBottomRef.current = true;
      setShowScrollToBottomButton(false);
      return;
    }

    const distanceToBottom =
      viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight;
    const isAtBottom = distanceToBottom < SHOW_SCROLL_TO_BOTTOM_DISTANCE;
    stickToBottomRef.current = isAtBottom;
    setShowScrollToBottomButton(!isAtBottom);
  }, []);

  const scrollToBottom = useCallback((behavior: ScrollBehavior = "smooth") => {
    const viewport = viewportRef.current;
    if (!viewport) return;

    if (behavior === "auto") {
      viewport.scrollTop = SCROLL_BOTTOM_SENTINEL;
      setShowScrollToBottomButton(false);
      return;
    }

    setShowScrollToBottomButton(false);
    viewport.scrollTo({
      behavior,
      top: SCROLL_BOTTOM_SENTINEL,
    });
  }, []);

  const handleScrollToBottomButtonClick = useCallback(() => {
    stickToBottomRef.current = true;
    setShowScrollToBottomButton(false);
    scrollToBottom("smooth");
  }, [scrollToBottom]);

  const stopScrollAnimation = useCallback(() => {
    if (scrollAnimationFrameRef.current) {
      window.cancelAnimationFrame(scrollAnimationFrameRef.current);
    }
    scrollAnimationFrameRef.current = 0;
  }, []);

  const animateToBottom = useCallback(() => {
    const viewport = viewportRef.current;
    if (!viewport) return;

    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      scrollToBottom("auto");
      return;
    }

    stopScrollAnimation();

    const startTop = viewport.scrollTop;
    const startTime = performance.now();

    const scrollStep = (time: number) => {
      const currentViewport = viewportRef.current;
      if (!currentViewport) {
        scrollAnimationFrameRef.current = 0;
        return;
      }

      const progress = Math.min(
        1,
        (time - startTime) / SCROLL_BOTTOM_ANIMATION_MS,
      );
      const easedProgress = 1 - Math.pow(1 - progress, 3);
      const targetTop = Math.max(
        0,
        currentViewport.scrollHeight - currentViewport.clientHeight,
      );
      currentViewport.scrollTop =
        startTop + (targetTop - startTop) * easedProgress;

      if (progress < 1) {
        scrollAnimationFrameRef.current =
          window.requestAnimationFrame(scrollStep);
        return;
      }

      scrollAnimationFrameRef.current = 0;
      currentViewport.scrollTop = SCROLL_BOTTOM_SENTINEL;
    };

    scrollAnimationFrameRef.current = window.requestAnimationFrame(scrollStep);
  }, [scrollToBottom, stopScrollAnimation]);

  const updateStickToBottomFromCurrentViewport = useCallback(() => {
    const viewport = viewportRef.current;
    if (!viewport) return;
    updateStickToBottom(viewport);
  }, [updateStickToBottom]);

  const startForceScrollToBottom = useCallback(
    (frameCount = FORCE_BOTTOM_FRAME_COUNT) => {
      forceStickToBottomRef.current = true;
      forceBottomRemainingFramesRef.current = Math.max(
        forceBottomRemainingFramesRef.current,
        frameCount,
      );
      if (forceBottomFrameRef.current) return;

      const scrollStep = () => {
        if (!scrollAnimationFrameRef.current) {
          scrollToBottom("auto");
        }
        forceBottomRemainingFramesRef.current -= 1;

        if (forceBottomRemainingFramesRef.current > 0) {
          forceBottomFrameRef.current =
            window.requestAnimationFrame(scrollStep);
          return;
        }

        forceBottomFrameRef.current = 0;
        forceStickToBottomRef.current = false;
        updateStickToBottomFromCurrentViewport();
      };

      forceBottomFrameRef.current = window.requestAnimationFrame(scrollStep);
    },
    [scrollToBottom, updateStickToBottomFromCurrentViewport],
  );

  const scheduleResizeScrollToBottom = useCallback(() => {
    if (!stickToBottomRef.current && !forceStickToBottomRef.current) return;
    if (resizeScrollFrameRef.current) return;

    resizeScrollFrameRef.current = window.requestAnimationFrame(() => {
      resizeScrollFrameRef.current = 0;
      if (!stickToBottomRef.current && !forceStickToBottomRef.current) return;
      if (forceStickToBottomRef.current && scrollAnimationFrameRef.current) {
        startForceScrollToBottom(6);
        return;
      }

      scrollToBottom("auto");
      if (forceStickToBottomRef.current) {
        startForceScrollToBottom(6);
      }
    });
  }, [scrollToBottom, startForceScrollToBottom]);

  const stopForceScrollToBottom = useCallback(() => {
    if (forceBottomFrameRef.current) {
      window.cancelAnimationFrame(forceBottomFrameRef.current);
    }
    if (resizeScrollFrameRef.current) {
      window.cancelAnimationFrame(resizeScrollFrameRef.current);
    }
    forceBottomFrameRef.current = 0;
    forceBottomRemainingFramesRef.current = 0;
    resizeScrollFrameRef.current = 0;
    forceStickToBottomRef.current = false;
    stopScrollAnimation();
  }, [stopScrollAnimation]);

  const prepareConversationReveal = useCallback(
    (nextTurnCount: number) => {
      snapNextScrollRef.current = true;
      forceStickToBottomRef.current = true;
      stickToBottomRef.current = true;
      previousTurnCountRef.current = nextTurnCount;
      startForceScrollToBottom();
    },
    [startForceScrollToBottom],
  );

  const resetConversationScroll = useCallback(() => {
    snapNextScrollRef.current = false;
    stopForceScrollToBottom();
  }, [stopForceScrollToBottom]);

  useConversationScrollViewportBinding({
    rootRef,
    showConversationLayout,
    updateStickToBottom,
    viewportRef,
  });
  useConversationScrollProgression({
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
  });
  useConversationContentResizeEffect({
    contentRef,
    hasTurns,
    scheduleResizeScrollToBottom,
  });
  useMarkdownContentResizeEffect({
    hasTurns,
    scheduleResizeScrollToBottom,
  });

  return {
    contentRef,
    handleScrollToBottomButtonClick,
    prepareConversationReveal,
    resetConversationScroll,
    rootRef,
    showScrollToBottomButton,
    stopForceScrollToBottom,
  };
}
