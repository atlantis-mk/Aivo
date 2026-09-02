import {
  useEffect,
  type ComponentProps,
  type CSSProperties,
  type ReactNode,
} from "react";
import { PanelBottom } from "lucide-react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useTerminalDock } from "@/features/projects/terminal/terminal-dock-store";

export function TerminalDockProvider({
  children,
  defaultOpen = false,
  onOpenChange,
  open: openProp,
}: {
  children: ReactNode;
  defaultOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
  open?: boolean;
}) {
  const open = useTerminalDock((dock) => dock.open);
  const setOpen = useTerminalDock((dock) => dock.setOpen);
  const toggleDock = useTerminalDock((dock) => dock.toggleDock);

  useEffect(() => {
    if (openProp !== undefined) {
      setOpen(openProp);
      return;
    }
    if (defaultOpen) {
      setOpen(true);
    }
  }, [defaultOpen, openProp, setOpen]);

  useEffect(() => {
    onOpenChange?.(open);
  }, [onOpenChange, open]);

  useEffect(() => {
    function handleShortcut(event: KeyboardEvent) {
      if ((!event.ctrlKey && !event.metaKey) || event.code !== "Backquote") return;
      event.preventDefault();
      toggleDock();
    }

    window.addEventListener("keydown", handleShortcut);
    return () => window.removeEventListener("keydown", handleShortcut);
  }, [toggleDock]);

  return children;
}

export function TerminalDockTrigger({
  children,
  className,
  onClick,
  ...props
}: ComponentProps<typeof Button>) {
  const { open, toggleDock } = useTerminalDock();

  return (
    <Button
      aria-pressed={open}
      className={cn(className)}
      data-slot="terminal-dock-trigger"
      onClick={(event) => {
        onClick?.(event);
        toggleDock();
      }}
      size="icon"
      type="button"
      variant="ghost"
      {...props}
    >
      {children ?? <PanelBottom />}
      <span className="sr-only">Toggle terminal dock</span>
    </Button>
  );
}

export function TerminalDockPanel({
  children,
  className,
  height = 300,
  style,
  ...props
}: ComponentProps<"section"> & {
  height?: number | string;
}) {
  const { state } = useTerminalDock();
  const panelHeight =
    typeof height === "number" ? `${height}px` : height;

  return (
    <section
      className={cn(
        "min-h-0 overflow-hidden transition-[height] duration-[var(--project-panel-transition-duration,200ms)] ease-linear",
        state === "expanded" ? "h-[var(--terminal-dock-height)]" : "h-0",
        className,
      )}
      data-slot="terminal-dock-panel"
      data-state={state}
      style={
        {
          "--terminal-dock-height": panelHeight,
          ...style,
        } as CSSProperties
      }
      {...props}
    >
      {children}
    </section>
  );
}
