import { useLayoutEffect, useRef, useState } from "react";

import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

export function ProjectSessionRuntimeStatsLine({ value }: { value: string }) {
  const lineRef = useRef<HTMLDivElement>(null);
  const [truncated, setTruncated] = useState(false);

  useLayoutEffect(() => {
    const element = lineRef.current;
    if (!element || !value) return;
    const measure = () => setTruncated(element.scrollWidth > element.clientWidth);
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(element);
    return () => observer.disconnect();
  }, [value]);

  if (!value) return null;
  const line = (
    <div
      aria-label={`会话运行统计：${value}`}
      className="w-full cursor-default overflow-hidden text-ellipsis whitespace-nowrap px-4 pt-1 text-center text-xs leading-5 text-muted-foreground/75"
      data-testid="session-runtime-stats"
      ref={lineRef}
    >
      {value}
    </div>
  );
  if (!truncated) return line;
  return (
    <TooltipProvider delayDuration={500}>
      <Tooltip>
        <TooltipTrigger asChild>{line}</TooltipTrigger>
        <TooltipContent className="max-w-[min(42rem,calc(100vw-2rem))]" side="top" sideOffset={6}>
          {value}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
