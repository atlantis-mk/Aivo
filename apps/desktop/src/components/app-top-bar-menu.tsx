import { useState } from "react";
import { Ellipsis } from "lucide-react";

import type { MoreMenuItem } from "@/components/app-top-bar-types";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

export function MoreConversationMenu({
  groups,
  onOpen,
}: {
  groups: MoreMenuItem[][];
  onOpen?: () => void;
}) {
  const [open, setOpen] = useState(false);

  function handleOpenChange(nextOpen: boolean) {
    setOpen(nextOpen);
    if (nextOpen) {
      onOpen?.();
    }
  }

  return (
    <DropdownMenu open={open} onOpenChange={handleOpenChange}>
      <DropdownMenuTrigger asChild>
        <Button
          aria-expanded={open}
          aria-haspopup="menu"
          aria-label="更多会话操作"
          className="text-muted-foreground data-[state=open]:bg-muted data-[state=open]:text-foreground"
          size="icon"
          type="button"
          variant="ghost"
        >
          <Ellipsis />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="start"
        className="w-80 border border-border bg-popover text-popover-foreground shadow-xl shadow-foreground/10"
        side="bottom"
        sideOffset={8}
      >
        {groups.map((group, groupIndex) => (
          <DropdownMenuGroup key={group.map((item) => item.id).join("-")}>
            {group.map((item) => (
              <MoreConversationMenuItem item={item} key={item.id} />
            ))}
            {groupIndex < groups.length - 1 && (
              <DropdownMenuSeparator className="mx-2 my-2 bg-border" />
            )}
          </DropdownMenuGroup>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function MoreConversationMenuItem({ item }: { item: MoreMenuItem }) {
  const Icon = item.icon;

  if (item.hasSubmenu) {
    return (
      <DropdownMenuSub>
        <DropdownMenuSubTrigger
          className="hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground"
          disabled={item.disabled}
        >
          <Icon className="text-muted-foreground" />
          <span className="min-w-0 flex-1 truncate">{item.label}</span>
        </DropdownMenuSubTrigger>
        <DropdownMenuSubContent className="border border-border bg-popover text-popover-foreground shadow-xl shadow-foreground/10">
          {(item.children ?? []).map((child) => (
            <MoreConversationMenuItem item={child} key={child.id} />
          ))}
        </DropdownMenuSubContent>
      </DropdownMenuSub>
    );
  }

  return (
    <DropdownMenuItem
      className="hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground"
      disabled={item.disabled}
      onClick={item.onClick}
    >
      <Icon className="text-muted-foreground" />
      <span className="min-w-0 flex-1 truncate">{item.label}</span>
      {item.shortcut && (
        <DropdownMenuShortcut className="tracking-normal text-muted-foreground">
          {item.shortcut}
        </DropdownMenuShortcut>
      )}
    </DropdownMenuItem>
  );
}
