import { memo } from "react";
import { toast } from "sonner";
import {
  ChevronDown,
  Copy,
  Ellipsis,
  FileText,
  FolderOpen,
  Pencil,
  X,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { ConversationSidebarItem } from "@/features/projects/project-conversation-sidebar-conversation-item";
import type { ProjectSidebarItemProps } from "@/features/projects/project-conversation-sidebar-model";
import { projectNameFromPath } from "@/features/projects/project-sidebar-model";
import { cn } from "@/lib/utils";

export const ProjectSidebarItem = memo(function ProjectSidebarItem({
  activeConversationId,
  collapsed,
  group,
  isActive,
  pinnedConversationIdSet,
  runningConversationIdSet,
  onArchiveConversation,
  onHideProject,
  onNewProjectConversation,
  onSelectConversation,
  onToggleProjectCollapsed,
  onTogglePinnedConversation,
  pendingPermissionCountsBySessionId,
}: ProjectSidebarItemProps) {
  const projectName =
    group.project.name || projectNameFromPath(group.projectPath);
  const conversationCount = group.conversations.length;

  return (
    <SidebarMenuItem className="min-w-0">
      <div className="group/project-row relative min-w-0">
        <SidebarMenuButton
          aria-expanded={!collapsed}
          className="min-w-0 pr-14 text-sidebar-foreground"
          isActive={isActive}
          onClick={() => onToggleProjectCollapsed(group.projectPath)}
          title={group.projectPath}
          type="button"
        >
          <FileText />
          <span className="flex min-w-0 items-center gap-1">
            <span className="min-w-0 truncate">{projectName}</span>
            <span className="relative inline-flex size-4 shrink-0 items-center justify-center text-muted-foreground">
              {conversationCount > 0 ? (
                <span className="text-xs leading-none transition-opacity duration-200 group-hover/project-row:opacity-0 group-focus-within/project-row:opacity-0">
                  {conversationCount}
                </span>
              ) : null}
              <ChevronDown
                className={cn(
                  "absolute opacity-0 transition-[opacity,transform] duration-200 group-hover/project-row:opacity-100 group-focus-within/project-row:opacity-100",
                  collapsed && "-rotate-90",
                )}
              />
            </span>
          </span>
        </SidebarMenuButton>
        <span className="pointer-events-none absolute right-1 top-1/2 flex -translate-y-1/2 items-center gap-0.5 opacity-0 transition-opacity group-hover/project-row:pointer-events-auto group-hover/project-row:opacity-100 group-focus-within/project-row:pointer-events-auto group-focus-within/project-row:opacity-100">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                aria-label="更多项目操作"
                onClick={(event) => event.stopPropagation()}
                size="icon-sm"
                type="button"
                variant="ghost"
              >
                <Ellipsis />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuLabel className="max-w-64 truncate">
                {projectName}
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onSelect={() => onNewProjectConversation(group.projectPath)}
              >
                <Pencil />
                新项目对话
              </DropdownMenuItem>
              <DropdownMenuItem
                onSelect={() => {
                  void window.aivo?.openPath(group.projectPath).catch((error) => {
                    toast.error(
                      error instanceof Error ? error.message : "打开项目失败",
                    );
                  });
                }}
              >
                <FolderOpen />
                打开目录
              </DropdownMenuItem>
              <DropdownMenuItem
                onSelect={() => {
                  void navigator.clipboard
                    ?.writeText(group.projectPath)
                    .then(() => toast.success("已复制项目路径"))
                    .catch(() => toast.error("复制项目路径失败"));
                }}
              >
                <Copy />
                复制路径
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onSelect={() => onHideProject(group.projectPath)}>
                <X />
                从侧边栏移除
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
          <Button
            aria-label="打开项目新对话"
            onClick={(event) => {
              event.preventDefault();
              event.stopPropagation();
              onNewProjectConversation(group.projectPath);
            }}
            size="icon-sm"
            type="button"
            variant="ghost"
          >
            <Pencil />
          </Button>
        </span>
      </div>
      {!collapsed && group.conversations.length > 0 ? (
        <SidebarMenu className="mt-1 min-w-0 gap-0.5 px-3">
          {group.conversations.map((conversation) => (
            <ConversationSidebarItem
              conversation={conversation}
              isActive={conversation.id === activeConversationId}
              isPinned={pinnedConversationIdSet.has(conversation.id)}
              isRunning={runningConversationIdSet.has(conversation.id)}
              key={conversation.id}
              onArchiveConversation={onArchiveConversation}
              onSelectConversation={onSelectConversation}
              onTogglePinnedConversation={onTogglePinnedConversation}
              pendingPermissionCount={
                pendingPermissionCountsBySessionId[conversation.id] ?? 0
              }
            />
          ))}
        </SidebarMenu>
      ) : null}
    </SidebarMenuItem>
  );
});
