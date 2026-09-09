import { useMemo } from "react";
import {
  Archive,
  Bell,
  ChevronDown,
  Folder,
  LoaderCircle,
  Pin,
  Search,
  Settings,
  SquarePen,
} from "lucide-react";

import { Link } from "@tanstack/react-router";
import { AnimatedTitle } from "@/components/animated-title";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useAppConfig } from "@/lib/app-config";
import { appNameFromConfig } from "@/lib/app-identity";
import { cn } from "@/lib/utils";
import { projectNameFromPath } from "@/features/projects/project-sidebar-model";
import type { domain } from "@/types/codex-domain";
import type { ProjectWorkspaceScreenViewProps } from "./project-workspace-screen-view";

type ConversationBucket = {
  label: string;
  conversations: domain.Session[];
};

const DAY_MS = 24 * 60 * 60 * 1000;

function conversationTime(conversation: domain.Session) {
  return new Date(
    conversation.timeUpdated || conversation.timeCreated || 0,
  ).getTime();
}

function bucketConversations(conversations: domain.Session[]) {
  const startOfToday = new Date();
  startOfToday.setHours(0, 0, 0, 0);
  const startOfTodayMs = startOfToday.getTime();
  const startOfYesterdayMs = startOfTodayMs - DAY_MS;
  const startOfLastWeekMs = startOfTodayMs - 7 * DAY_MS;

  return conversations.reduce<ConversationBucket[]>((buckets, conversation) => {
    const timestamp = conversationTime(conversation);
    const label =
      timestamp >= startOfTodayMs
        ? "今天"
        : timestamp >= startOfYesterdayMs
          ? "昨天"
          : timestamp >= startOfLastWeekMs
            ? "最近 7 天"
            : "更早";
    const bucket = buckets.find((candidate) => candidate.label === label);

    if (bucket) {
      bucket.conversations.push(conversation);
    } else {
      buckets.push({ conversations: [conversation], label });
    }

    return buckets;
  }, []);
}

function SidebarIconButton({
  className,
  label,
  title,
  children,
  onClick,
}: {
  className?: string;
  label?: string;
  title?: string;
  children: React.ReactNode;
  onClick?: (event: React.MouseEvent<HTMLButtonElement>) => void;
}) {
  return (
    <Button
      aria-label={title ?? label}
      className={cn(
        "size-8 rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground",
        className,
      )}
      onClick={onClick}
      size="icon"
      title={title ?? label}
      type="button"
      variant="ghost"
    >
      {children}
    </Button>
  );
}

