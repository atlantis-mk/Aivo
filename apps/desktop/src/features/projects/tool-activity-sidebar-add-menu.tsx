import { Globe, Plus } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { SUPPORTED_SIDEBAR_TABS } from "@/features/projects/tool-activity-sidebar-supported-tabs";

export function ToolActivitySidebarAddMenu({
  onOpenBrowser,
}: {
  onOpenBrowser?: (targetUrl?: string) => void;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          aria-label="添加右侧栏标签"
          className="size-7 text-muted-foreground transition-none"
          size="icon-sm"
          title="添加右侧栏标签"
          type="button"
          variant="ghost"
        >
          <Plus />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="z-[90] w-44">
        <DropdownMenuItem onClick={() => onOpenBrowser?.()}>
          <Globe />
          <span>内置浏览器</span>
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        {SUPPORTED_SIDEBAR_TABS.map((tab) => {
          const Icon = tab.icon;
          return (
            <DropdownMenuItem disabled key={tab.id}>
              <Icon />
              <span>{tab.label}</span>
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
