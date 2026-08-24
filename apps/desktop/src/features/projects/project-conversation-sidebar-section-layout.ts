import { cn } from "@/lib/utils";

export function collapsedSidebarSectionClassName(collapsed: boolean) {
  return cn(
    "grid origin-top overflow-hidden transition-[grid-template-rows,opacity,transform] duration-200 ease-out group-data-[collapsible=icon]:hidden",
    collapsed
      ? "pointer-events-none grid-rows-[0fr] -translate-y-1 opacity-0"
      : "grid-rows-[1fr] translate-y-0 opacity-100",
  );
}