export function ProjectWorkspaceAppSidebar({
  activeConversationId,
  conversations,
  isCollapsed,
  onArchiveConversation,
  onNewConversation,
  onSelectConversation,
  pinnedConversationIds,
  runningConversationIds,
  onTogglePinnedConversation,
}: Pick<
  ProjectWorkspaceScreenViewProps["leftSidebar"],
  | "activeConversationId"
  | "conversations"
  | "onArchiveConversation"
  | "onNewConversation"
  | "onSelectConversation"
  | "pinnedConversationIds"
  | "runningConversationIds"
  | "onTogglePinnedConversation"
> & {
  isCollapsed: boolean;
}) {
  const appName = appNameFromConfig(useAppConfig((state) => state.config));
  const pinnedConversationIdSet = useMemo(
    () => new Set(pinnedConversationIds),
    [pinnedConversationIds],
  );
  const pinnedConversations = conversations.filter((conversation) =>
    pinnedConversationIdSet.has(conversation.id),
  );
  const regularConversations = conversations
    .filter((conversation) => !pinnedConversationIdSet.has(conversation.id))
    .toSorted(
      (left, right) => conversationTime(right) - conversationTime(left),
    );
  const conversationBuckets = useMemo(
    () => bucketConversations(regularConversations),
    [regularConversations],
  );
  const runningConversationIdSet = new Set(runningConversationIds);
  const isMac = window.aivoDesktop?.platform === "darwin";

  const renderConversation = (conversation: domain.Session) => {
    const isActive = conversation.id === activeConversationId;
    const projectName = conversation.projectPath
      ? projectNameFromPath(conversation.projectPath)
      : "";
    const isPinned = pinnedConversationIdSet.has(conversation.id);

    return (
      <div
        aria-current={isActive ? "page" : undefined}
        className="group/item relative w-full min-w-0 rounded-lg px-3 py-1.5 transition-colors hover:bg-sidebar-accent aria-[current=page]:bg-muted"
        key={conversation.id}
      >
        <button
          aria-current={isActive ? "page" : undefined}
          className="block w-full min-w-0 pr-0 text-left transition-[padding-right] duration-150 group-hover/item:pr-16"
          onClick={() => onSelectConversation(conversation)}
          type="button"
        >
          <div className="flex min-w-0 items-center gap-1.5">
            <AnimatedTitle
              className="min-w-0 flex-1 leading-5 text-sidebar-foreground"
              value={conversation.title}
            />
            {(runningConversationIdSet.has(conversation.id) ||
              conversation.status === "inProgress") && (
              <LoaderCircle
                aria-hidden="true"
                className="size-3.5 shrink-0 animate-spin text-sidebar-foreground/70"
              />
            )}
          </div>
          <div className="mt-0.5 flex items-center gap-2 text-xs text-muted-foreground">
            {projectName ? (
              <>
                <Folder aria-hidden="true" className="size-3 shrink-0" />
                <span className="truncate">{projectName}</span>
              </>
            ) : null}
          </div>
        </button>
        <div className="pointer-events-none absolute right-2 top-2 flex items-center gap-1 opacity-0 transition-opacity group-hover/item:pointer-events-auto group-hover/item:opacity-100">
          <SidebarIconButton
            aria-label={isPinned ? "取消置顶" : "置顶对话"}
            className="size-7 rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground"
            onClick={(event) => {
              event.stopPropagation();
              onTogglePinnedConversation(conversation.id);
            }}
            title={isPinned ? "取消置顶" : "置顶对话"}
          >
            <Pin className={isPinned ? "fill-current" : undefined} />
          </SidebarIconButton>
          <SidebarIconButton
            aria-label="归档对话"
            className="size-7 rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground"
            onClick={(event) => {
              event.stopPropagation();
              onArchiveConversation(conversation.id);
            }}
            title="归档对话"
          >
            <Archive />
          </SidebarIconButton>
        </div>
      </div>
    );
  };

  return (
    <div
      className={cn(
        "pointer-events-none relative h-dvh shrink-0 transition-[width] duration-200 ease-out",
        isCollapsed ? "w-0" : "w-64",
      )}
    >
      <aside
        className={cn(
          "pointer-events-auto flex h-dvh w-64 shrink-0 flex-col border-r border-sidebar-border bg-sidebar transition-transform duration-200 ease-out",
          isMac && "pt-10",
          isCollapsed ? "-translate-x-full" : "translate-x-0",
        )}
      >
        <>
          <header
            className="flex h-12 shrink-0 items-center justify-between gap-2 px-2"
            data-app-no-drag
          >
            <Button
              className="min-w-0 gap-1 rounded-lg px-1 py-1 text-xl font-bold tracking-tight hover:bg-muted"
              size="lg"
              type="button"
              variant="ghost"
            >
              <span className="truncate">{appName}</span>
              <ChevronDown
                aria-hidden="true"
                className="size-5 text-muted-foreground"
              />
            </Button>
            <div className="flex items-center">
              <SidebarIconButton label="搜索">
                <Search />
              </SidebarIconButton>
              <SidebarIconButton label="通知">
                <Bell />
              </SidebarIconButton>
            </div>
          </header>
          <div className="flex shrink-0 items-center px-2" data-app-no-drag>
            <div
              className="flex w-full cursor-default items-center gap-3 rounded-lg px-2 py-1.5 text-left transition-colors hover:bg-sidebar-accent"
              onClick={onNewConversation}
              onKeyDown={(event) => {
                if (event.key === "Enter" || event.key === " ") {
                  event.preventDefault();
                  onNewConversation();
                }
              }}
              role="menuitem"
              tabIndex={0}
            >
              <SquarePen aria-hidden="true" className="size-4 shrink-0" />
              新对话
            </div>
          </div>
          <ScrollArea className="min-h-0 flex-1 overflow-x-hidden [&>[data-slot=scroll-area-viewport]]:overflow-x-hidden [&>[data-slot=scroll-area-viewport]>div]:!block [&>[data-slot=scroll-area-viewport]>div]:!w-full [&>[data-slot=scroll-area-viewport]>div]:!min-w-0">
            <div className="pb-6">
              {pinnedConversations.length > 0 ? (
                <section className="mt-1">
                  <div className="flex h-7 items-center gap-2 px-5 text-sm font-medium text-muted-foreground">
                    <Pin aria-hidden="true" className="size-3.5" />
                    优先级
                  </div>
                  <div className="space-y-0.5 px-2">
                    {pinnedConversations.map(renderConversation)}
                  </div>
                </section>
              ) : null}
              {conversationBuckets.map((bucket) => (
                <section className="mt-3" key={bucket.label}>
                  <h2 className="px-5 text-sm font-medium text-muted-foreground">
                    {bucket.label}
                  </h2>
                  <div className="mt-1 space-y-0.5 px-2">
                    {bucket.conversations.map(renderConversation)}
                  </div>
                </section>
              ))}
            </div>
          </ScrollArea>
          <div
            aria-hidden="true"
            className="h-px shrink-0 bg-foreground/15"
          />
          <div className="flex shrink-0 items-center px-2 py-2" data-app-no-drag>
            <Button
              aria-label="打开设置"
              asChild
              className="size-8 rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground"
              size="icon"
              title="打开设置"
              type="button"
              variant="ghost"
            >
              <Link to="/settings">
                <Settings />
              </Link>
            </Button>
          </div>
        </>
      </aside>
    </div>
  );
}
