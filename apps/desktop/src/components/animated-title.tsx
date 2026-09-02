import { useEffect, useState } from "react";

import { cn } from "@/lib/utils";

type AnimatedTitleProps = {
  value: string;
  className?: string;
  title?: string;
};

export function AnimatedTitle({ value, className, title }: AnimatedTitleProps) {
  const cleanValue = value.trim();
  const [displayValue, setDisplayValue] = useState(cleanValue);
  const [previousValue, setPreviousValue] = useState<string | null>(null);

  useEffect(() => {
    if (cleanValue === displayValue) return;
    let finishTimer = 0;
    const startTimer = window.setTimeout(() => {
      setPreviousValue(displayValue);
      setDisplayValue(cleanValue);
      finishTimer = window.setTimeout(() => setPreviousValue(null), 220);
    }, 0);
    return () => {
      window.clearTimeout(startTimer);
      window.clearTimeout(finishTimer);
    };
  }, [cleanValue, displayValue]);

  return (
    <span className={cn("relative block min-w-0 overflow-hidden", className)} title={title ?? displayValue}>
      {previousValue ? (
        <span className="title-change-out pointer-events-none absolute inset-0 truncate">
          {previousValue}
        </span>
      ) : null}
      <span className={cn("block truncate", previousValue ? "title-change-in" : "")}>
        {displayValue}
      </span>
    </span>
  );
}
