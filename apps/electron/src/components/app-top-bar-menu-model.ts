import {
  Archive,
  Clock,
  Copy,
  ExternalLink,
  GitBranch,
  MessageSquarePlus,
  Pencil,
  Pin,
} from "lucide-react";

import type { AppTopBarProps, MoreMenuItem } from "@/components/app-top-bar-types";

export const topBarMenuItemClassName =
  "relative flex min-h-7 cursor-default items-center gap-2 rounded-md px-2 py-1 text-xs/relaxed outline-hidden select-none hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-3.5";

export function createMoreMenuGroups({
  onAddScheduledTask,
  onArchiveConversation,
  onBranch,
  onCopy,
  onOpenInNewWindow,
  onOpenSideChat,
  onPinConversation,
  onRenameConversation,
}: Pick<
  AppTopBarProps,
  | "onAddScheduledTask"
  | "onArchiveConversation"
  | "onBranch"
  | "onCopy"
  | "onOpenInNewWindow"
  | "onOpenSideChat"
  | "onPinConversation"
  | "onRenameConversation"
>) {
  return [
    [
      {
        id: "pin-conversation",
        label: "置顶对话",
        icon: Pin,
        shortcut: "⌥⌘P",
        onClick: onPinConversation,
      },
      {
        id: "rename-conversation",
        label: "重命名对话",
        icon: Pencil,
        shortcut: "⌥⌘R",
        onClick: onRenameConversation,
      },
      {
        id: "archive-conversation",
        label: "归档对话",
        icon: Archive,
        shortcut: "⇧⌘A",
        onClick: onArchiveConversation,
      },
    ],
    [
      {
        id: "open-side-chat",
        label: "打开侧边聊天",
        icon: MessageSquarePlus,
        shortcut: "⌥⌘S",
        onClick: onOpenSideChat,
      },
      {
        id: "copy",
        label: "复制",
        icon: Copy,
        hasSubmenu: true,
        children: [
          {
            id: "copy-conversation",
            label: "复制对话",
            icon: Copy,
            onClick: onCopy,
          },
        ],
      },
      {
        id: "branch",
        label: "分支",
        icon: GitBranch,
        hasSubmenu: true,
        children: [
          {
            id: "create-branch",
            label: "创建分支",
            icon: GitBranch,
            onClick: onBranch,
          },
        ],
      },
      {
        id: "add-scheduled-task",
        label: "添加计划任务...",
        icon: Clock,
        onClick: onAddScheduledTask,
      },
    ],
    [
      {
        id: "open-in-new-window",
        label: "在新窗口中打开",
        icon: ExternalLink,
        onClick: onOpenInNewWindow,
      },
    ],
  ] satisfies MoreMenuItem[][];
}
