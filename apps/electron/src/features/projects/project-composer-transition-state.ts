import { useCallback, useLayoutEffect, useRef, useState } from "react";

import {
  CONVERSATION_OPEN_ANIMATION_MS,
} from "@/features/projects/project-workspace-state-model";

export function useProjectComposerTransitionState({
  activeSessionId,
  showConversationLayout,
}: {
  activeSessionId: string;
  showConversationLayout: boolean;
}) {
  const [composerHeight, setComposerHeight] = useState(116);
  const [, setComposerExtraHeight] = useState(0);
  const composerFrameRef = useRef<HTMLDivElement>(null);
  const pendingTransitionRectRef = useRef<DOMRect | null>(null);
  const transitionFrameRef = useRef(0);
  const transitionTimeoutRef = useRef(0);

  const handleComposerHeightChange = useCallback((height: number) => {
    const nextHeight = Math.round(height);
    setComposerHeight((currentHeight) =>
      currentHeight === nextHeight ? currentHeight : nextHeight,
    );
  }, []);

  const stopComposerTransition = useCallback(() => {
    if (transitionFrameRef.current) {
      window.cancelAnimationFrame(transitionFrameRef.current);
    }
    if (transitionTimeoutRef.current) {
      window.clearTimeout(transitionTimeoutRef.current);
    }
    transitionFrameRef.current = 0;
    transitionTimeoutRef.current = 0;
    const composerElement = composerFrameRef.current;
    if (!composerElement) return;
    composerElement.style.transform = "";
    composerElement.style.transition = "";
  }, []);

  const captureComposerTransitionStart = useCallback(() => {
    const composerElement = composerFrameRef.current;
    stopComposerTransition();
    pendingTransitionRectRef.current =
      composerElement?.getBoundingClientRect() ?? null;
    if (composerElement) {
      composerElement.style.transition = "none";
    }
  }, [stopComposerTransition]);

  useLayoutEffect(() => {
    const fromRect = pendingTransitionRectRef.current;
    const composerElement = composerFrameRef.current;
    pendingTransitionRectRef.current = null;
    if (!fromRect || !composerElement) return;

    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;

    const toRect = composerElement.getBoundingClientRect();
    const deltaX = fromRect.left - toRect.left;
    const deltaY = fromRect.top - toRect.top;
    if (Math.abs(deltaX) < 0.5 && Math.abs(deltaY) < 0.5) return;

    stopComposerTransition();
    composerElement.style.transition = "none";
    composerElement.style.transform = `translate(${deltaX}px, ${deltaY}px)`;
    composerElement.getBoundingClientRect();

    transitionFrameRef.current = window.requestAnimationFrame(() => {
      transitionFrameRef.current = 0;
      composerElement.style.transition = `transform ${CONVERSATION_OPEN_ANIMATION_MS}ms cubic-bezier(0.22,1,0.36,1)`;
      composerElement.style.transform = "";
      transitionTimeoutRef.current = window.setTimeout(() => {
        transitionTimeoutRef.current = 0;
        composerElement.style.transition = "";
      }, CONVERSATION_OPEN_ANIMATION_MS + 40);
    });
  }, [activeSessionId, showConversationLayout, stopComposerTransition]);

  const emptyComposerBottom = "clamp(2rem, 11vh, 7.5rem)";

  return {
    captureComposerTransitionStart,
    composerBottom: showConversationLayout ? "1rem" : emptyComposerBottom,
    composerBottomSm: showConversationLayout ? "1.5rem" : emptyComposerBottom,
    composerFrameRef,
    composerHeight,
    emptyComposerTop: "calc(50% - 2.5rem)",
    handleComposerHeightChange,
    setComposerExtraHeight,
    stopComposerTransition,
  };
}
