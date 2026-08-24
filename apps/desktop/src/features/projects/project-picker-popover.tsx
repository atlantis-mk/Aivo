import { useState } from "react";
import { FolderOpen, Plus, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import type { ProjectPickerProps } from "@/features/projects/project-prompt-composer-types";
import {
  projectNameFromPath,
  projectPickerLabel,
} from "@/features/projects/project-sidebar-model";
import { cn } from "@/lib/utils";

export function ProjectPicker({
  onAddProject,
  onProjectClear,
  onProjectSelect,
  project,
  projectPath,
  projects,
}: ProjectPickerProps) {
  const [open, setOpen] = useState(false);
  const label = projectPickerLabel(project, projectPath);
  const hasCurrentProject = Boolean(projectPath);

  return (
    <Popover onOpenChange={setOpen} open={open}>
      <div className="group/project-picker relative inline-flex">
        <PopoverTrigger asChild>
          <Button className="rounded-full" type="button" variant="ghost">
            <FolderOpen
              className={cn(
                "transition-opacity",
                hasCurrentProject && "group-hover/project-picker:opacity-0",
              )}
            />
            <span className="truncate">{label}</span>
          </Button>
        </PopoverTrigger>
        {hasCurrentProject ? (
          <button
            aria-label="清除当前项目选择"
            className="absolute left-1.5 top-1/2 z-10 inline-flex size-4 -translate-y-1/2 items-center justify-center rounded-sm text-muted-foreground opacity-0 transition-opacity hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/30 group-hover/project-picker:opacity-100"
            onClick={(event) => {
              event.preventDefault();
              event.stopPropagation();
              onProjectClear();
              setOpen(false);
            }}
            title="清除当前项目选择"
            type="button"
          >
            <X aria-hidden="true" />
          </button>
        ) : null}
      </div>
      <PopoverContent align="start" side="top">
        <Command>
          <div className="flex items-center gap-1 [&>[data-slot=command-input-wrapper]]:min-w-0 [&>[data-slot=command-input-wrapper]]:flex-1">
            <CommandInput placeholder="搜索项目" />
            <Button
              aria-label="新项目"
              className="mr-1"
              onClick={() => {
                onAddProject();
                setOpen(false);
              }}
              size="icon-lg"
              title="新项目"
              type="button"
              variant="ghost"
            >
              <Plus />
            </Button>
          </div>
          <CommandList>
            <CommandEmpty>没有找到项目</CommandEmpty>
            {projects.length > 0 ? (
              <CommandGroup>
                {projects.map((item) => (
                  <CommandItem
                    data-checked={item.rootPath === projectPath}
                    key={item.rootPath}
                    onSelect={() => {
                      onProjectSelect(item);
                      setOpen(false);
                    }}
                    value={`${item.name} ${item.rootPath}`}
                  >
                    <FolderOpen />
                    <span className="min-w-0 flex-1 truncate">
                      {item.name || projectNameFromPath(item.rootPath)}
                    </span>
                  </CommandItem>
                ))}
              </CommandGroup>
            ) : null}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
