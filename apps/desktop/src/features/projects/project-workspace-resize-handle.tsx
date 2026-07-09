import {
  type CSSProperties,
  type PointerEvent as ReactPointerEvent,
} from "react";

import { cn } from "@/lib/utils";

export function ProjectResizeHandle({
  ariaLabel,
  className,
  onResizeStart,
  orientation,
  style,
}: {
  ariaLabel: string;
  className?: string;
  onResizeStart: (event: ReactPointerEvent<HTMLButtonElement>) => void;
  orientation: "horizontal" | "vertical";
  style?: CSSProperties;
}) {
  return (
    <button
      aria-label={ariaLabel}
      className={cn(
        "group/resize relative shrink-0 bg-transparent outline-none ring-offset-background transition-colors focus-visible:ring-0",
        orientation === "vertical"
          ? "flex h-full w-px cursor-col-resize items-center justify-center after:absolute after:inset-y-0 after:left-1/2 after:w-2 after:-translate-x-1/2"
          : "flex h-px w-full cursor-row-resize items-center justify-center after:absolute after:left-0 after:top-1/2 after:h-2 after:w-full after:-translate-y-1/2",
        className,
      )}
      onPointerDown={onResizeStart}
      style={style}
      type="button"
    />
  );
}
