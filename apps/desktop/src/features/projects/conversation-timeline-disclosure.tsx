import { useEffect, useState, type ReactNode } from "react";

import { cn } from "@/lib/utils";

const DISCLOSURE_ANIMATION_MS = 200;

export function AnimatedDisclosure({
  children,
  className,
  open,
}: {
  children: ReactNode;
  className?: string;
  open: boolean;
}) {
  const [shouldRender, setShouldRender] = useState(open);
  const renderChildren = open || shouldRender;

  useEffect(() => {
    if (open) {
      setShouldRender(true);
      return;
    }

    const timeout = window.setTimeout(
      () => setShouldRender(false),
      DISCLOSURE_ANIMATION_MS,
    );
    return () => window.clearTimeout(timeout);
  }, [open]);

  if (!open && !renderChildren) return null;

  return (
    <div
      aria-hidden={!open}
      className={cn(
        "grid overflow-hidden transition-[grid-template-rows,opacity] duration-200 ease-out motion-reduce:transition-none",
        open ? "grid-rows-[1fr] opacity-100" : "grid-rows-[0fr] opacity-0",
        className,
      )}
    >
      {renderChildren ? (
        <div className="min-h-0 overflow-hidden">{children}</div>
      ) : null}
    </div>
  );
}
