import { type ComponentProps } from "react";
import { Maximize2, Minimize2, PanelBottom, PanelRight } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useSidebar } from "@/components/ui/sidebar";
import { TerminalDockTrigger } from "@/features/projects/terminal/terminal-dock";
import { cn } from "@/lib/utils";

export function ProjectTopBarIconButton({
  className,
  ...props
}: ComponentProps<typeof Button>) {
  return (
    <Button
      className={cn("text-muted-foreground", className)}
      size="icon"
      type="button"
      variant="ghost"
      {...props}
    />
  );
}

export function ProjectFloatingRightTopBarActions({
  isRightSidebarMaximized,
  onToggleRightSidebarMaximized,
  rightOpen,
  showTerminalButton,
}: {
  isRightSidebarMaximized: boolean;
  onToggleRightSidebarMaximized: () => void;
  rightOpen: boolean;
  showTerminalButton?: boolean;
}) {
  const { toggleSidebar: toggleRightSidebar } = useSidebar();

  return (
    <div
      className="pointer-events-auto absolute right-0 top-0 z-[80] flex h-9 shrink-0 items-center justify-end gap-2 pe-3 text-foreground"
      data-app-no-drag
    >
      {rightOpen ? (
        <ProjectTopBarIconButton
          aria-label={isRightSidebarMaximized ? "恢复右侧栏宽度" : "全屏右侧栏"}
          aria-pressed={isRightSidebarMaximized}
          onClick={onToggleRightSidebarMaximized}
          title={isRightSidebarMaximized ? "恢复右侧栏宽度" : "全屏右侧栏"}
        >
          {isRightSidebarMaximized ? <Minimize2 /> : <Maximize2 />}
        </ProjectTopBarIconButton>
      ) : null}
      {showTerminalButton ? (
        <TerminalDockTrigger
          aria-label="打开终端面板"
          className="text-muted-foreground"
        >
          <PanelBottom />
        </TerminalDockTrigger>
      ) : null}
      <ProjectTopBarIconButton
        aria-label="打开或关闭侧边面板"
        onClick={toggleRightSidebar}
      >
        <PanelRight />
      </ProjectTopBarIconButton>
    </div>
  );
}
