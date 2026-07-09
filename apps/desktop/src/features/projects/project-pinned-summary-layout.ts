import { useLayoutEffect, useRef, useState } from "react";

export function useProjectPinnedSummaryLayout() {
  const mainRef = useRef<HTMLDivElement>(null);
  const [canDockPinnedSummary, setCanDockPinnedSummary] = useState(false);
  const [shouldShiftPinnedSummaryLayout, setShouldShiftPinnedSummaryLayout] =
    useState(false);

  useLayoutEffect(() => {
    const mainElement = mainRef.current;
    if (!mainElement) return;

    const updatePinnedSummaryLayout = () => {
      const messageWidth = 680;
      const summaryWidth = 288;
      const sideInset = 24;
      const summaryGap = 24;
      const dockShift = 160;
      const mainWidth = mainElement.clientWidth;
      const contentWidth = Math.min(messageWidth, Math.max(0, mainWidth - 48));
      const contentRight = (mainWidth + contentWidth) / 2;
      const summaryLeft = mainWidth - sideInset - summaryWidth;
      const hasNaturalDockSpace = summaryLeft - contentRight >= summaryGap;
      const hasShiftedDockSpace =
        summaryLeft - (contentRight - dockShift) >= summaryGap;
      const nextCanDock = hasNaturalDockSpace || hasShiftedDockSpace;
      const nextShouldShift = !hasNaturalDockSpace && hasShiftedDockSpace;

      setCanDockPinnedSummary((current) =>
        current === nextCanDock ? current : nextCanDock,
      );
      setShouldShiftPinnedSummaryLayout((current) =>
        current === nextShouldShift ? current : nextShouldShift,
      );
    };

    updatePinnedSummaryLayout();
    const resizeObserver = new ResizeObserver(updatePinnedSummaryLayout);
    resizeObserver.observe(mainElement);

    return () => resizeObserver.disconnect();
  }, []);

  return {
    canDockPinnedSummary,
    mainRef,
    shouldShiftPinnedSummaryLayout,
  };
}
