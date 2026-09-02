import type { ComponentProps } from "react";
import { PanelBottom } from "lucide-react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export function WindowControls() {
  const isMac = window.aivo?.platform === "darwin";

  return (
    <div
      aria-hidden="true"
      className={cn("shrink-0", isMac ? "w-[54px]" : "w-0")}
      data-app-no-drag
    />
  );
}

export function TopBarIconButton({
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

export function TerminalTopBarButton({ onClick }: { onClick?: () => void }) {
  return (
    <TopBarIconButton aria-label="打开终端面板" onClick={onClick}>
      <PanelBottom />
    </TopBarIconButton>
  );
}
