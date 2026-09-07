import { ChevronDown, Ellipsis } from "lucide-react";

import { Button } from "@/components/ui/button";
import { SidebarGroupLabel } from "@/components/ui/sidebar";
import { cn } from "@/lib/utils";

export function SidebarSectionHeader({
  collapsed,
  label,
  moreLabel,
  onToggle,
}: {
  collapsed: boolean;
  label: string;
  moreLabel: string;
  onToggle: () => void;
}) {
  return (
    <SidebarGroupLabel
      aria-expanded={!collapsed}
      className="group/sidebar-section-heading flex h-6 cursor-pointer items-center gap-1 px-3 text-sm font-semibold text-muted-foreground group-data-[collapsible=icon]:mx-2"
      onClick={onToggle}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onToggle();
        }
      }}
      role="button"
      tabIndex={0}
    >
      <span className="min-w-0 truncate">{label}</span>
      <Button
        aria-label={collapsed ? `展开${label}` : `收起${label}`}
        onClick={(event) => {
          event.stopPropagation();
          onToggle();
        }}
        size="icon-xs"
        type="button"
        variant="ghost"
      >
        <ChevronDown
          className={cn(
            "transition-transform duration-200",
            collapsed && "-rotate-90",
          )}
        />
      </Button>
      <span className="min-w-0 flex-1" />
      <Button
        aria-label={moreLabel}
        onClick={(event) => event.stopPropagation()}
        size="icon-xs"
        type="button"
        variant="ghost"
      >
        <Ellipsis />
      </Button>
    </SidebarGroupLabel>
  );
}
