import type React from "react";

import {
  normalizeProjectPathKey,
  type ProjectConversationGroup,
} from "@/features/projects/project-sidebar-model";
import { parseTime } from "@/features/projects/project-time-model";
import type { domain } from "../../../bridge/go/models";

export type ConversationSidebarProps = {
  activeConversationId: string;
  activeProjectPage: "chat" | "extensions";
  conversations: domain.Session[];
  onNewConversation: () => void;
  onNewProjectConversation: (projectPath: string) => void;
  onArchiveConversation: (sessionId: string) => void;
  onHideProject: (projectPath: string) => void;
  onSelectConversation: (session: domain.Session) => void;
  onTogglePinnedConversation: (sessionId: string) => void;
  pendingPermissionCountsBySessionId: Record<string, number>;
  pinnedConversationIds: string[];
  projectGroups: ProjectConversationGroup[];
  runningConversationIds: string[];
  selectedProjectPath: string;
  topBar: React.ReactNode;
};

export type ProjectSidebarItemProps = {
  activeConversationId: string;
  collapsed: boolean;
  group: ProjectConversationGroup;
  isActive: boolean;
  pinnedConversationIdSet: ReadonlySet<string>;
  runningConversationIdSet: ReadonlySet<string>;
  onArchiveConversation: (sessionId: string) => void;
  onHideProject: (projectPath: string) => void;
  onNewProjectConversation: (projectPath: string) => void;
  onSelectConversation: (session: domain.Session) => void;
  onToggleProjectCollapsed: (projectPath: string) => void;
  onTogglePinnedConversation: (sessionId: string) => void;
  pendingPermissionCountsBySessionId: Record<string, number>;
};

export type ConversationSidebarItemProps = {
  conversation: domain.Session;
  isActive: boolean;
  isPinned: boolean;
  onArchiveConversation: (sessionId: string) => void;
  onSelectConversation: (session: domain.Session) => void;
  onTogglePinnedConversation: (sessionId: string) => void;
  pendingPermissionCount: number;
  isRunning: boolean;
};

export function relativeConversationTime(value?: string) {
  const date = parseTime(value);
  const elapsedSeconds = Math.max(
    0,
    Math.floor((Date.now() - date.getTime()) / 1000),
  );
  if (elapsedSeconds < 60) return "刚刚";
  const elapsedMinutes = Math.floor(elapsedSeconds / 60);
  if (elapsedMinutes < 60) return `${elapsedMinutes} 分`;
  const elapsedHours = Math.floor(elapsedMinutes / 60);
  if (elapsedHours < 24) return `${elapsedHours} 小时`;
  const elapsedDays = Math.floor(elapsedHours / 24);
  if (elapsedDays < 7) return `${elapsedDays} 天`;
  return `${Math.floor(elapsedDays / 7)} 周`;
}

export function partitionConversationsByPinnedState(
  conversations: domain.Session[],
  pinnedConversationIdSet: ReadonlySet<string>,
) {
  const pinnedConversations: domain.Session[] = [];
  const regularConversations: domain.Session[] = [];

  for (const conversation of conversations) {
    if (pinnedConversationIdSet.has(conversation.id)) {
      pinnedConversations.push(conversation);
    } else {
      regularConversations.push(conversation);
    }
  }

  return { pinnedConversations, regularConversations };
}

export function getActiveProjectPathKey({
  activeConversationId,
  projectGroups,
  selectedProjectPath,
}: {
  activeConversationId: string;
  projectGroups: ProjectConversationGroup[];
  selectedProjectPath: string;
}) {
  if (!activeConversationId) return normalizeProjectPathKey(selectedProjectPath);

  for (const group of projectGroups) {
    if (
      group.conversations.some(
        (conversation) => conversation.id === activeConversationId,
      )
    ) {
      return normalizeProjectPathKey(group.projectPath);
    }
  }

  return "";
}
