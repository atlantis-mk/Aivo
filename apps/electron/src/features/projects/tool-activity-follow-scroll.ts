import { useEffect, useRef } from "react";

export function useFollowScrollToEnd<TTrigger>(trigger: TTrigger) {
  const endRef = useRef<HTMLDivElement>(null);
  const scrollAreaRef = useRef<HTMLDivElement>(null);
  const followContentRef = useRef(true);

  useEffect(() => {
    const viewport = scrollAreaRef.current?.querySelector<HTMLElement>(
      "[data-slot=scroll-area-viewport]",
    );
    if (!viewport) return;

    const updateFollowState = () => {
      const distanceToBottom =
        viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight;
      followContentRef.current = distanceToBottom < 24;
    };

    viewport.addEventListener("scroll", updateFollowState);
    updateFollowState();

    return () => {
      viewport.removeEventListener("scroll", updateFollowState);
    };
  }, []);

  useEffect(() => {
    if (!followContentRef.current) return;
    const frame = requestAnimationFrame(() => {
      endRef.current?.scrollIntoView({ block: "end" });
    });
    return () => cancelAnimationFrame(frame);
  }, [trigger]);

  return {
    endRef,
    scrollAreaRef,
  };
}
